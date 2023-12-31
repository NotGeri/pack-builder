package providers

import (
	"fmt"
	"github.com/hashicorp/go-version"
	"strings"
)

type ExternalProvider interface {
	GetJARDownloadLinksFromLink(string) ([]string, error)
	GetExternalProviderName() string
}

type PluginProvider interface {
	GetPluginInfoFromLink(string) (PluginInfo, error)
	GetPluginInfoFromProjectName(string) (PluginInfo, error)
	GetPluginProviderName() string
}

type ModProvider interface {
	GetModInfoFromLink(string) (PluginInfo, error)
}

type PluginType string

const (
	Spigot   PluginType = "spigot"
	Modrinth PluginType = "modrinth"
)

type Version struct {
	Id           string   `json:"id"`
	Link         string   `json:"link"`
	IsExternal   bool     `json:"is_external"`
	URL          string   `json:"url"`
	Platforms    []string `json:"platforms"`
	GameVersions []string `json:"game_versions"`
}

type PluginInfo struct {
	Type         PluginType `json:"type"`
	Id           string     `json:"id"`
	Link         string     `json:"link"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Contributors string     `json:"contributors"`
	Premium      bool       `json:"premium"`
	Versions     []Version  `json:"versions"`
	IconLink     string     `json:"icon_link"`
}

// IsTestedVersion returns whether a provided version is marked as tested
// by the plugin's author for a file's version.
// As an example, passing 1.20.2 will return true if 1.20 is in the list of tested versions.
func (v *Version) IsTestedVersion(requiredVersion string) (bool, error) {
	baseVer, err := version.NewVersion(requiredVersion)
	if err != nil {
		return false, err
	}

	for _, v := range v.GameVersions {
		ver, err := version.NewVersion(v)
		if err != nil {
			fmt.Printf("Error parsing version '%s': %s, skipping..", v, err)
			continue
		}

		if baseVer.Segments()[0] == ver.Segments()[0] && baseVer.Segments()[1] == ver.Segments()[1] {
			return true, nil
		}
	}

	return false, nil
}

// GetFormattedTestedVersions returns a human-readable list of tested versions
func (v *Version) GetFormattedTestedVersions() string {
	if len(v.GameVersions) == 0 {
		return "-"
	} else {
		return strings.Join(v.GameVersions, ", ")
	}
}
