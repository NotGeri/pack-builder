package providers

import (
	"encoding/json"
	"fmt"
	"geri.dev/pack-builder/config"
	"geri.dev/pack-builder/utils"
	"io"
	"net/http"
	"regexp"
)

var modrinthBaseEndpoint = "https://api.modrinth.com/v2"
var modrinthUserAccessibleEndpoint = "https://modrinth.com"
var modrinthLinkRegex = regexp.MustCompile("https://modrinth\\.com/plugin/(?P<slug>.+)")

type ModrinthProvider struct {
	cfg *config.Config
	c   *http.Client
}

func NewModrinthProvider(cfg *config.Config) ModrinthProvider {
	return ModrinthProvider{
		cfg: cfg,
		c:   &http.Client{},
	}
}

// GetPluginProviderName returns the ID for the provider
func (mp *ModrinthProvider) GetPluginProviderName() string {
	return "modrinth"
}

// makeRequest sends a new Modrinth API request
func (mp *ModrinthProvider) makeRequest(method, url string, result interface{}) error {
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", modrinthBaseEndpoint, url), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", mp.cfg.Credentials.Modrinth.Token)
	req.Header.Set("User-Agent", mp.cfg.Credentials.UserAgent)

	resp, err := mp.c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get resource, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, result)
}

type modrinthPluginInfo struct {
	Id           string   `json:"id"`
	TeamId       string   `json:"team"`
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	GameVersions []string `json:"game_versions"`
	Loaders      []string `json:"loaders"`
	VersionIds   []string `json:"versions"`
	IconUrl      string   `json:"icon_url"`
}

type modrinthPluginFile struct {
	Url      string `json:"url"`
	FileName string `json:"file_name"`
	Size     int64  `json:"size"`
	Primary  bool   `json:"primary"`
}

type modrinthPluginDependency struct {
	VersionId      *string `json:"version_id"`
	ProjectId      *string `json:"project_id"`
	FileName       string  `json:"file_name"`
	DependencyType string  `json:"dependency_type"`
}

type modrinthPluginVersionInfo struct {
	Id            string                     `json:"id"`
	Name          string                     `json:"name"`
	VersionNumber string                     `json:"version_number"`
	Files         []modrinthPluginFile       `json:"files"`
	Dependencies  []modrinthPluginDependency `json:"dependencies"`
	Loaders       []string                   `json:"loaders"`
	GameVersions  []string                   `json:"game_versions"`
}

// GetPluginInfoFromLink attempts to parse the project ID of a link
// and get its details from the Modrinth API.
func (mp *ModrinthProvider) GetPluginInfoFromLink(link string) (info PluginInfo, err error) {

	// Parse the resource ID
	slug := utils.GetRegexGroup(modrinthLinkRegex, "slug", link)
	if slug == "" {
		err = fmt.Errorf("unable to parse Modrinth slug")
		return
	}

	// Get the base project information
	var rawInfo modrinthPluginInfo
	if err = mp.makeRequest("GET", fmt.Sprintf("/project/%s", slug), &rawInfo); err != nil {
		return PluginInfo{}, err
	}

	// Get the version information
	var rawVersions []modrinthPluginVersionInfo
	if err = mp.makeRequest("GET", fmt.Sprintf("/project/%s/version", slug), &rawVersions); err != nil {
		return PluginInfo{}, err
	}

	versions := make([]Version, 0)
	for _, version := range rawVersions {
		var primaryFile *modrinthPluginFile
		for _, file := range version.Files {
			if file.Primary {
				primaryFile = &file
				break
			}
		}

		if primaryFile == nil {
			continue
		}

		versions = append(versions, Version{
			Id:           version.Id,
			Link:         fmt.Sprintf("%s/%s/version/%s", modrinthUserAccessibleEndpoint, rawInfo.Slug, version.Id),
			IsExternal:   false,
			URL:          primaryFile.Url,
			Platforms:    version.Loaders,
			GameVersions: version.GameVersions,
		})
	}

	info = PluginInfo{
		Type:         Modrinth,
		Id:           fmt.Sprintf("%v", rawInfo.Id),
		Link:         fmt.Sprintf("%s/plugin/%s", modrinthUserAccessibleEndpoint, rawInfo.Slug),
		Name:         rawInfo.Title,
		Description:  rawInfo.Description,
		Contributors: rawInfo.TeamId,
		Versions:     versions,
		IconLink:     rawInfo.IconUrl,
	}

	return
}

func (mp *ModrinthProvider) GetPluginInfoFromProjectName(name string) (info PluginInfo, err error) {
	err = fmt.Errorf("fuck you")
	return
}
