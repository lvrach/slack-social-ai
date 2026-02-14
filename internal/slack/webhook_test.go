package slack

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyWebhook_Valid_NoText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("no_text"))
	}))
	defer srv.Close()

	err := VerifyWebhook(srv.URL)
	assert.NoError(t, err)
}

func TestVerifyWebhook_Valid_MissingText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("missing_text_or_fallback_or_attachments"))
	}))
	defer srv.Close()

	err := VerifyWebhook(srv.URL)
	assert.NoError(t, err)
}

func TestVerifyWebhook_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	err := VerifyWebhook(srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestVerifyWebhook_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	err := VerifyWebhook(srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestVerifyWebhook_Gone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
	}))
	defer srv.Close()

	err := VerifyWebhook(srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "410")
}

func TestVerifyWebhook_Unreachable(t *testing.T) {
	err := VerifyWebhook("http://127.0.0.1:1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unreachable")
}

func TestSendWebhook_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := SendWebhook(srv.URL, "test")
	assert.NoError(t, err)
}

func TestSendWebhook_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer srv.Close()

	err := SendWebhook(srv.URL, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}
