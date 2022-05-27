package gitea

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
)

// NewGiteaClient returns an authenticated gitea API client for the given server URL.
func NewGiteaClient(serverURL, accessToken string) (*gitea.Client, error) {
	return gitea.NewClient(serverURL, gitea.SetToken(accessToken))
}

// pageLinks is a struct representing the pagination links header in a Gitea API response.
type pageLinks struct {
	limit     int
	firstPage int
	lastPage  int
	prevPage  *int
	nextPage  *int
}

// parseLinks will parse the Link header and return pagination links that can be used to navigate pages as returned by
// the Gitea API.
func parseLinks(response *http.Response) (*pageLinks, error) {
	out := &pageLinks{}

	if links, ok := response.Header["Link"]; ok && len(links) > 0 {
		for _, link := range strings.Split(links[0], ",") {
			segments := strings.Split(strings.TrimSpace(link), ";")

			// link must at least have href and rel
			if len(segments) < 2 {
				continue
			}

			// ensure href is properly formatted
			if !strings.HasPrefix(segments[0], "<") || !strings.HasSuffix(segments[0], ">") {
				continue
			}

			// try to pull out page parameter
			url, err := url.Parse(segments[0][1 : len(segments[0])-1])
			if err != nil {
				continue
			}

			q := url.Query()
			pageStr := q.Get("page")
			page, err := strconv.Atoi(pageStr)
			if err != nil {
				return nil, err
			}

			limitStr := q.Get("limit")
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				return nil, err
			}

			if out.limit == 0 {
				out.limit = limit
			} else if out.limit != limit {
				return nil, fmt.Errorf("Unexpected limit on links returned by gitea: expected %d, got %d", out.limit, limit)
			}

			for _, segment := range segments[1:] {
				switch strings.TrimSpace(segment) {
				case `rel="next"`:
					out.nextPage = &page
				case `rel="prev"`:
					out.prevPage = &page
				case `rel="first"`:
					out.firstPage = page
				case `rel="last"`:
					out.lastPage = page
				}
			}
		}
	}

	return out, nil
}
