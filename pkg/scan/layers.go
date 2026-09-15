// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package scan

import (
	"github.com/bickus/skill-verdict/pkg/internal/layers/discovery"
	"github.com/bickus/skill-verdict/pkg/internal/layers/honeypot"
	"github.com/bickus/skill-verdict/pkg/internal/layers/judge"
	"github.com/bickus/skill-verdict/pkg/internal/layers/reveal"
	"github.com/bickus/skill-verdict/pkg/internal/layers/static"
	"github.com/bickus/skill-verdict/pkg/internal/layers/walk"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
)

func (r *runner) layers() []pipeline.Layer {
	l := r.cfg.Layers
	out := []pipeline.Layer{walk.Layer{Config: l.Walk}}
	if l.Reveal.Enabled {
		out = append(out, reveal.Layer{})
	}
	if l.Static.Enabled {
		out = append(out, static.Layer{})
	}
	if r.refs != nil {
		out = append(out, r.refs)
	}
	if l.Honeypot.Enabled {
		out = append(out, honeypot.Layer{Runner: r.model(l.Honeypot.Calls)})
	}
	if l.Discovery.Enabled {
		out = append(out, discovery.Layer{Runner: r.model(l.Discovery.Calls)})
	}
	if l.Judge.Enabled {
		out = append(out, judge.Layer{Runner: r.model(l.Judge.Calls)})
	}
	return out
}

func (r *runner) model(calls int) llmcall.Runner {
	return llmcall.Runner{Provider: r.provider, LLM: r.cfg.LLM, Calls: calls}
}
