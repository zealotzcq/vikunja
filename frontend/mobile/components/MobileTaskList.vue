<template>
	<div class="mobile-task-list">
		<div
			v-if="!hasTasks && !isLoading"
			class="empty-state"
		>
			<div class="empty-icon">
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
					<path d="M9 11l3 3L22 4" />
					<path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
				</svg>
			</div>
			<h3 class="empty-title">
				暂无任务
			</h3>
			<p class="empty-subtitle">
				该项目还没有任务
			</p>
		</div>

		<div
			v-else
			class="table-view"
		>
			<div class="table-header">
				<div class="table-cell cell-check" />
				<div class="table-cell cell-content">
					<div v-if="mode === 'home'" class="header-cell col-assignee">
						人员
					</div>
					<div class="header-cell" :class="mode === 'home' ? 'col-priority' : 'col-priority-first'">
						优先
					</div>
					<div class="header-cell col-percent">
						进度
					</div>
					<div class="header-cell col-due">
						截至
					</div>
					<div v-if="mode === 'project'" class="header-cell col-start">
						开始
					</div>
				</div>
			</div>
			<transition-group name="task-list">
				<MobileTaskItem
					v-for="task in tasks"
					:key="task.id"
					:task="task"
					:mode="mode"
					@openTask="openTask"
					@toggleTaskDone="toggleTaskDone"
					@refresh="emit('refresh')"
				/>
			</transition-group>
		</div>
	</div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import type { ITask } from '@/modelTypes/ITask'
import MobileTaskItem from './MobileTaskItem.vue'

const props = defineProps<{
  tasks: ITask[];
  isLoading: boolean;
  mode: 'home' | 'project';
}>()

const emit = defineEmits<{
  (e: 'openTask', task: ITask): void;
  (e: 'toggleTaskDone', task: ITask): void;
  (e: 'refresh'): void;
}>()

const hasTasks = computed(() => props.tasks.length > 0)

const openTask = (task: ITask) => {
	emit('openTask', task)
}

const toggleTaskDone = (task: ITask) => {
	emit('toggleTaskDone', task)
}
</script>

<style scoped>
.mobile-task-list {
  display: flex;
  flex-direction: column;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-2xl);
  text-align: center;
  margin-top: 20%;
}

.empty-icon {
  width: 80px;
  height: 80px;
  border-radius: var(--radius-xl);
  background: var(--color-primary-lighter);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
  margin-bottom: var(--spacing-md);
}

.empty-title {
  font-size: var(--font-size-xl);
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0 0 var(--spacing-sm) 0;
}

.empty-subtitle {
  font-size: var(--font-size-base);
  color: var(--color-text-muted);
  margin: 0;
}

.table-view {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.table-header {
  display: flex;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--color-surface);
  border-radius: var(--radius-md);
  font-weight: 600;
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.table-cell {
  display: flex;
  align-items: center;
  overflow: hidden;
}

.cell-check {
  width: 32px;
  flex-shrink: 0;
  justify-content: center;
}

.cell-content {
  flex: 1;
  display: flex;
  gap: var(--spacing-xs);
}

.cell-content .header-cell {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--color-text-muted);
  text-align: center;
}

.col-assignee {
  flex: 1;
}

.col-priority {
  flex: 1;
}

.col-priority-first {
  flex: 1;
}

.col-percent {
  flex: 1;
}

.col-due {
  flex: 1;
}

.col-start {
  flex: 1;
}

.task-list-enter-active {
  transition: all 0.3s ease;
}

.task-list-leave-active {
  transition: all 0.3s ease;
  position: absolute;
  width: 100%;
}

.task-list-enter-from {
  opacity: 0;
  transform: translateY(-10px);
}

.task-list-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

.task-list-move {
  transition: transform 0.3s ease, opacity 0.3s ease;
}

@media (prefers-reduced-motion: reduce) {
  .task-list-enter-active,
  .task-list-leave-active,
  .task-list-move {
    transition: none;
  }
}
</style>
