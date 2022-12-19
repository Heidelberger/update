package main

// Logic adapted from: https://github.com/yitsushi/totp-cli/blob/main/internal/cmd/update.go

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"

	"github.com/kardianos/osext"                   // Find the current Executable and ExecutableFolder.
	grc "github.com/yitsushi/github-release-check" // Check Github repo version
	// manage commands for your CLI tool
)

type infos struct {
	AppRepoOwner string
	AppName      string
	AppVersion   string
}

func newInfos(owner string, name string, ver string) *infos {
	i := infos{}
	i.AppRepoOwner = owner
	i.AppName = name
	i.AppVersion = ver
	return &i
}

// Update structure is the representation of the update command.
type Update struct{}

const (
	binaryChmodValue = 0o755
)

// DownloadError is an error during downloading an update.
type DownloadError struct {
	Message string
}

func (e DownloadError) Error() string {
	return fmt.Sprintf("download error: %s", e.Message)
}

// ImportError is an error during a file import.
type ImportError struct {
	Message string
}

func (e ImportError) Error() string {
	return fmt.Sprintf("import error: %s", e.Message)
}

// GenerateError is an error during code generation.
type GenerateError struct {
	Message string
}

func (e GenerateError) Error() string {
	return fmt.Sprintf("generate error: %s", e.Message)
}

// DeleteError is an error during entry deletion.
type DeleteError struct {
	Message string
}

func (e DeleteError) Error() string {
	return fmt.Sprintf("delete error: %s", e.Message)
}

// Execute is the main function. It will be called on update command.
func (c *Update) Execute(info *infos) {
	hasUpdate, release, _ := grc.Check(info.AppRepoOwner, info.AppName, info.AppVersion)

	if !hasUpdate {
		fmt.Printf("Your %s is up-to-date. \\o/\n", info.AppName)

		return
	}

	var (
		assetToDownload grc.Asset
		found           bool
	)

	for _, asset := range release.Assets {
		if asset.Name == c.buildFilename(release.TagName, info) {
			assetToDownload = asset
			found = true

			break
		}
	}

	if !found {
		fmt.Printf("Your %s is up-to-date. \\o/\n", info.AppName)

		return
	}

	downloadError := c.downloadBinary(assetToDownload.BrowserDownloadURL, info)
	if downloadError != nil {
		fmt.Printf("Error: %s\n", downloadError.Error())
	}

	fmt.Printf("Now you have a fresh new %s \\o/\n", info.AppName)
}

func (c *Update) buildFilename(version string, info *infos) string {
	return fmt.Sprintf("%s-%s-%s-%s.tar.gz", info.AppName, version, runtime.GOOS, runtime.GOARCH)
}

func (c *Update) downloadBinary(uri string, info *infos) error {
	fmt.Println(" -> Download...")

	client := http.Client{}

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, uri, nil)
	if err != nil {
		return DownloadError{Message: err.Error()}
	}

	response, err := client.Do(request)
	if err != nil {
		return DownloadError{Message: err.Error()}
	}

	defer response.Body.Close()

	gzipReader, _ := gzip.NewReader(response.Body)
	defer gzipReader.Close()

	fmt.Println(" -> Extract...")

	tarReader := tar.NewReader(gzipReader)

	_, err = tarReader.Next()
	if err != nil {
		return DownloadError{Message: err.Error()}
	}

	currentExecutable, _ := osext.Executable()
	originalPath := path.Dir(currentExecutable)

	file, err := os.CreateTemp(originalPath, info.AppName)
	if err != nil {
		return DownloadError{Message: err.Error()}
	}

	defer file.Close()

	_, err = io.Copy(file, tarReader) //nolint:gosec // I don't have better option right now.
	if err != nil {
		return DownloadError{Message: err.Error()}
	}

	err = file.Chmod(binaryChmodValue)
	if err != nil {
		return DownloadError{Message: err.Error()}
	}

	err = os.Rename(file.Name(), currentExecutable)
	if err != nil {
		return DownloadError{Message: err.Error()}
	}

	return nil
}

func main() {
	fmt.Printf("hello, world\n")

	// AppRepoOwner defined the owner of the repo on GitHub.
	const AppRepoOwner string = "Heidelberger"

	// AppName defined the application name.
	const AppName string = "update"

	// AppVersion defined current version of this application.
	const AppVersion string = "v1.0.0"

	infos := newInfos(AppRepoOwner, AppName, AppVersion)

	upd := Update{}
	upd.Execute(infos)
}
