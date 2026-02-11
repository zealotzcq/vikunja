<template>
	<Teleport to="body">
		<transition name="chat-slide">
			<aside
				v-if="chatStore.isOpen && chatStore.isAvailable"
				class="chat-assistant-panel"
			>
				<header class="chat-header">
					<h3>{{ $t('chatAssistant.title') }}</h3>
					<div class="header-actions">
						<BaseButton
							class="action-btn clear-btn"
							:title="$t('chatAssistant.clear')"
							@click="clearMessages"
						>
							<Icon icon="trash-alt" />
						</BaseButton>
						<BaseButton
							class="action-btn close-btn"
							:title="$t('chatAssistant.close')"
							@click="chatStore.toggleOpen"
						>
							<Icon icon="times" />
						</BaseButton>
					</div>
				</header>
				<div ref="messagesContainer" class="chat-messages">
					<div
						v-for="msg in chatStore.messages"
						:key="msg.id"
						:class="['message', msg.role]"
					>
						<div class="message-content">{{ msg.content }}</div>
						<span class="message-time">{{ formatTime(msg.timestamp) }}</span>
					</div>
				</div>
				<div class="chat-input">
					<div v-if="chatStore.error" class="error-message">
						{{ chatStore.error }}
					</div>
					<input
						v-model="userInput"
						class="input"
						:placeholder="$t('chatAssistant.placeholder')"
						@keyup.enter="sendMessage"
					>
					<BaseButton
						class="send-btn"
						:title="$t('chatAssistant.send')"
						@click="sendMessage"
					>
						<Icon icon="arrow-up-from-bracket" />
					</BaseButton>
				</div>
			</aside>
		</transition>
	</Teleport>
</template>

<script lang="ts" setup>
  import {ref, watch, nextTick, onMounted} from 'vue'
  import {useI18n} from 'vue-i18n'
  import {useRouter} from 'vue-router'

  import BaseButton from '@/components/base/BaseButton.vue'
  import Icon from '@/components/misc/Icon'

  import {useChatStore} from '@/stores/chat'

  const {t} = useI18n({useScope: 'global'})
  const chatStore = useChatStore()
  const router = useRouter()

  const userInput = ref('')
  const messagesContainer = ref<HTMLElement | null>(null)

  onMounted(() => {
	chatStore.loadSession()
	detectMobile()
	scrollToBottom()
  })

  function detectMobile() {
	const isMobileDevice = typeof window !== 'undefined' && window.innerWidth < 768
	chatStore.setMobile(isMobileDevice)
  }

  function sendMessage() {
	if (userInput.value.trim() === '') {
		return
	}
	chatStore.sendMessage(userInput.value, router)
	userInput.value = ''
  }

  function clearMessages() {
	chatStore.clearMessages()
  }

  function scrollToBottom() {
	nextTick(() => {
		if (messagesContainer.value) {
			messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
		}
	})
  }

  function formatTime(timestamp: number): string {
	const date = new Date(timestamp)
	const hours = date.getHours().toString().padStart(2, '0')
	const minutes = date.getMinutes().toString().padStart(2, '0')
	return `${hours}:${minutes}`
  }

  watch(
	() => chatStore.messages.length,
	() => {
		scrollToBottom()
	},
  )

  watch(
	() => chatStore.isOpen,
	() => {
		scrollToBottom()
	},
  )
</script>

<style lang="scss" scoped>
.chat-assistant-panel {
	position: fixed;
	background: var(--white);
	display: flex;
	flex-direction: column;
	border-inline-start: 1px solid var(--grey-200);
	box-shadow: -4px 0 12px rgba(0, 0, 0, 0.1);
	z-index: 3000;

	@media screen and (max-width: $tablet) {
		inset: 0 0 0 0;
		height: 100dvh;
		border-radius: 0;
		border-inline-start: none;
		border-top: none;
		transform: translateY(0);
		transition: transform 0.3s ease;
	}

	@media screen and (min-width: $tablet) {
		inset: $navbar-height 0 0 auto;
		inline-size: 400px;
		max-inline-size: 400px;
		transform: translateX(0);
		transition: transform 0.3s ease;
	}
}

.chat-slide-enter-from,
.chat-slide-leave-to {
	@media screen and (max-width: $tablet) {
		transform: translateY(100%);
	}

	@media screen and (min-width: $tablet) {
		transform: translateX(100%);
	}
}

.chat-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 0.5rem 1rem;
	border-block-end: 1px solid var(--grey-200);

	h3 {
		margin: 0;
		font-size: 0.95rem;
		font-weight: 600;
		color: var(--grey-800);
	}
}

.header-actions {
	display: flex;
	gap: 0.5rem;
}

.action-btn {
	font-size: 1rem;
	color: var(--grey-500);
	background: transparent;
	border: none;
	padding: 0.25rem 0.5rem;
	transition: color 0.2s;

	&:hover,
	&:focus {
		color: var(--grey-700);
	}
}

.close-btn {
	font-size: 1.2rem;
}

.chat-messages {
	flex: 1;
	overflow-y: auto;
	padding: 1rem;
	display: flex;
	flex-direction: column;
	gap: 0.75rem;
	background: var(--site-background);
}

.message {
	display: flex;
	flex-direction: column;
	gap: 0.25rem;
	max-inline-size: 85%;

	&.user {
		align-self: flex-end;
		align-items: flex-end;
	}

	&.assistant {
		align-self: flex-start;
		align-items: flex-start;
	}
}

.message-content {
	padding: 0.75rem 1rem;
	border-radius: 1rem;
	font-size: 0.95rem;
	line-height: 1.5;
	word-wrap: break-word;
	white-space: pre-wrap;

	.user & {
		background: var(--primary);
		color: var(--white);
		border-end-end-radius: 0.25rem;
	}

	.assistant & {
		background: var(--grey-100);
		color: var(--grey-800);
		border-start-start-radius: 0.25rem;
	}
}

.message-time {
	font-size: 0.75rem;
	color: var(--grey-400);
	margin-inline-start: 0.5rem;

	.user & {
		margin-inline-end: 0.5rem;
	}

	.assistant & {
		margin-inline-start: 1rem;
	}
}

.chat-input {
	display: flex;
	gap: 0.5rem;
	padding: 1rem;
	border-block-start: 1px solid var(--grey-200);
	background: var(--white);
	flex-shrink: 0;
}

.error-message {
	inline-size: 100%;
	padding: 0.5rem;
	background: #fee;
	border: 1px solid #fcc;
	border-radius: 0.375rem;
	color: #c33;
	font-size: 0.875rem;
	margin-block-end: 0.5rem;
}

.input {
	flex: 1;
	border: 1px solid var(--grey-200);
	border-radius: 0.5rem;
	padding: 0.625rem 0.875rem;
	font-size: 0.95rem;
	transition: border-color 0.2s;

	&:focus {
		outline: none;
		border-color: var(--primary);
	}
}

.send-btn {
	inline-size: 2.75rem;
	block-size: 2.75rem;
	display: flex;
	align-items: center;
	justify-content: center;
	background: var(--primary);
	color: var(--white);
	border: none;
	border-radius: 0.5rem;
	font-size: 1.1rem;
	transition: background-color 0.2s;
	flex-shrink: 0;

	&:hover,
	&:focus {
		background: var(--primary-dark);
	}
}

.dark .chat-assistant-panel {
	background: var(--grey-900);
	border-inline-start-color: var(--grey-700);
	border-top-color: var(--grey-700);

	.chat-header {
		border-block-end-color: var(--grey-700);

		h3 {
			color: var(--grey-100);
		}
	}

	.chat-messages {
		background: var(--grey-800);
	}

	.message-content {
		.assistant & {
			background: var(--grey-700);
			color: var(--grey-100);
		}
	}

	.chat-input {
		border-block-top-color: var(--grey-700);

		.input {
			background: var(--grey-800);
			border-color: var(--grey-700);
			color: var(--grey-100);

			&:focus {
				border-color: var(--primary);
			}
		}
	}
}
</style>
