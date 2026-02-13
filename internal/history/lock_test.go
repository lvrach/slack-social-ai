package history

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrentAppend(t *testing.T) {
	withTempDataDir(t)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	errs := make([]error, goroutines)
	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = Append("msg", "queued", time.Time{})
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		assert.NoErrorf(t, err, "goroutine %d failed", i)
	}

	entries, err := Load()
	require.NoError(t, err)
	assert.Len(t, entries, goroutines)

	// All IDs should be unique.
	ids := make(map[string]bool)
	for _, e := range entries {
		assert.False(t, ids[e.ID], "duplicate ID: %s", e.ID)
		ids[e.ID] = true
	}
}

func TestConcurrentClaimAndAppend(t *testing.T) {
	withTempDataDir(t)

	const numEntries = 20
	const claimers = 5
	const appenders = 5

	// Pre-populate some entries.
	for range numEntries {
		_, err := Append("initial", "queued", time.Time{})
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	wg.Add(claimers + appenders)

	claimedIDs := make([]string, 0, numEntries)
	var mu sync.Mutex

	// Claimers.
	for range claimers {
		go func() {
			defer wg.Done()
			for {
				entry, err := ClaimNextReady()
				if err != nil {
					return
				}
				if entry == nil {
					return
				}
				mu.Lock()
				claimedIDs = append(claimedIDs, entry.ID)
				mu.Unlock()
			}
		}()
	}

	// Appenders.
	appendErrors := make([]error, appenders)
	for i := range appenders {
		go func(idx int) {
			defer wg.Done()
			_, appendErrors[idx] = Append("concurrent", "queued", time.Time{})
		}(i)
	}

	wg.Wait()

	// No append errors.
	for i, err := range appendErrors {
		assert.NoErrorf(t, err, "appender %d failed", i)
	}

	// No double-claims: all claimed IDs should be unique.
	seen := make(map[string]bool)
	for _, id := range claimedIDs {
		assert.False(t, seen[id], "double claim detected for ID: %s", id)
		seen[id] = true
	}

	// Verify no entries are lost: total entries = those still on disk.
	entries, err := Load()
	require.NoError(t, err)

	// Count entries by status.
	statusCounts := make(map[string]int)
	for _, e := range entries {
		statusCounts[e.Status]++
	}

	totalOnDisk := len(entries)
	totalClaimed := len(claimedIDs)
	// All original queued entries should be either claimed (publishing) or still queued,
	// plus the appended entries.
	assert.Equal(t, numEntries+appenders, totalOnDisk,
		"expected %d entries on disk, got %d (status counts: %v)",
		numEntries+appenders, totalOnDisk, statusCounts)
	assert.LessOrEqual(t, totalClaimed, numEntries+appenders,
		"claimed more entries than exist: %d > %d", totalClaimed, numEntries+appenders)
}
