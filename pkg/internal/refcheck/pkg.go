// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/refs"
)

const (
	packageStats = "https://packages.ecosyste.ms/api/v1/registries/"
	npmRegistry  = "https://registry.npmjs.org/"
	pypiRegistry = "https://pypi.org/pypi/"
)

type PackageCollector struct {
	fetch fetcher
	cfg   config.ReferencesConfig
}

func NewPackage(client *http.Client, cfg config.ReferencesConfig) *PackageCollector {
	return &PackageCollector{fetch: newFetcher(client, cfg), cfg: cfg}
}

func (*PackageCollector) Kind() refs.Kind {
	return refs.Package
}

func (*PackageCollector) Extract(a *inventory.Artifact) []refs.Ref {
	return packageRefs(a)
}

func (p *PackageCollector) Collect(ctx context.Context, ref refs.Ref) refs.Report {
	rep := refs.Report{Ref: ref, Status: refs.Checked}
	registry, name, _ := strings.Cut(ref.Value, ":")
	if err := p.lookup(ctx, registry, name, &rep); err != nil {
		fail(&rep, err)
	}
	if popular, ok := typosquat(registry, name); ok {
		addFact(&rep, "typosquat_of", popular)
		addFinding(&rep, "PACKAGE-TYPOSQUAT")
	}
	return rep
}

func (p *PackageCollector) lookup(ctx context.Context, registry, name string, rep *refs.Report) error {
	var host string
	switch registry {
	case registryNPM:
		host = "npmjs.org"
	case registryPyPI:
		host = "pypi.org"
	default:
		return fmt.Errorf("unknown registry %q", registry)
	}
	found, err := p.stats(ctx, host, name, rep)
	if found {
		return nil
	}
	if err != nil {
		addFact(rep, "monthly_downloads", "unknown")
	}
	if registry == registryNPM {
		return p.npm(ctx, name, rep)
	}
	return p.pypi(ctx, name, rep)
}

func (p *PackageCollector) stats(ctx context.Context, host, name string, rep *refs.Report) (bool, error) {
	var meta struct {
		Downloads *int   `json:"downloads"`
		Period    string `json:"downloads_period"`
		First     string `json:"first_release_published_at"`
		Version   string `json:"latest_release_number"`
	}
	found, err := p.fetch.getJSON(ctx, packageStats+host+"/packages/"+url.PathEscape(name), nil, &meta)
	if err != nil || !found {
		return false, err
	}
	if first, perr := time.Parse(time.RFC3339, meta.First); perr == nil {
		p.published(rep, first)
	}
	if meta.Version != "" {
		addFact(rep, "version", meta.Version)
	}
	if meta.Downloads != nil && meta.Period == "last-month" {
		p.downloads(rep, *meta.Downloads)
	}
	return true, nil
}

func missing(rep *refs.Report) {
	addFact(rep, "published", "false")
	addFinding(rep, "PACKAGE-MISSING")
}

func (p *PackageCollector) npm(ctx context.Context, name string, rep *refs.Report) error {
	var meta struct {
		Version string `json:"version"`
	}
	found, err := p.fetch.getJSON(ctx, npmRegistry+name+"/latest", nil, &meta)
	if err != nil {
		return err
	}
	if !found {
		missing(rep)
		return nil
	}
	if meta.Version != "" {
		addFact(rep, "version", meta.Version)
	}
	return nil
}

func (p *PackageCollector) pypi(ctx context.Context, name string, rep *refs.Report) error {
	var meta struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
		Releases map[string][]struct {
			Upload string `json:"upload_time_iso_8601"`
		} `json:"releases"`
	}
	found, err := p.fetch.getJSON(ctx, pypiRegistry+name+"/json", nil, &meta)
	if err != nil {
		return err
	}
	if !found {
		missing(rep)
		return nil
	}
	var first time.Time
	for _, files := range meta.Releases {
		for _, f := range files {
			if t, perr := time.Parse(time.RFC3339, f.Upload); perr == nil && (first.IsZero() || t.Before(first)) {
				first = t
			}
		}
	}
	if !first.IsZero() {
		p.published(rep, first)
	}
	if meta.Info.Version != "" {
		addFact(rep, "version", meta.Info.Version)
	}
	return nil
}

func (p *PackageCollector) published(rep *refs.Report, t time.Time) {
	age := ageDays(t)
	addFact(rep, "first_published", t.Format(time.DateOnly))
	addFact(rep, "age_days", strconv.Itoa(age))
	if age < p.cfg.NewPackageDays {
		addFinding(rep, "PACKAGE-NEW")
	}
}

func (p *PackageCollector) downloads(rep *refs.Report, n int) {
	addFact(rep, "monthly_downloads", strconv.Itoa(n))
	if n < p.cfg.LowDownloads {
		addFinding(rep, "PACKAGE-LOW-DOWNLOADS")
	}
}
