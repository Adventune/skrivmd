package builder

import (
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/radovskyb/watcher"
	"github.com/rs/zerolog/log"
)

const (
	CONTENT_BUILD_DIR = "../content-build"
)

var (
	contentPath string
	wd          string
)

// Initializes the builder.
// It sets the content path, working directory and starts the content watcher
func Init(contentPathI, wdI string, noWatch bool) {
	contentPath = contentPathI
	wd = wdI

	if !noWatch {
		startContentWatcher(contentPath)
	}

	// Build the content
	initialBuild()
}

// Starts watching for changes in the content directory.
// Triggers rebuild when a content change is detected.
func startContentWatcher(contentPath string) {
	log.Debug().Str("path", contentPath).Msg("Watching content directory for changes")

	// Create a new file watcher
	w := watcher.New()
	w.SetMaxEvents(1)
	// Only watch for write, create, remove, rename and move events
	w.FilterOps(watcher.Write, watcher.Create, watcher.Remove, watcher.Rename, watcher.Move)

	// Only watch for markdown files
	r := regexp.MustCompile("^.*\\.md")
	w.AddFilterHook(watcher.RegexFilterHook(r, false))

	// Start the watching process
	go func() {
		for {
			select {
			case event := <-w.Event:
				rebuild(event)
			case err := <-w.Error:
				log.Error().Err(err).Msg("Watcher error")
			case <-w.Closed:
				return
			}
		}
	}()

	// Add the content directory to the watcher
	if err := w.AddRecursive(contentPath); err != nil {
		log.Fatal().Err(err).Msg("Failed to add content directory to watcher")
	}

	// Start the watching process - it'll check for changes every 100ms.
	go w.Start(time.Millisecond * 100)
	// Wait for the watcher to start before returning
	w.Wait()
}

// Resets the content-build directory and builds all content files.
func initialBuild() {
	// Build the content
	log.Info().Msg("Building content")

	// Get all content content files
	paths := getContentFiles(contentPath)

	// Create the build directory
	err := os.MkdirAll(CONTENT_BUILD_DIR, os.ModePerm)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create build directory")
	}

	// Remove all files in the build directory
	err = os.RemoveAll(CONTENT_BUILD_DIR)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to remove files from build directory")
	}

	// Build all content files
	for _, path := range paths {
		buildSingleContentFile(path)
	}
}

// Builds a single content file in the given path.
func buildSingleContentFile(path string) {
	log.Debug().Str("path", path).Msg("Building a content file")
	outputDir, err := getBuildPath(path)

	// Create the output directory
	err = os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create output directory")
	}

	// Create index.html into the output directory
	indexFile := filepath.Join(outputDir, "index.html")
	file, err := os.Create(indexFile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create index file")
	}

	defer file.Close()

	// Read the content file
	content, err := os.ReadFile(path)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read content file")
	}

	// Convert the content to html
	content = mdToHTML(content)

	// Write the content to the index file
	_, err = file.Write(content)
	if err != nil {
		log.Error().Err(err).Msg("Failed to write content to index file")
	}
}

// Rebuilds the content when a file change is detected.
// It will remove the old build and create a new one.
// Rebuild builds only the file that has been changed.
func rebuild(event watcher.Event) {
	log.Debug().Str("path", event.Path).Msg("Rebuilding content")

	switch event.Op {
	case watcher.Write, watcher.Create:
		log.Debug().Msg("File has been written to")

		buildSingleContentFile(event.Path)
	case watcher.Remove:
		log.Debug().Msg("File has been removed")

		dir, err := getBuildPath(event.Path)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get build directory")
		}

		err = os.Remove(dir + "/index.html")
		if err != nil {
			log.Error().Err(err).Msg("Failed to remove build")
		}

		cleanEmptyDirs()
	case watcher.Rename, watcher.Move:
		log.Debug().Msg("File has been moved")

		oldBuildPath, err := getBuildPath(event.OldPath)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get old build path")
		}

		newBuildPath, err := getBuildPath(event.Path)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get new build path")
		}

		err = os.MkdirAll(newBuildPath, os.ModePerm)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create new build directory")
		}

		err = os.Rename(oldBuildPath+"/index.html", newBuildPath+"/index.html")
		if err != nil {
			log.Error().Err(err).Msg("Failed to move build path")
		}

		cleanEmptyDirs()
	default:
		log.Debug().Msg("Unknown event type")
	}
}
