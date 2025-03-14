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

package issue

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/testutil"
	"github.com/abcxyz/tagrep/pkg/platform"
)

func TestApply_Process(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(t.Context(), logging.TestLogger(t))

	cases := []struct {
		name                  string
		err                   string
		expPlatformClientReqs []*platform.Request
	}{
		{
			name: "success",
			expPlatformClientReqs: []*platform.Request{
				{
					Name:   "Status",
					Params: []any{platform.StatusSuccess, &platform.StatusParams{HasDiff: true, Details: "terraform apply success", Dir: "testdir", Operation: "apply"}},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockPlatformClient := &platform.MockPlatform{}

			c := &ApplyCommand{
				platformClient: mockPlatformClient,
			}

			_, stdout, stderr := c.Pipe()

			err := c.Process(ctx)
			if diff := testutil.DiffErrString(err, tc.err); diff != "" {
				t.Error(diff)
			}

			if diff := cmp.Diff(mockPlatformClient.Reqs, tc.expPlatformClientReqs); diff != "" {
				t.Errorf("Platform calls not as expected; (-got,+want): %s", diff)
			}
		})
	}
}
