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
	Format      string
	ArrayTags   []string
	StringTags  []string
	BoolTags    []string
	OutputAll   bool
	PrettyPrint bool
}

func (c *Config) RegisterFlags(set *cli.FlagSet) {
	f := set.NewSection("TAG OPTIONS")

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
		Name:    "array-tags",
		Target:  &c.ArrayTags,
		Example: "TAG_1",
		Default: []string{},
		Usage:   "Tags to format as an array. e.g. treat TAG_1 as an array.",
	})
	f.StringSliceVar(&cli.StringSliceVar{
		Name:    "string-tags",
		Target:  &c.StringTags,
		Example: "TAG_1",
		Default: []string{},
		Usage:   "Tags to format as a string. e.g. treat TAG_1 as a string.",
	})
	f.StringSliceVar(&cli.StringSliceVar{
		Name:    "bool-tags",
		Target:  &c.BoolTags,
		Example: "TAG_1",
		Default: []string{},
		Usage:   "Tags to format as a bool. e.g. treat TAG_1 as a bool.",
	})
	f.BoolVar(&cli.BoolVar{
		Name:    "output-all",
		Target:  &c.OutputAll,
		Example: "true",
		Default: false,
		Usage:   "Whether to print out all tags present in the resource or only those explicitly set in -array-tags, -string-tags, -bool-tags.",
	})
	f.BoolVar(&cli.BoolVar{
		Name:    "pretty",
		Target:  &c.PrettyPrint,
		Example: "true",
		Default: false,
		Usage:   "Whether to pretty print results for json on multiple lines.",
	})

	set.AfterParse(func(merr error) error {
		c.Format = strings.ToLower(strings.TrimSpace(c.Format))

		if !slices.Contains(allowedFormats, c.Format) {
			merr = errors.Join(merr, fmt.Errorf("unsupported value for format flag: %s", c.Format))
		}

		return merr
	})
}
