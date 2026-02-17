import {computed, type Ref} from 'vue'
import {useNow} from '@vueuse/core'
import dayjs from 'dayjs'
import {getToken} from '@/helpers/auth'
import {useChatStore} from '@/stores/chat'

type RefreshPattern = '#0#' | '#1#' | '#2#' | '#3#'

const STORAGE_KEY = 'taskRefreshTimestamps'

function getStoredRefreshTimes(): Record<number, string> {
	if (typeof window === 'undefined') return {}
	const stored = localStorage.getItem(STORAGE_KEY)
	return stored ? JSON.parse(stored) : {}
}

function setStoredRefreshTimes(timestamps: Record<number, string>) {
	if (typeof window === 'undefined') return
	localStorage.setItem(STORAGE_KEY, JSON.stringify(timestamps))
}

function updateStoredRefreshTime(projectId: number, time: dayjs.Dayjs, pattern: RefreshPattern | null) {
	const timestamps = getStoredRefreshTimes()
	let storedTime = time

	if (pattern === '#0#') {
		storedTime = time.subtract(1, 'day')
	}

	timestamps[projectId] = storedTime.toISOString()
	setStoredRefreshTimes(timestamps)
}

function extractRefreshPattern(title: string): RefreshPattern | null {
	const patterns: RefreshPattern[] = ['#0#', '#1#', '#2#', '#3#']
	for (const pattern of patterns) {
		if (title.endsWith(pattern)) {
			return pattern
		}
	}
	return null
}

function shouldRefreshBasedOnPattern(pattern: RefreshPattern, lastRefreshTime: dayjs.Dayjs, now: dayjs.Dayjs): boolean {
	switch (pattern) {
		case '#0#':
			return true
		case '#1#':
			return !lastRefreshTime.isSame(now, 'day')
		case '#2#':
			return !lastRefreshTime.isSame(now, 'week')
		case '#3#':
			return !lastRefreshTime.isSame(now, 'month')
		default:
			return false
	}
}

function getPeriodStartTime(pattern: RefreshPattern): dayjs.Dayjs {
	const now = dayjs()
	switch (pattern) {
		case '#0#':
			return now.add(1, 'day').startOf('day')
		case '#1#':
			return now.startOf('day')
		case '#2#':
			return now.startOf('week')
		case '#3#':
			return now.startOf('month')
		default:
			return now.startOf('day')
	}
}

export function useProjectTaskRefresh(project: Ref<{id: number, title: string}>) {
	const now = useNow()

	console.log('[frontend] useProjectTaskRefresh initialized with project:', project.value)

	const refreshPattern = computed(() => {
		const pattern = extractRefreshPattern(project.value.title)
		console.log('[frontend] Extracted refresh pattern:', pattern, 'from title:', project.value.title)
		return pattern
	})

	const lastRefreshTime = computed(() => {
		const timestamps = getStoredRefreshTimes()
		const stored = timestamps[project.value.id]
		const lastTime = stored ? dayjs(stored) : null
		console.log('[frontend] Last refresh time for project', project.value.id, ':', lastTime ? lastTime.format() : 'never')
		return lastTime
	})

	const shouldRefresh = computed(() => {
		if (!refreshPattern.value) {
			console.log('[frontend] No refresh pattern, shouldRefresh = false')
			return false
		}

		if (refreshPattern.value === '#0#') {
			console.log('[frontend] #0# pattern, should always refresh = true')
			return true
		}

		if (!lastRefreshTime.value) {
			console.log('[frontend] No last refresh time, shouldRefresh = true')
			return true
		}
		const should = shouldRefreshBasedOnPattern(refreshPattern.value, lastRefreshTime.value, dayjs(now.value))
		if (should) {
			console.log('[frontend] shouldRefreshBasedOnPattern:', should, 'pattern:', refreshPattern.value)
		}
		return should
	})

	async function refreshTasks(): Promise<void> {
		console.log('[frontend] refreshTasks called')
		if (!refreshPattern.value) {
			console.log('[frontend] No refresh pattern, skipping refresh')
			return
		}

		const periodStart = getPeriodStartTime(refreshPattern.value)
		const sinceTime = periodStart.toISOString()
		console.log('[frontend] Refreshing tasks since:', sinceTime, 'periodStart:', periodStart.format())

		try {
			const requestBody = {
				project_id: project.value.id,
				since_time: sinceTime,
			}
			console.log('[frontend] Sending refresh request:', requestBody)

			const token = getToken()
			console.log('[frontend] Auth token exists:', !!token)

			const headers: Record<string, string> = {
				'Content-Type': 'application/json',
			}
			if (token) {
				headers['Authorization'] = `Bearer ${token}`
			}

			const response = await fetch(`/api/v1/projects/${project.value.id}/refresh-tasks`, {
				method: 'POST',
				headers,
				body: JSON.stringify(requestBody),
			})

			console.log('[frontend] Refresh response status:', response.status, response.statusText)

			if (!response.ok) {
				const errorText = await response.text()
				console.error('[frontend] Refresh failed:', response.status, response.statusText, errorText)
				throw new Error(`Failed to refresh tasks: ${response.statusText}`)
			}

			const responseData = await response.json()
			console.log('[frontend] Refresh response data:', responseData)

			const storedTime = dayjs(now.value)
			updateStoredRefreshTime(project.value.id, storedTime, refreshPattern.value)
			console.log('[frontend] Updated stored refresh time to:', storedTime.format(), '(pattern:', refreshPattern.value + ')')

			// Trigger frontend task list refresh
			const chatStore = useChatStore()
			chatStore.homeRefreshTrigger = Date.now()
		} catch (error) {
			console.error('[frontend] Error refreshing tasks:', error)
			throw error
		}
	}

	async function checkAndRefreshTasks(): Promise<boolean> {
		console.log('[frontend] checkAndRefreshTasks called, shouldRefresh:', shouldRefresh.value)
		if (shouldRefresh.value) {
			console.log('[frontend] Starting refresh...')
			await refreshTasks()
			return true
		}
		console.log('[frontend] Skipping refresh, shouldRefresh is false')
		return false
	}

	return {
		refreshPattern,
		shouldRefresh,
		refreshTasks,
		checkAndRefreshTasks,
		lastRefreshTime,
	}
}
