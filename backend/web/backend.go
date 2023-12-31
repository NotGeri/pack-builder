package web

import (
	"encoding/json"
	"fmt"
	"geri.dev/pack-builder/checker"
	"geri.dev/pack-builder/config"
	"geri.dev/pack-builder/providers"
	"geri.dev/pack-builder/utils"
	"geri.dev/pack-builder/web/sockets"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Backend struct {
	c         checker.Checker
	cfg       *config.Config
	sessions  map[uuid.UUID]*checker.Session
	downloads map[uuid.UUID]*checker.Package
	upgrader  websocket.Upgrader
}

// NewBackend initializes a new backend object
func NewBackend(cfg *config.Config) (backend Backend) {
	backend = Backend{
		cfg: cfg,

		c: checker.Checker{
			Spigot:         providers.NewSpigotProvider(cfg),
			Modrinth:       providers.NewModrinthProvider(cfg),
			GitHub:         providers.NewGitHubProvider(cfg),
			DirectDownload: providers.NewDirectDownloadProvider(cfg),
		},

		downloads: make(map[uuid.UUID]*checker.Package),

		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return r.Header.Get("Origin") == cfg.Web.Frontend
			},
		},
	}

	if err := backend.LoadSessions(); err != nil {
		backend.sessions = make(map[uuid.UUID]*checker.Session)
	}

	backend.c.PluginProviders = []providers.PluginProvider{
		&backend.c.Spigot,
		&backend.c.Modrinth,
	}
	backend.c.ExternalProviders = []providers.ExternalProvider{
		&backend.c.GitHub,
		&backend.c.DirectDownload,
	}

	return
}

// Todo (notgeri): temporary
func (b *Backend) LoadSessions() (err error) {
	data, err := os.ReadFile("recover.json")
	if err != nil {
		return
	}

	if err = json.Unmarshal(data, &b.sessions); err != nil {
		return
	}

	return
}

// see LoadSessions
func (b *Backend) SaveSessions() (err error) {
	data, err := json.MarshalIndent(b.sessions, "", "  ")
	if err != nil {
		return
	}

	// Write the formatted JSON data to the file
	if err = os.WriteFile("recover.json", data, 0644); err != nil {
		return
	}

	return
}

// getId attempts to parse the request's ID parameter as a UUID
// If it fails, it will reject the request and return nil
func (b *Backend) getId(w http.ResponseWriter, r *http.Request) *uuid.UUID {
	rawId := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawId)
	if err != nil {
		utils.SendJSON(w, 400, utils.Simple{Message: "invalid session ID"})
		return nil
	}

	return &id
}

// getSession uses getId to parse the request's ID and attempts to get the session
// If it fails, it will reject the request and return nil
func (b *Backend) getSession(w http.ResponseWriter, r *http.Request) *checker.Session {
	id := b.getId(w, r)
	if id == nil {
		return nil
	}

	session, ok := b.sessions[*id]
	if !ok {
		utils.SendJSON(w, 404, utils.Simple{Message: "no session found"})
		return nil
	}

	return session
}

// InfoHandler returns some basic information about the checker
func (b *Backend) InfoHandler(w http.ResponseWriter, r *http.Request) {
	utils.SendJSON(w, 200, b.c.GetSupportInfo())
}

// Todo (notgeri):
func (b *Backend) TemporaryHandler(w http.ResponseWriter, r *http.Request) {
	utils.SendJSON(w, 200, b.sessions)
}

// CreationHandler handles creating a new session
func (b *Backend) CreationHandler(w http.ResponseWriter, r *http.Request) {

	// Parse & verify some of the basic request data parts
	var request checker.Request
	if err := render.Bind(r, &request); err != nil {
		utils.SendJSON(w, 400, utils.Simple{Message: err.Error()})
		return
	}

	issues := make([]utils.Tracker, 0)
	for trackerId, link := range request.Links {
		// Verify the ID is a valid UUID
		if _, err := uuid.Parse(trackerId); err != nil {
			issues = append(issues, utils.Tracker{Id: trackerId, Message: "invalid ID"})
			continue
		}

		// Verify it's not duplicate
		count := 0
		for id := range request.Links {
			if trackerId == id {
				count++
			}
		}
		if count > 1 {
			issues = append(issues, utils.Tracker{Id: trackerId, Message: "duplicate ID"})
		}

		// Ensure the links at least look somewhat valid
		if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
			issues = append(issues, utils.Tracker{Id: trackerId, Message: "invalid link"})
			continue
		}
	}

	if len(issues) > 0 {
		utils.SendJSON(w, 400, utils.Simple{
			Message: "invalid links",
			Data:    issues,
		})
		return
	}

	id := uuid.New()
	session := checker.Session{
		Id:      id,
		Request: request,
	}
	session.Initialize()

	b.sessions[id] = &session
	utils.SendJSON(w, 200, map[string]interface{}{
		"id": id.String(),
	})
}

// IndexHandler returns information about a specific session
func (b *Backend) IndexHandler(w http.ResponseWriter, r *http.Request) {
	session := b.getSession(w, r)
	if session == nil {
		return
	}

	utils.SendJSON(w, 200, session)
}

// DownloadHandler handles downloading packages
func (b *Backend) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	session := b.getSession(w, r)
	if session == nil {
		return
	}

	rawId := chi.URLParam(r, "packageId")
	packageId, err := uuid.Parse(rawId)
	if err != nil {
		utils.SendJSON(w, 400, utils.Simple{Message: "invalid package ID"})
		return
	}

	pkg, ok := b.downloads[packageId]
	if !ok {
		utils.SendJSON(w, 404, utils.Simple{Message: "package not found"})
		return
	}

	// Open the file
	file, err := os.Open(pkg.Path)
	if err != nil {
		utils.SendJSON(w, 500, utils.Simple{Message: "error opening file"})
		return
	}

	defer file.Close()

	// Set the headers
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(pkg.Path))

	// Stream the file content to the response
	_, err = io.Copy(w, file)
	if err != nil {
		fmt.Printf("Unable to download package %s: %s\n", packageId, err)
	}
}

// SocketHandler upgrades a GET request to a websocket and keeps track of the channel
// for the specific session, so we can use it to send events
func (b *Backend) SocketHandler(w http.ResponseWriter, r *http.Request) {
	session := b.getSession(w, r)
	if session == nil {
		return
	}

	// Upgrade GET request to a websocket
	ws, err := b.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("unable to initialize websocket: %s", err)
		utils.SendJSON(w, 400, utils.Simple{Message: "unable to initialize websocket"})
		return
	}

	// Keep track of the socket with the session
	session.Sockets = append(session.Sockets, checker.SocketTracker{
		Socket: ws,
		Lock:   &sync.Mutex{},
	})

	// Ensure the socket is cleaned up
	defer func() {
		session.CloseSocket(ws)
	}()

	// Send an OK sign to the socket
	_ = session.BroadcastToSockets(sockets.Connected, nil)

	// Read requests from the client
	failedAttempts := 0
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			if failedAttempts > 10 {
				fmt.Printf("[%s] unable to read websocket message: %s, given up\n", session.Id, err)
				break
			}
			failedAttempts++
			continue
		}

		// Parse the raw message
		raw := strings.Split(string(message), " ")

		// The main command to execute for this session
		command := strings.ToLower(raw[0])

		// Frontend can pass some data as JSON which you can parse as a struct
		rawData := []byte(strings.Join(raw[1:], " "))

		switch sockets.Message(command) {
		case sockets.Preliminary:
			_ = session.BroadcastToSockets(sockets.PreliminaryStart, nil)
			b.c.PreliminaryChecks(session)
			_ = session.BroadcastToSockets(sockets.PreliminaryDone, session)
			break

		case sockets.ToggleLink:
			if session.Links == nil || !session.OverallState.Preliminary {
				return
			}

			var data struct {
				Id    string
				Link  string
				Value bool
			}

			if err := json.Unmarshal(rawData, &data); err != nil {
				return
			}

			id, err := uuid.Parse(data.Id)
			if err != nil {
				return
			}

			link, ok := session.Links[id]
			if !ok || link.Preliminary == nil {
				return
			}

			link.Preliminary.Links[data.Link] = data.Value
			break

		case sockets.Process:
			_ = session.BroadcastToSockets(sockets.ProcessStart, nil)
			b.c.DownloadFiles(session)
			b.c.PostProcessing(session)
			_ = session.BroadcastToSockets(sockets.ProcessDone, session)
			break

		case sockets.Package:
			_ = session.BroadcastToSockets(sockets.PackageStart, nil)
			b.c.Package(session)
			_ = session.BroadcastToSockets(sockets.PackageDone, session)
			break

		case sockets.GetDownload:
			_ = session.BroadcastToSockets(sockets.GetDownloadStart, nil)

			id, err := uuid.Parse(string(rawData))
			if err != nil {
				_ = session.BroadcastToSockets(sockets.GetDownloadError, utils.Simple{Message: err.Error()})
				continue
			}

			pkg, ok := session.Packages[id]
			if !ok {
				_ = session.BroadcastToSockets(sockets.GetDownloadError, utils.Simple{Message: "package not found"})
				continue
			}

			if pkg.Status != checker.Success {
				_ = session.BroadcastToSockets(sockets.GetDownloadError, utils.Simple{Message: "package is not complete"})
				continue
			}

			pkg.Downloadable = true
			b.downloads[id] = pkg
			_ = session.BroadcastToSockets(sockets.GetDownloadDone, session)
			break

		case sockets.Delete:
			session.Delete()
			delete(b.sessions, session.Id)
			break

		default:
			fmt.Printf("Unknown command receieved: %s, %s\n", command, rawData)
			break
		}

		fmt.Printf("[%s] %s\n", session.Id.String(), string(message))

		// Todo (notgeri):
		go b.SaveSessions()
	}
}

// PreliminaryHandler starts the preliminary checks for a session
func (b *Backend) PreliminaryHandler(w http.ResponseWriter, r *http.Request) {
	session := b.getSession(w, r)
	if session == nil {
		return
	}

	// Ensure it has been initialized
	if !session.OverallState.Initialized {
		utils.SendJSON(w, 400, utils.Simple{Message: "the session has now been initialized yet"})
		return
	}

	// Run the checks on a new thread
	go b.c.PreliminaryChecks(session)
	utils.SendJSON(w, 201, nil)
}

// ProcessHandler starts the downloading and post-processing stage for a session
func (b *Backend) ProcessHandler(w http.ResponseWriter, r *http.Request) {
	session := b.getSession(w, r)
	if session == nil {
		return
	}

	// Ensure preliminary checks are done
	if !session.OverallState.Preliminary {
		utils.SendJSON(w, 400, utils.Simple{Message: "preliminary checks have not been run for the session"})
		return
	}

	// Run the downloads and post-processing on a new thread
	go func() {
		b.c.DownloadFiles(session)
		b.c.PostProcessing(session)
	}()
	utils.SendJSON(w, 201, nil)
}

// DeletionHandler deletes a session
func (b *Backend) DeletionHandler(w http.ResponseWriter, r *http.Request) {
	session := b.getSession(w, r)
	if session == nil {
		return
	}

	session.Delete()
	delete(b.sessions, session.Id)
	utils.SendJSON(w, 201, nil)
}
