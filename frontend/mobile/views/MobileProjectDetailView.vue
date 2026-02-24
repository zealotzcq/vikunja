<template>
	<div class="mobile-project-detail">
		<div class="project-header">
			<div
				class="header-banner"
				:style="{ background: project?.hexColor || 'var(--color-primary)' }"
			>
				<h1 class="project-name">
					{{ project?.title || '项目' }}
				</h1>
			</div>
		</div>

		<main
			class="tasks-content"
			:class="{ 'is-loading': isLoading }"
		>
			<MobileTaskList
				:tasks="tasksArray"
				:is-loading="isLoading"
				mode="project"
				@openTask="openTask"
				@toggleTaskDone="toggleTaskDone"
				@refresh="loadProjectTasks"
			/>
		</main>
	</div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useProjectStore } from '@/stores/projects'
import { useTaskStore } from '@/stores/tasks'
import type { ITask } from '@/modelTypes/ITask'
import { useProjectTaskRefresh } from '@/composables/useProjectTaskRefresh'
import MobileTaskList from '../components/MobileTaskList.vue'

export default defineComponent({
	name: 'MobileProjectDetailView',
	components: {
		MobileTaskList,
	},
	setup() {
		const router = useRouter()
		const route = useRoute()
		const projectStore = useProjectStore()
		const taskStore = useTaskStore()
		const isLoading = ref(false)
    
		const projectId = computed(() => Number(route.params.projectId))
		const project = computed(() => projectStore.projects[projectId.value])
		const tasksArray = computed(() => {
			const tasks = Object.values(taskStore.tasks)
			return tasks.sort((a, b) => {
				if (a.done !== b.done) {
					return a.done ? 1 : -1
				}
				const dateA = a.dueDate ? new Date(a.dueDate).getTime() : 0
				const dateB = b.dueDate ? new Date(b.dueDate).getTime() : 0
				return dateB - dateA
			})
		})

		const loadedProjectId = ref(0)

		const { checkAndRefreshTasks } = useProjectTaskRefresh(computed(() => ({
			id: projectId.value,
			title: project.value?.title || '',
		})))

		watch(
			() => loadedProjectId.value,
			async (loadedId) => {
				if (loadedId !== 0 && loadedId === projectId.value && loadedId !== lastCheckedProjectId.value) {
					lastCheckedProjectId.value = loadedId
					try {
						const didRefresh = await checkAndRefreshTasks()
						if (didRefresh) {
							await loadProjectTasks()
						}
					} catch (error) {
						console.error('[mobile] Error refreshing tasks:', error)
					}
				}
			},
			{ immediate: true },
		)

		const openTask = (task: ITask) => {
			if (!task || !task.title) {
				console.error('Invalid task:', task)
				return
			}
			router.push(`/mobile/task/${task.id}`)
		}
    
		const toggleTaskDone = async (task: ITask) => {
			try {
				await taskStore.update(task)
			} catch (error) {
				console.error('Failed to toggle task:', error)
			}
		}
    
		const loadProjectTasks = async () => {
			try {
				isLoading.value = true
				await projectStore.loadProject(projectId.value)
				await taskStore.loadTasks({}, projectId.value)
				loadedProjectId.value = projectId.value
			} catch (error) {
				console.error('Failed to load project tasks:', error)
			} finally {
				isLoading.value = false
			}
		}

		const lastCheckedProjectId = ref(0)

		onMounted(() => {
			loadProjectTasks()
		})
    
		return {
			project,
			tasksArray,
			isLoading,
			openTask,
			toggleTaskDone,
			loadProjectTasks,
		}
	},
})
</script>

<style scoped>
.mobile-project-detail {
  display: flex;
  flex-direction: column;
  background: var(--color-background);
  height: 100%;
  overflow: hidden;
  min-height: 0;
  min-width: 0;
}

.project-header {
  position: relative;
  flex-shrink: 0;
}

.header-banner {
  height: 80px;
  background: var(--color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-md);
}

.project-name {
  font-size: var(--font-size-xl);
  font-weight: 700;
  color: white;
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

.tasks-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-md);
  padding-bottom: var(--spacing-md);
  min-height: 0;
  min-width: 0;
}

.tasks-content.is-loading {
  opacity: 0.6;
}

@media (prefers-reduced-motion: reduce) {
  .back-btn {
    transition: none;
  }
}
</style>

