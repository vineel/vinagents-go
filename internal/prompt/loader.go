package prompt

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"text/template"
)

// Loader handles loading and executing prompt templates
type Loader struct {
	promptsDir string
	cache      map[string]*template.Template
	mu         sync.RWMutex
}

// NewLoader creates a new prompt loader
func NewLoader(promptsDir string) *Loader {
	return &Loader{
		promptsDir: promptsDir,
		cache:      make(map[string]*template.Template),
	}
}

// Load loads a template by name (without the .prompt.tpl extension)
func (l *Loader) Load(name string) (*template.Template, error) {
	l.mu.RLock()
	if tmpl, ok := l.cache[name]; ok {
		l.mu.RUnlock()
		return tmpl, nil
	}
	l.mu.RUnlock()

	filename := name + ".prompt.tpl"
	path := filepath.Join(l.promptsDir, filename)

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt template %s: %w", name, err)
	}

	tmpl, err := template.New(name).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt template %s: %w", name, err)
	}

	l.mu.Lock()
	l.cache[name] = tmpl
	l.mu.Unlock()

	return tmpl, nil
}

// Execute loads a template and executes it with the given data
func (l *Loader) Execute(name string, data interface{}) (string, error) {
	tmpl, err := l.Load(name)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute prompt template %s: %w", name, err)
	}

	return buf.String(), nil
}

// ClearCache clears the template cache (useful for development/testing)
func (l *Loader) ClearCache() {
	l.mu.Lock()
	l.cache = make(map[string]*template.Template)
	l.mu.Unlock()
}

// ClauserWriteData is the data structure for the clause write prompt
type ClauserWriteData struct {
	AgreementA            string
	AgreementB            string
	ClauseA               string
	ClauseB               string
	RepresentedParty      string
	DraftingApproach      string
	Playbook              string
	CounterpartyRationale string
	BusinessContext       string
	Favorites             []FavoriteItem
	Instructions          string
}

// ClauserLensData is the data structure for the lens analysis prompt
type ClauserLensData struct {
	AgreementA            string
	AgreementB            string
	ClauseA               string
	ClauseB               string
	RepresentedParty      string
	DraftingApproach      string
	Playbook              string
	CounterpartyRationale string
	BusinessContext       string
	Favorites             []FavoriteItem
	Lenses                []string
}

// FavoriteItem represents a favorited output item
type FavoriteItem struct {
	Title string
	Body  string
}
