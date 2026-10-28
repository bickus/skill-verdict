// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refs

import (
	"cmp"
	"fmt"
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
	ID       string   `json:"id,omitempty"`
	Ref      Ref      `json:"reference"`
	Status   Status   `json:"status"`
	Detail   string   `json:"detail,omitempty"`
	Facts    []Fact   `json:"facts"`
	Usages   []string `json:"usages,omitempty"`
	Findings []string `json:"-"`
	Derived  []Ref    `json:"-"`
}

func (r Report) LocationID(i int) string {
	return fmt.Sprintf("%s.%d", r.ID, i+1)
}

func (r Report) Usage(i int) string {
	if i < len(r.Usages) {
		return r.Usages[i]
	}
	return ""
}

func Sort(reports []Report) {
	slices.SortStableFunc(reports, func(a, b Report) int {
		return cmp.Or(cmp.Compare(a.Ref.Kind, b.Ref.Kind), cmp.Compare(a.Ref.Value, b.Ref.Value))
	})
}

func Number(reports []Report) {
	Sort(reports)
	for i := range reports {
		reports[i].ID = fmt.Sprintf("R%d", i+1)
	}
}
