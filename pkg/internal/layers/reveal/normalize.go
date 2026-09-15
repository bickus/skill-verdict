// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package reveal

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const (
	invisible = "WORD-INVISIBLE-CHARS"
	lookAlike = "WORD-HOMOGLYPHS"
	spaced    = "WORD-SPACED"
)

type hit struct {
	rule  string
	start int
	end   int
}

type pass struct {
	src     string
	out     strings.Builder
	offsets []int
	hits    []hit
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

func normalize(text string) (string, []int, []hit) {
	p := &pass{src: text, offsets: make([]int, 0, len(text))}
	for i := 0; i < len(text); {
		r, n := utf8.DecodeRuneInString(text[i:])
		switch {
		case dropped(r):
			if joins(prevRune(text, i)) && joins(nextRune(text, i+n)) {
				p.add(invisible, i, i+n)
			}
		case r < utf8.RuneSelf:
			p.write(text[i:i+n], i)
		default:
			p.expand(r, i, n)
		}
		i += n
	}
	return p.out.String(), p.offsets, p.hits
}

func (p *pass) write(s string, at int) {
	p.out.WriteString(s)
	for range len(s) {
		p.offsets = append(p.offsets, at)
	}
}

func (p *pass) expand(r rune, at, size int) {
	folded, mapped := fold(r)
	p.write(mapped, at)
	if unicode.IsLetter(r) && isASCII(mapped) && (isASCII(folded) || mixed(p.src, at, at+size)) {
		p.add(lookAlike, at, at+size)
	}
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

func mixed(text string, start, end int) bool {
	word := text[wordStart(text, start):wordEnd(text, end)]
	return strings.IndexFunc(word, func(r rune) bool { return r < utf8.RuneSelf && unicode.IsLetter(r) }) >= 0
}

func wordStart(text string, i int) int {
	for i > 0 {
		r, n := utf8.DecodeLastRuneInString(text[:i])
		if !joins(r) && !dropped(r) {
			break
		}
		i -= n
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
