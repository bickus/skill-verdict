// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"syscall"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/internal/version"
	"github.com/bickus/skill-verdict/pkg/refs"
)

var userAgent = "skill-verdict/" + version.String() + " (+https://github.com/bickus/skill-verdict)"

func SafeDialer() *net.Dialer {
	return &net.Dialer{Timeout: 30 * time.Second, Control: refuseInternal}
}

func refuseInternal(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return fmt.Errorf("refusing %q: not an IP address", host)
	}
	if !public(addr) {
		return fmt.Errorf("refusing %s: not a public address", addr)
	}
	return nil
}

type fetcher struct {
	client   *http.Client
	maxBytes int64
}

func newFetcher(client *http.Client, cfg config.ReferencesConfig) fetcher {
	return fetcher{client: client, maxBytes: cfg.MaxBodyBytes}
}

func (f fetcher) getJSON(ctx context.Context, url string, headers map[string]string, out any) (found bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return false, err
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, f.maxBytes+1))
	if err = errors.Join(err, resp.Body.Close()); err != nil {
		return false, err
	}
	switch {
	case int64(len(body)) > f.maxBytes:
		return false, fmt.Errorf("%s: body over %d bytes", url, f.maxBytes)
	case resp.StatusCode == http.StatusNotFound:
		return false, nil
	case resp.StatusCode/100 != 2:
		return false, fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	if err = json.Unmarshal(body, out); err != nil {
		return false, fmt.Errorf("%s: %w", url, err)
	}
	return true, nil
}

func ageDays(t time.Time) int {
	return int(time.Since(t).Hours() / 24)
}

func fail(rep *refs.Report, err error) {
	if errors.Is(err, ErrRateLimited) {
		rep.Status, rep.Detail = refs.NotChecked, detailRateLimit
		return
	}
	rep.Status = refs.Failed
	rep.Detail = err.Error()
}

func addFact(rep *refs.Report, key, value string) {
	rep.Facts = append(rep.Facts, refs.Fact{Key: key, Value: value})
}

func addFinding(rep *refs.Report, id string) {
	if !slices.Contains(rep.Findings, id) {
		rep.Findings = append(rep.Findings, id)
	}
}
