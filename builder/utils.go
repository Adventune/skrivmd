package builder

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// Get all markdown files in the content directory.
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

// Custom walk function to visit all markdown files in the content directory.
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

// Get the folder where the content file should be built.
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

// Cleans empty directories in the build directory.
func cleanEmptyDirs() {
	// Walk the build directory and remove empty directories
	filepath.Walk(CONTENT_BUILD_DIR, func(path string, info os.FileInfo, err error) error {
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
}
