// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/rules"
)

func templateCmd(args []string) int {
	fs := flag.NewFlagSet("config-template", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil || fs.NArg() != 1 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	cfg := config.Default()
	set, err := rules.Load("", cfg.Layers.Rules())
	if err != nil {
		return fail(err)
	}
	cfg.Layers.SetRules(rules.Settings(set))
	if err = writeTemplate(fs.Arg(0), cfg); err != nil {
		return fail(err)
	}
	return 0
}

func writeTemplate(path string, cfg config.Config) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, err = f.Write(buf.Bytes())
	return errors.Join(err, f.Close())
}
