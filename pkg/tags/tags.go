// Copyright 2025 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package tags contains logic for parsing tags from a multiline string.
package tags

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/exp/maps"

	"github.com/abcxyz/pkg/logging"
)

const (
	DuplicateKeyStrategyUnspecified = ""
	DuplicateKeyStrategyTakeLast    = "take-last"
	DuplicateKeyStrategyArray       = "array"

	FormatUnspecified = ""
	FormatJSON        = "json"
	FormatRaw         = "raw"
)

var (
	allowedStrategies = map[string]struct{}{
		DuplicateKeyStrategyTakeLast: {},
		DuplicateKeyStrategyArray:    {},
	}
	// SortedStrategies are the sorted duplicate key strategies.
	SortedStrategies = func() []string {
		allowed := append([]string{}, DuplicateKeyStrategyTakeLast, DuplicateKeyStrategyArray)
		sort.Strings(allowed)
		return allowed
	}()

	allowedFormats = map[string]struct{}{
		FormatJSON: {},
		FormatRaw:  {},
	}
	// SortedFormats are the sorted formats.
	SortedFormats = func() []string {
		allowed := append([]string{}, FormatJSON, FormatRaw)
		sort.Strings(allowed)
		return allowed
	}()
	// tagPattern is a Regex pattern used to parse the tags from a multiline string.
	tagPattern = regexp.MustCompile(`([A-Za-z0-9_]*)=(.*)`)
)

// Tag represents the tag name and values from the issue or request.
type Tag struct {
	// Name of the tag.
	Name string
	// Value of the tag.
	Value string
}

type TagParser struct {
	cfg *Config
}

// NewTagParser creates a new tag parser.
func NewTagParser(ctx context.Context, cfg *Config) TagParser {
	return TagParser{cfg}
}

func (p *TagParser) format(ctx context.Context, ts map[string]any) (r string, merr error) {
	switch p.cfg.Format {
	case FormatRaw:
		var builder strings.Builder
		keys := maps.Keys(ts)
		sort.Strings(keys)
		for _, k := range keys {
			if _, err := builder.WriteString(fmt.Sprintf("%s=%s\n", k, ts[k])); err != nil {
				merr = errors.Join(merr, fmt.Errorf("failed to write tag(%s): %w", k, err))
			}
		}
		return builder.String(), merr
	case FormatJSON:
		jsonBytes, err := json.Marshal(ts)
		if err != nil {
			return "", fmt.Errorf("failed to parse as json: %w", err)
		}
		return string(jsonBytes), nil
	case FormatUnspecified:
	default:
		return "", fmt.Errorf("format '%s' is invalid", p.cfg.Format)
	}
	return "", fmt.Errorf("unknown error formatting tags")
}

func (p *TagParser) ParseTags(ctx context.Context, v string) (string, error) {
	tagStrs := make(map[string]any)
	ts := parseTags(ctx, v)
	for k, t := range ts {
		tagStrs[k] = t[0].Value
		if len(t) > 1 || sliceContains(p.cfg.ArrayFields, t[0].Name) {
			var err error
			if tagStrs[k], err = p.processDuplicateKeys(ctx, t); err != nil {
				return "", fmt.Errorf("failed to process duplicate keys: %w", err)
			}
		}
	}
	r, err := p.format(ctx, tagStrs)
	if err != nil {
		return "", fmt.Errorf("failed to format tags: %w", err)
	}
	return r, nil
}

func (p *TagParser) processDuplicateKeys(ctx context.Context, ts []*Tag) (any, error) {
	last := ts[len(ts)-1]
	if !sliceContains(p.cfg.ArrayFields, ts[0].Name) && p.cfg.DuplicateKeyStrategy != DuplicateKeyStrategyTakeLast {
		logging.DefaultLogger().WarnContext(ctx, "encountered duplicate keys that are not in array fields. Defaulting to take-last.",
			"key", last.Name,
			"array_fields", p.cfg.ArrayFields)
		return last.Value, nil
	}
	switch p.cfg.DuplicateKeyStrategy {
	case DuplicateKeyStrategyTakeLast:
		return last.Value, nil
	case DuplicateKeyStrategyArray:
		if p.cfg.Format == FormatJSON {
			// If the format is json this will be handled later in the format() func.
			return values(ts), nil
		}
		jsonBytes, err := json.Marshal(values(ts))
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s as array: %w", last.Name, err)
		}
		return string(jsonBytes), nil
	case DuplicateKeyStrategyUnspecified:
	default:
		return nil, fmt.Errorf("processing duplicate key '%s' with invalid duplicate key strategy '%s'",
			ts[0].Name, p.cfg.DuplicateKeyStrategy)
	}
	return nil, fmt.Errorf("unknown error processing duplicate keys")
}

func values(ts []*Tag) []string {
	resp := make([]string, len(ts))
	for i, t := range ts {
		resp[i] = t.Value
	}
	return resp
}

func parseTags(ctx context.Context, v string) map[string][]*Tag {
	resp := make(map[string][]*Tag)
	matches := tagPattern.FindAllStringSubmatch(v, -1)
	for _, m := range matches {
		if len(m) < 3 {
			logging.DefaultLogger().WarnContext(ctx, "unable to parse tag line", "invalid_match", m)
			continue
		}
		resp[m[1]] = append(resp[m[1]], &Tag{Name: m[1], Value: m[2]})
	}
	return resp
}

func sliceIndex[T comparable](haystack []T, needle T) int {
	for i, v := range haystack {
		if v == needle {
			return i
		}
	}
	return -1
}

func sliceContains[T comparable](haystack []T, needle T) bool {
	return sliceIndex(haystack, needle) != -1
}
