import {onMounted, onBeforeUnmount} from 'vue'

export function useTaskRefresh(refreshCallback: () => void | Promise<void>) {
	function handleChatButtonNavigationRendered() {
		console.log('[TaskRefresh] Chat button navigation rendered, refreshing tasks')
		refreshCallback()
	}

	onMounted(() => {
		if (typeof window !== 'undefined') {
			window.addEventListener('chat-button-navigation-rendered', handleChatButtonNavigationRendered)
		}
	})

	onBeforeUnmount(() => {
		if (typeof window !== 'undefined') {
			window.removeEventListener('chat-button-navigation-rendered', handleChatButtonNavigationRendered)
		}
	})

	return {
		handleChatButtonNavigationRendered,
	}
}
