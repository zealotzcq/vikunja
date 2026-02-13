	import {ref, watch} from 'vue'
	import {defineStore, acceptHMRUpdate} from 'pinia'
	import {useRouter} from 'vue-router'

	import type {IChatMessage} from '@/modelTypes/IChatMessage'
	import {saveChatHistory} from '@/composables/useChatHistory'
	import ChatService from '@/services/chat'
	import {useAuthStore} from '@/stores/auth'
	import {getToken} from '@/helpers/auth'

export const useChatStore = defineStore('chat', () => {
	const authStore = useAuthStore()
	const router = useRouter()
	const isMobile = ref(false)
	const isAvailable = ref(false)
	const isOpen = ref(false)
	const messages = ref<IChatMessage[]>([])
	const isLoading = ref(false)
	const error = ref<string | null>(null)
	const lastMessageId = ref<string>('')

	const chatService = new ChatService()
	let eventSource: EventSource | null = null

	function setMobile(value: boolean) {
		isMobile.value = value
		if (value && isOpen.value === false && authStore.authUser !== null && isAvailable.value) {
			isOpen.value = true
		}
	}

	function connectSSE() {
		if (eventSource) {
			eventSource.close()
		}

		const token = getToken()
		if (!token) {
			console.log('[Chat] No token available, skipping SSE connection')
			return
		}

		const streamUrl = `${window.API_URL}/chat/stream?token=${encodeURIComponent(token)}`
		console.log('[Chat] Connecting to SSE at:', streamUrl)

		eventSource = new EventSource(streamUrl)

		eventSource.onmessage = (event) => {
			try {
				const update = JSON.parse(event.data)
				console.log('[Chat] SSE update:', update)

				if (update.message_type === 'frontend_needed' && update.last_message_id !== lastMessageId.value) {
					console.log('[Chat] New message available, fetching history...')
					loadChatHistory()
				}
			} catch (err) {
				console.error('[Chat] Failed to parse SSE message:', err)
			}
		}

		eventSource.onerror = (err) => {
			console.error('[Chat] SSE error:', err)
			error.value = '连接丢失，正在重新连接...'
			setTimeout(() => connectSSE(), 5000)
		}

		eventSource.onopen = () => {
			console.log('[Chat] SSE connected')
			error.value = null
		}
	}

	function disconnectSSE() {
		if (eventSource) {
			console.log('[Chat] Disconnecting SSE...')
			eventSource.close()
			eventSource = null
		}
	}

	async function loadChatHistory() {
		if (isLoading.value) return
		if (!authStore.authUser) {
			return
		}
		isLoading.value = true
		error.value = null
		try {
			console.log('[Chat] Loading chat history...')
			const response = await chatService.getHistory()
			console.log('[Chat] Chat history loaded:', response)
			isAvailable.value = true

			const newMessages = response.messages.map(msg => ({
				id: msg.id,
				role: msg.role,
				content: msg.content,
				timestamp: msg.timestamp,
				navigationCommand: msg.navigationCommand,
			}))

			const existingIds = new Set(messages.value.map(m => m.id))
			const newItems = newMessages.filter(m => !existingIds.has(m.id))

			if (newItems.length > 0) {
				messages.value = newMessages
				lastMessageId.value = newMessages[newMessages.length - 1].id

				const lastMsg = newItems[newItems.length - 1]
				if (lastMsg.navigationCommand) {
					console.log('[Chat] Executing navigation:', lastMsg.navigationCommand)
					router.push({
						name: lastMsg.navigationCommand.routeName,
						params: lastMsg.navigationCommand.params,
					})
					if (isMobile.value) {
						isOpen.value = false
					}
				}
			}

			if (isMobile.value && isOpen.value === false) {
				isOpen.value = true
			}
		} catch (err: any) {
			console.error('[Chat] Failed to load chat history:', err)
			if (err?.response?.status === 403) {
				isAvailable.value = false
			} else if (err?.response?.status === 401) {
				error.value = '请先登录以使用聊天助手'
			} else {
				console.error('[Chat] Failed to load chat history:', err)
			}
		} finally {
			isLoading.value = false
		}
	}

	async function sendMessage(content: string) {
		if (!authStore.authUser) {
			error.value = '请先登录以使用聊天助手'
			return
		}
		error.value = null

		const msgId = `msg_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`

		const userMessage: IChatMessage = {
			id: msgId,
			role: 'user',
			content,
			timestamp: Date.now(),
		}
		messages.value.push(userMessage)
		lastMessageId.value = msgId
		saveChatHistory(messages.value)

		try {
			const route = router.currentRoute.value
			await chatService.sendMessage(
				content,
				route.name,
				route.params,
				msgId,
			)
		} catch (err: any) {
			if (err?.response?.status === 401) {
				error.value = '请先登录以使用聊天助手'
			} else if (err?.message === 'No authentication token available') {
				error.value = '请先登录以使用聊天助手'
			} else {
				error.value = '发送消息失败，请稍后重试'
				console.error('[Chat] Failed to send message:', err)
			}
		}
	}

	async function clearMessages() {
		error.value = null
		messages.value = []
		lastMessageId.value = ''
		await chatService.clearSession()
		localStorage.removeItem('vikunja-chat-history')
	}

	function toggleOpen() {
		isOpen.value = !isOpen.value
		localStorage.setItem('chatAssistantOpen', String(isOpen.value))
	}

	watch(isOpen, (newVal) => {
		localStorage.setItem('chatAssistantOpen', String(newVal))
		if (newVal) {
			connectSSE()
		} else {
			disconnectSSE()
		}
	}, {immediate: true})

	return {
		isMobile,
		isAvailable,
		isOpen,
		messages,
		isLoading,
		error,
		sendMessage,
		clearMessages,
		toggleOpen,
		loadChatHistory,
		setMobile,
		connectSSE,
		disconnectSSE,
	}
})

if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useChatStore, import.meta.hot))
}
