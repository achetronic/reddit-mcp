// Copyright 2024 Alby Hernández
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/itchyny/gojq"
)

// runJQ runs a jq filter over the given data and returns the result as a JSON string
func runJQ(filter string, data any) (string, error) {
	query, err := gojq.Parse(filter)
	if err != nil {
		return "", fmt.Errorf("invalid jq filter: %w", err)
	}

	iter := query.Run(data)

	var results []any
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return "", fmt.Errorf("jq error: %w", err)
		}
		results = append(results, v)
	}

	// If the filter produces a single value, return it directly (not wrapped in array)
	if len(results) == 1 {
		raw, err := json.Marshal(results[0])
		if err != nil {
			return "", err
		}
		// Don't wrap arrays in extra array
		s := strings.TrimSpace(string(raw))
		if strings.HasPrefix(s, "[") || strings.HasPrefix(s, "{") {
			return s, nil
		}
		return s, nil
	}

	raw, err := json.Marshal(results)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
