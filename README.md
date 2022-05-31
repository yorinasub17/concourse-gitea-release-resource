# concourse-gitea-release-resource

A Concourse CI resource for monitoring and pushing artifacts to gitea releases.

- [Examples](#examples)
- [Source Configuration](#source-configuration)
- [Behavior](#behavior)
    - [check](#check-check-for-released-versions)
    - [get](#get-fetch-assets-and-metadata-from-a-release)
    - [put](#put-publish-or-update-a-release)
- [Contributing](#contributing)


## Examples

``` yaml
resource_types:
  - name: gitea-release
    type: gitea-release
    source:
      repository: yorinasub17/gitea-release-resource:latest

resources:
- name: myrepo-release
  type: gitea-release
  source:
    gitea_url: https://gitea.com
    owner: yorinasub17
    repository: concourse-gitea-release-resource
    access_token: abcdef1234567890
```

``` yaml
- get: myrepo-release
```

``` yaml
- put: myrepo-release
  params:
    name_path: path/to/name/file
    tag_path: path/to/tag/file
    body_path: path/to/body/file
    target_path: path/to/target/file
    globs:
    - paths/to/files/to/upload-*.tgz
```

To get a specific version of a release:

``` yaml
- get: myrepo-release
  version: { tag: 'v0.0.1' }
```

## Source Configuration

| name                | required | description                                                                                                                                                                                                                                                  |
|---------------------|----------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `gitea_url`         | ✅       | The root domain of the Gitea server (e.g., `https://gitea.com`).                                                                                                                                                                                             |
| `owner`             | ✅       | The Gitea owner (user or organization) for the repository that contains the releases.                                                                                                                                                                        |
| `repository`        | ✅       | The name of the repository that contains the releases.                                                                                                                                                                                                       |
| `access_token`      |          | The API access token to use when authenticating to Gitea. Required if the repository is private.                                                                                                                                                             |
| `semver_constraint` |          | If set, constrain the returned [semver tags](https://semver.org/) according to the given constraints. The constraints are in the same format as [Terraform](https://www.terraform.io/language/expressions/version-constraints).                              |
| `pre_release`       |          | When `true`, `check` will include pre-releases in the list, while `put` will produce a pre-release. Note that `put` will only mark a release as pre-release when it is creating a new release. It will not update the pre-release flag on existing releases. |

## Behavior

### `check`: Check for released versions

List releases in the default order returned by Gitea (tag commit timestamp). The releases returned can be constrained
using the source configuration parameters.

The returned list contains an object of the following format for each release (with timestamp in the RFC3339 format):

```json
{
    "id": "12345",
    "tag": "v1.2.3",
    "timestamp": "2006-01-02T15:04:05.999999999Z"
}
```

When `check` is given such an object as the version parameter, it returns releases from the specified version on, as
ordered by Gitea. Otherwise it returns the latest release that matches the filters in the source configuration.

### `get`: Fetch assets and metadata from a release

Fetches release artifacts and metadata from the chosen release. The artifacts will be stored in a subfolder `assets` in
the destination directory.

By default all release assets will be downloaded. You can control this behavior using the `globs` input parameter. When
provided, only assets that have a file name matching the file globs will be downloaded.

The following metadata files will be available:

- `id`: The Gitea ID of the release.
- `name`: The release title.
- `tag`: The Git tag for the release.
- `target`: The target Git ref for the release.
- `url`: The direct URL to the release page.
- `body`: The release notes body for the release.
- `timestamp`: When the release was published.

### `put`: Publish or update a release

Publishes a new release on the repository if there isn't one mathing the provided tag. Otherwise, the existing release
will be updated with the provided information.

#### Parameters

| name          | required | description                                                                                                                                                     |
|---------------|----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `name_path`   | ✅       | The path to a file containing the release title.                                                                                                                |
| `tag_path`    | ✅       | The path to a file containing the Git tag to use for the release.                                                                                               |
| `body_path`   | ✅       | The path to a file containing the release body.                                                                                                                 |
| `target_path` |          | The path to a file containing a Git ref (SHA, branch, or existing tag) that should be used when cutting the release tag. Only used when creating a new release. |
| `id_path`     |          | The path to a file containing the release ID. When provided, automatically assume updating a release.                                                           |
| `globs`       |          | A list of unix globs for files that will be uploaded alongside the created release.                                                                             |

## Contributing

* If you think you've found a bug in the code or you have a question regarding the usage of this software, please reach
  out to us by [opening an issue](https://github.com/yorinasub17/concourse-gitea-release-resource/issues) in this GitHub
  repository.
* Contributions to this project are welcome: if you want to add a feature or fix a bug, please do so by [opening a Pull
  Request](https://github.com/yorinasub17/concourse-gitea-release-resource/pulls) in this GitHub repository. In case of
  feature contribution, we kindly ask you to open an issue to discuss it beforehand.
* Running tests:
    * This repo only contains integration tests with an active Gitea server. This means that you must have a running
      Gitea server with test data to run all the tests in this repo.
    * You can run a test Gitea server using [the provided Dockerfile in the test/env folder](/test/env).
    * All test data can be loaded using the Go CLI provided in [test/setup](/test/setup).
    * To make running the tests easier, you can trigger the tests by running the bash script
      [scripts/run_test.sh](/scripts/run_test.sh). This script will:
        1. Build the test Gitea container image.
        1. Start the test Gitea server in the background using the built image.
        1. Run the setup command to load the test data.
        1. Trigger tests using `go test`.
