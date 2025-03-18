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
	"reflect"
	"regexp"
	"slices"
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

	defaultJSONIndent = "  "
)

var (
	allowedStrategies = func() []string {
		allowed := append([]string{}, DuplicateKeyStrategyTakeLast, DuplicateKeyStrategyArray)
		sort.Strings(allowed)
		return allowed
	}()
	allowedFormats = func() []string {
		allowed := append([]string{}, FormatJSON, FormatRaw)
		sort.Strings(allowed)
		return allowed
	}()
	// tagPattern is a Regex pattern used to parse the tags from a multiline string.
	tagPattern = regexp.MustCompile(`([A-Za-z0-9_]*)=([^\n\r]*)`)
)

type TagParser struct {
	cfg *Config
}

// NewTagParser creates a new tag parser.
func NewTagParser(ctx context.Context, cfg *Config) TagParser {
	return TagParser{cfg}
}

func (p *TagParser) ParseTags(ctx context.Context, v string) (string, error) {
	tagStrs := make(map[string]any)
	ts := parseTags(ctx, v)
	for k, t := range ts {
		var err error
		if tagStrs[k], err = p.processTagValues(ctx, k, t); err != nil {
			return "", fmt.Errorf("failed to process duplicate keys: %w", err)
		}
	}
	r, err := p.format(ctx, tagStrs)
	if err != nil {
		return "", fmt.Errorf("failed to format tags: %w", err)
	}
	return r, nil
}

func (p *TagParser) format(ctx context.Context, ts map[string]any) (r string, merr error) {
	switch p.cfg.Format {
	case FormatRaw:
		var builder strings.Builder
		keys := maps.Keys(ts)
		sort.Strings(keys)
		for _, k := range keys {
			v, err := stringifyRaw(ts[k])
			if err != nil {
				merr = errors.Join(merr, fmt.Errorf("failed to parse %s as array: %w", k, err))
			}
			if _, err := builder.WriteString(fmt.Sprintf("%s=%s\n", k, v)); err != nil {
				merr = errors.Join(merr, fmt.Errorf("failed to write tag(%s): %w", k, err))
			}
		}
		return builder.String(), merr
	case FormatJSON:
		jsonBytes, err := json.MarshalIndent(ts, "", defaultJSONIndent)
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

// processTagValues either returns an array or a string value depending on the duplicate key strategy.
func (p *TagParser) processTagValues(ctx context.Context, key string, ts []string) (any, error) {
	last := ts[len(ts)-1]
	if len(ts) == 1 && !slices.Contains(p.cfg.ArrayFields, key) {
		return last, nil
	}
	if !slices.Contains(p.cfg.ArrayFields, key) && p.cfg.DuplicateKeyStrategy != DuplicateKeyStrategyTakeLast {
		logging.FromContext(ctx).WarnContext(ctx, "encountered duplicate keys that are not in array fields. Defaulting to take-last.",
			"key", key,
			"array_fields", p.cfg.ArrayFields)
		return last, nil
	}
	switch p.cfg.DuplicateKeyStrategy {
	case DuplicateKeyStrategyTakeLast:
		return last, nil
	case DuplicateKeyStrategyArray:
		return ts, nil
	case DuplicateKeyStrategyUnspecified:
	default:
		return nil, fmt.Errorf("processing duplicate key '%s' with invalid duplicate key strategy '%s'",
			key, p.cfg.DuplicateKeyStrategy)
	}
	return nil, fmt.Errorf("unknown error processing duplicate keys")
}

func parseTags(ctx context.Context, v string) map[string][]string {
	resp := make(map[string][]string)
	matches := tagPattern.FindAllStringSubmatch(v, -1)
	for _, m := range matches {
		if len(m) < 3 {
			logging.FromContext(ctx).WarnContext(ctx, "unable to parse tag line", "invalid_match", m)
			continue
		}
		resp[m[1]] = append(resp[m[1]], m[2])
	}
	return resp
}

func stringifyRaw(v any) (string, error) {
	switch reflect.TypeOf(v).Kind() {
	case reflect.String:
		return v.(string), nil
	case reflect.Slice:
		// Do not use MarshalIndent here because we want the output to be on a single line for "raw" output format.
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to parse as array: %w", err)
		}
		return string(jsonBytes), nil
	default:
		return "", fmt.Errorf("unsupported type for tag strigify %s", v)
	}
}
