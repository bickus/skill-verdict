// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/refs"
)

const (
	detailCap       = "cap"
	detailBudget    = "budget"
	detailRateLimit = "rate limit"
)

type Collector interface {
	Kind() refs.Kind
	Extract(a *inventory.Artifact) []refs.Ref
	Collect(ctx context.Context, ref refs.Ref) refs.Report
}

type Runner struct {
	Collectors []Collector
	Config     config.ReferencesConfig
	Capped     func(first refs.Ref, total int)
}

func (r *Runner) Run(ctx context.Context, skill *inventory.Skill) []refs.Report {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(r.Config.Budget))
	defer cancel()
	c := &coordinator{runner: r, collectors: make(map[refs.Kind]Collector), index: make(map[string]int)}
	for _, col := range r.Collectors {
		c.collectors[col.Kind()] = col
	}
	c.add(r.extract(skill))
	c.wave(ctx, 0)
	extracted := len(c.reports)
	c.add(c.derived())
	c.wave(ctx, extracted)
	return c.reports
}

func (r *Runner) extract(skill *inventory.Skill) []refs.Ref {
	var out []refs.Ref
	for i := range skill.Artifacts {
		a := &skill.Artifacts[i]
		if !wanted(a) {
			continue
		}
		for _, col := range r.Collectors {
			out = append(out, col.Extract(a)...)
		}
	}
	return out
}

func wanted(a *inventory.Artifact) bool {
	if a.Kind != inventory.Content {
		return false
	}
	switch a.ContentType {
	case inventory.Markdown, inventory.Code, inventory.Text, inventory.Data:
		return true
	case inventory.Media, inventory.Unknown, inventory.Symlink, inventory.Special, inventory.Directory:
	}
	return false
}

type coordinator struct {
	runner     *Runner
	collectors map[refs.Kind]Collector
	reports    []refs.Report
	index      map[string]int
	capped     bool
}

func (c *coordinator) add(found []refs.Ref) {
	for _, ref := range found {
		i, ok := c.index[ref.Key()]
		if !ok {
			c.index[ref.Key()] = len(c.reports)
			c.reports = append(c.reports, refs.Report{Ref: ref, Status: refs.NotChecked})
			continue
		}
		for _, loc := range ref.Locations {
			if !slices.Contains(c.reports[i].Ref.Locations, loc) {
				c.reports[i].Ref.Locations = append(c.reports[i].Ref.Locations, loc)
			}
		}
		for _, u := range ref.URLs {
			if !slices.Contains(c.reports[i].Ref.URLs, u) {
				c.reports[i].Ref.URLs = append(c.reports[i].Ref.URLs, u)
			}
		}
	}
}

func (c *coordinator) derived() []refs.Ref {
	var out []refs.Ref
	for _, rep := range c.reports {
		for _, d := range rep.Derived {
			out = append(out, refs.Ref{Kind: d.Kind, Value: d.Value, Locations: slices.Clone(rep.Ref.Locations), Derived: true})
		}
	}
	return out
}

func (c *coordinator) wave(ctx context.Context, from int) {
	limit := c.runner.Config.MaxReferences
	if len(c.reports) > limit && !c.capped {
		c.capped = true
		c.runner.Capped(c.reports[limit].Ref, len(c.reports))
	}
	var pending []int
	for i := from; i < len(c.reports); i++ {
		if i >= limit {
			c.reports[i].Detail = detailCap
			continue
		}
		pending = append(pending, i)
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range max(c.runner.Config.Concurrency, 1) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				c.work(ctx, i)
			}
		}()
	}
	for _, i := range pending {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
}

func (c *coordinator) work(ctx context.Context, i int) {
	ref := c.reports[i].Ref
	var rep refs.Report
	if ctx.Err() != nil {
		rep = notChecked(detailBudget)
	} else {
		rep = c.collectors[ref.Kind].Collect(ctx, ref)
		if rep.Status == refs.Failed && ctx.Err() != nil {
			rep = notChecked(detailBudget)
		}
	}
	rep.Ref = ref
	c.reports[i] = rep
}

func notChecked(detail string) refs.Report {
	return refs.Report{Status: refs.NotChecked, Detail: detail}
}
