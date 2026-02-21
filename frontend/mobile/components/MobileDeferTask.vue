<template>
	<div class="mobile-defer-task">
		<div
			class="defer-btn"
			@click.stop="deferDays(3)"
		>
			+3天
		</div>
		<div
			class="defer-btn"
			@click.stop="deferDays(7)"
		>
			+7天
		</div>
		<div
			class="defer-btn"
			@click.stop="deferDays(30)"
		>
			+30天
		</div>
	</div>
</template>

<script setup lang="ts">
import { shallowReactive, onMounted, ref } from 'vue'
import TaskService from '@/services/task'
import type { ITask } from '@/modelTypes/ITask'

const props = defineProps<{
  modelValue: ITask
}>()

const emit = defineEmits<{
  'update:modelValue': [value: ITask]
}>()

const taskService = shallowReactive(new TaskService())
const isInitialized = ref(false)

onMounted(() => {
	setTimeout(() => {
		isInitialized.value = true
	}, 100)
})

async function deferDays(days: number) {
	console.log('deferDays called with:', days)

	if (!isInitialized.value) {
		console.log('Not initialized yet, ignoring click')
		return
	}

	if (!props.modelValue.dueDate) return

	const newDueDate = new Date(props.modelValue.dueDate)
	newDueDate.setDate(newDueDate.getDate() + days)

	try {
		const newTask = await taskService.update({
			...props.modelValue,
			dueDate: newDueDate,
		})
		emit('update:modelValue', newTask)
	} catch (error) {
		console.error('Error deferring task:', error)
	}
}
</script>

<style scoped>
.mobile-defer-task {
	display: flex;
	flex-direction: row;
	background: var(--color-surface);
	border-radius: var(--radius-md);
	box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
	width: 100%;
	z-index: 9999;
}

.defer-btn {
	flex: 1;
	padding: 10px 8px;
	font-size: var(--font-size-sm);
	color: var(--color-text-primary);
	transition: background-color 0.2s ease;
	cursor: pointer;
	text-align: center;
	border-right: 1px solid var(--border-light);
	min-width: 60px;
	white-space: nowrap;
	position: relative;
}

.defer-btn:last-of-type {
	border-right: none;
}

.defer-btn::before {
	content: '';
	position: absolute;
	top: 50%;
	left: 50%;
	transform: translate(-50%, -50%);
	width: calc(100% - 16px);
	height: calc(100% - 16px);
	border: 2px solid var(--color-primary-lighter);
	border-radius: var(--radius-sm);
	opacity: 0;
	transition: opacity 0.2s ease;
}

.defer-btn:hover::before {
	opacity: 1;
}

.defer-btn:active {
	color: var(--color-primary);
}

.defer-btn:active::before {
	border-color: var(--color-primary);
}
</style>

