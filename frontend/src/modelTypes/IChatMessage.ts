export type MessageRole = 'user' | 'assistant' | 'tool'
export type MessageType = 'user_input' | 'tool_call' | 'tool_result' | 'assistant_response' | 'question' | 'button_navigation' | 'question_answer'

export interface IChatMessage {
	id: string
	type: MessageType
	role: MessageRole
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
	questionAnswered?: boolean
	toolName?: string
	toolInput?: string
	toolOutput?: string
	toolCallID?: string
}

export interface INavigationCommand {
	routeName: string
	params?: Record<string, unknown>
	label: string
	title?: string
}

export interface IQuestion {
	question: string
	header: string
	options: IQuestionOption[]
	multiple?: boolean
}

export interface IQuestionOption {
	label: string
	description: string
}
