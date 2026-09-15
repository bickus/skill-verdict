// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package scan

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/layers/references"
	"github.com/bickus/skill-verdict/pkg/internal/layers/walk"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/llm"
	"github.com/bickus/skill-verdict/pkg/llm/chatgpt"
	"github.com/bickus/skill-verdict/pkg/llm/openai"
	"github.com/bickus/skill-verdict/pkg/rules"
)

type Options struct {
	Config   config.Config
	Rules    *rules.Set
	Provider llm.Provider
	Client   *http.Client
	Each     func(Result) error
}

type StoppedError struct {
	Skipped int
}

func (e *StoppedError) Error() string {
	return fmt.Sprintf("scan stopped, %d skills skipped", e.Skipped)
}

type runner struct {
	cfg      config.Config
	set      *rules.Set
	refs     pipeline.Layer
	provider llm.Provider
}

func Run(ctx context.Context, paths []string, opts Options) ([]Result, error) {
	targets, err := targets(paths)
	if err != nil {
		return nil, err
	}
	if opts.Rules == nil {
		return nil, errors.New("scan: no rule set")
	}
	cfg := opts.Config
	if err = cfg.Validate(); err != nil {
		return nil, err
	}
	p, err := provider(opts, cfg)
	if err != nil {
		return nil, err
	}
	r := &runner{cfg: cfg, set: opts.Rules, provider: p}
	if refs := cfg.Layers.References; refs.Enabled {
		r.refs = references.New(client(opts), refs, token(refs))
	}
	return r.run(ctx, targets, opts.Each)
}

func (r *runner) run(ctx context.Context, targets []string, each func(Result) error) ([]Result, error) {
	results := make([]Result, 0, len(targets))
	for i, target := range targets {
		res, err := r.scan(ctx, target)
		if ctx.Err() != nil {
			return results, &StoppedError{Skipped: len(targets) - i}
		}
		if err != nil {
			return nil, err
		}
		results = append(results, res)
		if each != nil {
			if err = each(res); err != nil {
				return nil, err
			}
		}
	}
	return results, nil
}

func (r *runner) scan(ctx context.Context, target string) (Result, error) {
	root, err := walk.Resolve(target)
	if err != nil {
		return Result{}, err
	}
	s := &pipeline.State{Skill: &inventory.Skill{Root: root}, Rules: r.set}
	pipeline.Run(ctx, s, r.layers())
	return result(s, r.cfg), nil
}

func provider(opts Options, cfg config.Config) (llm.Provider, error) {
	if opts.Provider != nil || !cfg.Layers.ModelEnabled() {
		return opts.Provider, nil
	}
	key, err := cfg.APIKey()
	if err != nil {
		return nil, err
	}
	if cfg.LLM.Provider == config.ChatGPTSubscription {
		return chatgpt.New(cfg.LLM, key, &http.Client{}), nil
	}
	return openai.New(cfg.LLM, key, &http.Client{}), nil
}

func client(opts Options) *http.Client {
	if opts.Client != nil {
		return opts.Client
	}
	return &http.Client{Transport: &http.Transport{
		DialContext:         refcheck.SafeDialer().DialContext,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 10 * time.Second,
		MaxIdleConns:        16,
		IdleConnTimeout:     30 * time.Second,
	}}
}

func token(cfg config.ReferencesConfig) string {
	if cfg.GitHubTokenEnv == "" {
		return ""
	}
	return os.Getenv(cfg.GitHubTokenEnv)
}

func targets(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, errors.New("scan: no path")
	}
	var all []string
	for _, path := range paths {
		found, err := skills(path)
		if err != nil {
			return nil, err
		}
		all = append(all, found...)
	}
	return all, nil
}

func skills(path string) ([]string, error) {
	if isSkill(path) {
		return []string{path}, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var found []string
	for _, e := range entries {
		if child := filepath.Join(path, e.Name()); isSkill(child) {
			found = append(found, child)
		}
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("%s: no SKILL.md found", path)
	}
	return found, nil
}

func isSkill(dir string) bool {
	_, err := walk.Resolve(dir)
	return err == nil
}
