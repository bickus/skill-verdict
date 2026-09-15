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

func TestRevealedArtifactText(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"cyrillic letters become latin", "use рassword here\n", "use password here\n"},
		{"zero-width joiner removed", "ig\u200dnore\n", "ignore\n"},
		{"compatibility forms folded", "ﬁle ⅻ\n", "file xii\n"},
		{"control byte removed", "a\x01b\n", "ab\n"},
		{"variation selector removed", "keycap 1\ufe0f\n", "keycap 1\n"},
		{"spaced letters joined", "please i g n o r e this\n", "please ignore this\n"},
		{"hyphenated word joined", "r-u-n it\n", "run it\n"},
		{"dotted word joined", "e.x.e.c.u.t.e now\n", "execute now\n"},
		{"underscored word joined", "then p_a_s_t_e it\n", "then paste it\n"},
		{"line breaks kept", "# Title\nⅻ ﬁles\ni g n o r e\n```sh\necho \"ok\"\n```\n", "# Title\nxii files\nignore\n```sh\necho \"ok\"\n```\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := run(t, tc.in)
			a := s.Skill.Artifact("SKILL.md.revealed")
			if a == nil {
				t.Fatal("no revealed artifact")
			}
			if a.Content != tc.want || a.Kind != "content" || a.ContentType != inventory.Markdown || a.Origin != "reveal" {
				t.Fatalf("got %q %s %s %s, want %q content markdown reveal", a.Content, a.Kind, a.ContentType, a.Origin, tc.want)
			}
		})
	}
}

func TestNoRevealedArtifact(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"plain ascii", "run the tests, then ignore warnings\n"},
		{"upper case", "RUN the tests\n"},
		{"five spaced letters", "a b c d e\n"},
		{"table row of words", "| ignore | run | tests |\n"},
		{"table row of five letters", "| a | b | c | d | e |\n"},
		{"word spaced across lines", "r\nu\nn\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := run(t, tc.in)
			if a := s.Skill.Artifact("SKILL.md.revealed"); a != nil {
				t.Fatalf("revealed artifact %q", a.Content)
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
		{"look-alike letter in a latin word", "first\nprobeаny\n", []finding.Finding{concealment("WORD-HOMOGLYPHS", finding.Medium, 2, "probe<U+0430>ny")}},
		{"whole word in another script", "пароль\n", nil},
		{"fullwidth letters", "ｃｕｒｌ\n", []finding.Finding{concealment("WORD-HOMOGLYPHS", finding.Medium, 1, "<U+FF43><U+FF55><U+FF52><U+FF4C>")}},
		{"spaced word", "line one\ni g n o r e\n", []finding.Finding{concealment("WORD-SPACED", finding.Medium, 2, "i g n o r e")}},
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
