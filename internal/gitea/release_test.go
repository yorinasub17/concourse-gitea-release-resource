package gitea

import (
	"sort"
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testServerURL = "http://localhost:3000"
	testUsername  = "testadmin"
	testPassword  = "asdf1234"

	privateRepo                    = "fooprivate"
	publicRepo                     = "foo"
	publicRepoWithPrereleaseLatest = "foo-pre-latest"
)

func TestGetReleases(t *testing.T) {
	t.Parallel()

	clt, err := gitea.NewClient(testServerURL, gitea.SetBasicAuth(testUsername, testPassword))
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
			publicRepo,
			"",
			false,
			[]string{
				"v0.0.0",
				"v0.0.1",
			},
		},
		{
			"IncludePrereleases",
			publicRepo,
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
			publicRepo,
			">= 0.0.0, < 0.0.1",
			true,
			[]string{
				"v0.0.0",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts, err := NewListReleaseOpts(testUsername, tc.repo, tc.semverConstraint, tc.includePreRelease)
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
