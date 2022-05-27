package gitea

import (
	"code.gitea.io/sdk/gitea"
	"github.com/hashicorp/go-version"
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
			if opts.SemverConstraint.Check(v) {
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
