	import {ref, watch} from 'vue'
	import {defineStore, acceptHMRUpdate} from 'pinia'
	import {useRouter} from 'vue-router'

	import type {IChatMessage} from '@/modelTypes/IChatMessage'
	import {saveChatHistory} from '@/composables/useChatHistory'
	import ChatService from '@/services/chat'
	import {useAuthStore} from '@/stores/auth'
	import {useCompanyStore} from '@/stores/company'
	import {getToken} from '@/helpers/auth'

export const useChatStore = defineStore('chat', () => {
	const authStore = useAuthStore()
	const companyStore = useCompanyStore()
	const router = useRouter()
	const isMobile = ref(false)
	const isAvailable = ref(false)
	const isOpen = ref(localStorage.getItem('chatAssistantOpen') === 'true')
	const messages = ref<IChatMessage[]>([])
	const isLoading = ref(false)
	const error = ref<string | null>(null)
	const lastMessageId = ref<string>('')

	const chatService = new ChatService()
	let eventSource: EventSource | null = null

	function getProcessedRefreshMessages(): Set<string> {
		const stored = localStorage.getItem('vikunja-chat-processed-refresh')
		if (!stored) return new Set()
		try {
			return new Set(JSON.parse(stored))
		} catch {
			return new Set()
		}
	}

	function addProcessedRefreshMessage(msgId: string) {
		const set = getProcessedRefreshMessages()
		set.add(msgId)
		localStorage.setItem('vikunja-chat-processed-refresh', JSON.stringify([...set]))
	}

	function clearProcessedRefreshMessages() {
		localStorage.removeItem('vikunja-chat-processed-refresh')
	}

	function setMobile(value: boolean) {
		isMobile.value = value
		if (value && isOpen.value === false && authStore.authUser !== null && isAvailable.value) {
			isOpen.value = true
		}
	}

	watch(() => companyStore.currentCompanyId, async (newCompanyId) => {
		console.log('[Chat] currentCompanyId changed:', newCompanyId, 'authUser:', authStore.authUser)
		if (authStore.authUser && newCompanyId) {
			messages.value = []
			lastMessageId.value = ''
			await loadChatHistory()
			if (!isAvailable.value && isOpen.value) {
				console.log('[Chat] User not allowed in this company, closing chat panel')
				isOpen.value = false
				return
			}
			if (isOpen.value) {
				disconnectSSE()
				connectSSE()
			}
		} else {
			isAvailable.value = false
		}
	}, {immediate: true})

	function connectSSE() {
		if (eventSource) {
			eventSource.close()
		}

		const token = getToken()
		if (!token) {
			console.log('[Chat] No token available, skipping SSE connection')
			return
		}

		if (!companyStore.currentCompanyId) {
			console.log('[Chat] No company ID available, skipping SSE connection')
			return
		}

		const streamUrl = `${window.API_URL}/chat/stream?token=${encodeURIComponent(token)}&company_id=${companyStore.currentCompanyId}`
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
		console.log('[Chat] Loading chat history, currentCompanyId:', companyStore.currentCompanyId, 'companies:', companyStore.companies)
		if (!companyStore.currentCompanyId) {
			console.log('[Chat] No company ID available, skipping load')
			isAvailable.value = false
			return
		}
		isLoading.value = true
		error.value = null
		try {
			console.log('[Chat] Loading chat history for company:', companyStore.currentCompanyId)
			const response = await chatService.getHistory(companyStore.currentCompanyId)
			console.log('[Chat] Chat history loaded:', response)
			isAvailable.value = true

			const newMessages = response.messages.map(msg => ({
				id: msg.id,
				role: msg.role,
				content: msg.content,
				timestamp: msg.timestamp,
				navigationCommand: msg.navigationCommand,
				buttonNavigation: msg.buttonNavigation,
				questionData: msg.questionData,
			}))

			messages.value = newMessages

			if (newMessages.length > 0) {
				const lastMessage = newMessages[newMessages.length - 1]!
				lastMessageId.value = lastMessage.id

				if (lastMessage.navigationCommand) {
					console.log('[Chat] Executing navigation:', lastMessage.navigationCommand)
					router.push({
						name: lastMessage.navigationCommand.routeName,
						params: lastMessage.navigationCommand.params,
					})
					if (isMobile.value) {
						isOpen.value = false
					}
				}

				const processed = getProcessedRefreshMessages()
				if (lastMessage.buttonNavigation && !processed.has(lastMessage.id)) {
					console.log('[Chat] New button navigation message received, reloading page')
					addProcessedRefreshMessage(lastMessage.id)
					window.location.reload()
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

	async function executeButtonNavigation(routeName: string, params?: Record<string, any>) {
		console.log('[Chat] Executing button navigation:', {routeName, params})
		router.push({
			name: routeName,
			params: params,
		})
		if (isMobile.value) {
			isOpen.value = false
		}
	}

	async function submitQuestionAnswer(answer: string) {
		if (!authStore.authUser) {
			error.value = '请先登录以使用聊天助手'
			return
		}
		error.value = null

		try {
			await chatService.submitQuestionAnswer(answer, companyStore.currentCompanyId || undefined)
		} catch (err: any) {
			if (err?.response?.status === 401) {
				error.value = '请先登录以使用聊天助手'
			} else if (err?.message === 'No authentication token available') {
				error.value = '请先登录以使用聊天助手'
			} else {
				error.value = '提交答案失败，请稍后重试'
				console.error('[Chat] Failed to submit question answer:', err)
			}
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
				String(route.name),
				route.params,
				msgId,
				companyStore.currentCompanyId || undefined,
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
		clearProcessedRefreshMessages()
		await chatService.clearSession(companyStore.currentCompanyId || undefined)
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
		executeButtonNavigation,
		submitQuestionAnswer,
	}
})

if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useChatStore, import.meta.hot))
}
