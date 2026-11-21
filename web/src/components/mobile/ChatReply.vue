<template>
  <div class="mobile-message-reply">
    <div class="chat-icon" aria-hidden="true">
      <van-image :src="icon" round width="28" height="28" />
    </div>

    <div class="reply-body">
      <div class="reply-panel" :class="{ 'reply-panel--thinking': thinking }">
        <div v-if="thinking" class="reply-thinking">
          <span class="reply-thinking__text">{{ t('chat.thinking') }}</span>
          <van-loading type="spinner" size="16px" color="#0f766e" />
        </div>

        <template v-else>
          <p v-if="usedWebSearch" class="reply-web-hint">
            <van-icon name="globe-o" size="12" />
            <span>{{ t('chat.webSearchUsedBadge') }}</span>
          </p>
          <div
            ref="contentRef"
            :data-clipboard-text="orgContent"
            class="reply-content markdown-chat"
            v-html="content"
          ></div>
          <div v-if="createdAt || tokens" class="reply-meta">
            <span v-if="createdAt">{{ createdAt }}</span>
            <span v-if="tokens">{{ tokens }} tokens</span>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { nextTick, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n";
import {showImagePreview} from "vant";
import { enhanceMarkdownRoot } from "@/utils/markdownEnhance";

const { t } = useI18n();

const props = defineProps({
  content: {
    type: String,
    default: '',
  },
  orgContent: {
    type: String,
    default: '',
  },
  icon: {
    type: String,
    default: '/images/gpt-icon.png',
  },
  thinking: {
    type: Boolean,
    default: false,
  },
  createdAt: {
    type: String,
    default: '',
  },
  tokens: {
    type: [String, Number],
    default: '',
  },
  usedWebSearch: {
    type: Boolean,
    default: false,
  },
});

const contentRef = ref(null)

const bindRichContent = () => {
  if (!contentRef.value) {
    return
  }
  enhanceMarkdownRoot(contentRef.value)
  const imgs = contentRef.value.querySelectorAll('img')
  for (let i = 0; i < imgs.length; i++) {
    if (!imgs[i].src || imgs[i].dataset.previewBound) {
      continue
    }
    imgs[i].dataset.previewBound = '1'
    imgs[i].addEventListener('click', (e) => {
      e.stopPropagation()
      showImagePreview([imgs[i].src]);
    })
  }
}

onMounted(() => {
  bindRichContent()
})

watch(
  () => [props.content, props.thinking],
  () => {
    nextTick(bindRichContent)
  },
)
</script>

<style lang="stylus">
@import "@/assets/css/markdown-chat.styl"

@keyframes reply-thinking-blink
  0%, 100%
    opacity 1
  50%
    opacity 0.35

.mobile-message-reply
  display flex
  align-items flex-start
  gap 10px
  width 100%
  font-family 'Plus Jakarta Sans', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif

  .chat-icon
    flex-shrink 0
    margin-top 6px

  .reply-body
    flex 1
    min-width 0
    text-align left

  .reply-panel
    background var(--ai-surface, #ffffff)
    border 1px solid var(--ai-line, rgba(17, 24, 39, 0.08))
    border-radius 16px
    padding 13px 15px 14px
    box-shadow 0 1px 2px rgba(17, 24, 39, 0.04)

    &--thinking
      padding 12px 15px

  .reply-thinking
    display flex
    align-items center
    gap 8px
    color var(--ai-muted, #6b7280)
    font-size 15px

    &__text::after
      content '…'
      animation reply-thinking-blink 1.2s ease-in-out infinite

  .reply-web-hint
    display inline-flex
    align-items center
    gap 4px
    margin 0 0 10px
    padding 4px 8px
    border-radius 8px
    font-size 11px
    font-weight 600
    color #0369a1
    background rgba(14, 165, 233, 0.1)
    line-height 1.3

  .reply-content
    text-align left
    width 100%

  .reply-meta
    display flex
    flex-wrap wrap
    gap 10px
    margin-top 12px
    padding-top 10px
    border-top 1px solid rgba(17, 24, 39, 0.06)
    font-size 11px
    color #9ca3af

.van-theme-dark
  .mobile-message-reply
    .reply-panel
      background #1f2937
      border-color rgba(255, 255, 255, 0.08)
      box-shadow none

    .reply-thinking
      color #9ca3af

    .reply-web-hint
      color #7dd3fc
      background rgba(14, 165, 233, 0.15)

    .reply-meta
      border-top-color rgba(255, 255, 255, 0.08)
</style>
