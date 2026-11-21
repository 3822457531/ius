<template>
  <div class="image-logo">
    <div class="image-logo__bg" aria-hidden="true">
      <span class="image-logo__orb image-logo__orb--1"></span>
      <span class="image-logo__orb image-logo__orb--2"></span>
      <span class="image-logo__orb image-logo__orb--3"></span>
    </div>

    <header class="image-logo__header van-safe-area-top">
      <button type="button" class="image-logo__header-btn" @click="closeToHub">
        <van-icon name="arrow-left" size="20" />
      </button>
      <div class="image-logo__header-center">
        <span class="image-logo__header-title">{{ t('imageLogo.heroTitle') }}</span>
        <!-- <span class="image-logo__header-badge">GPT-Image2</span> -->
      </div>
      <button type="button" class="image-logo__header-btn" @click="closeToHub">
        <van-icon name="cross" size="20" />
      </button>
    </header>

    <div v-if="logoMaintenance" class="image-logo__maintenance">
      <van-icon name="warning-o" size="22" color="#6366f1" />
      <p>{{ t('imageApps.ailogo') }}{{ t('imageApps.maintenance') }}</p>
    </div>

    <div class="image-logo__body">
      <section class="image-logo__hero">
        <div class="image-logo__hero-icon">
          <van-icon name="apps-o" size="28" color="#fff" />
        </div>
        <div class="image-logo__hero-copy">
          <h1 class="image-logo__hero-title">{{ t('imageLogo.heroSubtitle') }}</h1>
          <p class="image-logo__hero-desc">{{ t('imageLogo.intro') }}</p>
        </div>
      </section>

      <section class="image-logo__preview-card">
        <div class="image-logo__canvas" :style="{ background: previewGradient }">
          <div class="image-logo__canvas-inner">
            <div
              v-if="activeLayout.count > 1"
              class="image-logo__canvas-grid"
              :style="{
                gridTemplateColumns: `repeat(${activeLayout.cols}, 1fr)`,
                gridTemplateRows: `repeat(${activeLayout.rows}, 1fr)`,
              }"
            >
              <span
                v-for="n in activeLayout.count"
                :key="n"
                class="image-logo__canvas-cell"
              >
                <i class="image-logo__canvas-cell-dot"></i>
              </span>
            </div>
            <div v-else class="image-logo__canvas-mark">
              <span class="image-logo__canvas-shape"></span>
              <span class="image-logo__canvas-shape image-logo__canvas-shape--2"></span>
              <span class="image-logo__canvas-shape image-logo__canvas-shape--3"></span>
            </div>
            <p v-if="brandName.trim()" class="image-logo__canvas-brand">{{ brandName.trim() }}</p>
            <p v-else class="image-logo__canvas-hint">{{ t('imageLogo.canvasPlaceholder') }}</p>
            <p v-if="activeLayout.count > 1" class="image-logo__canvas-layout">
              {{ t('imageLogo.layoutPreview', { n: activeLayout.count, grid: activeLayout.label }) }}
            </p>
          </div>
        </div>
        <div class="image-logo__summary">
          <span
            v-for="tag in summaryTags"
            :key="tag.key"
            class="image-logo__summary-tag"
          >
            <van-icon :name="tag.icon" size="12" />
            {{ tag.value }}
          </span>
        </div>
      </section>

      <van-form @submit="generate">
        <section
          v-for="section in optionSections"
          :key="section.key"
          class="image-logo__section"
        >
          <div class="image-logo__section-head">
            <span class="image-logo__section-step">{{ section.step }}</span>
            <div class="image-logo__section-meta">
              <span class="image-logo__section-title">{{ t(section.labelKey) }}</span>
              <span class="image-logo__section-sub">{{ t(section.subKey) }}</span>
            </div>
            <van-icon :name="section.icon" class="image-logo__section-icon" size="18" />
          </div>

          <div
            class="image-logo__chip-scroll"
            :class="{ 'image-logo__chip-scroll--layout': section.key === 'layout' }"
          >
            <button
              v-for="item in section.options"
              :key="item.value"
              type="button"
              class="image-logo__chip"
              :class="{
                'image-logo__chip--active': selections[section.key] === item.value,
                'image-logo__chip--color': section.key === 'colorTone',
                'image-logo__chip--layout': section.key === 'layout',
              }"
              @click="selections[section.key] = item.value"
            >
              <span
                v-if="section.key === 'layout'"
                class="image-logo__chip-grid"
                :style="{
                  gridTemplateColumns: `repeat(${item.cols}, 1fr)`,
                  gridTemplateRows: `repeat(${item.rows}, 1fr)`,
                }"
              >
                <i v-for="n in item.count" :key="n"></i>
              </span>
              <span
                v-else-if="item.swatch"
                class="image-logo__chip-swatch"
                :style="{ background: item.swatch }"
              ></span>
              <van-icon v-else-if="item.icon" :name="item.icon" size="14" />
              <span class="image-logo__chip-text">{{ t(item.labelKey) }}</span>
            </button>
          </div>
        </section>

        <section class="image-logo__section image-logo__section--brand">
          <div class="image-logo__section-head">
            <span class="image-logo__section-step">06</span>
            <div class="image-logo__section-meta">
              <span class="image-logo__section-title">{{ t('imageLogo.brandSectionTitle') }}</span>
              <span class="image-logo__section-sub">{{ t('imageLogo.brandSectionSub') }}</span>
            </div>
            <van-icon name="edit" class="image-logo__section-icon" size="18" />
          </div>
          <div class="image-logo__brand-wrap">
            <van-icon name="shop-o" size="18" class="image-logo__brand-icon" />
            <input
              v-model="brandName"
              class="image-logo__brand-input"
              type="text"
              maxlength="30"
              :placeholder="t('imageLogo.placeholderBrandName')"
            />
            <button
              v-if="brandName"
              type="button"
              class="image-logo__brand-clear"
              @click="brandName = ''"
            >
              <van-icon name="clear" size="16" />
            </button>
          </div>
        </section>

        <div class="image-logo__footer-spacer"></div>
      </van-form>

      <section class="image-logo__panel">
        <div class="image-logo__panel-head">
          <div>
            <h3 class="image-logo__panel-title">{{ t('imageMj.runningTitle') }}</h3>
            <p class="image-logo__panel-sub">{{ t('imageLogo.runningSub') }}</p>
          </div>
          <van-tag v-if="runningJobs.length" round type="primary">{{ runningJobs.length }}</van-tag>
        </div>

        <div v-if="runningJobs.length === 0" class="image-logo__panel-empty">
          <van-icon name="clock-o" size="28" color="#c8c9cc" />
          <span>{{ t('imageLogo.runningEmpty') }}</span>
        </div>

        <div v-else class="image-logo__running-list">
          <div v-for="item in runningJobs" :key="item.id" class="image-logo__running-item">
            <div class="image-logo__running-thumb">
              <van-image src="/images/img-holder.png" width="100%" height="100%" fit="cover" radius="12" />
              <div v-if="item.progress > 0" class="image-logo__running-progress">
                <van-circle
                  v-model:current-rate="item.progress"
                  :rate="item.progress"
                  :speed="100"
                  :text="item.progress + '%'"
                  :stroke-width="60"
                  size="44px"
                  color="#0f766e"
                  layer-color="rgba(17,24,39,0.12)"
                />
              </div>
              <div v-else class="image-logo__running-queue">
                <van-loading type="spinner" size="22" color="#fff" />
                <span class="image-logo__running-dots">
                  <i
                    v-for="i in 3"
                    :key="i"
                    :class="{ active: loadingDots >= i }"
                  ></i>
                </span>
              </div>
            </div>
            <div class="image-logo__running-meta">
              <span class="image-logo__running-label">{{ t('imageMj.loadingText') }}</span>
              <span class="image-logo__running-hint">{{ t('imageLogo.runningHint') }}</span>
            </div>
          </div>
        </div>
      </section>

      <section class="image-logo__panel">
        <div class="image-logo__panel-head">
          <div>
            <h3 class="image-logo__panel-title">{{ t('imageLogo.historyTitle') }}</h3>
            <p class="image-logo__panel-sub">
              {{ finishedJobs.length ? t('imageLogo.historyCount', { n: finishedJobs.length }) : t('imageLogo.emptyDesc') }}
            </p>
          </div>
        </div>

        <van-empty
          v-if="finishedJobs.length === 0"
          class="image-logo__history-empty"
          image="https://fastly.jsdelivr.net/npm/@vant/assets/custom-empty-image.png"
          image-size="72"
          :description="t('imageLogo.emptyDesc')"
        />

        <van-list
          v-else
          v-model:error="error"
          v-model:loading="loading"
          :finished="finished"
          :error-text="t('imageMj.listErrorText')"
          :finished-text="t('imageMj.listFinishedText')"
          @load="onLoad"
        >
          <div class="image-logo__history-grid">
            <article
              v-for="item in finishedJobs"
              :key="item.id"
              class="image-logo__history-card"
            >
              <button type="button" class="image-logo__history-thumb" @click="imageView(item)">
                <van-image
                  :src="item.thumb_url"
                  width="100%"
                  height="100%"
                  lazy-load
                  radius="14"
                  show-loading
                  fit="cover"
                >
                  <template #loading>
                    <van-loading type="spinner" size="20" />
                  </template>
                </van-image>
                <span class="image-logo__history-zoom">
                  <van-icon name="enlarge" size="16" color="#fff" />
                </span>
              </button>
              <div class="image-logo__history-actions">
                <button type="button" class="image-logo__history-btn" @click="showPrompt(item)">
                  <van-icon name="description" size="15" />
                </button>
                <button type="button" class="image-logo__history-btn image-logo__history-btn--danger" @click="removeImage(item)">
                  <van-icon name="delete-o" size="15" />
                </button>
              </div>
            </article>
          </div>
        </van-list>
      </section>
    </div>

    <footer class="image-logo__footer van-safe-area-bottom">
      <div class="image-logo__footer-inner">
        <div class="image-logo__footer-info">
          <span class="image-logo__footer-label">{{ t('imageLogo.generateNow') }}</span>
          <span class="image-logo__footer-power">{{ t('imageLogo.powerCost', { n: mjPower }) }}</span>
        </div>
        <van-button
          round
          type="primary"
          class="image-logo__footer-btn"
          :loading="submitting"
          @click="generate"
        >
          <van-icon name="fire-o" size="18" />
          {{ t('imageLogo.generateNow') }}
        </van-button>
      </div>
    </footer>
  </div>
</template>

<script setup>
import { computed, inject, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  showConfirmDialog,
  showDialog,
  showFailToast,
  showImagePreview,
  showNotify,
  showSuccessToast,
  showToast,
} from 'vant'
import { httpGet, httpPost, resolveWsHost } from '@/utils/http'
import { getSystemInfo } from '@/store/cache'
import { parseAppFlags } from '@/utils/appFeatures'
import { getSessionId } from '@/store/session'
import { randString } from '@/utils/libs'
import { checkSession } from '@/action/session'
import { IMAGE_RETURN_HUB_KEY } from '@/constants/imageAppLaunch'
import { applyThumbUrl, resolveMjImageUrl, unwrapMjJobList, LOGO_PROMPT_MARKER } from '@/utils/mjJobs'

const { t } = useI18n()
const router = useRouter()
const returnToHubNative = inject(IMAGE_RETURN_HUB_KEY, null)
const logoMaintenance = ref(false)

async function loadAppFlags() {
  try {
    const sys = await getSystemInfo()
    logoMaintenance.value = !parseAppFlags(sys).logo
  } catch {
    logoMaintenance.value = false
  }
}

async function closeToHub() {
  if (typeof returnToHubNative === 'function') {
    await returnToHubNative()
    return
  }
  router.push({ name: 'mobile-image-hub' })
}

const industries = [
  { value: 'tech', labelKey: 'imageLogo.industryTech', icon: 'desktop-o' },
  { value: 'finance', labelKey: 'imageLogo.industryFinance', icon: 'gold-coin-o' },
  { value: 'food', labelKey: 'imageLogo.industryFood', icon: 'shop-o' },
  { value: 'fashion', labelKey: 'imageLogo.industryFashion', icon: 'bag-o' },
  { value: 'education', labelKey: 'imageLogo.industryEducation', icon: 'bookmark-o' },
  { value: 'health', labelKey: 'imageLogo.industryHealth', icon: 'like-o' },
  { value: 'realestate', labelKey: 'imageLogo.industryRealestate', icon: 'home-o' },
  { value: 'entertainment', labelKey: 'imageLogo.industryEntertainment', icon: 'video-o' },
  { value: 'travel', labelKey: 'imageLogo.industryTravel', icon: 'location-o' },
  { value: 'sports', labelKey: 'imageLogo.industrySports', icon: 'fire-o' },
]

const styles = [
  { value: 'minimalist', labelKey: 'imageLogo.styleMinimalist', icon: 'minus' },
  { value: 'modern', labelKey: 'imageLogo.styleModern', icon: 'gem-o' },
  { value: 'geometric', labelKey: 'imageLogo.styleGeometric', icon: 'apps-o' },
  { value: 'handdrawn', labelKey: 'imageLogo.styleHanddrawn', icon: 'edit' },
  { value: 'luxury', labelKey: 'imageLogo.styleLuxury', icon: 'gem-o' },
  { value: 'playful', labelKey: 'imageLogo.stylePlayful', icon: 'smile-o' },
  { value: 'corporate', labelKey: 'imageLogo.styleCorporate', icon: 'manager-o' },
  { value: 'vintage', labelKey: 'imageLogo.styleVintage', icon: 'clock-o' },
]

const colorTones = [
  { value: 'warm', labelKey: 'imageLogo.colorWarm', swatch: 'linear-gradient(135deg, #ff6b6b, #feca57)' },
  { value: 'cool', labelKey: 'imageLogo.colorCool', swatch: 'linear-gradient(135deg, #4facfe, #667eea)' },
  { value: 'monochrome', labelKey: 'imageLogo.colorMonochrome', swatch: 'linear-gradient(135deg, #2c2c2c, #ffffff)' },
  { value: 'vibrant', labelKey: 'imageLogo.colorVibrant', swatch: 'linear-gradient(135deg, #f093fb, #f5576c)' },
  { value: 'pastel', labelKey: 'imageLogo.colorPastel', swatch: 'linear-gradient(135deg, #fbc2eb, #a6c1ee)' },
  { value: 'dark', labelKey: 'imageLogo.colorDark', swatch: 'linear-gradient(135deg, #141e30, #243b55)' },
  { value: 'gold', labelKey: 'imageLogo.colorGold', swatch: 'linear-gradient(135deg, #f7971e, #ffd200)' },
  { value: 'bluegreen', labelKey: 'imageLogo.colorBluegreen', swatch: 'linear-gradient(135deg, #11998e, #38ef7d)' },
]

const themes = [
  { value: 'innovation', labelKey: 'imageLogo.themeInnovation', icon: 'bulb-o' },
  { value: 'nature', labelKey: 'imageLogo.themeNature', icon: 'like-o' },
  { value: 'trust', labelKey: 'imageLogo.themeTrust', icon: 'passed' },
  { value: 'speed', labelKey: 'imageLogo.themeSpeed', icon: 'upgrade' },
  { value: 'community', labelKey: 'imageLogo.themeCommunity', icon: 'friends-o' },
  { value: 'elegance', labelKey: 'imageLogo.themeElegance', icon: 'gem-o' },
  { value: 'energy', labelKey: 'imageLogo.themeEnergy', icon: 'fire-o' },
  { value: 'global', labelKey: 'imageLogo.themeGlobal', icon: 'location-o' },
]

const layouts = [
  { value: '1x1', labelKey: 'imageLogo.layout1', label: '1×1', cols: 1, rows: 1, count: 1 },
  { value: '2x2', labelKey: 'imageLogo.layout4', label: '2×2', cols: 2, rows: 2, count: 4 },
  { value: '3x3', labelKey: 'imageLogo.layout9', label: '3×3', cols: 3, rows: 3, count: 9 },
  { value: '2x3', labelKey: 'imageLogo.layout6', label: '2×3', cols: 2, rows: 3, count: 6 },
  { value: '1x4', labelKey: 'imageLogo.layoutRow4', label: '1×4', cols: 4, rows: 1, count: 4 },
  { value: '4x1', labelKey: 'imageLogo.layoutCol4', label: '4×1', cols: 1, rows: 4, count: 4 },
]

const gridPositions = [
  'top-left',
  'top-center',
  'top-right',
  'middle-left',
  'center',
  'middle-right',
  'bottom-left',
  'bottom-center',
  'bottom-right',
]

const cellVariationPrompts = [
  'icon-only symbol mark, bold silhouette, highly memorable',
  'lettermark using brand initials, strong typography focus',
  'combination mark pairing icon with wordmark, balanced composition',
  'abstract geometric symbol with unique negative space',
  'minimal line-art emblem, elegant thin strokes',
  'badge or seal style logo inside a contained shape',
  'dynamic asymmetric mark with forward motion',
  'organic flowing symbol inspired by natural forms',
  'monogram-style intertwined initials, premium feel',
]

const optionSections = [
  { key: 'industry', step: '01', labelKey: 'imageLogo.fieldIndustry', subKey: 'imageLogo.fieldIndustrySub', icon: 'apps-o', options: industries },
  { key: 'style', step: '02', labelKey: 'imageLogo.fieldStyle', subKey: 'imageLogo.fieldStyleSub', icon: 'edit', options: styles },
  { key: 'colorTone', step: '03', labelKey: 'imageLogo.fieldColorTone', subKey: 'imageLogo.fieldColorToneSub', icon: 'photo-o', options: colorTones },
  { key: 'theme', step: '04', labelKey: 'imageLogo.fieldTheme', subKey: 'imageLogo.fieldThemeSub', icon: 'star-o', options: themes },
  { key: 'layout', step: '05', labelKey: 'imageLogo.fieldLayout', subKey: 'imageLogo.fieldLayoutSub', icon: 'apps-o', options: layouts },
]

const colorPreviewMap = {
  warm: 'linear-gradient(145deg, #fff5f0 0%, #ffe8cc 45%, #ffd6a5 100%)',
  cool: 'linear-gradient(145deg, #eef6ff 0%, #dbeafe 45%, #c7d2fe 100%)',
  monochrome: 'linear-gradient(145deg, #f5f5f5 0%, #e5e5e5 45%, #fafafa 100%)',
  vibrant: 'linear-gradient(145deg, #fff0f6 0%, #fce7f3 45%, #fbcfe8 100%)',
  pastel: 'linear-gradient(145deg, #fdf4ff 0%, #fae8ff 45%, #ede9fe 100%)',
  dark: 'linear-gradient(145deg, #1f2937 0%, #111827 45%, #374151 100%)',
  gold: 'linear-gradient(145deg, #fffbeb 0%, #fef3c7 45%, #fde68a 100%)',
  bluegreen: 'linear-gradient(145deg, #ecfdf5 0%, #d1fae5 45%, #a7f3d0 100%)',
}

const selections = ref({
  industry: 'tech',
  style: 'minimalist',
  colorTone: 'vibrant',
  theme: 'innovation',
  layout: '1x1',
})
const brandName = ref('')
const submitting = ref(false)

const activeLayout = computed(() => {
  return layouts.find((item) => item.value === selections.value.layout) || layouts[0]
})

const previewGradient = computed(() => colorPreviewMap[selections.value.colorTone] || colorPreviewMap.vibrant)

function findLabel(options, value) {
  const item = options.find((o) => o.value === value)
  return item ? t(item.labelKey) : ''
}

const summaryTags = computed(() => [
  { key: 'industry', icon: 'apps-o', value: findLabel(industries, selections.value.industry) },
  { key: 'style', icon: 'edit', value: findLabel(styles, selections.value.style) },
  { key: 'color', icon: 'photo-o', value: findLabel(colorTones, selections.value.colorTone) },
  { key: 'theme', icon: 'star-o', value: findLabel(themes, selections.value.theme) },
  { key: 'layout', icon: 'apps-o', value: findLabel(layouts, selections.value.layout) },
])

const industryPrompts = {
  tech: 'technology and software',
  finance: 'finance and banking',
  food: 'food and beverage',
  fashion: 'fashion and apparel',
  education: 'education and learning',
  health: 'healthcare and wellness',
  realestate: 'real estate and property',
  entertainment: 'entertainment and media',
  travel: 'travel and hospitality',
  sports: 'sports and fitness',
}

const stylePrompts = {
  minimalist: 'minimalist and clean',
  modern: 'modern and sleek',
  geometric: 'geometric and abstract',
  handdrawn: 'hand-drawn and organic',
  luxury: 'luxury and premium',
  playful: 'playful and friendly',
  corporate: 'corporate and professional',
  vintage: 'vintage and classic',
}

const colorPrompts = {
  warm: 'warm tones such as red, orange and gold',
  cool: 'cool tones such as blue, cyan and purple',
  monochrome: 'monochrome black and white',
  vibrant: 'vibrant and bold colors',
  pastel: 'soft pastel colors',
  dark: 'dark and moody palette',
  gold: 'gold and metallic accents',
  bluegreen: 'blue-green gradient palette',
}

const themePrompts = {
  innovation: 'innovation and future-forward',
  nature: 'nature and sustainability',
  trust: 'trust and reliability',
  speed: 'speed and efficiency',
  community: 'community and connection',
  elegance: 'elegance and sophistication',
  energy: 'energy and dynamism',
  global: 'global and international',
}

function buildBrandBrief() {
  const industry = industryPrompts[selections.value.industry] || industryPrompts.tech
  const style = stylePrompts[selections.value.style] || stylePrompts.minimalist
  const colorTone = colorPrompts[selections.value.colorTone] || colorPrompts.vibrant
  const theme = themePrompts[selections.value.theme] || themePrompts.innovation
  const brand = brandName.value.trim()
  const brandLine = brand ? `Brand name to incorporate: "${brand}". ` : ''
  return {
    industry,
    style,
    colorTone,
    theme,
    brandLine,
    sharedBrief:
      `Industry: ${industry}. Visual style: ${style}. Color tone: ${colorTone}. Creative theme: ${theme}. ${brandLine}`,
  }
}

function buildGridCellPrompts(layout, sharedBrief) {
  const cells = []
  for (let i = 0; i < layout.count; i++) {
    const position = gridPositions[i] || `cell-${i + 1}`
    const variation = cellVariationPrompts[i % cellVariationPrompts.length]
    cells.push(
      `Grid cell ${i + 1} (${position}): ${sharedBrief} Design direction: ${variation}.`
    )
  }
  return cells
}

const synthesizedPrompt = computed(() => {
  const brief = buildBrandBrief()
  const layout = activeLayout.value

  if (layout.count <= 1) {
    return (
      `Design a professional brand logo mark for the ${brief.industry} industry. ` +
      `Visual style: ${brief.style}. Color tone: ${brief.colorTone}. Creative theme: ${brief.theme}. ` +
      `${brief.brandLine}` +
      'Requirements: vector-style flat logo icon, clean composition, centered on white background, ' +
      'iconic symbol design, no mockup scene, no watermark, high quality, suitable for commercial branding.'
    )
  }

  const cellPrompts = buildGridCellPrompts(layout, brief.sharedBrief)
  return (
    `Create ONE single square image containing a clean ${layout.rows}x${layout.cols} grid collage with exactly ${layout.count} logo cells, ` +
    'evenly spaced with thin white dividers between cells. Each cell must show a different logo variation for the same brand brief. ' +
    'Shared brand brief for all cells: ' +
    brief.sharedBrief +
    ' Per-cell logo directions: ' +
    cellPrompts.join(' ') +
    ' Global requirements: each cell shows one centered logo on white background within its grid cell, ' +
    'consistent overall style and color tone across all cells but each design must be visually distinct, ' +
    'professional logo exploration sheet layout, vector-style flat logos, no mockup scenes, no watermarks, high quality.'
  )
})

const params = ref({
  client_id: randString(10),
  task_type: 'image',
  provider: 'image2',
  nano_mode: 'txt2img',
  rate: '1:1',
  model: 'gpt-image-2',
  chaos: 0,
  stylize: 0,
  seed: 0,
  img_arr: [],
  raw: false,
  iw: 0,
  prompt: '',
  neg_prompt: '',
  tile: false,
  quality: 0,
  cref: '',
  sref: '',
  cw: 0,
})

const userId = ref(0)
const runningJobs = ref([])
const finishedJobs = ref([])
const socket = ref(null)
const power = ref(0)
const mjPower = ref(1)
const loadingDots = ref(0)
const loading = ref(false)
const finished = ref(false)
const error = ref(false)
const page = ref(0)
const pageSize = ref(10)

let loadingTimer = null
let refreshTimer = null

onMounted(() => {
  void loadAppFlags()
  checkSession()
    .then((user) => {
      power.value = user.power
      userId.value = user.id
      fetchRunningJobs()
      fetchFinishJobs(1)
      try {
        connect()
      } catch (e) {
        console.warn('logo websocket connect failed', e)
      }
      startAutoRefresh()
    })
    .catch(() => {
      router.push('/login')
    })

  loadingTimer = setInterval(() => {
    loadingDots.value = (loadingDots.value + 1) % 4
  }, 500)
})

onUnmounted(() => {
  socket.value = null
  if (loadingTimer) {
    clearInterval(loadingTimer)
    loadingTimer = null
  }
  stopAutoRefresh()
})

httpGet('/api/config/get?key=system')
  .then((res) => {
    mjPower.value = res.data.mj_power2 || res.data.mj_power || 1
  })
  .catch((e) => {
    showNotify({ type: 'danger', message: t('imageMj.configLoadFailPrefix') + e.message })
  })

const heartbeatHandle = ref(null)

function connect() {
  const host = resolveWsHost()

  const sendHeartbeat = () => {
    clearTimeout(heartbeatHandle.value)
    Promise.resolve()
      .then(() => {
        if (socket.value !== null) {
          socket.value.send(JSON.stringify({ type: 'heartbeat', content: 'ping' }))
        }
      })
      .then(() => {
        heartbeatHandle.value = setTimeout(() => sendHeartbeat(), 5000)
      })
  }

  const _socket = new WebSocket(`${host}/api/mj/client?user_id=${userId.value}`)
  _socket.addEventListener('open', () => {
    socket.value = _socket
    sendHeartbeat()
  })
  _socket.addEventListener('message', () => {
    fetchRunningJobs()
    fetchFinishJobs(1)
    if (!refreshTimer && runningJobs.value.length > 0) {
      startAutoRefresh()
    }
  })
  _socket.addEventListener('close', () => {
    if (socket.value !== null) {
      connect()
    }
  })
}

function startAutoRefresh() {
  stopAutoRefresh()
  refreshTimer = setInterval(() => {
    if (runningJobs.value.length > 0) {
      fetchRunningJobs()
      fetchFinishJobs(1)
    }
  }, 3000)
}

function stopAutoRefresh() {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

function fetchRunningJobs() {
  httpGet('/api/mj/jobs?finish=0&app_source=logo')
    .then((res) => {
      const jobs = unwrapMjJobList(res)
      const _jobs = []
      for (let i = 0; i < jobs.length; i++) {
        if (jobs[i].progress === -1) {
          showNotify({
            message: t('imageMj.jobFailPrefix') + (jobs[i].err_msg || ''),
            type: 'danger',
          })
          power.value += mjPower.value
          continue
        }
        if (jobs[i].progress >= 100) {
          fetchFinishJobs(1)
          continue
        }
        _jobs.push(jobs[i])
      }
      runningJobs.value = _jobs
      if (_jobs.length === 0) {
        stopAutoRefresh()
      } else if (!refreshTimer) {
        startAutoRefresh()
      }
    })
    .catch((e) => {
      showNotify({ type: 'danger', message: t('imageMj.fetchJobsFailPrefix') + e.message })
    })
}

function fetchFinishJobs(pageNo) {
  loading.value = true
  httpGet(`/api/mj/jobs?finish=1&page=${pageNo}&page_size=${pageSize.value}&app_source=logo`)
    .then((res) => {
      const jobs = res.data.items || res.data
      for (let i = 0; i < jobs.length; i++) {
        if (jobs[i].progress === -1) {
          power.value += mjPower.value
          continue
        }
        applyThumbUrl(jobs[i])
      }
      if (jobs.length < pageSize.value) {
        finished.value = true
      }
      if (pageNo === 1) {
        finishedJobs.value = jobs
      } else {
        finishedJobs.value = finishedJobs.value.concat(jobs)
      }
      nextTick(() => {
        loading.value = false
      })
    })
    .catch((e) => {
      loading.value = false
      error.value = true
      showFailToast(t('imageMj.fetchJobsFailPrefix') + e.message)
    })
}

const onLoad = () => {
  page.value += 1
  fetchFinishJobs(page.value)
}

function generate() {
  if (logoMaintenance.value) {
    showToast(`${t('imageApps.ailogo')}${t('imageApps.maintenance')}`)
    return
  }
  submitting.value = true
  params.value.prompt = `${LOGO_PROMPT_MARKER} ${synthesizedPrompt.value}`
  params.value.nano_mode = 'txt2img'
  params.value.task_type = 'image'
  params.value.session_id = getSessionId()
  params.value.img_arr = []

  const payload = { ...params.value, app_source: 'logo' }
  console.log('[ImageLogo] 完整提示词:', payload.prompt)
  console.log('[ImageLogo] 完整提交参数:', JSON.parse(JSON.stringify(payload)))

  httpPost('/api/mj/image', payload)
    .then(() => {
      showToast(t('imageLogo.pushSuccess'))
      power.value -= mjPower.value
      fetchRunningJobs()
    })
    .catch((e) => {
      showFailToast(t('imageMj.sendTaskFailPrefix') + e.message)
    })
    .finally(() => {
      submitting.value = false
    })
}

function removeImage(item) {
  showConfirmDialog({
    title: t('imageMj.deleteTitle'),
    message: t('imageMj.deleteMessage'),
  })
    .then(() => {
      httpPost('/api/mj/remove', { id: item.id, user_id: item.user_id || userId.value }).then(() => {
        showSuccessToast(t('imageMj.deleteSuccess'))
        fetchFinishJobs(1)
      })
    })
    .catch(() => {
      showToast(t('imageMj.cancelOperation'))
    })
}

function showPrompt(item) {
  showDialog({
    title: t('imageLogo.promptDialogTitle'),
    message: item.prompt?.replace(LOGO_PROMPT_MARKER, '').trim() || item.prompt,
  }).catch(() => {})
}

function imageView(item) {
  const url = resolveMjImageUrl(item.img_url, item.org_url)
  if (url) showImagePreview([url])
}
</script>

<style lang="stylus" scoped>
.image-logo
  --ai-bg: #f4f5f7
  --ai-surface: #ffffff
  --ai-text: #111827
  --ai-muted: #6b7280
  --ai-line: rgba(17, 24, 39, 0.08)
  --ai-accent: #0f766e
  position relative
  min-height 100vh
  min-height 100dvh
  background: radial-gradient(ellipse 80% 45% at 0% 0%, rgba(30, 58, 138, 0.07), transparent 55%), radial-gradient(ellipse 70% 40% at 100% 0%, rgba(190, 24, 93, 0.05), transparent 50%), #f4f5f7
  color var(--ai-text)
  padding-bottom calc(88px + env(safe-area-inset-bottom, 0px))
  font-family 'Plus Jakarta Sans', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif
  -webkit-font-smoothing antialiased

.image-logo__bg
  position fixed
  inset 0
  overflow hidden
  pointer-events none
  z-index 0

.image-logo__orb
  position absolute
  border-radius 50%
  filter blur(60px)
  opacity 0.35

.image-logo__orb--1
  width 280px
  height 280px
  top -40px
  left -60px
  background #93c5fd

.image-logo__orb--2
  width 240px
  height 240px
  top 120px
  right -80px
  background #99f6e4

.image-logo__orb--3
  width 200px
  height 200px
  bottom 120px
  left 20%
  background #fbcfe8

.image-logo__maintenance
  display flex
  align-items flex-start
  gap 10px
  margin 12px 16px 0
  padding 12px 14px
  border-radius 12px
  background #fffbeb
  border 1px solid #fcd34d
  font-size 13px
  line-height 1.5
  color #92400e

.image-logo__header
  position sticky
  top 0
  z-index 30
  display flex
  align-items center
  justify-content space-between
  gap 8px
  padding 10px 12px
  background rgba(255, 255, 255, 0.92)
  backdrop-filter blur(16px)
  border-bottom 1px solid var(--ai-line)

.image-logo__header-btn
  display flex
  align-items center
  justify-content center
  width 40px
  height 40px
  border none
  border-radius 12px
  background #f3f4f6
  color #374151

.image-logo__header-center
  flex 1
  display flex
  flex-direction column
  align-items center
  gap 4px

.image-logo__header-title
  font-size 16px
  font-weight 800
  letter-spacing (-0.02em)
  color var(--ai-text)

.image-logo__header-badge
  font-size 10px
  font-weight 600
  padding 2px 8px
  border-radius 999px
  color var(--ai-accent)
  background rgba(15, 118, 110, 0.1)
  border 1px solid rgba(15, 118, 110, 0.2)

.image-logo__body
  position relative
  z-index 1
  padding 16px 16px 24px

.image-logo__hero
  display flex
  align-items flex-start
  gap 14px
  margin-bottom 16px

.image-logo__hero-icon
  flex-shrink 0
  width 52px
  height 52px
  display flex
  align-items center
  justify-content center
  border-radius 16px
  background linear-gradient(135deg, #1e3a8a 0%, #be185d 100%)
  box-shadow 0 10px 24px rgba(30, 58, 138, 0.28)

.image-logo__hero-title
  margin 0 0 6px
  font-size 18px
  font-weight 800
  line-height 1.35
  letter-spacing (-0.02em)
  color var(--ai-text)

.image-logo__hero-desc
  margin 0
  font-size 13px
  line-height 1.6
  color var(--ai-muted)

.image-logo__preview-card
  margin-bottom 18px
  padding 14px
  border-radius 18px
  background var(--ai-surface)
  border 1px solid var(--ai-line)
  box-shadow 0 1px 3px rgba(17, 24, 39, 0.04), 0 8px 24px rgba(17, 24, 39, 0.04)

.image-logo__canvas
  border-radius 16px
  padding 14px
  transition background 0.35s ease

.image-logo__canvas-inner
  display flex
  flex-direction column
  align-items center
  justify-content center
  min-height 148px
  border-radius 14px
  background rgba(255, 255, 255, 0.92)
  box-shadow inset 0 1px 0 rgba(255, 255, 255, 0.8)

.image-logo__canvas-mark
  position relative
  width 72px
  height 72px
  margin-bottom 10px

.image-logo__canvas-shape
  position absolute
  border-radius 14px
  background linear-gradient(135deg, #1e3a8a, #0f766e)

  &:first-child
    width 42px
    height 42px
    top 0
    left 15px
    opacity 0.95

  &--2
    width 24px
    height 24px
    bottom 8px
    left 0
    border-radius 50%
    background linear-gradient(135deg, #be185d, #f97316)
    opacity 0.85

  &--3
    width 18px
    height 18px
    bottom 0
    right 8px
    border-radius 6px
    background linear-gradient(135deg, #10b981, #06b6d4)
    opacity 0.8

.image-logo__canvas-brand
  margin 0
  font-size 18px
  font-weight 800
  letter-spacing 0.04em
  color #111827

.image-logo__canvas-hint
  margin 0
  font-size 12px
  color #9ca3af

.image-logo__canvas-grid
  display grid
  gap 6px
  width min(100%, 168px)
  aspect-ratio 1
  margin-bottom 10px
  padding 8px
  border-radius 12px
  background rgba(15, 118, 110, 0.06)
  border 1px dashed rgba(15, 118, 110, 0.22)

.image-logo__canvas-cell
  display flex
  align-items center
  justify-content center
  border-radius 8px
  background rgba(255, 255, 255, 0.95)
  border 1px solid rgba(15, 118, 110, 0.12)
  box-shadow inset 0 1px 0 rgba(255, 255, 255, 0.8)

.image-logo__canvas-cell-dot
  width 10px
  height 10px
  border-radius 50%
  background linear-gradient(135deg, #1e3a8a, #0f766e)

.image-logo__canvas-layout
  margin 8px 0 0
  font-size 11px
  font-weight 600
  color var(--ai-accent)

.image-logo__summary
  display flex
  flex-wrap wrap
  gap 8px
  margin-top 12px

.image-logo__summary-tag
  display inline-flex
  align-items center
  gap 4px
  padding 6px 10px
  border-radius 999px
  font-size 11px
  font-weight 600
  color #374151
  background #f3f4f6
  border 1px solid var(--ai-line)

.image-logo__section
  margin-bottom 14px
  padding 14px
  border-radius 18px
  background var(--ai-surface)
  border 1px solid var(--ai-line)
  box-shadow 0 1px 3px rgba(17, 24, 39, 0.04)

.image-logo__section-head
  display flex
  align-items center
  gap 10px
  margin-bottom 12px

.image-logo__section-step
  flex-shrink 0
  width 34px
  height 34px
  display flex
  align-items center
  justify-content center
  border-radius 10px
  font-size 11px
  font-weight 800
  color var(--ai-accent)
  background rgba(15, 118, 110, 0.1)

.image-logo__section-meta
  flex 1
  min-width 0

.image-logo__section-title
  display block
  font-size 14px
  font-weight 700
  color var(--ai-text)

.image-logo__section-sub
  display block
  margin-top 2px
  font-size 11px
  color var(--ai-muted)

.image-logo__section-icon
  flex-shrink 0
  color #9ca3af

.image-logo__chip-scroll
  display flex
  gap 8px
  overflow-x auto
  padding-bottom 2px
  -webkit-overflow-scrolling touch
  scrollbar-width none

  &::-webkit-scrollbar
    display none

.image-logo__chip
  flex-shrink 0
  display inline-flex
  align-items center
  gap 6px
  border none
  border-radius 14px
  padding 10px 14px
  font-size 13px
  font-weight 600
  color #4b5563
  background #f3f4f6
  border 1px solid transparent
  transition all 0.22s ease

  &--active
    color #fff
    background #111827
    border-color transparent
    box-shadow 0 6px 16px rgba(17, 24, 39, 0.16)
    transform translateY(-1px)

  &--color
    padding-left 10px

  &--layout
    flex-direction column
    align-items center
    gap 8px
    min-width 72px
    padding 12px 14px

.image-logo__chip-grid
  display grid
  gap 3px
  width 34px
  height 34px
  padding 4px
  border-radius 8px
  background rgba(17, 24, 39, 0.06)
  border 1px solid rgba(17, 24, 39, 0.08)

  i
    display block
    border-radius 2px
    background rgba(17, 24, 39, 0.35)

.image-logo__chip--active .image-logo__chip-grid
  background rgba(255, 255, 255, 0.18)
  border-color rgba(255, 255, 255, 0.35)

  i
    background rgba(255, 255, 255, 0.92)

.image-logo__chip-scroll--layout
  padding-bottom 4px

.image-logo__chip-swatch
  width 16px
  height 16px
  border-radius 50%
  border 2px solid rgba(255, 255, 255, 0.85)
  box-shadow 0 2px 6px rgba(0, 0, 0, 0.12)

.image-logo__chip-text
  white-space nowrap

.image-logo__brand-wrap
  display flex
  align-items center
  gap 10px
  padding 12px 14px
  border-radius 14px
  background #f9fafb
  border 1px solid var(--ai-line)

.image-logo__brand-icon
  flex-shrink 0
  color #9ca3af

.image-logo__brand-input
  flex 1
  min-width 0
  border none
  outline none
  background transparent
  font-size 15px
  font-weight 600
  color var(--ai-text)

  &::placeholder
    color #9ca3af
    font-weight 500

.image-logo__brand-clear
  flex-shrink 0
  display flex
  align-items center
  justify-content center
  width 28px
  height 28px
  border none
  border-radius 50%
  background #e5e7eb
  color #6b7280

.image-logo__footer-spacer
  height 8px

.image-logo__footer
  position fixed
  left 0
  right 0
  bottom 0
  z-index 40
  padding 10px 16px calc(10px + env(safe-area-inset-bottom, 0px))
  background rgba(244, 245, 247, 0.94)
  backdrop-filter blur(18px)
  border-top 1px solid var(--ai-line)
  box-shadow 0 -8px 24px rgba(17, 24, 39, 0.06)

.image-logo__footer-inner
  display flex
  align-items center
  gap 12px

.image-logo__footer-info
  flex 1
  min-width 0
  display flex
  flex-direction column
  gap 2px

.image-logo__footer-label
  font-size 13px
  font-weight 700
  color var(--ai-text)

.image-logo__footer-power
  font-size 11px
  color var(--ai-muted)

.image-logo__footer-btn
  flex-shrink 0
  min-width 148px
  height 46px
  border none
  border-radius 14px
  background #111827 !important
  border-color #111827 !important
  box-shadow 0 8px 20px rgba(17, 24, 39, 0.2)

  :deep(.van-button__content)
    gap 6px
    font-weight 700

.image-logo__panel
  margin-top 16px
  padding 16px
  border-radius 18px
  background var(--ai-surface)
  border 1px solid var(--ai-line)
  box-shadow 0 1px 3px rgba(17, 24, 39, 0.04)

.image-logo__panel-head
  display flex
  align-items flex-start
  justify-content space-between
  gap 12px
  margin-bottom 14px

.image-logo__panel-title
  margin 0
  font-size 16px
  font-weight 800
  color var(--ai-text)

.image-logo__panel-sub
  margin 4px 0 0
  font-size 12px
  color var(--ai-muted)

.image-logo__panel-empty
  display flex
  flex-direction column
  align-items center
  gap 8px
  padding 24px 12px
  border-radius 16px
  background #f9fafb
  font-size 13px
  color var(--ai-muted)

.image-logo__running-list
  display flex
  flex-direction column
  gap 12px

.image-logo__running-item
  display flex
  align-items center
  gap 12px
  padding 10px
  border-radius 16px
  background #f9fafb
  border 1px solid var(--ai-line)

.image-logo__running-thumb
  position relative
  flex-shrink 0
  width 72px
  height 72px
  border-radius 12px
  overflow hidden

.image-logo__running-progress,
.image-logo__running-queue
  position absolute
  inset 0
  display flex
  flex-direction column
  align-items center
  justify-content center
  gap 6px
  background rgba(17, 24, 39, 0.45)

.image-logo__running-dots
  display flex
  gap 4px

  i
    width 5px
    height 5px
    border-radius 50%
    background rgba(255, 255, 255, 0.35)
    transition background 0.2s ease

    &.active
      background #fff

.image-logo__running-label
  display block
  font-size 14px
  font-weight 700
  color var(--ai-text)

.image-logo__running-hint
  display block
  margin-top 2px
  font-size 12px
  color var(--ai-muted)

.image-logo__history-empty
  :deep(.van-empty__description)
    color var(--ai-muted)

.image-logo__history-grid
  display grid
  grid-template-columns repeat(2, minmax(0, 1fr))
  gap 12px

.image-logo__history-card
  position relative
  border-radius 16px
  overflow hidden
  background #f9fafb
  border 1px solid var(--ai-line)

.image-logo__history-thumb
  position relative
  display block
  width 100%
  aspect-ratio 1
  padding 0
  border none
  background transparent

  :deep(.van-image)
    width 100%
    height 100%

.image-logo__history-zoom
  position absolute
  top 8px
  right 8px
  width 28px
  height 28px
  display flex
  align-items center
  justify-content center
  border-radius 50%
  background rgba(0, 0, 0, 0.45)
  backdrop-filter blur(4px)
  color #fff

.image-logo__history-actions
  display flex
  gap 8px
  padding 10px

.image-logo__history-btn
  flex 1
  display flex
  align-items center
  justify-content center
  height 34px
  border none
  border-radius 10px
  color #374151
  background #e5e7eb

  &--danger
    background #fef2f2
    color #dc2626
</style>
