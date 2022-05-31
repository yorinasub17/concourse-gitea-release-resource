package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/mitchellh/colorstring"
	"github.com/yorinasub17/concourse-gitea-release-resource/cmd"
	"github.com/yorinasub17/concourse-gitea-release-resource/internal/gitea"
	"github.com/yorinasub17/concourse-gitea-release-resource/internal/resource"
)

func main() {
	request := resource.CheckRequest{}
	cmd.InputRequest(&request)

	semverConstraint := request.Source.SemverConstraint
	emptyVersion := resource.Version{}
	if request.Version != emptyVersion {
		// If request has a version, constrain to only include those after the current version.
		v, err := version.NewVersion(request.Version.Tag)
		if err != nil {
			fmt.Fprintf(os.Stderr, colorstring.Color("[red]error parsing version from existing release tag: %s\n"), err)
			os.Exit(1)
		}
		greaterThan := "> " + v.String()
		if semverConstraint == "" {
			semverConstraint = greaterThan
		} else {
			semverConstraint = strings.Join([]string{semverConstraint, greaterThan}, ", ")
		}
	}

	clt, err := gitea.NewGiteaClient(request.Source.GiteaURL, request.Source.AccessToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, colorstring.Color("[red]error constructing gitea client: %s\n"), err)
		os.Exit(1)
	}

	opts, err := gitea.NewListReleaseOpts(
		request.Source.Owner,
		request.Source.Repository,
		semverConstraint,
		request.Source.PreRelease,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, colorstring.Color("[red]error constructing list filters: %s\n"), err)
		os.Exit(1)
	}

	filteredReleases, err := gitea.GetReleases(clt, *opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, colorstring.Color("[red]error getting releases: %s\n"), err)
		os.Exit(1)
	}

	outputVersions := []resource.Version{}
	if len(filteredReleases) > 0 && request.Version == emptyVersion {
		// If there are releases and request didn't include a version, return the first release.
		outputVersions = append(outputVersions, resource.VersionFromRelease(filteredReleases[0]))
	} else if len(filteredReleases) > 0 {
		// If there are releases, and request included a version, return all releases.
		for _, release := range filteredReleases {
			outputVersions = append(outputVersions, resource.VersionFromRelease(release))
		}
	}
	// For all other cases, return empty release list.
	cmd.OutputResponse(outputVersions)
}
