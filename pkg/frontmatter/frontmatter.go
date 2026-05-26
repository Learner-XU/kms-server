package frontmatter

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Frontmatter struct {
	ID      string   `yaml:"id"`
	Title   string   `yaml:"title"`
	Created string   `yaml:"created"`
	Updated string   `yaml:"updated"`
	Tags    []string `yaml:"tags"`
	Type    string   `yaml:"type"`
	Status  string   `yaml:"status"`
	Source  string   `yaml:"source,omitempty"`
	Links   []string `yaml:"links,omitempty"`
	Summary string   `yaml:"summary,omitempty"`
}

const delimiter = "---"

// Parse splits markdown content into frontmatter and body
func Parse(content string) (*Frontmatter, string, error) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, delimiter) {
		return nil, content, nil
	}

	endIdx := strings.Index(content[len(delimiter):], "\n"+delimiter)
	if endIdx < 0 {
		return nil, content, nil
	}

	yamlStr := content[len(delimiter) : len(delimiter)+endIdx]
	body := strings.TrimSpace(content[len(delimiter)+endIdx+len(delimiter):])

	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(yamlStr), &fm); err != nil {
		return nil, content, fmt.Errorf("parse frontmatter: %w", err)
	}
	return &fm, body, nil
}

// Marshal converts frontmatter + body back to markdown
func Marshal(fm *Frontmatter, body string) (string, error) {
	data, err := yaml.Marshal(fm)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("---\n%s---\n\n%s", string(data), body), nil
}

// DefaultFrontmatter creates a new frontmatter with defaults
func DefaultFrontmatter(id, title string) *Frontmatter {
	now := time.Now().Format(time.RFC3339)
	return &Frontmatter{
		ID:      id,
		Title:   title,
		Created: now,
		Updated: now,
		Tags:    []string{},
		Type:    "note",
		Status:  "seed",
	}
}
