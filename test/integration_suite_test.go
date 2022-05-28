package test

import (
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/gruntwork-io/terratest/modules/docker"
	"github.com/gruntwork-io/terratest/modules/git"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	imgTag = "yorinasub17/concourse-gitea-release-resource:test"
)

var (
	accessToken string
	giteaClt    *gitea.Client
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	t := GinkgoT()

	root, err := git.GetRepoRootE(t)
	Ω(err).ShouldNot(HaveOccurred())

	buildOpts := &docker.BuildOptions{Tags: []string{imgTag}}
	Ω(docker.BuildE(t, root, buildOpts)).To(Succeed())

	clt, err := gitea.NewClient(ServerURL, gitea.SetBasicAuth(Username, Password))
	Ω(err).ShouldNot(HaveOccurred())
	giteaClt = clt

	token, _, err := giteaClt.CreateAccessToken(gitea.CreateAccessTokenOption{Name: "IntegrationTestToken"})
	Ω(err).ShouldNot(HaveOccurred())
	accessToken = token.Token
})

var _ = AfterSuite(func() {
	t := GinkgoT()
	Ω(docker.DeleteImageE(t, imgTag, nil)).To(Succeed())

	_, err := giteaClt.DeleteAccessToken("IntegrationTestToken")
	Ω(err).ShouldNot(HaveOccurred())
})
