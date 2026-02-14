package launchd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"

	"howett.net/plist"
)

const Label = "com.slack-social-ai.publish"

// plistData represents the launchd plist structure.
type plistData struct {
	Label                string            `plist:"Label"`
	ProgramArguments     []string          `plist:"ProgramArguments"`
	StartInterval        int               `plist:"StartInterval"`
	StandardOutPath      string            `plist:"StandardOutPath"`
	StandardErrorPath    string            `plist:"StandardErrorPath"`
	RunAtLoad            bool              `plist:"RunAtLoad"`
	EnvironmentVariables map[string]string `plist:"EnvironmentVariables"`
}

// plistDir is overridable for testing.
var plistDir = defaultPlistDir

func defaultPlistDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents")
}

// PlistPath returns the path to the launchd plist file.
func PlistPath() string {
	return filepath.Join(plistDir(), Label+".plist")
}

// LogPath returns the path for publish command logs.
func LogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "slack-social-ai", "publish.log")
}

// GeneratePlist creates the plist XML for the publishing schedule.
func GeneratePlist(binaryPath string) ([]byte, error) {
	home, _ := os.UserHomeDir()

	data := plistData{
		Label:             Label,
		ProgramArguments:  []string{binaryPath, "publish", "--json"},
		StartInterval:     600, // 10 minutes
		StandardOutPath:   LogPath(),
		StandardErrorPath: LogPath(),
		RunAtLoad:         false,
		EnvironmentVariables: map[string]string{
			"HOME": home,
		},
	}

	var buf bytes.Buffer
	encoder := plist.NewEncoder(&buf)
	encoder.Indent("\t")
	if err := encoder.Encode(data); err != nil {
		return nil, fmt.Errorf("encode plist: %w", err)
	}
	return buf.Bytes(), nil
}

// IsInstalled checks if the plist file exists.
func IsInstalled() bool {
	_, err := os.Stat(PlistPath())
	return err == nil
}

// Install writes the plist and bootstraps it with launchctl.
func Install(binaryPath string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf(
			"automatic scheduling requires macOS (launchd). For Linux/other, set up a cron job manually:\n  */10 * * * * %s publish --json >> ~/.local/share/slack-social-ai/publish.log 2>&1",
			binaryPath,
		)
	}

	plistBytes, err := GeneratePlist(binaryPath)
	if err != nil {
		return err
	}

	dir := plistDir()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	path := PlistPath()

	// If already installed, bootout first (ignore errors — may not be loaded).
	if IsInstalled() {
		uid := currentUID()
		_ = exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%s/%s", uid, Label)).Run() //nolint:gosec // launchctl path constructed from constants
	}

	if err := os.WriteFile(path, plistBytes, 0o600); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	uid := currentUID()
	cmd := exec.Command("launchctl", "bootstrap", fmt.Sprintf("gui/%s", uid), path) //nolint:gosec // launchctl path constructed from constants
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl bootstrap: %s (%w)", string(output), err)
	}

	return nil
}

// Uninstall removes the plist and bootout from launchctl.
func Uninstall() error {
	uid := currentUID()
	// Bootout first (ignore error if not loaded).
	_ = exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%s/%s", uid, Label)).Run() //nolint:gosec // launchctl path constructed from constants

	path := PlistPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}
	return nil
}

// IsLoaded checks if the service is currently loaded in launchctl.
func IsLoaded() bool {
	uid := currentUID()
	err := exec.Command("launchctl", "print", fmt.Sprintf("gui/%s/%s", uid, Label)).Run() //nolint:gosec // launchctl path constructed from constants
	return err == nil
}

func currentUID() string {
	u, err := user.Current()
	if err != nil {
		return "501" // common default macOS UID
	}
	return u.Uid
}
