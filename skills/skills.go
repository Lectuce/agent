package skills

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

type skill struct {
	Name        string
	Description string
	Content     string
}
type SkillMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

var SkillRegistry = map[string]skill{}

func ScanSkill() error {
	entries, err := os.ReadDir(config.SKILLSDIR)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(config.SKILLSDIR, entry.Name())
		manifest := filepath.Join(dir, "SKILL.md")
		data, err := os.ReadFile(manifest)
		if err != nil {
			if entry.IsDir() {
				continue
			}
			return err
		}
		raw := string(data)
		meta, body, err := parseFrontmatter(raw)

		if err != nil {
			return err
		}

		name := meta.Name
		if name == "" {
			name = entry.Name()
		}

		desc := meta.Description
		if desc == "" {
			lines := strings.Split(body, "\n")
			if len(lines) > 0 {
				desc = strings.TrimSpace(
					strings.TrimPrefix(lines[0], "#"),
				)
			}
		}
		SkillRegistry[name] = skill{
			Name:        name,
			Description: desc,
			Content:     body,
		}

	}
	return nil
}

func ListSkill() string {
	lines := make([]string, 0)
	for _, skill := range SkillRegistry {
		lines = append(lines,
			fmt.Sprintf("- **%v**: %v", skill.Name, skill.Description),
		)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")

}

func parseFrontmatter(raw string) (SkillMeta, string, error) {
	var meta SkillMeta

	body, err := frontmatter.Parse(
		bytes.NewReader([]byte(raw)),
		&meta,
	)
	if err != nil {
		return meta, "", err
	}

	return meta, string(body), nil
}

// func buildSystem() string {
// 	catalog := listSkill()

// 	return fmt.Sprintf("You are a coding agent at %v.\n Skills available:\n%v\n Use load_skill to get full details when needed.\n", config.WORKDIR, catalog)
// }

func LoadSkill(name string) string {
	skill, ok := SkillRegistry[name]
	if !ok {
		return fmt.Sprintf("Skill not found: %v\n", skill)
	}
	return skill.Content

}

func BuildSystem() string {
	catalog := ListSkill()

	return fmt.Sprintf("You are a coding agent at %v.\n Skills available:\n%v\n Use load_skill to get full details when needed.\n", config.WORKDIR, catalog)
}
