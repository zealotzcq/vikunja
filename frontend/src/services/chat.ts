import {AuthenticatedHTTPFactory} from '@/helpers/fetcher'
import {getToken} from '@/helpers/auth'

export default class ChatService {
	http = AuthenticatedHTTPFactory()

	async sendMessage(
		message: string,
		pageRoute: string,
		pageParams: Record<string, any> = {},
		messageID?: string,
		companyID?: number,
	): Promise<void> {
		const token = getToken()
		if (!token) {
			throw new Error('No authentication token available')
		}

		await this.http.post('/chat/send', {
			message_id: messageID,
			message,
			page_info: {
				route_name: pageRoute,
				params: pageParams,
			},
			use_agent: true,
			company_id: companyID,
		})
	}

	async getHistory(companyID?: number): Promise<{
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

		const response = await this.http.get('/chat/history', {
			company_id: companyID,
		})
		return response.data
	}

	async clearSession(companyID?: number): Promise<void> {
		const token = getToken()
		if (!token) {
			return
		}
		await this.http.delete('/chat/session', {
			company_id: companyID,
		})
	}

	async submitQuestionAnswer(
		answer: string,
		companyID?: number,
	): Promise<void> {
		const token = getToken()
		if (!token) {
			throw new Error('No authentication token available')
		}

		await this.http.post('/chat/submit-question-answer', {
			answer,
			company_id: companyID,
		})
	}
}
