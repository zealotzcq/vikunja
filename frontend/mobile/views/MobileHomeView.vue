<template>
	<div class="mobile-home">
		<main class="home-content">
			<section class="stats-section">
				<div class="stats-grid">
					<div
						class="stat-card stat-pending"
						:class="{ 'active': activeCategory === 'inProgress' }"
						@click="setCategory('inProgress')"
					>
						<div class="stat-value">
							{{ stats.inProgress }}
						</div>
						<div class="stat-label">
							进行中
						</div>
					</div>
					<div
						class="stat-card stat-overdue"
						:class="{ 'active': activeCategory === 'overdue' }"
						@click="setCategory('overdue')"
					>
						<div class="stat-value">
							{{ stats.overdue }}
						</div>
						<div class="stat-label">
							过期
						</div>
					</div>
					<div
						class="stat-card stat-recent"
						:class="{ 'active': activeCategory === 'recent' }"
						@click="setCategory('recent')"
					>
						<div class="stat-value">
							{{ stats.recent }}
						</div>
						<div class="stat-label">
							最近完成
						</div>
					</div>
					<div
						class="stat-card stat-completed"
						:class="{ 'active': activeCategory === 'completed' }"
						@click="setCategory('completed')"
					>
						<div class="stat-value">
							{{ stats.completed }}
						</div>
						<div class="stat-label">
							所有任务
						</div>
					</div>
				</div>
			</section>

			<section class="tasks-section">
				<MobileTaskList
					:tasks="filteredTasks"
					:is-loading="loading"
					@openTask="goToTask"
					@toggleTaskDone="toggleTaskDone"
					@refresh="loadTasks"
				/>
			</section>
		</main>
	</div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useProjectStore } from '@/stores/projects'
import { useTaskStore } from '@/stores/tasks'
import MobileTaskList from '../components/MobileTaskList.vue'
import type { ITask } from '@/modelTypes/ITask'

const router = useRouter()
const projectStore = useProjectStore()
const taskStore = useTaskStore()

const tasks = computed<ITask[]>(() => {
	return Object.values(taskStore.tasks)
})

const loading = computed(() => taskStore.isLoading)
const activeCategory = ref<'inProgress' | 'overdue' | 'recent' | 'completed'>('inProgress')

const now = new Date()
const threeDaysAgo = new Date(now.getTime() - 3 * 24 * 60 * 60 * 1000)

const stats = computed(() => {
	return {
		inProgress: tasks.value.filter(t => !t.done && (!t.dueDate || new Date(t.dueDate) >= now)).length,
		overdue: tasks.value.filter(t => !t.done && t.dueDate && new Date(t.dueDate) < now).length,
		recent: tasks.value.filter(t => t.done && t.doneAt && new Date(t.doneAt) >= threeDaysAgo).length,
		completed: tasks.value.length,
	}
})

const filteredTasks = computed(() => {
	let result: ITask[] = []

	switch (activeCategory.value) {
		case 'inProgress':
			result = tasks.value.filter(t => !t.done && (!t.dueDate || new Date(t.dueDate) >= now))
			break
		case 'overdue':
			result = tasks.value.filter(t => !t.done && t.dueDate && new Date(t.dueDate) < now)
			break
		case 'recent':
			result = tasks.value.filter(t => t.done && t.doneAt && new Date(t.doneAt) >= threeDaysAgo)
			break
		case 'completed':
			result = tasks.value
			break
	}

	const sortedResult = result.sort((a, b) => {
		if (activeCategory.value === 'inProgress') {
			const dueDateA = a.dueDate ? new Date(a.dueDate).getTime() : Infinity
			const dueDateB = b.dueDate ? new Date(b.dueDate).getTime() : Infinity

			if (dueDateA === Infinity && dueDateB === Infinity) {
				return 0
			}
			if (dueDateA === Infinity) {
				return 1
			}
			if (dueDateB === Infinity) {
				return -1
			}
			return dueDateA - dueDateB
		} else {
			const updateA = new Date(a.updated).getTime()
			const updateB = new Date(b.updated).getTime()
			return updateB - updateA
		}
	})

	return sortedResult.map(task => ({
		...task,
		projectTitle: projectStore.projects[task.projectId]?.title || '',
	}))
})

const toggleTaskDone = async (task: ITask) => {
	await taskStore.update(task)
}

const loadTasks = async () => {
	try {
		await taskStore.loadTasks({
			filter: '',
			filter_include_nulls: false,
			s: '',
			sort_by: ['id'],
			order_by: ['desc'],
		}, null)
	} catch (error) {
		console.error('Failed to load tasks:', error)
	}
}

const setCategory = (category: 'inProgress' | 'overdue' | 'recent' | 'completed') => {
	activeCategory.value = category
	loadTasks()
}

const goToTask = (task: ITask) => {
	if (!task || !task.title) {
		console.error('Invalid task:', task)
		return
	}
	router.push(`/mobile/task/${task.id}`)
}

onMounted(() => {
	loadTasks()
})
</script>

<style scoped>
.mobile-home {
  display: flex;
  flex-direction: column;
  background: var(--color-background);
  height: 100%;
  overflow: hidden;
}

.home-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-lg) var(--spacing-md);
}

.stats-section {
  margin-bottom: var(--spacing-xl);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--spacing-md);
}

.stat-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-md);
  border-radius: var(--radius-xl);
  background: var(--color-surface);
  border: 2px solid transparent;
  cursor: pointer;
  transition: all var(--transition-normal);
  min-width: 70px;
  min-height: 80px;
  box-shadow: var(--shadow-sm);
  position: relative;
  overflow: hidden;
}

.stat-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
  border-radius: var(--radius-xl) var(--radius-xl) 0 0;
  transition: all var(--transition-normal);
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.stat-card:active {
  transform: translateY(0);
}

.stat-card.active {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.stat-pending::before {
  background: #3B82F6;
}

.stat-pending.active {
  border-color: #3B82F6;
  background: #EFF6FF;
}

.stat-overdue::before {
  background: #EF4444;
}

.stat-overdue.active {
  border-color: #EF4444;
  background: #FEF2F2;
}

.stat-recent::before {
  background: #10B981;
}

.stat-recent.active {
  border-color: #10B981;
  background: #ECFDF5;
}

.stat-completed::before {
  background: #6B7280;
}

.stat-completed.active {
  border-color: #6B7280;
  background: #F9FAFB;
}

.stat-value {
  font-size: var(--font-size-2xl);
  font-weight: 800;
  color: var(--color-text-primary);
  margin: 0;
  line-height: 1;
  margin-bottom: var(--spacing-xs);
}

.stat-label {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  font-weight: 600;
  margin: 0;
  text-align: center;
  letter-spacing: 0.02em;
}

.tasks-section {
  margin-bottom: var(--spacing-xl);
}

.loading-container {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: var(--spacing-xl);
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .stat-card,
  .loading-spinner {
    transition: none;
  }

  .loading-spinner {
    animation: none;
  }
}
</style>
