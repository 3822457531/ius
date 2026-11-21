<template>
  <div class="story-page">
    <header class="story-page__head van-safe-area-top">
      <button type="button" class="story-page__icon-btn" aria-label="records" @click="showRecords = true">
        <van-icon name="notes-o" size="20" />
      </button>
      <div class="story-page__head-main">
        <div class="story-page__head-badge">
          <van-icon name="description" size="22" color="#fff" />
        </div>
        <div>
          <h1 class="story-page__title">{{ t('dramaStory.title') }}</h1>
          <p class="story-page__subtitle">{{ t('dramaStory.subtitle') }}</p>
        </div>
      </div>
      <button type="button" class="story-page__icon-btn" aria-label="close" @click="closeToHub">
        <van-icon name="cross" size="20" />
      </button>
    </header>

    <div v-if="dramaMaintenance" class="story-page__maintenance">
      <van-icon name="warning-o" size="22" color="#8b5cf6" />
      <p>{{ t('imageApps.aiDouyin') }}{{ t('imageApps.maintenance') }}</p>
    </div>

    <div class="story-page__steps" role="tablist">
      <button
        v-for="(label, idx) in stepLabels"
        :key="idx"
        type="button"
        class="story-page__step"
        :class="{
          'story-page__step--active': step === idx,
          'story-page__step--done': step > idx,
        }"
        @click="goStep(idx)"
      >
        <span class="story-page__step-num">{{ idx + 1 }}</span>
        <span class="story-page__step-label">{{ label }}</span>
      </button>
    </div>

    <div class="story-page__scroll">
      <!-- Step 0: Script -->
      <section v-show="step === 0" class="story-card">
        <div class="story-card__head">
          <h2 class="story-card__title">{{ t('dramaStory.stepScriptTitle') }}</h2>
          <p class="story-card__desc">{{ t('dramaStory.stepScriptDesc') }}</p>
        </div>
        <van-field
          v-model="topic"
          class="story-card__field"
          type="textarea"
          :border="false"
          :placeholder="t('dramaStory.topicPlaceholder')"
          :autosize="{ minHeight: 88, maxHeight: 140 }"
          maxlength="80"
          show-word-limit
        />
        <van-button
          round
          block
          type="primary"
          class="story-card__btn"
          color="linear-gradient(135deg, #8b5cf6 0%, #6366f1 100%)"
          :loading="loadingScript"
          :disabled="!topic.trim()"
          @click="onGenerateScript"
        >
          {{ scriptPayload ? t('dramaStory.regenerateScript') : t('dramaStory.generateScript') }}
        </van-button>

        <div v-if="scriptPayload" class="story-script">
          <van-field
            v-model="scriptPayload.title"
            class="story-script__title-field"
            :border="false"
            :placeholder="t('dramaStory.scriptTitlePlaceholder')"
            @blur="onScriptOutlineEdited"
          />
          <article
            v-for="(block, idx) in scriptPayload.outline"
            :key="block.id || 'seg-' + idx"
            class="story-script__seg"
          >
            <div class="story-script__seg-head">
              <van-field
                v-model="block.title"
                class="story-script__seg-title-field"
                :border="false"
                :placeholder="t('dramaStory.segmentFallback', { n: idx + 1 })"
                @blur="onScriptOutlineEdited"
              />
              <button
                type="button"
                class="story-script__seg-del"
                :aria-label="t('dramaStory.deleteSegment')"
                @click="onDeleteSegment(idx)"
              >
                <van-icon name="delete-o" size="18" />
              </button>
            </div>
            <van-field
              v-model="block.text"
              class="story-script__seg-text-field"
              type="textarea"
              :border="false"
              :autosize="{ minHeight: 72, maxHeight: 200 }"
              :placeholder="t('dramaStory.segmentTextPlaceholder')"
              @blur="onScriptOutlineEdited"
            />
          </article>
          <van-button plain hairline round size="small" class="story-script__regen" @click="regenScript">
            {{ t('dramaStory.regenerateScript') }}
          </van-button>
        </div>
      </section>

      <!-- Step 1: TTS -->
      <section v-show="step === 1" class="story-card">
        <div class="story-card__head">
          <h2 class="story-card__title">{{ t('dramaStory.stepVoiceTitle') }}</h2>
          <p class="story-card__desc">{{ t('dramaStory.stepVoiceDesc') }}</p>
        </div>
        <div v-if="ttsLoading" class="story-status">
          <van-loading type="spinner" size="20" color="#8b5cf6" />
          <span>{{ t('dramaStory.voiceGenerating') }}</span>
          <van-progress :percentage="ttsProgress" stroke-width="6" color="#8b5cf6" />
        </div>
        <template v-else-if="ttsItems.length">
          <div v-for="item in ttsItems" :key="item.segmentId" class="story-voice">
            <div class="story-voice__meta">
              <van-icon name="volume-o" size="16" color="#8b5cf6" />
              <span>{{ item.fileName || item.segmentId }}</span>
            </div>
            <audio v-if="item.url" :src="teachingMediaSrc(item.url)" controls class="story-voice__audio" />
          </div>
          <van-button plain round size="small" @click="runTts">{{ t('dramaStory.regenerateVoice') }}</van-button>
        </template>
        <van-empty v-else :description="t('dramaStory.voiceEmpty')" image-size="72" />
      </section>

      <!-- Step 2: Prompts -->
      <section v-show="step === 2" class="story-card">
        <div class="story-card__head">
          <h2 class="story-card__title">{{ t('dramaStory.stepBoardTitle') }}</h2>
          <p class="story-card__desc">{{ t('dramaStory.stepBoardDesc') }}</p>
        </div>
        <div v-if="promptsLoading" class="story-status">
          <van-loading type="spinner" size="20" color="#8b5cf6" />
          <span>{{ t('dramaStory.boardGenerating') }}</span>
        </div>
        <template v-else-if="imagePromptRows.length">
          <p class="story-board-count">{{ t('dramaStory.boardCount', { n: imagePromptRows.length }) }}</p>
          <div v-for="(row, idx) in imagePromptRows" :key="row.segmentId || row.id || idx" class="story-board">
            <div class="story-board__head">
              <span class="story-board__idx">{{ idx + 1 }}</span>
              <strong>{{ row.title || row.segmentId }}</strong>
            </div>
            <p class="story-board__prompt">{{ row.prompt }}</p>
          </div>
          <van-button plain round size="small" :loading="promptsLoading" @click="loadImagePrompts">
            {{ t('dramaStory.regenerateBoard') }}
          </van-button>
        </template>
        <van-empty v-else :description="t('dramaStory.boardEmpty')" image-size="72" />
      </section>

      <!-- Step 3: Visuals -->
      <section v-show="step === 3" class="story-card">
        <div class="story-card__head">
          <h2 class="story-card__title">{{ t('dramaStory.stepVisualTitle') }}</h2>
          <p class="story-card__desc">{{ t('dramaStory.stepVisualDesc') }}</p>
        </div>

        <div v-if="recordVideoUrl" class="story-video">
          <p class="story-video__label">{{ t('dramaStory.videoReady') }}</p>
          <video :src="teachingMediaSrc(recordVideoUrl)" controls playsinline class="story-video__player" />
        </div>

        <div v-if="assetTasks.length" class="story-shots">
          <article v-for="task in assetTasks" :key="task.segmentId" class="story-shot">
            <div class="story-shot__head">
              <span>{{ task.label }}</span>
              <van-tag v-if="task.status === 'done'" type="success" size="medium">{{ t('dramaStory.statusDone') }}</van-tag>
              <van-tag v-else-if="task.status === 'running'" type="warning" size="medium">{{ t('dramaStory.statusRunning') }}</van-tag>
              <van-tag v-else-if="task.status === 'error'" type="danger" size="medium">{{ t('dramaStory.statusFailed') }}</van-tag>
              <van-tag v-else type="primary" plain size="medium">{{ t('dramaStory.statusPending') }}</van-tag>
            </div>
            <van-progress
              v-if="task.status === 'running'"
              :percentage="task.progress"
              stroke-width="5"
              color="#8b5cf6"
            />
            <p v-if="task.errorMsg" class="story-shot__error">{{ task.errorMsg }}</p>
            <div v-if="task.previewImage" class="story-shot__preview">
              <van-image
                :src="teachingMediaSrc(task.previewImage)"
                width="100%"
                height="160"
                fit="cover"
                radius="12"
                lazy-load
              />
            </div>
            <div v-if="task.audioUrl" class="story-shot__audio">
              <audio :src="teachingMediaSrc(task.audioUrl)" controls class="story-voice__audio" />
            </div>
            <van-button
              v-if="task.status !== 'running'"
              plain
              hairline
              round
              size="small"
              @click="regenOneImage(task.segmentId)"
            >
              {{ t('dramaStory.regenerateShot') }}
            </van-button>
          </article>
        </div>
        <van-empty v-else :description="t('dramaStory.visualEmpty')" image-size="72" />

        <van-button
          v-if="canRenderVideo"
          round
          block
          class="story-card__btn story-card__btn--secondary"
          :loading="renderVideoLoading"
          @click="onRenderVideo"
        >
          {{ t('dramaStory.renderVideo') }}
        </van-button>
      </section>

      <div class="story-page__scroll-spacer" />
    </div>

    <footer class="story-page__footer van-safe-area-bottom">
      <van-button round plain hairline :disabled="step <= 0" @click="onPrev">
        {{ t('dramaStory.prev') }}
      </van-button>
      <van-button
        round
        type="primary"
        color="linear-gradient(135deg, #8b5cf6 0%, #6366f1 100%)"
        :loading="nextLoading"
        :disabled="!canNext && step < 3"
        @click="onNext"
      >
        {{ step === 3 ? t('dramaStory.finish') : t('dramaStory.next') }}
      </van-button>
    </footer>

    <van-popup
      v-model:show="showRecords"
      position="left"
      :style="{ width: '82%', height: '100%' }"
      class="story-records-popup"
    >
      <div class="story-records">
        <div class="story-records__head">
          <h2>{{ t('dramaStory.recordsTitle') }}</h2>
          <van-button type="primary" size="small" round @click="onNewDraft">{{ t('dramaStory.newDraft') }}</van-button>
        </div>
        <van-loading v-if="recordsLoading" class="story-records__loading" />
        <div v-else-if="recordsList.length" class="story-records__list">
          <button
            v-for="r in recordsList"
            :key="r.id"
            type="button"
            class="story-records__item"
            :class="{ 'story-records__item--active': recordId === r.id }"
            @click="onPickRecord(r.id)"
          >
            <span class="story-records__item-title">{{ r.title_summary || r.topic || t('dramaStory.untitled') }}</span>
            <time class="story-records__item-time">{{ formatRecordTime(r.updated_at) }}</time>
          </button>
        </div>
        <van-empty v-else :description="t('dramaStory.recordsEmpty')" image-size="64" />
      </div>
    </van-popup>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { showConfirmDialog } from 'vant'
import { IMAGE_RETURN_HUB_KEY } from '@/constants/imageAppLaunch'
import { getSystemInfo } from '@/store/cache'
import { parseAppFlags } from '@/utils/appFeatures'
import { useTeachingStory } from '@/composables/useTeachingStory'

const { t } = useI18n()
const router = useRouter()
const returnToHubNative = inject(IMAGE_RETURN_HUB_KEY, null)
const showRecords = ref(false)
const dramaMaintenance = ref(false)

async function loadAppFlags() {
  try {
    const sys = await getSystemInfo()
    dramaMaintenance.value = !parseAppFlags(sys).drama
  } catch {
    dramaMaintenance.value = false
  }
}

onMounted(() => {
  void loadAppFlags()
})

const stepLabels = computed(() => [
  t('dramaStory.stepScript'),
  t('dramaStory.stepVoice'),
  t('dramaStory.stepBoard'),
  t('dramaStory.stepVisual'),
])

const {
  step,
  topic,
  loadingScript,
  scriptPayload,
  ttsLoading,
  ttsProgress,
  ttsItems,
  imagePromptRows,
  assetTasks,
  recordId,
  recordsList,
  recordsLoading,
  recordVideoUrl,
  renderVideoLoading,
  nextLoading,
  promptsLoading,
  canNext,
  canRenderVideo,
  teachingMediaSrc,
  formatRecordTime,
  loadRecord,
  newDraft,
  onGenerateScript,
  regenScript,
  onScriptOutlineEdited,
  removeScriptSegment,
  runTts,
  loadImagePrompts,
  regenOneImage,
  onRenderVideo,
  onPrev,
  onNext,
} = useTeachingStory()

async function closeToHub() {
  if (typeof returnToHubNative === 'function') {
    await returnToHubNative()
    return
  }
  router.push({ name: 'mobile-image-hub' })
}

function goStep(idx) {
  if (idx < 0 || idx > 3) return
  if (idx === 1 && !scriptPayload.value) return
  if (idx === 2 && !ttsItems.value.length) return
  if (idx === 3 && !imagePromptRows.value.length) return
  step.value = idx
}

async function onPickRecord(id) {
  await loadRecord(id)
  showRecords.value = false
}

function onNewDraft() {
  newDraft()
  showRecords.value = false
}

function onDeleteSegment(idx) {
  showConfirmDialog({
    title: t('dramaStory.deleteSegment'),
    message: t('dramaStory.deleteSegmentConfirm'),
    confirmButtonColor: '#ef4444',
  })
    .then(() => {
      removeScriptSegment(idx)
    })
    .catch(() => {})
}
</script>

<style scoped lang="stylus">
@import "@/assets/css/custom-scroll.styl"

.story-page
  display flex
  flex-direction column
  min-height 100vh
  min-height 100dvh
  background linear-gradient(168deg, #faf5ff 0%, #fff9f0 38%, #f0f7ff 100%)
  color #1a2233

.story-page__maintenance
  display flex
  align-items flex-start
  gap 10px
  margin 12px 14px 0
  padding 12px 14px
  border-radius 12px
  background rgba(139, 92, 246, 0.08)
  border 1px solid rgba(139, 92, 246, 0.18)
  font-size 13px
  line-height 1.5
  color #5b21b6

.story-page__head
  flex-shrink 0
  display flex
  align-items flex-start
  justify-content space-between
  gap 8px
  padding 12px 14px 14px
  background rgba(255, 255, 255, 0.9)
  border-bottom 1px solid rgba(99, 102, 241, 0.08)
  backdrop-filter blur(12px)
  -webkit-backdrop-filter blur(12px)

.story-page__head-main
  flex 1
  min-width 0
  display flex
  align-items center
  gap 12px

.story-page__head-badge
  flex-shrink 0
  width 44px
  height 44px
  border-radius 14px
  display flex
  align-items center
  justify-content center
  background linear-gradient(145deg, #a78bfa 0%, #6366f1 55%, #8b5cf6 100%)
  box-shadow 0 8px 22px rgba(99, 102, 241, 0.28)

.story-page__title
  margin 0
  font-size 18px
  font-weight 700
  color #1e1b4b
  line-height 1.3

.story-page__subtitle
  margin 4px 0 0
  font-size 12px
  line-height 1.45
  color rgba(30, 27, 75, 0.55)

.story-page__icon-btn
  flex-shrink 0
  display flex
  align-items center
  justify-content center
  width 36px
  height 36px
  padding 0
  border none
  border-radius 50%
  background rgba(99, 102, 241, 0.08)
  color #5b21b6
  cursor pointer
  -webkit-tap-highlight-color transparent

  &:active
    transform scale(0.94)

.story-page__steps
  flex-shrink 0
  display flex
  gap 6px
  padding 10px 12px 4px
  overflow-x auto
  -webkit-overflow-scrolling touch

  &::-webkit-scrollbar
    display none

.story-page__step
  flex 0 0 auto
  display flex
  align-items center
  gap 6px
  padding 8px 12px
  border none
  border-radius 999px
  background rgba(255, 255, 255, 0.72)
  border 1px solid rgba(99, 102, 241, 0.1)
  color rgba(30, 27, 75, 0.55)
  font-size 12px
  cursor pointer
  -webkit-tap-highlight-color transparent

  &--active
    background linear-gradient(135deg, rgba(139, 92, 246, 0.16), rgba(99, 102, 241, 0.12))
    border-color rgba(99, 102, 241, 0.35)
    color #5b21b6
    font-weight 600

  &--done
    color #4338ca

.story-page__step-num
  width 18px
  height 18px
  border-radius 50%
  display inline-flex
  align-items center
  justify-content center
  font-size 11px
  font-weight 700
  background rgba(99, 102, 241, 0.12)

.story-page__step-label
  white-space nowrap

.story-page__scroll
  flex 1
  overflow-y auto
  padding 12px 14px 0
  -webkit-overflow-scrolling touch

.story-page__scroll-spacer
  height 88px

.story-card
  padding 16px
  border-radius 20px
  background rgba(255, 255, 255, 0.82)
  border 1px solid rgba(255, 255, 255, 0.95)
  box-shadow 0 10px 32px rgba(79, 70, 229, 0.07)

.story-card__head
  margin-bottom 14px

.story-card__title
  margin 0
  font-size 16px
  font-weight 700
  color #1e1b4b

.story-card__desc
  margin 6px 0 0
  font-size 12px
  line-height 1.55
  color rgba(30, 27, 75, 0.52)

.story-card__field
  margin-bottom 12px
  border-radius 14px
  background rgba(250, 245, 255, 0.85)
  overflow hidden

.story-card__btn
  margin-top 8px

  &--secondary
    margin-top 16px

.story-script
  margin-top 16px
  padding-top 14px
  border-top 1px dashed rgba(99, 102, 241, 0.18)

.story-script__title-field
  margin-bottom 12px
  border-radius 12px
  background rgba(250, 245, 255, 0.85)
  overflow hidden

  :deep(.van-field__control)
    font-size 17px
    font-weight 700
    color #312e81
    line-height 1.45

.story-script__seg
  margin-bottom 12px
  padding 12px
  border-radius 14px
  background linear-gradient(180deg, rgba(250, 245, 255, 0.9), rgba(255, 255, 255, 0.6))
  border 1px solid rgba(99, 102, 241, 0.08)

.story-script__seg-head
  display flex
  align-items center
  gap 6px
  margin-bottom 4px

.story-script__seg-title-field
  flex 1
  min-width 0
  padding 0
  background transparent

  :deep(.van-field__control)
    font-size 13px
    font-weight 600
    color #5b21b6

.story-script__seg-del
  flex-shrink 0
  display flex
  align-items center
  justify-content center
  width 32px
  height 32px
  padding 0
  border none
  border-radius 10px
  background rgba(239, 68, 68, 0.08)
  color #ef4444
  cursor pointer
  -webkit-tap-highlight-color transparent

  &:active
    transform scale(0.94)

.story-script__seg-text-field
  padding 0
  background transparent

  :deep(.van-field__control)
    font-size 13px
    line-height 1.65
    color rgba(30, 27, 75, 0.78)

.story-script__regen
  margin-top 4px

.story-status
  display flex
  flex-direction column
  align-items center
  gap 10px
  padding 20px 8px
  font-size 13px
  color rgba(30, 27, 75, 0.62)

.story-voice
  margin-bottom 12px
  padding 12px
  border-radius 14px
  background #fff
  border 1px solid rgba(99, 102, 241, 0.08)

.story-voice__meta
  display flex
  align-items center
  gap 6px
  margin-bottom 8px
  font-size 13px
  font-weight 500
  color #4338ca

.story-voice__audio
  width 100%
  height 36px

.story-board-count
  margin 0 0 12px
  font-size 13px
  color #6366f1
  font-weight 600

.story-board
  margin-bottom 10px
  padding 12px
  border-radius 14px
  background rgba(250, 245, 255, 0.65)
  border 1px solid rgba(99, 102, 241, 0.08)

.story-board__head
  display flex
  align-items center
  gap 8px
  margin-bottom 6px
  font-size 13px
  color #312e81

.story-board__idx
  width 22px
  height 22px
  border-radius 8px
  display inline-flex
  align-items center
  justify-content center
  font-size 11px
  font-weight 700
  color #fff
  background linear-gradient(135deg, #a78bfa, #6366f1)

.story-board__prompt
  margin 0
  font-size 12px
  line-height 1.55
  color rgba(30, 27, 75, 0.68)

.story-shots
  display flex
  flex-direction column
  gap 12px

.story-shot
  padding 12px
  border-radius 16px
  background #fff
  border 1px solid rgba(99, 102, 241, 0.08)
  box-shadow 0 4px 16px rgba(79, 70, 229, 0.05)

.story-shot__head
  display flex
  align-items center
  justify-content space-between
  gap 8px
  margin-bottom 8px
  font-size 13px
  font-weight 600
  color #312e81

.story-shot__error
  margin 6px 0 0
  font-size 12px
  color #dc2626

.story-shot__preview
  margin 8px 0
  border-radius 12px
  overflow hidden

.story-shot__audio
  margin 8px 0

.story-video
  margin-bottom 16px
  padding 12px
  border-radius 16px
  background rgba(99, 102, 241, 0.06)

.story-video__label
  margin 0 0 8px
  font-size 13px
  font-weight 600
  color #4338ca

.story-video__player
  width 100%
  border-radius 12px
  background #000

.story-page__footer
  position fixed
  left 0
  right 0
  bottom 0
  z-index 20
  display flex
  gap 10px
  padding 10px 14px calc(10px + env(safe-area-inset-bottom))
  background rgba(255, 255, 255, 0.94)
  border-top 1px solid rgba(99, 102, 241, 0.1)
  backdrop-filter blur(12px)
  -webkit-backdrop-filter blur(12px)

  .van-button
    flex 1

.story-records
  display flex
  flex-direction column
  height 100%
  padding 16px 14px
  background linear-gradient(180deg, #faf5ff, #fff)

.story-records__head
  display flex
  align-items center
  justify-content space-between
  gap 8px
  margin-bottom 14px

  h2
    margin 0
    font-size 17px
    font-weight 700
    color #1e1b4b

.story-records__loading
  margin 24px auto

.story-records__list
  flex 1
  overflow-y auto
  -webkit-overflow-scrolling touch

.story-records__item
  display block
  width 100%
  margin-bottom 8px
  padding 12px
  text-align left
  border none
  border-radius 14px
  background #fff
  border 1px solid rgba(99, 102, 241, 0.08)
  cursor pointer

  &--active
    border-color rgba(99, 102, 241, 0.35)
    background rgba(139, 92, 246, 0.08)

.story-records__item-title
  display block
  font-size 14px
  font-weight 600
  color #312e81
  line-height 1.4

.story-records__item-time
  display block
  margin-top 4px
  font-size 11px
  color rgba(30, 27, 75, 0.45)
</style>
