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

package parse

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/testutil"
	"github.com/abcxyz/tagrep/pkg/platform"
	"github.com/abcxyz/tagrep/pkg/tags"
)

func TestParse_ProcessRequest(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(t.Context(), logging.TestLogger(t))

	cases := []struct {
		name                  string
		err                   string
		parseType             string
		mockPlatform          *platform.MockPlatform
		tagParser             tags.TagParser
		expPlatformClientReqs []*platform.Request
		expStdout             string
		expStderr             string
	}{
		{
			name:         "empty",
			parseType:    TypeRequest,
			mockPlatform: &platform.MockPlatform{},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				Format: tags.FormatRaw,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
		},
		{
			name:      "empty_when_tag_not_explicitly_set",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=my-tag-value
TAG_2=123143
TAG_3=A message about the tag. Something.
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				Format: tags.FormatRaw,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: "",
		},
		{
			name:      "output_all",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=my-tag-value
TAG_2=123143
TAG_3=A message about the tag. Something.
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				Format:    tags.FormatRaw,
				OutputAll: true,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: `
TAG_1=my-tag-value
TAG_2=123143
TAG_3=A message about the tag. Something.`,
		},
		{
			name:      "output_all_upper_case",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

tag_1=my-tag-value
tag_2=123143
tag_3=A message about the tag. Something.
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				Format:    tags.FormatRaw,
				OutputAll: true,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: `
TAG_1=my-tag-value
TAG_2=123143
TAG_3=A message about the tag. Something.`,
		},
		{
			name:      "output_all_respects_array_tag",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=my-tag-value1
TAG_1=my-tag-value2
TAG_2=123143
TAG_3=A message about the tag. Something.
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				Format:    tags.FormatRaw,
				OutputAll: true,
				ArrayTags: []string{"TAG_1"},
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: `
TAG_1=my-tag-value1,my-tag-value2
TAG_2=123143
TAG_3=A message about the tag. Something.`,
		},
		{
			name:      "ignores_tag_inline",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR. TAG_1=my-tag-value

TAG_2=123143
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				Format:    tags.FormatRaw,
				OutputAll: true,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: "TAG_2=123143",
		},
		{
			name:      "one_value_in_array_fields_raw",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=my-tag-value
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				ArrayTags: []string{"TAG_1"},
				Format:    tags.FormatRaw,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: "TAG_1=my-tag-value",
		},
		{
			name:      "multiple_values_in_array_fields_raw",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=my-tag-value1
TAG_1=my-tag-value2
TAG_1=my-tag-value3
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				ArrayTags: []string{"TAG_1"},
				Format:    tags.FormatRaw,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: "TAG_1=my-tag-value1,my-tag-value2,my-tag-value3",
		},
		{
			name:      "commas_are_escaped_correctly",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=some value, ok
TAG_1=another value, yes
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				ArrayTags: []string{"TAG_1"},
				Format:    tags.FormatRaw,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: `TAG_1=some value\, ok,another value\, yes`,
		},
		{
			name:      "one_value_in_array_fields_json",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=my-tag-value
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				ArrayTags:   []string{"TAG_1"},
				Format:      tags.FormatJSON,
				PrettyPrint: true,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: `{
  "TAG_1": [
    "my-tag-value"
  ]
}`,
		},
		{
			name:      "multiple_array_values_in_array_field_json",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=my-tag-value1
TAG_1=my-tag-value2
TAG_1=my-tag-value3
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				ArrayTags:   []string{"TAG_1"},
				Format:      tags.FormatJSON,
				PrettyPrint: true,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: `{
  "TAG_1": [
    "my-tag-value1",
    "my-tag-value2",
    "my-tag-value3"
  ]
}`,
		},
		{
			name:      "multiple_array_values_in_array_field_json_one_line",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=my-tag-value1
TAG_1=my-tag-value2
TAG_1=my-tag-value3
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				ArrayTags:   []string{"TAG_1"},
				Format:      tags.FormatJSON,
				PrettyPrint: false,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: `{"TAG_1":["my-tag-value1","my-tag-value2","my-tag-value3"]}`,
		},
		{
			name:      "string_tags",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=my-tag-value1
TAG_2=my-tag-value2
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				StringTags: []string{"TAG_1", "TAG_2"},
				Format:     tags.FormatRaw,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: `
TAG_1=my-tag-value1
TAG_2=my-tag-value2`,
		},
		{
			name:      "bool_tags",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=yes
TAG_2=0
TAG_3=FALSE
TAG_4=True
TAG_5=t
TAG_6=y
TAG_7=n
TAG_8=no
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				BoolTags: []string{"TAG_1", "TAG_2", "TAG_3", "TAG_4", "TAG_5", "TAG_6", "TAG_7", "TAG_8"},
				Format:   tags.FormatRaw,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: `
TAG_1=true
TAG_2=false
TAG_3=false
TAG_4=true
TAG_5=true
TAG_6=true
TAG_7=false
TAG_8=false`,
		},
		{
			name:      "bool_tags_json",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=yes
TAG_2=0
TAG_3=FALSE
TAG_4=True
TAG_5=t
TAG_6=y
TAG_7=n
TAG_8=no
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				BoolTags: []string{"TAG_1", "TAG_2", "TAG_3", "TAG_4", "TAG_5", "TAG_6", "TAG_7", "TAG_8"},
				Format:   tags.FormatJSON,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: `{"TAG_1":true,"TAG_2":false,"TAG_3":false,"TAG_4":true,"TAG_5":true,"TAG_6":true,"TAG_7":false,"TAG_8":false}`,
		},
		{
			name:      "all_tag_types",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=yes
TAG_2=0
TAG_3=FALSE
TAG_4=True
TAG_5=t
TAG_6=y
TAG_7=n
TAG_8=no
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				BoolTags:   []string{"TAG_1", "TAG_2"},
				StringTags: []string{"TAG_3", "TAG_4", "TAG_5", "TAG_6"},
				ArrayTags:  []string{"TAG_7", "TAG_8"},
				Format:     tags.FormatRaw,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: `
TAG_1=true
TAG_2=false
TAG_3=FALSE
TAG_4=True
TAG_5=t
TAG_6=y
TAG_7=n
TAG_8=no`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := &ParseCommand{
				FlagType:       tc.parseType,
				platformClient: tc.mockPlatform,
				tagParser:      tc.tagParser,
			}

			_, stdout, stderr := c.Pipe()

			err := c.Process(ctx)
			if diff := testutil.DiffErrString(err, tc.err); diff != "" {
				t.Error(diff)
			}

			if diff := cmp.Diff(tc.mockPlatform.Reqs, tc.expPlatformClientReqs); diff != "" {
				t.Errorf("Platform calls not as expected; (-got,+want): %s", diff)
			}

			if got, want := strings.TrimSpace(stdout.String()), strings.TrimSpace(tc.expStdout); got != want {
				t.Errorf("expected stdout\n\n%s\n\nto contain\n\n%s\n\n", got, want)
			}
			if got, want := strings.TrimSpace(stderr.String()), strings.TrimSpace(tc.expStderr); !strings.Contains(got, want) {
				t.Errorf("expected stderr\n\n%s\n\nto contain\n\n%s\n\n", got, want)
			}
		})
	}
}

func TestParse_ProcessIssue(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(t.Context(), logging.TestLogger(t))

	cases := []struct {
		name                  string
		err                   string
		parseType             string
		mockPlatform          *platform.MockPlatform
		tagParser             tags.TagParser
		expPlatformClientReqs []*platform.Request
		expStdout             string
		expStderr             string
	}{
		{
			name:         "empty",
			parseType:    TypeIssue,
			mockPlatform: &platform.MockPlatform{},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				Format: tags.FormatRaw,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetIssueBody",
					Params: []any{},
				},
			},
		},
		{
			name:      "single_value_tags",
			parseType: TypeIssue,
			mockPlatform: &platform.MockPlatform{
				GetIssueBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=my-tag-value
TAG_2=123143
TAG_3=A message about the tag. Something.
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				Format:    tags.FormatRaw,
				OutputAll: true,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetIssueBody",
					Params: []any{},
				},
			},
			expStdout: `
TAG_1=my-tag-value
TAG_2=123143
TAG_3=A message about the tag. Something.`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := &ParseCommand{
				FlagType:       tc.parseType,
				platformClient: tc.mockPlatform,
				tagParser:      tc.tagParser,
			}

			_, stdout, stderr := c.Pipe()

			err := c.Process(ctx)
			if diff := testutil.DiffErrString(err, tc.err); diff != "" {
				t.Error(diff)
			}

			if diff := cmp.Diff(tc.mockPlatform.Reqs, tc.expPlatformClientReqs); diff != "" {
				t.Errorf("Platform calls not as expected; (-got,+want): %s", diff)
			}

			if got, want := strings.TrimSpace(stdout.String()), strings.TrimSpace(tc.expStdout); !strings.Contains(got, want) {
				t.Errorf("expected stdout\n\n%s\n\nto contain\n\n%s\n\n", got, want)
			}
			if got, want := strings.TrimSpace(stderr.String()), strings.TrimSpace(tc.expStderr); !strings.Contains(got, want) {
				t.Errorf("expected stderr\n\n%s\n\nto contain\n\n%s\n\n", got, want)
			}
		})
	}
}
