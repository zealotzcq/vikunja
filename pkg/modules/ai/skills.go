package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"go.yaml.in/yaml/v2"
)

// SkillInfo represents basic skill information from SKILL.md frontmatter
type SkillInfo struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Location    string
}

// SkillMetadata represents the full skill metadata from SKILL.md frontmatter
type SkillMetadata struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
}

// SkillContent represents a skill with full metadata and content
type SkillContent struct {
	Metadata SkillMetadata
	Content  string
	Dir      string
}

// SkillManager manages available skills
type SkillManager struct {
	skills map[string]*SkillInfo
	mu     sync.RWMutex
}

var (
	skillManager *SkillManager
	skillOnce    sync.Once
)

// skillSearchPaths returns the directories to search for skills
func skillSearchPaths() []string {
	return []string{
		"skills",
	}
}

// discoverSkills scans configured directories for SKILL.md files
func discoverSkills() (map[string]*SkillInfo, error) {
	skills := make(map[string]*SkillInfo)
	searchPaths := skillSearchPaths()

	for _, basePath := range searchPaths {
		if _, err := os.Stat(basePath); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(basePath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			skillName := entry.Name()
			skillFile := filepath.Join(basePath, skillName, "SKILL.md")

			if _, err := os.Stat(skillFile); os.IsNotExist(err) {
				continue
			}

			content, err := os.ReadFile(skillFile)
			if err != nil {
				continue
			}

			metadata, _, err := parseSkillMetadata(content)
			if err != nil {
				continue
			}

			if metadata.Name == "" || metadata.Description == "" {
				continue
			}

			if _, exists := skills[metadata.Name]; exists {
				continue
			}

        skills[metadata.Name] = &SkillInfo{
            Name:        metadata.Name,
            Description: metadata.Description,
            Location:    skillFile,
        }
    }
}

return skills, nil
}

// preprocessFrontmatter preprocesses YAML frontmatter to handle values containing colons
func preprocessFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}

	endIndex := strings.Index(content[4:], "---")
	if endIndex == -1 {
		return content
	}

	frontmatter := content[4 : endIndex+4]
	contentAfter := content[endIndex+8:]
	lines := strings.Split(frontmatter, "\n")
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			result = append(result, line)
			continue
		}

		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			result = append(result, line)
			continue
		}

		re := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\s*:\s*(.*)$`)
		matches := re.FindStringSubmatch(line)
		if matches == nil {
			result = append(result, line)
			continue
		}

		key := matches[1]
		value := strings.TrimSpace(matches[2])

		if value == "" || value == ">" || value == "|" || strings.HasPrefix(value, `"`) || strings.HasPrefix(value, `'`) {
			result = append(result, line)
			continue
		}

		if strings.Contains(value, ":") {
			result = append(result, fmt.Sprintf("%s: |", key))
			result = append(result, fmt.Sprintf("  %s", value))
			continue
		}

		result = append(result, line)
	}

	processed := strings.Join(result, "\n")
	return "---\n" + processed + "\n---\n" + contentAfter
}

// parseSkillMetadata extracts YAML frontmatter and content from SKILL.md
func parseSkillMetadata(content []byte) (*SkillMetadata, string, error) {
	contentStr := string(content)

	if !strings.HasPrefix(contentStr, "---") {
		return nil, contentStr, fmt.Errorf("missing frontmatter delimiter")
	}

	endIndex := strings.Index(contentStr[4:], "---")
	if endIndex == -1 {
		return nil, contentStr, fmt.Errorf("missing frontmatter end delimiter")
	}

	skillContent := contentStr[endIndex+8:]

	preprocessed := preprocessFrontmatter(contentStr)
	preprocessedEndIndex := strings.Index(preprocessed[4:], "---")
	if preprocessedEndIndex == -1 {
		return nil, skillContent, fmt.Errorf("missing preprocessed frontmatter end delimiter")
	}
	preprocessedFrontmatter := preprocessed[4 : preprocessedEndIndex+4]

	var metadata SkillMetadata
	if err := yaml.Unmarshal([]byte(preprocessedFrontmatter), &metadata); err != nil {
		return nil, skillContent, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	skillContent = preprocessed[preprocessedEndIndex+8:]

	return &metadata, skillContent, nil
}

// GetSkillManager returns the singleton skill manager
func GetSkillManager() *SkillManager {
	skillOnce.Do(func() {
		skillManager = &SkillManager{
			skills: make(map[string]*SkillInfo),
		}
		skillManager.initialize()
	})
	return skillManager
}

// initialize loads skills from configured directories
func (sm *SkillManager) initialize() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skills, err := discoverSkills()
	if err != nil {

		return
	}

	sm.skills = skills

}

// Reload reloads skills from configured directories
func (sm *SkillManager) Reload() {
	sm.initialize()
}

// GetSkill returns a skill by name
func (sm *SkillManager) GetSkill(name string) (*SkillInfo, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	skill, exists := sm.skills[name]
	return skill, exists
}

// LoadSkillContent loads the full content of a skill
func (sm *SkillManager) LoadSkillContent(name string) (*SkillContent, error) {
	sm.mu.RLock()
	skillInfo, exists := sm.skills[name]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("skill '%s' not found", name)
	}

	content, err := os.ReadFile(skillInfo.Location)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	metadata, skillContent, err := parseSkillMetadata(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse skill: %w", err)
	}

	return &SkillContent{
		Metadata: *metadata,
		Content:  skillContent,
		Dir:      filepath.Dir(skillInfo.Location),
	}, nil
}

// GetAllSkills returns all loaded skills
func (sm *SkillManager) GetAllSkills() map[string]*SkillInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]*SkillInfo, len(sm.skills))
	for k, v := range sm.skills {
		result[k] = v
	}
	return result
}

// GetEnabledSkills returns skills that are enabled in the configuration
func (sm *SkillManager) GetEnabledSkills() map[string]*SkillInfo {
	cfg := GetConfig()
	if len(cfg.EnabledSkills) == 0 {
		return sm.GetAllSkills()
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	enabled := make(map[string]*SkillInfo)
	for _, name := range cfg.EnabledSkills {
		if skill, exists := sm.skills[name]; exists {
			enabled[name] = skill
		}
	}
	return enabled
}

// FormatSkillsForTool formats available skills for the skill tool description
func (sm *SkillManager) FormatSkillsForTool() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if len(sm.skills) == 0 {
		return "<available_skills>\n  No skills available\n</available_skills>"
	}

	var sb strings.Builder
	sb.WriteString("<available_skills>\n")
	for _, skill := range sm.skills {
		sb.WriteString(fmt.Sprintf("  <skill>\n    <name>%s</name>\n    <description>%s</description>\n  </skill>\n",
			skill.Name, skill.Description))
	}
	sb.WriteString("</available_skills>")
	return sb.String()
}
