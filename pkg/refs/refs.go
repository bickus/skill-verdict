// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refs

import (
	"cmp"
	"slices"

	"github.com/bickus/skill-verdict/pkg/finding"
)

type Kind string

const (
	Domain  Kind = "domain"
	GitHub  Kind = "github"
	Package Kind = "package"
	Address Kind = "address"
)

func Kinds() []Kind {
	return []Kind{Domain, GitHub, Package, Address}
}

type Status string

const (
	Checked    Status = "checked"
	NotChecked Status = "not-checked"
	Failed     Status = "failed"
)

type Ref struct {
	Kind      Kind               `json:"kind"`
	Value     string             `json:"value"`
	URLs      []string           `json:"urls,omitempty"`
	Locations []finding.Location `json:"locations"`
	Derived   bool               `json:"derived,omitempty"`
}

func (r Ref) Key() string {
	return string(r.Kind) + ":" + r.Value
}

type Fact struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Report struct {
	Ref      Ref      `json:"reference"`
	Status   Status   `json:"status"`
	Detail   string   `json:"detail,omitempty"`
	Facts    []Fact   `json:"facts"`
	Findings []string `json:"-"`
	Derived  []Ref    `json:"-"`
}

func Sort(reports []Report) {
	slices.SortStableFunc(reports, func(a, b Report) int {
		return cmp.Or(cmp.Compare(a.Ref.Kind, b.Ref.Kind), cmp.Compare(a.Ref.Value, b.Ref.Value))
	})
}
