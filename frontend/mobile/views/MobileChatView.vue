<template>
  <div class="mobile-chat">
    <main class="chat-area">
      <div v-if="chatStore.error" class="status-message error-message">
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
        class="messages-container"
        ref="messagesContainer"
      >
        <div
          v-for="msg in visibleMessages"
          :key="msg.id"
          class="message"
          :class="[msg.role]"
        >
          <div v-if="msg.content" class="message-content">
            {{ msg.content }}
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
              </div>
            </div>
          </div>

          <span v-if="msg.content || msg.buttonNavigation || msg.questionData" class="message-time">{{ formatTime(msg.timestamp) }}</span>
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
      <button class="clear-input-button" @click="clearMessages" :title="$t('chatAssistant.clear')">
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
        @click="handleSend"
        :disabled="!userInput.trim() || sendDisabled"
      >
        <Icon icon="arrow-up-from-bracket" />
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue';
import { useRoute } from 'vue-router';
import { useI18n } from 'vue-i18n';
import Icon from '@/components/misc/Icon';
import { useChatStore } from '@/stores/chat';
import type { IChatMessage, IQuestion, IQuestionOption } from '@/modelTypes/IChatMessage';

export default defineComponent({
  name: 'MobileChatView',
  components: {
    Icon,
  },
  setup() {
    const route = useRoute();
    const { t } = useI18n({ useScope: 'global' });
    const chatStore = useChatStore();

    const userInput = ref('');
    const messagesContainer = ref<HTMLElement | null>(null);
    const messageInput = ref<HTMLInputElement | null>(null);
    const isProcessing = ref(false);
    const processingText = ref('');
    const sendDisabled = ref(false);

    const ANSWERED_TOOL_CALLS_KEY = 'vikunja-answered-tool-calls';

    function getAnsweredToolCalls(): Set<string> {
      const stored = localStorage.getItem(ANSWERED_TOOL_CALLS_KEY);
      if (!stored) return new Set();
      try {
        return new Set(JSON.parse(stored));
      } catch {
        return new Set();
      }
    }

    function setToolCallAnswered(toolCallId: string) {
      const set = getAnsweredToolCalls();
      set.add(toolCallId);
      localStorage.setItem(ANSWERED_TOOL_CALLS_KEY, JSON.stringify([...set]));
    }

    watch(
      () => chatStore.hasPendingResponse,
      (pending) => {
        if (!pending) {
          isProcessing.value = false;
          processingText.value = '';
        }
      },
    );

    const lastMessageWithQuestion = computed(() => {
      const answeredToolCalls = getAnsweredToolCalls();
      const messagesWithQuestions = chatStore.messages.filter(msg =>
        msg.questionData &&
        msg.toolCallID &&
        !answeredToolCalls.has(msg.toolCallID),
      );
      return messagesWithQuestions[messagesWithQuestions.length - 1] || undefined;
    });

    const visibleMessages = computed(() => {
      return chatStore.messages.filter(msg =>
        msg.type === 'user_input' ||
        msg.type === 'assistant_response' ||
        msg.type === 'question' ||
        msg.type === 'button_navigation',
      );
    });

    function parseQuestions(questionData: string): IQuestion[] {
      if (!questionData) {
        return [];
      }
      try {
        return JSON.parse(questionData);
      } catch (e) {
        return [];
      }
    }

    function isQuestionDisabled(msg: IChatMessage): boolean {
      if (!msg.toolCallID) {
        return false;
      }

      const msgIndex = chatStore.messages.findIndex(m => m.id === msg.id);
      if (msgIndex === -1) {
        return false;
      }

      const messagesAfter = chatStore.messages.slice(msgIndex + 1);
      return messagesAfter.some(m => m.type === 'user_input');
    }

    onMounted(async () => {
      await chatStore.loadChatHistory();
      chatStore.setMobile(true);
      chatStore.isOpen = true;
      await nextTick();
      setTimeout(scrollToBottom, 100);
    });

    onUnmounted(() => {
      chatStore.isOpen = false;
    });

    watch(
      () => route.path,
      (newPath) => {
        if (newPath.startsWith('/mobile/chat')) {
          chatStore.setMobile(true);
          chatStore.isOpen = true;
        } else {
          chatStore.isOpen = false;
        }
      },
    );

    function scrollToBottom() {
      const scroll = () => {
        if (messagesContainer.value) {
          messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
        }
      };
      nextTick(() => {
        scroll();
        requestAnimationFrame(scroll);
        requestAnimationFrame(() => {
          requestAnimationFrame(scroll);
        });
      });
    }

    function formatTime(timestamp: number): string {
      const date = new Date(timestamp);
      const hours = date.getHours().toString().padStart(2, '0');
      const minutes = date.getMinutes().toString().padStart(2, '0');
      return `${hours}:${minutes}`;
    }

    function handleEnter() {
      if (isProcessing.value) {
        return;
      }
      handleSend();
    }

    function handleSend() {
      if (userInput.value.trim() === '' || sendDisabled.value) {
        return;
      }
      
      const message = userInput.value.trim();
      userInput.value = '';
      
      sendDisabled.value = true;
      isProcessing.value = true;
      processingText.value = t('chatAssistant.processing');
      
      chatStore.sendMessage(message);
      
      setTimeout(() => {
        sendDisabled.value = false;
      }, 500);
    }

    function sendMessage() {
      handleSend();
    }

    async function clearMessages() {
      await chatStore.clearMessages();
      isProcessing.value = false;
      processingText.value = '';
      localStorage.removeItem(ANSWERED_TOOL_CALLS_KEY);
    }

    async function handleQuestionOption(question: IQuestion, option: IQuestionOption) {
      const answer = option.label;
      const toolCallId = lastMessageWithQuestion.value?.toolCallID;

      if (toolCallId) {
        setToolCallAnswered(toolCallId);
      }

      const userMessageText = t('chatAssistant.selectedOption', { option: answer });
      chatStore.addMessage({
        id: `msg_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`,
        type: 'user_input',
        role: 'user',
        content: userMessageText,
        timestamp: Date.now(),
      });

      isProcessing.value = true;
      processingText.value = t('chatAssistant.processing');

      await chatStore.submitQuestionAnswer(answer);
    }

    watch(
      () => visibleMessages.value.length,
      () => {
        scrollToBottom();
      },
    );

    watch(
      visibleMessages,
      (newMessages) => {
        if (newMessages.length === 0) {
          isProcessing.value = false;
          processingText.value = '';
          return;
        }

        const lastMessage = newMessages[newMessages.length - 1];
        if (!lastMessage) return;

        isProcessing.value = lastMessage.role === 'user';
        processingText.value = isProcessing.value ? t('chatAssistant.processing') : '';
      },
      { deep: true },
    );

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
      t,
    };
  },
});
</script>

<style scoped>
.mobile-chat {
  display: flex;
  flex-direction: column;
  background: var(--color-background);
  height: 100%;
  min-height: 0;
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
  border: 1px solid var(--color-border);
  border-bottom-left-radius: 4px;
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
