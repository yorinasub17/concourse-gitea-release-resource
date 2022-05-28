package gitea

import (
	"fmt"
	"sort"
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yorinasub17/concourse-gitea-release-resource/test"
)

func TestGetReleases(t *testing.T) {
	t.Parallel()

	clt, err := gitea.NewClient(test.ServerURL, gitea.SetBasicAuth(test.Username, test.Password))
	require.NoError(t, err)

	testCases := []struct {
		name              string
		repo              string
		semverConstraint  string
		includePreRelease bool
		expectedTags      []string
	}{
		{
			"NoPrereleaseIgnoresPrereleases",
			test.PublicRepo,
			"",
			false,
			[]string{
				"v0.0.0",
				"v0.0.1",
			},
		},
		{
			"IncludePrereleases",
			test.PublicRepo,
			"",
			true,
			[]string{
				"v0.0.0",
				"v0.0.0-alpha.1",
				"v0.0.1",
				"v0.0.1-alpha.1",
			},
		},
		{
			"SemverConstraint",
			test.PublicRepo,
			">= 0.0.0, < 0.0.1",
			false,
			[]string{
				"v0.0.0",
			},
		},
		{
			"SemverConstraintWithPrerelease",
			test.PublicRepo,
			">= 0.0.0, < 0.0.1",
			true,
			[]string{
				"v0.0.0",
				"v0.0.0-alpha.1",
			},
		},

		{
			"PrivateNoPrereleaseIgnoresPrereleases",
			test.PrivateRepo,
			"",
			false,
			[]string{
				"v0.0.0",
				"v0.0.1",
			},
		},
		{
			"PrivateIncludePrereleases",
			test.PrivateRepo,
			"",
			true,
			[]string{
				"v0.0.0",
				"v0.0.0-alpha.1",
				"v0.0.1",
				"v0.0.1-alpha.1",
			},
		},
		{
			"PrivateSemverConstraint",
			test.PrivateRepo,
			">= 0.0.0, < 0.0.1",
			false,
			[]string{
				"v0.0.0",
			},
		},
		{
			"PrivateSemverConstraintWithPrerelease",
			test.PrivateRepo,
			">= 0.0.0, < 0.0.1",
			true,
			[]string{
				"v0.0.0",
				"v0.0.0-alpha.1",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts, err := NewListReleaseOpts(test.Username, tc.repo, tc.semverConstraint, tc.includePreRelease)
			require.NoError(t, err)
			releases, err := GetReleases(clt, *opts)
			require.NoError(t, err)

			tags := []string{}
			for _, rel := range releases {
				tags = append(tags, rel.TagName)
			}
			sort.Strings(tags)
			assert.Equal(t, tc.expectedTags, tags)
		})
	}
}

func TestGetReleaseByIDAndTag(t *testing.T) {
	t.Parallel()

	clt, err := gitea.NewClient(test.ServerURL, gitea.SetBasicAuth(test.Username, test.Password))
	require.NoError(t, err)

	relByTag, err := GetReleaseByTag(clt, test.Username, test.PublicRepo, "v0.0.1")
	require.NoError(t, err)

	relByID, err := GetReleaseByID(clt, test.Username, test.PublicRepo, fmt.Sprintf("%d", relByTag.ID))
	require.NoError(t, err)

	assert.Equal(t, *relByID, *relByTag)
}

func TestGetReleasesWithPagination(t *testing.T) {
	// This test is intentionally not run in parallel due to the page size adjustment which slows down the other tests.
	defer func() {
		defaultPageSize = 100
	}()
	defaultPageSize = 1

	clt, err := gitea.NewClient(test.ServerURL, gitea.SetBasicAuth(test.Username, test.Password))
	require.NoError(t, err)

	opts, err := NewListReleaseOpts(test.Username, test.PublicRepo, "", true)
	require.NoError(t, err)
	releases, err := GetReleases(clt, *opts)
	require.NoError(t, err)

	tags := []string{}
	for _, rel := range releases {
		tags = append(tags, rel.TagName)
	}
	sort.Strings(tags)
	assert.Equal(t, []string{"v0.0.0", "v0.0.0-alpha.1", "v0.0.1", "v0.0.1-alpha.1"}, tags)
}
