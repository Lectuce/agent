package memory

import (
	"agent/config"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adrg/frontmatter"
)

var MEMORY_TYPES = []string{
	"user",
	"feedback",
	"project",
	"reference",
}

type MemoryFile struct {
	Filename    string
	Name        string
	Description string
	Type        string
	Body        string
}

func parseFrontmatter(raw string) (map[string]string, string, error) {
	var meta = map[string]string{}
	body, err := frontmatter.Parse(
		bytes.NewReader([]byte(raw)),
		&meta,
	)
	if err != nil {
		return meta, "", err
	}

	return meta, string(body), nil
}

func writeMemoryFile(name string, memoryType string, description string, body string) (string, error) {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "/", "-")

	filename := slug + ".md"
	path := filepath.Join(config.MEMORY_DIR, filename)

	content := fmt.Sprintf(
		"---\n"+
			"name: %v\n"+
			"description: %v\n"+
			"type: %v\n"+
			"---\n\n"+
			"%v\n",
		name,
		description,
		memoryType,
		body,
	)
	err := os.MkdirAll(config.MEMORY_DIR, 0755)
	if err != nil {
		return "", err
	}
	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	_, err = rebuildIndex()
	if err != nil {
		return "", err
	}

	return path, nil
}

func rebuildIndex() (string, error) {
	data, err := os.ReadFile(config.MEMORY_INDEX)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	text := strings.TrimSpace(string(data))
	return text, nil

}

func readMemoryFile(filename string) (string, error) {
	path := filepath.Join(config.MEMORY_DIR, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return string(data), nil
}

func listMemoryFiles() ([]MemoryFile, error) {
	pattern := filepath.Join(config.MEMORY_DIR, "*.md")

	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	sort.Strings(files)

	result := make([]MemoryFile, 0)

	for _, path := range files {
		filename := filepath.Base(path)

		if filename == "MEMORY.md" {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		raw := string(data)

		meta, body, err := parseFrontmatter(raw)
		if err != nil {
			return nil, err
		}

		name := meta["name"]
		if name == "" {
			name = strings.TrimSuffix(filename, filepath.Ext(filename))
		}

		description := meta["description"]

		memType := meta["type"]
		if memType == "" {
			memType = "user"
		}

		result = append(result, MemoryFile{
			Filename:    filename,
			Name:        name,
			Description: description,
			Type:        memType,
			Body:        body,
		})
	}

	return result, nil
}
