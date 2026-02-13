package manifest

import (
	"encoding/json"
	"strings"
)

// manifest mirrors the Slack app manifest schema (relevant fields only).
type manifest struct {
	Metadata           metadata           `json:"_metadata"`
	DisplayInformation displayInformation `json:"display_information"`
	Features           features           `json:"features"`
	OAuthConfig        oauthConfig        `json:"oauth_config"`
	Settings           settings           `json:"settings"`
}

type metadata struct {
	MajorVersion int `json:"major_version"`
	MinorVersion int `json:"minor_version"`
}

type displayInformation struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	BackgroundColor string `json:"background_color"`
}

type features struct {
	BotUser botUser `json:"bot_user"`
}

type botUser struct {
	DisplayName  string `json:"display_name"`
	AlwaysOnline bool   `json:"always_online"`
}

type oauthConfig struct {
	Scopes oauthScopes `json:"scopes"`
}

type oauthScopes struct {
	Bot []string `json:"bot"`
}

type settings struct {
	IncomingWebhooks     incomingWebhooks `json:"incoming_webhooks"`
	OrgDeployEnabled     bool             `json:"org_deploy_enabled"`
	SocketModeEnabled    bool             `json:"socket_mode_enabled"`
	TokenRotationEnabled bool             `json:"token_rotation_enabled"`
}

type incomingWebhooks struct {
	Enabled bool `json:"incoming_webhooks_enabled"`
}

// Generate returns a Slack app manifest as pretty-printed JSON.
func Generate(appName string) string {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "slack-social-ai"
	}

	m := manifest{
		Metadata: metadata{MajorVersion: 1, MinorVersion: 1},
		DisplayInformation: displayInformation{
			Name:            appName,
			Description:     "Post messages to Slack from the terminal",
			BackgroundColor: "#4A154B",
		},
		Features: features{
			BotUser: botUser{DisplayName: appName, AlwaysOnline: false},
		},
		OAuthConfig: oauthConfig{
			Scopes: oauthScopes{Bot: []string{"incoming-webhook"}},
		},
		Settings: settings{
			IncomingWebhooks:     incomingWebhooks{Enabled: true},
			OrgDeployEnabled:     false,
			SocketModeEnabled:    false,
			TokenRotationEnabled: false,
		},
	}

	b, _ := json.MarshalIndent(m, "", "  ")
	return string(b) + "\n"
}
