<template>
  <div class="mobile-container mobile-viewport">
    <div class="login-wrapper">
      <div class="logo-section">
        <div class="logo-icon">
          <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 2L2 7l10 5 10-5-10-5z"></path>
            <path d="M2 17l10 5 10-5"></path>
            <path d="M2 12l10 5 10-5"></path>
          </svg>
        </div>
        <h1 class="app-name">任务助手</h1>
        <p class="app-tagline">高效任务管理</p>
      </div>

      <div class="login-card">
        <h2 class="login-title">欢迎回来</h2>
        <p class="login-subtitle">登录以继续管理您的任务</p>

        <form @submit.prevent="onLogin" class="login-form">
          <div class="form-group">
            <label for="username" class="form-label">用户名</label>
            <div class="input-wrapper">
              <svg class="input-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
                <circle cx="12" cy="7" r="4"></circle>
              </svg>
              <input
                id="username"
                v-model="username"
                placeholder="输入用户名"
                class="input-field"
                type="text"
                autocomplete="username"
              />
            </div>
          </div>

          <div class="form-group">
            <label for="password" class="form-label">密码</label>
            <div class="input-wrapper">
              <svg class="input-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
                <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
              </svg>
              <input
                id="password"
                v-model="password"
                placeholder="输入密码"
                class="input-field"
                type="password"
                autocomplete="current-password"
              />
            </div>
          </div>

          <button type="submit" class="submit-button" :disabled="isLoading">
            <span v-if="!isLoading">登录</span>
            <span v-else class="loading-spinner"></span>
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import '../assets/mobile.css'

import { defineComponent, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../../src/stores/auth';
import { useChatStore } from '../../src/stores/chat';

export default defineComponent({
  name: 'MobileLoginView',
  setup() {
    const router = useRouter();
    const authStore = useAuthStore();
    const chatStore = useChatStore();
    const username = ref('');
    const password = ref('');
    const isLoading = ref(false);

    const onLogin = async () => {
      try {
        isLoading.value = true;
        await authStore.login({
          username: username.value,
          password: password.value,
        });
        const redirectPath = chatStore.isOpen ? '/mobile/chat' : '/mobile/home';
        router.replace(redirectPath);
      } catch (e) {
        console.error(e);
        alert('登录失败，请检查凭证');
      } finally {
        isLoading.value = false;
      }
    };

    return {
      username,
      password,
      isLoading,
      onLogin,
    };
  },
});
</script>

<style scoped>
.mobile-container {
  display: flex;
  justify-content: center;
  align-items: center;
  background: var(--color-background);
  overflow-y: auto;
}

.login-wrapper {
  width: 100%;
  padding: var(--spacing-lg) var(--spacing-md);
  display: flex;
  flex-direction: column;
  min-height: 100%;
  justify-content: center;
}

.logo-section {
  text-align: center;
  margin-bottom: var(--spacing-xl);
}

.logo-icon {
  width: 80px;
  height: 80px;
  margin: 0 auto var(--spacing-md);
  background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-primary-light) 100%);
  border-radius: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  box-shadow: var(--shadow-lg);
}

.app-name {
  font-size: var(--font-size-4xl);
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0 0 var(--spacing-xs) 0;
  font-family: var(--font-family);
}

.app-tagline {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
  margin: 0;
}

.login-card {
  background: var(--color-surface);
  border-radius: var(--radius-xl);
  padding: var(--spacing-xl) var(--spacing-lg);
  border: 1px solid var(--color-border);
  box-shadow: var(--shadow-md);
}

.login-title {
  font-size: var(--font-size-2xl);
  font-weight: 700;
  margin: 0 0 var(--spacing-xs) 0;
  text-align: center;
  color: var(--color-text-primary);
  font-family: var(--font-family);
}

.login-subtitle {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
  margin: 0 0 var(--spacing-xl) 0;
  text-align: center;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
  margin-bottom: var(--spacing-lg);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}

.form-label {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-text-primary);
}

.input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.input-icon {
  position: absolute;
  left: var(--spacing-sm);
  color: var(--color-text-muted);
  pointer-events: none;
}

.input-field {
  width: 100%;
  padding: 14px var(--spacing-sm) 14px 44px;
  border: 2px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--font-size-base);
  background: var(--color-background);
  color: var(--color-text-primary);
  font-family: var(--font-family);
  transition: border-color var(--transition-fast), background-color var(--transition-fast);
}

.input-field:focus {
  border-color: var(--color-primary);
  background: var(--color-surface);
}

.input-field::placeholder {
  color: var(--color-text-muted);
}

.submit-button {
  width: 100%;
  padding: 16px;
  min-height: 52px;
  background: var(--color-primary);
  color: white;
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--font-size-base);
  font-weight: 600;
  cursor: pointer;
  transition: background-color var(--transition-fast), transform var(--transition-fast);
  margin-top: var(--spacing-sm);
  font-family: var(--font-family);
  display: flex;
  align-items: center;
  justify-content: center;
}

.submit-button:hover:not(:disabled) {
  background: var(--color-primary-hover);
  transform: translateY(-1px);
}

.submit-button:active:not(:disabled) {
  background: var(--color-primary);
  transform: translateY(0);
}

.submit-button:disabled {
  background: var(--color-primary-lighter);
  color: var(--color-primary);
  cursor: not-allowed;
}

.loading-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .submit-button,
  .input-field,
  .loading-spinner {
    transition: none;
    animation: none;
  }

  .submit-button:hover:not(:disabled) {
    transform: none;
  }

  .submit-button:active:not(:disabled) {
    transform: none;
  }
}
</style>
