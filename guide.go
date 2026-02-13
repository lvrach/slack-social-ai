package main

import (
	_ "embed"
	"fmt"
)

//go:embed slack-social-ai.guide.md
var guideContent string

// GuideCmd prints the posting guide to stdout.
type GuideCmd struct{}

func (cmd *GuideCmd) Run(globals *Globals) error {
	fmt.Print(guideContent)
	return nil
}
