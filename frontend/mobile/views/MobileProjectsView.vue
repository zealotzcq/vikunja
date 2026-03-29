<template>
	<div class="mobile-projects">
		<main
			class="projects-content"
			:class="{ 'is-loading': projectStore.isLoading }"
		>
			<div
				v-if="!hasProjects && !projectStore.isLoading"
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
				</div>
				<h3 class="empty-title">
					暂无项目
				</h3>
				<p class="empty-subtitle">
					创建您的第一个项目开始管理任务
				</p>
			</div>

			<div
				v-else
				class="projects-list"
			>
				<div 
					v-for="project in filteredProjects" 
					:key="project.id" 
					class="project-card card card-interactive"
					@click="openProject(project)"
				>
					<div 
						class="project-color" 
						:style="{ background: project.hexColor || 'var(--color-primary)' }"
					/>
					<div class="project-info">
						<h3 class="project-title">
							{{ project.title }}
						</h3>
						<div class="project-meta">
							<span
								v-if="project.isArchived"
								class="badge badge-warning"
							>已归档</span>
							<span
								v-if="project.parentProjectId !== 0"
								class="badge badge-secondary"
							>子项目</span>
						</div>
					</div>
					<svg
						class="project-arrow"
						width="20"
						height="20"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<polyline points="9 18 15 12 9 6" />
					</svg>
				</div>
			</div>
		</main>

		<div class="filter-tabs">
			<button 
				class="filter-tab"
				:class="{ active: activeFilter === 'active' }"
				@click="activeFilter = 'active'"
			>
				进行中
			</button>
			<button 
				class="filter-tab"
				:class="{ active: activeFilter === 'archived' }"
				@click="activeFilter = 'archived'"
			>
				已归档
			</button>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useProjectStore } from '@/stores/projects'
import { useCompanyStore } from '@/stores/company'
import type { IProject } from '@/modelTypes/IProject'

export default defineComponent({
	name: 'MobileProjectsView',
	setup() {
		const router = useRouter()
		const projectStore = useProjectStore()
		const companyStore = useCompanyStore()
		const activeFilter = ref<'active' | 'archived'>('active')

		function filterProjectsByCompany(projectList: readonly IProject[]): IProject[] {
			const currentCompanyId = companyStore.currentCompanyId

			if (!currentCompanyId) {
				return [...projectList]
			}

			const projectIdsToHide = new Set<number>()
			for (const map of companyStore.companyProjectMap) {
				if (map.company_id !== currentCompanyId) {
					for (const projectId of map.project_ids) {
						projectIdsToHide.add(projectId)
					}
				}
			}

			return projectList.filter(project => !projectIdsToHide.has(project.id))
		}

		const filteredProjects = computed(() => {
			const allProjects = projectStore.projectsArray
			const filteredByStatus = activeFilter.value === 'archived'
				? allProjects.filter(p => p.isArchived)
				: allProjects.filter(p => !p.isArchived)

			return filterProjectsByCompany(filteredByStatus)
		})

		const hasProjects = computed(() => filteredProjects.value.length > 0)

		const openProject = (project: IProject) => {
			router.push(`/mobile/project/${project.id}`)
		}

		const loadProjects = async () => {
			try {
				await projectStore.loadAllProjects()
			} catch (error) {
				console.error('Failed to load projects:', error)
			}
		}

		onMounted(async () => {
			await companyStore.loadCompanyProjectMap()
			loadProjects()
		})

		return {
			projectStore,
			companyStore,
			activeFilter,
			filteredProjects,
			hasProjects,
			openProject,
		}
	},
})
</script>

<style scoped>
.mobile-projects {
  display: flex;
  flex-direction: column;
  background: var(--color-background);
  height: 100%;
  overflow: hidden;
  min-height: 0;
  min-width: 0;
}

.projects-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-lg) var(--spacing-md) calc(48px + var(--spacing-lg));
  min-height: 0;
  min-width: 0;
}

.projects-content.is-loading {
  opacity: 0.6;
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

.projects-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.project-card {
  display: flex;
  align-items: center;
  padding: var(--spacing-sm) var(--spacing-md);
  gap: var(--spacing-md);
  position: relative;
  min-height: 52px;
}

.project-color {
  width: 4px;
  border-radius: 2px;
  height: 40px;
  flex-shrink: 0;
}

.project-info {
  flex: 1;
  min-width: 0;
}

.project-title {
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0 0 4px 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.project-meta {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  flex-wrap: wrap;
}

.project-arrow {
  flex-shrink: 0;
  color: var(--color-text-muted);
}

.filter-tabs {
  position: fixed;
  bottom: 64px;
  left: 0;
  right: 0;
  background: var(--color-surface);
  display: flex;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  border-top: 1px solid var(--color-border);
  box-shadow: 0 -2px 10px rgba(0, 0, 0, 0.05);
}

.filter-tab {
  flex: 1;
  padding: var(--spacing-sm) var(--spacing-md);
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-text-muted);
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-family: var(--font-family);
}

.filter-tab:hover {
  background: var(--color-primary-lighter);
  color: var(--color-primary);
}

.filter-tab.active {
  background: var(--color-primary);
  color: white;
  border-color: var(--color-primary);
}

@media (prefers-reduced-motion: reduce) {
  .project-arrow,
  .filter-tab {
    transition: none;
  }
}
</style>
