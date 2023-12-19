package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/colorstring"
	"github.com/yorinasub17/concourse-gitea-release-resource/cmd"
	"github.com/yorinasub17/concourse-gitea-release-resource/internal/gitea"
	"github.com/yorinasub17/concourse-gitea-release-resource/internal/resource"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, colorstring.Color("[red]usage: %s <sources directory>\n"), os.Args[0])
		os.Exit(1)
	}

	var request resource.InRequest
	cmd.InputRequest(&request)

	destDir := os.Args[1]

	clt, err := gitea.NewGiteaClient(request.Source.GiteaURL, request.Source.AccessToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, colorstring.Color("[red]error constructing gitea client: %s\n"), err)
		os.Exit(1)
	}

	// Try fetching by ID first, and then by tag
	maybeRel, err := gitea.GetReleaseByID(clt, request.Source.Owner, request.Source.Repository, request.Version.ID)
	if err != nil {
		maybeRel, err = gitea.GetReleaseByTag(clt, request.Source.Owner, request.Source.Repository, request.Version.Tag)
		if err != nil {
			fmt.Fprintf(
				os.Stderr,
				colorstring.Color("[red]error getting release - could not find by ID (%s) or tag (%s): %s\n"),
				request.Version.ID, request.Version.Tag, err,
			)
			os.Exit(1)
		}
	}

	if maybeRel.ID > 0 {
		writeOutput(destDir, "id", fmt.Sprintf("%d", maybeRel.ID))
	}

	if maybeRel.Title != "" {
		writeOutput(destDir, "name", maybeRel.Title)
	}

	if maybeRel.Target != "" {
		writeOutput(destDir, "target", maybeRel.Target)
	}

	if maybeRel.HTMLURL != "" {
		writeOutput(destDir, "url", maybeRel.HTMLURL)
	}

	if maybeRel.TagName != "" {
		writeOutput(destDir, "tag", maybeRel.TagName)
	}

	if maybeRel.Note != "" {
		writeOutput(destDir, "body", maybeRel.Note)
	}

	ts, err := maybeRel.PublishedAt.MarshalText()
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			colorstring.Color("[red]error marshalling published at time %s: %s\n"),
			maybeRel.PublishedAt, err,
		)
		os.Exit(1)
	}
	writeOutput(destDir, "timestamp", string(ts))

	if len(maybeRel.Attachments) > 0 {
		assetsDir := filepath.Join(destDir, "assets")
		if err := os.Mkdir(assetsDir, 0755); err != nil {
			fmt.Fprintf(
				os.Stderr,
				colorstring.Color("[red]error creating release assets dir %s: %s\n"),
				assetsDir, err,
			)
			os.Exit(1)
		}

		if err := gitea.DownloadReleaseAssets(clt, maybeRel, assetsDir, request.Params.Globs); err != nil {
			fmt.Fprintf(
				os.Stderr,
				colorstring.Color("[red]error downloading release assets to dest dir %s: %s\n"),
				destDir, err,
			)
			os.Exit(1)
		}
	}

	resp := resource.InOutResponse{
		Version:  resource.VersionFromRelease(maybeRel),
		Metadata: resource.MetadataFromRelease(maybeRel),
	}
	cmd.OutputResponse(resp)
}

func writeOutput(destDir, fname, content string) {
	path := filepath.Join(destDir, fname)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Fprintf(
			os.Stderr,
			colorstring.Color("[red]error writing %s to destination %s: %s\n"),
			content, path, err,
		)
		os.Exit(1)
	}
}
