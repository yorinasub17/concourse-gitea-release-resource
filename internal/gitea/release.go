package gitea

import (
	"os"
	"path/filepath"
	"strconv"

	"code.gitea.io/sdk/gitea"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"

	"github.com/yorinasub17/concourse-gitea-release-resource/internal/http"
)

var defaultPageSize = 100

// ListReleaseOpts is a struct representing the filters to apply when querying for releases.
type ListReleaseOpts struct {
	// Owner is the owner of the repository in the target Gitea instance.
	Owner string
	// Repo is the name of the repository in the target Gitea instance.
	Repo string
	// SemverConstraint is a semantic versioning constraint in the same syntax as Terraform. Refer to
	// https://www.terraform.io/language/expressions/version-constraints for supported syntax.
	SemverConstraint version.Constraints
	// IncludePreRelease indicates if pre releases should be included in the query.
	IncludePreRelease bool
}

// CreateReleaseOpts is a struct representing the metadata for creating a new release.
type CreateReleaseOpts struct {
	// Owner is the owner of the repository in the target Gitea instance.
	Owner string
	// Repo is the name of the repository in the target Gitea instance.
	Repo string
	// Tag represents the tag to create for the release.
	Tag string
	// Target is the git ref (SHA, tag, branch) where the release tag should be applied.
	Target string
	// Title is the title to apply to the release.
	Title string
	// Body is the body to apply to the release.
	Body string
	// IsPreRelease indicates whether the new release should be marked as a prerelease.
	IsPreRelease bool
}

// NewListReleaseOpts constructs a new ListReleaseOpts filter based on the provided raw values.
func NewListReleaseOpts(owner, repo, semverConstraintStr string, includePreRelease bool) (*ListReleaseOpts, error) {
	out := &ListReleaseOpts{owner, repo, nil, includePreRelease}

	if semverConstraintStr != "" {
		semverConstraint, err := version.NewConstraint(semverConstraintStr)
		if err != nil {
			return nil, err
		}
		out.SemverConstraint = semverConstraint
	}

	return out, nil
}

// GetReleaseByID returns the corresponding release for the given ID string.
func GetReleaseByID(clt *gitea.Client, owner, repo, releaseIDStr string) (*gitea.Release, error) {
	releaseID, err := strconv.ParseInt(releaseIDStr, 10, 64)
	if err != nil {
		return nil, err
	}

	rel, _, err := clt.GetRelease(owner, repo, releaseID)
	return rel, err
}

// GetReleaseByTag returns the corresponding release for the given tag name.
func GetReleaseByTag(clt *gitea.Client, owner, repo, tagName string) (*gitea.Release, error) {
	rel, _, err := clt.GetReleaseByTag(owner, repo, tagName)
	return rel, err
}

// GetReleases returns all the releases that match the provided filter options. This will handle pagination, going
// through all release pages.
func GetReleases(clt *gitea.Client, opts ListReleaseOpts) ([]*gitea.Release, error) {
	releases, resp, err := getReleasesPageWithFilter(clt, opts, 1)
	if err != nil {
		return nil, err
	}

	for hasNextPage(resp) {
		linksOutput, err := parseLinks(resp.Response)
		if err != nil {
			return nil, err
		}

		nextPage := *linksOutput.nextPage
		pagedReleases, pagedResp, err := getReleasesPageWithFilter(clt, opts, nextPage)
		if err != nil {
			return nil, err
		}
		releases = append(releases, pagedReleases...)
		resp = pagedResp
	}

	return releases, nil
}

func getReleasesPageWithFilter(clt *gitea.Client, opts ListReleaseOpts, page int) ([]*gitea.Release, *gitea.Response, error) {
	apiOpts := gitea.ListReleasesOptions{
		ListOptions: gitea.ListOptions{Page: page, PageSize: defaultPageSize},
	}
	if !opts.IncludePreRelease {
		apiOpts.IsPreRelease = &opts.IncludePreRelease
	}
	releases, resp, err := clt.ListReleases(opts.Owner, opts.Repo, apiOpts)
	if err != nil {
		return nil, nil, err
	}

	releasesOut := []*gitea.Release{}
	if len(opts.SemverConstraint) > 0 {
		// Filter out releases by those that match the given semver constraint.
		for _, release := range releases {
			v, err := version.NewVersion(release.TagName)
			if err != nil {
				// ignore releases that don't have parsable semver tags
				continue
			}
			// Check against the core version so that versions with modifiers (like '-alpha.1') are also included in the
			// check.
			if opts.SemverConstraint.Check(v.Core()) {
				releasesOut = append(releasesOut, release)
			}
		}
	} else {
		releasesOut = releases
	}
	return releasesOut, resp, nil
}

func hasNextPage(resp *gitea.Response) bool {
	if resp == nil {
		return false
	}

	linksOutput, err := parseLinks(resp.Response)
	if err != nil {
		return false
	}

	return linksOutput.nextPage != nil
}

// DownloadReleaseAssets downloads the associated assets from the given release to the provided destination directory.
// The release assets to download can be filtered using glob syntax.
func DownloadReleaseAssets(clt *gitea.Client, release *gitea.Release, destDir string, globs []string) error {
	var allErr error
	for _, attachment := range release.Attachments {
		var matchFound bool
		if len(globs) == 0 {
			matchFound = true
		} else {
			for _, glob := range globs {
				matches, err := filepath.Match(glob, attachment.Name)
				if err != nil {
					allErr = multierror.Append(allErr, err)
					continue
				}

				if matches {
					matchFound = true
					break
				}
			}
		}

		if !matchFound {
			continue
		}

		attachmentPath := filepath.Join(destDir, attachment.Name)
		if err := http.DownloadFileOverHTTP(attachment.DownloadURL, attachmentPath); err != nil {
			allErr = multierror.Append(allErr, err)
		}
	}
	return allErr
}

// CreateRelease will create a new release with the given parameters.
func CreateRelease(clt *gitea.Client, opts CreateReleaseOpts) (*gitea.Release, error) {
	apiOpts := gitea.CreateReleaseOption{
		TagName:      opts.Tag,
		Target:       opts.Target,
		Title:        opts.Title,
		Note:         opts.Body,
		IsPrerelease: opts.IsPreRelease,
	}
	rel, _, err := clt.CreateRelease(opts.Owner, opts.Repo, apiOpts)
	return rel, err
}

// UpdateRelease will update an existing release with the given parameters.
func UpdateRelease(clt *gitea.Client, id int64, opts CreateReleaseOpts) (*gitea.Release, error) {
	apiOpts := gitea.EditReleaseOption{
		TagName:      opts.Tag,
		Target:       opts.Target,
		Title:        opts.Title,
		Note:         opts.Body,
		IsPrerelease: &opts.IsPreRelease,
	}
	rel, _, err := clt.EditRelease(opts.Owner, opts.Repo, id, apiOpts)
	return rel, err
}

// UploadReleaseAssetFromPath will upload the file at the given path to the provided release as an asset, using the file
// basename as the asset name.
func UploadReleaseAssetFromPath(clt *gitea.Client, path, owner, repo string, releaseID int64) error {
	basename := filepath.Base(path)

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, _, uploadErr := clt.CreateReleaseAttachment(owner, repo, releaseID, f, basename)
	return uploadErr
}
