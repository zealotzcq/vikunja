# AI Skills System

Vikunja的AI智能体支持从项目`./skills`目录自动加载技能。

## 技能目录

Vikunja只从项目根目录下的`./skills`目录加载技能：

- `./skills/<name>/SKILL.md`

**注意**: 其他应用（如OpenCode、Claude等）使用的技能目录（如`.opencode/skills/`、`~/.config/opencode/skills/`等）不会被Vikunja加载，因为它们是其他应用的技能。Vikunja的技能应该放置在`./skills/`目录下。

## 技能文件格式

每个技能必须是一个目录，包含`SKILL.md`文件：

```
skills/
└── ancient-poet/
    ├── SKILL.md
    └── references/
        ├── 李白.md
        ├── 杜甫.md
        └── 苏轼.md
```

### SKILL.md 格式

`SKILL.md`必须以YAML frontmatter开头：

```markdown
---
name: skill-name
description: Brief description of what this skill does (1-1024 chars)
license: MIT
compatibility: opencode
---

# Skill Title

Detailed instructions, workflows, and references...
```

### Frontmatter 字段

| 字段 | 必需 | 说明 |
|------|------|------|
| `name` | 是 | 技能名称，必须与目录名一致 |
| `description` | 是 | 技能描述，1-1024字符 |
| `license` | 否 | 许可证 |
| `compatibility` | 否 | 兼容性标识 |
| `metadata` | 否 | 自定义元数据映射 |

## 使用技能

### 智能体自动加载

智能体初始化时会自动扫描所有配置的目录并加载技能：

```go
agent, err := ai.GetAgent()
// 技能已经自动加载
```

### 通过Skill Tool使用

智能体可以使用`skill`工具加载特定技能的完整内容：

```
TOOL: skill
INPUT: {"name": "ancient-poet"}
```

响应将包含技能的完整内容、元数据和基础目录。

### 编程访问

```go
import "code.vikunja.io/api/pkg/modules/ai"

// 获取技能管理器
sm := ai.GetSkillManager()

// 获取所有技能
skills := sm.GetAllSkills()

// 获取特定技能
skill, exists := sm.GetSkill("ancient-poet")

// 加载技能完整内容
content, err := sm.LoadSkillContent("ancient-poet")
```

## 配置

在`config.yml`中配置启用的技能（可选）：

```yaml
ai:
  enabled_skills:
    - ancient-poet
    - chinese-novelist
```

如果`enabled_skills`为空或未设置，将加载所有找到的技能。

## 示例技能

### Ancient Poet

位置：`skills/ancient-poet/`

功能：模仿古代诗人风格创作古诗

使用：
```
请模仿李白的风格写一首关于明月的诗
```

## 技能开发指南

### 1. 创建技能目录

```bash
mkdir -p skills/my-skill
```

### 2. 创建SKILL.md

```markdown
---
name: my-skill
description: Description of what this skill does
---

# My Skill

## 技能目的
...

## 使用时机
...

## 使用方法
...
```

### 3. 添加资源（可选）

可以添加以下目录：
- `references/` - 参考文档
- `scripts/` - 脚本文件
- `assets/` - 资产文件

### 4. 测试

```go
import "code.vikunja.io/api/pkg/modules/ai"

sm := ai.GetSkillManager()
skill, exists := sm.GetSkill("my-skill")

if !exists {
    // 技能未找到
}

content, err := sm.LoadSkillContent("my-skill")
```

## 技能预处理器

系统会自动预处理技能文件中的YAML frontmatter，以处理包含冒号的值：

```yaml
# 输入
license: See LICENSE.txt for details

# 自动转换为
license: |
  See LICENSE.txt for details
```

这确保YAML解析器能正确处理复杂的值。

## 调试

### 查看已加载的技能

```go
sm := ai.GetSkillManager()
skills := sm.GetAllSkills()

for name, info := range skills {
    fmt.Printf("%s: %s\n", name, info.Description)
}
```

### 日志输出

技能加载过程会输出日志：

```
[AI Skills] Loaded skill: ancient-poet - 此技能用于模仿古代诗人风格创作古诗...
[AI Skills] Loaded 5 skills
```

### Agent统计

```go
agent, err := ai.GetAgent()
stats := agent.GetStats()
fmt.Printf("Loaded %d skills\n", stats["num_skills"])
```

## 故障排除

### 技能未加载

1. 检查`SKILL.md`文件名是否正确（全大写）
2. 检查frontmatter是否包含`name`和`description`字段
3. 检查技能名称是否唯一
4. 查看日志中的错误信息

### 解析错误

1. 确保frontmatter以`---`开头和结尾
2. 确保name和description字段都有值
3. 如果description包含特殊字符（如冒号），系统会自动处理

### 技能工具不可用

1. 确认agent已正确初始化
2. 确认`RegisterDefaultTools()`已被调用
3. 检查配置中的enabled_tools设置

## 扩展

### 自定义技能目录

可以通过修改`skillSearchPaths()`函数添加其他搜索路径。

### 权限控制

可以扩展SkillManager以支持基于模式的权限控制，类似于OpenCode的实现。

### 技能依赖

可以添加技能依赖关系和版本控制功能。
