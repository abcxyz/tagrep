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

package platform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sethvargo/go-githubactions"

	"github.com/abcxyz/pkg/logging"
)

func TestGitHubConfigDefaults_Load(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(t.Context(), logging.TestLogger(t))

	cases := []struct {
		name          string
		githubContext *githubactions.GitHubContext
		exp           *gitHubConfigDefaults
	}{
		{
			name: "pull_request",
			githubContext: &githubactions.GitHubContext{
				Repository: "owner/repo",
				EventName:  "pull_request",
				Event: map[string]any{
					"number": 123,
					"pull_request": map[string]any{
						"body": "this-is-a-pull-request-body",
					},
				},
			},
			exp: &gitHubConfigDefaults{
				Owner:             "owner",
				Repo:              "repo",
				PullRequestNumber: 123,
				PullRequestBody:   "this-is-a-pull-request-body",
			},
		},
		{
			name: "pull_request_target",
			githubContext: &githubactions.GitHubContext{
				Repository: "owner/repo",
				EventName:  "pull_request_target",
				Event: map[string]any{
					"number": 123,
					"pull_request": map[string]any{
						"body": "this-is-a-pull-request-body",
					},
				},
			},
			exp: &gitHubConfigDefaults{
				Owner:             "owner",
				Repo:              "repo",
				PullRequestNumber: 123,
				PullRequestBody:   "this-is-a-pull-request-body",
			},
		},
		{
			name: "pull_request_review",
			githubContext: &githubactions.GitHubContext{
				Repository: "owner/repo",
				EventName:  "pull_request_review",
				Event: map[string]any{
					"pull_request": map[string]any{
						"body":   "this-is-a-pull-request-body",
						"number": 123,
					},
				},
			},
			exp: &gitHubConfigDefaults{
				Owner:             "owner",
				Repo:              "repo",
				PullRequestNumber: 123,
				PullRequestBody:   "this-is-a-pull-request-body",
			},
		},
		{
			name: "merge_group_main_branch",
			githubContext: &githubactions.GitHubContext{
				Repository: "owner/repo",
				EventName:  "merge_group",
				Event: map[string]any{
					"merge_group": map[string]any{
						"head_ref": "refs/heads/gh-readonly-queue/main/pr-123",
					},
				},
			},
			exp: &gitHubConfigDefaults{
				Owner:             "owner",
				Repo:              "repo",
				PullRequestNumber: 123,
				PullRequestBody:   "",
			},
		},
		{
			name: "merge_group_master_branch",
			githubContext: &githubactions.GitHubContext{
				Repository: "owner/repo",
				EventName:  "merge_group",
				Event: map[string]any{
					"merge_group": map[string]any{
						"head_ref": "refs/heads/gh-readonly-queue/master/pr-123",
					},
				},
			},
			exp: &gitHubConfigDefaults{
				Owner:             "owner",
				Repo:              "repo",
				PullRequestNumber: 123,
				PullRequestBody:   "",
			},
		},
		{
			name: "merge_group_release_branch",
			githubContext: &githubactions.GitHubContext{
				Repository: "owner/repo",
				EventName:  "merge_group",
				Event: map[string]any{
					"merge_group": map[string]any{
						"head_ref": "refs/heads/gh-readonly-queue/release123/pr-123",
					},
				},
			},
			exp: &gitHubConfigDefaults{
				Owner:             "owner",
				Repo:              "repo",
				PullRequestNumber: 123,
				PullRequestBody:   "",
			},
		},
		{
			name: "merge_group_nested_branch",
			githubContext: &githubactions.GitHubContext{
				Repository: "owner/repo",
				EventName:  "merge_group",
				Event: map[string]any{
					"merge_group": map[string]any{
						"head_ref": "refs/heads/gh-readonly-queue/dcreey/my-branch/pr-123",
					},
				},
			},
			exp: &gitHubConfigDefaults{
				Owner:             "owner",
				Repo:              "repo",
				PullRequestNumber: 123,
				PullRequestBody:   "",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := &gitHubConfigDefaults{}
			c.Load(ctx, tc.githubContext)

			if diff := cmp.Diff(c, tc.exp); diff != "" {
				t.Errorf("GitHubConfigDefaults not as expected; (-got,+want): %s", diff)
			}
		})
	}
}
