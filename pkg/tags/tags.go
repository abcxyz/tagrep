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
	"strconv"
	"strings"

	"golang.org/x/exp/maps"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/sets"
)

const (
	FormatUnspecified = ""
	FormatJSON        = "json"
	FormatRaw         = "raw"

	defaultJSONIndent = "  "
)

var (
	allowedFormats = func() []string {
		allowed := append([]string{}, FormatJSON, FormatRaw)
		sort.Strings(allowed)
		return allowed
	}()
	// tagPattern is a Regex pattern used to parse the tags from a multiline string.
	tagPattern = regexp.MustCompile(`(?m)^([A-Za-z0-9_]*)=([^\n\r]*)`)
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
	targetTags := sets.Union(p.cfg.ArrayTags, p.cfg.StringTags, p.cfg.BoolTags)
	for k, t := range ts {
		var err error
		key := strings.ToUpper(k)
		if !p.cfg.OutputAll && !slices.Contains(targetTags, key) {
			continue
		}
		if tagStrs[key], err = p.processTagValues(ctx, key, t); err != nil {
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
		var jsonBytes []byte
		var err error
		if p.cfg.PrettyPrint {
			jsonBytes, err = json.MarshalIndent(ts, "", defaultJSONIndent)
			if err != nil {
				return "", fmt.Errorf("failed to parse as json with indent: %w", err)
			}
		} else {
			jsonBytes, err = json.Marshal(ts)
			if err != nil {
				return "", fmt.Errorf("failed to parse as json: %w", err)
			}
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
	if slices.Contains(p.cfg.ArrayTags, key) {
		return ts, nil
	}
	if len(ts) > 1 {
		logging.FromContext(ctx).WarnContext(ctx, "encountered duplicate keys that are not in -array-tags. Defaulting to taking the last value.",
			"key", key,
			"array_tags", p.cfg.ArrayTags,
			"string_tags", p.cfg.StringTags,
			"bool_tags", p.cfg.BoolTags)
	}
	last := ts[len(ts)-1]
	if slices.Contains(p.cfg.StringTags, key) {
		return last, nil
	}
	if slices.Contains(p.cfg.BoolTags, key) {
		b, err := parseBoolValue(last)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bool: %w", err)
		}
		return b, nil
	}

	return last, nil
}

func parseBoolValue(v string) (bool, error) {
	vtl := strings.ToLower(strings.TrimSpace(v))
	// Handle cases not handled in ParseBool, which accepts:
	//   1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False.
	// https://pkg.go.dev/strconv#ParseBool
	switch vtl {
	case "yes":
		return true, nil
	case "y":
		return true, nil
	case "no":
		return false, nil
	case "n":
		return false, nil
	default:
		b, err := strconv.ParseBool(vtl)
		if err != nil {
			return false, fmt.Errorf("failed to parse %s as bool: %w", vtl, err)
		}
		return b, nil
	}
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
	switch reflect.TypeOf(v).Kind() { //nolint:exhaustive
	case reflect.String:
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("failed to cast string as string %s", v)
		}
		return s, nil
	case reflect.Bool:
		b, ok := v.(bool)
		if !ok {
			return "", fmt.Errorf("failed to cast string as bool %s", v)
		}
		return strconv.FormatBool(b), nil
	case reflect.Slice:
		// Do not use MarshalIndent here because we want the output to be on a single line for "raw" output format.
		a, ok := v.([]string)
		if !ok {
			return "", fmt.Errorf("failed to cast as slice %v", v)
		}
		final := make([]string, len(a))
		for i, s := range a {
			final[i] = escapeCommas(s)
		}
		return strings.Join(final, ","), nil
	default:
		return "", fmt.Errorf("unsupported type for tag strigify %s", v)
	}
}

func escapeCommas(v string) string {
	return strings.ReplaceAll(v, ",", `\,`)
}
