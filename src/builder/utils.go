package builder

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/rs/zerolog/log"
)

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

// Converts markdown elements to raw unstyled HTML.
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

// Cleans empty directories in the build directory
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
