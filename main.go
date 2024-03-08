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
	buildOnly   bool
)

func main() {
	// Logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("Starting...")

	// Command line flags
	flag.BoolVar(&debug, "debug", false, "Sets log level to debug")
	flag.StringVar(&contentPath, "content", "./content", "Path to the content directory")
	flag.BoolVar(&noWatch, "no-watch", false, "Disable the content watcher")
	flag.BoolVar(&buildOnly, "build-only", false, "Build the content and exit")
	flag.Parse()

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get working directory")
	}

	// Set the log level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	log.Debug().Msg("Debug logging has been enabled")

	// If build only flag is set, build the content and exit
	if buildOnly {
		builder.Init(contentPath, wd, true)
		log.Info().Msg("Build only flag is set. Exiting...")
		return
	}

	// Initialize the builder
	builder.Init(contentPath, wd, noWatch)

	// Serve content build directory
	http.Handle("/", http.FileServer(http.Dir(builder.CONTENT_BUILD_DIR)))

	// Start the server
	log.Info().Msg("Listening on port 8000")
	err = http.ListenAndServe(":8000", nil)
	log.Fatal().Err(err).Msg("Failed to start server")
}
