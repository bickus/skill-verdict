// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package syntax

import (
	"bytes"
	"encoding/binary"
	"errors"
)

const (
	columns  = 5
	rowBytes = columns * 4
)

type Node struct {
	Kind     string
	Field    string
	Start    int
	End      int
	Named    bool
	Error    bool
	Missing  bool
	Children []*Node
}

type Tree struct {
	Root   *Node
	Source string
}

func (t *Tree) Text(n *Node) string {
	if n == nil {
		return ""
	}
	return t.Source[n.Start:n.End]
}

func (n *Node) Child(field string) *Node {
	for _, c := range n.Children {
		if c.Field == field {
			return c
		}
	}
	return nil
}

func (n *Node) NamedChildren() []*Node {
	var out []*Node
	for _, c := range n.Children {
		if c.Named {
			out = append(out, c)
		}
	}
	return out
}

func (n *Node) Walk(visit func(*Node) bool) {
	if !visit(n) {
		return
	}
	for _, c := range n.Children {
		c.Walk(visit)
	}
}

type names struct {
	symbols []string
	fields  []string
}

func parseNames(raw []byte) (*names, error) {
	if len(raw) < 8 {
		return nil, errors.New("syntax: short name table")
	}
	symbols := int(binary.LittleEndian.Uint32(raw))
	fields := int(binary.LittleEndian.Uint32(raw[4:]))
	parts := bytes.Split(raw[8:], []byte{0})
	if len(parts) < symbols+fields {
		return nil, errors.New("syntax: name table does not match its counts")
	}
	t := &names{symbols: make([]string, symbols), fields: make([]string, fields+1)}
	for i := range symbols {
		t.symbols[i] = string(parts[i])
	}
	for i := range fields {
		t.fields[i+1] = string(parts[symbols+i])
	}
	return t, nil
}

func (t *names) symbol(id uint32) string {
	if int(id) < len(t.symbols) {
		return t.symbols[id]
	}
	return ""
}

func (t *names) field(id uint32) string {
	if int(id) < len(t.fields) {
		return t.fields[id]
	}
	return ""
}

func build(rows []byte, table *names, source string) *Tree {
	tree := &Tree{Source: source}
	var stack []*Node
	for at := 0; at+rowBytes <= len(rows); at += rowBytes {
		cell := func(i int) uint32 { return binary.LittleEndian.Uint32(rows[at+4*i:]) }
		flags := cell(3)
		n := &Node{
			Kind: table.symbol(cell(0)), Field: table.field(cell(4)),
			Start: min(int(cell(1)), len(source)), End: min(int(cell(2)), len(source)),
			Named: flags&1 != 0, Missing: flags&2 != 0, Error: flags&4 != 0,
		}
		depth := int(flags >> 3)
		if depth == 0 || depth > len(stack) {
			if tree.Root != nil {
				continue
			}
			tree.Root = n
			stack = []*Node{n}
			continue
		}
		stack = stack[:depth]
		parent := stack[depth-1]
		parent.Children = append(parent.Children, n)
		stack = append(stack, n)
	}
	return tree
}
