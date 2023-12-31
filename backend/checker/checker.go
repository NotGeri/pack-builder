package checker

import (
	"encoding/json"
	"fmt"
	"geri.dev/pack-builder/providers"
	"geri.dev/pack-builder/utils"
	"geri.dev/pack-builder/web/sockets"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-version"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
)

type Checker struct {
	Spigot            providers.SpigotProvider
	Modrinth          providers.ModrinthProvider
	GitHub            providers.GitHubProvider
	DirectDownload    providers.DirectDownloadProvider
	PluginProviders   []providers.PluginProvider
	ExternalProviders []providers.ExternalProvider
}

type SocketTracker struct {
	Socket *websocket.Conn
	Lock   *sync.Mutex
}

type packageType string

const (
	Client packageType = "client"
	Server packageType = "server"
	Misc   packageType = "misc"
)

type Package struct {
	Session      *Session `json:"-"`
	Downloadable bool     `json:"downloadable"`

	Status  status      `json:"status"`
	Message string      `json:"message"`
	Name    string      `json:"name"`
	Type    packageType `json:"type"`
	Path    string      `json:"-"`
	Size    int64       `json:"size"`
}

// Session represents a session with a user that is trying to create
// a mod or plugin pack
type Session struct {
	Id                 uuid.UUID              `json:"id"`
	WorkingDirectory   string                 `json:"-"`
	DownloadsDirectory string                 `json:"-"`
	Packages           map[uuid.UUID]*Package `json:"packages"`

	Request Request `json:"request"`

	Sockets []SocketTracker `json:"-"`

	OverallState OverallState         `json:"overall_state"`
	Links        map[uuid.UUID]*State `json:"links"`
}

// OverallState represents the overall state of a session
type OverallState struct {
	Initialized    bool `json:"initialized"`
	Preliminary    bool `json:"preliminary"`
	Download       bool `json:"download"`
	PostProcessing bool `json:"post_processing"`
	Package        bool `json:"package"`
	Deleted        bool `json:"deleted"`
}

// Initialize initializes a session
func (s *Session) Initialize() {

	// Initialize the sockets list
	s.Sockets = make([]SocketTracker, 0)

	// Convert all the IDs that have already been verified as valid UUIDs
	s.Links = make(map[uuid.UUID]*State)
	for rawId, link := range s.Request.Links {
		id := uuid.MustParse(rawId)
		s.Links[id] = &State{
			Id:   id,
			Link: link,
		}
	}

	// Ensure the working folders exist
	baseDirectory, _ := os.Getwd()
	s.WorkingDirectory = path.Join(baseDirectory, s.Id.String())
	s.DownloadsDirectory = path.Join(s.WorkingDirectory, "downloads")
	_ = os.MkdirAll(s.DownloadsDirectory, 0660)

	s.OverallState = OverallState{Initialized: true}
}

// Delete cleans up a session
func (s *Session) Delete() {
	if err := os.RemoveAll(s.WorkingDirectory); err != nil {
		fmt.Printf("Unable to clean up session folder %s: %s\n", s.Id, err)
	}

	// Close all sockets
	_ = s.BroadcastToSockets(sockets.Deleted, nil)
	s.CloseSockets()

	s.OverallState.Deleted = true
}

// CloseSocket closes a specific websocket and stops tracking it
func (s *Session) CloseSocket(ws *websocket.Conn) {
	for i, tracker := range s.Sockets {
		if tracker.Socket == ws {
			_ = tracker.Socket.Close()
			if len(s.Sockets) > 1 {
				s.Sockets = append(s.Sockets[:i], s.Sockets[i+1:]...)
			} else {
				s.Sockets = make([]SocketTracker, 0)
			}
			break
		}
	}
}

// CloseSockets calls CloseSocket on all sockets
func (s *Session) CloseSockets() {
	for _, tracker := range s.Sockets {
		s.CloseSocket(tracker.Socket)
	}
}

// BroadcastToSockets sends a message to all active sockets
func (s *Session) BroadcastToSockets(message sockets.Message, rawData interface{}) error {

	// If we have additional data, we'll format it as JSON
	payload := string(message)
	if rawData != nil {
		data, err := json.Marshal(rawData)
		if err != nil {
			return err
		}

		payload = fmt.Sprintf("%s %s", message, data)
	}

	// Send the message to each socket
	for _, tracker := range s.Sockets {

		// If the socket is in an errored state, we'll clean it up
		if tracker.Socket == nil {
			s.CloseSocket(tracker.Socket)
			continue
		}

		// Send the payload as text
		tracker.Lock.Lock()
		if err := tracker.Socket.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
			fmt.Printf("[%s] unable to send to websocket: %s\n", s.Id, err)
		}
		tracker.Lock.Unlock()
	}

	return nil
}

// Request represents a mod or plugin pack creation request made by a user
type Request struct {
	Mode            modeType          `json:"-"`
	Platform        platformType      `json:"platform"`
	PlatformVersion string            `json:"platform_version"`
	GameVersion     string            `json:"game_version"`
	Links           map[string]string `json:"links"`
}

// Bind is called when we want to parse some JSON as this struct
func (request *Request) Bind(r *http.Request) error {

	// Ensure we have some links
	if request.Links == nil || len(request.Links) == 0 {
		return fmt.Errorf("no links provided")
	}

	// Ensure the platform is valid
	if !request.Platform.isValid() {
		return fmt.Errorf("invalid platform: %s", request.Platform)
	}

	// Set the mode based on the platform
	request.Mode = request.Platform.getMode()

	// Parse the versions as semver
	if request.Mode == Mods {
		if _, err := version.NewVersion(request.PlatformVersion); err != nil {
			return fmt.Errorf("invalid platform version: %s", request.PlatformVersion)
		}
	}
	if _, err := version.NewVersion(request.GameVersion); err != nil {
		return fmt.Errorf("invalid game version: %s", request.GameVersion)
	}

	return nil
}

// State represents the state of a single link
type State struct {
	Id             uuid.UUID       `json:"id"`
	Link           string          `json:"link"`
	Preliminary    *Preliminary    `json:"preliminary"`
	Download       *Download       `json:"download"`
	PostProcessing *PostProcessing `json:"post_processing"`
}

// Preliminary represents a result for a specific link
// in the preliminary checking stage
type Preliminary struct {
	Status         status                       `json:"status"`
	Error          sockets.ErrorType            `json:"error"`
	Message        string                       `json:"message"`
	FailedAttempts map[string]map[string]string `json:"failed_attempts,omitempty"`
	PluginInfo     *providers.PluginInfo        `json:"plugin_info"`
	Links          map[string]bool              `json:"links"`
	Certain        bool                         `json:"certain"`
}

// Download represents the state of a specific link
// in the download stage
type Download struct {
	Status  status `json:"status"`
	Message string `json:"message"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
}

// PostProcessing represents the state of a specific link
// after the download stage
type PostProcessing struct {
	Dependencies []Dependency `json:"dependencies,omitempty"`
}

// Dependency represents the status of a single found dependency
type Dependency struct {
	Name string `json:"name"`

	// If the dependency is one of the other plugins
	OtherPlugin bool `json:"other_plugin"`

	// The status of finding it online
	Search *Preliminary `json:"search"`

	// The status of the downloading
	Download *Download `json:"download"`
}

type modeType string

const (
	Plugins modeType = "plugins"
	Mods    modeType = "mods"
)

// Represents the target platform
type platformType string

const (
	Spigot   platformType = "spigot"
	Fabric   platformType = "fabric"
	Quilt    platformType = "quilt"
	Forge    platformType = "forge"
	NeoForge platformType = "neoforge"
)

// platformInfo represents some basic information about a platform
type platformInfo struct {
	Name             string   `json:"name"`
	PlatformVersions []string `json:"platform_versions"`
	GameVersions     []string `json:"game_versions"`
}

// isValid returns true if the value is in the enum
// Curse you Go!
func (pt platformType) isValid() bool {
	switch pt {
	case Spigot, Fabric, Quilt, Forge, NeoForge:
		return true
	default:
		return false
	}
}

// getMode returns the mode for the platform
func (pt platformType) getMode() modeType {
	switch pt {
	case Spigot:
		return Plugins
	case Fabric, Quilt, Forge, NeoForge:
		return Mods
	}
	panic("invalid platform type: " + pt)
}

// Represents the status of an operation
type status string

const (
	Success status = "success"
	Warning status = "warning"
	Error   status = "error"
)

// PreliminaryChecks goes through each link and using all the available providers,
// attempts to parse it and retrieve its basic information and a list of possible downloads
func (c *Checker) PreliminaryChecks(session *Session) {

	switch session.Request.Mode {

	case Plugins:
		for _, state := range session.Links {
			result := c.getPluginInformation(session, getPluginInformationOptions{checkWithLink: true, link: state.Link})
			state.Preliminary = &result
			_ = session.BroadcastToSockets(sockets.PreliminaryStep, state)
		}
		break

	case Mods:
		break
	}

	session.OverallState.Preliminary = true
	return
}

// getPluginInformationOptions allows us to download plugins
// based on just a link or just a name, or both
type getPluginInformationOptions struct {
	// Check by project links
	checkWithLink bool
	link          string

	// Check by project names
	checkWithName bool
	name          string

	// Whether the name or link is already an external one
	external bool
}

// getPluginInformation goes through all of our plugin providers
// and attempts to get the plugin information and the necessary version
// for a specific session and a link or project name
func (c *Checker) getPluginInformation(session *Session, options getPluginInformationOptions) (result Preliminary) {

	// New opportunities n all that
	result.Status = Success

	// Store each failed attempt for all providers, in case none of them work
	result.FailedAttempts = make(map[string]map[string]string)
	for _, provider := range c.PluginProviders {
		result.FailedAttempts[provider.GetPluginProviderName()] = make(map[string]string)
	}
	for _, provider := range c.ExternalProviders {
		result.FailedAttempts[provider.GetExternalProviderName()] = make(map[string]string)
	}

	// Try each provider until one of them can handle the link
	var info *providers.PluginInfo

	// First try it as a link
	if options.checkWithLink {
		for _, provider := range c.PluginProviders {
			if i, err := provider.GetPluginInfoFromLink(options.link); err != nil { // Store the failed attempt
				result.FailedAttempts[provider.GetPluginProviderName()]["link"] = err.Error()
				continue
			} else {
				info = &i
			}
		}
	}

	// If none of them were able to parse it as a link, we
	// will try to look it up as a project name
	if info == nil && options.checkWithName {
		for _, provider := range c.PluginProviders {
			if i, err := provider.GetPluginInfoFromProjectName(options.name); err != nil { // Store the failed attempt
				result.FailedAttempts[provider.GetPluginProviderName()]["name"] = err.Error()
				continue
			} else {
				info = &i
			}
		}
	}

	if info == nil {
		result.Status = Error
		result.Message = "none of the providers were able to handle the link"
		return
	}

	result.PluginInfo = info

	// Try each file we got from the provider until we find
	// one that matches our version
	for _, version := range info.Versions {

		// Check if one of the supported platform match
		// with what we are looking for
		// Some providers, like Spigot do not specify this, so
		// we will just skip it at this stage
		if version.Platforms != nil {
			loaderFound := false
			for _, loader := range version.Platforms {
				if strings.ToLower(loader) == string(session.Request.Platform) {
					loaderFound = true
					break
				}
			}
			if !loaderFound {
				continue
			}
		}

		// Check if the supported versions include what we are looking for
		// Some older plugins will work just fine but do not have versions specified,
		// so we will just skip it
		// Todo (notgeri): Add a warning if there aren't any others, so the user can decide if they want to include it
		if version.GameVersions != nil && len(version.GameVersions) > 0 {
			if isTested, err := version.IsTestedVersion(session.Request.GameVersion); err != nil || !isTested {
				continue
			}
		}

		// Let's try to get a working direct download link
		// If it's not a regular a direct .jar link,
		// we will try the other providers, such as GitHub
		if !version.IsExternal {
			result.Certain = true
			result.Links = map[string]bool{version.URL: true}
			return
		}

		// We will not try external providers for links
		// that already are
		if version.IsExternal && options.external {
			continue
		}

		// Ensure we have an external URL
		if version.URL == "" {
			result.Status = Error
			result.Message = "no external URL found for external resource"
			continue
		}

		// Sometimes developers link people from Spigot to Modrinth or similar,
		// so first, try each primary provider
		options.external = true
		primaryResult := c.getPluginInformation(session, options)
		if primaryResult.Status == Success {
			result.Status = Success
			result.PluginInfo = primaryResult.PluginInfo
			result.Links = primaryResult.Links
			return
		}

		// Todo (notgeri): Add support for more, such as:
		// - https://gitlab.com/
		// - https://essentialsx.net/downloads.html

		// If that does not work, we will try each external provider
		for _, provider := range c.ExternalProviders {
			rawLinks, err := provider.GetJARDownloadLinksFromLink(version.URL)
			if err != nil {
				result.FailedAttempts[provider.GetExternalProviderName()]["name"] = err.Error()
				continue
			}

			links := make(map[string]bool)
			for _, link := range rawLinks {
				links[link] = true
			}

			result.Status = Success
			result.Certain = false
			result.Links = links
			return
		}
	}

	result.Status = Error
	result.Error = sockets.NoSuitableVersion
	return
}

// DownloadFiles attempts to download the fetched release
// for all links in a session
func (c *Checker) DownloadFiles(session *Session) {

	// Batch the download jobs to be done concurrently
	batchSize := 5
	batch := 0
	var wg sync.WaitGroup

	for linkId, state := range session.Links {

		wg.Add(1)
		go func(linkId uuid.UUID, state *State) {
			defer func() {
				_ = session.BroadcastToSockets(sockets.ProcessStep, state)
				wg.Done()
			}()

			if state.Preliminary.Status != Success {
				session.Links[linkId].Download = &Download{
					Status:  Error,
					Message: "no download link from previous stage",
				}
				return
			}

			// Try each link until one succeeds or we run out
			for availableLink, shouldUse := range state.Preliminary.Links {
				if !shouldUse {
					continue
				}

				// Download and verify the JAR // Todo (notgeri): we should use the name that is provided
				result := c.downloadAndVerifyJar(availableLink, session.DownloadsDirectory, state.Preliminary.PluginInfo.Name+".jar")

				// If the download was successful, we have nothing else to do here
				session.Links[linkId].Download = &result
				if result.Status == Success {
					return
				}
			}

			session.Links[linkId].Download = &Download{
				Status:  Error,
				Message: "none of the downloads worked",
			}

		}(linkId, state)

		// Wait for the batch to complete
		batch++
		if batch >= batchSize {
			wg.Wait()
			batch = 0
		}
	}

	// Wait for the last batch if it wasn't full
	wg.Wait()
	session.OverallState.Download = true
	return
}

// downloadAndVerifyJar downloads to a specific path and verifies the link as a JAR
// This is done just with a simple size check and by checking the magic bytes
func (c *Checker) downloadAndVerifyJar(link, folderPath, fileName string) (result Download) {

	result.Status = Success
	fullPath := path.Join(path.Join(folderPath, fileName))

	// Download the file
	resp, err := http.Get(link)
	if err != nil {
		result.Status = Error
		result.Message = fmt.Sprintf("error downloading: %s", err)
		return
	}

	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(fullPath)
	if err != nil {
		result.Status = Error
		result.Message = fmt.Sprintf("error creating file: %s", err)
		return
	}

	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		result.Status = Error
		result.Message = fmt.Sprintf("error writing to file: %s", err)
		return
	}

	// Check file size
	fi, err := os.Stat(fullPath)
	if err != nil {
		result.Status = Error
		result.Message = fmt.Sprintf("unable to check file: %s", err)
		return
	}

	result.Path = fullPath
	result.Size = fi.Size()
	if fi.Size() == 0 {
		result.Status = Error
		result.Message = fmt.Sprintf("file is empty")
		return
	}

	// Check magic bytes
	file, err := os.Open(fullPath)
	if err != nil {
		result.Status = Error
		result.Message = fmt.Sprintf("unable to open file: %s", err)
		return
	}

	defer file.Close()

	magicBytes := make([]byte, 4)
	if _, err := file.Read(magicBytes); err != nil {
		result.Status = Error
		result.Message = fmt.Sprintf("unable to read magic bytes: %s", err)
		return
	}

	// Magic bytes for JAR files (PK3 4)
	if magicBytes[0] != 0x50 || magicBytes[1] != 0x4B || magicBytes[2] != 0x03 || magicBytes[3] != 0x04 {
		result.Status = Error
		result.Message = fmt.Sprintf("not a valid JAR file, magic bytes: %s", magicBytes)
		return
	}

	return
}

// PostProcessing handles any remaining steps, such as checking for
// additional dependencies, cleaning up, and so on
func (c *Checker) PostProcessing(session *Session) {
	switch session.Request.Mode {
	case Plugins: // For plugins, we will check the plugin.yml for any dependencies
		c.checkPluginDependencies(session)
		break
	}

	session.OverallState.PostProcessing = true
}

// CheckPluginDependencies goes through each downloaded
// JAR file and checks if there are any missing hard
// dependencies in the plugin.yml config file
func (c *Checker) checkPluginDependencies(session *Session) {

	// Go through each plugin JAR and check their name and dependencies
	downloadedPlugins := make(map[uuid.UUID]string)
	requiredDependencies := make(map[uuid.UUID][]string)

	for linkId, state := range session.Links {
		result := state.Download
		if result.Status != Success {
			continue
		}

		plugin, err := utils.ParsePluginYaml(result.Path)
		if err != nil {
			fmt.Printf("Unable to parse plugin %s (%s): %s\n", linkId, result.Path, err)
			continue
		}

		if plugin.Name == "" {
			fmt.Printf("Plugin name was somehow empty %s\n", linkId)
			continue
		}

		// Ensure the name and all the dependencies are in lowercase
		downloadedPlugins[linkId] = strings.ToLower(plugin.Name)
		if len(plugin.Depends) > 0 {
			pluginDependencies := make([]string, 0)
			for _, dependency := range plugin.Depends {
				pluginDependencies = append(pluginDependencies, strings.ToLower(dependency))
			}

			requiredDependencies[linkId] = pluginDependencies
		}
	}

	// Go through each plugin's dependencies and check if there are any that are missing
	downloadedDependencies := make(map[string]Dependency)
	for parentId, pluginDependencies := range requiredDependencies {
		dependencies := make([]Dependency, 0)

		for _, dependencyName := range pluginDependencies {
			dependency := Dependency{
				Name: dependencyName,
			}

			// See if it's one we already downloaded
			downloadedDependency, found := downloadedDependencies[dependency.Name]
			if found {
				dependency.OtherPlugin = downloadedDependency.OtherPlugin
				dependency.Search = downloadedDependency.Search
				dependency.Download = downloadedDependency.Download
			}

			// See if it's one of the other plugins
			if !found {
				for _, downloadedPlugin := range downloadedPlugins {
					if dependency.Name == downloadedPlugin {
						found = true
						dependency.OtherPlugin = true
						break
					}
				}
			}

			// If it's still not found; we will attempt to download it
			if !found {
				fmt.Printf("Missing dependency: Plugin %s requires %s\n", downloadedPlugins[parentId], dependency)

				searchResult := c.getPluginInformation(session, getPluginInformationOptions{checkWithName: true, name: dependency.Name})
				dependency.Search = &searchResult

				if dependency.Search.Status == Success {

					// Try each link until one succeeds or we run out
					for availableLink := range dependency.Search.Links {

						// Let's give it a name
						fileName := dependency.Name
						if dependency.Search.PluginInfo != nil {
							fileName = dependency.Search.PluginInfo.Name
						}

						// Download and verify the JAR
						downloadResult := c.downloadAndVerifyJar(availableLink, session.DownloadsDirectory, fileName+".jar")
						dependency.Download = &downloadResult

						// Store it, so we don't download it again
						downloadedDependencies[dependencyName] = dependency
					}
				}
			}

			dependencies = append(dependencies, dependency)
		}

		state := PostProcessing{Dependencies: dependencies}
		session.Links[parentId].PostProcessing = &state
	}
}

// Package finalizes the files
func (c *Checker) Package(session *Session) {
	switch session.Request.Mode {
	case Plugins:

		pack := Package{
			Session: session,
			Status:  Success,
			Name:    "Plugin Pack",
			Type:    Server,
		}

		// Create a ZIP
		info, err := utils.ZipFolder(path.Join(session.WorkingDirectory, "pack.zip"), session.DownloadsDirectory)
		if err != nil {
			pack.Status = Error
			pack.Message = err.Error()
		} else {
			pack.Size = info.Size
			pack.Path = info.Path
		}

		session.Packages = make(map[uuid.UUID]*Package)
		session.Packages[uuid.New()] = &pack
		break
	}

	session.OverallState.Package = true
}

// Todo (notgeri): fetch these once a day or something
func (c *Checker) GetSupportInfo() utils.H {
	versions := []string{"1.8.9", "1.12.2", "1.16.5", "1.18.2", "1.20.4"}
	fakeVersions := []string{"0.15.3", "0.15.2", "0.15.1"}
	return utils.H{
		"platforms": map[platformType]platformInfo{
			Spigot:   {Name: "Spigot", GameVersions: []string{"1.8.8", "1.18.2", "1.20.4"}},
			Fabric:   {Name: "Fabric", GameVersions: versions, PlatformVersions: fakeVersions},
			Quilt:    {Name: "Quilt", GameVersions: versions, PlatformVersions: fakeVersions},
			Forge:    {Name: "Forge", GameVersions: versions, PlatformVersions: fakeVersions},
			NeoForge: {Name: "NeoForge", GameVersions: versions, PlatformVersions: fakeVersions},
		},
	}
}
