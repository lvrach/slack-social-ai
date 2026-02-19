package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lvrach/slack-social-ai/internal/config"
)

//go:embed slack-social-ai.guide.md
var guideContent string

// GuideCmd prints the posting guide to stdout.
type GuideCmd struct{}

func personaPath() string {
	return filepath.Join(config.ConfigDir(), "persona.md")
}

func loadPersona() (string, error) {
	data, err := os.ReadFile(personaPath())
	if os.IsNotExist(err) {
		return "", nil
	}
	return string(data), err
}

func (cmd *GuideCmd) Run(globals *Globals) error { //nolint:unparam
	fmt.Print(guideContent)

	persona, err := loadPersona()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not read persona file: %v\n", err)
		return nil
	}

	fmt.Println()
	if persona != "" {
		fmt.Println("## Your Persona")
		fmt.Println()
		fmt.Print(persona)
	} else {
		fmt.Printf("## Your Persona\n\n")
		fmt.Printf("No persona file found. Create `%s` to define your posting voice.\n", personaPath())
		fmt.Printf("The persona customizes tone, humor, and topic focus — the guide's rules still apply.\n")
	}

	return nil
}
