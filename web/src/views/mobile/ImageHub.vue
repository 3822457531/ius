<template>
  <div class="image-hub">
    <header class="image-hub__header">
      <p class="image-hub__greet">{{ t(greetingKey) }}</p>
      <p class="image-hub__date">{{ dateLine }}</p>
      <p class="image-hub__subtitle">{{ t('imageHub.subtitle') }}</p>
      <!-- <button type="button" class="image-hub__search-hint" @click="onSearchHint">
        {{ t('imageHub.searchHint') }}
      </button> -->
    </header>

    <!-- <div class="image-hub__floating">
      <div class="image-hub__float-row">
        <button type="button" class="hub-tile" @click="openMj">
          <span class="hub-tile__icon hub-tile__icon--mj">
            <van-icon name="photo-o" size="28" color="#fff" />
          </span>
          <span class="hub-tile__label">{{ t('imageTabs.tabImage') }}</span>
        </button>
        <button type="button" class="hub-tile" @click="openMusic">
          <span class="hub-tile__icon hub-tile__icon--music">
            <van-icon name="music-o" size="28" color="#fff" />
          </span>
          <span class="hub-tile__label">{{ t('imageTabs.tabMusic') }}</span>
        </button>
        <button type="button" class="hub-tile hub-tile--muted" @click="soon">
          <span class="hub-tile__icon hub-tile__icon--tasks">
            <van-icon name="todo-list-o" size="26" color="#fff" />
          </span>
          <span class="hub-tile__label">{{ t('imageApps.tasks') }}</span>
        </button>
      </div>
      <div class="image-hub__float-row image-hub__float-row--tight">
        <button type="button" class="hub-tile hub-tile--muted" @click="soon">
          <span class="hub-tile__icon hub-tile__icon--gallery">
            <van-icon name="photograph" size="26" color="#fff" />
          </span>
          <span class="hub-tile__label">{{ t('imageApps.nb2v') }}</span>
        </button>
      </div>
    </div> -->

    <div class="image-hub__card">
      <div class="image-hub__grid">
        <button type="button" class="hub-cell hub-cell--live" @click="openAiSearch">
          <span class="hub-cell__icon hub-cell__icon--search">
            <van-icon name="search" size="28" color="#fff" />
          </span>
          <span class="hub-cell__label">{{ t('imageApps.aiSearch') }}</span>
        </button>
        <button type="button" class="hub-cell hub-cell--live" @click="openMj">
          <span class="hub-cell__icon hub-cell__icon--a">
            <van-icon name="photo-o" size="28" color="#fff" />
          </span>
          <span class="hub-cell__label">{{ t('imageApps.aiImage') }}</span>
        </button>
        <button type="button" class="hub-cell hub-cell--live" @click="openMusic">
          <span class="hub-cell__icon hub-cell__icon--b">
            <van-icon name="music-o" size="28" color="#fff" />
          </span>
          <span class="hub-cell__label">{{ t('imageApps.aiMusic') }}</span>
        </button>
        <!-- 视频暂下线
        <button type="button" class="hub-cell hub-cell--live" @click="openVideo">
          <span class="hub-cell__icon hub-cell__icon--video">
            <van-icon name="video-o" size="28" color="#fff" />
          </span>
          <span class="hub-cell__label">{{ t('imageApps.aiVideo') }}</span>
        </button>
        -->

        <button type="button" class="hub-cell hub-cell--muted" @click="openPpt">
          <span class="hub-cell__icon hub-cell__icon--ppt">
            <van-icon name="description" size="28" color="#fff" />
          </span>
          <span class="hub-cell__label">{{ t('imageApps.aiPpt') }}</span>
        </button>
<!--         
        <button type="button" class="hub-cell hub-cell--muted" @click="soon">
          <span class="hub-cell__icon hub-cell__icon--douyin">
            <van-icon name="live" size="28" color="#fff" />
          </span>
          <span class="hub-cell__label">{{ t('imageApps.aiDouyin') }}</span>
        </button> -->


        <button type="button" class="hub-cell hub-cell--live" @click="openLogo">
          <span class="hub-cell__icon hub-cell__icon--logo">
            <van-icon name="apps-o" size="28" color="#fff" />
          </span>
          <span class="hub-cell__label">{{ t('imageApps.ailogo') }}</span>
        </button>

        <!-- 文字故事暂下线
        <button type="button" class="hub-cell hub-cell--live" @click="openDrama">
          <span class="hub-cell__icon hub-cell__icon--douyin">
            <van-icon name="live" size="28" color="#fff" />
          </span>
          <span class="hub-cell__label">{{ t('imageApps.aiDouyin') }}</span>
        </button>
        -->
        
      </div>



      <h3 class="image-hub__section-title">{{ t('imageHub.customAppsTitle') }}</h3>
      <div class="image-hub__grid image-hub__grid--custom">
        <!-- <button
          v-for="(item, idx) in placeholderApps"
          :key="'ph-' + idx"
          type="button"
          class="hub-cell hub-cell--muted"
          @click="soon"
        >
          <span class="hub-cell__icon" :class="item.iconClass">
            <van-icon :name="item.icon" size="26" color="#fff" />
          </span>
          <span class="hub-cell__label">{{ item.label }}</span>
        </button> -->

        <div
          v-for="item in userCustomApps"
          :key="item.id"
          class="hub-grid-slot"
        >
          <div class="hub-custom-item">
            <button
              type="button"
              class="hub-cell hub-cell--muted"
              @click="openCustomLink(item)"
            >
              <span class="hub-cell__icon hub-cell__icon--custom">
                <van-icon :name="item.icon || 'link-o'" size="26" color="#fff" />
              </span>
              <span class="hub-cell__label">{{ item.label }}</span>
            </button>
            <div class="hub-custom-toolbar">
              <button
                type="button"
                class="hub-custom-tool"
                :aria-label="t('imageHub.editCustom')"
                @click.stop.prevent="openEditCustom(item)"
              >
                <van-icon name="edit" size="15" color="#1a6cb5" />
              </button>
              <button
                type="button"
                class="hub-custom-tool hub-custom-tool--danger"
                :aria-label="t('imageHub.removeCustom')"
                @click.stop.prevent="confirmDeleteCustom(item)"
              >
                <van-icon name="delete-o" size="15" color="#ee0a24" />
              </button>
            </div>
          </div>
        </div>

        <button type="button" class="hub-cell hub-cell--add" @click="openAddCustom">
          <span class="hub-cell__icon hub-cell__icon--add">
            <van-icon name="plus" size="28" color="#4facfe" />
          </span>
          <span class="hub-cell__label">{{ t('imageHub.addCustom') }}</span>
        </button>
      </div>
    </div>

    <van-dialog
      v-model:show="showAddCustom"
      :title="editingCustomId ? t('imageHub.dialogEditTitle') : t('imageHub.dialogAddTitle')"
      show-cancel-button
      :confirm-button-text="t('imageHub.dialogConfirm')"
      :cancel-button-text="t('imageHub.dialogCancel')"
      :before-close="beforeCloseAddCustom"
    >
      <div class="image-hub__dialog-fields">
        <van-field
          v-model="customForm.label"
          :label="t('imageHub.fieldCustomName')"
          :placeholder="t('imageHub.placeholderCustomName')"
          maxlength="20"
        />
        <van-field
          v-model="customForm.url"
          :label="t('imageHub.fieldCustomUrl')"
          :placeholder="t('imageHub.placeholderCustomUrl')"
        />
        <div class="image-hub__open-mode">
          <div class="image-hub__open-mode-label">{{ t('imageHub.fieldOpenMode') }}</div>
          <div class="image-hub__open-mode-row">
            <button
              type="button"
              :class="['image-hub__mode-btn', { 'image-hub__mode-btn--active': customForm.openMode === 'embed' }]"
              @click="customForm.openMode = 'embed'"
            >
              {{ t('imageHub.openModeEmbed') }}
            </button>
            <button
              type="button"
              :class="['image-hub__mode-btn', { 'image-hub__mode-btn--active': customForm.openMode === 'new' }]"
              @click="customForm.openMode = 'new'"
            >
              {{ t('imageHub.openModeNew') }}
            </button>
          </div>

          <!-- <p v-if="customForm.openMode === 'embed'" class="image-hub__open-mode-hint">
            {{ t('imageHub.openModeEmbedHint') }}
          </p> -->

        </div>
      </div>
    </van-dialog>

    <van-popup
      v-model:show="showEmbedPopup"
      position="bottom"
      :z-index="EMBED_POPUP_Z_INDEX"
      :style="embedPopupPanelStyle"
      class="image-hub__embed-popup"
      closeable
      close-icon-position="top-right"
      :safe-area-inset-top="true"
      :safe-area-inset-bottom="true"
      teleport="body"
      @closed="onEmbedClosed"
    >
      <div class="image-hub__embed-bar">
        <span class="image-hub__embed-title">{{ embedTitle }}</span>
      </div>
      <iframe
        v-if="embedUrl"
        ref="embedFrameRef"
        :key="embedUrl"
        class="image-hub__embed-frame"
        :src="embedUrl"
        :title="embedTitle"
        sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox allow-downloads"
        referrerpolicy="no-referrer-when-downgrade"
      />
    </van-popup>

    <!-- <footer class="image-hub__footer">
      <p class="image-hub__quick-title">{{ t('imageHub.quickTitle') }}</p>
      <div class="image-hub__quick-row">
        <button type="button" class="image-hub__text-link" @click="openMj">
          {{ t('imageHub.quickImage') }}
        </button>
        <span class="image-hub__sep" aria-hidden="true">{{ t('imageHub.quickSep') }}</span>
        <button type="button" class="image-hub__text-link" @click="openMusic">
          {{ t('imageHub.quickMusic') }}
        </button>
      </div>
      <div class="image-hub__tips">
        <p class="image-hub__tips-title">{{ t('imageHub.tipTitle') }}</p>
        <p class="image-hub__tips-line">{{ t('imageHub.tipLine1') }}</p>
        <p class="image-hub__tips-line">{{ t('imageHub.tipLine2') }}</p>
      </div>
    </footer> -->
  </div>
</template>

<script setup>
import { computed, ref, reactive, onMounted, inject, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { IMAGE_APP_LAUNCH_KEY, IMAGE_EMBED_LAUNCH_KEY } from '@/constants/imageAppLaunch'

/** 低于 Image.vue 加载层 z-index，避免盖住「正在连接服务」卡片 */
const EMBED_POPUP_Z_INDEX = 3000
import { showToast, showConfirmDialog } from 'vant'
import { httpGet, httpPost } from '@/utils/http'

const CUSTOM_APPS_MAX = 15

const router = useRouter()
const { t, locale } = useI18n()
const runAppLaunch = inject(IMAGE_APP_LAUNCH_KEY, null)
const runWithLaunchOverlay = inject(IMAGE_EMBED_LAUNCH_KEY, null)
const embedFrameRef = ref(null)

const greetingKey = computed(() => {
  const h = new Date().getHours()
  if (h < 12) return 'imageHub.greetMorning'
  if (h < 18) return 'imageHub.greetAfternoon'
  return 'imageHub.greetEvening'
})

const dateLine = computed(() => {
  const map = {
    'zh-CN': 'zh-CN',
    'en-US': 'en-US',
    'ja-JP': 'ja-JP',
    'kr-KR': 'ko-KR',
  }
  const loc = map[locale.value] || 'en-US'
  try {
    return new Intl.DateTimeFormat(loc, {
      weekday: 'long',
      month: 'long',
      day: 'numeric',
    }).format(new Date())
  } catch {
    return new Intl.DateTimeFormat('en-US', {
      weekday: 'long',
      month: 'long',
      day: 'numeric',
    }).format(new Date())
  }
})

function onSearchHint() {
  showToast(t('imageHub.searchSoon'))
}

function openAiSearch() {
  if (typeof runAppLaunch === 'function') {
    void runAppLaunch(t('imageApps.aiSearch'), 'mobile-ai-search')
    return
  }
  router.push({ name: 'mobile-ai-search' })
}

function openMj() {
  if (typeof runAppLaunch === 'function') {
    void runAppLaunch(t('imageApps.aiImage'), 'mobile-image-mj')
    return
  }
  router.push({ name: 'mobile-image-mj' })
}

function openMusic() {
  if (typeof runAppLaunch === 'function') {
    void runAppLaunch(t('imageApps.aiMusic'), 'mobile-image-music')
    return
  }
  router.push({ name: 'mobile-image-music' })
}

// 视频暂下线
// function openVideo() {
//   if (typeof runAppLaunch === 'function') {
//     void runAppLaunch(t('imageApps.aiVideo'), 'mobile-image-video')
//     return
//   }
//   router.push({ name: 'mobile-image-video' })
// }

function openLogo() {
  if (typeof runAppLaunch === 'function') {
    void runAppLaunch(t('imageApps.ailogo'), 'mobile-image-logo')
    return
  }
  router.push({ name: 'mobile-image-logo' })
}

// 文字故事暂下线
// function openDrama() {
//   if (typeof runAppLaunch === 'function') {
//     void runAppLaunch(t('imageApps.aiDouyin'), 'mobile-image-drama')
//     return
//   }
//   router.push({ name: 'mobile-image-drama' })
// }

function waitForEmbedIframeLoad() {
  return new Promise((resolve) => {
    const bind = () => {
      const iframe = embedFrameRef.value
      if (!iframe) {
        resolve()
        return
      }
      const finish = () => resolve()
      const timer = setTimeout(finish, 15000)
      const onLoad = () => {
        clearTimeout(timer)
        iframe.removeEventListener('load', onLoad)
        finish()
      }
      iframe.addEventListener('load', onLoad)
      try {
        if (iframe.contentDocument?.readyState === 'complete') {
          clearTimeout(timer)
          iframe.removeEventListener('load', onLoad)
          finish()
        }
      } catch {
        // 跨域时无法读 contentDocument，仅依赖 load
      }
    }
    nextTick().then(() => nextTick().then(bind))
  })
}

async function openPpt() {
  const token = localStorage.getItem(`${process.env.VUE_APP_KEY_PREFIX}Authorization`) || ''
  const pptBase = (process.env.VUE_APP_PPT_APP_URL || '').trim()
  if (!pptBase) {
    showToast('PPT 地址未配置')
    return
  }
  const url = new URL(pptBase)
  if (token) {
    url.searchParams.set('ius_token', token)
  }
  const urlString = url.toString()
  const title = t('imageApps.aiPpt')

  if (typeof runWithLaunchOverlay === 'function') {
    await runWithLaunchOverlay(title, async () => {
      embedTitle.value = title
      embedUrl.value = urlString
      showEmbedPopup.value = true
      await nextTick()
      await nextTick()
      await waitForEmbedIframeLoad()
    })
    return
  }

  embedTitle.value = title
  embedUrl.value = urlString
  showEmbedPopup.value = true
}

function soon() {
  showToast(t('imageApps.soon'))
}

const userCustomApps = ref([])
const showAddCustom = ref(false)
const editingCustomId = ref(null)
const customForm = reactive({
  label: '',
  url: '',
  /** @type {'embed' | 'new'} */
  openMode: 'embed',
})

const showEmbedPopup = ref(false)
const embedUrl = ref('')
const embedTitle = ref('')

/** 内嵌页：全屏铺满视口，关闭按钮由 van-popup closeable 提供 */
const embedPopupPanelStyle = {
  width: '100%',
  maxWidth: '100vw',
  height: '100dvh',
  maxHeight: '100dvh',
  boxSizing: 'border-box',
}

function openAddCustom() {
  editingCustomId.value = null
  customForm.label = ''
  customForm.url = ''
  customForm.openMode = 'embed'
  showAddCustom.value = true
}

function openEditCustom(item) {
  if (!item?.id) return
  editingCustomId.value = item.id
  customForm.label = item.label || ''
  customForm.url = item.url || ''
  customForm.openMode = item.openMode === 'new' ? 'new' : 'embed'
  showAddCustom.value = true
}

function onEmbedClosed() {
  embedUrl.value = ''
  embedTitle.value = ''
}

function normalizeCustomApps(arr) {
  if (!Array.isArray(arr)) return []
  return arr
    .filter((item) => item && typeof item === 'object')
    .map((item) => ({
      id: String(item.id || `c_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`),
      label: String(item.label || '').trim(),
      url: String(item.url || '').trim(),
      openMode: item.openMode === 'new' ? 'new' : 'embed',
      icon: item.icon || 'link-o',
    }))
    .filter((item) => item.label && item.url)
    .slice(0, CUSTOM_APPS_MAX)
}

async function loadCustomApps() {
  try {
    const res = await httpGet('/api/user/imageHubCustomApps')
    userCustomApps.value = normalizeCustomApps(res.data)
  } catch {
    userCustomApps.value = []
  }
}

async function saveCustomApps() {
  await httpPost('/api/user/imageHubCustomApps/update', {
    apps: normalizeCustomApps(userCustomApps.value),
  })
}

onMounted(() => {
  loadCustomApps()
})

function openCustomLink(item) {
  if (!item?.url) return
  const mode = item.openMode === 'new' ? 'new' : 'embed'
  if (mode === 'embed') {
    embedTitle.value = item.label || ''
    embedUrl.value = item.url
    showEmbedPopup.value = true
  } else {
    window.open(item.url, '_blank', 'noopener,noreferrer')
  }
}

async function removeCustomApp(id) {
  userCustomApps.value = userCustomApps.value.filter((x) => x.id !== id)
  await saveCustomApps()
}

function confirmDeleteCustom(item) {
  if (!item?.id) return
  showConfirmDialog({
    title: t('imageHub.removeCustomConfirmTitle'),
    message: t('imageHub.removeCustomConfirmMessage', { name: item.label || '' }),
    confirmButtonText: t('imageHub.dialogConfirm'),
    cancelButtonText: t('imageHub.dialogCancel'),
  })
    .then(async () => {
      try {
        await removeCustomApp(item.id)
        showToast(t('imageHub.customRemoved'))
      } catch {
        showToast('网络异常，请稍后重试')
      }
    })
    .catch(() => {})
}

function beforeCloseAddCustom(action) {
  return new Promise((resolve) => {
    ;(async () => {
    if (action === 'cancel') {
      customForm.label = ''
      customForm.url = ''
      customForm.openMode = 'embed'
      editingCustomId.value = null
      resolve(true)
      return
    }
    const label = customForm.label.trim()
    const urlRaw = customForm.url.trim()
    if (!label) {
      showToast(t('imageHub.nameRequired'))
      resolve(false)
      return
    }
    if (!urlRaw) {
      showToast(t('imageHub.urlRequired'))
      resolve(false)
      return
    }
    let url = urlRaw
    if (!/^https?:\/\//i.test(url)) {
      url = `https://${url}`
    }
    try {
      new URL(url)
    } catch {
      showToast(t('imageHub.invalidUrl'))
      resolve(false)
      return
    }

    const openMode = customForm.openMode === 'embed' ? 'embed' : 'new'

    const editId = editingCustomId.value
    if (editId) {
      const idx = userCustomApps.value.findIndex((x) => x.id === editId)
      if (idx === -1) {
        editingCustomId.value = null
        resolve(true)
        return
      }
      const prev = userCustomApps.value[idx]
      userCustomApps.value[idx] = {
        ...prev,
        label,
        url,
        openMode,
      }
      try {
        await saveCustomApps()
      } catch {
        showToast('网络异常，请稍后重试')
        resolve(false)
        return
      }
      showToast(t('imageHub.customUpdated'))
    } else {
      if (userCustomApps.value.length >= CUSTOM_APPS_MAX) {
        showToast(t('imageHub.customMax', { n: CUSTOM_APPS_MAX }))
        resolve(false)
        return
      }
      userCustomApps.value.push({
        id: `c_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`,
        label,
        url,
        openMode,
        icon: 'link-o',
      })
      try {
        await saveCustomApps()
      } catch {
        showToast('网络异常，请稍后重试')
        resolve(false)
        return
      }
      showToast(t('imageHub.customAdded'))
    }

    customForm.label = ''
    customForm.url = ''
    customForm.openMode = 'embed'
    editingCustomId.value = null
    resolve(true)
    })()
  })
}

const placeholderApps = computed(() => [
  { icon: 'calendar-o', label: t('imageApps.phCalendar'), iconClass: 'hub-cell__icon--c' },
  { icon: 'cluster-o', label: t('imageApps.phCloud'), iconClass: 'hub-cell__icon--d' },
  { icon: 'gem-o', label: t('imageApps.phGame'), iconClass: 'hub-cell__icon--f' },
  { icon: 'play-circle-o', label: t('imageApps.phPlayer'), iconClass: 'hub-cell__icon--g' },
  { icon: 'service-o', label: t('imageApps.phDesign'), iconClass: 'hub-cell__icon--h' },
  { icon: 'apps-o', label: t('imageApps.phMode'), iconClass: 'hub-cell__icon--i' },
  { icon: 'bookmark-o', label: t('imageApps.phCulture'), iconClass: 'hub-cell__icon--j' },
  { icon: 'balance-o', label: t('imageApps.phCalc'), iconClass: 'hub-cell__icon--k' },
  { icon: 'notes-o', label: t('imageApps.phMeasure'), iconClass: 'hub-cell__icon--l' },
  { icon: 'hot-o', label: t('imageApps.phWeather'), iconClass: 'hub-cell__icon--m' },
  { icon: 'envelop-o', label: t('imageApps.phMail'), iconClass: 'hub-cell__icon--n' },
  { icon: 'chat-o', label: t('imageApps.phChat'), iconClass: 'hub-cell__icon--o' },
  { icon: 'guide-o', label: t('imageApps.phExplore'), iconClass: 'hub-cell__icon--q' },
])
</script>

<style lang="stylus" scoped>
.image-hub
  position relative
  z-index 1
  display flex
  flex-direction column
  align-items center
  box-sizing border-box
  width 100%
  max-width 100%
  min-height calc(100vh - 50px)
  min-height calc(100dvh - 50px)
  padding 28px 24px calc(64px + env(safe-area-inset-bottom, 0px))
  overflow-x hidden
  overflow-y auto
  overscroll-behavior-x none
  -webkit-overflow-scrolling touch
  touch-action pan-y
  background linear-gradient(180deg, #b8e3ff 0%, #7ec4e8 42%, #5fb8d9 100%)

.image-hub__header
  flex-shrink 0
  width 100%
  max-width 420px
  margin-bottom 20px
  text-align left

.image-hub__greet
  margin 0
  font-size 26px
  font-weight 700
  color #0f2744
  letter-spacing 0.02em
  line-height 1.25

.image-hub__date
  margin 8px 0 0
  font-size 14px
  font-weight 500
  color rgba(15, 39, 68, 0.62)

.image-hub__subtitle
  margin 12px 0 0
  font-size 15px
  font-weight 500
  color rgba(15, 39, 68, 0.78)
  line-height 1.45

.image-hub__search-hint
  display block
  width 100%
  margin-top 14px
  padding 12px 16px
  border none
  border-radius 14px
  text-align left
  font-size 13px
  color rgba(15, 39, 68, 0.42)
  background rgba(255, 255, 255, 0.72)
  box-shadow 0 2px 14px rgba(25, 80, 120, 0.08)
  backdrop-filter blur(10px)
  -webkit-backdrop-filter blur(10px)
  cursor pointer
  -webkit-tap-highlight-color transparent

  &:active
    opacity 0.88
    transform scale(0.99)

.image-hub__footer
  flex-shrink 0
  width 100%
  max-width 420px
  margin-top auto
  padding-top 8px

.image-hub__quick-title
  margin 0 0 10px
  font-size 12px
  font-weight 700
  letter-spacing 0.06em
  color rgba(15, 39, 68, 0.48)
  text-transform uppercase

.image-hub__quick-row
  display flex
  flex-wrap wrap
  align-items center
  gap 6px 10px
  margin-bottom 16px

.image-hub__text-link
  margin 0
  padding 0
  border none
  background transparent
  font-size 14px
  font-weight 600
  color #1a6cb5
  text-decoration underline
  text-underline-offset 3px
  cursor pointer
  -webkit-tap-highlight-color transparent

  &:active
    opacity 0.75

.image-hub__sep
  font-size 14px
  color rgba(15, 39, 68, 0.35)
  user-select none

.image-hub__tips
  padding 14px 16px 16px
  border-radius 16px
  background rgba(255, 255, 255, 0.52)
  border 1px solid rgba(255, 255, 255, 0.85)
  box-shadow 0 4px 20px rgba(25, 80, 120, 0.06)

.image-hub__tips-title
  margin 0 0 8px
  font-size 12px
  font-weight 700
  color rgba(15, 39, 68, 0.52)

.image-hub__tips-line
  margin 0
  font-size 12px
  line-height 1.6
  color rgba(15, 39, 68, 0.72)

  & + &
    margin-top 6px

.image-hub__floating
  margin-bottom 20px

.image-hub__float-row
  display flex
  flex-wrap wrap
  gap 28px 20px
  padding 8px 4px 0

.image-hub__float-row--tight
  margin-top 4px
  padding-top 0

.hub-tile
  display flex
  flex-direction column
  align-items center
  gap 8px
  padding 0
  border none
  background transparent
  cursor pointer
  -webkit-tap-highlight-color transparent

  &:active
    transform scale(0.96)

.hub-tile--muted:active
  transform scale(0.96)

.hub-tile__icon
  width 58px
  height 58px
  border-radius 16px
  display flex
  align-items center
  justify-content center
  box-shadow 0 8px 20px rgba(0, 0, 0, 0.12)

.hub-tile__icon--mj
  background linear-gradient(145deg, #667eea 0%, #764ba2 100%)

.hub-tile__icon--music
  background linear-gradient(145deg, #f093fb 0%, #f5576c 100%)

.hub-tile__icon--tasks
  background linear-gradient(145deg, #4facfe 0%, #00f2fe 100%)

.hub-tile__icon--gallery
  background linear-gradient(145deg, #43e97b 0%, #38f9d7 100%)

.hub-tile__label
  font-size 12px
  font-weight 600
  color #1a2b3c
  max-width 72px
  text-align center
  line-height 1.25

.hub-tile--muted .hub-tile__icon
  opacity 0.88

.image-hub__card
  flex-shrink 0
  width 100%
  max-width 420px
  margin-bottom 8px
  box-sizing border-box
  overflow hidden
  background rgba(255, 255, 255, 0.97)
  border-radius 24px
  padding 28px 22px 32px
  box-shadow 0 16px 48px rgba(30, 80, 120, 0.12), 0 2px 8px rgba(0, 0, 0, 0.04)
  border 1px solid rgba(255, 255, 255, 0.95)

.image-hub__section-title
  width 100%
  margin 18px 0 2px
  padding 0 4px
  font-size 14px
  font-weight 700
  color rgba(15, 39, 68, 0.55)
  letter-spacing 0.04em
  border none
  background transparent

.image-hub__grid
  display grid
  grid-template-columns repeat(3, 1fr)
  gap 24px 18px
  justify-items center

  // 仅当最后一行只有 1 个图标时（总数 ≡ 1 mod 3）整行居中
  .hub-cell:last-child:nth-child(3n + 1)
    grid-column 1 / -1
    justify-self center

.image-hub__grid--custom
  margin-top 12px

.image-hub__dialog-fields
  padding 12px 16px 16px

.image-hub__open-mode
  margin-top 8px
  padding 0 4px 4px

.image-hub__open-mode-label
  font-size 13px
  font-weight 600
  color #323233
  margin-bottom 10px

.image-hub__open-mode-row
  display flex
  gap 10px

.image-hub__mode-btn
  flex 1
  padding 10px 8px
  font-size 13px
  font-weight 600
  color #646566
  background #f7f8fa
  border 1px solid #ebedf0
  border-radius 10px
  cursor pointer
  -webkit-tap-highlight-color transparent

  &--active
    color #1989fa
    background #ecf5ff
    border-color #b3d8ff

.image-hub__open-mode-hint
  margin 10px 0 0
  font-size 12px
  line-height 1.45
  color #969799

.image-hub__embed-popup
  display flex
  flex-direction column
  overflow hidden
  box-sizing border-box
  // 确保关闭层叠在 iframe 之上
  :deep(.van-popup__close-icon)
    z-index 10

.image-hub__embed-bar
  flex-shrink 0
  padding 14px 44px 12px 16px
  border-bottom 1px solid #ebedf0
  background #fff

.image-hub__embed-title
  font-size 15px
  font-weight 600
  color #323233
  display block
  overflow hidden
  text-overflow ellipsis
  white-space nowrap

.image-hub__embed-frame
  flex 1
  width 100%
  min-height 0
  border none
  background #fff

.hub-grid-slot
  width 100%
  min-width 0
  display flex
  justify-content center

.hub-custom-item
  display flex
  flex-direction column
  align-items center
  width 100%
  max-width 118px

.hub-custom-toolbar
  margin-top 8px
  display flex
  align-items center
  justify-content center
  gap 10px
  z-index 2

.hub-custom-tool
  width 30px
  height 30px
  padding 0
  border none
  border-radius 10px
  background rgba(255, 255, 255, 0.96)
  box-shadow 0 1px 8px rgba(0, 0, 0, 0.12)
  display flex
  align-items center
  justify-content center
  cursor pointer
  -webkit-tap-highlight-color transparent

  &:active
    transform scale(0.96)
    opacity 0.9

.hub-custom-tool--danger
  background rgba(255, 255, 255, 0.96)

.hub-cell__icon--custom
  background linear-gradient(145deg, #5b86e5 0%, #36d1dc 100%)
  box-shadow 0 4px 12px rgba(91, 134, 229, 0.3)

.hub-cell__icon--add
  background rgba(255, 255, 255, 0.9)
  border 2px dashed rgba(79, 172, 254, 0.5)
  box-shadow none

.hub-cell--add .hub-cell__label
  color #1a6cb5

.hub-cell--add:active
  opacity 0.9

.hub-cell
  display flex
  flex-direction column
  align-items center
  gap 10px
  width 100%
  max-width 118px
  padding 0
  border none
  background transparent
  cursor pointer
  -webkit-tap-highlight-color transparent

  &:active
    transform scale(0.95)

.hub-cell[disabled]
  pointer-events none

.hub-cell--muted:active
  transform scale(0.95)

.hub-cell--disabled
  position relative
  opacity 0.56
  cursor not-allowed
  filter saturate(0.5)

  .hub-cell__label
    color rgba(44, 62, 80, 0.68)

  .hub-cell__icon
    box-shadow none

  &:active
    transform none

.hub-cell__badge
  position absolute
  top -6px
  right -2px
  z-index 2
  padding 2px 8px
  border-radius 999px
  font-size 10px
  line-height 1.4
  font-weight 700
  color #7a5600
  background rgba(255, 220, 120, 0.95)
  border 1px solid rgba(255, 187, 0, 0.5)

.hub-cell__icon
  width 56px
  height 56px
  border-radius 17px
  display flex
  align-items center
  justify-content center
  box-shadow 0 4px 12px rgba(0, 0, 0, 0.1)

.hub-cell__icon--a
  background linear-gradient(145deg, #667eea 0%, #764ba2 100%)

.hub-cell__icon--search
  background linear-gradient(145deg, #11998e 0%, #38ef7d 100%)
  box-shadow 0 4px 14px rgba(17, 153, 142, 0.35)

.hub-cell__icon--b
  background linear-gradient(145deg, #f093fb 0%, #f5576c 100%)

.hub-cell__icon--video
  background linear-gradient(145deg, #0c3483 0%, #4b6cb7 45%, #00d4ff 100%)
  box-shadow 0 4px 14px rgba(12, 52, 131, 0.35)

.hub-cell__icon--douyin
  background linear-gradient(145deg, #0f0f0f 0%, #1a1a2e 40%, #e94560 100%)
  box-shadow 0 4px 14px rgba(233, 69, 96, 0.25)

.hub-cell__icon--ppt
  background linear-gradient(145deg, #f7971e 0%, #ff512f 50%, #dd2476 100%)
  box-shadow 0 4px 14px rgba(247, 151, 30, 0.35)

.hub-cell__icon--ppt .van-icon
  color #fff !important

.hub-cell__icon--logo
  background linear-gradient(145deg, #1d2671 0%, #c33764 100%)
  box-shadow 0 4px 14px rgba(74, 48, 144, 0.35)

.hub-cell__icon--c
  background linear-gradient(145deg, #fa709a 0%, #fee140 100%)

.hub-cell__icon--d
  background linear-gradient(145deg, #30cfd0 0%, #330867 100%)

.hub-cell__icon--e
  background linear-gradient(145deg, #a8edea 0%, #fed6e3 100%)

.hub-cell__icon--e .van-icon
  color #5c4d7a !important

.hub-cell__icon--f
  background linear-gradient(145deg, #ff9a56 0%, #ff6a88 100%)

.hub-cell__icon--g
  background linear-gradient(145deg, #a18cd1 0%, #fbc2eb 100%)

.hub-cell__icon--h
  background linear-gradient(145deg, #84fab0 0%, #8fd3f4 100%)

.hub-cell__icon--h .van-icon
  color #2c5282 !important

.hub-cell__icon--i
  background linear-gradient(145deg, #ffecd2 0%, #fcb69f 100%)

.hub-cell__icon--i .van-icon
  color #c05621 !important

.hub-cell__icon--j
  background linear-gradient(145deg, #f6d365 0%, #fda085 100%)

.hub-cell__icon--k
  background linear-gradient(145deg, #e0c3fc 0%, #8ec5fc 100%)

.hub-cell__icon--l
  background linear-gradient(145deg, #cfd9df 0%, #e2ebf0 100%)

.hub-cell__icon--l .van-icon
  color #4a5568 !important

.hub-cell__icon--m
  background linear-gradient(145deg, #ffeaa7 0%, #fdcb6e 100%)

.hub-cell__icon--m .van-icon
  color #d35400 !important

.hub-cell__icon--n
  background linear-gradient(145deg, #74b9ff 0%, #0984e3 100%)

.hub-cell__icon--o
  background linear-gradient(145deg, #a29bfe 0%, #6c5ce7 100%)

.hub-cell__icon--p
  background linear-gradient(145deg, #55efc4 0%, #00b894 100%)

.hub-cell__icon--q
  background linear-gradient(145deg, #81ecec 0%, #00cec9 100%)

.hub-cell__label
  font-size 12px
  font-weight 600
  color #2c3e50
  text-align center
  line-height 1.4
  width 100%
  max-width 118px
  overflow hidden
  display -webkit-box
  -webkit-line-clamp 2
  -webkit-box-orient vertical
  word-break break-all

.hub-cell--muted .hub-cell__icon
  opacity 0.94

.hub-cell--live .hub-cell__icon
  box-shadow 0 4px 16px rgba(79, 172, 254, 0.35)
</style>
