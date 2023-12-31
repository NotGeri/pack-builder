package main

import (
	"fmt"
	"geri.dev/pack-builder/config"
	"geri.dev/pack-builder/web"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"os"
	"path"
)

func main() {

	// Load the config
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("unable to load config: %s", err))
	}

	// Initialize the backend
	backend := web.NewBackend(&cfg)

	// Create a new HTTP router
	router := chi.NewRouter()

	// Enable CORS for the frontend
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", cfg.Web.Frontend)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			if r.Method != "OPTIONS" {
				next.ServeHTTP(w, r)
			}
		})
	})

	// This is where we define the routes
	router.Route("/", func(router chi.Router) {
		router.Route("/api", func(router chi.Router) {
			router.Get("/info", backend.InfoHandler)

			router.Route("/sessions", func(router chi.Router) {
				router.Get("/", backend.TemporaryHandler)
				router.Post("/", backend.CreationHandler)

				router.Route("/{id}", func(router chi.Router) {
					router.HandleFunc("/socket", backend.SocketHandler)
					router.Get("/", backend.IndexHandler)
					router.Get("/download/{packageId}", backend.DownloadHandler)
					router.Post("/preliminary", backend.PreliminaryHandler)
					router.Post("/process", backend.ProcessHandler)
					router.Delete("/", backend.DeletionHandler)
				})
			})
		})

		// Serve static files for the frontend
		workingDirectory, _ := os.Getwd()
		fs := http.FileServer(http.Dir(path.Join(workingDirectory, "public")))
		router.Handle("/*", fs)
	})

	// Generate the listening address
	address := fmt.Sprintf("%s:%v", cfg.Web.Address, cfg.Web.Port)

	// Print the endpoint
	protocol := "http"
	if cfg.Web.SSL.Enabled {
		protocol = "https"
	}

	endpoint := fmt.Sprintf("%s://%s", protocol, address)
	log.Printf("Listening: %s\n", endpoint)

	// If we don't have a public URL defined, we will reuse this one
	if cfg.Web.PublicUrl == "" {
		cfg.Web.PublicUrl = endpoint
	}

	// Determine whether to use TLS
	if cfg.Web.SSL.Enabled {
		err = http.ListenAndServeTLS(
			address,
			cfg.Web.SSL.CertPath,
			cfg.Web.SSL.KeyPath,
			router,
		)
	} else {
		err = http.ListenAndServe(
			address,
			router,
		)
	}

	if err != nil {
		log.Fatal("Unable to start webserver:", err)
		return
	}
}
