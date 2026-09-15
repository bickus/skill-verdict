// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package llmcall

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
)

type object struct {
	Type       string                     `json:"type"`
	Properties map[string]json.RawMessage `json:"properties"`
	Required   []string                   `json:"required"`
	Additional bool                       `json:"additionalProperties"`
}

func Args[T any](raw json.RawMessage) (T, error) {
	var out T
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		return out, fmt.Errorf("bad arguments: %w", err)
	}
	return out, nil
}

func Property(schema json.RawMessage, name string, fields map[string]any) json.RawMessage {
	o := parse(schema)
	merged := make(map[string]any)
	if raw, ok := o.Properties[name]; ok {
		if err := json.Unmarshal(raw, &merged); err != nil {
			panic(err)
		}
	}
	for k, v := range fields {
		merged[k] = v
	}
	o.Properties[name] = must(json.Marshal(merged))
	if !slices.Contains(o.Required, name) {
		o.Required = append(o.Required, name)
	}
	return must(json.Marshal(o))
}

func List(items json.RawMessage, field string) json.RawMessage {
	list := must(json.Marshal(map[string]any{"type": "array", "items": items}))
	return must(json.Marshal(object{Type: "object", Properties: map[string]json.RawMessage{field: list}, Required: []string{field}}))
}

func parse(schema json.RawMessage) object {
	var o object
	if err := json.Unmarshal(schema, &o); err != nil {
		panic(err)
	}
	return o
}

func must(data []byte, err error) json.RawMessage {
	if err != nil {
		panic(err)
	}
	return data
}
