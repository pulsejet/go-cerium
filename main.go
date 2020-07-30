package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/gorilla/mux"
	"github.com/pulsejet/go-cerium/controllers"

	"github.com/joho/godotenv"
)

// spaHandler implements the http.Handler interface, so we can use it
// to respond to HTTP requests. The path to the static directory and
// path to the index file within that static directory are used to
// serve the SPA in the given static directory.
type spaHandler struct {
	staticPath string
	indexPath  string
}

// ServeHTTP inspects the URL path to locate a file within the static dir
// on the SPA handler. If a file is found, it will be served. If not, the
// file located at the index path on the SPA handler will be served. This
// is suitable behavior for serving an SPA (single page application).
func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// get the absolute path to prevent directory traversal
	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		// if we failed to get the absolute path respond with a 400 bad request
		// and stop
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// prepend the path with the path to the static directory
	path = filepath.Join(h.staticPath, path)

	// check whether a file exists at the given path
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		// file does not exist, serve index.html
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	} else if err != nil {
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// otherwise, use http.FileServer to serve the static dir
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}

func main() {
	// Load configuration
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Logging options
	customFormatter := new(log.TextFormatter)
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
	logLevel, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)

	// Create new router
	router := mux.NewRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Info("Started server on port ", port)
	rand.Seed(time.Now().UnixNano())

	// Handle API calls
	router.HandleFunc("/api/form", controllers.CreateForm).Methods("POST")
	router.HandleFunc("/api/forms", controllers.GetAllForms).Methods("GET")
	router.HandleFunc("/api/form/{id}", controllers.CreateForm).Methods("PUT")
	router.HandleFunc("/api/form/{id}", controllers.GetForm).Methods("GET")
	router.HandleFunc("/api/form/{id}", controllers.DeleteForm).Methods("DELETE")
	router.HandleFunc("/api/response/{formid}", controllers.CreateResponse).Methods("POST")
	router.HandleFunc("/api/responses/{formid}", controllers.GetResponses).Methods("POST")

	// Handle auth API calls
	router.HandleFunc("/api/login", controllers.Login).Methods("POST", "GET")
	router.HandleFunc("/api/logout", controllers.Logout).Methods("GET")

	// Handlse SPA
	spa := spaHandler{staticPath: "dist/cerium", indexPath: "index.html"}
	router.PathPrefix("/").Handler(spa)

	// Start listening
	err = http.ListenAndServe(":"+port, router)
	if err != nil {
		fmt.Print(err)
	}
}
