export interface IChatMessage {
	id: string
	role: 'user' | 'assistant'
	content: string
	timestamp: number
	navigationCommand?: {
		routeName: string
		params?: Record<string, any>
		label: string
	}
	buttonNavigation?: {
		routeName: string
		params?: Record<string, any>
		label: string
	}
	questionData?: string
	questionAnswered?: boolean
}

export interface INavigationCommand {
	routeName: string
	params?: Record<string, any>
	label: string
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
