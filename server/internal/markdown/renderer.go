package markdown

import (
	"html"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/88250/lute"
	"github.com/88250/lute/ast"
	"github.com/88250/lute/parse"
	"github.com/microcosm-cc/bluemonday"
)

const (
	FormatPlain    = "plain"
	FormatMarkdown = "markdown"
)

var (
	contactDestinationPattern = regexp.MustCompile(`^contact:([0-9a-fA-F-]{36})$`)
	contactMentionPattern     = regexp.MustCompile(`@\[(?:\\[\\\]]|[^\]\r\n])+\]\(contact:[0-9a-fA-F-]{36}\)`)
	fileDestinationPattern    = regexp.MustCompile(`^bonds-file:([1-9][0-9]*)$`)
	engineOnce                sync.Once
	engine                    *lute.Lute
	policy                    *bluemonday.Policy
)

func NormalizeFormat(format string) string {
	if format == FormatMarkdown {
		return FormatMarkdown
	}
	return FormatPlain
}

func IsValidFormat(format string) bool {
	return format == "" || format == FormatPlain || format == FormatMarkdown
}

func configuredEngine() (*lute.Lute, *bluemonday.Policy) {
	engineOnce.Do(func() {
		engine = lute.New()
		engine.SetGFMTable(true)
		engine.SetGFMStrikethrough(true)
		engine.SetGFMTaskListItem(true)
		engine.SetGFMAutoLink(true)
		engine.SetFootnotes(true)
		engine.SetCodeSyntaxHighlight(false)
		engine.SetHeadingAnchor(false)
		engine.SetSanitize(false) // A separate strict policy is the security boundary.
		engine.Md2HTMLRendererFuncs[ast.NodeLink] = renderLink
		engine.Md2HTMLRendererFuncs[ast.NodeImage] = renderImage
		engine.Md2HTMLRendererFuncs[ast.NodeHTMLBlock] = renderRawHTMLAsText
		engine.Md2HTMLRendererFuncs[ast.NodeInlineHTML] = renderRawHTMLAsText

		policy = bluemonday.UGCPolicy()
		policy.AllowAttrs(
			"data-bonds-contact",
			"data-bonds-file",
			"data-bonds-kind",
			"data-bonds-name",
		).OnElements("span")
		policy.AllowAttrs("class").OnElements("code", "pre", "table", "thead", "tbody", "tr", "th", "td", "ul", "ol", "li")
		policy.AllowAttrs("type", "checked", "disabled").OnElements("input")
		policy.RequireNoFollowOnLinks(true)
		policy.RequireNoReferrerOnLinks(true)
		policy.AddTargetBlankToFullyQualifiedLinks(true)
	})
	return engine, policy
}

func Render(content, format string) string {
	configuredLute, sanitizer := configuredEngine()
	if NormalizeFormat(format) == FormatPlain {
		return sanitizer.Sanitize(renderPlain(content))
	}
	// The leading @ is part of Bonds' mention marker, not the link label.
	// Remove it only for rendering; the stored Markdown remains canonical.
	prepared := contactMentionPattern.ReplaceAllStringFunc(content, func(marker string) string {
		return strings.TrimPrefix(marker, "@")
	})
	return sanitizer.Sanitize(configuredLute.MarkdownStr("bonds", prepared))
}

func renderPlain(content string) string {
	var result strings.Builder
	result.WriteString("<p>")
	last := 0
	for _, location := range contactMentionPattern.FindAllStringIndex(content, -1) {
		result.WriteString(strings.ReplaceAll(html.EscapeString(content[last:location[0]]), "\n", "<br>\n"))
		marker := content[location[0]:location[1]]
		open := strings.Index(marker, "[")
		close := strings.LastIndex(marker, "](contact:")
		idStart := close + len("](contact:")
		idEnd := len(marker) - 1
		if open >= 0 && close > open && idStart < idEnd {
			name := strings.ReplaceAll(marker[open+1:close], `\]`, "]")
			result.WriteString(`<span data-bonds-contact="`)
			result.WriteString(html.EscapeString(marker[idStart:idEnd]))
			result.WriteString(`" data-bonds-name="`)
			result.WriteString(html.EscapeString(name))
			result.WriteString(`">`)
			result.WriteString(html.EscapeString(name))
			result.WriteString(`</span>`)
		} else {
			result.WriteString(html.EscapeString(marker))
		}
		last = location[1]
	}
	result.WriteString(strings.ReplaceAll(html.EscapeString(content[last:]), "\n", "<br>\n"))
	result.WriteString("</p>")
	return result.String()
}

func renderRawHTMLAsText(node *ast.Node, entering bool) (string, ast.WalkStatus) {
	if !entering {
		return "", ast.WalkContinue
	}
	return html.EscapeString(node.TokensStr()), ast.WalkSkipChildren
}

func linkParts(node *ast.Node) (label, destination string) {
	if text := node.ChildByType(ast.NodeLinkText); text != nil {
		label = text.TokensStr()
	}
	if dest := node.ChildByType(ast.NodeLinkDest); dest != nil {
		destination = dest.TokensStr()
	}
	return
}

func renderLink(node *ast.Node, entering bool) (string, ast.WalkStatus) {
	if !entering {
		return "", ast.WalkContinue
	}
	label, destination := linkParts(node)
	if match := contactDestinationPattern.FindStringSubmatch(destination); match != nil {
		return `<span data-bonds-contact="` + html.EscapeString(match[1]) + `" data-bonds-name="` + html.EscapeString(label) + `">` + html.EscapeString(label) + `</span>`, ast.WalkSkipChildren
	}
	if match := fileDestinationPattern.FindStringSubmatch(destination); match != nil {
		return `<span data-bonds-file="` + match[1] + `" data-bonds-kind="file" data-bonds-name="` + html.EscapeString(label) + `">` + html.EscapeString(label) + `</span>`, ast.WalkSkipChildren
	}
	return `<a href="` + html.EscapeString(destination) + `">` + html.EscapeString(label) + `</a>`, ast.WalkSkipChildren
}

func renderImage(node *ast.Node, entering bool) (string, ast.WalkStatus) {
	if !entering {
		return "", ast.WalkContinue
	}
	label, destination := linkParts(node)
	if match := fileDestinationPattern.FindStringSubmatch(destination); match != nil {
		return `<span data-bonds-file="` + match[1] + `" data-bonds-kind="image" data-bonds-name="` + html.EscapeString(label) + `"></span>`, ast.WalkSkipChildren
	}
	return `<img src="` + html.EscapeString(destination) + `" alt="` + html.EscapeString(label) + `">`, ast.WalkSkipChildren
}

func ExtractFileIDs(content string) []uint {
	configuredLute, _ := configuredEngine()
	tree := parse.Parse("bonds-file-references", []byte(content), configuredLute.ParseOptions)
	seen := make(map[uint]struct{})
	var result []uint
	ast.Walk(tree.Root, func(node *ast.Node, entering bool) ast.WalkStatus {
		if !entering || (node.Type != ast.NodeLink && node.Type != ast.NodeImage) {
			return ast.WalkContinue
		}
		_, destination := linkParts(node)
		match := fileDestinationPattern.FindStringSubmatch(destination)
		if match == nil {
			return ast.WalkContinue
		}
		parsed, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil || parsed == 0 || uint64(uint(parsed)) != parsed {
			return ast.WalkContinue
		}
		id := uint(parsed)
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			result = append(result, id)
		}
		return ast.WalkContinue
	})
	return result
}
