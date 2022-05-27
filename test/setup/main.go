// A utility for setting up a test instance of gitea with dummy repository data
package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/gruntwork-io/go-commons/git"
	"github.com/gruntwork-io/go-commons/random"
	"github.com/gruntwork-io/go-commons/shell"
)

const (
	testServerURL = "http://localhost:3000"
	testEmail     = "testadmin@foo.com"
	testUsername  = "testadmin"
	testPassword  = "asdf1234"
)

func main() {
	clt := mustBasicAuthClient()

	privateRepo := mustCreateRepo(clt, "fooprivate", true)
	mustSetupRepoWithTestReleases(clt, privateRepo, 4)
	publicRepo := mustCreateRepo(clt, "foo", false)
	mustSetupRepoWithTestReleases(clt, publicRepo, 4)
	publicRepoWithPrereleaseLatest := mustCreateRepo(clt, "foo-pre-latest", false)
	mustSetupRepoWithTestReleases(clt, publicRepoWithPrereleaseLatest, 5)
	fmt.Fprintf(os.Stderr, "INFO: successfully created repos foo, foo-pre-latest, and fooprivate with test releases\n")
}

func mustBasicAuthClient() *gitea.Client {
	clt, err := gitea.NewClient(testServerURL, gitea.SetBasicAuth(testUsername, testPassword))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not initialize client to local test server: %s\n", err)
		os.Exit(1)
	}
	return clt
}

func mustCreateRepo(clt *gitea.Client, repoName string, private bool) *gitea.Repository {
	repo, _, err := clt.CreateRepo(gitea.CreateRepoOption{Name: repoName, Private: private})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not create repo %s on local test server: %s\n", repoName, err)
		os.Exit(1)
	}
	return repo
}

func mustSetupRepoWithTestReleases(clt *gitea.Client, repo *gitea.Repository, releaseCount int) {
	cloneURL := repo.CloneURL
	parsed, err := url.Parse(cloneURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not parse repo clone URL %s: %s\n", cloneURL, err)
		os.Exit(1)
	}
	parsed.User = url.UserPassword(testUsername, testPassword)
	cloneURLAuthed := parsed.String()

	tmpDir, err := ioutil.TempDir("", "gitea-test*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not create temp dir for cloning: %s\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	if err := git.Clone(nil, cloneURLAuthed, tmpDir); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not clone to temp dir %s: %s\n", tmpDir, err)
		os.Exit(1)
	}

	if err := setupTestGitConfig(tmpDir); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not configure git: %s\n", err)
		os.Exit(1)
	}

	// Add a file, and then update it N times, cutting a new release each time it is updated. Alternate between
	// prerelease and new release.
	for i := 0; i < releaseCount; i++ {
		isPreRelease := false
		releaseName := fmt.Sprintf("v0.0.%d", i/2)
		if i%2 == 0 {
			releaseName += "-alpha.1"
			isPreRelease = true
		}

		randomFPath := filepath.Join(tmpDir, "foo.txt")
		uniqueStr, err := random.RandomString(6, random.Base62Chars)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: could not create random string: %s\n", err)
			os.Exit(1)
		}
		if err := ioutil.WriteFile(randomFPath, []byte(uniqueStr), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: could not write random string to file %s: %s\n", randomFPath, err)
			os.Exit(1)
		}

		if err := commitAndPushFile(tmpDir, randomFPath, "random file"); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: could not commit and push random file %s: %s\n", randomFPath, err)
			os.Exit(1)
		}

		sha, err := getCurrentCommitSHA(tmpDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: could not get commit SHA: %s\n", err)
			os.Exit(1)
		}

		// Sleep before cutting release to ensure repo is in sync on server
		time.Sleep(5 * time.Second)
		mustCutRelease(clt, repo.Owner.UserName, repo.Name, releaseName, sha, isPreRelease)
	}
}

// setupTestGitConfig will configure git with the test username and email.
func setupTestGitConfig(repoRoot string) error {
	opts := shell.NewShellOptions()
	opts.WorkingDir = repoRoot
	if err := shell.RunShellCommand(opts, "git", "config", "user.name", testUsername); err != nil {
		return err
	}
	if err := shell.RunShellCommand(opts, "git", "config", "user.email", testEmail); err != nil {
		return err
	}
	return nil
}

// commitAndPushFile will commit the given file path in the repo and push upstream.
func commitAndPushFile(repoRoot, fpath, commitMsg string) error {
	opts := shell.NewShellOptions()
	opts.WorkingDir = repoRoot
	if err := shell.RunShellCommand(opts, "git", "add", fpath); err != nil {
		return err
	}
	if err := shell.RunShellCommand(opts, "git", "commit", "-m", commitMsg); err != nil {
		return err
	}
	if err := shell.RunShellCommand(opts, "git", "push", "origin", "master"); err != nil {
		return err
	}
	return nil
}

// getCurrentCommitSHA will return the commit SHA of HEAD for the given local repo.
func getCurrentCommitSHA(repoRoot string) (string, error) {
	opts := shell.NewShellOptions()
	opts.WorkingDir = repoRoot
	out, err := shell.RunShellCommandAndGetOutput(opts, "git", "rev-parse", "HEAD")
	return strings.TrimSpace(out), err
}

func mustCutRelease(clt *gitea.Client, repoOwner, repoName, releaseTag, commitSHA string, prerelease bool) *gitea.Release {
	release, resp, err := clt.CreateRelease(repoOwner, repoName, gitea.CreateReleaseOption{
		TagName:      releaseTag,
		Target:       commitSHA,
		Title:        releaseTag,
		IsPrerelease: prerelease,
	})
	if err != nil && (resp == nil || resp.Response.StatusCode > 400) {
		fmt.Fprintf(os.Stderr, "ERROR: could not create release %s on local test server: %s\n", releaseTag, err)
		os.Exit(1)
	}
	return release
}
