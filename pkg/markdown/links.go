package markdown

import (
	"fmt"
	"regexp"
	"strings"
)

var linkPattern = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

type LinkRef struct {
	Target  string
	Display string
}

// ExtractLinks extracts all [[bidirectional links]] from markdown content
func ExtractLinks(content string) []LinkRef {
	matches := linkPattern.FindAllStringSubmatch(content, -1)
	var links []LinkRef
	seen := make(map[string]bool)

	for _, match := range matches {
		target := strings.TrimSpace(match[1])
		display := target
		if len(match) > 2 && match[2] != "" {
			display = strings.TrimSpace(match[2])
		}
		if !seen[target] {
			links = append(links, LinkRef{Target: target, Display: display})
			seen[target] = true
		}
	}
	return links
}

// ReplaceLinks converts [[links]] to HTML <a> tags
func ReplaceLinks(content string, noteMap map[string]string) string {
	return linkPattern.ReplaceAllStringFunc(content, func(match string) string {
		submatch := linkPattern.FindStringSubmatch(match)
		target := strings.TrimSpace(submatch[1])
		display := target
		if len(submatch) > 2 && submatch[2] != "" {
			display = strings.TrimSpace(submatch[2])
		}
		if id, ok := noteMap[target]; ok {
			return fmt.Sprintf(`<a class="internal-link" data-note-id="%s">%s</a>`, id, display)
		}
		return fmt.Sprintf(`<a class="internal-link unresolved" data-target="%s">%s</a>`, target, display)
	})
}

// CountLinks returns the number of [[links]] in content
func CountLinks(content string) int {
	return len(linkPattern.FindAllString(content, -1))
}
