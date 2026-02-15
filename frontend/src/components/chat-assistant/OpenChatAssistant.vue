<template>
	<BaseButton
		class="chat-assistant-trigger d-print-none"
		:title="$t('chatAssistant.title')"
		@click="chatStore.toggleOpen"
	>
		<span class="is-sr-only">{{ $t('chatAssistant.title') }}</span>
		<Icon icon="robot" />
	</BaseButton>
</template>

<script lang="ts" setup>
import BaseButton from '@/components/base/BaseButton.vue'
import Icon from '@/components/misc/Icon'

import {useChatStore} from '@/stores/chat'

const chatStore = useChatStore()
</script>

<style lang="scss" scoped>
.chat-assistant-trigger {
	position: fixed;
	inset-block-end: calc(1rem + 3rem);
	inset-inline-end: 1rem;
	z-index: 4500;
	inline-size: 3rem;
	block-size: 3rem;
	display: flex;
	justify-content: center;
	align-items: center;
	font-size: 1.5rem;
	color: var(--grey-500);
	background: var(--white);
	border: 1px solid var(--grey-200);
	border-radius: 50%;
	box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
	transition: all 0.2s;

	&:hover,
	&:focus {
		color: var(--grey-700);
		background: var(--grey-50);
		transform: scale(1.05);
	}
}

.dark .chat-assistant-trigger {
	color: var(--grey-400);
	background: var(--grey-800);
	border-color: var(--grey-700);

	&:hover,
	&:focus {
		color: var(--grey-200);
		background: var(--grey-700);
	}
}
</style>
