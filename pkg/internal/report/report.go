// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/refs"
	"github.com/bickus/skill-verdict/pkg/rules"
	"github.com/bickus/skill-verdict/pkg/scan"
)

const (
	descriptionLabel = "Description"
	locationLabel    = "Location"
	analysisLabel    = "Analysis"
	verdictLabel     = "Judge"
	modelLabel       = "Model"
	costLabel        = "Cost"
	unpriced         = "n/a"
	labelWidth       = max(len(descriptionLabel), len(locationLabel), len(analysisLabel), len(verdictLabel))
)

func Skill(w io.Writer, r scan.Result, byID map[string]rules.Rule, p Layout, gap bool) error {
	if p.Width < 1 {
		p.Width = MaxWidth
	}
	var b strings.Builder
	if gap {
		b.WriteString("\n")
	}
	skill(&b, p, byID, r)
	_, err := io.WriteString(w, b.String())
	return err
}

func Summary(w io.Writer, results []scan.Result, skipped int, p Layout) error {
	if p.Width < 1 {
		p.Width = MaxWidth
	}
	var b strings.Builder
	summary(&b, p, results, skipped)
	if t := totals(results); t.Model != nil {
		usage(&b, p, t)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func skill(b *strings.Builder, p Layout, byID map[string]rules.Rule, r scan.Result) {
	b.WriteString(ruled(p, heading(p, r)))
	keyed(b, p, locationLabel, piece{"", safe(r.Skill.Path)})
	for _, l := range r.Layers {
		if l.Interrupt != "" {
			line(b, p, yellow, fmt.Sprintf("interrupted by %s in %s", safe(l.Interrupt), safe(l.Layer)))
		}
	}
	references(b, p, r.References)
	findings(b, p, byID, r)
	if r.Verdict == scan.Incomplete {
		incomplete(b, p, r)
	}
	usage(b, p, r)
}

func heading(p Layout, r scan.Result) cell {
	v := string(r.Verdict)
	name := cut(safe(r.Skill.Name), p.Width-7-count(v))
	return cell{
		p.paint(bold, name) + p.paint(dim, " : ") + p.paint(verdictCode(r.Verdict), v),
		name + " : " + v,
	}
}

func section(b *strings.Builder, p Layout, name string) {
	b.WriteString("\n")
	b.WriteString(p.paint(bold, strings.ToUpper(name)))
	b.WriteString("\n")
}

func references(b *strings.Builder, p Layout, reports []refs.Report) {
	if len(reports) == 0 {
		return
	}
	section(b, p, "References")
	for _, r := range reports {
		row := []piece{{"", safe(r.Ref.Key()) + sep}, {statusCode(r.Status), string(r.Status)}}
		if detail := safe(r.Detail); detail != "" {
			row = append(row, piece{"", sep + detail})
		}
		pieces(b, p, row...)
	}
}

func findings(b *strings.Builder, p Layout, byID map[string]rules.Rule, r scan.Result) {
	section(b, p, "Findings")
	if len(r.Findings) == 0 && len(r.Dismissed) == 0 {
		line(b, p, "", "no findings")
		return
	}
	for i, f := range r.Findings {
		one(b, p, byID[f.Rule], f, piece{severityCode(f.Severity), string(f.Severity)}, i > 0)
	}
	for i, f := range r.Dismissed {
		one(b, p, byID[f.Rule], f, piece{cyan, string(finding.Dismissed)}, i+len(r.Findings) > 0)
	}
}

func one(b *strings.Builder, p Layout, r rules.Rule, f finding.Finding, mark piece, gap bool) {
	if gap {
		b.WriteString("\n")
	}
	head := []piece{mark, {"", sep + safe(f.Rule)}}
	if title := safe(r.Title); title != "" {
		head = append(head, piece{"", sep + title})
	}
	if category := safe(r.Category); category != "" {
		head = append(head, piece{"", sep + category})
	}
	wrap(b, p, labelWidth+2, head...)
	field(b, p, descriptionLabel, piece{"", safe(r.Description)})
	field(b, p, locationLabel, located(f, r.Kind)...)
	if r.Kind == finding.KindLLM && f.Judgement == nil {
		field(b, p, analysisLabel, piece{"", safe(f.Evidence)})
	}
	if f.Judgement != nil {
		field(b, p, verdictLabel, judged(f.Judgement)...)
	}
}

func located(f finding.Finding, kind finding.Kind) []piece {
	out := []piece{{"", safe(location(f.Location))}}
	if kind == finding.KindLLM {
		return out
	}
	if evidence := safe(f.Evidence); evidence != "" {
		out = append(out, piece{"", sep + evidence})
	}
	return out
}

func judged(j *finding.Judgement) []piece {
	out := []piece{{"", string(j.State)}}
	if reason := safe(j.Reason); reason != "" {
		out = append(out, piece{"", sep + reason})
	}
	return out
}

func location(loc finding.Location) string {
	if loc.Line == 0 {
		return loc.File
	}
	return fmt.Sprintf("%s:%d", loc.File, loc.Line)
}

func usage(b *strings.Builder, p Layout, r scan.Result) {
	if r.Model == nil {
		return
	}
	section(b, p, "Model usage")
	model := []piece{{"", safe(r.Model.Name)}}
	if effort := safe(r.Model.Effort); effort != "" {
		model = append(model, piece{"", sep + effort})
	}
	keyed(b, p, modelLabel, model...)
	var parts []string
	for _, l := range r.Layers {
		if l.Tokens != nil {
			parts = append(parts, fmt.Sprintf("%s %d requests", l.Layer, l.Tokens.Requests))
		}
	}
	cost := unpriced
	if r.Cost > 0 {
		cost = fmt.Sprintf("$%.4f", r.Cost)
	}
	keyed(b, p, costLabel, piece{"", cost + sep + strings.Join(parts, ", ")})
}

func incomplete(b *strings.Builder, p Layout, r scan.Result) {
	section(b, p, "Incomplete")
	for _, a := range r.Artifacts {
		if a.Inspected() {
			continue
		}
		entry(b, p, "artifact", fmt.Sprintf("%s  %s %s, %s", safe(a.Path), a.ReadStatus,
			a.ContentType, safe(strings.Join(a.Notes, "; "))))
	}
	for _, rep := range r.References {
		if rep.Status == refs.Checked {
			continue
		}
		entry(b, p, "reference", fmt.Sprintf("%s  %s, %s", safe(rep.Ref.Key()), rep.Status,
			safe(rep.Detail)))
	}
	for _, l := range r.Layers {
		if l.Inspected() {
			continue
		}
		layer(b, p, l)
	}
}

func layer(b *strings.Builder, p Layout, l scan.LayerStatus) {
	text := fmt.Sprintf("%s  %s", l.Layer, l.Status)
	if l.Error != "" {
		text += ", " + safe(l.Error)
	}
	entry(b, p, "layer", text)
	for _, e := range l.Errors {
		line(b, p, "", safe(e.Artifact+sep+e.Error))
	}
}

func entry(b *strings.Builder, p Layout, kind, detail string) {
	pieces(b, p, piece{"", kind + sep + detail})
}

func summary(b *strings.Builder, p Layout, results []scan.Result, skipped int) {
	counts := make(map[scan.Verdict]int)
	for _, r := range results {
		counts[r.Verdict]++
	}
	var shown, plain []string
	for _, v := range []scan.Verdict{scan.Block, scan.Incomplete, scan.Review, scan.Clean} {
		if counts[v] == 0 {
			continue
		}
		s := fmt.Sprintf("%d %s", counts[v], v)
		shown = append(shown, p.paint(verdictCode(v), s))
		plain = append(plain, s)
	}
	if skipped > 0 {
		s := fmt.Sprintf("%d skipped", skipped)
		shown = append(shown, s)
		plain = append(plain, s)
	}
	left := fmt.Sprintf("%d skills", len(results)+skipped)
	b.WriteString("\n")
	b.WriteString(ruled(p, cell{
		p.paint(bold, left) + p.paint(dim, " : ") + strings.Join(shown, ", "),
		left + " : " + strings.Join(plain, ", "),
	}))
}

func totals(results []scan.Result) scan.Result {
	var t scan.Result
	index := map[string]int{}
	for _, r := range results {
		if r.Model == nil {
			continue
		}
		if t.Model == nil {
			t.Model = r.Model
		}
		t.Cost += r.Cost
		for _, l := range r.Layers {
			if l.Tokens == nil {
				continue
			}
			i, ok := index[l.Layer]
			if !ok {
				i = len(t.Layers)
				index[l.Layer] = i
				t.Layers = append(t.Layers, scan.LayerStatus{Layer: l.Layer, Tokens: &scan.Tokens{}})
			}
			t.Layers[i].Tokens.Requests += l.Tokens.Requests
		}
	}
	return t
}
