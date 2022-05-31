package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/gruntwork-io/go-commons/shell"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/yorinasub17/concourse-gitea-release-resource/internal/resource"
)

var (
	metadataAssets = []string{
		"id",
		"name",
		"target",
		"url",
		"tag",
		"body",
		"timestamp",
	}
	expectedReleaseAssets = []string{
		"tag",
		"asset1",
		"asset2",
	}
)

var _ = Describe("Integration In", func() {
	var (
		inputRepo       string
		inputVersionTag string
		globs           []string

		output    resource.InOutResponse
		outputDir string
	)

	JustBeforeEach(func() {
		tmpDir, err := ioutil.TempDir("", "concourse-gitea-release-resource-incmdtest-*")
		Ω(err).ShouldNot(HaveOccurred())
		outputDir = tmpDir

		inRequest := resource.InRequest{
			Source: resource.Source{
				GiteaURL:    ServerURL,
				Owner:       Username,
				Repository:  inputRepo,
				AccessToken: accessToken,
			},
			Version: &resource.Version{
				Tag: inputVersionTag,
			},
			Params: resource.InParams{
				Globs: globs,
			},
		}

		jsonBytes, err := json.Marshal(inRequest)
		Ω(err).ShouldNot(HaveOccurred())

		var stdout bytes.Buffer
		cmd := exec.Command(
			"docker", "run",
			"-i", "--rm", "--network", "host",
			"-v", fmt.Sprintf("%s:/output", outputDir),
			imgTag, "/opt/resource/in", "/output",
		)
		cmd.Stdin = bytes.NewReader(jsonBytes)
		cmd.Stdout = &stdout
		cmd.Stderr = os.Stderr
		Ω(err).ShouldNot(HaveOccurred())
		Ω(cmd.Run()).To(Succeed())

		// chown the files to the current UID and GID so that it can be removed later. We use a docker container so that
		// we can run with root without prompting for sudo password.
		u, err := user.Current()
		Ω(err).ShouldNot(HaveOccurred())
		Ω(
			shell.RunShellCommand(
				shell.NewShellOptions(),
				"docker", "run",
				"--rm", "-v", fmt.Sprintf("%s:/output", outputDir),
				"alpine:3", "chown", "-R", fmt.Sprintf("%s:%s", u.Uid, u.Gid), "/output",
			),
		).ShouldNot(HaveOccurred())

		outputStr := strings.TrimSpace(stdout.String())
		Ω(json.Unmarshal([]byte(outputStr), &output)).To(Succeed())
	})

	// Clear out input parameters for each testcase
	JustAfterEach(func() {
		inputRepo = ""
		inputVersionTag = ""
		globs = []string{}

		Ω(os.RemoveAll(outputDir)).To(Succeed())
	})

	Context("when release has no assets", func() {
		BeforeEach(func() {
			inputRepo = PrivateRepo
			inputVersionTag = "v0.0.0"
		})

		It("outputs only release metadata", func() {
			for _, fname := range metadataAssets {
				_, err := os.Stat(filepath.Join(outputDir, fname))
				Ω(err).ShouldNot(HaveOccurred())
			}
			for _, fname := range expectedReleaseAssets {
				_, err := os.Stat(filepath.Join(outputDir, "assets", fname))
				Ω(err).Should(HaveOccurred())
			}

			Ω(ioutil.ReadFile(filepath.Join(outputDir, "tag"))).To(Equal([]byte("v0.0.0")))
			Ω(ioutil.ReadFile(filepath.Join(outputDir, "body"))).To(Equal([]byte("release v0.0.0")))
		})
	})

	Context("when release has assets", func() {
		BeforeEach(func() {
			inputRepo = PublicRepo
			inputVersionTag = "v0.0.0"
		})

		It("outputs release metadata", func() {
			for _, fname := range metadataAssets {
				_, err := os.Stat(filepath.Join(outputDir, fname))
				Ω(err).ShouldNot(HaveOccurred())
			}

			Ω(ioutil.ReadFile(filepath.Join(outputDir, "tag"))).To(Equal([]byte("v0.0.0")))
			Ω(ioutil.ReadFile(filepath.Join(outputDir, "body"))).To(Equal([]byte("release v0.0.0")))
		})

		It("outputs release assets", func() {
			for _, fname := range expectedReleaseAssets {
				_, err := os.Stat(filepath.Join(outputDir, "assets", fname))
				Ω(err).ShouldNot(HaveOccurred())
			}

			contents, err := ioutil.ReadFile(filepath.Join(outputDir, "assets", "tag"))
			Ω(err).ShouldNot(HaveOccurred())
			data := strings.Split(strings.TrimSpace(string(contents)), "\n")
			Ω(len(data)).Should(Equal(3))
			Ω(data[0]).Should(Equal("v0.0.0"))

			asset1Bytes := []byte(data[1])
			Ω(ioutil.ReadFile(filepath.Join(outputDir, "assets", "asset1"))).Should(Equal(asset1Bytes))

			asset2Bytes := []byte(data[2])
			Ω(ioutil.ReadFile(filepath.Join(outputDir, "assets", "asset2"))).Should(Equal(asset2Bytes))
		})

		Context("with globs", func() {
			BeforeEach(func() {
				globs = []string{"t*"}
			})

			It("outputs only release assets that match globs", func() {
				_, err := os.Stat(filepath.Join(outputDir, "assets", "tag"))
				Ω(err).ShouldNot(HaveOccurred())

				for _, fname := range []string{"asset1", "asset2"} {
					_, err := os.Stat(filepath.Join(outputDir, "assets", fname))
					Ω(err).Should(HaveOccurred())
				}
			})
		})
	})
})
