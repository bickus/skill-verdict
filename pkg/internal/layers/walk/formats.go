// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package walk

import (
	"bytes"
	"encoding/binary"
)

const (
	firstJavaVersion  = 45
	firstPython3Magic = 3000
	lastPython3Magic  = 3999
)

type signature struct {
	rule   string
	format string
	match  func([]byte) bool
}

var signatures = []signature{
	{installer, "Debian package", prefix("!<arch>\ndebian")},
	{installer, "RPM package", prefix("\xed\xab\xee\xdb")},
	{installer, "Windows cabinet", prefix("MSCF\x00\x00\x00\x00")},
	{installer, "macOS installer package", prefix("xar!")},
	{installer, "SquashFS image", prefix("hsqs", "sqsh")},
	{executable, "ELF binary", prefix("\x7fELF")},
	{executable, "Windows PE binary", portableExecutable},
	{executable, "Mach-O binary", prefix("\xfe\xed\xfa\xce", "\xfe\xed\xfa\xcf", "\xce\xfa\xed\xfe", "\xcf\xfa\xed\xfe")},
	{executable, "Mach-O universal binary", universalBinary},
	{executable, "Java class", javaClass},
	{executable, "static library", prefix("!<arch>\n")},
	{executable, "WebAssembly module", prefix("\x00asm")},
	{executable, "Python bytecode", pythonBytecode},
	{executable, "Android DEX bytecode", prefix("dex\n0")},
	{executable, "Lua bytecode", prefix("\x1bLua", "\x1bLJ")},
	{executable, "Erlang BEAM bytecode", erlangBeam},
	{executable, "Ruby YARV bytecode", prefix("YARB")},
}

func detect(raw []byte) (signature, bool) {
	for _, sig := range signatures {
		if sig.match(raw) {
			return sig, true
		}
	}
	return signature{}, false
}

func prefix(magic ...string) func([]byte) bool {
	return func(raw []byte) bool {
		for _, m := range magic {
			if bytes.HasPrefix(raw, []byte(m)) {
				return true
			}
		}
		return false
	}
}

func portableExecutable(raw []byte) bool {
	if len(raw) < 64 || !bytes.HasPrefix(raw, []byte("MZ")) {
		return false
	}
	offset := uint64(binary.LittleEndian.Uint32(raw[60:64]))
	return offset+4 <= uint64(len(raw)) && string(raw[offset:offset+4]) == "PE\x00\x00"
}

func universalBinary(raw []byte) bool {
	return cafebabe(raw) && binary.BigEndian.Uint32(raw[4:8]) < firstJavaVersion
}

func javaClass(raw []byte) bool {
	return cafebabe(raw) && binary.BigEndian.Uint32(raw[4:8]) >= firstJavaVersion
}

func cafebabe(raw []byte) bool {
	return len(raw) >= 8 && bytes.HasPrefix(raw, []byte("\xca\xfe\xba\xbe"))
}

func erlangBeam(raw []byte) bool {
	return len(raw) >= 12 && bytes.HasPrefix(raw, []byte("FOR1")) && string(raw[8:12]) == "BEAM"
}

func pythonBytecode(raw []byte) bool {
	if len(raw) < 4 || string(raw[2:4]) != "\r\n" {
		return false
	}
	magic := binary.LittleEndian.Uint16(raw[:2])
	return magic >= firstPython3Magic && magic <= lastPython3Magic
}
