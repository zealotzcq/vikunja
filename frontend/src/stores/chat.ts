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
	const hasPendingResponse = ref(false)

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
		if (authStore.authUser && newCompanyId) {
			messages.value = []
			lastMessageId.value = ''
			await loadChatHistory()
			if (!isAvailable.value && isOpen.value) {
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
			return
		}

		if (!companyStore.currentCompanyId) {
			return
		}

		const streamUrl = `${window.API_URL}/chat/stream?token=${encodeURIComponent(token)}&company_id=${companyStore.currentCompanyId}`

		eventSource = new EventSource(streamUrl)

		eventSource.onmessage = (event) => {
			try {
				const update = JSON.parse(event.data)

				if (update.message_type === 'frontend_needed' && update.last_message_id !== lastMessageId.value) {
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
			error.value = null
		}
	}

	function disconnectSSE() {
		if (eventSource) {
			eventSource.close()
			eventSource = null
		}
	}

	async function loadChatHistory() {
		if (isLoading.value) return
		if (!authStore.authUser) {
			return
		}
		if (!companyStore.currentCompanyId) {
			isAvailable.value = false
			return
		}
		isLoading.value = true
		error.value = null
		try {
			const response = await chatService.getHistory(companyStore.currentCompanyId)
			isAvailable.value = true

			const newMessages = response.messages.map(msg => ({
				id: msg.id,
				type: msg.type || (msg.role === 'user' ? 'user_input' : 'assistant_response'),
				role: msg.role,
				content: msg.content,
				timestamp: msg.timestamp * 1000,
				navigationCommand: msg.navigationCommand,
				buttonNavigation: msg.buttonNavigation,
				questionData: msg.questionData,
				toolName: msg.toolName,
				toolInput: msg.toolInput,
				toolOutput: msg.toolOutput,
				toolCallID: msg.toolCallID,
			}))

			messages.value = newMessages

			if (newMessages.length > 0) {
				const lastMessage = newMessages[newMessages.length - 1]!
				lastMessageId.value = lastMessage.id

				if (hasPendingResponse.value && (lastMessage.type === 'assistant_response' || lastMessage.type === 'question' || lastMessage.type === 'button_navigation')) {
					hasPendingResponse.value = false
				}

				if (lastMessage.type === 'assistant_response' && lastMessage.navigationCommand) {
					router.push({
						name: lastMessage.navigationCommand.routeName,
						params: lastMessage.navigationCommand.params,
					})
					if (isMobile.value) {
						isOpen.value = false
					}
				}

				const processed = getProcessedRefreshMessages()

				const isButtonType = lastMessage.type === 'button_navigation' || (lastMessage.type === 'assistant_response' && lastMessage.buttonNavigation)

				if (isButtonType && lastMessage.buttonNavigation && !processed.has(lastMessage.id)) {
					const shouldAutoNavigate = !lastMessage.buttonNavigation.routeName.startsWith('task.detail')

					if (shouldAutoNavigate) {
						addProcessedRefreshMessage(lastMessage.id)
						router.push({
							name: lastMessage.buttonNavigation.routeName,
							params: lastMessage.buttonNavigation.params,
						})
						if (isMobile.value) {
							isOpen.value = false
						}
					}
				}
			}

			if (isMobile.value && isOpen.value === false) {
				isOpen.value = true
			}
		} catch (err) {
			const errObj = err as {response?: {status: number}}
			console.error('[Chat] Failed to load chat history:', err)
			if (errObj?.response?.status === 403) {
				isAvailable.value = false
			} else if (errObj?.response?.status === 401) {
				error.value = '请先登录以使用聊天助手'
			} else {
				console.error('[Chat] Failed to load chat history:', err)
			}
		} finally {
			isLoading.value = false
		}
	}

	async function executeButtonNavigation(routeName: string, params?: Record<string, unknown>) {
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
		} catch (err) {
			const errObj = err as {response?: {status: number}; message?: string}
			if (errObj?.response?.status === 401) {
				error.value = '请先登录以使用聊天助手'
			} else if (errObj?.message === 'No authentication token available') {
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
			type: 'user_input',
			role: 'user',
			content,
			timestamp: Date.now(),
		}
		messages.value.push(userMessage)
		lastMessageId.value = msgId
		saveChatHistory(messages.value)
		hasPendingResponse.value = true

		try {
			const route = router.currentRoute.value
			await chatService.sendMessage(
				content,
				String(route.name),
				route.params,
				msgId,
				companyStore.currentCompanyId || undefined,
			)
		} catch (err) {
			const errObj = err as {response?: {status: number}; message?: string}
			if (errObj?.response?.status === 401) {
				error.value = '请先登录以使用聊天助手'
			} else if (errObj?.message === 'No authentication token available') {
				error.value = '请先登录以使用聊天助手'
			} else {
				error.value = '发送消息失败，请稍后重试'
				console.error('[Chat] Failed to send message:', err)
			}
			hasPendingResponse.value = false
		}
	}

	function addMessage(message: IChatMessage) {
		messages.value.push(message)
		lastMessageId.value = message.id
		saveChatHistory(messages.value)
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
		hasPendingResponse,
		sendMessage,
		clearMessages,
		toggleOpen,
		loadChatHistory,
		setMobile,
		connectSSE,
		disconnectSSE,
		executeButtonNavigation,
		submitQuestionAnswer,
		addMessage,
	}
})

if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useChatStore, import.meta.hot))
}
