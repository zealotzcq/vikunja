import type {IChatMessage} from '@/modelTypes/IChatMessage'

const CHAT_HISTORY_KEY = 'vikunja-chat-history'

export function saveChatHistory(messages: IChatMessage[]) {
	localStorage.setItem(CHAT_HISTORY_KEY, JSON.stringify(messages))
}

export function loadChatHistory(): IChatMessage[] {
	const stored = localStorage.getItem(CHAT_HISTORY_KEY)
	if (stored === null) {
		return []
	}
	try {
		return JSON.parse(stored) as IChatMessage[]
	} catch (e) {
		console.error('Error loading chat history:', e)
		return []
	}
}

export function clearChatHistory() {
	localStorage.removeItem(CHAT_HISTORY_KEY)
}
