// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package transport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bickus/skill-verdict/pkg/internal/textutil"
)

const (
	retries       = 2
	backoffBase   = 200 * time.Millisecond
	maxReplyBytes = 16 << 20
)

type Client struct {
	HTTP    *http.Client
	Timeout time.Duration
}

func (c Client) Post(ctx context.Context, url string, header http.Header, body []byte) ([]byte, error) {
	var last error
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(backoffBase << (attempt - 1)):
			case <-ctx.Done():
				return nil, errors.Join(last, ctx.Err())
			}
		}
		data, retry, err := c.post(ctx, url, header, body)
		if err == nil {
			return data, nil
		}
		if !retry {
			return nil, err
		}
		last = err
	}
	return nil, last
}

func (c Client) post(ctx context.Context, url string, header http.Header, body []byte) ([]byte, bool, error) {
	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}
	req.Header = header.Clone()
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, true, err
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxReplyBytes))
	if err = errors.Join(err, resp.Body.Close()); err != nil {
		return nil, true, err
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
		return nil, true, fmt.Errorf("llm: HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode/100 != 2 {
		return nil, false, fmt.Errorf("llm: HTTP %d: %s", resp.StatusCode, textutil.Clip(strings.TrimSpace(string(data))))
	}
	return data, false, nil
}
