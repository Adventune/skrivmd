package builder

import (
	"bytes"

	"github.com/adrg/frontmatter"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

func render(content []byte) []byte {
	var frontMatter struct {
		Title  string   `yaml:"title"`
		Date   string   `yaml:"date"`
		Author string   `yaml:"author"`
		Tags   []string `yaml:"tags"`
	}

	rest, err := frontmatter.Parse(bytes.NewReader(content), &frontMatter)
	if err != nil {
		return nil
	}

	return mdToHTML(rest)
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
