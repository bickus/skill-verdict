// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package rules

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"text/tabwriter"
)

func Render(s *Set, format string, w io.Writer) error {
	rows := [][]string{{"ID", "Severity", "Layer", "Enabled", "Interrupt", "Description"}}
	for _, id := range slices.Sorted(maps.Keys(s.ByID)) {
		r := s.ByID[id]
		rows = append(rows, []string{r.ID, severity(r, format), r.Layer, yesNo(r.Enabled), yesNo(r.Interrupt), r.Description})
	}
	switch format {
	case "text":
		return writeText(w, rows)
	case "markdown":
		return writeMarkdown(w, rows)
	}
	return fmt.Errorf("unknown format %q", format)
}

func writeText(w io.Writer, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, row := range rows {
		if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeMarkdown(w io.Writer, rows [][]string) error {
	var b strings.Builder
	for i, row := range rows {
		b.WriteString("| " + strings.Join(row, " | ") + " |\n")
		if i == 0 {
			b.WriteString("|" + strings.Repeat("---|", len(row)) + "\n")
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func severity(r Rule, format string) string {
	code := "%s"
	if format == "markdown" {
		code = "`%s`"
	}
	cell := fmt.Sprintf(code, r.Severity)
	switch {
	case !r.Judge.Downgrade:
		return cell
	case r.Judge.DowngradeFloor == "":
		return cell + ", dismissable"
	}
	return cell + ", downgradeable to " + fmt.Sprintf(code, r.Judge.DowngradeFloor)
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
