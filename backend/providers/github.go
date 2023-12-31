package providers

import (
	"context"
	"fmt"
	"geri.dev/pack-builder/config"
	"geri.dev/pack-builder/utils"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"regexp"
	"strings"
)

var githubRegex = regexp.MustCompile("https://github\\.com/(?P<owner>.*)/(?P<repo>.*)/releases(?:/tag/(?P<tag>.*))?")

type GitHubProvider struct {
	cfg    *config.Config
	ctx    context.Context
	client *github.Client
}

// NewGitHubProvider initializes a new GitHub API manager and returns the provider
func NewGitHubProvider(cfg *config.Config) GitHubProvider {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.Credentials.GitHub.Token})
	ctx := context.Background()
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return GitHubProvider{cfg, ctx, client}
}

// GetExternalProviderName returns the ID for the external provider
func (gp *GitHubProvider) GetExternalProviderName() string {
	return "github"
}

// GetReleaseFromLink attempts to get all the release from a repository url
func (ghp *GitHubProvider) GetReleaseFromLink(link string) (release *github.RepositoryRelease, err error) {

	// Parse the owner and repository from the link
	groups := utils.GetRegexGroups(githubRegex, link)
	owner := groups["owner"]
	repo := groups["repo"]
	tag := groups["tag"]
	if owner == "" && repo == "" {
		err = fmt.Errorf("unable to parse repo from link")
		return
	}

	// Attempt to get the repository
	repository, _, err := ghp.client.Repositories.Get(ghp.ctx, owner, repo)
	if err != nil {
		return
	}

	if repository == nil {
		err = fmt.Errorf("no repository found")
		return
	}

	// If we parsed a tag from the link, we will try to look that up specifically
	if tag != "" {
		release, _, err = ghp.client.Repositories.GetReleaseByTag(ghp.ctx, owner, repo, tag)
	}

	// If that fails, we'll fall back to the latest
	if release == nil {
		release, _, err = ghp.client.Repositories.GetLatestRelease(ghp.ctx, owner, repo)
	}

	return
}

// GetJARDownloadLinksFromLink attempts to return a list of download links for a release's JARs
func (ghp *GitHubProvider) GetJARDownloadLinksFromLink(link string) (downloadLinks []string, err error) {

	release, err := ghp.GetReleaseFromLink(link)
	if err != nil {
		return
	}

	if release == nil {
		err = fmt.Errorf("no release found for repository")
		return
	}

	// If we have release, we will gather the JAR assets as potential downloads
	downloadLinks = make([]string, 0)
	for _, asset := range release.Assets {
		if strings.Contains(asset.GetName(), ".jar") {
			downloadLinks = append(downloadLinks, asset.GetBrowserDownloadURL())
		}
	}

	return
}
