package markdown

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/88250/lute/ast"
	lutehtml "github.com/88250/lute/html"
	"github.com/88250/lute/parse"
)

var contactDestinationStartPattern = regexp.MustCompile(`(?:\]\(\s*<?|\]:\s*<?|<)`)

// ContactReference locates the original (possibly encoded) ID spelling.
// A merge replaces that span only, without reformatting user prose.
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
	// Lute has no source offsets. Prefix candidates in a parsing-only copy,
	// leaving reference labels and the original destination bytes untouched.
	// Candidates retain their raw address spelling. Let Lute decode it in its
	// original context: inline/reference destinations decode entities and escapes,
	// while autolinks do not. Replacing whole destinations before parsing would
	// accidentally turn encoded autolinks into live contact links.
	starts := contactDestinationStartPattern.FindAllStringIndex(content, -1)
	prefix := "bonds-contact-reference-"
	for strings.Contains(content, prefix) {
		prefix += "x"
	}
	configured, _ := configuredEngine()
	original := parse.Parse("bonds-contact-reference-destinations", []byte(content), configured.ParseOptions)
	// Raw-text uniqueness is insufficient: an unrelated address can encode the
	// prefix and impersonate a marked code example after Lute decodes it.
	// Check the parser's actual destination namespace before adding markers.
	// Appending to the prefix keeps it absent from previously visited addresses.
	ast.Walk(original.Root, func(node *ast.Node, entering bool) ast.WalkStatus {
		if entering && node.Type == ast.NodeLinkDest {
			for strings.Contains(node.TokensStr(), prefix) {
				prefix += "x"
			}
		}
		return ast.WalkContinue
	})
	var marked strings.Builder
	offsets := make(map[string]int)
	candidates := []ContactReference{}
	last := 0
	for _, start := range starts {
		// Scan starts separately: consuming a non-contact token such as <3]
		// must not hide the ]( destination in a label like [<3](contact:...).
		destinationStart := start[1]
		raw := content[destinationStart:]
		if end := strings.IndexAny(raw, " \t\n\r\f\v<>()"); end >= 0 {
			raw = raw[:end]
		}
		decoded := string(lutehtml.UnescapeBytes([]byte(raw)))
		contact := contactDestinationPattern.FindStringSubmatch(decoded)
		if contact == nil {
			continue
		}
		// Use Lute's decoder for the source boundary too, including encoded scheme
		// characters. UUID escapes/entities are replaced along with the UUID.
		idStart := 0
		for end := 1; end <= len(raw); end++ {
			if string(lutehtml.UnescapeBytes([]byte(raw[:end]))) == "contact:" {
				idStart = end
				break
			}
		}
		if idStart == 0 {
			continue
		}
		marker := prefix + strconv.Itoa(len(candidates)) + "-"
		offsets[marker+decoded] = len(candidates)
		candidates = append(candidates, ContactReference{contact[1], destinationStart + idStart, destinationStart + len(raw)})
		marked.WriteString(content[last:destinationStart])
		marked.WriteString(marker)
		last = destinationStart
	}
	if len(candidates) == 0 {
		return refs
	}
	marked.WriteString(content[last:])
	tree := parse.Parse("bonds-contact-references", []byte(marked.String()), configured.ParseOptions)
	live := make(map[int]bool)
	ast.Walk(tree.Root, func(node *ast.Node, entering bool) ast.WalkStatus {
		if !entering {
			return ast.WalkContinue
		}
		// The renderer skips image children too: Markdown inside alt text is not
		// an interactive contact link, even if Lute represents it as a child link.
		// Lute renders footnotes with a separate renderer without our contact
		// extension; those destinations are sanitized text, not contact links.
		if node.Type == ast.NodeLinkRefDefBlock || node.Type == ast.NodeImage || node.Type == ast.NodeFootnotesDefBlock {
			return ast.WalkSkipChildren
		}
		if node.Type == ast.NodeLink {
			_, destination := linkParts(node)
			if index, ok := offsets[destination]; ok {
				live[index] = true
			}
			// renderLink emits its label and skips the entire child subtree.
			return ast.WalkSkipChildren
		}
		return ast.WalkContinue
	})
	for i, candidate := range candidates {
		if live[i] {
			refs = append(refs, candidate)
		}
	}
	return refs
}
