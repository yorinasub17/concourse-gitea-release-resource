package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	gogitea "code.gitea.io/sdk/gitea"
	"github.com/gruntwork-io/go-commons/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/yorinasub17/concourse-gitea-release-resource/internal/gitea"
	"github.com/yorinasub17/concourse-gitea-release-resource/internal/http"
	"github.com/yorinasub17/concourse-gitea-release-resource/internal/resource"
)

var _ = Describe("Integration Out", func() {
	const (
		defaultBodyStr = "Default body"
		asset1Str      = "This is an uploaded asset"
		asset2Str      = "This is another asset that may be omitted"
	)

	var (
		clt *gogitea.Client

		srcDir       string
		isPreRelease bool

		nameStr   string
		tagStr    string
		idStr     string
		uniqueStr string
		globs     []string

		newRelease *gogitea.Release
	)

	BeforeEach(func() {
		rawClt, err := gitea.NewGiteaClient(ServerURL, accessToken)
		Ω(err).ShouldNot(HaveOccurred())
		clt = rawClt

		tmpDir, err := ioutil.TempDir("", "concourse-gitea-release-resource-outtest-*")
		Ω(err).ShouldNot(HaveOccurred())
		srcDir = tmpDir

		randomStr, err := random.RandomString(6, random.Base62Chars)
		Ω(err).ShouldNot(HaveOccurred())

		uniqueStr = randomStr
		nameStr = uniqueStr
		tagStr = "a" + strings.ToLower(uniqueStr)
	})

	JustBeforeEach(func() {
		outRequest := resource.OutRequest{
			Source: resource.Source{
				GiteaURL:    ServerURL,
				Owner:       Username,
				Repository:  EmptyRepo,
				AccessToken: accessToken,
				PreRelease:  isPreRelease,
			},
			Params: resource.OutParams{
				NamePath:   "name",
				TagPath:    "tag",
				TargetPath: "target",
				Globs:      globs,
			},
		}

		Ω(ioutil.WriteFile(filepath.Join(srcDir, "name"), []byte(nameStr), 0o644)).Should(Succeed())
		Ω(ioutil.WriteFile(filepath.Join(srcDir, "tag"), []byte(tagStr), 0o644)).Should(Succeed())
		Ω(ioutil.WriteFile(filepath.Join(srcDir, "target"), []byte("master"), 0o644)).Should(Succeed())
		Ω(ioutil.WriteFile(filepath.Join(srcDir, "body"), []byte(defaultBodyStr), 0o644)).Should(Succeed())
		outRequest.Params.BodyPath = "body"
		if idStr != "" {
			Ω(ioutil.WriteFile(filepath.Join(srcDir, "id"), []byte(idStr), 0o644)).Should(Succeed())
			outRequest.Params.IDPath = "id"
		}

		jsonBytes, err := json.Marshal(outRequest)
		Ω(err).ShouldNot(HaveOccurred())

		var stdout bytes.Buffer
		cmd := exec.Command(
			"docker", "run",
			"-i", "--rm", "--network", "host",
			"-v", fmt.Sprintf("%s:/input", srcDir),
			imgTag, "/opt/resource/out", "/input",
		)
		cmd.Stdin = bytes.NewReader(jsonBytes)
		cmd.Stdout = &stdout
		cmd.Stderr = os.Stderr
		Ω(cmd.Run()).To(Succeed())

		var output resource.InOutResponse
		outputStr := strings.TrimSpace(stdout.String())
		Ω(json.Unmarshal([]byte(outputStr), &output)).To(Succeed())

		releaseID := output.Version.ID
		rawRel, err := gitea.GetReleaseByID(clt, Username, EmptyRepo, releaseID)
		Ω(err).ShouldNot(HaveOccurred())
		newRelease = rawRel
	})

	// Clear out input parameters for each testcase
	AfterEach(func() {
		Ω(os.RemoveAll(srcDir)).To(Succeed())

		srcDir = ""
		isPreRelease = false
		nameStr = ""
		tagStr = ""
		idStr = ""
		globs = []string{}
		clt = nil
	})

	Context("when creating a new release", func() {
		Context("without assets", func() {
			It("creates release with right data", func() {
				Ω(newRelease.Title).Should(Equal(uniqueStr))
				Ω(newRelease.TagName).Should(Equal(tagStr))
				Ω(newRelease.Note).Should(Equal(defaultBodyStr))
			})
		})

		Context("with assets", func() {
			BeforeEach(func() {
				Ω(os.Mkdir(filepath.Join(srcDir, "assets"), 0o755)).Should(Succeed())
				Ω(ioutil.WriteFile(filepath.Join(srcDir, "assets", "myfile"), []byte(asset1Str), 0o644)).Should(Succeed())
				Ω(ioutil.WriteFile(filepath.Join(srcDir, "assets", "otherfile"), []byte(asset2Str), 0o644)).Should(Succeed())
			})

			Context("without glob", func() {
				It("creates release without assets", func() {
					Ω(newRelease.Title).Should(Equal(uniqueStr))
					Ω(newRelease.TagName).Should(Equal(tagStr))
					Ω(len(newRelease.Attachments)).Should(Equal(0))
				})
			})

			Context("with glob selecting all", func() {
				BeforeEach(func() {
					globs = []string{"assets/myf*", "assets/other*"}
				})

				It("creates release with assets", func() {
					Ω(newRelease.Title).Should(Equal(uniqueStr))
					Ω(newRelease.TagName).Should(Equal(tagStr))
					Ω(len(newRelease.Attachments)).Should(Equal(2))
				})
			})

			Context("with glob omitting all", func() {
				BeforeEach(func() {
					globs = []string{"noassets*"}
				})

				It("creates release without assets", func() {
					Ω(newRelease.Title).Should(Equal(uniqueStr))
					Ω(newRelease.TagName).Should(Equal(tagStr))
					Ω(len(newRelease.Attachments)).Should(Equal(0))
				})
			})

			Context("with glob selecting some", func() {
				BeforeEach(func() {
					globs = []string{"assets/myf*"}
				})

				It("creates release with one asset", func() {
					Ω(newRelease.Title).Should(Equal(uniqueStr))
					Ω(newRelease.TagName).Should(Equal(tagStr))
					Ω(len(newRelease.Attachments)).Should(Equal(1))

					attc := newRelease.Attachments[0]
					Ω(attc.Name).Should(Equal("myfile"))

					tmpFile, err := ioutil.TempFile("", "")
					Ω(err).ShouldNot(HaveOccurred())
					tmpFile.Close()
					defer os.Remove(tmpFile.Name())

					Ω(http.DownloadFileOverHTTP(attc.DownloadURL, tmpFile.Name())).Should(Succeed())

					Ω(ioutil.ReadFile(tmpFile.Name())).Should(Equal([]byte(asset1Str)))
				})
			})
		})
	})

	Context("when updating an existing release", func() {
		var existingID int64

		BeforeEach(func() {
			opts := gitea.CreateReleaseOpts{
				Owner:  Username,
				Repo:   EmptyRepo,
				Tag:    tagStr,
				Title:  "Previous release",
				Target: "master",
				Body:   "Previously created release for tag",
			}
			rel, err := gitea.CreateRelease(clt, opts)
			Ω(err).ShouldNot(HaveOccurred())
			existingID = rel.ID
		})

		Context("without new assets", func() {
			It("updates existing release", func() {
				Ω(newRelease.ID).Should(Equal(existingID))
				Ω(newRelease.Title).Should(Equal(uniqueStr))
				Ω(newRelease.TagName).Should(Equal(tagStr))
				Ω(newRelease.Note).Should(Equal(defaultBodyStr))
				Ω(len(newRelease.Attachments)).Should(Equal(0))
			})
		})

		Context("with new assets", func() {
			BeforeEach(func() {
				Ω(os.Mkdir(filepath.Join(srcDir, "assets"), 0o755)).Should(Succeed())
				Ω(ioutil.WriteFile(filepath.Join(srcDir, "assets", "myfile"), []byte(asset1Str), 0o644)).Should(Succeed())
				Ω(ioutil.WriteFile(filepath.Join(srcDir, "assets", "otherfile"), []byte(asset2Str), 0o644)).Should(Succeed())
				globs = []string{"assets/*"}
			})

			It("updates release with assets", func() {
				Ω(newRelease.ID).Should(Equal(existingID))
				Ω(newRelease.Title).Should(Equal(uniqueStr))
				Ω(newRelease.TagName).Should(Equal(tagStr))
				Ω(newRelease.Note).Should(Equal(defaultBodyStr))
				Ω(len(newRelease.Attachments)).Should(Equal(2))
			})
		})

		Context("that has existing assets", func() {
			BeforeEach(func() {
				tmpF, err := ioutil.TempFile("", "")
				Ω(err).ShouldNot(HaveOccurred())
				_, writeErr := tmpF.Write([]byte("hello world"))
				Ω(writeErr).ShouldNot(HaveOccurred())
				Ω(gitea.UploadReleaseAssetFromPath(clt, tmpF.Name(), Username, EmptyRepo, existingID)).Should(Succeed())
			})

			Context("without new assets", func() {
				It("updates existing release", func() {
					Ω(newRelease.ID).Should(Equal(existingID))
					Ω(newRelease.Title).Should(Equal(uniqueStr))
					Ω(newRelease.TagName).Should(Equal(tagStr))
					Ω(newRelease.Note).Should(Equal(defaultBodyStr))
					Ω(len(newRelease.Attachments)).Should(Equal(1))
				})
			})

			Context("with new assets", func() {
				BeforeEach(func() {
					Ω(os.Mkdir(filepath.Join(srcDir, "assets"), 0o755)).Should(Succeed())
					Ω(ioutil.WriteFile(filepath.Join(srcDir, "assets", "myfile"), []byte(asset1Str), 0o644)).Should(Succeed())
					Ω(ioutil.WriteFile(filepath.Join(srcDir, "assets", "otherfile"), []byte(asset2Str), 0o644)).Should(Succeed())
					globs = []string{"assets/*"}
				})

				It("updates release with assets", func() {
					Ω(newRelease.ID).Should(Equal(existingID))
					Ω(newRelease.Title).Should(Equal(uniqueStr))
					Ω(newRelease.TagName).Should(Equal(tagStr))
					Ω(newRelease.Note).Should(Equal(defaultBodyStr))
					Ω(len(newRelease.Attachments)).Should(Equal(3))
				})
			})
		})
	})
})
