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
				<div
					ref="messagesContainer"
					class="chat-messages"
				>
					<div
						v-for="msg in visibleMessages"
						:key="msg.id"
						class="message"
						:class="[msg.role]"
					>
						<div class="message-content">
							{{ msg.content }}
						</div>

						<div
							v-if="msg.buttonNavigation"
							class="button-navigation"
						>
							<BaseButton
								class="nav-button"
								@click="chatStore.executeButtonNavigation(msg.buttonNavigation.routeName, msg.buttonNavigation.params)"
							>
								{{ msg.buttonNavigation.label }}
								<span
									v-if="msg.buttonNavigation.title"
									class="nav-title"
								>
									: {{ msg.buttonNavigation.title }}
								</span>
							</BaseButton>
						</div>

						<div
							v-if="msg.questionData && msg.toolCallID === lastMessageWithQuestion?.toolCallID"
							class="question-panel"
						>
							<div
								v-for="(question, qIndex) in parsedQuestions"
								:key="qIndex"
								class="question-item"
							>
								<div class="question-text">
									{{ question.question }}
								</div>
								<div class="question-options">
									<div
										v-for="(option, oIndex) in question.options"
										:key="oIndex"
										class="question-option"
										@click="handleQuestionOption(question, option)"
									>
										<div class="option-label">
											{{ option.label }}
										</div>
										<div class="option-description">
											{{ option.description }}
										</div>
									</div>
								</div>
							</div>
						</div>

						<span class="message-time">{{ formatTime(msg.timestamp) }}</span>
					</div>

					<div
						v-if="isProcessing"
						class="message assistant loading-message"
					>
						<div class="message-content loading-content">
							<div class="loading-dots">
								<span class="dot" />
								<span class="dot" />
								<span class="dot" />
							</div>
							<div class="loading-text">
								{{ processingText }}
							</div>
						</div>
					</div>
				</div>
				<div class="chat-input">
					<div
						v-if="chatStore.error"
						class="error-message"
					>
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
import {ref, watch, nextTick, onMounted, onUnmounted, computed} from 'vue'
import {useI18n} from 'vue-i18n'

import BaseButton from '@/components/base/BaseButton.vue'
import Icon from '@/components/misc/Icon'

import {useChatStore} from '@/stores/chat'
import type {IQuestion, IQuestionOption} from '@/modelTypes/IChatMessage'

const {t} = useI18n({useScope: 'global'})
const chatStore = useChatStore()

const userInput = ref('')
const messagesContainer = ref<HTMLElement | null>(null)
const isProcessing = ref(false)
const processingText = ref('')

const ANSWERED_TOOL_CALLS_KEY = 'vikunja-answered-tool-calls'

function getAnsweredToolCalls(): Set<string> {
	const stored = localStorage.getItem(ANSWERED_TOOL_CALLS_KEY)
	if (!stored) return new Set()
	try {
		return new Set(JSON.parse(stored))
	} catch {
		return new Set()
	}
}

function setToolCallAnswered(toolCallId: string) {
	const set = getAnsweredToolCalls()
	set.add(toolCallId)
	localStorage.setItem(ANSWERED_TOOL_CALLS_KEY, JSON.stringify([...set]))
}

watch(
	() => chatStore.hasPendingResponse,
	(pending) => {
		if (!pending) {
			isProcessing.value = false
			processingText.value = ''
		}
	},
)

const lastMessageWithQuestion = computed(() => {
	const answeredToolCalls = getAnsweredToolCalls()
	const messagesWithQuestions = chatStore.messages.filter(msg => 
		msg.questionData && 
		msg.toolCallID && 
		!answeredToolCalls.has(msg.toolCallID),
	)
	return messagesWithQuestions[messagesWithQuestions.length - 1] || undefined
})

const visibleMessages = computed(() => {
	return chatStore.messages.filter(msg => 
		msg.type === 'user_input' || 
		msg.type === 'assistant_response' || 
		msg.type === 'question' ||
		msg.type === 'button_navigation',
	)
})

const parsedQuestions = computed<IQuestion[]>(() => {
	if (!lastMessageWithQuestion.value?.questionData) {
		return []
	}
	try {
		return JSON.parse(lastMessageWithQuestion.value.questionData)
	} catch (e) {
		console.error('[Chat] Failed to parse question data:', e)
		return []
	}
})

onMounted(() => {
	chatStore.loadChatHistory()
	detectMobile()
	scrollToBottom()
})

onUnmounted(() => {
	chatStore.disconnectSSE()
})

function detectMobile() {
	const isMobileDevice = typeof window !== 'undefined' && window.innerWidth < 768
	chatStore.setMobile(isMobileDevice)
}

function sendMessage() {
	if (userInput.value.trim() === '') {
		return
	}
	chatStore.sendMessage(userInput.value)
	userInput.value = ''
	isProcessing.value = true
	processingText.value = t('chatAssistant.processing')
}

function clearMessages() {
	chatStore.clearMessages()
	isProcessing.value = false
	processingText.value = ''
	localStorage.removeItem(ANSWERED_TOOL_CALLS_KEY)
}

async function handleQuestionOption(question: IQuestion, option: IQuestionOption) {
	const answer = option.label
	const toolCallId = lastMessageWithQuestion.value?.toolCallID

	if (toolCallId) {
		setToolCallAnswered(toolCallId)
	}

	const userMessageText = t('chatAssistant.selectedOption', {option: answer})
	chatStore.addMessage({
		id: `msg_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`,
		type: 'user_input',
		role: 'user',
		content: userMessageText,
		timestamp: Date.now(),
	})

	isProcessing.value = true
	processingText.value = t('chatAssistant.processing')

	await chatStore.submitQuestionAnswer(answer)
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
	() => visibleMessages.value.length,
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

watch(
	visibleMessages,
	(newMessages) => {
		if (newMessages.length === 0) {
			isProcessing.value = false
			processingText.value = ''
			return
		}

		const lastMessage = newMessages[newMessages.length - 1]
		if (!lastMessage) return

		isProcessing.value = lastMessage.role === 'user'
		processingText.value = isProcessing.value ? t('chatAssistant.processing') : ''
	},
	{deep: true},
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
		inset: auto 0 0;
		block-size: 75dvh;
		border-radius: 0;
		border-inline-start: none;
		border-block-start: none;
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

.loading-message {
	.loading-content {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.75rem 1rem;
		background: var(--grey-100);
		border-radius: 1rem;
		border-start-start-radius: 0.25rem;

		.loading-dots {
			display: flex;
			gap: 0.25rem;

			.dot {
				inline-size: 0.375rem;
				block-size: 0.375rem;
				background: var(--primary);
				border-radius: 50%;
				animation: bounce 1.4s infinite ease-in-out both;

				&:nth-child(1) {
					animation-delay: -0.32s;
				}

				&:nth-child(2) {
					animation-delay: -0.16s;
				}
			}
		}

		.loading-text {
			font-size: 0.875rem;
			color: var(--grey-700);
			animation: pulse 1.5s ease-in-out infinite;
		}
	}
}

.button-navigation {
	margin-block-start: 0.5rem;
}

.nav-button {
	padding: 0.5rem 1rem;
	font-size: 0.875rem;
	background: var(--primary);
	color: var(--white);
	border: none;
	border-radius: 0.375rem;
	cursor: pointer;
	transition: background-color 0.2s;
	display: flex;
	align-items: center;
	gap: 0.25rem;

	.nav-title {
		opacity: 0.85;
		font-weight: 400;
	}

	&:hover {
		background: var(--primary-dark);
	}
}

.question-panel {
	margin-block-start: 0.5rem;
	padding: 0.75rem;
	background: var(--grey-50);
	border-radius: 0.5rem;
}

.question-item {
	margin-block-end: 0.75rem;

	&:last-child {
		margin-block-end: 0;
	}
}

.question-text {
	font-size: 0.875rem;
	font-weight: 600;
	color: var(--grey-800);
	margin-block-end: 0.5rem;
}

.question-options {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
}

.question-option {
	padding: 0.625rem;
	background: var(--white);
	border: 1px solid var(--grey-200);
	border-radius: 0.375rem;
	cursor: pointer;
	transition: all 0.2s;

	&:hover:not(.disabled) {
		background: var(--grey-50);
		border-color: var(--primary);
	}

	&:active:not(.disabled) {
		transform: scale(0.98);
	}

	&.disabled {
		cursor: not-allowed;
		opacity: 0.6;
		background: var(--grey-100);
	}
}

.option-label {
	font-size: 0.875rem;
	font-weight: 600;
	color: var(--grey-800);
	margin-block-end: 0.25rem;
}

.option-description {
	font-size: 0.8125rem;
	color: var(--grey-600);
	line-height: 1.4;
}

.dark .question-panel {
	background: var(--grey-700);

	.question-text {
		color: var(--grey-100);
	}
}

.dark .question-option {
	background: var(--grey-800);
	border-color: var(--grey-600);

	&:hover:not(.disabled) {
		background: var(--grey-700);
		border-color: var(--primary);
	}

	&.disabled {
		background: var(--grey-750);
	}

	.option-label {
		color: var(--grey-100);
	}

	.option-description {
		color: var(--grey-400);
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
	background: #ffeeee;
	border: 1px solid #ffcccc;
	border-radius: 0.375rem;
	color: #cc3333;
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
	border-block-start-color: var(--grey-700);

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
		border-block-start-color: var(--grey-700);

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

@keyframes bounce {
	0%,
	80%,
	100% {
		transform: scale(0);
	}

	40% {
		transform: scale(1);
	}
}

@keyframes pulse {
	0%,
	100% {
		opacity: 1;
	}

	50% {
		opacity: 0.6;
	}
}
</style>
