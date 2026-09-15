// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package llmcall_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
)

func skill() *inventory.Skill {
	s := &inventory.Skill{Name: "s", Artifacts: []inventory.Artifact{
		{Path: "SKILL.md", Kind: inventory.Content, ContentType: inventory.Markdown, ReadStatus: inventory.ReadOK, Size: 6, Content: "a\nb\nc\n"},
		{Path: "data.json", Kind: inventory.Content, ContentType: inventory.Data, ReadStatus: inventory.ReadOK, Size: 7, Content: `{"a":1}`},
		{Path: "empty.txt", Kind: inventory.Content, ContentType: inventory.Text, ReadStatus: inventory.ReadOK},
		{Path: "link", Kind: inventory.Content, ContentType: inventory.Symlink, ReadStatus: inventory.ReadOK, Content: "../target"},
		{Path: "logo.png", Kind: inventory.Content, ContentType: inventory.Media, ReadStatus: inventory.ReadOK, Size: 123},
		{Path: "blob", Kind: inventory.Content, ContentType: inventory.Unknown, ReadStatus: inventory.ReadOK, Size: 9},
		{Path: "sock", Kind: inventory.Content, ContentType: inventory.Special, ReadStatus: inventory.ReadOK},
		{Path: "dir/", Kind: inventory.Content, ContentType: inventory.Directory, ReadStatus: inventory.ReadFailed, ScanNotes: []string{"permission denied"}},
	}}
	for i := range s.Artifacts {
		s.Artifacts[i].Origin = inventory.Original
	}
	return s
}

func TestReadCall(t *testing.T) {
	cases := []struct {
		name     string
		args     string
		maxChars int
		want     string
		wantErr  string
	}{
		{"whole file numbered", `{"path":"SKILL.md"}`, 1000, "## File: SKILL.md (lines 1-3 of 3)\nL1: a\nL2: b\nL3: c\n", ""},
		{"offset and limit", `{"path":"SKILL.md","offset":2,"limit":1}`, 1000,
			"## File: SKILL.md (lines 2-2 of 3)\nLines after 2 were not returned; call read again with offset 3.\nL2: b\n", ""},
		{"cut at the size cap", `{"path":"SKILL.md"}`, 6,
			"## File: SKILL.md (lines 1-1 of 3)\nLines after 1 were not returned; call read again with offset 2.\nL1: a\n", ""},
		{"data file", `{"path":"data.json"}`, 1000, "## File: data.json (lines 1-1 of 1)\nL1: {\"a\":1}\n", ""},
		{"empty file", `{"path":"empty.txt"}`, 1000, "## File: empty.txt (empty)\n", ""},
		{"symlink target", `{"path":"link"}`, 1000, "## Symlink: link -> ../target\n", ""},
		{"media has no text", `{"path":"logo.png"}`, 1000, "", "logo.png: media content of 123 bytes, no text is available"},
		{"unknown bytes have no text", `{"path":"blob"}`, 1000, "", "blob: unknown content of 9 bytes"},
		{"special file", `{"path":"sock"}`, 1000, "", "sock: a special, it has no content"},
		{"unlisted directory", `{"path":"dir/"}`, 1000, "", "dir/: the scanner did not read it (failed): permission denied"},
		{"unknown path", `{"path":"../etc/passwd"}`, 1000, "", `no artifact at "../etc/passwd"`},
		{"offset past the end", `{"path":"SKILL.md","offset":9}`, 1000, "", "SKILL.md has 3 lines, offset 9 is past the end"},
		{"bad arguments", `{"path":1}`, 1000, "", "bad arguments"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := llmcall.Read{Skill: skill(), MaxChars: tc.maxChars}
			got, err := r.Call(context.Background(), json.RawMessage(tc.args))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err %v, want one containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}
