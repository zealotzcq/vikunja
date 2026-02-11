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
}

export interface INavigationCommand {
	routeName: string
	params?: Record<string, any>
	label: string
}
