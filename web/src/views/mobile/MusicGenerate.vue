<template>
  <div class="music-page">
    <header class="music-page__head van-safe-area-top">
      <button type="button" class="music-page__close" aria-label="close" @click="closeToHub">
        <van-icon name="cross" size="20" />
      </button>
      <div class="music-page__head-main">
        <div class="music-page__head-icon">
          <van-icon name="music-o" size="22" color="#fff" />
        </div>
        <div>
          <h1 class="music-page__head-title">{{ t('imageTabs.tabMusic') }}</h1>
          <p class="music-page__head-sub">{{ t('musicGenerate.heroSubtitle') }}</p>
        </div>
      </div>
    </header>

    <div class="music-page__scroll">
      <div v-if="musicMaintenance" class="music-page__maintenance">
        <van-icon name="warning-o" size="22" color="#0f766e" />
        <p>{{ t('musicGenerate.maintenance') }}</p>
      </div>

      <div class="music-page__mode" role="tablist">
        <button
          type="button"
          class="music-page__mode-btn"
          :class="{ 'music-page__mode-btn--active': activeMode === 'inspiration' }"
          role="tab"
          :aria-selected="activeMode === 'inspiration'"
          @click="activeMode = 'inspiration'"
        >
          {{ t('musicGenerate.tabInspiration') }}
        </button>
        <button
          type="button"
          class="music-page__mode-btn"
          role="tab"
          :aria-selected="false"
          @click="onCustomModeClick"
        >
          {{ t('musicGenerate.tabCustom') }}
        </button>
      </div>

      <section class="music-page__card">
        <van-form @submit="handleGenerate">
          <div class="music-page__theme-box">
            <label class="music-page__theme-label" for="music-theme-input">
              <van-icon name="edit" size="16" color="#0f766e" />
              {{ t('musicGenerate.fieldTheme') }}
            </label>
            <van-field
              id="music-theme-input"
              v-model="theme"
              class="music-page__theme-field"
              type="textarea"
              :border="false"
              :disabled="musicMaintenance"
              :placeholder="t('musicGenerate.placeholderTheme')"
              :rules="[{ required: true, message: t('musicGenerate.errorThemeRequired') }]"
              :autosize="{ minHeight: 120, maxHeight: 200 }"
            />
          </div>

          <button type="submit" class="music-page__submit-hidden" aria-hidden="true" tabindex="-1" />
        </van-form>
      </section>

      <section v-if="musicResult && !isMusicDone(musicResult)" class="music-page__status">
        <div class="music-page__status-head">
          <van-loading type="spinner" size="18" color="#0f766e" />
          <span class="music-page__status-title">{{ statusText }}</span>
        </div>
        <div class="music-page__progress-track">
          <div class="music-page__progress-fill" :style="{ width: progressPercentage + '%' }" />
        </div>
        <span class="music-page__progress-text">{{ progressPercentage }}%</span>
      </section>

      <section class="music-page__panel">
        <div class="music-page__panel-head">
          <div>
            <h2 class="music-page__panel-title">{{ t('musicGenerate.historyTitle') }}</h2>
            <p class="music-page__panel-sub">
              {{ sortedMusicList.length ? t('musicGenerate.historyCount', { n: sortedMusicList.length }) : t('musicGenerate.historyEmpty') }}
            </p>
          </div>
        </div>

        <div v-if="sortedMusicList.length === 0" class="music-page__empty">
          <div class="music-page__empty-icon">
            <van-icon name="music" size="32" color="#0f766e" />
          </div>
          <p>{{ t('musicGenerate.historyEmpty') }}</p>
        </div>

        <div v-else class="music-page__tracks">
          <article
            v-for="item in sortedMusicList"
            :key="item.Id"
            class="music-page__track"
            :class="{ 'music-page__track--playing': playingId === item.Id }"
            @click="openPlayer(item)"
          >
            <button
              type="button"
              class="music-page__track-cover"
              :disabled="!item.AudioUrl"
              @click.stop="openPlayer(item)"
            >
              <img
                v-if="item.ImageUrl"
                :src="item.ImageUrl"
                :alt="item.Content"
                class="music-page__track-img"
              />
              <div v-else class="music-page__track-fallback">
                <van-icon name="music-o" size="22" color="#fff" />
              </div>
              <span v-if="!item.AudioUrl" class="music-page__track-badge">
                <van-loading type="spinner" size="14" color="#fff" />
              </span>
              <span v-else-if="playingId === item.Id" class="music-page__track-play music-page__track-play--active">
                <van-icon name="pause" size="18" color="#fff" />
              </span>
              <span v-else class="music-page__track-play">
                <van-icon name="play" size="18" color="#fff" />
              </span>
            </button>

            <div class="music-page__track-body">
              <div class="music-page__track-meta">
                <h3 class="music-page__track-title">{{ item.Content || t('musicGenerate.untitled') }}</h3>
                <time class="music-page__track-time">{{ formatTime(item.CreatedAt) }}</time>
              </div>

              <p v-if="!item.AudioUrl" class="music-page__track-pending">
                <van-loading type="spinner" size="14" color="#0f766e" />
                <span>{{ t('musicGenerate.statusPending') }}</span>
              </p>
              <p v-else-if="item.Lyrics" class="music-page__track-lyric-hint">
                {{ t('musicGenerate.tapForLyrics') }}
              </p>
              <p v-else class="music-page__track-pending music-page__track-pending--muted">
                {{ t('musicGenerate.noLyrics') }}
              </p>
            </div>

            <button
              v-if="item.AudioUrl"
              type="button"
              class="music-page__track-dl"
              :disabled="downloadingId === item.Id"
              @click.stop="downloadTrack(item)"
            >
              <van-icon :name="downloadingId === item.Id ? 'loading' : 'down'" size="16" />
            </button>
          </article>
        </div>
      </section>

      <div class="music-page__scroll-spacer" />
    </div>

    <van-popup
      v-model:show="showPlayer"
      position="bottom"
      round
      class="music-player-popup"
      :style="{ height: '92dvh' }"
      teleport="body"
      @closed="onPlayerClosed"
    >
      <div v-if="currentTrack" class="music-player">
        <div class="music-player__head">
          <button type="button" class="music-player__close" @click="showPlayer = false">
            <van-icon name="arrow-down" size="20" />
          </button>
          <span class="music-player__head-label">{{ t('musicGenerate.nowPlaying') }}</span>
          <button
            type="button"
            class="music-player__dl"
            :disabled="!currentTrack.AudioUrl || downloadingId === currentTrack.Id"
            @click="downloadTrack(currentTrack)"
          >
            <van-icon :name="downloadingId === currentTrack.Id ? 'loading' : 'down'" size="18" />
          </button>
        </div>

        <div class="music-player__cover-wrap">
          <img
            v-if="currentTrack.ImageUrl"
            :src="currentTrack.ImageUrl"
            :alt="currentTrack.Content"
            class="music-player__cover"
            :class="{ 'music-player__cover--spin': isPlaying }"
          />
          <div v-else class="music-player__cover music-player__cover--fallback" :class="{ 'music-player__cover--spin': isPlaying }">
            <van-icon name="music-o" size="48" color="#fff" />
          </div>
        </div>

        <h2 class="music-player__title">{{ currentTrack.Content || t('musicGenerate.untitled') }}</h2>

        <div class="music-player__progress">
          <span class="music-player__time">{{ formatAudioTime(currentTime) }}</span>
          <van-slider
            v-model="sliderValue"
            class="music-player__slider"
            :max="sliderMax"
            :disabled="!playerDuration"
            active-color="#0f766e"
            inactive-color="rgba(17,24,39,0.12)"
            @change="onSeek"
            @drag-start="isSeeking = true"
            @drag-end="isSeeking = false"
          />
          <span class="music-player__time">{{ formatAudioTime(playerDuration) }}</span>
        </div>

        <div class="music-player__controls">
          <button type="button" class="music-player__ctrl" @click="seekRelative(-10)">-10s</button>
          <button type="button" class="music-player__play" @click="togglePlay">
            <van-icon :name="isPlaying ? 'pause' : 'play'" size="28" color="#fff" />
          </button>
          <button type="button" class="music-player__ctrl" @click="seekRelative(10)">+10s</button>
        </div>

        <div class="music-player__lyrics-head">{{ t('musicGenerate.lyricsLabel') }}</div>
        <div ref="lyricsScrollRef" class="music-player__lyrics">
          <template v-if="parsedLyrics.length">
            <p
              v-for="line in parsedLyrics"
              :key="line.id"
              :ref="(el) => setLyricLineRef(line.id, el)"
              class="music-player__lyric-line"
              :class="{
                'music-player__lyric-line--active': line.id === activeLyricId || isSectionActive(line),
                'music-player__lyric-line--section': line.type === 'section',
                'music-player__lyric-line--meta': line.type === 'meta',
              }"
            >
              {{ line.text }}
            </p>
          </template>
          <p v-else class="music-player__lyrics-empty">{{ t('musicGenerate.noLyrics') }}</p>
        </div>

        <audio
          v-if="currentTrack.AudioUrl"
          ref="playerAudioRef"
          :src="currentTrack.AudioUrl"
          class="music-player__audio"
          preload="metadata"
          @timeupdate="onTimeUpdate"
          @loadedmetadata="onLoadedMetadata"
          @play="onPlayerPlay"
          @pause="onPlayerPause"
          @ended="onPlayerEnded"
        />
      </div>
    </van-popup>

    <footer class="music-page__footer van-safe-area-bottom">
      <van-button
        round
        block
        type="primary"
        class="music-page__footer-btn"
        :loading="generating"
        :disabled="musicMaintenance"
        @click="submitForm"
      >
        <van-icon name="fire-o" size="18" />
        {{ musicMaintenance ? t('musicGenerate.maintenanceShort') : t('musicGenerate.buttonGenerate') }}
      </van-button>
    </footer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, inject, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { IMAGE_RETURN_HUB_KEY } from '@/constants/imageAppLaunch'
import { getSystemInfo } from '@/store/cache'
import { parseAppFlags } from '@/utils/appFeatures'
import { httpGet, httpPost, resolveApiBaseURL } from '@/utils/http'
import { getUserToken } from '@/store/session'
import { parseLyricsLines, buildLyricTimeline, getActiveLyricIdFromTimeline, formatAudioTime, normalizeMusicJob, normalizeMusicStatus, isMusicJobDone, getMusicJobProgress } from '@/utils/musicLyrics'
import { showToast, showLoadingToast, closeToast, showFailToast } from 'vant'
import dayjs from 'dayjs'

const { t } = useI18n()
const router = useRouter()
const returnToHubNative = inject(IMAGE_RETURN_HUB_KEY, null)
const musicMaintenance = ref(false)

async function loadAppFlags() {
  try {
    const sys = await getSystemInfo()
    musicMaintenance.value = !parseAppFlags(sys).music
  } catch {
    musicMaintenance.value = false
  }
}

async function closeToHub() {
  if (typeof returnToHubNative === 'function') {
    await returnToHubNative()
    return
  }
  router.push({ name: 'mobile-image-hub' })
}

const activeMode = ref('inspiration')
const theme = ref('')
const makeInstrumental = ref(false)
const modelVersion = ref('chirp-v4')
const generating = ref(false)
const musicResult = ref(null)
const musicList = ref([])
const playingId = ref(null)
const downloadingId = ref(null)
const showPlayer = ref(false)
const currentTrack = ref(null)
const playerAudioRef = ref(null)
const lyricsScrollRef = ref(null)
const lyricLineRefs = new Map()
const currentTime = ref(0)
const playerDuration = ref(0)
const isPlaying = ref(false)
const isSeeking = ref(false)
const sliderValue = ref(0)
const activeLyricId = ref(-1)
let lyricRafId = null

function stopLyricSyncLoop() {
  if (lyricRafId) {
    cancelAnimationFrame(lyricRafId)
    lyricRafId = null
  }
}

function startLyricSyncLoop() {
  stopLyricSyncLoop()
  const loop = () => {
    const audio = playerAudioRef.value
    if (!audio || audio.paused || isSeeking.value) {
      lyricRafId = null
      return
    }
    updateActiveLyric(audio.currentTime)
    lyricRafId = requestAnimationFrame(loop)
  }
  lyricRafId = requestAnimationFrame(loop)
}

const sliderMax = computed(() => Math.max(playerDuration.value, 1))

const parsedLyrics = computed(() => {
  const lyrics = currentTrack.value?.Lyrics || ''
  return parseLyricsLines(lyrics)
})

const lyricTimeline = computed(() => {
  const duration = playerDuration.value
  if (!duration || !parsedLyrics.value.length) return []
  return buildLyricTimeline(parsedLyrics.value, duration)
})

function updateActiveLyric(time) {
  if (!lyricTimeline.value.length) {
    activeLyricId.value = -1
    return
  }
  activeLyricId.value = getActiveLyricIdFromTimeline(lyricTimeline.value, time)
}

function onCustomModeClick() {
  showToast(t('musicGenerate.toastCustomNotSupported'))
}

const statusText = computed(() => {
  if (!musicResult.value) return ''
  const status = normalizeMusicStatus(musicResult.value.Status)
  switch (status) {
    case 'success':
      return t('musicGenerate.statusSuccess')
    case 'failed':
      return t('musicGenerate.statusFailed')
    default:
      return t('musicGenerate.statusPending')
  }
})

function isMusicDone(item) {
  return isMusicJobDone(item)
}

const progressPercentage = computed(() => getMusicJobProgress(musicResult.value))

function setLyricLineRef(id, el) {
  if (el) lyricLineRefs.set(id, el)
  else lyricLineRefs.delete(id)
}

function openPlayer(item) {
  if (!item?.AudioUrl) return
  void openPlayerAsync(item)
}

function guessMusicExt(mime) {
  const m = String(mime || '').toLowerCase()
  if (m.includes('wav')) return 'wav'
  if (m.includes('ogg')) return 'ogg'
  if (m.includes('m4a') || m.includes('mp4')) return 'm4a'
  return 'mp3'
}

async function downloadTrack(item) {
  if (!item?.Id || downloadingId.value === item.Id) return
  if (!item.AudioUrl) {
    showToast(t('musicGenerate.downloadNoUrl'))
    return
  }
  downloadingId.value = item.Id
  const safeName = String(item.Content || 'music')
    .replace(/[\\/:*?"<>|]+/g, '_')
    .slice(0, 40)
  const fallbackOpen = (url) => {
    const a = document.createElement('a')
    a.href = url
    a.target = '_blank'
    a.rel = 'noopener'
    a.download = `${safeName}_${item.Id}.mp3`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  }
  try {
    const base = resolveApiBaseURL()
    const token = getUserToken()
    const proxyUrl = `${base}/api/suno/download/${item.Id}`
    const res = await fetch(proxyUrl, {
      headers: token ? { Authorization: token } : {},
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const blob = await res.blob()
    if (!blob || blob.size === 0) throw new Error('empty')
    const ext = guessMusicExt(blob.type || res.headers.get('Content-Type'))
    const objectUrl = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = objectUrl
    a.download = `${safeName}_${item.Id}.${ext}`
    a.rel = 'noopener'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    setTimeout(() => URL.revokeObjectURL(objectUrl), 2000)
    showToast(t('musicGenerate.downloadStart'))
  } catch (err) {
    try {
      // 同源代理失败时回退直链
      const res = await fetch(item.AudioUrl)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const blob = await res.blob()
      const ext = guessMusicExt(blob.type)
      const objectUrl = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = objectUrl
      a.download = `${safeName}_${item.Id}.${ext}`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      setTimeout(() => URL.revokeObjectURL(objectUrl), 2000)
      showToast(t('musicGenerate.downloadStart'))
    } catch (e2) {
      try {
        fallbackOpen(item.AudioUrl)
        showToast(t('musicGenerate.downloadStart'))
      } catch (e3) {
        showToast(t('musicGenerate.downloadFailPrefix') + (err?.message || e2?.message || ''))
      }
    }
  } finally {
    if (downloadingId.value === item.Id) downloadingId.value = null
  }
}

async function openPlayerAsync(item) {
  let track = normalizeMusicJob(item)
  if (!track.Lyrics && track.TaskId) {
    try {
      const res = await httpGet(`/api/suno/status/${track.TaskId}`, { refresh: 1 })
      if (res.code === 0 && res.data) {
        track = normalizeMusicJob(res.data)
        const idx = musicList.value.findIndex((x) => x.Id === item.Id)
        if (idx >= 0) musicList.value[idx] = track
      }
    } catch {
      // ignore
    }
  }

  currentTrack.value = track
  showPlayer.value = true
  playingId.value = track.Id
  currentTime.value = 0
  playerDuration.value = Number(track.Duration) || 0
  sliderValue.value = 0
  activeLyricId.value = -1
  await nextTick()
  const audio = playerAudioRef.value
  if (!audio) return
  audio.currentTime = 0
  audio.play().catch(() => {})
}

function onPlayerClosed() {
  stopLyricSyncLoop()
  const audio = playerAudioRef.value
  if (audio) {
    audio.pause()
  }
  isPlaying.value = false
  playingId.value = null
  currentTrack.value = null
  activeLyricId.value = -1
}

function togglePlay() {
  const audio = playerAudioRef.value
  if (!audio) return
  if (audio.paused) {
    audio.play().catch(() => {})
  } else {
    audio.pause()
  }
}

function onPlayerPlay() {
  isPlaying.value = true
  if (currentTrack.value) playingId.value = currentTrack.value.Id
  startLyricSyncLoop()
}

function onPlayerPause() {
  isPlaying.value = false
  stopLyricSyncLoop()
}

function onPlayerEnded() {
  isPlaying.value = false
  stopLyricSyncLoop()
  currentTime.value = 0
  sliderValue.value = 0
  activeLyricId.value = -1
}

function isSectionActive(line) {
  if (line.type !== 'section' || !lyricTimeline.value.length) return false
  const entry = lyricTimeline.value.find((e) => e.id === line.id)
  if (!entry) return false
  const t = currentTime.value
  return t >= entry.start && t < entry.end + 0.5
}

function onLoadedMetadata() {
  const audio = playerAudioRef.value
  if (!audio) return
  const duration = audio.duration
  if (Number.isFinite(duration) && duration > 0) {
    playerDuration.value = duration
  } else if (currentTrack.value?.Duration) {
    playerDuration.value = Number(currentTrack.value.Duration)
  }
  updateActiveLyric(audio.currentTime || 0)
}

function onTimeUpdate() {
  const audio = playerAudioRef.value
  if (!audio || isSeeking.value) return
  currentTime.value = audio.currentTime
  sliderValue.value = audio.currentTime
  updateActiveLyric(audio.currentTime)
}

function onSeek(value) {
  const audio = playerAudioRef.value
  if (!audio) return
  audio.currentTime = value
  currentTime.value = value
  updateActiveLyric(value)
}

function seekRelative(delta) {
  const audio = playerAudioRef.value
  if (!audio) return
  const duration = playerDuration.value || audio.duration || 0
  const next = Math.min(Math.max(audio.currentTime + delta, 0), duration || audio.currentTime + delta)
  audio.currentTime = next
  onSeek(next)
}

watch(activeLyricId, (id, prev) => {
  if (id < 0) return
  nextTick(() => {
    const el = lyricLineRefs.get(id)
    if (el && lyricsScrollRef.value) {
      el.scrollIntoView({
        behavior: prev < 0 || Math.abs(id - prev) > 2 ? 'auto' : 'smooth',
        block: 'center',
      })
    }
  })
})

watch(lyricTimeline, () => {
  updateActiveLyric(currentTime.value)
})

function submitForm() {
  const form = document.querySelector('.music-page__card form')
  if (form) {
    form.requestSubmit()
  }
}

const handleGenerate = async () => {
  if (musicMaintenance.value) {
    showToast(t('musicGenerate.maintenance'))
    return
  }
  try {
    generating.value = true
    showLoadingToast({
      message: t('musicGenerate.toastSubmitting'),
      forbidClick: true,
    })

    const trimmed = theme.value.trim()
    const data = {
      gpt_description_prompt: trimmed,
      make_instrumental: makeInstrumental.value,
      mv: modelVersion.value,
      prompt: trimmed,
    }

    const res = await httpPost('/api/suno/generate', data)

    closeToast()
    if (res.code === 0 && res.data) {
      showToast(t('musicGenerate.toastSubmitSuccess'))
      musicResult.value = normalizeMusicJob(res.data)
      const taskId = musicResult.value?.TaskId
      if (taskId) {
        startPolling(taskId)
      }
    } else {
      showFailToast(res.msg || t('musicGenerate.toastGenerateFail'))
    }
  } catch (error) {
    closeToast()
    showFailToast(error.msg || t('musicGenerate.toastRequestFail'))
  } finally {
    generating.value = false
  }
}

let ws = null
let pollTimer = null
let pollingTaskId = null

function stopPolling() {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
  pollingTaskId = null
}

async function fetchTaskStatus(taskId) {
  const res = await httpGet(`/api/suno/status/${taskId}`)
  if (res.code !== 0 || !res.data) return null
  return normalizeMusicJob(res.data)
}

async function handleTaskUpdate(job) {
  if (!job) return
  musicResult.value = job

  if (!isMusicJobDone(job)) return

  stopPolling()

  if (normalizeMusicStatus(job.Status) === 'failed') {
    showToast(t('musicGenerate.toastGenerateFailWithReason') + (job.ErrMsg || ''))
    return
  }

  showToast(t('musicGenerate.toastGenerateSuccess'))
  musicResult.value = null
  await loadMusicList()

  const finished = musicList.value.find((x) => x.TaskId === job.TaskId || x.Id === job.Id)
  if (finished?.AudioUrl) {
    void openPlayerAsync(finished)
  }
}

function startPolling(taskId) {
  if (!taskId) return
  stopPolling()
  pollingTaskId = taskId

  const tick = async () => {
    if (pollingTaskId !== taskId) return
    try {
      const job = await fetchTaskStatus(taskId)
      if (!job || pollingTaskId !== taskId) return

      if (isMusicJobDone(job)) {
        await handleTaskUpdate(job)
        return
      }

      musicResult.value = job
      pollTimer = setTimeout(tick, 3000)
    } catch (error) {
      console.error('Poll status error:', error)
      if (pollingTaskId === taskId) {
        pollTimer = setTimeout(tick, 5000)
      }
    }
  }

  void tick()
}

function resumePendingTask() {
  const pending = musicList.value.find((item) => !isMusicJobDone(item))
  if (!pending?.TaskId) return
  musicResult.value = pending
  startPolling(pending.TaskId)
}

const loadMusicList = async () => {
  try {
    const res = await httpGet('/api/suno/list')
    if (res.code === 0) {
      musicList.value = (res.data || []).map(normalizeMusicJob)

      if (pollingTaskId) {
        const current = musicList.value.find((x) => x.TaskId === pollingTaskId)
        if (current && isMusicJobDone(current)) {
          await handleTaskUpdate(current)
        }
      }
    }
  } catch (error) {
    console.error('Load music list error:', error)
  }
}

const sortedMusicList = computed(() => {
  return [...musicList.value].sort((a, b) => new Date(b.CreatedAt) - new Date(a.CreatedAt))
})

const formatTime = (time) => dayjs(time).format('MM-DD HH:mm')

const connectWebSocket = () => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/api/ws`

  ws = new WebSocket(wsUrl)

  ws.onmessage = (event) => {
    if (event.data === 'Task Updated') {
      void loadMusicList()
    }
  }

  ws.onclose = () => {
    setTimeout(connectWebSocket, 3000)
  }
}

onMounted(async () => {
  await loadAppFlags()
  await loadMusicList()
  resumePendingTask()
  connectWebSocket()
})

onBeforeUnmount(() => {
  stopPolling()
  stopLyricSyncLoop()
  if (ws) {
    ws.close()
  }
  const audio = playerAudioRef.value
  if (audio) audio.pause()
})
</script>

<style scoped lang="stylus">
@import "@/assets/css/custom-scroll.styl"

.music-page
  --ai-bg: #f4f5f7
  --ai-surface: #ffffff
  --ai-text: #111827
  --ai-muted: #6b7280
  --ai-accent: #0f766e
  --ai-line: rgba(17, 24, 39, 0.08)
  display flex
  flex-direction column
  min-height 100vh
  min-height 100dvh
  background: radial-gradient(ellipse 80% 45% at 0% 0%, rgba(219, 39, 119, 0.06), transparent 55%), radial-gradient(ellipse 70% 40% at 100% 0%, rgba(15, 118, 110, 0.05), transparent 50%), #f4f5f7
  color var(--ai-text)
  font-family 'Plus Jakarta Sans', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif
  -webkit-font-smoothing antialiased

.music-page__head
  position relative
  flex-shrink 0
  padding 12px 16px 18px
  background rgba(255, 255, 255, 0.92)
  border-bottom 1px solid var(--ai-line)
  backdrop-filter blur(14px)
  -webkit-backdrop-filter blur(14px)

.music-page__close
  position absolute
  top 12px
  right 12px
  z-index 2
  display flex
  align-items center
  justify-content center
  width 36px
  height 36px
  padding 0
  border none
  border-radius 12px
  background #f3f4f6
  color #4b5563
  cursor pointer
  -webkit-tap-highlight-color transparent

  &:active
    transform scale(0.94)

.music-page__head-main
  display flex
  align-items center
  gap 12px
  padding-right 40px

.music-page__head-icon
  flex-shrink 0
  width 44px
  height 44px
  border-radius 14px
  display flex
  align-items center
  justify-content center
  background linear-gradient(145deg, #db2777 0%, #f43f5e 100%)
  box-shadow 0 8px 20px rgba(219, 39, 119, 0.28)

.music-page__head-title
  margin 0
  font-size 18px
  font-weight 800
  letter-spacing (-0.02em)
  color var(--ai-text)
  line-height 1.3

.music-page__head-sub
  margin 4px 0 0
  font-size 12px
  line-height 1.45
  color var(--ai-muted)

.music-page__scroll
  flex 1
  min-height 0
  overflow-x hidden
  overflow-y auto
  -webkit-overflow-scrolling touch
  padding 16px 16px 0

.music-page__maintenance
  display flex
  align-items flex-start
  gap 10px
  margin-bottom 14px
  padding 12px 14px
  border-radius 12px
  background rgba(232, 77, 138, 0.1)
  border 1px solid rgba(232, 77, 138, 0.22)

  p
    margin 0
    flex 1
    font-size 14px
    line-height 1.5
    color rgba(15, 39, 68, 0.78)

.music-page__mode
  display flex
  gap 8px
  padding 4px
  margin-bottom 14px
  border-radius 14px
  background #fff
  border 1px solid var(--ai-line)
  box-shadow 0 1px 3px rgba(17, 24, 39, 0.04)

.music-page__mode-btn
  flex 1
  min-width 0
  padding 10px 8px
  border none
  border-radius 10px
  background transparent
  font-size 13px
  font-weight 600
  color var(--ai-muted)
  cursor pointer
  transition background 0.2s ease, color 0.2s ease, box-shadow 0.2s ease
  -webkit-tap-highlight-color transparent

  &--active
    color #fff
    background #111827
    box-shadow 0 4px 12px rgba(17, 24, 39, 0.16)

.music-page__card
  padding 16px
  border-radius 18px
  background var(--ai-surface)
  border 1px solid var(--ai-line)
  box-shadow 0 1px 3px rgba(17, 24, 39, 0.04), 0 8px 24px rgba(17, 24, 39, 0.04)

.music-page__theme-box
  padding 14px 14px 10px
  border-radius 16px
  background #f9fafb
  border 1.5px solid var(--ai-line)
  box-shadow none
  transition border-color 0.2s ease, box-shadow 0.2s ease

  &:focus-within
    border-color rgba(15, 118, 110, 0.45)
    box-shadow 0 0 0 3px rgba(15, 118, 110, 0.1)

.music-page__theme-label
  display flex
  align-items center
  gap 6px
  margin 0 0 10px 2px
  font-size 13px
  font-weight 700
  color #374151

.music-page__theme-field
  padding 0
  background transparent

.music-page :deep(.music-page__theme-field .van-cell)
  padding 0
  background transparent

.music-page :deep(.music-page__theme-field .van-field__body)
  background transparent

.music-page :deep(.music-page__theme-field .van-field__control)
  font-size 15px
  line-height 1.6
  color var(--ai-text)
  background transparent

.music-page :deep(.music-page__theme-field .van-field__control::placeholder)
  color #9ca3af
  font-size 14px

.music-page__submit-hidden
  display none

.music-page__status
  margin-top 14px
  padding 14px 16px
  border-radius 16px
  background var(--ai-surface)
  border 1px solid var(--ai-line)
  box-shadow 0 1px 3px rgba(17, 24, 39, 0.04)

.music-page__status-head
  display flex
  align-items center
  gap 8px
  margin-bottom 10px

.music-page__status-title
  font-size 13px
  font-weight 600
  color rgba(15, 39, 68, 0.72)

.music-page__progress-track
  height 6px
  border-radius 999px
  background rgba(15, 39, 68, 0.08)
  overflow hidden

.music-page__progress-fill
  height 100%
  border-radius inherit
  background linear-gradient(90deg, #0f766e 0%, #14b8a6 100%)
  transition width 0.35s ease

.music-page__progress-text
  display block
  margin-top 6px
  font-size 11px
  font-weight 600
  color rgba(15, 39, 68, 0.42)
  text-align right

.music-page__panel
  margin-top 18px
  padding 16px
  border-radius 18px
  background var(--ai-surface)
  border 1px solid var(--ai-line)
  box-shadow 0 1px 3px rgba(17, 24, 39, 0.04)

.music-page__panel-head
  margin-bottom 14px

.music-page__panel-title
  margin 0
  font-size 15px
  font-weight 700
  color #0f2744

.music-page__panel-sub
  margin 4px 0 0
  font-size 12px
  color rgba(15, 39, 68, 0.45)

.music-page__empty
  display flex
  flex-direction column
  align-items center
  gap 10px
  padding 28px 12px
  text-align center

  p
    margin 0
    font-size 13px
    color rgba(15, 39, 68, 0.45)
    line-height 1.5

.music-page__empty-icon
  width 64px
  height 64px
  border-radius 50%
  display flex
  align-items center
  justify-content center
  background linear-gradient(145deg, rgba(240, 147, 251, 0.18), rgba(245, 87, 108, 0.14))

.music-page__tracks
  display flex
  flex-direction column
  gap 10px

.music-page__track
  display flex
  align-items center
  gap 12px
  padding 12px
  border-radius 16px
  background #fff
  border 1px solid rgba(15, 39, 68, 0.05)
  box-shadow 0 2px 12px rgba(25, 80, 120, 0.04)
  transition box-shadow 0.2s ease, transform 0.2s ease
  cursor pointer
  -webkit-tap-highlight-color transparent

  &:active
    transform scale(0.99)

  &--playing
    border-color rgba(245, 87, 108, 0.22)
    box-shadow 0 6px 20px rgba(245, 87, 108, 0.12)

.music-page__track-cover
  position relative
  flex-shrink 0
  width 56px
  height 56px
  padding 0
  border none
  border-radius 14px
  overflow hidden
  cursor pointer
  background transparent

.music-page__track-img
  width 100%
  height 100%
  object-fit cover
  display block

.music-page__track-fallback
  width 100%
  height 100%
  display flex
  align-items center
  justify-content center
  background linear-gradient(145deg, #db2777 0%, #f43f5e 100%)

.music-page__track-badge
  position absolute
  inset 0
  display flex
  align-items center
  justify-content center
  background rgba(15, 39, 68, 0.35)

.music-page__track-play
  position absolute
  inset 0
  display flex
  align-items center
  justify-content center
  background rgba(15, 39, 68, 0.22)
  opacity 0.85
  transition opacity 0.2s ease

  &--active
    opacity 1
    background rgba(15, 118, 110, 0.4)

.music-page__track-lyric-hint
  margin 0
  font-size 12px
  color var(--ai-accent)
  line-height 1.4

.music-page__track-body
  flex 1
  min-width 0

.music-page__track-dl
  flex-shrink 0
  width 36px
  height 36px
  border none
  border-radius 50%
  background rgba(15, 118, 110, 0.1)
  color #0f766e
  display flex
  align-items center
  justify-content center
  &:disabled
    opacity 0.55

.music-page__track-meta
  display flex
  align-items flex-start
  justify-content space-between
  gap 8px
  margin-bottom 8px

.music-page__track-title
  margin 0
  font-size 14px
  font-weight 700
  color #0f2744
  line-height 1.35
  overflow hidden
  text-overflow ellipsis
  display -webkit-box
  -webkit-line-clamp 2
  -webkit-box-orient vertical

.music-page__track-time
  flex-shrink 0
  font-size 11px
  color rgba(15, 39, 68, 0.38)
  white-space nowrap

.music-page__track-pending
  display flex
  align-items center
  gap 6px
  margin 0
  font-size 12px
  color rgba(15, 39, 68, 0.45)
  line-height 1.4

  &--muted
    color rgba(15, 39, 68, 0.32)

.music-page__audio
  display none

.music-page__scroll-spacer
  height calc(88px + env(safe-area-inset-bottom, 0px))

.music-page__footer
  position fixed
  left 0
  right 0
  bottom 0
  z-index 20
  padding 10px 16px calc(10px + env(safe-area-inset-bottom, 0px))
  background linear-gradient(180deg, rgba(244, 245, 247, 0) 0%, rgba(244, 245, 247, 0.92) 28%, #f4f5f7 100%)
  backdrop-filter blur(8px)
  -webkit-backdrop-filter blur(8px)

.music-page__footer-btn
  height 48px
  border none
  border-radius 14px
  font-size 15px
  font-weight 700
  background #111827
  box-shadow 0 8px 24px rgba(17, 24, 39, 0.2)

  :deep(.van-button__text)
    display inline-flex
    align-items center
    gap 6px

.music-player-popup
  overflow hidden

.music-player
  display flex
  flex-direction column
  height 100%
  padding 0 20px calc(16px + env(safe-area-inset-bottom, 0px))
  background #f4f5f7
  box-sizing border-box
  font-family 'Plus Jakarta Sans', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif

.music-player__head
  display flex
  align-items center
  justify-content space-between
  padding 10px 0 8px

.music-player__close
  display flex
  align-items center
  justify-content center
  width 36px
  height 36px
  border none
  border-radius 50%
  background rgba(15, 39, 68, 0.06)
  color #5c6b7a
  cursor pointer

.music-player__head-label
  font-size 12px
  font-weight 600
  color rgba(15, 39, 68, 0.45)
  letter-spacing 0.06em

.music-player__dl
  display flex
  align-items center
  justify-content center
  width 36px
  height 36px
  border none
  border-radius 50%
  background rgba(15, 118, 110, 0.12)
  color #0f766e
  cursor pointer
  &:disabled
    opacity 0.5

.music-player__head-spacer
  width 36px

.music-player__cover-wrap
  display flex
  justify-content center
  padding 8px 0 16px

.music-player__cover
  width 168px
  height 168px
  border-radius 22px
  object-fit cover
  box-shadow 0 16px 40px rgba(245, 87, 108, 0.22)

  &--fallback
    display flex
    align-items center
    justify-content center
    background linear-gradient(145deg, #db2777 0%, #f43f5e 100%)

  &--spin
    animation music-cover-spin 18s linear infinite

@keyframes music-cover-spin
  from
    transform rotate(0deg)
  to
    transform rotate(360deg)

.music-player__title
  margin 0 0 14px
  text-align center
  font-size 20px
  font-weight 700
  color #0f2744
  line-height 1.35

.music-player__progress
  display flex
  align-items center
  gap 10px
  margin-bottom 12px

.music-player__time
  flex-shrink 0
  width 38px
  font-size 11px
  font-weight 600
  color rgba(15, 39, 68, 0.42)
  text-align center

  &:last-child
    text-align right

.music-player__slider
  flex 1

.music-player__controls
  display flex
  align-items center
  justify-content center
  gap 28px
  margin-bottom 14px

.music-player__ctrl
  display flex
  align-items center
  justify-content center
  min-width 44px
  height 44px
  padding 0 8px
  border none
  border-radius 12px
  background rgba(15, 39, 68, 0.05)
  color rgba(15, 39, 68, 0.62)
  font-size 12px
  font-weight 700
  cursor pointer

.music-player__play
  display flex
  align-items center
  justify-content center
  width 58px
  height 58px
  border none
  border-radius 50%
  background #111827
  box-shadow 0 8px 24px rgba(17, 24, 39, 0.22)
  cursor pointer

.music-player__lyrics-head
  flex-shrink 0
  margin-bottom 8px
  font-size 12px
  font-weight 700
  color rgba(15, 39, 68, 0.45)
  letter-spacing 0.04em

.music-player__lyrics
  flex 1
  min-height 0
  overflow-y auto
  -webkit-overflow-scrolling touch
  padding 12px 8px 20px
  border-radius 16px
  background #fff
  border 1px solid rgba(17, 24, 39, 0.08)
  mask-image linear-gradient(180deg, transparent 0, #000 24px, #000 calc(100% - 24px), transparent 100%)

.music-player__lyric-line
  margin 0
  padding 7px 10px
  font-size 15px
  line-height 1.65
  color rgba(15, 39, 68, 0.38)
  text-align center
  transition color 0.25s ease, transform 0.25s ease, font-size 0.25s ease
  border-radius 10px

  &--section
    margin-top 10px
    font-size 12px
    font-weight 700
    color rgba(15, 39, 68, 0.32)
    letter-spacing 0.04em

  &--meta
    font-size 13px
    color rgba(15, 39, 68, 0.3)

  &--active
    color #0f766e
    font-size 17px
    font-weight 700
    transform scale(1.02)
    background rgba(15, 118, 110, 0.08)

.music-player__lyrics-empty
  margin 0
  padding 24px 12px
  text-align center
  font-size 13px
  color rgba(15, 39, 68, 0.4)

.music-player__audio
  display none
</style>
