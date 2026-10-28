// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package reveal_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/layers/reveal"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/rules"
)

func run(t *testing.T, text string) *pipeline.State {
	t.Helper()
	set, err := rules.Load("", config.Default().Layers.Rules())
	if err != nil {
		t.Fatal(err)
	}
	s := &pipeline.State{
		Skill: &inventory.Skill{Artifacts: []inventory.Artifact{
			{Path: "SKILL.md", Kind: inventory.Content, Origin: inventory.Original, ContentType: inventory.Markdown, ReadStatus: inventory.ReadOK, Size: int64(len(text)), Content: text},
		}},
		Rules: set,
	}
	pipeline.Run(context.Background(), s, []pipeline.Layer{reveal.Layer{}})
	return s
}

func TestCleanText(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"cyrillic letter in a latin word becomes latin", "use рassword here\n", "use password here\n"},
		{"zero-width joiner removed", "ig\u200dnore\n", "ignore\n"},
		{"compatibility forms folded", "ﬁle ⅻ\n", "file xii\n"},
		{"control byte removed", "a\x01b\n", "ab\n"},
		{"variation selector removed", "keycap 1\ufe0f\n", "keycap 1\n"},
		{"line breaks kept", "# Title\nⅻ ﬁles\n```sh\necho \"ok\"\n```\n", "# Title\nxii files\n```sh\necho \"ok\"\n```\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := run(t, tc.in)
			if got := s.Skill.Artifact("SKILL.md").Text(); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTextLeftAsWritten(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"plain ascii", "run the tests, then ignore warnings\n"},
		{"upper case", "RUN the tests\n"},
		{"spaced letters", "please i g n o r e this\n"},
		{"table row of words", "| ignore | run | tests |\n"},
		{"whole word in another script", "пароль\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := run(t, tc.in)
			if got := s.Skill.Artifact("SKILL.md").Text(); got != tc.in {
				t.Fatalf("got %q, want %q", got, tc.in)
			}
			if len(s.Findings) != 0 {
				t.Fatalf("findings %+v", s.Findings)
			}
		})
	}
}

func concealment(rule string, severity finding.Severity, line int, evidence string) finding.Finding {
	return finding.Finding{Rule: rule, Severity: severity,
		Location: finding.Location{File: "SKILL.md", Line: line}, Evidence: evidence}
}

func TestConcealmentFindings(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []finding.Finding
	}{
		{"invisible inside a word", "ig\u200dnore\n", []finding.Finding{concealment("WORD-INVISIBLE-CHARS", finding.High, 1, "ig<U+200D>nore")}},
		{"invisible between emoji", "👨\u200d👩\n", nil},
		{"invisible before a space", "run\u200b now\n", nil},
		{"look-alike letter in a latin word", "first\nprobeаny\n", []finding.Finding{concealment("WORD-HOMOGLYPHS", finding.Medium, 2, "probe<U+0430>ny reads as probeany")}},
		{"whole word in another script", "пароль\n", nil},
		{"fullwidth letters", "ｃｕｒｌ\n", nil},
		{"latin letter of another alphabet", "yakın\n", nil},
		{"other script letter that does not print like latin", "bслово\n", nil},
		{"escape sequence before another script", "line\\nΑΒ\n", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := run(t, tc.in)
			if !reflect.DeepEqual(s.Findings, tc.want) {
				t.Fatalf("got %+v, want %+v", s.Findings, tc.want)
			}
		})
	}
}
