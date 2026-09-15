// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/bickus/skill-verdict/pkg/rules"
)

func rulesCmd(args []string, stdout io.Writer) int {
	fs := flag.NewFlagSet("rules", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "", "configuration file")
	rulesDir := fs.String("rules-dir", "", "directory of additional rule files")
	format := fs.String("format", "text", "output format: text or markdown")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	cfg, err := loadConfig(*configPath, *rulesDir)
	if err != nil {
		return fail(err)
	}
	set, err := rules.Load(cfg.Rules.Dir, cfg.Layers.Rules())
	if err != nil {
		return fail(err)
	}
	if err = rules.Render(set, *format, stdout); err != nil {
		return fail(err)
	}
	return 0
}
