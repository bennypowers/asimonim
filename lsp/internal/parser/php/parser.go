package php

import (
	"fmt"
	"sync"

	"bennypowers.dev/asimonim/lsp/internal/parser/css"
	"bennypowers.dev/asimonim/lsp/internal/parser/html"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_php "github.com/tree-sitter/tree-sitter-php/bindings/go"
)

// Parser handles parsing PHP files to extract CSS regions.
// It uses a two-stage pipeline:
//  1. tree-sitter-php extracts HTML text nodes from the PHP source
//  2. tree-sitter-html parses the reconstructed HTML for style elements
//
// PHP blocks are replaced with whitespace to preserve line/column positions.
type Parser struct {
	phpParser *sitter.Parser
	textQuery *sitter.Query
}

var phpLang = sitter.NewLanguage(tree_sitter_php.LanguagePHP())

var parserPool = sync.Pool{
	New: func() any {
		parser := sitter.NewParser()
		if err := parser.SetLanguage(phpLang); err != nil {
			panic(fmt.Sprintf("failed to set PHP language: %v", err))
		}

		textQuery, qerr := sitter.NewQuery(phpLang, `(text) @html`)
		if qerr != nil {
			panic(fmt.Sprintf("failed to compile text query: %v", qerr))
		}

		return &Parser{
			phpParser: parser,
			textQuery: textQuery,
		}
	},
}

// AcquireParser gets a parser from the pool
func AcquireParser() *Parser {
	v := parserPool.Get()
	if v == nil {
		return nil
	}
	p := v.(*Parser)
	p.phpParser.Reset()
	return p
}

// ReleaseParser returns a parser to the pool
func ReleaseParser(p *Parser) {
	if p != nil {
		parserPool.Put(p)
	}
}

// Close closes the parser and releases its resources
func (p *Parser) Close() {
	if p.phpParser != nil {
		p.phpParser.Close()
	}
	if p.textQuery != nil {
		p.textQuery.Close()
	}
}

// ClosePool drains the parser pool and closes all cached parsers.
func ClosePool() {
	oldNew := parserPool.New
	parserPool.New = nil
	defer func() { parserPool.New = oldNew }()

	for {
		v := parserPool.Get()
		if v == nil {
			break
		}
		if p, ok := v.(*Parser); ok {
			p.Close()
		}
	}
}

// extractHTML uses tree-sitter-php to find HTML text nodes, then replaces
// PHP blocks with whitespace to produce valid HTML with preserved positions.
func (p *Parser) extractHTML(source []byte) []byte {
	tree := p.phpParser.Parse(source, nil)
	if tree == nil {
		return nil
	}
	defer tree.Close()

	// Start with a buffer where PHP blocks are whitespace
	buf := make([]byte, len(source))
	for i, b := range source {
		if b == '\n' {
			buf[i] = '\n'
		} else {
			buf[i] = ' '
		}
	}

	// Copy HTML text nodes back into position
	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	matches := cursor.Matches(p.textQuery, tree.RootNode(), source)
	for m := matches.Next(); m != nil; m = matches.Next() {
		for _, c := range m.Captures {
			n := c.Node
			copy(buf[n.StartByte():n.EndByte()], source[n.StartByte():n.EndByte()])
		}
	}

	return buf
}

// ParseCSSRegions extracts CSS regions from a PHP file by first extracting
// the HTML portions, then delegating to the HTML parser.
func (p *Parser) ParseCSSRegions(source string) []html.CSSRegion {
	htmlBytes := p.extractHTML([]byte(source))
	if htmlBytes == nil {
		return nil
	}

	hp := html.AcquireParser()
	defer html.ReleaseParser(hp)

	return hp.ParseCSSRegions(string(htmlBytes))
}

// ParseCSS extracts CSS from a PHP file and parses it, mapping positions
// back to the original PHP document coordinates.
func (p *Parser) ParseCSS(source string) (*css.ParseResult, error) {
	htmlBytes := p.extractHTML([]byte(source))
	if htmlBytes == nil {
		return &css.ParseResult{
			Variables: []*css.Variable{},
			VarCalls:  []*css.VarCall{},
		}, nil
	}

	hp := html.AcquireParser()
	defer html.ReleaseParser(hp)

	return hp.ParseCSS(string(htmlBytes))
}
