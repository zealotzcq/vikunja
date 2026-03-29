<template>
	<div class="mobile-task-detail">
		<div class="task-header">
			<div
				class="header-banner"
				:style="{ background: getPriorityColor(task?.priority) }"
			>
				<button
					class="back-btn"
					aria-label="返回"
					@click="goBack"
				>
					<svg
						width="24"
						height="24"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<polyline points="15 18 9 12 15 6" />
					</svg>
					<span class="back-text">返回</span>
				</button>
			</div>
		</div>

		<main
			class="task-content"
			:class="{ 'is-loading': isLoading }"
		>
			<div
				v-if="loadError && !task && !isLoading"
				class="empty-state error"
			>
				<svg
					width="64"
					height="64"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="1.5"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<circle
						cx="12"
						cy="12"
						r="10"
					/>
					<line
						x1="12"
						y1="8"
						x2="12"
						y2="12"
					/>
					<line
						x1="12"
						y1="16"
						x2="12.01"
						y2="16"
					/>
				</svg>
				<h3 class="empty-title">
					无法加载任务
				</h3>
				<p class="empty-subtitle">
					{{ loadError }}
				</p>
				<button
					class="retry-btn"
					@click="loadTask"
				>
					<svg
						width="20"
						height="20"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<polyline points="23 4 23 10 17 10" />
						<path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
					</svg>
					重新加载
				</button>
			</div>

			<div
				v-else-if="!task && !isLoading"
				class="empty-state"
			>
				<h3 class="empty-title">
					任务不存在
				</h3>
			</div>

			<div
				v-else-if="task"
				class="task-details"
			>
				<section
					v-if="task.projectId"
					class="project-section"
				>
					<div class="project-badge">
						<svg
							width="16"
							height="16"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
						>
							<rect
								x="3"
								y="3"
								width="18"
								height="18"
								rx="2"
								ry="2"
							/>
							<line
								x1="9"
								y1="3"
								x2="9"
								y2="21"
							/>
						</svg>
						{{ projectName }}
					</div>
				</section>

				<section class="detail-section">
					<div class="title-section">
						<input
							v-if="isEditingTitle"
							ref="titleInputRef"
							v-model="tempTitle"
							class="task-title-input"
							@blur="saveTitle"
							@keyup.enter="saveTitle"
						>
						<h1
							v-else
							class="task-title"
							:class="{ 'done': task.done }"
						>
							{{ task.title }}
						</h1>
						<button
							class="edit-title-btn"
							:aria-label="isEditingTitle ? '保存' : '编辑'"
							@click="toggleEditTitle"
						>
							<svg
								v-if="!isEditingTitle"
								width="18"
								height="18"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
							>
								<path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
								<path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
							</svg>
							<svg
								v-else
								width="18"
								height="18"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
							>
								<polyline points="20 6 9 17 4 12" />
							</svg>
						</button>
					</div>
				</section>

				<section
					v-if="task.description"
					class="detail-section"
				>
					<h3 class="section-title">
						描述
					</h3>
					<div
						class="description"
						v-html="task.description"
					/>
				</section>

				<section class="detail-section">
					<h3 class="section-title">
						属性
					</h3>
					<div class="attributes-list">
						<div class="attribute-item">
							<span class="attribute-label">优先级</span>
							<div class="attribute-value">
								<select
									v-model="tempPriority"
									class="priority-select"
									:disabled="isSaving"
									@change="onPriorityChange(tempPriority)"
								>
									<option :value="PRIORITIES.LOW">
										{{ getPriorityLabel(PRIORITIES.LOW) }}
									</option>
									<option :value="PRIORITIES.MEDIUM">
										{{ getPriorityLabel(PRIORITIES.MEDIUM) }}
									</option>
									<option :value="PRIORITIES.HIGH">
										{{ getPriorityLabel(PRIORITIES.HIGH) }}
									</option>
									<option :value="PRIORITIES.URGENT">
										{{ getPriorityLabel(PRIORITIES.URGENT) }}
									</option>
								</select>
							</div>
						</div>
						<div class="attribute-item">
							<span class="attribute-label">截止日期</span>
							<span
								class="attribute-value"
								:class="{ 'overdue': isOverdue(task.dueDate) }"
							>
								{{ task.dueDate ? formatDate(task.dueDate) : '无' }}
							</span>
						</div>
						<div class="attribute-item">
							<span class="attribute-label">进度</span>
							<div class="attribute-value percent-done-container">
								<input
									v-model.number="tempPercentDone"
									type="range"
									min="0"
									max="100"
									step="10"
									class="percent-done-slider"
									:disabled="isSaving"
									@input="onPercentDoneChange(tempPercentDone)"
								>
								<span class="percent-done-value">{{ tempPercentDone }}%</span>
							</div>
						</div>
					</div>
				</section>

				<section
					v-if="task.assignees && task.assignees.length > 0"
					class="detail-section"
				>
					<h3 class="section-title">
						指派给
					</h3>
					<div class="assignees-list">
						<div
							v-for="assignee in task.assignees"
							:key="assignee.id"
							class="assignee-item"
						>
							<div class="assignee-avatar">
								<img
									v-if="assignee.avatarUrl"
									:src="assignee.avatarUrl"
									alt=""
								>
								<svg
									v-else
									width="24"
									height="24"
									viewBox="0 0 24 24"
									fill="none"
									stroke="currentColor"
									stroke-width="2"
									stroke-linecap="round"
									stroke-linejoin="round"
								>
									<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
									<circle
										cx="12"
										cy="7"
										r="4"
									/>
								</svg>
							</div>
							<span class="assignee-name">{{ assignee.username || assignee.name }}</span>
						</div>
					</div>
				</section>

				<section
					v-if="task.labels && task.labels.length > 0"
					class="detail-section"
				>
					<h3 class="section-title">
						标签
					</h3>
					<div class="labels-list">
						<span 
							v-for="label in task.labels" 
							:key="label.id" 
							class="label-badge"
							:style="{ background: label.hexColor }"
						>
							{{ label.title }}
						</span>
					</div>
				</section>

				<section
					v-if="task.attachments && task.attachments.length > 0"
					class="detail-section"
				>
					<h3 class="section-title">
						附件
					</h3>
					<div class="attachments-list">
						<div 
							v-for="attachment in task.attachments" 
							:key="attachment.id" 
							class="attachment-item"
						>
							<div class="attachment-icon">
								<svg
									width="24"
									height="24"
									viewBox="0 0 24 24"
									fill="none"
									stroke="currentColor"
									stroke-width="2"
									stroke-linecap="round"
									stroke-linejoin="round"
								>
									<path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z" />
									<polyline points="13 2 13 9 20 9" />
								</svg>
							</div>
							<span class="attachment-name">{{ attachment.file.name }}</span>
						</div>
					</div>
				</section>

				<section class="detail-section">
					<h3 class="section-title">
						评论 ({{ comments.length }})
					</h3>
					<div class="comments-list">
						<div
							v-for="comment in comments"
							:key="comment.id"
							class="comment-item"
						>
							<div class="comment-header">
								<span class="comment-author">{{ comment.author?.username || comment.author?.name }}</span>
								<span class="comment-time">{{ formatTime(comment.created) }}</span>
							</div>
							<div
								class="comment-content"
								v-html="comment.comment"
							/>
						</div>
						<div
							v-if="comments.length === 0"
							class="no-comments"
						>
							暂无评论
						</div>
					</div>
					<div class="add-comment-section">
						<textarea
							v-model="newCommentText"
							placeholder="添加评论..."
							class="comment-input"
							:disabled="isAddingComment"
							rows="3"
						/>
						<button
							class="submit-comment-btn"
							:disabled="!newCommentText.trim() || isAddingComment"
							@click="addComment"
						>
							<span v-if="isAddingComment">发送中...</span>
							<span v-else>发送评论</span>
						</button>
					</div>
				</section>
			</div>
		</main>
	</div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useTaskStore } from '@/stores/tasks'
import { useProjectStore } from '@/stores/projects'
import { useBaseStore } from '@/stores/base'
import { useChatStore } from '@/stores/chat'
import { useCompanyStore } from '@/stores/company'
import TaskService from '@/services/task'
import TaskCommentService from '@/services/taskComment'
import TaskCommentModel from '@/models/taskComment'
import ChatService from '@/services/chat'
import { success } from '@/message'
import type { ITask } from '@/modelTypes/ITask'
import type { ITaskComment } from '@/modelTypes/ITaskComment'
import { PRIORITIES } from '@/constants/priorities'

const router = useRouter()
const route = useRoute()
const taskStore = useTaskStore()
const projectStore = useProjectStore()
const baseStore = useBaseStore()
const chatStore = useChatStore()
const companyStore = useCompanyStore()
const chatService = new ChatService()
const isLoading = ref(false)
const loadError = ref<string | null>(null)
const taskData = ref<ITask | null>(null)
const taskCommentService = new TaskCommentService()
const comments = ref<ITaskComment[]>([])
const newCommentText = ref('')
const isAddingComment = ref(false)
const tempPriority = ref<number>(0)
const tempPercentDone = ref<number>(0)
const tempTitle = ref('')
const isEditingTitle = ref(false)
const titleInputRef = ref<HTMLInputElement | null>(null)
const isSaving = ref(false)

const taskId = computed(() => Number(route.params.taskId))

const task = computed(() => {
	if (!taskData.value) return null

	return {
		...taskData.value,
		title: isEditingTitle.value ? tempTitle.value : taskData.value.title,
		assignees: taskData.value.assignees || [],
		labels: taskData.value.labels || [],
		attachments: taskData.value.attachments || [],
		comments: taskData.value.comments || [],
		priority: tempPriority.value,
		percentDone: tempPercentDone.value,
	}
})

const projectName = computed(() => {
	if (!task.value) return ''
	const project = projectStore.projects[task.value.projectId]
	return project?.title || '未知项目'
})

const goBack = () => {
	router.back()
}

const toggleTaskDone = async () => {
	if (!task.value) return
	try {
		await taskStore.update({
			...task.value,
			done: !task.value.done,
		})
	} catch (error) {
		console.error('Failed to toggle task:', error)
	}
}

const toggleEditTitle = async () => {
	if (isEditingTitle.value) {
		await saveTitle()
	} else {
		tempTitle.value = task.value?.title || ''
		isEditingTitle.value = true
		await nextTick()
		titleInputRef.value?.focus()
	}
}

const saveTitle = async () => {
	if (!task.value || !tempTitle.value.trim()) {
		isEditingTitle.value = false
		return
	}

	try {
		isSaving.value = true
		const updatedTask = await taskStore.update({
			...task.value,
			title: tempTitle.value.trim(),
		})
		if (taskData.value) {
			taskData.value = updatedTask
		}
		success({ message: '标题已更新' })
	} catch (error) {
		console.error('Failed to update title:', error)
		tempTitle.value = task.value?.title || ''
	} finally {
		isSaving.value = false
		isEditingTitle.value = false
	}
}

const isOverdue = (dueDate: Date | null) => {
	if (!dueDate) return false
	return new Date(dueDate) < new Date()
}

const formatDate = (dateString: Date | string) => {
	const date = new Date(dateString)
	const now = new Date()
	const diff = date.getTime() - now.getTime()
	const days = Math.ceil(diff / (1000 * 60 * 60 * 24))

	if (days === 0) return '今天'
	if (days === 1) return '明天'
	if (days === -1) return '昨天'
	if (days > 1) return `${days}天后`
	if (days < -1) return `${Math.abs(days)}天前`

	return date.toLocaleDateString('zh-CN', {
		year: 'numeric',
		month: 'short',
		day: 'numeric',
		hour: '2-digit',
		minute: '2-digit',
	})
}

const formatTime = (dateString: Date | string) => {
	const date = new Date(dateString)
	const now = new Date()
	const diff = now.getTime() - date.getTime()
	const minutes = Math.floor(diff / (1000 * 60))
	const hours = Math.floor(diff / (1000 * 60 * 60))
	const days = Math.floor(diff / (1000 * 60 * 60 * 24))

	if (minutes < 1) return '刚刚'
	if (minutes < 60) return `${minutes}分钟前`
	if (hours < 24) return `${hours}小时前`
	if (days < 7) return `${days}天前`

	return date.toLocaleDateString('zh-CN', {
		month: 'short',
		day: 'numeric',
	})
}

const getPriorityLabel = (priority: number) => {
	const labels = { 0: '无', 1: '低', 2: '中', 3: '高', 4: '紧急', 5: '立即' }
	return labels[priority as keyof typeof labels] || '无'
}

const getPriorityColor = (priority: number) => {
	const colors = { 0: '#9CA3AF', 1: '#10B981', 2: '#F59E0B', 3: '#F97316', 4: '#EF4444', 5: '#7C3AED' }
	return colors[priority as keyof typeof colors] || '#9CA3AF'
}

const loadTask = async () => {
	try {
		isLoading.value = true
		loadError.value = null

		const taskService = new TaskService()
		const loaded = await taskService.get({ id: taskId.value }, {
			expand: ['reactions', 'comments', 'is_unread'],
		})

		taskData.value = loaded
		taskStore.tasks[taskId.value] = loaded

		tempTitle.value = loaded.title
		tempPriority.value = loaded.priority
		tempPercentDone.value = loaded.percentDone

		if (loaded.projectId) {
			try {
				await projectStore.loadProject(loaded.projectId)
			} catch (e) {
				console.warn('Failed to load project info:', e)
			}
		}

		if (loaded.isUnread) {
			await taskStore.markTaskAsRead(loaded.id)
			loaded.isUnread = false
		}

		if (loaded.id && loaded.title && loaded.projectId) {
			try {
				await chatService.setCurrentTask(
					loaded.id,
					loaded.title,
					loaded.projectId,
					companyStore.currentCompanyId,
				)
				await chatStore.setCurrentTask(loaded.id, loaded.title, loaded.projectId)
			} catch (e) {
				console.error('Failed to set current task in mobile:', e)
			}
		}

		await loadComments()
	} catch (error: any) {
		console.error('Failed to load task:', error)
		if (error?.response?.status === 404) {
			loadError.value = '任务不存在或您没有权限访问'
		} else {
			loadError.value = '加载任务时出错，请稍后重试'
		}
	} finally {
		isLoading.value = false
	}
}

const loadComments = async () => {
	if (!task.value) return
	try {
		comments.value = await taskCommentService.getAll({ taskId: task.value.id })
	} catch (error) {
		console.error('Failed to load comments:', error)
	}
}

const saveTaskChanges = async () => {
	if (!taskData.value) return

	try {
		isSaving.value = true
		const updatedTask = await taskStore.update({
			...taskData.value,
			priority: tempPriority.value,
			percentDone: tempPercentDone.value,
		})
		taskData.value = updatedTask
		taskStore.tasks[taskId.value] = updatedTask
		success({ message: '任务已更新' })
	} catch (error) {
		console.error('Failed to update task:', error)
	} finally {
		isSaving.value = false
	}
}

const onPriorityChange = async (value: number) => {
	if (!task.value) return
	tempPriority.value = value
	await saveTaskChanges()
}

const onPercentDoneChange = async (value: number) => {
	if (!taskData.value) return
	tempPercentDone.value = value
	await saveTaskChanges()
}

const addComment = async () => {
	if (!task.value || !newCommentText.value.trim()) return

	try {
		isAddingComment.value = true
		const comment = new TaskCommentModel()
		comment.taskId = task.value.id
		comment.comment = newCommentText.value

		const created = await taskCommentService.create(comment)
		comments.value.push(created)
		newCommentText.value = ''
		success({ message: '评论已添加' })
	} catch (error) {
		console.error('Failed to add comment:', error)
	} finally {
		isAddingComment.value = false
	}
}

onMounted(() => {
	loadTask()
})
</script>

<style scoped>
.mobile-task-detail {
  display: flex;
  flex-direction: column;
  background: var(--color-background);
  height: 100%;
  overflow: hidden;
}

.task-header {
  flex-shrink: 0;
  position: relative;
}

.header-banner {
  height: 60px;
  display: flex;
  align-items: center;
  padding: var(--spacing-md);
}

.back-btn {
  display: inline-flex;
  align-items: center;
  gap: var(--spacing-xs);
  padding: var(--spacing-xs) var(--spacing-md);
  border-radius: var(--radius-lg);
  background: rgba(255, 255, 255, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.3);
  cursor: pointer;
  color: white;
  font-size: var(--font-size-sm);
  font-weight: 600;
  transition: all var(--transition-fast);
  backdrop-filter: blur(8px);
}

.back-text {
  display: inline;
}

.back-btn:hover {
  background: rgba(255, 255, 255, 0.3);
}

.back-btn:active {
  transform: scale(0.98);
}

.title-section {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-sm);
  position: relative;
}

.task-title-input {
  flex: 1;
  font-size: var(--font-size-xl);
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0;
  padding: var(--spacing-sm);
  border: 2px solid var(--color-primary);
  border-radius: var(--radius-md);
  outline: none;
  line-height: 1.4;
}

.edit-title-btn {
  position: absolute;
  top: 0;
  right: 0;
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--color-text-muted);
  transition: all var(--transition-fast);
  flex-shrink: 0;
}

.edit-title-btn:hover {
  background: var(--color-primary-lighter);
  color: var(--color-primary);
}

.detail-section .task-title {
  flex: 1;
  font-size: var(--font-size-xl);
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0 0 var(--spacing-sm) 0;
  line-height: 1.4;
  padding-right: 40px;
}

.detail-section .task-title.done {
  text-decoration: line-through;
  color: var(--color-text-muted);
}

.project-badge svg {
  flex-shrink: 0;
}

.task-content {
  flex: 1;
  overflow-y: auto;
  padding-top: var(--spacing-lg);
}

.task-content.is-loading {
  opacity: 0.6;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-2xl);
  text-align: center;
}

.empty-state.error svg {
  color: var(--color-error);
  margin-bottom: var(--spacing-md);
}

.empty-title {
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--color-text-muted);
  margin: 0 0 var(--spacing-sm) 0;
}

.empty-subtitle {
  font-size: var(--font-size-base);
  color: var(--color-text-muted);
  margin: 0 0 var(--spacing-md) 0;
}

.retry-btn {
  display: inline-flex;
  align-items: center;
  gap: var(--spacing-xs);
  padding: var(--spacing-sm) var(--spacing-lg);
  background: var(--color-primary);
  color: white;
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.retry-btn:hover {
  background: var(--color-primary-dark);
  transform: scale(1.02);
}

.retry-btn:active {
  transform: scale(0.98);
}

.task-details {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.project-section {
  background: transparent;
  padding: 0 var(--spacing-md);
  margin-top: -8px;
}

.project-section .project-badge {
  display: inline-flex;
  align-items: center;
  gap: var(--spacing-xs);
  padding: var(--spacing-xs) var(--spacing-md);
  background: var(--color-surface);
  color: var(--color-primary);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: 600;
  box-shadow: var(--shadow-sm);
}

.detail-section {
  background: var(--color-surface);
  border-radius: var(--radius-md);
  padding: var(--spacing-md);
  box-shadow: var(--shadow-sm);
}

.detail-section:first-child {
  border-radius: var(--radius-xl) var(--radius-xl) var(--radius-md) var(--radius-md);
  margin-top: 0;
  position: relative;
  z-index: 2;
}

.section-title {
  font-size: var(--font-size-sm);
  font-weight: 700;
  color: var(--color-text-muted);
  margin: 0 0 var(--spacing-md) 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.project-info {
  display: flex;
  align-items: center;
}

.project-name {
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--color-primary);
}

.description {
  font-size: var(--font-size-base);
  line-height: 1.6;
  color: var(--color-text-secondary);
}

.attributes-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.attribute-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-sm) 0;
  border-bottom: 1px solid var(--color-border);
}

.attribute-item:last-child {
  border-bottom: none;
}

.attribute-label {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
  font-weight: 500;
}

.attribute-value {
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
  font-weight: 600;
}

.attribute-value.overdue {
  color: var(--color-error);
}

.priority-badge {
  display: inline-block;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: white;
}

.priority-badge.priority-0 { background: #9CA3AF; }
.priority-badge.priority-1 { background: #10B981; }
.priority-badge.priority-2 { background: #F59E0B; }
.priority-badge.priority-3 { background: #F97316; }
.priority-badge.priority-4 { background: #EF4444; }

.assignees-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.assignee-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.assignee-avatar {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-full);
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary-lighter);
  color: var(--color-primary);
  flex-shrink: 0;
}

.assignee-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.assignee-name {
  flex: 1;
  font-size: var(--font-size-base);
  font-weight: 500;
  color: var(--color-text-primary);
}

.labels-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-sm);
}

.label-badge {
  display: inline-block;
  padding: 4px 12px;
  border-radius: var(--radius-md);
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: white;
}

.attachments-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.attachment-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm);
  background: var(--color-background);
  border-radius: var(--radius-md);
}

.attachment-icon {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: var(--color-primary-lighter);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
  flex-shrink: 0;
}

.attachment-name {
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.comments-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.comment-item {
  background: var(--color-background);
  border-radius: var(--radius-md);
  padding: var(--spacing-sm);
}

.comment-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--spacing-sm);
}

.comment-author {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-text-primary);
}

.comment-time {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.comment-content {
  font-size: var(--font-size-sm);
  line-height: 1.6;
  color: var(--color-text-secondary);
}

.priority-select {
  padding: 6px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  background: var(--color-background);
  color: var(--color-text-primary);
  outline: none;
  cursor: pointer;
}

.priority-select:focus {
  border-color: var(--color-primary);
}

.percent-done-container {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.percent-done-slider {
  flex: 1;
  -webkit-appearance: none;
  appearance: none;
  width: 100%;
  height: 6px;
  border-radius: 3px;
  background: var(--color-border);
  outline: none;
}

.percent-done-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--color-primary);
  cursor: pointer;
}

.percent-done-slider::-moz-range-thumb {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--color-primary);
  cursor: pointer;
  border: none;
}

.percent-done-slider:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.percent-done-value {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-primary);
  min-width: 40px;
  text-align: right;
}

.no-comments {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
  text-align: center;
  padding: var(--spacing-md);
}

.add-comment-section {
  margin-top: var(--spacing-md);
  border-top: 1px solid var(--color-border);
  padding-top: var(--spacing-md);
}

.comment-input {
  width: 100%;
  padding: var(--spacing-sm) var(--spacing-md);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--font-size-base);
  font-family: inherit;
  resize: vertical;
  outline: none;
  margin-bottom: var(--spacing-sm);
}

.comment-input:focus {
  border-color: var(--color-primary);
}

.comment-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.submit-comment-btn {
  width: 100%;
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--color-primary);
  color: white;
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.submit-comment-btn:hover:not(:disabled) {
  background: var(--color-primary-dark);
}

.submit-comment-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
