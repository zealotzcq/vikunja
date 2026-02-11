# 技能系统更新 - 仅从./skills目录加载

## 更新说明

根据用户要求，Vikunja的AI智能体现在**只从项目的`./skills`目录加载技能**，不再加载其他应用（如OpenCode、Claude）使用的技能目录。

## 主要更改

### 代码修改

**文件**: `pkg/modules/ai/skills.go`

**修改**: `skillSearchPaths()`函数

```go
// 修改前
func skillSearchPaths() []string {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        homeDir = "~"
    }

    return []string{
        ".opencode/skills",
        "skills",
        filepath.Join(homeDir, ".config", "opencode", "skills"),
        ".claude/skills",
        filepath.Join(homeDir, ".claude", "skills"),
        ".agents/skills",
        filepath.Join(homeDir, ".agents", "skills"),
    }
}

// 修改后
func skillSearchPaths() []string {
    return []string{
        "skills",
    }
}
```

### 文档更新

1. **`docs/ai-skills.md`**:
   - 更新技能目录说明，只列出`./skills/`目录
   - 添加重要说明，解释其他应用的技能不会被加载
   - 移除对其他应用技能目录的引用

2. **`IMPLEMENTATION.md`**:
   - 更新概述部分，明确说明只从`./skills`目录加载
   - 更新已加载技能列表，移除其他应用技能
   - 更新技能发现和加载部分，添加单一目录说明

3. **`skills/ancient-poet/SKILL.md`**:
   - 修正frontmatter格式

## 验证

### 已加载技能

系统现在只从`./skills`目录加载技能：

```
2026/02/11 21:10:55 [AI Skills] Loaded skill: ancient-poet - 此技能用于模仿古代诗人风格创作古诗...
2026/02/11 21:10:55 [AI Skills] Loaded 1 skills
```

只有`ancient-poet`技能被加载，其他应用（OpenCode、Claude）的技能不再被加载。

### 测试

所有测试通过：

```bash
$ go test ./pkg/modules/ai/...
ok  	code.vikunja.io/api/pkg/modules/ai	0.488s
```

### 构建

项目成功构建：

```bash
$ mage build
C:\Program Files\Go\bin\go.exe build -v -tags  -ldflags -s -w \
  -X "code.vikunja.io/api/pkg/version.Version=d1d1126755" \
  -X "main.Tags=" -o vikunja.exe
```

## 技能放置指南

Vikunja的技能应该放置在项目的`./skills/`目录下：

```
vikunja/
├── skills/
│   ├── ancient-poet/
│   │   ├── SKILL.md
│   │   └── references/
│   │       ├── 李白.md
│   │       ├── 杜甫.md
│   │       └── 苏轼.md
│   └── my-custom-skill/
│       └── SKILL.md
└── ...
```

## 与其他应用的隔离

现在Vikunja与其他应用的技能完全隔离：

| 应用 | 技能目录 | 是否加载 |
|------|----------|---------|
| Vikunja | `./skills/` | ✓ 是 |
| OpenCode | `.opencode/skills/`, `~/.config/opencode/skills/` | ✗ 否 |
| Claude | `.claude/skills/`, `~/.claude/skills/` | ✗ 否 |
| 其他 | `.agents/skills/`, `~/.agents/skills/` | ✗ 否 |

这确保了：
1. Vikunja只使用自己的技能
2. 不会与OpenCode、Claude等应用产生冲突
3. 每个应用管理自己的技能集合

## 总结

✅ 成功修改技能系统，使其只从`./skills`目录加载技能
✅ 移除了对其他应用技能目录的扫描
✅ 更新了相关文档
✅ 所有测试通过
✅ 项目成功构建

现在Vikunja的AI智能体只会使用项目`./skills/`目录下的技能，不会加载其他应用的技能。
