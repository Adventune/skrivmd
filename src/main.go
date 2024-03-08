package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"adventune/skrivmd/builder"
)

var (
	contentPath string
	debug       bool
	noWatch     bool
)

func main() {
	// Logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("Starting...")

	// Command line flags
	debugF := flag.Bool("debug", false, "Sets log level to debug")
	contentPathF := flag.String("content", "./content", "Path to the content directory")
	noWatchF := flag.Bool("nowatch", false, "Disable the content watcher")
	flag.Parse()

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get working directory")
	}

	// Set the global variables
	debug = *debugF
	contentPath = *contentPathF
	noWatch = *noWatchF

	// Set the log level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	log.Debug().Msg("Debug logging has been enabled")

	// Initialize the builder
	builder.Init(contentPath, wd, noWatch)

	// Serve content build directory
	http.Handle("/", http.FileServer(http.Dir(builder.CONTENT_BUILD_DIR)))

	// Initial content build
	builder.Build()

	// Start the server
	log.Info().Msg("Listening on port 8000")
	http.ListenAndServe(":8000", nil)
}
