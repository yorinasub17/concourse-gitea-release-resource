package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	gogitea "code.gitea.io/sdk/gitea"
	"github.com/mattn/go-zglob"
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

	var request resource.OutRequest
	cmd.InputRequest(&request)

	srcDir := os.Args[1]

	name := readFile(srcDir, request.Params.NamePath)
	tag := readFile(srcDir, request.Params.TagPath)
	target := readFile(srcDir, request.Params.TargetPath)

	var body *string = nil
	if request.Params.BodyPath != "" {
		rawBody := readFile(srcDir, request.Params.BodyPath)
		body = &rawBody
	}

	var idStr *string = nil
	if request.Params.IDPath != "" {
		rawIDStr := readFile(srcDir, request.Params.IDPath)
		idStr = &rawIDStr
	}

	clt, err := gitea.NewGiteaClient(request.Source.GiteaURL, request.Source.AccessToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, colorstring.Color("[red]error constructing gitea client: %s\n"), err)
		os.Exit(1)
	}

	// If id is provided, assume the release already exists and attempt to retrieve it so that it can be updated.
	// Otherwise, attempt to determine if the release already exists by trying to retrieve the release by Tag and seeing
	// if it exists.
	var maybeExistingRel *gogitea.Release
	if idStr != nil {
		maybeExistingRel, err = gitea.GetReleaseByID(clt, request.Source.Owner, request.Source.Repository, *idStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, colorstring.Color("[red]error getting release with ID %s: %s\n"), *idStr, err)
			os.Exit(1)
		}
	} else {
		maybeRel, err := gitea.GetReleaseByTag(clt, request.Source.Owner, request.Source.Repository, tag)
		if err == nil && maybeRel != nil {
			maybeExistingRel = maybeRel
		}
	}

	// If the release doesn't already exist, create it. Otherwise, update the existing release with the provided
	// information. Note that in this scenario, only the name, body, and assets are updated to the provided values.
	if maybeExistingRel == nil {
		maybeExistingRel = createNewRelease(clt, srcDir, request.Source, name, tag, target, body)
	} else {
		maybeExistingRel = updateExistingRelease(clt, maybeExistingRel, srcDir, request.Source, name, tag, body)
	}
	uploadReleaseAssets(clt, maybeExistingRel, srcDir, request.Source, request.Params.Globs)

	resp := resource.InOutResponse{
		Version:  resource.VersionFromRelease(maybeExistingRel),
		Metadata: resource.MetadataFromRelease(maybeExistingRel),
	}
	cmd.OutputResponse(resp)
}

func createNewRelease(
	clt *gogitea.Client,
	srcDir string,
	src resource.Source,
	name, tag, target string,
	body *string,
) *gogitea.Release {
	opts := gitea.CreateReleaseOpts{
		Owner:        src.Owner,
		Repo:         src.Repository,
		Tag:          tag,
		Target:       target,
		Title:        name,
		IsPreRelease: src.PreRelease,
	}
	if body != nil {
		opts.Body = *body
	}
	rel, err := gitea.CreateRelease(clt, opts)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			colorstring.Color("[red]error creating new release: %s\n"),
			err,
		)
		os.Exit(1)
	}
	return rel
}

func updateExistingRelease(
	clt *gogitea.Client,
	rel *gogitea.Release,
	srcDir string,
	src resource.Source,
	name, tag string,
	body *string,
) *gogitea.Release {
	opts := gitea.CreateReleaseOpts{
		Owner:        src.Owner,
		Repo:         src.Repository,
		Tag:          tag,
		Target:       rel.Target,
		Body:         rel.Note,
		Title:        name,
		IsPreRelease: rel.IsPrerelease,
	}
	if body != nil {
		opts.Body = *body
	}
	rel, err := gitea.UpdateRelease(clt, rel.ID, opts)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			colorstring.Color("[red]error updating release %d: %s\n"),
			rel.ID, err,
		)
		os.Exit(1)
	}
	return rel
}

func uploadReleaseAssets(
	clt *gogitea.Client,
	release *gogitea.Release,
	srcDir string,
	src resource.Source,
	globs []string,
) {
	for _, fileGlob := range globs {
		matches, err := zglob.Glob(filepath.Join(srcDir, fileGlob))
		if err != nil {
			fmt.Fprintf(
				os.Stderr,
				colorstring.Color("[red]error interpretting glob %s: %s\n"),
				fileGlob, err,
			)
			os.Exit(1)
		}

		if len(matches) == 0 {
			continue
		}

		for _, filePath := range matches {
			if err := gitea.UploadReleaseAssetFromPath(clt, filePath, src.Owner, src.Repository, release.ID); err != nil {
				fmt.Fprintf(
					os.Stderr,
					colorstring.Color("[red]error uploading asset %s: %s\n"),
					filePath, err,
				)
				os.Exit(1)
			}
		}
	}
}

func readFile(srcDir, fname string) string {
	path := filepath.Join(srcDir, fname)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			colorstring.Color("[red]error reading source %s: %s\n"),
			path, err,
		)
		os.Exit(1)
	}
	return string(data)
}
