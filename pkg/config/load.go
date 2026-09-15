// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "skill-verdict", "config.json"), nil
}

func Load(path string) (Config, error) {
	cfg := Default()
	discovered := path == ""
	if discovered {
		p, err := Path()
		if err != nil {
			return cfg, err
		}
		path = p
	}
	data, err := os.ReadFile(path)
	if discovered && errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

func (c Config) APIKey() (string, error) {
	if key := os.Getenv(c.LLM.KeyEnv); key != "" {
		return strings.TrimSpace(key), nil
	}
	if c.LLM.KeyFile == "" {
		return "", nil
	}
	data, err := os.ReadFile(c.LLM.KeyFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
