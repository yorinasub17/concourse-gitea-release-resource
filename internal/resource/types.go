package resource

import (
	"strconv"
	"time"

	"code.gitea.io/sdk/gitea"
)

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

type MetadataPair struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	URL      string `json:"url"`
	Markdown bool   `json:"markdown"`
}

func MetadataFromRelease(release *gitea.Release) []MetadataPair {
	metadata := []MetadataPair{}

	if release.Title != "" {
		nameMeta := MetadataPair{
			Name:  "name",
			Value: release.Title,
		}

		if release.HTMLURL != "" {
			nameMeta.URL = release.HTMLURL
		}

		metadata = append(metadata, nameMeta)
	}

	if release.HTMLURL != "" {
		metadata = append(metadata, MetadataPair{
			Name:  "url",
			Value: release.HTMLURL,
		})
	}

	if release.Note != "" {
		metadata = append(metadata, MetadataPair{
			Name:     "body",
			Value:    release.Note,
			Markdown: true,
		})
	}

	if release.TagName != "" {
		metadata = append(metadata, MetadataPair{
			Name:  "tag",
			Value: release.TagName,
		})
	}

	if release.Target != "" {
		metadata = append(metadata, MetadataPair{
			Name:  "commit_sha",
			Value: release.Target,
		})
	}

	if release.IsPrerelease {
		metadata = append(metadata, MetadataPair{
			Name:  "pre-release",
			Value: "true",
		})
	}
	return metadata
}

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

type InRequest struct {
	Source  Source   `json:"source"`
	Version *Version `json:"version"`
	Params  InParams `json:"params"`
}

type InParams struct {
	Globs []string `json:"globs"`
}

type InResponse struct {
	Version  Version        `json:"version"`
	Metadata []MetadataPair `json:"metadata"`
}
