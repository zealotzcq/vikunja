<template>
	<div
		class="table-row card"
		:class="{ 'task-done': localTask.done }"
	>
		<div class="table-cell cell-check">
			<button
				class="task-check-small"
				:class="{ 'checked': localTask.done }"
				@click.stop="toggleDone"
			>
				<svg
					class="check-circle"
					width="24"
					height="24"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<path
						class="check-bg"
						d="M22 11.08V12a10 10 0 1 1-5.93-9.14"
						:class="{ 'visible': localTask.done }"
					/>
					<polyline
						class="check-mark"
						points="22 4 12 14.01 9 11.01"
						:class="{ 'visible': localTask.done }"
					/>
					<circle
						class="check-circle-outline"
						cx="12"
						cy="12"
						r="10"
					/>
				</svg>
			</button>
		</div>
		<div class="table-cell cell-content">
			<div
				ref="titleCellRef"
				class="task-title"
				:class="{ 'done': localTask.done }"
				@click="openTask"
			>
				{{ localTask.title }}
			</div>
			<div
				ref="metaRowRef"
				class="task-meta"
			>
				<div
					v-if="mode === 'project'"
					class="meta-cell col-start"
				>
					{{ localTask.startDate ? formatDays(localTask.startDate) : '-' }}
				</div>
				<div
					v-if="mode === 'home'"
					class="meta-cell col-assignee"
				>
					<span
						v-if="localTask.assignees && localTask.assignees.length > 0 && localTask.assignees[0]?.username"
						class="assignee-name"
					>
						{{ localTask.assignees[0].name || localTask.assignees[0].username }}
					</span>
					<span
						v-else
						class="no-assignee"
					>-</span>
				</div>
				<div
					ref="priorityCellRef"
					class="meta-cell"
					:class="mode === 'home' ? 'col-priority' : 'col-priority-first'"
				>
					<span
						class="priority-text"
						:class="`priority-${localTask.priority}`"
					>
						{{ getPriorityText(localTask.priority) }}
					</span>
				</div>
				<div class="meta-cell col-percent">
					<div class="percent-bar">
						<div
							class="percent-fill"
							:style="{ width: localTask.percentDone + '%' }"
						/>
					</div>
					<span class="percent-text">{{ localTask.percentDone || 0 }}%</span>
				</div>
				<div
					class="meta-cell col-due"
					:class="{ 'overdue': isOverdue(localTask.dueDate) }"
				>
					<SimplePopup
						v-if="localTask.dueDate && !localTask.done"
						:open="showDeferPopup"
						:popup-style="popupStyle"
						@update:open="showDeferPopup = $event"
					>
						<template #trigger>
							<BaseButton
								class="due-date-btn"
								@click.prevent.stop="handleDueDateClick"
							>
								{{ formatDays(localTask.dueDate) }}
							</BaseButton>
						</template>
						<template #default>
							<MobileDeferTask
								v-model="localTask"
								@update:modelValue="deferTaskUpdate"
							/>
						</template>
					</SimplePopup>
					<span v-else>{{ localTask.dueDate ? formatDays(localTask.dueDate) : '-' }}</span>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import type { ITask } from '@/modelTypes/ITask'
import { useTaskStore } from '@/stores/tasks'
import SimplePopup from './SimplePopup.vue'
import MobileDeferTask from './MobileDeferTask.vue'
import BaseButton from '@/components/base/BaseButton.vue'

const props = defineProps<{
  task: ITask;
  mode: 'home' | 'project';
}>()

const emit = defineEmits<{
  (e: 'openTask', task: ITask): void;
  (e: 'toggleTaskDone', task: ITask): void;
  (e: 'refresh'): void;
}>()

const taskStore = useTaskStore()
const localTask = ref<ITask>({} as ITask)
const showDeferPopup = ref(false)
const popupStyle = ref({ top: '0px', left: '0px', width: '0px' })
const titleCellRef = ref<HTMLElement | null>(null)
const metaRowRef = ref<HTMLElement | null>(null)
const priorityCellRef = ref<HTMLElement | null>(null)

watch(
	() => props.task,
	(newTask) => {
		localTask.value = { ...newTask }
	},
	{
		immediate: true,
		deep: true,
	},
)

const openTask = () => {
	emit('openTask', localTask.value)
}

const deferTaskUpdate = (newTask: ITask) => {
	localTask.value = newTask
	showDeferPopup.value = false
	setTimeout(() => {
		emit('refresh')
	}, 500)
}

const toggleDone = async () => {
	const originalDone = localTask.value.done
	const newDoneState = !originalDone

	const updateFunc = async () => {
		const newTask = await taskStore.update({ ...localTask.value, done: newDoneState })
		localTask.value = newTask
		emit('toggleTaskDone', newTask)
	}

	if (newDoneState) {
		localTask.value.done = newDoneState
		setTimeout(updateFunc, 300)
	} else {
		localTask.value.done = newDoneState
		await updateFunc()
	}
}

const isOverdue = (dueDate: Date | string | null) => {
	if (!dueDate) return false
	return new Date(dueDate) < new Date()
}

const formatDays = (dateString: Date | string) => {
	const date = new Date(dateString)
	const now = new Date()
	const diff = date.getTime() - now.getTime()
	const days = Math.ceil(diff / (1000 * 60 * 60 * 24))

	if (days === 0) return '今天'
	if (days > 0) return `${days}天后`
	return `${Math.abs(days)}天前`
}

const getPriorityText = (priority: number) => {
	if (priority <= 1) return ''
	if (priority === 2) return '!'
	if (priority === 3) return '!!'
	if (priority === 4) return '!!!'
	return ''
}

const handleDueDateClick = () => {
	if (!localTask.value.dueDate || localTask.value.done) {
		return
	}

	console.log('[MobileTaskItem] Due date clicked, showing popup for task:', localTask.value.id)
	showDeferPopup.value = true
}

watch(showDeferPopup, async (newValue) => {
	console.log('[MobileTaskItem] showDeferPopup changed to:', newValue)
	if (newValue && priorityCellRef.value) {
		await nextTick()

		const priorityRect = priorityCellRef.value.getBoundingClientRect()
		const scrollY = window.scrollY || window.pageYOffset || 0
		const scrollX = window.scrollX || window.pageXOffset || 0

		const menuTop = priorityRect.top + scrollY
		const menuLeft = priorityRect.left + scrollX

		popupStyle.value = {
			top: `${menuTop}px`,
			left: `${menuLeft}px`,
		}
		console.log('[MobileTaskItem] Popup style set:', popupStyle.value)
	}
})
</script>

<style scoped>
.table-row {
  display: flex;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  align-items: center;
  position: relative;
  background: var(--color-surface);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  margin-bottom: var(--spacing-xs);
}

.table-cell {
  display: flex;
  overflow: hidden;
}

.cell-check {
  width: 32px;
  flex-shrink: 0;
  justify-content: center;
  padding-top: 4px;
}

.cell-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.task-title {
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--color-text-primary);
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 4px 8px;
  background: var(--color-background);
  border-radius: var(--radius-sm);
  text-align: center;
  box-shadow: var(--shadow-xs);
}

.task-title.done {
  text-decoration: line-through;
  color: var(--color-text-muted);
  transition: color 0.3s ease, text-decoration 0.3s ease;
}

.task-meta {
  display: flex;
  gap: var(--spacing-xs);
}

.meta-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.col-assignee {
  flex: 1;
}

.assignee-name {
  font-size: var(--font-size-xs);
  color: var(--color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.no-assignee {
  color: var(--color-text-muted);
  opacity: 0.5;
}

.col-priority {
  flex: 1;
}

.col-priority-first {
  flex: 1;
}

.col-percent {
  flex: 1;
  position: relative;
}

.percent-bar {
  width: 100%;
  height: 24px;
  background: #E5E7EB;
  border-radius: 12px;
  overflow: hidden;
}

.percent-fill {
  height: 100%;
  background: var(--color-primary);
  border-radius: 12px;
  transition: width 0.3s ease;
}

.percent-text {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: var(--font-size-xs);
  color: white;
  font-weight: 600;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
  z-index: 1;
}

.col-due {
  flex: 1;
}

.col-start {
  flex: 1;
}

.col-due.overdue {
  color: var(--color-error);
}

.due-date-btn {
  background: transparent;
  border: none;
  padding: 0;
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  min-width: auto;
}

.due-date-btn:hover {
  color: var(--color-primary);
}

.priority-text {
  font-size: var(--font-size-xs);
  font-weight: 700;
}

.priority-text.priority-0 { color: transparent; }
.priority-text.priority-1 { color: transparent; }
.priority-text.priority-2 { color: #F59E0B; }
.priority-text.priority-3 { color: #EF4444; }
.priority-text.priority-4 { color: #DC2626; }

.percent-bar {
  width: 40px;
  height: 6px;
  background: #E5E7EB;
  border-radius: 3px;
  overflow: hidden;
}

.percent-fill {
  height: 100%;
  background: var(--color-primary);
  border-radius: 3px;
}

.percent-text {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  font-weight: 500;
}

.task-check-small {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-full);
  background: transparent;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--color-text-muted);
  transition: color 0.2s ease, transform 0.2s ease;
}

.task-check-small:active {
  transform: scale(0.9);
}

.task-check-small.checked {
  color: var(--color-success);
}

.task-check-small svg {
  transition: all 0.3s ease;
}

.check-bg {
  opacity: 0;
  stroke-dasharray: 60;
  stroke-dashoffset: 60;
  transition: opacity 0.3s ease, stroke-dashoffset 0.3s ease;
}

.check-bg.visible {
  opacity: 1;
  stroke-dashoffset: 0;
}

.check-mark {
  opacity: 0;
  stroke-dasharray: 42;
  stroke-dashoffset: 42;
  transition: opacity 0.3s ease, stroke-dashoffset 0.3s ease 0.1s;
}

.check-mark.visible {
  opacity: 1;
  stroke-dashoffset: 0;
}

.table-row.task-done {
  background-color: var(--color-background-hover);
  transition: background-color 0.3s ease;
}
</style>
