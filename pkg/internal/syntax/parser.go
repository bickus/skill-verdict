// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package syntax

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed treesitter.wasm
var module []byte

type Language uint32

const (
	Python Language = iota + 1
	JavaScript
	TypeScript
	TSX
	PowerShell
)

const (
	MaxBytes = 1 << 20
	timeout  = 5 * time.Second
	cacheDir = "skill-verdict/wazero"
)

var ErrTooLarge = errors.New("syntax: source larger than the parse limit")

type Parser struct {
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	module   api.Module
	names    map[Language]*names
}

func New(ctx context.Context) (*Parser, error) {
	cfg := wazero.NewRuntimeConfig().WithCloseOnContextDone(true)
	if base, err := os.UserCacheDir(); err == nil {
		if cache, err := wazero.NewCompilationCacheWithDir(filepath.Join(base, cacheDir)); err == nil {
			cfg = cfg.WithCompilationCache(cache)
		}
	}
	p := &Parser{runtime: wazero.NewRuntimeWithConfig(ctx, cfg), names: make(map[Language]*names)}
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, p.runtime); err != nil {
		return nil, errors.Join(err, p.runtime.Close(ctx))
	}
	compiled, err := p.runtime.CompileModule(ctx, module)
	if err != nil {
		return nil, errors.Join(err, p.runtime.Close(ctx))
	}
	p.compiled = compiled
	if err = p.instantiate(ctx); err != nil {
		return nil, errors.Join(err, p.runtime.Close(ctx))
	}
	return p, nil
}

func (p *Parser) Close(ctx context.Context) error {
	return p.runtime.Close(ctx)
}

func (p *Parser) instantiate(ctx context.Context) error {
	var closing error
	if p.module != nil {
		closing = p.module.Close(ctx)
	}
	module, err := p.runtime.InstantiateModule(ctx, p.compiled, wazero.NewModuleConfig().WithName("").WithStartFunctions("_initialize"))
	p.module = module
	return errors.Join(closing, err)
}

func (p *Parser) Parse(ctx context.Context, lang Language, source string) (*Tree, error) {
	if len(source) > MaxBytes {
		return nil, ErrTooLarge
	}
	table, err := p.table(ctx, lang)
	if err != nil {
		return nil, err
	}
	text := normalize(source)
	rows, err := p.guarded(ctx, "sv_parse", uint64(lang), []byte(text))
	if err != nil {
		return nil, err
	}
	return build(rows, table, text), nil
}

func (p *Parser) table(ctx context.Context, lang Language) (*names, error) {
	if t, ok := p.names[lang]; ok {
		return t, nil
	}
	raw, err := p.guarded(ctx, "sv_names", uint64(lang), nil)
	if err != nil {
		return nil, err
	}
	t, err := parseNames(raw)
	if err != nil {
		return nil, err
	}
	p.names[lang] = t
	return t, nil
}

func (p *Parser) guarded(ctx context.Context, function string, lang uint64, input []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, err := p.call(ctx, function, lang, input)
	if err == nil {
		return out, nil
	}
	if restart := p.instantiate(context.WithoutCancel(ctx)); restart != nil {
		return nil, errors.Join(err, restart)
	}
	return nil, err
}

func (p *Parser) call(ctx context.Context, function string, lang uint64, input []byte) (out []byte, err error) {
	args := []uint64{lang}
	if input != nil {
		at, failed := p.write(ctx, input)
		if failed != nil {
			return nil, failed
		}
		defer func() { err = errors.Join(err, p.free(ctx, at)) }()
		args = append(args, at, uint64(len(input)))
	}
	got, err := p.module.ExportedFunction(function).Call(ctx, args...)
	if err != nil {
		return nil, err
	}
	if got[0] == 0 {
		return nil, fmt.Errorf("syntax: %s returned nothing", function)
	}
	defer func() { err = errors.Join(err, p.free(ctx, got[0])) }()
	size, err := p.module.ExportedFunction("sv_size").Call(ctx)
	if err != nil {
		return nil, err
	}
	view, ok := p.module.Memory().Read(api.DecodeU32(got[0]), api.DecodeU32(size[0]))
	if !ok {
		return nil, errors.New("syntax: result outside module memory")
	}
	return append([]byte(nil), view...), nil
}

func (p *Parser) write(ctx context.Context, input []byte) (uint64, error) {
	got, err := p.module.ExportedFunction("sv_alloc").Call(ctx, uint64(len(input)+1))
	if err != nil {
		return 0, err
	}
	if got[0] == 0 || !p.module.Memory().Write(api.DecodeU32(got[0]), input) {
		return 0, errors.New("syntax: no module memory for the source")
	}
	return got[0], nil
}

func (p *Parser) free(ctx context.Context, at uint64) error {
	_, err := p.module.ExportedFunction("sv_free").Call(ctx, at)
	return err
}

func normalize(source string) string {
	text := []byte(source)
	for i, c := range text {
		if c == '\r' && (i+1 == len(text) || text[i+1] != '\n') {
			text[i] = '\n'
		}
	}
	return string(text)
}
