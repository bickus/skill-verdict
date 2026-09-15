// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package inventory

import (
	"bytes"
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"
	"unicode/utf8"
)

type Kind string

const (
	Content Kind = "content"
	Brief   Kind = "brief"
)

type ContentType string

const (
	Markdown  ContentType = "markdown"
	Code      ContentType = "code"
	Text      ContentType = "text"
	Data      ContentType = "data"
	Media     ContentType = "media"
	Unknown   ContentType = "unknown"
	Symlink   ContentType = "symlink"
	Special   ContentType = "special"
	Directory ContentType = "directory"
)

type ReadStatus string

const (
	ReadOK     ReadStatus = "ok"
	ReadFailed ReadStatus = "failed"
)

type Origin string

type OriginInfo struct {
	Name        Origin
	Description string
}

var origins []OriginInfo

func RegisterOrigin(name Origin, description string) Origin {
	if slices.ContainsFunc(origins, func(o OriginInfo) bool { return o.Name == name }) {
		panic("inventory: origin " + string(name) + " registered twice")
	}
	origins = append(origins, OriginInfo{Name: name, Description: description})
	return name
}

func Origins() []OriginInfo {
	return slices.Clone(origins)
}

var Original = RegisterOrigin("original", "Original raw item from the skill bundle")

const revealedSuffix = ".revealed"

type Artifact struct {
	Path        string
	Kind        Kind
	Origin      Origin
	ContentType ContentType
	ReadStatus  ReadStatus
	Size        int64
	Content     string
	ScanNotes   []string
}

func (a *Artifact) Note(text string) {
	a.ScanNotes = append(a.ScanNotes, text)
}

func Regular(rel string, raw []byte) Artifact {
	ctype, media := classify(rel, raw[:min(len(raw), sniffLen)])
	a := Artifact{Path: rel, Kind: Content, Origin: Original, ContentType: ctype, ReadStatus: ReadOK, Size: int64(len(raw))}
	if ctype == Media || ctype == Unknown {
		a.Note("type: " + media)
		return a
	}
	text, replaced := decode(raw)
	a.Content = text
	if replaced > 0 {
		a.Note(fmt.Sprintf("utf8: %d invalid bytes replaced", replaced))
	}
	return a
}

func RevealedPath(path string) string {
	return path + revealedSuffix
}

func OriginalPath(path string) string {
	return strings.TrimSuffix(path, revealedSuffix)
}

const sniffLen = 8192

func classify(rel string, head []byte) (ContentType, string) {
	media, _, _ := strings.Cut(http.DetectContentType(head), ";")
	media = strings.TrimSpace(media)
	if !bytes.ContainsRune(head, 0) && textual(media) {
		return contentTypeOf(rel, head), media
	}
	if mediaFile(media) {
		return Media, media
	}
	return Unknown, media
}

func textual(media string) bool {
	return strings.HasPrefix(media, "text/") || media == "application/json" || media == "application/xml"
}

func mediaFile(media string) bool {
	for _, prefix := range []string{"image/", "font/", "audio/", "video/"} {
		if strings.HasPrefix(media, prefix) {
			return true
		}
	}
	return media == "application/pdf"
}

func contentTypeOf(rel string, head []byte) ContentType {
	base := path.Base(rel)
	switch strings.ToLower(path.Ext(base)) {
	case ".md", ".markdown":
		return Markdown
	case ".txt", ".rst":
		return Text
	case ".json", ".yaml", ".yml", ".toml", ".xml", ".xsd", ".csv":
		return Data
	case ".py", ".sh", ".bash", ".zsh", ".js", ".mjs", ".cjs", ".ts", ".tsx", ".rb", ".go", ".rs",
		".pl", ".php", ".ps1", ".bat", ".cmd":
		return Code
	case "":
		if bytes.HasPrefix(head, []byte("#!")) || base == "Makefile" || base == "Dockerfile" {
			return Code
		}
	}
	return Text
}

func decode(raw []byte) (string, int) {
	if utf8.Valid(raw) {
		return string(raw), 0
	}
	var b strings.Builder
	b.Grow(len(raw))
	replaced := 0
	for i := 0; i < len(raw); {
		r, n := utf8.DecodeRune(raw[i:])
		if r == utf8.RuneError && n == 1 {
			replaced++
		}
		b.WriteRune(r)
		i += n
	}
	return b.String(), replaced
}
