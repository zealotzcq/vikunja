<template>
  <nav class="mobile-tab-bar">
    <button
      v-for="tab in tabs"
      :key="tab.name"
      class="tab-item"
      :class="{ active: activeTab === tab.name }"
      @click="handleTabClick(tab)"
    >
      <component :is="tab.icon" :size="24" class="tab-icon" />
      <span class="tab-label">{{ tab.label }}</span>
    </button>
  </nav>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import { useRouter } from 'vue-router';
import HomeIcon from './icons/HomeIcon.vue';
import ChatIcon from './icons/ChatIcon.vue';
import ProjectsIcon from './icons/ProjectsIcon.vue';

interface Tab {
  name: string;
  label: string;
  icon: any;
  path: string;
}

export default defineComponent({
  name: 'MobileTabBar',
  components: {
    HomeIcon,
    ChatIcon,
    ProjectsIcon,
  },
  props: {
    activeTab: {
      type: String,
      required: true,
    },
  },
  setup() {
    const router = useRouter();

    const tabs: Tab[] = [
      { name: 'home', label: '首页', icon: HomeIcon, path: '/mobile/home' },
      { name: 'projects', label: '项目', icon: ProjectsIcon, path: '/mobile/projects' },
      { name: 'chat', label: '聊天', icon: ChatIcon, path: '/mobile/chat' },
    ];

    const handleTabClick = (tab: Tab) => {
      router.push(tab.path);
    };

    return {
      tabs,
      handleTabClick,
    };
  },
});
</script>

<style scoped>
.mobile-tab-bar {
  flex-shrink: 0;
  display: flex;
  justify-content: space-around;
  align-items: center;
  height: 64px;
  padding: 0;
  background: var(--color-surface);
  border-top: 1px solid var(--color-border);
  box-shadow: var(--shadow-sm);
  position: relative;
  z-index: 10;
}

.tab-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  gap: 4px;
  padding: 8px 12px;
  min-height: 52px;
  background: transparent;
  border: none;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background-color var(--transition-fast), transform var(--transition-fast);
}

.tab-item:hover {
  background: var(--color-surface-hover);
}

.tab-item:active {
  transform: scale(0.95);
}

.tab-item.active {
  background: transparent;
}

.tab-item.active .tab-icon {
  color: var(--color-primary);
}

.tab-item.active .tab-label {
  color: var(--color-primary);
}

.tab-icon {
  color: var(--color-text-muted);
  transition: color var(--transition-fast);
}

.tab-label {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--color-text-muted);
  font-family: var(--font-family);
  letter-spacing: 0.02em;
  transition: color var(--transition-fast);
}

@media (prefers-reduced-motion: reduce) {
  .tab-item {
    transition: none;
  }

  .tab-item:active {
    transform: none;
  }

  .tab-icon,
  .tab-label {
    transition: none;
  }
}
</style>
