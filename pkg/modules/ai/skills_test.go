package ai

import (
	"strings"
	"testing"
)

func TestSkillManager_DiscoverSkills(t *testing.T) {
	sm := GetSkillManager()
	skills := sm.GetAllSkills()

	if len(skills) == 0 {
		t.Logf("No skills found - this is expected if skills/ directory doesn't exist or is empty")
		return
	}

	t.Logf("Loaded %d skills:", len(skills))
	for name, info := range skills {
		t.Logf("  - %s: %s (location: %s)", name, info.Description, info.Location)
	}
}

func TestSkillManager_LoadSkillContent(t *testing.T) {
	sm := GetSkillManager()

	skills := sm.GetAllSkills()
	if len(skills) == 0 {
		t.Skip("No skills available for testing")
	}

	for name := range skills {
		content, err := sm.LoadSkillContent(name)
		if err != nil {
			t.Errorf("Failed to load skill '%s': %v", name, err)
			continue
		}

		if content.Metadata.Name != name {
			t.Errorf("Skill name mismatch: expected %s, got %s", name, content.Metadata.Name)
		}

		if content.Content == "" {
			t.Errorf("Skill '%s' has empty content", name)
		}

		t.Logf("Successfully loaded skill '%s' with %d characters of content", name, len(content.Content))
	}
}

func TestSkillManager_FormatSkillsForTool(t *testing.T) {
	sm := GetSkillManager()
	formatted := sm.FormatSkillsForTool()

	if formatted == "" {
		t.Error("Formatted skills output is empty")
	}

	t.Logf("Formatted skills:\n%s", formatted)
}

func TestParseSkillMetadata(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name: "valid frontmatter",
			content: `---
name: test-skill
description: A test skill
license: MIT
compatibility: opencode
---
This is the skill content.`,
			expectError: false,
		},
		{
			name:        "missing delimiter",
			content:     `name: test-skill\ndescription: A test skill`,
			expectError: true,
		},
		{
			name: "description with colon",
			content: `---
name: test-skill
description: https://example.com
---
This is the skill content.`,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metadata, content, err := parseSkillMetadata([]byte(tc.content))

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if metadata.Name != "test-skill" {
				t.Errorf("Expected name 'test-skill', got '%s'", metadata.Name)
			}

			if tc.name == "description with colon" {
				expectedDesc := "https://example.com"
				actualDesc := strings.TrimSpace(metadata.Description)
				if actualDesc != expectedDesc {
					t.Errorf("Expected description '%s', got '%s'", expectedDesc, metadata.Description)
				}
			} else if metadata.Description != "A test skill" {
				t.Errorf("Expected description 'A test skill', got '%s'", metadata.Description)
			}

			if content == "" {
				t.Error("Content is empty")
			}
		})
	}
}
