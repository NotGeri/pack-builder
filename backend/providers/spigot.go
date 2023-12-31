package providers

import (
	"encoding/json"
	"fmt"
	"geri.dev/pack-builder/config"
	"geri.dev/pack-builder/utils"
	"github.com/google/uuid"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var spigotBaseEndpoint = "https://api.spiget.org/v2"
var spigotUserAccessibleEndpoint = "https://spigotmc.org"
var spigotLinkRegex = regexp.MustCompile("https://(?:www\\.)?spigotmc\\.org/resources/.+?\\.(?P<id>[0-9]*)")

type SpigotProvider struct {
	cfg *config.Config
	c   *http.Client
}

type icon struct {
	Url  string `json:"url"`
	Data string `json:"data"`
}

type file struct {
	Id          string    `json:"id"`
	Uuid        uuid.UUID `json:"uuid"`
	Type        string    `json:"type"`
	Size        float64   `json:"size"`
	Url         *string   `json:"url"`
	ExternalUrl *string   `json:"externalUrl"`
}

func (i *file) IsRegular() bool {
	return i.Type == ".jar"
}

func (i *file) IsExternal() bool {
	return i.Type == "external"
}

type spigetInfo struct {
	Id             int64    `json:"id"`
	Name           string   `json:"name"`
	Tag            string   `json:"tag"`
	Contributors   string   `json:"contributors"`
	Premium        bool     `json:"premium"`
	TestedVersions []string `json:"testedVersions"`
	Icon           icon     `json:"icon"`
	File           file     `json:"file"`
}

// ToPluginInfo converts a spigetInfo into a generic PluginInfo struct
func (i spigetInfo) ToPluginInfo() PluginInfo {
	fileUrl := fmt.Sprintf("%s/resources/%v/download", spigotBaseEndpoint, i.Id)
	if i.File.IsExternal() {
		if i.File.ExternalUrl != nil {
			fileUrl = *i.File.ExternalUrl
		} else {
			fileUrl = ""
		}
	}

	return PluginInfo{
		Type:         Spigot,
		Id:           fmt.Sprintf("%v", i.Id),
		Link:         fmt.Sprintf("%s/resources/%v", spigotUserAccessibleEndpoint, i.Id),
		Name:         i.Name,
		Description:  i.Tag,
		Contributors: i.Contributors,
		Premium:      i.Premium,
		Versions: []Version{
			{
				Id:           i.File.Id,
				Link:         fmt.Sprintf("%s/resources/%v/updates", spigotUserAccessibleEndpoint, i.Id),
				IsExternal:   i.File.IsExternal(),
				URL:          fileUrl,
				Platforms:    nil,
				GameVersions: i.TestedVersions,
			},
		},
		IconLink: i.Icon.Url,
	}
}

func NewSpigotProvider(cfg *config.Config) SpigotProvider {
	return SpigotProvider{cfg: cfg, c: &http.Client{}}
}

// GetPluginProviderName returns the ID for the provider
func (sp *SpigotProvider) GetPluginProviderName() string {
	return "spigot"
}

// makeRequest sends a new Spiget API request
func (sp *SpigotProvider) makeRequest(method, url string, result interface{}) error {
	fmt.Println("that's an api call right there") // Todo (notgeri):

	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", spigotBaseEndpoint, url), nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", sp.cfg.Credentials.UserAgent)

	resp, err := sp.c.Do(req)
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

// GetPluginInfoFromLink attempts to parse the project ID of a link
// and get its details from the Spiget API.
// If there are any issues reaching the API or parsing the response,
// we will simply return an error
func (sp *SpigotProvider) GetPluginInfoFromLink(link string) (info PluginInfo, err error) {

	// Parse the resource ID
	id := utils.GetRegexGroup(spigotLinkRegex, "id", link)
	if id == "" {
		err = fmt.Errorf("unable to parse Spigot ID")
		return
	}

	// Get the resource by its ID
	var rawInfo spigetInfo
	if err = sp.makeRequest("GET", fmt.Sprintf("/resources/%s", id), &rawInfo); err != nil {
		return
	} else { // Convert it to a generic plugin info
		info = rawInfo.ToPluginInfo()
	}

	return
}

// GetPluginInfoFromProjectName attempts to get the details of a plugin
// from a project's name. Sadly, if there are several project with the same
// name, we can't guarantee it is the specific one, so we'll just sort by downloads
// and ask the user to confirm
func (sp *SpigotProvider) GetPluginInfoFromProjectName(name string) (info PluginInfo, err error) {

	// Get the resource by its ID
	var plugins []spigetInfo
	if err = sp.makeRequest("GET", fmt.Sprintf("/search/resources/%s?field=name&sort=-downloads", name), &plugins); err != nil {
		return
	} else {
		// Convert the first result that has that exact name into a generic plugin info
		for _, plugin := range plugins {
			if strings.ToLower(plugin.Name) == strings.ToLower(name) {
				info = plugin.ToPluginInfo()
				return
			}
		}

		// Otherwise, we will just fail
		err = fmt.Errorf("no project found with this exact name")
	}

	return
}
