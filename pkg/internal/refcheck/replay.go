// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"sync"
)

type answer struct {
	status int
	header http.Header
	body   []byte
}

type replay struct {
	mu       sync.Mutex
	answers  map[string]answer
	maxBytes int64
}

func newReplay(maxBytes int64) *replay {
	return &replay{answers: make(map[string]answer), maxBytes: maxBytes}
}

func (r *replay) get(req *http.Request) (*http.Response, bool) {
	r.mu.Lock()
	a, ok := r.answers[req.URL.String()]
	r.mu.Unlock()
	if !ok {
		return nil, false
	}
	return &http.Response{
		StatusCode: a.status, Header: a.header.Clone(), Body: io.NopCloser(bytes.NewReader(a.body)),
		ContentLength: int64(len(a.body)), Request: req,
	}, true
}

func (r *replay) keep(req *http.Request, resp *http.Response) (*http.Response, error) {
	if resp.StatusCode/100 != 2 && resp.StatusCode != http.StatusNotFound {
		return resp, nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, r.maxBytes+1))
	if err != nil {
		return nil, errors.Join(err, resp.Body.Close())
	}
	if int64(len(body)) > r.maxBytes {
		resp.Body = &joined{Reader: io.MultiReader(bytes.NewReader(body), resp.Body), Closer: resp.Body}
		return resp, nil
	}
	if err = resp.Body.Close(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.answers[req.URL.String()] = answer{status: resp.StatusCode, header: resp.Header.Clone(), body: body}
	r.mu.Unlock()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

type joined struct {
	io.Reader
	io.Closer
}
