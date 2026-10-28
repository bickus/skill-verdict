// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package reveal

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"github.com/bickus/skill-verdict/pkg/internal/inventory"
)

const (
	invisible = "WORD-INVISIBLE-CHARS"
	lookAlike = "WORD-HOMOGLYPHS"
)

type hit struct {
	rule  string
	start int
	end   int
}

type pass struct {
	src     string
	out     strings.Builder
	shifts  []inventory.Shift
	hits    []hit
	until   int
	swapped bool
}

func needsNormalize(text string) bool {
	for i := range len(text) {
		if b := text[i]; b >= utf8.RuneSelf || control(rune(b)) {
			return true
		}
	}
	return false
}

func control(r rune) bool {
	return (r < 0x20 && r != '\t' && r != '\n' && r != '\r') || r == 0x7F
}

func dropped(r rune) bool {
	if r < utf8.RuneSelf {
		return control(r)
	}
	if unicode.IsSpace(r) {
		return false
	}
	return unicode.Is(unicode.Other_Default_Ignorable_Code_Point, r) ||
		unicode.Is(unicode.Cf, r) ||
		unicode.Is(unicode.Variation_Selector, r)
}

func normalize(text string) *pass {
	p := &pass{src: text}
	for i := 0; i < len(text); {
		r, n := utf8.DecodeRuneInString(text[i:])
		switch {
		case dropped(r):
			if joins(prevRune(text, i)) && joins(nextRune(text, i+n)) {
				p.add(invisible, i, i+n)
			}
		case r < utf8.RuneSelf:
			p.write(text[i:i+n], i, n)
		default:
			p.expand(r, i, n)
		}
		i += n
	}
	return p
}

func (p *pass) write(s string, at, size int) {
	if s == p.src[at:at+size] {
		p.mark(p.out.Len(), at)
	} else {
		for j := range len(s) {
			p.mark(p.out.Len()+j, at)
		}
	}
	p.out.WriteString(s)
}

func (p *pass) mark(clean, source int) {
	shift := 0
	if n := len(p.shifts); n > 0 {
		shift = p.shifts[n-1].Source - p.shifts[n-1].Clean
	}
	if source-clean != shift {
		p.shifts = append(p.shifts, inventory.Shift{Clean: clean, Source: source})
	}
}

func (p *pass) expand(r rune, at, size int) {
	folded, mapped := fold(r)
	switch {
	case !foreign(r):
		p.write(mapped, at, size)
	case p.swap(at):
		p.write(mapped, at, size)
		p.add(lookAlike, at, at+size)
	default:
		p.write(folded, at, size)
	}
}

func (p *pass) swap(at int) bool {
	if at < p.until {
		return p.swapped
	}
	start, end := wordStart(p.src, at), wordEnd(p.src, at)
	if at < start {
		return false
	}
	p.until, p.swapped = end, lookalike(p.src[start:end])
	return p.swapped
}

func (p *pass) add(rule string, start, end int) {
	if n := len(p.hits); n > 0 && p.hits[n-1].rule == rule && p.hits[n-1].end == start {
		p.hits[n-1].end = end
		return
	}
	p.hits = append(p.hits, hit{rule, start, end})
}

func fold(r rune) (string, string) {
	folded := norm.NFKC.String(string(r))
	var b strings.Builder
	for _, out := range folded {
		if ascii, ok := skeleton()[out]; ok {
			b.WriteString(ascii)
		} else {
			b.WriteRune(out)
		}
	}
	return folded, b.String()
}

func concealed(r rune) bool {
	if r < utf8.RuneSelf || !unicode.IsLetter(r) {
		return false
	}
	_, mapped := fold(r)
	return isASCII(mapped)
}

func foreign(r rune) bool {
	return unicode.IsLetter(r) && !unicode.In(r, unicode.Latin, unicode.Common, unicode.Inherited)
}

func lookalike(word string) bool {
	latin := false
	for _, r := range word {
		switch {
		case foreign(r):
			if !concealed(r) {
				return false
			}
		case unicode.Is(unicode.Latin, r):
			latin = true
		}
	}
	return latin
}

func wordStart(text string, i int) int {
	for i > 0 {
		r, n := utf8.DecodeLastRuneInString(text[:i])
		if !joins(r) && !dropped(r) {
			break
		}
		i -= n
	}
	if i > 0 && text[i-1] == '\\' {
		_, n := utf8.DecodeRuneInString(text[i:])
		i += n
	}
	return i
}

func wordEnd(text string, i int) int {
	for i < len(text) {
		r, n := utf8.DecodeRuneInString(text[i:])
		if !joins(r) && !dropped(r) {
			break
		}
		i += n
	}
	return i
}

func prevRune(text string, i int) rune {
	for i > 0 {
		r, n := utf8.DecodeLastRuneInString(text[:i])
		if !dropped(r) {
			return r
		}
		i -= n
	}
	return utf8.RuneError
}

func nextRune(text string, i int) rune {
	for i < len(text) {
		r, n := utf8.DecodeRuneInString(text[i:])
		if !dropped(r) {
			return r
		}
		i += n
	}
	return utf8.RuneError
}

func joins(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isASCII(s string) bool {
	for i := range len(s) {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
