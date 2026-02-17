<template>
	<div class="content has-text-centered">
		<h2 v-if="salutation">
			{{ salutation }}
		</h2>

		<Message
			v-if="deletionScheduledAt !== null"
			variant="danger"
			class="mbe-4"
		>
			{{
				$t('user.deletion.scheduled', {
					date: formatDisplayDate(deletionScheduledAt),
					dateSince: formatDateSince(deletionScheduledAt),
				})
			}}
			<RouterLink :to="{name: 'user.settings.deletion'}">
				{{ $t('user.deletion.scheduledCancel') }}
			</RouterLink>
		</Message>
		<AddTask
			class="is-max-width-desktop"
			@taskAdded="updateTaskKey"
		/>
		<ImportHint v-if="tasksLoaded" />
		<div
			v-if="projectHistory.length > 0"
			class="is-max-width-desktop has-text-start mbs-4"
		>
			<h3>{{ $t('home.lastViewed') }}</h3>
			<ProjectCardGrid
				v-cy="'projectCardGrid'"
				:projects="projectHistory"
				:show-even-number-of-projects="true"
			/>
		</div>
		<ShowTasks
			v-if="projectStore.hasProjects"
			:key="showTasksKey"
			:label-ids="labelIds"
			class="show-tasks"
			@tasksLoaded="tasksLoaded = true"
			@clearLabelFilter="handleClearLabelFilter"
		/>
	</div>
</template>

<script lang="ts" setup>
import {ref, computed, watch} from 'vue'
import {useRoute, useRouter} from 'vue-router'

import Message from '@/components/misc/Message.vue'
import ShowTasks from '@/views/tasks/ShowTasks.vue'
import ProjectCardGrid from '@/components/project/partials/ProjectCardGrid.vue'
import AddTask from '@/components/tasks/AddTask.vue'
import ImportHint from '@/components/home/ImportHint.vue'

import {getHistory} from '@/modules/projectHistory'
import {parseDateOrNull} from '@/helpers/parseDateOrNull'
import {formatDateSince, formatDisplayDate} from '@/helpers/time/formatDate'
import {useDaytimeSalutation} from '@/composables/useDaytimeSalutation'

import {useProjectStore} from '@/stores/projects'
import {useAuthStore} from '@/stores/auth'
import {useChatStore} from '@/stores/chat'
import type {IProject} from '@/modelTypes/IProject'

const salutation = useDaytimeSalutation()

const authStore = useAuthStore()
const projectStore = useProjectStore()
const chatStore = useChatStore()
const route = useRoute()
const router = useRouter()

const projectHistory = computed<IProject[]>(() => {
	// If we don't check this, it tries to load the project background right after logging out	
	if(!authStore.authenticated) {
		return []
	}
	
	const history: IProject[] = []
	getHistory().forEach(l => {
		const project = projectStore.projects[l.id]
		if (project) {
			history.push(project as IProject)
		}
	})
	return history
})

const tasksLoaded = ref(false)

const deletionScheduledAt = computed(() => {
	const date = authStore.info?.deletionScheduledAt
	return date ? parseDateOrNull(date) : null
})

// Extract label IDs from query parameter
const labelIds = computed((): string[] | undefined => {
	const labelsParam = route.query.labels
	if (!labelsParam) {
		return undefined
	}
	const labels = Array.isArray(labelsParam) ? labelsParam : [labelsParam]
	return labels.filter((l): l is string => typeof l === 'string')
})

// This is to reload the tasks list after adding a new task through the global task add.
// FIXME: Should use pinia (somehow?)
const showTasksKey = ref(0)

function updateTaskKey() {
	showTasksKey.value++
}

// Watch for chat store refresh trigger to reload task list
watch(() => chatStore.homeRefreshTrigger, () => {
	updateTaskKey()
})

function handleClearLabelFilter() {
	const query = {...route.query}
	delete query.labels
	router.push({
		name: route.name as string,
		query,
	})
}
</script>

<style scoped lang="scss">
.show-tasks {
	margin-block-start: 2rem;
}
</style>
