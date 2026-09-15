// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/internal/version"
)

const usage = `usage:
  skill-verdict scan [--config P] [--rules-dir D] [--format text|json] [--verbose] [--fail-on LEVEL] <path>...
  skill-verdict rules [--config P] [--rules-dir D] [--format text|markdown]
  skill-verdict config-template <path>
  skill-verdict --version
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout))
}

func run(args []string, stdout io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	switch args[0] {
	case "scan":
		return scanCmd(args[1:], stdout)
	case "rules":
		return rulesCmd(args[1:], stdout)
	case "config-template":
		return templateCmd(args[1:])
	case "--version", "version":
		return emit(stdout, version.String()+"\n")
	case "-h", "--help", "help":
		return emit(stdout, usage)
	}
	fmt.Fprint(os.Stderr, usage)
	return 2
}

func emit(w io.Writer, s string) int {
	if _, err := io.WriteString(w, s); err != nil {
		return fail(err)
	}
	return 0
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, err)
	return 2
}

func loadConfig(path, rulesDir string) (config.Config, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return cfg, err
	}
	if rulesDir != "" {
		cfg.Rules.Dir = rulesDir
	}
	return cfg, nil
}
