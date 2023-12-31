package providers

import (
	"errors"
	"geri.dev/pack-builder/config"
	"io"
	"net/http"
)

type DirectDownloadProvider struct {
	cfg *config.Config
	c   *http.Client
}

func NewDirectDownloadProvider(cfg *config.Config) DirectDownloadProvider {
	return DirectDownloadProvider{
		cfg: cfg,
		c:   &http.Client{},
	}
}

// GetExternalProviderName returns the ID for the external provider
func (ddp *DirectDownloadProvider) GetExternalProviderName() string {
	return "direct_download"
}

// GetJARDownloadLinksFromLink attempts to verify that an
// external link is a direct download link to a JAR
func (ddp *DirectDownloadProvider) GetJARDownloadLinksFromLink(link string) (downloadLinks []string, err error) {

	// First, try with a HEAD request
	headResp, err := ddp.c.Head(link)
	if err != nil {
		return
	}

	defer headResp.Body.Close()

	// Check if the Content-Type header indicates a JAR file
	if contentType := headResp.Header.Get("Content-Type"); contentType == "application/java-archive" {
		downloadLinks = []string{link}
		return
	}

	// If HEAD request fails, fall back to a GET request
	getResp, err := ddp.c.Get(link)
	if err != nil {
		return
	}

	defer getResp.Body.Close()

	// Read the first few bytes of the response body
	buf := make([]byte, 4)
	if _, err = io.ReadFull(getResp.Body, buf); err != nil {
		return
	}

	// Check if the bytes match the ZIP file signature (PK)
	if buf[0] == 0x50 && buf[1] == 0x4B {
		downloadLinks = []string{link}
		return
	}

	err = errors.New("link does not point to a valid JAR file")
	return
}
