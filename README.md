# tagrep

A cli tool for parsing TAG_KEY=TAG_VALUE from GitHub pull requests, GitHub
Issues, GitLab Merge Requests, GitLab Issues, etc.

**This is not an official Google product.**

## Problem Statement

Users of version control platforms (e.g. GitHub, GitLab) often need to enable
more robust functionality in their workflows. For example:

* Free form justification for a specific action being taken (e.g. Access on
  Demand request, audit trail of specific action)
* Acknowledgement of a condition (e.g. ignore code freeze)
* Turn on and off features (e.g. requiring all reviewers to approve before
  merge)
* etc

Existing functionality in GitHub and GitLab is not sufficient to natively
support enabling all of these use cases. Instead, **we need a tool that can
parse metadata from GitHub and GitLab merge/pull requests and issues.**

## How it works

Users can add any metadata as semi-structured tags in their Merge/Pull Request
or Issue body/description and the data will be fetched and parsed into a
machine readable format.

Example merge/pull request:
```
fix: some bug

WANT_LGTM=all
ACK_OTHER_THING=yes
```

Example issue:
```
AOD request for access to production service ABC

JUSTIFICATION=I need access so I can delete the database
```

You can use `tagrep` in a GitHub or GitLab workflow to fetch and parse these tags:

```
tagrep parse -type=request -format=json
# Outputs:
{
  "WANT_LGTM":"all",
  "ACK_OTHER_THING":"yes"
}

```

```
tagrep parse -type=issue -format=raw
# Outputs:
JUSTIFICATION=I need access so I can delete the database
```

## Usage

For more information about available commands and options, run a command with
`-help` to use detailed usage instructions.

### parse

The `parse` command fetches the target request or issue and parses all tags

```
# Parse an issue and output the format in ENV var syntax.
tagrep parse -type=issue -format=raw

# Parse a github pull request or gitlab merge request and output the format as a JSON object.
tagrep parse -type=request -format=json
```

#### CLI Flags

| flag           | required | possible values    | description                                                                                                                                                    |
|----------------|----------|--------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `-type`        | x        | `issue`, `request` | Whether to fetch a github/gitlab issue or pull/merge request.                                                                                                  |
| `-format`      |          | `json`, `raw`      | The format to output as. `json` will output as a single json object. `raw` will output as separate rows parsable into env variables.                           |
| `-array-tags`  |          | {{any}}            | The tags that should be treated as an array.                                                                                                                   |
| `-string-tags` |          | {{any}}            | The tags that should be treated as a string.                                                                                                                   |
| `-bool-tags`   |          | {{any}}            | The tags that should be treated as a bool.                                                                                                                     |
| `-output-all`  |          | true,false         | Whether to output all found tags or just those in `-array-tags`, `-string-tags`, and `-bool-tags`. Defaults to false (just those in the `-{type}-tags` flags). |

#### GitHub Optional Flags

These options will be automatically parsed from the GitHub context if available.

| flag                          | description                                                                      |
|-------------------------------|----------------------------------------------------------------------------------|
| `-github-token`               | The github token to use. Defaults to taking the GITHUB_TOKEN env variable.       |
| `-github-owner`               | The github organization/owner of the repository.                                 |
| `-github-repo`                | The github repository to access.                                                 |
| `-github-app-id`              | The ID of the github application to auth as. Used instead of the `-github-token` |
| `-github-app-installation-id` | The installation of ID of the github application.                                |
| `-github-app-private-key-pem` | The private key pem file to use to auth to the github application.               |
| `-github-pull-request-number` | The number of the pull request to parse.                                         |
| `-github-issue-number`        | The number of the issue to parse.                                                |

#### GitLab Optional Flags

These options will be automatically parsed from the GitLab context if available.

| flag                        | description                                                                       |
|-----------------------------|-----------------------------------------------------------------------------------|
| `-tagrep-gitlab-token`      | The gitlab token to use. Defaults to taking the TAGREP_GITLAB_TOKEN env variable. |
| `-gitlab-base-url`          | The base url to send api requests to. Defaults to the GITLAB_BASE_URL env var.    |
| `-gitlab-project-id`        | The ID of the project to access                                                   |
| `-gitlab-merge-request-iid` | The GitLab project-level merge request internal ID.                               |
| `-gitlab-issue-iid`         | The GitLab project-level issue internal ID.                                       |


## Examples

### GitHub - Exporting tags as environment variables

```yml
on:
  issues:
    types: ['opened', 'edited']

permissions:
  pull-requests: 'read' # Only required if type is 'request'
  issues: 'read'        # Only required if type is 'issue'

tagrep:
  runs-on: 'ubuntu-latest'
  steps:
  - name: 'Export Tags to ENV'
    shell: 'bash'
    env:
      GITHUB_TOKEN: '${{ secrets.GITHUB_TOKEN }}'
    run: |
      tagrep parse -type={{type}} -format=raw >> "$GITHUB_ENV"

  - name: 'Print out env vars'
    shell: 'bash'
    run: |
      echo "$SOME_TAG_THAT_MAY_BE_IN_ISSUE"
```

### GitHub - Exporting tags as a single JSON output

```yml
permissions:
  pull-requests: 'read' # Only required if type is 'request'
  issues: 'read'        # Only required if type is 'issue'

tagrep:
  runs-on: 'ubuntu-latest'
  steps:
  - name: 'Export Tags as json to Output'
    id: 'json'
    shell: 'bash'
    env:
      GITHUB_TOKEN: '${{ secrets.GITHUB_TOKEN }}'
    run: |
      echo "tags=$(tagrep parse -type={{type}} -format=json)" >> "$GITHUB_OUTPUT"

  - name: 'Print output'
    shell: 'bash'
    run: |
      echo "${{ fromJSON(steps.json.outputs.tags)["SOME_TAG_THAT_MAY_BE_IN_ISSUE"] }}"
```

### GitHub - Usage in a GitHub Pull Request or Merge Queue

```yml
on:
  pull_request:
  merge_group:

permissions:
  pull-requests: 'read'

tagrep:
  runs-on: 'ubuntu-latest'
  steps:
  - name: 'Export Tags to ENV'
    shell: 'bash'
    env:
      GITHUB_TOKEN: '${{ secrets.GITHUB_TOKEN }}'
    run: |
      tagrep parse -type=request -format=raw >> "$GITHUB_ENV"
```

### GitHub - Usage in a GitHub Issue

```yml
on:
  issues:
    types: ['opened', 'edited']

permissions:
  issues: 'read'

tagrep:
  runs-on: 'ubuntu-latest'
  steps:
  - name: 'Export Tags to ENV'
    shell: 'bash'
    env:
      GITHUB_TOKEN: '${{ secrets.GITHUB_TOKEN }}'
    run: |
      tagrep parse -type=issue -format=raw >> "$GITHUB_ENV"
```

### GitLab - Exporting tags as environment variables

TODO

### GitLab - Exporting tags as a single JSON output

TODO

### GitLab - Usage in a GitLab Merge Request or Merge Train

TODO

### GitLab - Usage in a GitLab Issue

TODO

## Auth

* The cli tool first resolves the GITHUB_TOKEN (github) or CI_JOB_TOKEN
  (gitlab) automatically from the environment.
* You can instead auth as a GitHub application by including the following
  flags: `-github-app-id`, `-github-app-installation-id`,
  `-github-app-private-key-pem`.
