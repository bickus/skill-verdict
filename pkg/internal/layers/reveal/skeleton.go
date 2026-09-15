// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package reveal

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed data/confusables.json
var confusablesJSON []byte

var skeleton = sync.OnceValue(func() map[rune]string {
	var file struct {
		Map map[string]string `json:"map"`
	}
	if err := json.Unmarshal(confusablesJSON, &file); err != nil {
		panic(err)
	}
	table := make(map[rune]string, len(file.Map))
	for key, ascii := range file.Map {
		var r rune
		if _, err := fmt.Sscanf(key, "%x", &r); err != nil {
			panic(err)
		}
		table[r] = ascii
	}
	return table
})
