package launchd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempPlistDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	original := plistDir
	plistDir = func() string { return dir }
	t.Cleanup(func() { plistDir = original })
	return dir
}

func TestLabel(t *testing.T) {
	want := "com.slack-social-ai.publish"
	if Label != want {
		t.Errorf("Label = %q, want %q", Label, want)
	}
}

func TestPlistPath(t *testing.T) {
	dir := withTempPlistDir(t)

	got := PlistPath()
	want := filepath.Join(dir, "com.slack-social-ai.publish.plist")
	if got != want {
		t.Errorf("PlistPath() = %q, want %q", got, want)
	}
}

func TestGeneratePlist(t *testing.T) {
	plistBytes, err := GeneratePlist("/usr/local/bin/slack-social-ai")
	if err != nil {
		t.Fatalf("GeneratePlist() error = %v", err)
	}

	xml := string(plistBytes)

	checks := []struct {
		name    string
		contain string
	}{
		{"Label", "<string>com.slack-social-ai.publish</string>"},
		{"binary path", "<string>/usr/local/bin/slack-social-ai</string>"},
		{"publish arg", "<string>publish</string>"},
		{"json flag", "<string>--json</string>"},
		{"StartInterval", "<integer>900</integer>"},
		{"publish.log", "publish.log"},
		{"RunAtLoad false", "<false/>"},
		{"HOME env key", "<key>HOME</key>"},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(xml, c.contain) {
				t.Errorf("plist XML does not contain %q\n\nGot:\n%s", c.contain, xml)
			}
		})
	}
}

func TestIsInstalled_True(t *testing.T) {
	dir := withTempPlistDir(t)

	// Create the plist file.
	path := filepath.Join(dir, Label+".plist")
	if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if !IsInstalled() {
		t.Error("IsInstalled() = false, want true when plist file exists")
	}
}

func TestIsInstalled_False(t *testing.T) {
	withTempPlistDir(t)

	if IsInstalled() {
		t.Error("IsInstalled() = true, want false when plist file does not exist")
	}
}
