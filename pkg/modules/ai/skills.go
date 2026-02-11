package ai

import (
	"fmt"
	"sync"
)

// Skill represents a capability that the agent can use
type Skill interface {
	// Name returns the skill name
	Name() string
	// Description returns a description of what the skill does
	Description() string
	// Execute executes the skill with the given input
	Execute(ctx *AgentContext, input string) (string, error)
	// CanHandle returns true if this skill can handle the given input
	CanHandle(input string) bool
}

// SkillManager manages available skills
type SkillManager struct {
	skills map[string]Skill
	mu     sync.RWMutex
}

var (
	skillManager *SkillManager
	skillOnce    sync.Once
)

// GetSkillManager returns the singleton skill manager
func GetSkillManager() *SkillManager {
	skillOnce.Do(func() {
		skillManager = &SkillManager{
			skills: make(map[string]Skill),
		}
	})
	return skillManager
}

// RegisterSkill registers a new skill
func (sm *SkillManager) RegisterSkill(skill Skill) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if skill == nil {
		return fmt.Errorf("skill cannot be nil")
	}

	name := skill.Name()
	if name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	sm.skills[name] = skill
	return nil
}

// UnregisterSkill removes a skill
func (sm *SkillManager) UnregisterSkill(name string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.skills, name)
}

// GetSkill returns a skill by name
func (sm *SkillManager) GetSkill(name string) (Skill, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	skill, exists := sm.skills[name]
	return skill, exists
}

// GetAllSkills returns all registered skills
func (sm *SkillManager) GetAllSkills() []Skill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	skills := make([]Skill, 0, len(sm.skills))
	for _, skill := range sm.skills {
		skills = append(skills, skill)
	}
	return skills
}

// FindMatchingSkills returns skills that can handle the given input
func (sm *SkillManager) FindMatchingSkills(input string) []Skill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var matching []Skill
	for _, skill := range sm.skills {
		if skill.CanHandle(input) {
			matching = append(matching, skill)
		}
	}
	return matching
}

// GetEnabledSkills returns skills that are enabled in the configuration
func (sm *SkillManager) GetEnabledSkills() []Skill {
	cfg := GetConfig()
	if len(cfg.EnabledSkills) == 0 {
		return sm.GetAllSkills()
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	enabled := make([]Skill, 0)
	for _, name := range cfg.EnabledSkills {
		if skill, exists := sm.skills[name]; exists {
			enabled = append(enabled, skill)
		}
	}
	return enabled
}
