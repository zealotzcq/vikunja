import {ref, watch} from 'vue'
import {defineStore, acceptHMRUpdate} from 'pinia'

import type {IChatMessage} from '@/modelTypes/IChatMessage'
import {saveChatHistory} from '@/composables/useChatHistory'
import ChatService from '@/services/chat'
import {useAuthStore} from '@/stores/auth'

export const useChatStore = defineStore('chat', () => {
	const authStore = useAuthStore()
	const isOpen = ref(false)
	const messages = ref<IChatMessage[]>([])
	const isLoading = ref(false)
	const error = ref<string | null>(null)

	const chatService = new ChatService()

	async function loadSession() {
		if (isLoading.value) return
		if (!authStore.authUser) {
			return
		}
		isLoading.value = true
		error.value = null
		try {
			const sessionData = await chatService.getSession()
			messages.value = sessionData.messages.map(msg => ({
				id: msg.id,
				role: msg.role,
				content: msg.content,
				timestamp: msg.timestamp,
			}))
		} catch (err: any) {
			if (err?.response?.status === 401) {
				error.value = '请先登录以使用聊天助手'
			} else {
				console.error('Failed to load chat session:', err)
			}
		} finally {
			isLoading.value = false
		}
	}

	async function sendMessage(content: string, router?: any) {
		if (!authStore.authUser) {
			error.value = '请先登录以使用聊天助手'
			return
		}
		error.value = null
		const userMessage: IChatMessage = {
			id: `msg-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`,
			role: 'user',
			content,
			timestamp: Date.now(),
		}
		messages.value.push(userMessage)
		saveChatHistory(messages.value)

		if (router) {
			try {
				const route = router.currentRoute.value
				const response = await chatService.sendMessage(
					content,
					route.name,
					route.params,
				)

				const assistantMessage: IChatMessage = {
					id: response.id,
					role: 'assistant',
					content: response.content,
					timestamp: Date.now(),
				}
				messages.value.push(assistantMessage)
				saveChatHistory(messages.value)
			} catch (err: any) {
				if (err?.response?.status === 401) {
					error.value = '请先登录以使用聊天助手'
				} else if (err?.message === 'No authentication token available') {
					error.value = '请先登录以使用聊天助手'
				} else {
					error.value = '发送消息失败，请稍后重试'
					console.error('Failed to send message:', err)
				}
			}
		}
	}

	async function clearMessages() {
		error.value = null
		messages.value = []
		await chatService.clearSession()
		localStorage.removeItem('vikunja-chat-history')
	}

	function toggleOpen() {
		isOpen.value = !isOpen.value
		localStorage.setItem('chatAssistantOpen', String(isOpen.value))
	}

	watch(isOpen, (newVal) => {
		localStorage.setItem('chatAssistantOpen', String(newVal))
	}, {immediate: true})

	return {
		isOpen,
		messages,
		isLoading,
		error,
		sendMessage,
		clearMessages,
		toggleOpen,
		loadSession,
	}
})

if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useChatStore, import.meta.hot))
}
