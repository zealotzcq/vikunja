<template>
	<div class="mobile-chat">
		<header
			v-if="chatStore.currentTask"
			class="chat-header"
		>
			<div class="header-content">
				<span class="header-title">{{ $t('chatAssistant.title') }}</span>
				<span class="current-task-badge">
					{{ $t('chatAssistant.currentTask', { taskId: chatStore.currentTask.task_id, taskTitle: chatStore.currentTask.title }) }}
				</span>
			</div>
		</header>
		<main class="chat-area">
			<div
				v-if="chatStore.error"
				class="status-message error-message"
			>
				{{ chatStore.error }}
			</div>

			<div
				v-else-if="chatStore.isLoading"
				class="status-message loading-indicator"
			>
				<div class="loading-spinner" />
				<span>{{ $t('chatAssistant.loading') }}</span>
			</div>

			<div
				v-else-if="!chatStore.isAvailable"
				class="status-message unavailable-message"
			>
				{{ $t('chatAssistant.unavailable') }}
			</div>

			<div
				v-else
				ref="messagesContainer"
				class="messages-container"
			>
				<div
					v-for="msg in visibleMessages"
					:key="msg.id"
					class="message"
					:class="[msg.role]"
				>
					<div
						v-if="msg.content"
						class="message-content"
					>
						<template v-if="isTableContent(msg.content)">
							<table class="message-table">
								<tbody>
									<tr
										v-for="(row, index) in parseTableContent(msg.content)"
										:key="index"
									>
										<td
											v-for="(cell, cellIndex) in row"
											:key="cellIndex"
											:class="{ 'header-cell': index === 0 }"
										>
											{{ cell }}
										</td>
									</tr>
								</tbody>
							</table>
						</template>
						<template v-else>
							{{ msg.content }}
						</template>
					</div>

					<div
						v-if="msg.buttonNavigation"
						class="button-navigation"
					>
						<button
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
						</button>
					</div>

					<div
						v-if="msg.questionData"
						class="question-panel"
					>
						<div
							v-for="(question, qIndex) in parseQuestions(msg.questionData)"
							:key="qIndex"
							class="question-item"
						>
							<div class="question-text">
								{{ question.question }}
							</div>
							<div class="question-options">
								<button
									v-for="(option, oIndex) in question.options"
									:key="oIndex"
									class="question-option"
									:class="{ disabled: isQuestionDisabled(msg) }"
									@click="!isQuestionDisabled(msg) && handleQuestionOption(question, option)"
								>
									<div class="option-label">
										{{ option.label }}
									</div>
									<div class="option-description">
										{{ option.description }}
									</div>
								</button>
								<button
									class="question-option custom-option"
									:class="{ disabled: isQuestionDisabled(msg) }"
									@click="!isQuestionDisabled(msg) && showCustomInput(question)"
								>
									<div class="option-label">
										{{ $t('chatAssistant.customInput') }}
									</div>
									<div class="option-description">
										{{ $t('chatAssistant.customInputDescription') }}
									</div>
								</button>
							</div>
						</div>
					</div>

					<span
						v-if="msg.content || msg.buttonNavigation || msg.questionData"
						class="message-time"
					>{{ formatTime(msg.timestamp) }}</span>
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
		</main>

		<div
			v-if="chatStore.isAvailable"
			class="input-area"
		>
			<button
				class="clear-input-button"
				:title="$t('chatAssistant.clear')"
				@click="clearMessages"
			>
				<Icon icon="trash-alt" />
			</button>
			<input
				ref="messageInput"
				v-model="userInput"
				class="message-input"
				:placeholder="$t('chatAssistant.placeholder')"
				@keyup.enter="handleEnter"
			>
			<button
				class="send-button"
				:disabled="!userInput.trim() || sendDisabled"
				@click="handleSend"
			>
				<Icon icon="arrow-up-from-bracket" />
			</button>
		</div>

		<div
			v-if="showCustomInputModal"
			class="custom-input-modal"
			@click.self="closeCustomInput"
		>
			<div class="custom-input-content">
				<h3 class="custom-input-title">
					{{ $t('chatAssistant.customInputTitle') }}
				</h3>
				<textarea
					v-model="customInputValue"
					class="custom-input-textarea"
					:placeholder="$t('chatAssistant.customInputPlaceholder')"
					@keyup.enter.ctrl="confirmCustomInput"
				/>
				<div class="custom-input-buttons">
					<button
						class="custom-input-button cancel"
						@click="closeCustomInput"
					>
						{{ $t('misc.cancel') }}
					</button>
					<button
						class="custom-input-button confirm"
						:disabled="!customInputValue.trim()"
						@click="confirmCustomInput"
					>
						{{ $t('misc.confirm') }}
					</button>
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/misc/Icon'
import { useChatStore } from '@/stores/chat'
import { useCompanyStore } from '@/stores/company'
import type { IChatMessage, IQuestion, IQuestionOption } from '@/modelTypes/IChatMessage'

export default defineComponent({
	name: 'MobileChatView',
	components: {
		Icon,
	},
	setup() {
		const route = useRoute()
		const { t } = useI18n({ useScope: 'global' })
		const chatStore = useChatStore()
		const companyStore = useCompanyStore()

		const userInput = ref('')
		const messagesContainer = ref<HTMLElement | null>(null)
		const messageInput = ref<HTMLInputElement | null>(null)
		const isProcessing = ref(false)
		const processingText = ref('')
		const sendDisabled = ref(false)
		const showCustomInputModal = ref(false)
		const customInputValue = ref('')
		const currentQuestion = ref<IQuestion | null>(null)
		let historyPollingTimer: number | null = null

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

		watch(
			() => chatStore.messages.length,
			(newLength, oldLength) => {
				if (oldLength === 0 && newLength > 0) {
					nextTick().then(() => {
						setTimeout(scrollToBottom, 100)
					})
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
        msg.type === 'button_navigation' ||
        msg.type === 'question_answer',
			)
		})

		function parseQuestions(questionData: string): IQuestion[] {
			if (!questionData) {
				return []
			}
			try {
				return JSON.parse(questionData)
			} catch {
				return []
			}
		}

		function isQuestionDisabled(msg: IChatMessage): boolean {
			if (!msg.toolCallID) {
				return false
			}

			const msgIndex = chatStore.messages.findIndex(m => m.id === msg.id)
			if (msgIndex === -1) {
				return false
			}

			const messagesAfter = chatStore.messages.slice(msgIndex + 1)
			return messagesAfter.some(m => m.type === 'user_input')
		}

		onMounted(async () => {
			await chatStore.loadChatHistory()
			chatStore.setMobile(true)
			chatStore.isOpen = true
			await nextTick()
			setTimeout(scrollToBottom, 100)

			historyPollingTimer = window.setInterval(async () => {
				if (chatStore.isOpen && chatStore.isAvailable) {
					const previousLastMessageId = chatStore.messages.length > 0 ? chatStore.messages[chatStore.messages.length - 1]?.id ?? '' : ''

					const result = await chatStore.checkNewMessages(companyStore.currentCompanyId ?? undefined, previousLastMessageId)

					if (result.has_new) {
						await chatStore.loadChatHistory()
						await nextTick()
						scrollToBottom()
					}
				}
			}, 5000)
		})

		onUnmounted(() => {
			chatStore.isOpen = false
			if (historyPollingTimer !== null) {
				clearInterval(historyPollingTimer)
				historyPollingTimer = null
			}
		})

		watch(
			() => route.path,
			(newPath) => {
				if (newPath.startsWith('/mobile/chat')) {
					chatStore.setMobile(true)
					chatStore.isOpen = true
				} else {
					chatStore.isOpen = false
				}
			},
		)

		function scrollToBottom() {
			const scroll = () => {
				if (messagesContainer.value) {
					messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
				}
			}
			nextTick().then(() => {
				scroll()
				requestAnimationFrame(scroll)
				requestAnimationFrame(() => {
					requestAnimationFrame(scroll)
				})
			})
		}

		function formatTime(timestamp: number): string {
			const date = new Date(timestamp)
			const hours = date.getHours().toString().padStart(2, '0')
			const minutes = date.getMinutes().toString().padStart(2, '0')
			return `${hours}:${minutes}`
		}

		function isTableContent(content: string): boolean {
			if (!content) return false
			const lines = content.trim().split('\n')
			return lines.length > 1 && (lines[0]?.includes('\t') ?? false)
		}

		function parseTableContent(content: string): string[][] {
			if (!content) return []
			const lines = content.trim().split('\n')
			return lines.map(line => line.split('\t'))
		}

		function handleEnter() {
			if (isProcessing.value) {
				return
			}
			handleSend()
		}

		function handleSend() {
			if (userInput.value.trim() === '' || sendDisabled.value) {
				return
			}
      
			const message = userInput.value.trim()
			userInput.value = ''
      
			sendDisabled.value = true
			isProcessing.value = true
			processingText.value = t('chatAssistant.processing')
      
			chatStore.sendMessage(message)
      
			setTimeout(() => {
				sendDisabled.value = false
			}, 500)
		}

		function sendMessage() {
			handleSend()
		}

		async function clearMessages() {
			await chatStore.clearMessages()
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

			const userMessageText = t('chatAssistant.selectedOption', { option: answer })
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

		function showCustomInput(question: IQuestion) {
			currentQuestion.value = question
			customInputValue.value = ''
			showCustomInputModal.value = true
		}

		function closeCustomInput() {
			showCustomInputModal.value = false
			customInputValue.value = ''
			currentQuestion.value = null
		}

		async function confirmCustomInput() {
			if (!customInputValue.value.trim() || !currentQuestion.value) {
				return
			}

			const answer = customInputValue.value.trim()
			const toolCallId = lastMessageWithQuestion.value?.toolCallID

			if (toolCallId) {
				setToolCallAnswered(toolCallId)
			}

			const userMessageText = t('chatAssistant.selectedOption', { option: answer })
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
			closeCustomInput()
		}

		watch(
			() => visibleMessages.value.length,
			(newLength, oldLength) => {
				if (oldLength === 0 && newLength > 0) {
					setTimeout(scrollToBottom, 100)
				} else {
					scrollToBottom()
				}
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
			{ deep: true },
		)

		return {
			chatStore,
			userInput,
			messagesContainer,
			messageInput,
			isProcessing,
			sendDisabled,
			processingText,
			visibleMessages,
			lastMessageWithQuestion,
			parseQuestions,
			isQuestionDisabled,
			scrollToBottom,
			formatTime,
			sendMessage,
			handleSend,
			handleEnter,
			clearMessages,
			handleQuestionOption,
			showCustomInput,
			closeCustomInput,
			confirmCustomInput,
			showCustomInputModal,
			customInputValue,
			t,
			isTableContent,
			parseTableContent,
		}
	},
})
</script>

<style scoped>
.mobile-chat {
  display: flex;
  flex-direction: column;
  background: var(--color-background);
  height: 100%;
  min-height: 0;
}

.chat-header {
  flex-shrink: 0;
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
}

.header-content {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.header-title {
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--color-text-primary);
}

.current-task-badge {
  padding: var(--spacing-xs) var(--spacing-sm);
  background: var(--color-primary-light);
  color: var(--color-primary);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
  font-weight: 500;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
  min-height: 0;
  min-width: 0;
}

.status-message {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 0;
  overflow-y: auto;
  padding: var(--spacing-md);
}

.error-message {
  background: #FEE2E2;
  border: 2px solid #FCA5A5;
  color: #DC2626;
  font-size: var(--font-size-sm);
  text-align: center;
  border-radius: var(--radius-md);
  padding: var(--spacing-lg);
  max-width: 90%;
}

.loading-indicator {
  gap: var(--spacing-sm);
  color: var(--color-text-muted);
  font-size: var(--font-size-base);
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--color-primary-lighter);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.unavailable-message {
  color: var(--color-text-muted);
  font-size: var(--font-size-base);
  text-align: center;
  padding: var(--spacing-lg);
}

.messages-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: var(--spacing-md);
  overflow-y: auto;
  gap: var(--spacing-sm);
  min-height: 0;
  min-width: 0;
  align-content: flex-start;
  flex-grow: 1;
  flex-basis: 0;
}

.message {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-width: 80%;
  flex-shrink: 0;
}

.message.user {
  align-self: flex-end;
  align-items: flex-end;
}

.message.assistant {
  align-self: flex-start;
  align-items: flex-start;
}

.message-content {
  padding: var(--spacing-sm) var(--spacing-md);
  border-radius: var(--radius-lg);
  font-size: var(--font-size-base);
  line-height: 1.5;
  word-wrap: break-word;
  white-space: pre-wrap;
}

.message.user .message-content {
  background: var(--color-primary);
  color: white;
  border-bottom-right-radius: 4px;
}

.message.assistant .message-content {
  background: var(--color-surface);
  color: var(--color-text-primary);
  border:1px solid var(--color-border);
  border-bottom-left-radius: 4px;
}

.message-table {
  border-collapse: collapse;
  width: 100%;
  font-size: var(--font-size-sm);

  tbody {
    tr {
      border-bottom:1px solid var(--color-border);

      &:last-child {
        border-bottom: none;
      }

      td {
        padding: var(--spacing-xs) var(--spacing-sm);
        text-align: left;
      }

      .header-cell {
        font-weight: 600;
        color: var(--color-text-primary);
        background: var(--color-surface-hover);
      }
    }
  }
}

.message-time {
  font-size: 10px;
  color: var(--color-text-muted);
  margin-top: 4px;
}

.message.user .message-time {
  margin-right: 4px;
}

.message.assistant .message-time {
  margin-left: 4px;
}

.button-navigation {
  margin-top: var(--spacing-sm);
}

.nav-button {
  width: 100%;
  padding: var(--spacing-sm) var(--spacing-md);
  min-height: 44px;
  font-size: var(--font-size-sm);
  font-weight: 600;
  background: var(--color-primary);
  color: white;
  border: none;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background-color var(--transition-fast), transform var(--transition-fast);
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  font-family: var(--font-family);
}

.nav-button:hover {
  background: var(--color-primary-hover);
  transform: translateY(-1px);
}

.nav-button:active {
  background: var(--color-primary);
  transform: translateY(0);
}

.nav-title {
  opacity: 0.9;
  font-weight: 400;
}

.question-panel {
  margin-top: var(--spacing-sm);
  padding: var(--spacing-sm);
  background: var(--color-surface);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
}

.question-item {
  margin-bottom: var(--spacing-sm);
}

.question-item:last-child {
  margin-bottom: 0;
}

.question-text {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: var(--spacing-sm);
  font-family: var(--font-family);
}

.question-options {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.question-option {
  padding: var(--spacing-sm);
  min-height: 44px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background-color var(--transition-fast), border-color var(--transition-fast);
  text-align: left;
}

.question-option:hover:not(.disabled) {
  background: var(--color-surface-hover);
  border-color: var(--color-primary);
}

.question-option:active:not(.disabled) {
  background: var(--color-primary-lighter);
}

.question-option.disabled {
  cursor: not-allowed;
  opacity: 0.5;
  background: var(--color-background);
}

.option-label {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 4px;
  font-family: var(--font-family);
}

.option-description {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  line-height: 1.4;
}

.question-option.custom-option {
  border-color: var(--color-primary);
  background: var(--color-primary-lighter);
}

.custom-input-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: var(--spacing-md);
}

.custom-input-content {
  background: var(--color-surface);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
  width: 100%;
  max-width: 400px;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.custom-input-title {
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
  font-family: var(--font-family);
}

.custom-input-textarea {
  width: 100%;
  min-height: 120px;
  padding: var(--spacing-md);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--font-size-base);
  font-family: var(--font-family);
  resize: vertical;
  background: var(--color-background);
  color: var(--color-text-primary);
  transition: border-color var(--transition-fast);
}

.custom-input-textarea:focus {
  border-color: var(--color-primary);
  outline: none;
  background: var(--color-surface);
}

.custom-input-buttons {
  display: flex;
  gap: var(--spacing-sm);
  justify-content: flex-end;
}

.custom-input-button {
  padding: var(--spacing-sm) var(--spacing-lg);
  min-height: 40px;
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: 600;
  font-family: var(--font-family);
  cursor: pointer;
  transition: background-color var(--transition-fast);
}

.custom-input-button.cancel {
  background: var(--color-surface-hover);
  color: var(--color-text-primary);
}

.custom-input-button.cancel:hover {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
}

.custom-input-button.confirm {
  background: var(--color-primary);
  color: white;
}

.custom-input-button.confirm:hover:not(:disabled) {
  background: var(--color-primary-hover);
}

.custom-input-button.confirm:disabled {
  background: var(--color-primary-lighter);
  color: var(--color-primary);
  cursor: not-allowed;
}

.loading-message {
  align-self: flex-start !important;
  max-width: 80%;
  flex-shrink: 0;
}

.loading-content {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  border-bottom-left-radius: 4px;
}

.loading-dots {
  display: flex;
  gap: 4px;
}

.loading-dots .dot {
  width: 8px;
  height: 8px;
  background: var(--color-primary);
  border-radius: 50%;
  animation: bounce 1.4s infinite ease-in-out both;
}

.loading-dots .dot:nth-child(1) {
  animation-delay: -0.32s;
}

.loading-dots .dot:nth-child(2) {
  animation-delay: -0.16s;
}

.loading-text {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
  animation: pulse 1.5s ease-in-out infinite;
}

.input-area {
  flex-shrink: 0;
  display: flex;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--color-surface);
  border-top: 1px solid var(--color-border);
  margin-top: 0;
}

.clear-input-button {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-surface-hover);
  color: var(--color-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--font-size-base);
  cursor: pointer;
  transition: background-color var(--transition-fast), border-color var(--transition-fast);
  flex-shrink: 0;
}

.clear-input-button:hover {
  background: var(--color-primary-lighter);
  border-color: var(--color-primary);
}

.clear-input-button:active {
  background: var(--color-primary-light);
}

.message-input {
  flex: 1;
  padding: 10px 14px;
  min-height: 40px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--font-size-base);
  background: var(--color-background);
  transition: border-color var(--transition-fast), background-color var(--transition-fast);
  color: var(--color-text-primary);
  font-family: var(--font-family);
}

.message-input:focus {
  border-color: var(--color-primary);
  background: var(--color-surface);
  outline: none;
}

.message-input::placeholder {
  color: var(--color-text-muted);
}

.message-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.send-button {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary);
  color: white;
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--font-size-base);
  cursor: pointer;
  transition: background-color var(--transition-fast);
  flex-shrink: 0;
}

.send-button:hover:not(:disabled) {
  background: var(--color-primary-hover);
}

.send-button:active:not(:disabled) {
  background: var(--color-primary);
}

.send-button:disabled {
  background: var(--color-primary-lighter);
  color: var(--color-primary);
  cursor: not-allowed;
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

@media (prefers-reduced-motion: reduce) {
  .loading-spinner,
  .nav-button,
  .question-option,
  .clear-input-button,
  .message-input,
  .send-button,
  .loading-dots .dot,
  .loading-text {
    transition: none;
    animation: none;
  }

  .nav-button:hover,
  .nav-button:active {
    transform: none;
  }
}
</style>
