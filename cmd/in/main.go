package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mitchellh/colorstring"
	"github.com/yorinasub17/concourse-gitea-release-resource/internal/gitea"
	"github.com/yorinasub17/concourse-gitea-release-resource/internal/resource"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, colorstring.Color("[red]usage: %s <sources directory>\n"), os.Args[0])
		os.Exit(1)
	}

	destDir := os.Args[1]

	var request resource.InRequest
	inputRequest(&request)

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

	if err := gitea.DownloadReleaseAssets(clt, maybeRel, destDir, request.Params.Globs); err != nil {
		fmt.Fprintf(
			os.Stderr,
			colorstring.Color("[red]error downloading release assets to dest dir %s: %s\n"),
			destDir, err,
		)
		os.Exit(1)
	}

	resp := resource.InResponse{
		Version:  resource.VersionFromRelease(maybeRel),
		Metadata: resource.MetadataFromRelease(maybeRel),
	}
	outputResponse(resp)
}

func inputRequest(request *resource.InRequest) {
	if err := json.NewDecoder(os.Stdin).Decode(request); err != nil {
		fmt.Fprintf(os.Stderr, colorstring.Color("[red]error reading request form stdin: %s\n"), err)
		os.Exit(1)
	}
}

func outputResponse(response resource.InResponse) {
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, colorstring.Color("[red]error writing response to stdout: %s\n"), err)
		os.Exit(1)
	}
}

func writeOutput(destDir, fname, content string) {
	path := filepath.Join(destDir, fname)
	if err := ioutil.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Fprintf(
			os.Stderr,
			colorstring.Color("[red]error writing %s to destination %s: %s\n"),
			content, path, err,
		)
		os.Exit(1)
	}
}
