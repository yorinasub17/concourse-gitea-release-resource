package test

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/yorinasub17/concourse-gitea-release-resource/internal/resource"
)

var _ = Describe("Integration Check", func() {
	var (
		inputRepo         string
		semverConstraint  string
		includePreRelease bool = true

		output []resource.Version
	)

	JustBeforeEach(func() {
		checkRequest := resource.CheckRequest{
			Source: resource.Source{
				GiteaURL:          ServerURL,
				Owner:             Username,
				Repository:        inputRepo,
				AccessToken:       accessToken,
				IncludePreRelease: includePreRelease,
				SemverConstraint:  semverConstraint,
			},
		}

		jsonBytes, err := json.Marshal(checkRequest)
		Ω(err).ShouldNot(HaveOccurred())

		var stdout bytes.Buffer
		cmd := exec.Command("docker", "run", "-i", "--rm", "--network", "host", imgTag, "/opt/resource/check")
		cmd.Stdin = bytes.NewReader(jsonBytes)
		cmd.Stdout = &stdout
		Ω(err).ShouldNot(HaveOccurred())
		Ω(cmd.Run()).To(Succeed())

		outputStr := strings.TrimSpace(stdout.String())
		Ω(json.Unmarshal([]byte(outputStr), &output)).To(Succeed())
	})

	Context("when this is the first time that the resource has been run", func() {
		Context("when there are no releases", func() {
			BeforeEach(func() {
				inputRepo = NoReleasesRepo
			})

			It("returns no versions", func() {
				Ω(output).Should(BeEmpty())
			})
		})

		Context("when there are releases with prelease latest", func() {
			BeforeEach(func() {
				inputRepo = PublicRepoWithPrereleaseLatest
			})

			Context("and prerelease included", func() {
				It("returns latest prerelease version", func() {
					Ω(len(output)).Should(Equal(1))
					Ω(output[0].Tag).Should(Equal("v0.0.2-alpha.1"))
				})
			})

			Context("and prerelease not included", func() {
				BeforeEach(func() {
					includePreRelease = false
				})

				It("returns latest production release version", func() {
					Ω(len(output)).Should(Equal(1))
					Ω(output[0].Tag).Should(Equal("v0.0.1"))
				})
			})
		})

		Context("when there are releases with production latest", func() {
			BeforeEach(func() {
				inputRepo = PublicRepo
			})

			Context("and prerelease included", func() {
				It("returns latest production release version", func() {
					Ω(len(output)).Should(Equal(1))
					Ω(output[0].Tag).Should(Equal("v0.0.1"))
				})
			})

			Context("and prerelease not included", func() {
				BeforeEach(func() {
					includePreRelease = false
				})

				It("returns latest production release version", func() {
					Ω(len(output)).Should(Equal(1))
					Ω(output[0].Tag).Should(Equal("v0.0.1"))
				})
			})

			Context("and semver constraint omit latest", func() {
				BeforeEach(func() {
					semverConstraint = "< v0.0.1"
				})

				It("returns latest matching", func() {
					Ω(len(output)).Should(Equal(1))
					Ω(output[0].Tag).Should(Equal("v0.0.0"))
				})
			})

			Context("and semver constraint include latest", func() {
				BeforeEach(func() {
					semverConstraint = "< v0.0.2"
				})

				It("returns latest matching", func() {
					Ω(len(output)).Should(Equal(1))
					Ω(output[0].Tag).Should(Equal("v0.0.1"))
				})
			})

			Context("and semver constraint exclude all", func() {
				BeforeEach(func() {
					semverConstraint = "~> v0.1.0"
				})

				It("returns no versions", func() {
					Ω(output).Should(BeEmpty())
				})
			})
		})
	})
})
