package http

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// DownloadFileOverHTTP will retrieve the given URL over HTTP and download the contents to the given destination path.
func DownloadFileOverHTTP(url, destPath string) error {
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file `%s`: HTTP status %d", filepath.Base(destPath), resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
