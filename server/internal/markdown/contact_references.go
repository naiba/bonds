package markdown

import (
	"strconv"
	"strings"

	"github.com/88250/lute/ast"
	"github.com/88250/lute/parse"
)

// ContactReference locates only the ID, so a merge never reformats user prose.
type ContactReference struct {
	ID         string
	Start, End int
}

// ContactReferences follows the renderer's format semantics. A regexp alone
// treats code and escaped Markdown examples as live links, rejecting innocent
// text on save and rewriting it during merges.
func ContactReferences(content, format string) []ContactReference {
	var refs []ContactReference
	if NormalizeFormat(format) == FormatPlain {
		for _, match := range contactMentionPattern.FindAllStringSubmatchIndex(content, -1) {
			refs = append(refs, ContactReference{content[match[2]:match[3]], match[2], match[3]})
		}
		return refs
	}
	matches := contactDestinationInTextPattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return refs
	}
	// Lute does not expose source offsets. Distinguish each candidate destination
	// in a parsing-only copy; identical UUIDs inside code must remain untouched.
	// Only destination spellings are marked, leaving reference labels unchanged.
	// The unique prefix cannot collide with any destination in the original text.
	prefix := "bonds-contact-reference-"
	for strings.Contains(content, prefix) {
		prefix += "x"
	}
	var marked strings.Builder
	offsets := make(map[string]int, len(matches))
	last := 0
	for i, match := range matches {
		destination := "contact:" + prefix + strconv.Itoa(i)
		offsets[destination] = i
		marked.WriteString(content[last:match[2]])
		marked.WriteString(destination)
		last = match[3]
	}
	marked.WriteString(content[last:])
	configured, _ := configuredEngine()
	tree := parse.Parse("bonds-contact-references", []byte(marked.String()), configured.ParseOptions)
	live := make(map[int]bool)
	ast.Walk(tree.Root, func(node *ast.Node, entering bool) ast.WalkStatus {
		if !entering {
			return ast.WalkContinue
		}
		// The renderer skips image children too: Markdown inside alt text is not
		// an interactive contact link, even if Lute represents it as a child link.
		if node.Type == ast.NodeLinkRefDefBlock || node.Type == ast.NodeImage {
			return ast.WalkSkipChildren
		}
		if node.Type == ast.NodeLink {
			_, destination := linkParts(node)
			if index, ok := offsets[destination]; ok {
				live[index] = true
			}
		}
		return ast.WalkContinue
	})
	for i, match := range matches {
		if live[i] {
			refs = append(refs, ContactReference{content[match[4]:match[5]], match[4], match[5]})
		}
	}
	return refs
}
