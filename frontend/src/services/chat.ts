import {AuthenticatedHTTPFactory} from '@/helpers/fetcher'
import {getToken} from '@/helpers/auth'

export default class ChatService {
	http = AuthenticatedHTTPFactory()

	async sendMessage(
		message: string,
		pageRoute: string,
		pageParams: Record<string, unknown> = {},
		messageID?: string,
		companyID: number | undefined = undefined,
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
			type?: string
			role: 'user' | 'assistant' | 'tool'
			content: string
			timestamp: number
			navigationCommand?: {
				routeName: string
				params?: Record<string, unknown>
				label: string
			}
			buttonNavigation?: {
				routeName: string
				params?: Record<string, unknown>
				label: string
				title?: string
			}
			questionData?: string
			toolName?: string
			toolInput?: string
			toolOutput?: string
			toolCallID?: string
		}>
		expires_at: number
	}> {
		const token = getToken()
		if (!token) {
			throw new Error('No authentication token available')
		}

		const response = await this.http.get(`/chat/history?company_id=${companyID}`)
		return response.data
	}

	async clearSession(companyID: number | undefined = undefined): Promise<void> {
		const token = getToken()
		if (!token) {
			return
		}
		await this.http.delete(`/chat/session?company_id=${companyID}`)
	}

	async checkNewMessages(companyID?: number, lastMessageId?: string): Promise<{
		has_new: boolean
		last_message_id: string
	}> {
		const token = getToken()
		if (!token) {
			throw new Error('No authentication token available')
		}

		const lastIdParam = lastMessageId ? `&last_message_id=${encodeURIComponent(lastMessageId)}` : ''
		const response = await this.http.get(`/chat/check-new?company_id=${companyID}${lastIdParam}`)
		return response.data
	}

	async submitQuestionAnswer(
		answer: string,
		companyID: number | undefined = undefined,
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

	async setCurrentTask(
		taskId: number,
		title: string,
		projectId: number,
		companyID: number | undefined = undefined,
	): Promise<void> {
		const token = getToken()
		if (!token) {
			throw new Error('No authentication token available')
		}

		await this.http.post('/chat/set-current-task', {
			task_id: taskId,
			title,
			project_id: projectId,
			company_id: companyID,
		})
	}
}
