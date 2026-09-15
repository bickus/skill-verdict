// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package chatgpt

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

type claims struct {
	Auth struct {
		AccountID string `json:"chatgpt_account_id"`
	} `json:"https://api.openai.com/auth"`
}

func accountID(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return ""
	}
	var c claims
	if json.Unmarshal(payload, &c) != nil {
		return ""
	}
	return c.Auth.AccountID
}
