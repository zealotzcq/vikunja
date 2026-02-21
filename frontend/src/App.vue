<template>
	<!-- Mobile routes: show with MobileReady component -->
	<MobileReady v-if="isMobileRoute">
		<RouterView />
	</MobileReady>

	<!-- Desktop routes: show with desktop layout -->
	<Ready v-else>
		<template v-if="authStore.authUser">
			<AppHeader />
			<ContentAuth />
		</template>
		<ContentLinkShare v-else-if="authStore.authLinkShare" />
		<NoAuthWrapper
			v-else
			show-api-config
		>
			<RouterView />
		</NoAuthWrapper>

		<KeyboardShortcuts v-if="keyboardShortcutsActive" />

		<Teleport to="body">
			<AddToHomeScreen />
			<UpdateNotification />
			<Notification />
			<DemoMode />
			<ChatAssistant />
		</Teleport>
	</Ready>
</template>

<script lang="ts" setup>
import {computed, watch, onMounted, onUnmounted} from 'vue'
import {useRoute} from 'vue-router'
import {useI18n} from 'vue-i18n'
import isTouchDevice from 'is-touch-device'

import Notification from '@/components/misc/Notification.vue'
import UpdateNotification from '@/components/home/UpdateNotification.vue'
import KeyboardShortcuts from '@/components/misc/keyboard-shortcuts/index.vue'

import AppHeader from '@/components/home/AppHeader.vue'
import ContentAuth from '@/components/home/ContentAuth.vue'
import ContentLinkShare from '@/components/home/ContentLinkShare.vue'
import NoAuthWrapper from '@/components/misc/NoAuthWrapper.vue'
import Ready from '@/components/misc/Ready.vue'
import MobileReady from '../mobile/components/MobileReady.vue'

import {DEFAULT_LANGUAGE, setLanguage} from '@/i18n'

import {useAuthStore} from '@/stores/auth'
import {useBaseStore} from '@/stores/base'

import {useColorScheme} from '@/composables/useColorScheme'
import {useBodyClass} from '@/composables/useBodyClass'
import AddToHomeScreen from '@/components/home/AddToHomeScreen.vue'
import DemoMode from '@/components/home/DemoMode.vue'
import ChatAssistant from '@/components/chat-assistant/ChatAssistant.vue'

const importAccountDeleteService = () => import('@/services/accountDelete')
import {success} from '@/message'

const authStore = useAuthStore()
const baseStore = useBaseStore()

const route = useRoute()

// Check if current route is a mobile route
const isMobileRoute = computed(() => route.path.startsWith('/mobile'))

// Track mobile route state for CSS selectors
watch(isMobileRoute, (isMobile) => {
  if (isMobile) {
    document.body.classList.add('mobile-route')
  } else {
    document.body.classList.remove('mobile-route')
  }
})

onMounted(() => {
  if (isMobileRoute.value) {
    document.body.classList.add('mobile-route')
  }
})

onUnmounted(() => {
  document.body.classList.remove('mobile-route')
})

useBodyClass('is-touch', isTouchDevice())
useBodyClass('compact-mode', baseStore.compactMode)
const keyboardShortcutsActive = computed(() => baseStore.keyboardShortcutsActive)

const {t} = useI18n({useScope: 'global'})

// setup account deletion verification
const accountDeletionConfirm = computed(() => route.query?.accountDeletionConfirm as (string | undefined))
watch(accountDeletionConfirm, async (accountDeletionConfirm) => {
	if (accountDeletionConfirm === undefined) {
		return
	}

	const AccountDeleteService = (await importAccountDeleteService()).default
	const accountDeletionService = new AccountDeleteService()
	await accountDeletionService.confirm(accountDeletionConfirm)
	success({message: t('user.deletion.confirmSuccess')})
	authStore.refreshUserInfo()
}, { immediate: true })

setLanguage(authStore.settings.language ?? DEFAULT_LANGUAGE)
useColorScheme()
</script>

<style lang="scss" src="@/styles/global.scss" />
