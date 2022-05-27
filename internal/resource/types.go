package resource

import (
	"strconv"
	"time"

	"code.gitea.io/sdk/gitea"
)

type Source struct {
	// Required
	GiteaURL   string `json:"gitea_url"`
	Owner      string `json:"owner"`
	Repository string `json:"repository"`

	// Optional
	AccessToken       string `json:"access_token"`
	SemverConstraint  string `json:"semver_constraint"`
	IncludePreRelease bool   `json:"include_pre_release"`
}

type CheckRequest struct {
	Source  Source  `json:"source"`
	Version Version `json:"version"`
}

type Version struct {
	Tag       string    `json:"tag,omitempty"`
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
}

func VersionFromRelease(release *gitea.Release) Version {
	if release == nil {
		return Version{}
	}

	return Version{
		ID:        strconv.FormatInt(release.ID, 10),
		Tag:       release.TagName,
		Timestamp: release.PublishedAt,
	}
}
