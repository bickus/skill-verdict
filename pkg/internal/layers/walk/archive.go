// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package walk

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
)

var (
	sevenZipMagic = []byte{'7', 'z', 0xBC, 0xAF, 0x27, 0x1C}
	aesCoder      = []byte{0x06, 0xF1, 0x07, 0x01}
)

func encryptedArchive(raw []byte) string {
	if bytes.HasPrefix(raw, sevenZipMagic) {
		if sevenZipEncrypted(raw) {
			return "7z"
		}
		return ""
	}
	if zipEncrypted(raw) {
		return "zip"
	}
	return ""
}

func zipEncrypted(raw []byte) bool {
	r, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return false
	}
	for _, f := range r.File {
		if f.Flags&0x1 != 0 {
			return true
		}
	}
	return false
}

func sevenZipEncrypted(raw []byte) bool {
	const start = 32
	if len(raw) < start {
		return false
	}
	total := uint64(len(raw))
	offset := binary.LittleEndian.Uint64(raw[12:20])
	size := binary.LittleEndian.Uint64(raw[20:28])
	end := start + offset + size
	if end < offset || end > total {
		return false
	}
	return bytes.Contains(raw[start+offset:end], aesCoder)
}
