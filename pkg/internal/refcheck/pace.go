// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
)

var ErrRateLimited = errors.New("rate limit")

type host struct {
	mu     sync.Mutex
	ready  time.Time
	closed bool
}

type pacer struct {
	next    http.RoundTripper
	delay   time.Duration
	retries []time.Duration
	timeout time.Duration
	slots   chan struct{}
	replay  *replay
	mu      sync.Mutex
	hosts   map[string]*host
}

func Paced(client *http.Client, cfg config.ReferencesConfig) *http.Client {
	p := &pacer{
		next:    client.Transport,
		delay:   time.Duration(cfg.RequestDelay),
		timeout: time.Duration(cfg.Timeout),
		slots:   make(chan struct{}, max(cfg.Concurrency, 1)),
		replay:  newReplay(cfg.MaxBodyBytes),
		hosts:   make(map[string]*host),
	}
	if p.next == nil {
		p.next = http.DefaultTransport
	}
	for _, d := range cfg.RetryDelays {
		p.retries = append(p.retries, time.Duration(d))
	}
	paced := *client
	paced.Transport = p
	return &paced
}

func (p *pacer) host(name string) *host {
	p.mu.Lock()
	defer p.mu.Unlock()
	h, ok := p.hosts[name]
	if !ok {
		h = &host{}
		p.hosts[name] = h
	}
	return h
}

func (p *pacer) RoundTrip(req *http.Request) (*http.Response, error) {
	if resp, ok := p.replay.get(req); ok {
		return resp, nil
	}
	name := req.URL.Hostname()
	h := p.host(name)
	h.mu.Lock()
	defer h.mu.Unlock()
	if resp, ok := p.replay.get(req); ok {
		return resp, nil
	}
	if h.closed {
		return nil, fmt.Errorf("%s: %w", name, ErrRateLimited)
	}
	for attempt := 0; ; attempt++ {
		resp, err := p.attempt(req, h)
		limited := err == nil && rateLimited(resp)
		if err == nil && !limited && resp.StatusCode < 500 {
			return p.replay.keep(req, resp)
		}
		if attempt == len(p.retries) {
			if limited {
				h.closed = true
				return nil, errors.Join(discard(resp), fmt.Errorf("%s: %w", name, ErrRateLimited))
			}
			return resp, err
		}
		if err = errors.Join(discard(resp), wait(req.Context(), p.retries[attempt])); err != nil {
			return nil, err
		}
	}
}

func (p *pacer) attempt(req *http.Request, h *host) (*http.Response, error) {
	if err := wait(req.Context(), time.Until(h.ready)); err != nil {
		return nil, err
	}
	select {
	case p.slots <- struct{}{}:
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}
	defer func() { <-p.slots }()
	defer func() { h.ready = time.Now().Add(p.delay) }()
	ctx, cancel := context.WithTimeout(req.Context(), p.timeout)
	resp, err := p.next.RoundTrip(req.Clone(ctx))
	if err != nil {
		cancel()
		return nil, err
	}
	resp.Body = &cancelOnClose{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelOnClose) Close() error {
	defer c.cancel()
	return c.ReadCloser.Close()
}

func rateLimited(resp *http.Response) bool {
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return resp.StatusCode == http.StatusForbidden &&
		(resp.Header.Get("X-RateLimit-Remaining") == "0" || resp.Header.Get("Retry-After") != "")
}

func discard(resp *http.Response) error {
	if resp == nil {
		return nil
	}
	_, err := io.Copy(io.Discard, resp.Body)
	return errors.Join(err, resp.Body.Close())
}

func wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
