<template>
	<slot
		name="trigger"
		:toggle="toggle"
		:open="isOpen"
	/>
	<teleport to="body">
		<div
			v-if="isOpen"
			ref="menuRef"
			class="simple-popup"
			:style="mergedStyle"
		>
			<slot />
		</div>
	</teleport>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { onClickOutside } from '@vueuse/core'

const props = defineProps<{
  open: boolean
  popupStyle?: {
  	top: string
  	left: string
  	width?: string
  }
  clickPosition?: {
  	x: number
  	y: number
  }
}>()

const emit = defineEmits<{
	'update:open': [open: boolean]
}>()

defineSlots<{
	trigger(props: {
		toggle: () => void,
		open: boolean
	}): void
	default(): void
}>()

const isOpen = ref(false)
const menuRef = ref<HTMLElement | null>(null)

watch(
	() => props.open,
	(newValue) => {
		isOpen.value = newValue
	},
	{
		immediate: true,
	},
)

function toggle() {
	isOpen.value = !isOpen.value
	emit('update:open', isOpen.value)
}

const mergedStyle = computed(() => {
	const style = props.popupStyle

	return {
  	position: 'fixed' as const,
  	zIndex: 100 as number,
  	top: style?.top || '0px',
  	left: style?.left || '0px',
  	width: style?.width || 'auto',
	}
})

onClickOutside(menuRef, () => {
	if (isOpen.value) {
		isOpen.value = false
		emit('update:open', false)
	}
})
</script>

<style scoped>
.simple-popup {
	position: fixed;
	z-index: 100;
	background: var(--color-surface);
	border-radius: var(--radius-md);
	box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}
</style>

