<template>
  <div class="mobile-tab-layout mobile-viewport">
    <header v-if="!isDetailPage" class="mobile-header">
      <div class="header-left">
        <div
          v-if="companyStore.companies.length > 1"
          class="company-selector"
          @click="toggleCompanyDropdown"
        >
          <span class="company-name">{{ companyName }}</span>
          <svg class="dropdown-arrow" :class="{ 'rotate': showCompanyDropdown }" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="6 9 12 15 18 9"></polyline>
          </svg>
        </div>
        <span v-else class="company-name">{{ companyName }}</span>
        <transition name="fade">
          <div v-if="showCompanyDropdown" class="company-dropdown">
            <button
              v-for="company in companyStore.companies"
              :key="company.id"
              class="company-dropdown-item"
              :class="{ 'active': company.id === companyStore.currentCompanyId }"
              @click="handleCompanySwitch(company.id)"
            >
              {{ company.description }}
            </button>
          </div>
        </transition>
      </div>
      <div class="header-right">
        <div class="notification-wrapper">
          <Notifications />
        </div>
        <div class="avatar-wrapper" @click="toggleMenu">
          <div class="user-avatar">
            <img
              v-if="avatarUrl"
              :src="avatarUrl"
              alt=""
              width="40"
              height="40"
            >
            <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
              <circle cx="12" cy="7" r="4"></circle>
            </svg>
          </div>
        </div>
        <transition name="fade">
          <div v-if="showMenu" class="avatar-menu">
            <button class="menu-item logout-item" @click="handleLogout">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
                <polyline points="16 17 21 12 16 7"></polyline>
                <line x1="21" y1="12" x2="9" y2="12"></line>
              </svg>
              登出
            </button>
          </div>
        </transition>
      </div>
    </header>
    <main class="layout-content" :class="{ 'full-height': isDetailPage }">
      <RouterView />
    </main>
    <MobileTabBar v-if="!isDetailPage" :active-tab="activeTab" />
  </div>
</template>

<script lang="ts">
import '../assets/mobile.css'

import { defineComponent, computed, ref, onMounted, onUnmounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import MobileTabBar from './MobileTabBar.vue';
import { useCompanyStore } from '@/stores/company';
import { useAuthStore } from '@/stores/auth';
import Notifications from '@/components/notifications/Notifications.vue';

export default defineComponent({
  name: 'MobileTabLayout',
  components: {
    MobileTabBar,
    Notifications,
  },
  setup() {
    const route = useRoute();
    const router = useRouter();
    const companyStore = useCompanyStore();
    const authStore = useAuthStore();

    const activeTab = computed(() => {
      if (route.path.startsWith('/mobile/chat')) return 'chat';
      if (route.path.startsWith('/mobile/task') || route.path.startsWith('/mobile/project') || route.path.startsWith('/mobile/projects')) return 'projects';
      return 'home';
    });

    const isDetailPage = computed(() => {
      return route.path.startsWith('/mobile/task/');
    });

    const companyName = computed(() => companyStore.currentCompany?.description || '任务助手');
    const showMenu = ref(false);
    const showCompanyDropdown = ref(false);

    const toggleMenu = () => {
      showMenu.value = !showMenu.value;
      showCompanyDropdown.value = false;
    };

    const toggleCompanyDropdown = () => {
      showCompanyDropdown.value = !showCompanyDropdown.value;
      showMenu.value = false;
    };

    const closeMenu = () => {
      showMenu.value = false;
    };

    const closeCompanyDropdown = () => {
      showCompanyDropdown.value = false;
    };

    const handleLogout = () => {
      closeMenu();
      authStore.logout();
      router.replace('/mobile/login');
    };

    const handleCompanySwitch = (companyId: number) => {
      companyStore.setCurrentCompany(companyId);
      closeCompanyDropdown();
    };

    onMounted(() => {
      authStore.checkAuth();
      document.addEventListener('click', handleClickOutside);
      companyStore.loadCompanies();
    });

    onUnmounted(() => {
      document.removeEventListener('click', handleClickOutside);
    });

    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as HTMLElement;
      const avatarWrapper = target.closest('.avatar-wrapper');
      const menu = target.closest('.avatar-menu');
      if (!avatarWrapper && !menu) {
        closeMenu();
      }

      const companySelector = target.closest('.company-selector');
      const companyDropdown = target.closest('.company-dropdown');
      if (!companySelector && !companyDropdown) {
        closeCompanyDropdown();
      }
    };

    return {
      activeTab,
      isDetailPage,
      companyStore,
      companyName,
      avatarUrl: authStore.avatarUrl,
      showMenu,
      showCompanyDropdown,
      toggleMenu,
      toggleCompanyDropdown,
      closeMenu,
      handleLogout,
      handleCompanySwitch,
    };
  },
});
</script>

<style scoped>
.mobile-header {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
  height: 56px;
  min-height: 56px;
  box-shadow: var(--shadow-sm);
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 1000;
  width: 100%;
}

.header-left {
  flex: 1;
  display: flex;
  align-items: center;
  position: relative;
}

.company-selector {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  cursor: pointer;
  padding: var(--spacing-xs) var(--spacing-sm);
  border-radius: var(--radius-md);
  transition: background-color var(--transition-fast);
}

.company-selector:active {
  background: var(--color-surface-hover);
}

.company-name {
  font-size: var(--font-size-lg);
  font-weight: 700;
  color: var(--color-text-primary);
  font-family: var(--font-family);
}

.dropdown-arrow {
  color: var(--color-text-muted);
  transition: transform var(--transition-fast);
  flex-shrink: 0;
}

.dropdown-arrow.rotate {
  transform: rotate(180deg);
}

.company-dropdown {
  position: absolute;
  top: calc(100% + var(--spacing-xs));
  left: 0;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  min-width: 180px;
  max-height: 280px;
  overflow-y: auto;
  z-index: 100;
  padding: var(--spacing-xs);
}

.company-dropdown-item {
  width: 100%;
  padding: var(--spacing-sm) var(--spacing-md);
  min-height: 44px;
  text-align: left;
  border: none;
  background: transparent;
  color: var(--color-text-primary);
  font-size: var(--font-size-sm);
  font-weight: 500;
  font-family: var(--font-family);
  cursor: pointer;
  transition: background-color var(--transition-fast);
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  white-space: nowrap;
}

.company-dropdown-item:hover {
  background: var(--color-surface-hover);
}

.company-dropdown-item:active {
  background: var(--color-primary-lighter);
}

.company-dropdown-item.active {
  background: var(--color-primary-lighter);
  color: var(--color-primary);
  font-weight: 600;
}

.header-right {
  position: relative;
  display: flex;
  align-items: center;
}

.notification-wrapper {
  margin-right: var(--spacing-sm);
}

.notification-wrapper :deep(.trigger-button) {
  width: 40px;
  height: 40px;
  min-width: 40px;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  color: var(--color-text-primary);
  position: relative;
  border-radius: var(--radius-full);
  transition: background-color var(--transition-fast);
}

.notification-wrapper :deep(.trigger-button:active) {
  background: var(--color-surface-hover);
}

.notification-wrapper :deep(.trigger-button .icon) {
  width: 24px;
  height: 24px;
}

.notification-wrapper :deep(.unread-indicator) {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 10px;
  height: 10px;
  background: var(--color-cta);
  border-radius: 50%;
  border: 2px solid var(--color-surface);
}

.avatar-wrapper {
  cursor: pointer;
  transition: transform var(--transition-fast);
}

.avatar-wrapper:active {
  transform: scale(0.95);
}

.user-avatar {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-full);
  background: var(--color-primary-lighter);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
  border: 2px solid var(--color-primary-lighter);
  overflow: hidden;
}

.user-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.avatar-menu {
  position: absolute;
  top: calc(100% + var(--spacing-sm));
  right: 0;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  min-width: 140px;
  z-index: 100;
  overflow: hidden;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity var(--transition-fast), transform var(--transition-fast);
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

.menu-item {
  width: 100%;
  padding: var(--spacing-sm) var(--spacing-md);
  min-height: 44px;
  text-align: left;
  border: none;
  background: transparent;
  color: var(--color-text-primary);
  font-size: var(--font-size-sm);
  font-weight: 500;
  font-family: var(--font-family);
  cursor: pointer;
  transition: background-color var(--transition-fast);
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.menu-item:hover {
  background: var(--color-surface-hover);
}

.menu-item:active {
  background: var(--color-primary-lighter);
}

.logout-item {
  color: #DC2626;
  font-weight: 600;
}

.logout-item:hover {
  background: #FEE2E2;
}

.layout-content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
  padding-top: 56px;
  padding-bottom: 64px;
  box-sizing: border-box;
}

@media (min-width: 768px) {
  .mobile-header {
    position: static;
  }

  .layout-content {
    padding-top: 0;
    padding-bottom: 0;
  }
}

.layout-content.full-height {
  height: 100%;
}

@media (prefers-reduced-motion: reduce) {
  .avatar-wrapper,
  .company-selector,
  .fade-enter-active,
  .fade-leave-active,
  .menu-item,
  .company-dropdown-item {
    transition: none;
  }

  .fade-enter-from,
  .fade-leave-to {
    transform: none;
  }

  .avatar-wrapper:active {
    transform: none;
  }

  .company-selector:active {
    background: none;
  }

  .dropdown-arrow.rotate {
    transform: none;
  }
}

@media (min-width: 768px) {
  .mobile-header {
    position: static;
  }

  .layout-content {
    padding-top: 0;
    padding-bottom: 0;
  }
}
</style>
