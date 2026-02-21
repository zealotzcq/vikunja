# 任务助手 Mobile App Design

一个精美的商业级手机App界面，使用9:16竖屏布局。

## 设计系统

基于 **ui-ux-pro-max** 技能创建的设计系统，包含：

### 配色方案
- **主色**: `#0D9488` (Teal)
- **辅助色**: `#14B8A6` (Lighter Teal)
- **CTA色**: `#F97316` (Orange)
- **背景色**: `#F0FDFA` (Light Teal)
- **文本色**: `#134E4A` (Dark Teal)

### 字体
- **字体**: Plus Jakarta Sans
- **字重**: 300, 400, 500, 600, 700
- **大小**: 12px - 32px

### 设计风格
- **模式**: App Store Style Landing
- **风格**: Micro-interactions
- **特点**: 简洁、现代、专业、易用

## 9:16 竖屏布局

移动端页面强制使用9:16竖屏比例，无论浏览器尺寸如何。

### 功能特性
- ✅ 自动检测横屏并旋转显示
- ✅ 桌面端居中显示手机模拟器
- ✅ 防止横屏内容显示
- ✅ 流畅的过渡动画

### 响应式断点
- **手机竖屏**: < 768px
- **平板/桌面**: ≥ 768px (居中显示)
- **横屏模式**: 自动旋转

## 组件结构

```
frontend/mobile/
├── assets/
│   └── mobile.css              # 主样式入口
├── components/
│   ├── icons/                  # SVG图标
│   │   ├── HomeIcon.vue
│   │   ├── ChatIcon.vue
│   │   ├── ProjectsIcon.vue
│   │   ├── SearchIcon.vue
│   │   └── SettingsIcon.vue
│   ├── MobileTabBar.vue        # 底部导航栏
│   └── MobileTabLayout.vue     # 主布局
├── views/
│   ├── MobileLoginView.vue     # 登录页
│   ├── MobileHomeView.vue      # 首页
│   └── MobileChatView.vue      # 聊天页
├── router/
│   └── index.ts                # 路由配置
└── styles/
    ├── design-system.css       # 设计系统
    ├── mobile-viewport.css     # 9:16布局
    └── reset.css               # 样式重置
```

## 主要页面

### 1. 登录页 (MobileLoginView)
- 精美的Logo展示
- 渐变色图标
- 表单验证
- 加载状态
- 忘记密码/注册链接

### 2. 首页 (MobileHomeView)
- 用户问候区
- 搜索栏
- 今日概览统计卡片
- 任务列表（带优先级指示）
- 项目进度展示
- 浮动操作按钮（FAB）

### 3. 聊天页 (MobileChatView)
- 消息气泡
- 智能助手交互
- 问题面板
- 导航按钮
- 加载动画

## 样式使用指南

### CSS变量

所有样式使用CSS变量定义，可以轻松自定义：

```css
/* 颜色 */
--color-primary
--color-secondary
--color-cta
--color-background
--color-surface
--color-text-primary
--color-text-secondary
--color-border

/* 间距 */
--spacing-xs: 4px
--spacing-sm: 8px
--spacing-md: 16px
--spacing-lg: 24px
--spacing-xl: 32px

/* 圆角 */
--radius-sm: 8px
--radius-md: 12px
--radius-lg: 16px
--radius-xl: 20px
--radius-full: 9999px

/* 过渡 */
--transition-fast: 150ms
--transition-base: 200ms
--transition-slow: 300ms
```

### 通用组件类

```css
.card           /* 卡片 */
.btn-primary    /* 主按钮 */
.btn-cta        /* CTA按钮 */
.btn-secondary  /* 次要按钮 */
.btn-outline    /* 轮廓按钮 */
.btn-ghost      /* 幽灵按钮 */
.badge-primary  /* 主徽章 */
.badge-success  /* 成功徽章 */
.badge-warning  /* 警告徽章 */
.badge-error    /* 错误徽章 */
```

## 图标使用

使用SVG图标而非emoji，所有图标位于 `components/icons/` 目录：

```vue
<template>
  <HomeIcon :size="24" />
  <ChatIcon :size="24" />
  <ProjectsIcon :size="24" />
  <SearchIcon :size="24" />
  <SettingsIcon :size="24" />
</template>

<script>
import HomeIcon from './icons/HomeIcon.vue';
import ChatIcon from './icons/ChatIcon.vue';
import ProjectsIcon from './icons/ProjectsIcon.vue';
import SearchIcon from './icons/SearchIcon.vue';
import SettingsIcon from './icons/SettingsIcon.vue';
</script>
```

## 可访问性

- ✅ 键盘导航支持
- ✅ 焦点可见状态
- ✅ 屏幕阅读器友好
- ✅ 减少动画模式支持
- ✅ 足够的颜色对比度
- ✅ 触摸目标至少44x44px

## 性能优化

- 使用CSS变量减少重复样式
- 使用transform和opacity进行动画
- 避免布局抖动
- 使用will-change优化动画
- 图片懒加载（如需要）

## 浏览器支持

- Chrome 90+
- Safari 14+
- Firefox 88+
- Edge 90+
- iOS Safari 14+
- Chrome Android 90+

## 开发建议

1. **使用设计系统变量**：避免硬编码颜色和间距
2. **保持一致性**：使用相同的间距、圆角和过渡
3. **测试响应式**：在不同设备和屏幕尺寸上测试
4. **性能优先**：避免不必要的动画和重绘
5. **可访问性**：确保键盘导航和屏幕阅读器支持

## 未来增强

- [ ] 添加更多页面（项目列表、搜索、设置）
- [ ] 实现主题切换（深色模式）
- [ ] 添加手势支持
- [ ] 优化动画性能
- [ ] 添加更多交互效果
- [ ] 实现离线支持
