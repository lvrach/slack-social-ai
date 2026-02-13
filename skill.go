package main

import (
	_ "embed"
	"fmt"
)

//go:embed slack-social-ai.skill.md
var skillContent string

// SkillCmd prints the agent skill instructions to stdout.
type SkillCmd struct{}

func (cmd *SkillCmd) Run(globals *Globals) error {
	fmt.Print(skillContent)
	return nil
}
