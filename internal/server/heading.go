package server

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// unicodeIDs generates heading IDs that preserve unicode characters (e.g. Chinese).
type unicodeIDs struct {
	seen map[string]int
}

func (u *unicodeIDs) Generate(value []byte, kind ast.NodeKind) []byte {
	id := make([]byte, 0, len(value))
	for _, b := range value {
		switch {
		case b >= 'a' && b <= 'z', b >= '0' && b <= '9', b == '-', b == '_':
			id = append(id, b)
		case b >= 'A' && b <= 'Z':
			id = append(id, b+32) // lowercase
		case b == ' ', b == '\t':
			id = append(id, '-')
		default:
			id = append(id, b) // keep unicode bytes as-is
		}
	}

	if len(id) == 0 {
		id = []byte("heading")
	}

	key := string(id)
	if cnt, ok := u.seen[key]; ok {
		u.seen[key] = cnt + 1
		id = append(id, []byte(fmt.Sprintf("-%d", cnt))...)
	} else {
		u.seen[key] = 1
	}

	return id
}

func (u *unicodeIDs) Put(value []byte) {
	u.seen[string(value)] = 1
}

// headingIDTransformer replaces heading IDs with unicode-preserving ones.
type headingIDTransformer struct{}

func (t *headingIDTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	ids := &unicodeIDs{seen: map[string]int{}}
	source := reader.Source()

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		// recursively collect all text content from the heading
		headingText := collectText(heading, source)
		id := ids.Generate(headingText, heading.Kind())
		heading.SetAttributeString("id", id)

		return ast.WalkContinue, nil
	})
}

// collectText recursively extracts all text content from an AST node.
func collectText(node ast.Node, source []byte) []byte {
	var buf []byte
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Text:
			buf = append(buf, v.Value(source)...)
		case *ast.CodeSpan:
			// extract text from code span children
			for child := v.FirstChild(); child != nil; child = child.NextSibling() {
				if t, ok := child.(*ast.Text); ok {
					segment := t.Segment
					buf = append(buf, segment.Value(source)...)
				}
			}
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return buf
}
