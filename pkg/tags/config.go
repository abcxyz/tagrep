// Copyright 2025 The Authors (see AUTHORS file)
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

package tags

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/posener/complete/v2"

	"github.com/abcxyz/pkg/cli"
)

// Config is the configuration needed to parse tags.
type Config struct {
	DuplicateKeyStrategy string
	Format               string
	ArrayFields          []string
}

func (c *Config) RegisterFlags(set *cli.FlagSet) {
	f := set.NewSection("TAG OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:    "duplicate-key-strategy",
		Target:  &c.DuplicateKeyStrategy,
		Example: "array",
		Default: DuplicateKeyStrategyArray,
		Usage:   fmt.Sprintf("How to handle lines with duplicate tag keys. Allowed values are %q. Defaults to concatenating all tag values into an array.", allowedStrategies),
		Predict: complete.PredictFunc(func(prefix string) []string {
			return allowedStrategies
		}),
	})
	f.StringVar(&cli.StringVar{
		Name:    "format",
		Target:  &c.Format,
		Example: "json",
		Default: FormatRaw,
		Usage:   fmt.Sprintf("Format for the output. Allowed values are %q. Defaults to raw (outputs the deduplicated tags as they were in the PR for easy parsing into bash env variables).", allowedFormats),
		Predict: complete.PredictFunc(func(prefix string) []string {
			return allowedFormats
		}),
	})
	f.StringSliceVar(&cli.StringSliceVar{
		Name:    "array-fields",
		Target:  &c.ArrayFields,
		Example: "TAG_1",
		Default: []string{},
		Usage:   "Fields to format as an array. e.g. treat TAG_1 as an array.",
	})

	set.AfterParse(func(merr error) error {
		c.DuplicateKeyStrategy = strings.ToLower(strings.TrimSpace(c.DuplicateKeyStrategy))
		c.Format = strings.ToLower(strings.TrimSpace(c.Format))

		if !slices.Contains(allowedStrategies, c.DuplicateKeyStrategy) {
			merr = errors.Join(merr, fmt.Errorf("unsupported value for duplicate key strategy flag: %s", c.DuplicateKeyStrategy))
		}

		if !slices.Contains(allowedFormats, c.Format) {
			merr = errors.Join(merr, fmt.Errorf("unsupported value for format flag: %s", c.Format))
		}

		return merr
	})
}
