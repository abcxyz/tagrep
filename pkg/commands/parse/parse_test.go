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
				DuplicateKeyStrategy: tags.DuplicateKeyStrategyTakeLast,
				Format:               tags.FormatRaw,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
		},
		{
			name:      "single_value_tags",
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
				DuplicateKeyStrategy: tags.DuplicateKeyStrategyTakeLast,
				Format:               tags.FormatRaw,
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
			name:      "ignores_tag_inline",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR. TAG_1=my-tag-value

TAG_2=123143
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				DuplicateKeyStrategy: tags.DuplicateKeyStrategyTakeLast,
				Format:               tags.FormatRaw,
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
			name:      "ignores_tag_inline",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR. TAG_1=my-tag-value

TAG_2=123143
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				DuplicateKeyStrategy: tags.DuplicateKeyStrategyTakeLast,
				Format:               tags.FormatRaw,
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
			name:      "duplicate_key_array_one_value_not_in_array_fields",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=my-tag-value
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				DuplicateKeyStrategy: tags.DuplicateKeyStrategyArray,
				Format:               tags.FormatRaw,
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
			name:      "duplicate_key_array_one_value_in_array_fields",
			parseType: TypeRequest,
			mockPlatform: &platform.MockPlatform{
				GetRequestBodyResponse: `A description of a PR.

Some details about a PR.

TAG_1=my-tag-value
`,
			},
			tagParser: tags.NewTagParser(ctx, &tags.Config{
				DuplicateKeyStrategy: tags.DuplicateKeyStrategyArray,
				ArrayFields:          []string{"TAG_1"},
				Format:               tags.FormatRaw,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: "TAG_1=[\"my-tag-value\"]",
		},
		{
			name:      "duplicate_key_array_multiple_values_in_array_fields",
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
				DuplicateKeyStrategy: tags.DuplicateKeyStrategyArray,
				ArrayFields:          []string{"TAG_1"},
				Format:               tags.FormatRaw,
			}),
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "GetRequestBody",
					Params: []any{},
				},
			},
			expStdout: "TAG_1=[\"my-tag-value1\",\"my-tag-value2\",\"my-tag-value3\"]",
		},
		{
			name:      "json_multiple_array_values_in_array_field",
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
				DuplicateKeyStrategy: tags.DuplicateKeyStrategyArray,
				ArrayFields:          []string{"TAG_1"},
				Format:               tags.FormatJSON,
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
				DuplicateKeyStrategy: tags.DuplicateKeyStrategyTakeLast,
				Format:               tags.FormatRaw,
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
				DuplicateKeyStrategy: tags.DuplicateKeyStrategyTakeLast,
				Format:               tags.FormatRaw,
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
