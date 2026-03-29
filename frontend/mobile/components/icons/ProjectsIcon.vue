<template>
	<svg
		:width="size"
		:height="size"
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
	>
		<rect
			x="2"
			y="4"
			width="20"
			height="16"
			rx="2"
		/>
		<path d="M7 15h0M2 9.5h20" />
	</svg>
</template>

<script lang="ts">
import { defineComponent } from 'vue'

export default defineComponent({
	name: 'ProjectsIcon',
	props: {
		size: {
			type: Number,
			default: 24,
		},
	},
})
</script>
