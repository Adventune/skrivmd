package main

import (
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/radovskyb/watcher"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	CONTENT_BUILD_DIR = "../content-build"
)

var (
	contentPath string
	debug       bool
	noWatch     bool
	wd          string
)

func main() {
	// Logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("Booting up")

	// Command line flags
	debugF := flag.Bool("debug", false, "Sets log level to debug")
	contentPathF := flag.String("content", "./content", "Path to the content directory")
	noWatchF := flag.Bool("nowatch", false, "Disable the content watcher")
	flag.Parse()

	// Set the global variables
	debug = *debugF
	contentPath = *contentPathF
	noWatch = *noWatchF
	wdRes, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get working directory")
	}
	wd = wdRes

	// Set the log level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	log.Debug().Msg("Debug logging has been enabled")

	// Get current content paths
	paths := getContentFiles(contentPath)

	// Serve built content
	http.Handle("/", http.FileServer(http.Dir(CONTENT_BUILD_DIR)))

	// Start the content watcher
	if !noWatch {
		startContentWatcher(contentPath)
	}

	build(paths)

	// Start the server
	log.Info().Msg("Listening on port 8000")
	http.ListenAndServe(":8000", nil)
}

func getContentFiles(contentPath string) []string {
	// Traverse the content directory and return a list of paths
	paths := []string{}
	walkFn := visit(&paths)
	err := filepath.Walk(contentPath, walkFn)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to walk the content directory")
	}

	return paths
}

func startContentWatcher(contentPath string) {
	// Watch the content directory for changes
	log.Debug().Str("path", contentPath).Msg("Watching content directory for changes")
	w := watcher.New()
	w.SetMaxEvents(1)
	w.FilterOps(watcher.Write, watcher.Create, watcher.Remove, watcher.Rename, watcher.Move)

	r := regexp.MustCompile("^.*\\.md")
	w.AddFilterHook(watcher.RegexFilterHook(r, false))

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

	if err := w.AddRecursive(contentPath); err != nil {
		log.Fatal().Err(err).Msg("Failed to add content directory to watcher")
	}

	// Start the watching process - it'll check for changes every 100ms.
	go w.Start(time.Millisecond * 100)
	w.Wait()
}

func visit(paths *[]string) filepath.WalkFunc {
	return func(path string, f os.FileInfo, err error) error {
		if err != nil {
			log.Error().Err(err).Msg("Failed to access path")
			return nil // continue walking elsewhere
		}
		if f.IsDir() {
			return nil // not a file. ignore.
		}

		if filepath.Ext(path) == ".md" {
			*paths = append(*paths, path)
			log.Debug().Str("path", path).Msg("Found file")
		}
		return nil
	}
}

func build(paths []string) {
	// Build the content
	log.Info().Msg("Building content")

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

	for _, path := range paths {
		buildSingle(path)
	}
}

func mdToHTML(md []byte) []byte {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}

func buildSingle(path string) {
	println(path)
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

func getBuildPath(mdFilePath string) (string, error) {
	if filepath.IsAbs(mdFilePath) {
		relPath, err := filepath.Rel(wd, mdFilePath)
		if err != nil {
			return "", err
		}
		mdFilePath = relPath
	}

	relPath, err := filepath.Rel(contentPath, mdFilePath)
	if err != nil {
		return "", err
	}

	if relPath == "index.md" {
		relPath = ""
	} else {
		relPath = relPath[:len(relPath)-3]
	}

	outputDir := filepath.Join(CONTENT_BUILD_DIR, relPath)
	return outputDir, nil
}

func rebuild(event watcher.Event) {
	log.Debug().Str("path", event.Path).Msg("Rebuilding content")

	// Switch of event type
	switch event.Op {
	case watcher.Write, watcher.Create:
		log.Debug().Msg("File has been written")
		buildSingle(event.Path)
	case watcher.Remove:
		log.Debug().Msg("File has been removed")
		dir, err := getBuildPath(event.Path)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get build directory")
		}
		err = os.RemoveAll(dir)
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

		println(event.OldPath, event.Path, oldBuildPath, newBuildPath)

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

func cleanEmptyDirs() {
	// Clean empty directories in the build directory
	err := filepath.Walk(CONTENT_BUILD_DIR, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			err = os.Remove(path)
			if err != nil {
				// If error ends with "directory not empty" do not log anything
				if !strings.HasSuffix(err.Error(), "directory not empty") {
					log.Warn().Err(err).Msg("Failed to clean empty directory")
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to remove empty directories")
	}
}
