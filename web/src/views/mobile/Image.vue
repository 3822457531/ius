<template>
  <div class="mobile-image-layout" :class="{ 'mobile-image-layout--mj': isMjStandalone }">
    <van-nav-bar
      v-if="showSubNav"
      fixed
      placeholder
      safe-area-inset-top
      :title="subTitle"
      :left-arrow="!isMjStandalone"
      :border="false"
      :class="isMjStandalone ? 'mobile-image-nav mobile-image-nav--sheet' : 'mobile-image-nav'"
      @click-left="onBack"
    >
      <template v-if="isMjStandalone" #right>
        <van-icon name="cross" class="mobile-image-nav__close" @click.stop="onBack" />
      </template>
    </van-nav-bar>
    <div class="mobile-image__view-shell">
      <router-view v-slot="{ Component }">
        <component :is="Component" :key="route.name" class="mobile-image__view-page" />
      </router-view>
    </div>

    <!-- 算力中心 → 图片 / 音乐 / PPT：加载卡片 -->
    <van-popup
      v-model:show="showLaunchOverlay"
      position="center"
      round
      :z-index="LAUNCH_Z_INDEX"
      overlay-class="mobile-image__launch-overlay"
      class="mobile-image__launch-popup"
      :style="launchCardStyle"
      :overlay="true"
      :close-on-click-overlay="false"
      teleport="body"
    >
      <div class="mobile-image__launch-card">
        <div class="mobile-image__launch-bar">
          <span class="mobile-image__launch-title">{{ launchTitle }}</span>
        </div>
        <div class="mobile-image__launch-body">
          <div class="mobile-image__http-track" aria-hidden="true">
            <div :key="launchAnimKey" class="mobile-image__http-fill" />
          </div>
          <van-loading type="spinner" color="#1989fa" size="36px" />
          <p class="mobile-image__launch-hint">{{ t('imageHub.launchLoadingHint') }}</p>
        </div>
      </div>
    </van-popup>

    <!-- 仿原生 iframe/网页：白屏文档自下而上盖住视口，极短停留后淡出（与 PPT 进页时空白帧接近） -->
    <Teleport to="body">
      <div
        v-show="nativeBlankShow"
        class="mobile-image__native-blank"
        :class="nativeBlankLayerClass"
        aria-hidden="true"
      />
    </Teleport>
  </div>
</template>

<script setup>
import { computed, ref, provide, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  IMAGE_APP_LAUNCH_KEY,
  IMAGE_EMBED_LAUNCH_KEY,
  IMAGE_RETURN_HUB_KEY,
} from '@/constants/imageAppLaunch'

/** 加载卡片 */
const LAUNCH_Z_INDEX = 3100

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const MJ_STANDALONE_ROUTES = ['mobile-image-mj', 'mobile-image-mj-gpt']

const showSubNav = computed(() => MJ_STANDALONE_ROUTES.includes(route.name))
const isMjStandalone = computed(() => MJ_STANDALONE_ROUTES.includes(route.name))
const subTitle = computed(() => {
  if (MJ_STANDALONE_ROUTES.includes(route.name)) return t('imageTabs.tabImage')
  return ''
})

function onBack() {
  router.push({ name: 'mobile-image-hub' })
}

const showLaunchOverlay = ref(false)
const launchTitle = ref('')
const launchAnimKey = ref(0)
const launchBusy = ref(false)

const launchCardStyle = {
  width: 'min(86vw, 360px)',
  maxWidth: '100%',
  overflow: 'hidden',
}

const nativeBlankShow = ref(false)
/** off | prep | slide | fade | exitSlide */
const nativeBlankPhase = ref('off')
/** 白屏进/出动画互斥，避免连点或开闭交错 */
const nativeBlankAnimBusy = ref(false)

const nativeBlankLayerClass = computed(() => {
  if (!nativeBlankShow.value || nativeBlankPhase.value === 'off') return ''
  const base = 'mobile-image__native-blank'
  if (nativeBlankPhase.value === 'prep') return base
  if (nativeBlankPhase.value === 'slide') return `${base} ${base}--slide`
  if (nativeBlankPhase.value === 'fade') return `${base} ${base}--slide ${base}--fade`
  if (nativeBlankPhase.value === 'exitSlide') return `${base} ${base}--slide ${base}--exit-down`
  return base
})

/**
 * 模拟浏览器打开新文档：整屏白底从底部推入 → 短停 → 淡出露底下列表页
 *（不挪真实 DOM，仅视觉层；PPT 仍走真实 iframe 白屏）
 */
async function playNativeBlankWebLayer() {
  if (nativeBlankAnimBusy.value) return
  nativeBlankAnimBusy.value = true
  try {
    nativeBlankPhase.value = 'prep'
    nativeBlankShow.value = true
    await nextTick()
    await new Promise((r) => requestAnimationFrame(r))
    await new Promise((r) => requestAnimationFrame(r))
    nativeBlankPhase.value = 'slide'
    const slideMs = 330
    await new Promise((r) => setTimeout(r, slideMs))
    const holdMs = 60 + Math.floor(Math.random() * 90)
    await new Promise((r) => setTimeout(r, holdMs))
    nativeBlankPhase.value = 'fade'
    const fadeMs = 150
    await new Promise((r) => setTimeout(r, fadeMs))
    nativeBlankShow.value = false
    nativeBlankPhase.value = 'off'
  } finally {
    nativeBlankAnimBusy.value = false
  }
}

/**
 * 仿关闭网页：白屏自下而上盖住当前页 → 回算力中心 → 白屏再向下滑出
 */
async function playNativeBlankCloseLayer() {
  if (nativeBlankAnimBusy.value) return
  if (route.name === 'mobile-image-hub') return
  nativeBlankAnimBusy.value = true
  try {
    nativeBlankPhase.value = 'prep'
    nativeBlankShow.value = true
    await nextTick()
    await new Promise((r) => requestAnimationFrame(r))
    await new Promise((r) => requestAnimationFrame(r))
    nativeBlankPhase.value = 'slide'
    const coverMs = 330
    await new Promise((r) => setTimeout(r, coverMs))
    await router.push({ name: 'mobile-image-hub' })
    await nextTick()
    const holdMs = 40 + Math.floor(Math.random() * 70)
    await new Promise((r) => setTimeout(r, holdMs))
    nativeBlankPhase.value = 'exitSlide'
    const exitMs = 330
    await new Promise((r) => setTimeout(r, exitMs))
    nativeBlankShow.value = false
    nativeBlankPhase.value = 'off'
  } finally {
    nativeBlankAnimBusy.value = false
  }
}

/**
 * @param {object} [options]
 * @param {boolean} [options.nativeBlankAfter] 仅图片/音乐：关闭加载卡片后接「白屏网页」层（PPT 勿开）
 */
async function runWithLaunchOverlay(title, task, options = {}) {
  const { nativeBlankAfter = false } = options
  if (launchBusy.value) return
  launchBusy.value = true
  try {
    launchTitle.value = title
    launchAnimKey.value += 1
    showLaunchOverlay.value = true
    await nextTick()
    const delayMs = Math.floor(200 + Math.random() * 1301)
    await new Promise((r) => setTimeout(r, delayMs))
    await task()
  } finally {
    showLaunchOverlay.value = false
  }
  if (nativeBlankAfter) {
    await playNativeBlankWebLayer()
  }
  launchBusy.value = false
}

async function runAppLaunch(title, routeName) {
  await runWithLaunchOverlay(
    title,
    () => router.push({ name: routeName }),
    { nativeBlankAfter: true }
  )
}

provide(IMAGE_APP_LAUNCH_KEY, runAppLaunch)
provide(IMAGE_EMBED_LAUNCH_KEY, (title, task) => runWithLaunchOverlay(title, task))
provide(IMAGE_RETURN_HUB_KEY, playNativeBlankCloseLayer)
</script>

<style lang="stylus">
.mobile-image-layout
  position relative
  display flex
  flex-direction column
  width 100%
  max-width 100%
  min-height 100vh
  min-height 100dvh
  overflow-x hidden
  box-sizing border-box

.mobile-image__view-shell
  position relative
  flex 1
  min-height 0
  width 100%
  overflow-x hidden
  overflow-y auto
  -webkit-overflow-scrolling touch

.mobile-image__view-page
  width 100%
  min-height 100%
  box-sizing border-box

/* 全屏白层：默认在视口下方，与未绘制的 about:blank 观感一致 */
.mobile-image__native-blank
  position fixed
  left 0
  right 0
  top 0
  bottom 0
  z-index 3040
  background #fff
  transform translateY(100%)
  opacity 1
  pointer-events none
  will-change transform, opacity

.mobile-image__native-blank--slide
  transition transform 0.33s cubic-bezier(0.32, 0.72, 0, 1)
  transform translateY(0)

.mobile-image__native-blank--fade
  transition transform 0.33s cubic-bezier(0.32, 0.72, 0, 1), opacity 0.14s ease-out
  opacity 0
  transform translateY(0)

/* 关闭：在已铺满（--slide）状态下整体向下移出视口 */
.mobile-image__native-blank--slide.mobile-image__native-blank--exit-down
  transform translateY(100%)

.mobile-image-nav--sheet
  background rgba(255, 255, 255, 0.92) !important
  backdrop-filter blur(12px)
  :deep(.van-nav-bar__title)
    font-weight 700
  :deep(.van-nav-bar__arrow)
    color #2c3e50

.mobile-image-nav__close
  display flex
  align-items center
  justify-content center
  width 36px
  height 36px
  margin-right -6px
  font-size 22px
  color #2c3e50
  border-radius 50%
  transition background 0.2s ease
  &:active
    background rgba(0, 0, 0, 0.06)

.mobile-image-layout--mj
  min-height 100vh
  min-height 100dvh

.mobile-image__launch-overlay
  background rgba(79, 172, 254, 0.42) !important
  backdrop-filter blur(2px)

.mobile-image__launch-popup
  background transparent
  box-shadow none

.mobile-image__launch-card
  display flex
  flex-direction column
  background #fff
  border 1px solid #ebedf0
  border-radius 16px
  box-shadow 0 12px 40px rgba(0, 0, 0, 0.12)
  overflow hidden

.mobile-image__launch-bar
  flex-shrink 0
  padding 16px 16px 14px
  border-bottom 1px solid #ebedf0
  background #fff

.mobile-image__launch-title
  display block
  font-size 17px
  font-weight 700
  color #323233
  text-align center
  letter-spacing 0.02em

.mobile-image__launch-body
  display flex
  flex-direction column
  align-items center
  justify-content center
  padding 32px 24px 36px
  gap 22px
  min-height 200px
  background #fff

.mobile-image__http-track
  width 100%
  max-width 280px
  height 4px
  border-radius 4px
  background #ebedf0
  overflow hidden

.mobile-image__http-fill
  display block
  height 100%
  width 100%
  transform-origin left center
  transform scaleX(0)
  border-radius 4px
  background linear-gradient(90deg, #1989fa 0%, #39c3fa 45%, #07c160 100%)
  animation mobile-image-http-fill 0.95s cubic-bezier(0.22, 1, 0.36, 1) forwards

.mobile-image__launch-hint
  margin 0
  font-size 14px
  color #969799
  text-align center
  line-height 1.5

@keyframes mobile-image-http-fill
  0%
    transform scaleX(0)
  55%
    transform scaleX(0.72)
  100%
    transform scaleX(1)
</style>
