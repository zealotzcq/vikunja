import {AuthenticatedHTTPFactory} from '@/helpers/fetcher'
import {getToken} from '@/helpers/auth'

export default class ChatService {
	http = AuthenticatedHTTPFactory()

	async sendMessage(message: string, pageRoute: string, pageParams: Record<string, any> = {}): Promise<{
		id: string
		role: 'user' | 'assistant'
		content: string
		timestamp: number
		navigationCommand?: {
			routeName: string
			params?: Record<string, any>
			label: string
		}
	}> {
		const token = getToken()
		if (!token) {
			throw new Error('No authentication token available')
		}

		const response = await this.http.post('/chat/send', {
			message,
			page_info: {
				route_name: pageRoute,
				params: pageParams,
			},
		})
		return response.data
	}

	async getSession(): Promise<{
		id: string
		user_id: number
		created_at: number
		messages: Array<{
			id: string
			role: 'user' | 'assistant'
			content: string
			timestamp: number
			navigationCommand?: {
				routeName: string
				params?: Record<string, any>
				label: string
			}
		}>
		expires_at: number
	}> {
		const token = getToken()
		if (!token) {
			throw new Error('No authentication token available')
		}

		const response = await this.http.get('/chat/session')
		return response.data
	}

	async clearSession(): Promise<void> {
		const token = getToken()
		if (!token) {
			return
		}
		await this.http.delete('/chat/session')
	}
}
