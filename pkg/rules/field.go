// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package rules

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/bickus/skill-verdict/pkg/config"
)

type Field struct {
	Question string            `json:"question"`
	Options  map[string]string `json:"options"`
	Accept   []string          `json:"accept"`
}

func (r Rule) validateFields() error {
	if len(r.Fields) > 0 && r.Layer != config.DiscoveryLayer {
		return fmt.Errorf("fields: only a rule of the %s layer has fields", config.DiscoveryLayer)
	}
	for _, name := range slices.Sorted(maps.Keys(r.Fields)) {
		if err := r.Fields[name].validate(); err != nil {
			return fmt.Errorf("fields.%s.%w", name, err)
		}
	}
	return nil
}

func (f Field) validate() error {
	switch {
	case f.Question == "":
		return errors.New("question: a field needs a question")
	case len(f.Options) < 2:
		return errors.New("options: a field needs at least two options")
	case len(f.Accept) == 0:
		return errors.New("accept: a field accepts at least one option")
	}
	for _, option := range f.Accept {
		if _, ok := f.Options[option]; !ok {
			return fmt.Errorf("accept: unknown option %q", option)
		}
	}
	return nil
}
