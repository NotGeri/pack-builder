package providers

import (
	"geri.dev/pack-builder/config"
	"net/http"
)

type BukkitProvider struct {
	cfg *config.Config
	c   *http.Client
}

func NewBukkitProvider(cfg *config.Config) BukkitProvider {
	return BukkitProvider{
		cfg: cfg,
		c:   &http.Client{},
	}
}

// GetPluginProviderName returns the ID for the provider
func (bp *BukkitProvider) GetPluginProviderName() string {
	return "bukkit"
}

func (bp *BukkitProvider) GetPluginInfoFromLink(link string) (info PluginInfo, err error) {

	// Todo (notgeri):  Bukkit scraper
	/*
					listing container = .primary-content > .project-file-list > .listing-container
					pagination: the listing container > .listing-header .b-pagination-list > li > a
					rows: the listing container > .listing-body > table > tbody > tr
					each row:
				    - name: .project-file-name-container > a
			        - game versions: .project-file-game-version > .version-label, .project-file-game-version > .additional-versions's title tag <div>(?P<version>.*)</div>
		            - download link:: .project-file-download-button > a's href + dev.bukkit.org/ prefix
	*/
	return
}

func (bp *BukkitProvider) GetPluginInfoFromProjectName(name string) (info PluginInfo, err error) {
	// same as with url, just /projects/<id> or /projects/<name>
	return
}
