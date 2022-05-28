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
		inputRepo string
		output    string
	)

	JustBeforeEach(func() {
		checkRequest := resource.CheckRequest{
			Source: resource.Source{
				GiteaURL:    ServerURL,
				Owner:       Username,
				Repository:  inputRepo,
				AccessToken: accessToken,
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
		output = strings.TrimSpace(stdout.String())
	})

	Context("when this is the first time that the resource has been run", func() {
		Context("when there are no releases", func() {
			BeforeEach(func() {
				inputRepo = NoReleasesRepo
			})

			It("returns no versions", func() {
				Ω(output).Should(Equal("[]"))
			})
		})
	})
})
