// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/bickus/skill-verdict/pkg/internal/report"
	"github.com/bickus/skill-verdict/pkg/internal/term"
	"github.com/bickus/skill-verdict/pkg/rules"
	"github.com/bickus/skill-verdict/pkg/scan"
)

type scanFlags struct {
	config   string
	rulesDir string
	format   string
	failOn   string
	verbose  bool
}

type output struct {
	each    func(scan.Result) error
	finish  func([]scan.Result) error
	stopped func([]scan.Result, int) error
}

var levels = map[string]int{
	"never":      scan.Block.Rank() + 1,
	"block":      scan.Block.Rank(),
	"incomplete": scan.Incomplete.Rank(),
	"review":     scan.Review.Rank(),
}

func scanCmd(args []string, stdout io.Writer) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var f scanFlags
	fs.StringVar(&f.config, "config", "", "configuration file")
	fs.StringVar(&f.rulesDir, "rules-dir", "", "directory of additional rule files")
	fs.StringVar(&f.format, "format", "text", "report format: text or json")
	fs.BoolVar(&f.verbose, "verbose", false, "write every field of the JSON report")
	fs.StringVar(&f.failOn, "fail-on", "block", "exit 1 at this verdict or worse: block, incomplete, review, never")
	if err := fs.Parse(args); err != nil || fs.NArg() == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	level, ok := levels[f.failOn]
	if !ok {
		return fail(fmt.Errorf("unknown --fail-on level %q", f.failOn))
	}
	cfg, err := loadConfig(f.config, f.rulesDir)
	if err != nil {
		return fail(err)
	}
	set, err := rules.Load(cfg.Rules.Dir, cfg.Layers.Rules())
	if err != nil {
		return fail(err)
	}
	out, err := writer(f.format, f.verbose, stdout, set.ByID)
	if err != nil {
		return fail(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	results, err := scan.Run(ctx, fs.Args(), scan.Options{Config: cfg, Rules: set, Each: out.each})
	var stopped *scan.StoppedError
	if errors.As(err, &stopped) {
		return interrupted(out, results, stopped.Skipped)
	}
	if err != nil {
		return fail(err)
	}
	if err = out.finish(results); err != nil {
		return fail(err)
	}
	if worst(results).Rank() >= level {
		return 1
	}
	return 0
}

func interrupted(out output, results []scan.Result, skipped int) int {
	if out.stopped != nil {
		if err := out.stopped(results, skipped); err != nil {
			return fail(err)
		}
	}
	fmt.Fprintln(os.Stderr, "scan stopped")
	return 130
}

func worst(results []scan.Result) scan.Verdict {
	w := scan.Clean
	for _, r := range results {
		if r.Verdict.Rank() > w.Rank() {
			w = r.Verdict
		}
	}
	return w
}

func writer(format string, verbose bool, w io.Writer, byID map[string]rules.Rule) (output, error) {
	switch format {
	case "json":
		encode := report.JSON
		if verbose {
			encode = report.VerboseJSON
		}
		return output{finish: func(results []scan.Result) error {
			return encode(w, results, byID)
		}}, nil
	case "text":
		p := layout(w)
		n := 0
		summary := func(results []scan.Result, skipped int) error {
			if len(results)+skipped < 2 {
				return nil
			}
			return report.Summary(w, results, skipped, p)
		}
		return output{
			each: func(r scan.Result) error {
				n++
				return report.Skill(w, r, byID, p, n > 1)
			},
			finish:  func(results []scan.Result) error { return summary(results, 0) },
			stopped: summary,
		}, nil
	}
	return output{}, fmt.Errorf("unknown format %q", format)
}

func layout(w io.Writer) report.Layout {
	l := report.Layout{Width: report.MaxWidth}
	f, ok := w.(*os.File)
	if !ok {
		return l
	}
	info, err := f.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return l
	}
	if cols := term.Width(f); cols > 0 {
		l.Width = min(cols, report.MaxWidth)
	}
	l.Color = os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	return l
}
