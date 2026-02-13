package keyring

import (
	"errors"

	gokeyring "github.com/zalando/go-keyring"
)

// ErrNotFound is returned when no webhook URL is stored.
var ErrNotFound = gokeyring.ErrNotFound

const (
	serviceName = "slack-social"
	userName    = "webhook-url"
)

// IsNotFound reports whether err indicates a missing keyring entry.
func IsNotFound(err error) bool {
	return errors.Is(err, gokeyring.ErrNotFound)
}

// Get retrieves the stored webhook URL from the system keychain.
func Get() (string, error) {
	return gokeyring.Get(serviceName, userName)
}

// Set stores the webhook URL in the system keychain.
func Set(url string) error {
	return gokeyring.Set(serviceName, userName, url)
}

// Delete removes the webhook URL from the system keychain.
func Delete() error {
	return gokeyring.Delete(serviceName, userName)
}
