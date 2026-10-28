// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	"regexp"
	"strings"
)

var registryMoved = regexp.MustCompile(`(?i)--registry|--site|CLAWHUB_REGISTRY|CLAWHUB_SITE`)

type SelfInstall struct{ re *regexp.Regexp }

func NewSelfInstall(folder string) SelfInstall {
	if folder == "" {
		return SelfInstall{}
	}
	name := regexp.QuoteMeta(strings.ToLower(folder))
	return SelfInstall{re: regexp.MustCompile("(?i)(?:^|[\\s;&|(`\"'])((?:npx\\s+(?:-y\\s+)?)?(?:claw[d]?hub(?:@\\S+)?|openclaw\\s+(?:skills?|plugins?))\\s+install\\s+(?:--?[\\w-]+\\s+)*(?:clawhub:)?" + name + ")(?:[\\s`\"'),;&|]|$)")}
}

func (s SelfInstall) Spans(line string) [][]int {
	if s.re == nil || registryMoved.MatchString(line) {
		return nil
	}
	var out [][]int
	for _, m := range s.re.FindAllStringSubmatchIndex(line, -1) {
		out = append(out, m[2:4])
	}
	return out
}
