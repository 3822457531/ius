<template>
  <div class="sv">
    <div class="sv__nav">
      <button v-if="view !== 'home'" class="sv__nav-btn" type="button" @click="goBack">
        <van-icon name="arrow-left" size="18" />
      </button>
      <span v-else class="sv__nav-btn sv__nav-btn--ghost" />
      <div class="sv__nav-title">{{ navTitle }}</div>
      <button v-if="view !== 'progress'" class="sv__nav-btn" type="button" @click="closePage">
        <van-icon name="cross" size="18" />
      </button>
      <span v-else class="sv__nav-btn sv__nav-btn--ghost" />
    </div>

    <!-- Home -->
    <div v-if="view === 'home'" class="sv__home">
      <div v-if="videoMaintenance" class="sv__maintenance">
        <van-icon name="warning-o" size="22" color="#f59e0b" />
        <p>{{ t('imageApps.aiVideo') }}{{ t('imageApps.maintenance') }}</p>
      </div>
      <!-- <div class="sv__brand">AI视频</div> -->
      <section class="sv__hero">
        <div class="sv__hero-title">AI 视频生成</div>
        <div class="sv__hero-sub">用文字或图片，创造无限可能的视频</div>
        <div class="sv__ill" aria-hidden="true"><div class="sv__ill-core" /></div>
      </section>

      <section class="sv__entry">
        <button type="button" class="sv__entry-card" @click="openForm('t2v')">
          <div class="sv__entry-icon sv__entry-icon--t"><van-icon name="records" size="22" /></div>
          <div class="sv__entry-text">
            <div class="sv__entry-title">文生视频</div>
            <div class="sv__entry-sub">输入文字生成视频</div>
        </div>
        </button>
        <button type="button" class="sv__entry-card" @click="openForm('i2v')">
          <div class="sv__entry-icon sv__entry-icon--i"><van-icon name="live" size="22" /></div>
          <div class="sv__entry-text">
            <div class="sv__entry-title">图生视频</div>
            <div class="sv__entry-sub">上传图片生成视频</div>
          </div>
        </button>
      </section>
     
      <section class="sv__mine">
        <div class="sv__mine-head">
          <div class="sv__mine-title">我的视频</div>
          <button type="button" class="sv__mine-more" @click="openList">全部 <van-icon name="arrow" /></button>
    </div>

        <div v-if="previewList.length" class="sv__thumbs">
          <button v-for="item in previewList" :key="item.id" type="button" class="sv__thumb" @click="openDetail(item)">
            <div class="sv__thumb-media">
              <video v-if="item.status === 'completed' && item.video_url" :src="item.video_url" preload="metadata" muted playsinline />
              <div v-else class="sv__thumb-skeleton">
                <template v-if="item.status === 'processing' || item.status === 'pending'">
                  <van-loading type="spinner" size="18px" />
                  <div class="sv__thumb-state">生成中</div>
                </template>
                <template v-else-if="item.status === 'failed'">
                  <van-icon name="warning-o" size="20" />
                  <div class="sv__thumb-state">失败</div>
                </template>
                <template v-else>
                  <van-icon name="video-o" size="18" />
                </template>
              </div>
              <div class="sv__thumb-play"><van-icon name="play" size="18" /></div>
        </div>
            <div class="sv__thumb-meta">
              <div class="sv__thumb-title">{{ item.prompt || '—' }}</div>
              <div class="sv__thumb-sub">{{ formatDateTime(item.created_at) }}</div>
      </div>
          </button>
        </div>
        <div v-else class="sv__empty">
          <div class="sv__empty-title">还没有视频</div>
          <div class="sv__empty-sub">去生成一个试试吧</div>
        </div>
      </section>
    </div>
    
    <!-- Form -->
    <div v-else-if="view === 'form'" class="sv__form">
      <section v-if="mode === 'i2v'" class="sv__upload">
        <button type="button" class="sv__upload-card" @click="uploadOneImage">
          <div v-if="uploadPreviewUrl" class="sv__upload-preview">
            <img :src="uploadPreviewUrl" alt="" />
            <button type="button" class="sv__upload-remove" @click.stop="clearUpload">
              <van-icon name="cross" size="16" />
            </button>
      </div>
          <div v-else class="sv__upload-placeholder">
            <van-icon name="photo-o" size="28" />
            <div class="sv__upload-tip">点击上传图片</div>
            <div class="sv__upload-sub">支持 JPG / PNG，最大 10MB</div>
    </div>
        </button>
      </section>

      <section class="sv__field">
        <div class="sv__field-title">输入提示词 <span class="sv__field-sub">（必填）</span></div>
      <van-field
          v-model="form.prompt"
        type="textarea"
        rows="6"
          maxlength="1000"
          show-word-limit
          class="sv__textarea"
          :placeholder="t('sora.promptPlaceholder')"
        />
      </section>

      <!-- <section class="sv__section">
        <div class="sv__section-title">选择参数</div>
        <div class="sv__row">
          <div class="sv__row-label">视频时长</div>
          <div class="sv__chips">
            <button v-for="d in durationOptions" :key="d.value" type="button" :class="chipClass(form.duration, d.value)" @click="form.duration = d.value">{{ d.label }}</button>
            </div>
          </div>
        <div class="sv__row">
          <div class="sv__row-label">视频比例</div>
          <div class="sv__chips">
            <button v-for="r in ratioOptions" :key="r.value" type="button" :class="chipClass(form.aspectRatio, r.value)" @click="form.aspectRatio = r.value">{{ r.label }}</button>
        </div>
         </div>
        <div class="sv__row">
          <div class="sv__row-label">视频质量</div>
          <div class="sv__chips">
            <button v-for="q in qualityOptions" :key="q.value" type="button" :class="chipClass(form.quality, q.value)" @click="form.quality = q.value">{{ q.label }}</button>
            </div>
          </div>
      </section> -->

      <section class="sv__cta">
        <van-button type="primary" block size="large" class="sv__cta-btn" :loading="creating" :disabled="!canSubmit" @click="submit">立即生成</van-button>
        <div class="sv__cta-sub">消耗算力：500算力</div>
      </section>
        </div>

    <!-- Progress -->
    <div v-else-if="view === 'progress'" class="sv__progress">
      <div class="sv__ring">
        <div class="sv__ring-circle" :style="ringStyle">
          <div class="sv__rate">{{ progressRate }}%</div>
        </div>
            </div>
      <div class="sv__p-title">视频生成中...</div>
      <!-- <div class="sv__p-sub">预计剩余时间：{{ etaText }}</div> -->
      <div class="sv__steps">
        <div v-for="(s, idx) in stepItems" :key="idx" class="sv__step">
          <div :class="['sv__dot', { 'sv__dot--on': s.done }]"><van-icon v-if="s.done" name="success" size="14" /></div>
          <div class="sv__step-text">{{ s.text }}</div>
          </div>
          </div>
      <!-- <div class="sv__p-foot">请勿关闭页面，生成完成后将自动保存</div> -->
        </div>

    <!-- List -->
    <div v-else-if="view === 'list'" class="sv__list">
      <van-tabs v-model:active="listTab" animated class="sv__tabs">
        <van-tab title="全部" name="all" />
        <van-tab title="生成中" name="processing" />
        <van-tab title="已完成" name="completed" />
        <van-tab title="失败" name="failed" />
      </van-tabs>
      <div class="sv__list-body">
        <div v-if="listLoading" class="sv__center"><van-loading type="spinner" size="22px" /></div>
        <div v-else-if="filteredList.length === 0" class="sv__center"><van-empty description="暂无数据" /></div>
        <div v-else class="sv__cards">
          <div v-for="item in filteredList" :key="item.id" class="sv__card" @click="openDetail(item)">
            <div class="sv__card-media">
              <video v-if="item.status === 'completed' && item.video_url" :src="item.video_url" preload="metadata" muted playsinline />
              <div v-else class="sv__card-placeholder">
                <van-loading v-if="item.status === 'processing' || item.status === 'pending'" type="spinner" size="18px" />
                <van-icon v-else name="warning-o" size="22" />
                <div class="sv__card-ph">{{ statusLabel(item.status) }}</div>
              </div>
              <div v-if="item.status === 'completed'" class="sv__card-play"><van-icon name="play" size="16" /></div>
            </div>
            <div class="sv__card-info">
              <div class="sv__card-title">{{ item.prompt || '—' }}</div>
              <div class="sv__card-sub">
                {{ formatDateTime(item.created_at) }} <span class="sv__sep">·</span> {{ getDurationText(item) }} <span class="sv__sep">·</span> {{ item.aspect_ratio || '-' }}
              </div>
            </div>
            <button
              v-if="canDeleteVideo(item)"
              type="button"
              class="sv__card-delete"
              aria-label="删除"
              @click.stop="confirmDelete(item)"
            >
              <van-icon name="delete-o" size="18" />
            </button>
          </div>
      </div>
    </div>
    </div>

    <!-- Detail -->
    <div v-else-if="view === 'detail'" class="sv__detail">
      <div class="sv__player">
        <video v-if="detailItem?.video_url" :src="detailItem.video_url" controls preload="auto" autoplay playsinline webkit-playsinline x5-playsinline>
                {{ t('sora.videoNotSupport') }}
              </video>
        <div v-else class="sv__player-ph">
          <van-loading v-if="detailItem?.status === 'processing' || detailItem?.status === 'pending'" type="spinner" size="22px" />
          <van-icon v-else name="warning-o" size="28" />
          <div class="sv__player-ph-text">{{ detailItem ? statusLabel(detailItem.status) : '暂无数据' }}</div>
            </div>
            </div>
      <div v-if="detailItem" class="sv__detail-body">
        <div class="sv__detail-title">{{ detailItem.prompt || '—' }}</div>
        <div class="sv__detail-time">{{ formatDateTime(detailItem.created_at) }}</div>
        <div class="sv__detail-desc">{{ detailItem.prompt_ext || detailItem.prompt || '' }}</div>
        <div class="sv__meta">
          <div class="sv__meta-row"><div class="sv__k">视频时长</div><div class="sv__v">{{ getDurationText(detailItem) }}</div></div>
          <div class="sv__meta-row"><div class="sv__k">视频比例</div><div class="sv__v">{{ detailItem.aspect_ratio || '-' }}</div></div>
          <div class="sv__meta-row"><div class="sv__k">视频质量</div><div class="sv__v">{{ detailItem.quality || '-' }}</div></div>
            </div>
        <div class="sv__actions">
          <van-button type="primary" plain class="sv__btn" @click="regenerate(detailItem)"><van-icon name="replay" />重新生成</van-button>
          <van-button type="primary" class="sv__btn" :disabled="!detailItem.video_url" @click="download(detailItem)"><van-icon name="down" />下载视频</van-button>
          <van-button type="default" class="sv__btn" :disabled="!detailItem.video_url" @click="share(detailItem)"><van-icon name="share-o" />分享</van-button>
          <van-button
            v-if="canDeleteVideo(detailItem)"
            type="danger"
            plain
            class="sv__btn sv__btn--delete"
            @click="confirmDelete(detailItem)"
          >
            <van-icon name="delete-o" />删除
          </van-button>
                </div>
              </div>
            </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { showConfirmDialog, showToast } from 'vant'
import { checkSession, getSystemInfo } from '@/store/cache'
import { parseAppFlags } from '@/utils/appFeatures'
import { httpGet, httpPost } from '@/utils/http'

const { t } = useI18n()
const router = useRouter()

const view = ref('home')
const mode = ref('t2v')
const isLogin = ref(false)
const videoMaintenance = ref(true)

async function loadAppFlags() {
  try {
    const sys = await getSystemInfo()
    videoMaintenance.value = !parseAppFlags(sys).video
  } catch {
    videoMaintenance.value = true
  }
}

const form = reactive({ prompt: '', duration: '10', aspectRatio: '9:16', quality: 'hd' })
const creating = ref(false)
const progressRate = ref(0)
const etaSeconds = ref(300)
const pollTimer = ref(null)
const progressStartAt = ref(0)
const progressTotalSeconds = 300

const listTab = ref('all')
const listLoading = ref(false)
const list = ref([])
const detailItem = ref(null)

const uploadPreviewUrl = ref('')
const uploadedImageUrl = ref('')

const durationOptions = [
  { value: '5', label: '5秒' },
  { value: '10', label: '10秒' },
  { value: '20', label: '20秒' },
]
const ratioOptions = [
  { value: '16:9', label: '16:9' },
  { value: '9:16', label: '9:16' },
  { value: '1:1', label: '1:1' },
]
const qualityOptions = [
  { value: 'standard', label: '标清' },
  { value: 'hd', label: '高清' },
  { value: 'uhd', label: '超清' },
]

const canSubmit = computed(() => !!form.prompt.trim() && (mode.value !== 'i2v' || !!uploadedImageUrl.value))
const navTitle = computed(() => (view.value === 'home' ? 'AI视频' : view.value === 'form' ? (mode.value === 'i2v' ? '图生视频' : '文生视频') : view.value === 'progress' ? '生成中' : view.value === 'list' ? '我的视频' : view.value === 'detail' ? '视频详情' : ''))
const previewList = computed(() => (list.value || []).slice(0, 2))
const filteredList = computed(() => (listTab.value === 'all' ? list.value : (list.value || []).filter((x) => x.status === listTab.value)))
const etaText = computed(() => {
  const s = Math.max(0, Number(etaSeconds.value) || 0)
  const mm = String(Math.floor(s / 60)).padStart(2, '0')
  const ss = String(Math.floor(s % 60)).padStart(2, '0')
  return `${mm}:${ss}`
})
const stepItems = computed(() => {
  const p = progressRate.value
  return [
    { text: '模型初始化', done: p >= 5 },
    { text: '提示词分析', done: p >= 25 },
    { text: '视频片段生成', done: p >= 55 },
    { text: '视频合成', done: p >= 85 },
  ]
})
const ringStyle = computed(() => {
  const p = Math.max(0, Math.min(100, Number(progressRate.value) || 0))
  return {
    background: `conic-gradient(#4ea2ff ${p * 3.6}deg, rgba(255,255,255,0.08) 0deg)`,
  }
})

function chipClass(cur, val) {
  return ['sv__chip', { 'sv__chip--active': String(cur) === String(val) }]
}
function statusLabel(status) {
  if (status === 'completed') return '已完成'
  if (status === 'processing') return '生成中'
  if (status === 'pending') return '等待中'
  if (status === 'failed') return '失败'
  return status || '未知'
}
function formatDateTime(v) {
  if (!v) return ''
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return String(v)
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd} ${hh}:${mi}`
}
function getDurationText(item) {
  const v = String(item?.duration || '').trim()
  if (v) return /^\d+(\.\d+)?$/.test(v) ? `${Number(v)}秒` : v
  try {
    if (item?.task_info) {
      const info = JSON.parse(item.task_info)
      const d = info?.params?.duration || info?.duration
      if (d !== undefined && d !== null && String(d).trim() !== '') return `${String(d).trim()}秒`
    }
  } catch {}
  return '-'
}

function canDeleteVideo(item) {
  const status = String(item?.status || '')
  return status === 'completed' || status === 'failed'
}

function confirmDelete(item) {
  if (!item?.id) return
  showConfirmDialog({
    title: t('sora.deleteTitle'),
    message: t('sora.deleteMessage'),
    confirmButtonText: t('sora.deleteConfirmText'),
    cancelButtonText: t('sora.deleteCancelText'),
  })
    .then(() => removeVideo(item))
    .catch(() => {})
}

async function removeVideo(item) {
  if (!item?.id) return
  try {
    await httpGet('/api/sora2/remove', { id: item.id })
    showToast({ type: 'success', message: t('sora.deleteSuccess') })
    if (detailItem.value?.id === item.id) {
      detailItem.value = null
      view.value = 'list'
    }
    list.value = (list.value || []).filter((x) => String(x.id) !== String(item.id))
    await refreshList()
  } catch (e) {
    showToast({ type: 'fail', message: (t('sora.deleteFailPrefix') || '') + (e?.message || e?.msg || '') })
  }
}

function openForm(nextMode) {
  if (videoMaintenance.value) {
    showToast({ type: 'fail', message: `${t('imageApps.aiVideo')}${t('imageApps.maintenance')}` })
    return
  }
  mode.value = nextMode
  view.value = 'form'
  progressRate.value = 0
  etaSeconds.value = progressTotalSeconds
  progressStartAt.value = 0
  if (nextMode === 't2v') clearUpload()
}
function openList() {
  view.value = 'list'
  void refreshList()
}
function openDetail(item) {
  detailItem.value = item
  view.value = 'detail'
}
function goBack() {
  if (view.value === 'detail') return (view.value = 'list')
  if (view.value === 'list') return (view.value = 'home')
  if (view.value === 'progress') return (view.value = 'list')
  if (view.value === 'form') return (view.value = 'home')
  view.value = 'home'
}
function closePage() {
  try {
    router.push({ name: 'mobile-image-hub' })
  } catch {
    router.back()
  }
}

function clearUpload() {
  if (uploadPreviewUrl.value) URL.revokeObjectURL(uploadPreviewUrl.value)
  uploadPreviewUrl.value = ''
  uploadedImageUrl.value = ''
}

async function uploadOneImage() {
  if (!isLogin.value) return requireLogin()
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = 'image/*'
  input.multiple = false
  input.onchange = async (e) => {
    const file = e.target.files?.[0]
    if (!file) return
    try {
      const fd = new FormData()
      fd.append('file', file)
      const res = await httpPost('/api/upload', fd)
      let url = res?.data?.url || ''
      if (url && !/^https?:\/\//i.test(url)) url = location.protocol + '//' + location.host + url
      clearUpload()
      uploadedImageUrl.value = url
      uploadPreviewUrl.value = URL.createObjectURL(file)
      showToast({ type: 'success', message: '图片上传成功' })
  } catch (err) {
      showToast({ type: 'fail', message: err?.message || '图片上传失败' })
    }
  }
  input.click()
}

function stopPolling() {
  if (pollTimer.value) {
    clearInterval(pollTimer.value)
    pollTimer.value = null
  }
}

function normalizeMediaUrl(rawUrl) {
  let url = String(rawUrl || '').trim()
  if (!url) return ''

  if (!/^https?:\/\//i.test(url)) {
    url = `${location.protocol}//${location.host}${url.startsWith('/') ? '' : '/'}${url}`
  }

  try {
    const parsed = new URL(url)
    const isLocal = parsed.hostname === 'localhost' || parsed.hostname === '127.0.0.1'
    if (isLocal && parsed.port === '8888') {
      parsed.port = '5678'
      return parsed.toString()
    }
    return url
  } catch {
    return url
  }
}

async function pollProgress(jobId) {
  stopPolling()
  if (!progressStartAt.value) progressStartAt.value = Date.now()
  pollTimer.value = setInterval(async () => {
    try {
      if (!jobId) return
      const elapsed = Math.max(0, Math.floor((Date.now() - progressStartAt.value) / 1000))
      // 5分钟内若上游未返回成功，进度最高停在99%
      const simulatedRate = Math.min(99, Math.floor((elapsed / progressTotalSeconds) * 100))
      progressRate.value = Math.max(progressRate.value, simulatedRate)
      etaSeconds.value = Math.max(0, progressTotalSeconds - elapsed)
      const res = await httpGet(`/api/sora2/progress/${jobId}`)
      const out = res?.data?.output || res?.data?.video_url
      const apiStatus = String(res?.data?.status || '').toUpperCase()
      const isSuccess = out || apiStatus === 'SUCCESS' || apiStatus === 'COMPLETED'
      if (isSuccess) {
        progressRate.value = 100
        stopPolling()
        showToast({ type: 'success', message: '视频生成成功', duration: 4500 })
        await refreshList()
        const found = (list.value || []).find((x) => String(x.id) === String(jobId)) || null
        if (found) {
          detailItem.value = found
          view.value = 'detail'
  } else {
          view.value = 'list'
        }
      }
    } catch {}
  }, 1000)
}

async function refreshList() {
  if (!isLogin.value) return
  listLoading.value = true
  try {
    const res = await httpGet('/api/sora2/list', { page: 1, page_size: 20 })
    const rows = Array.isArray(res?.data?.data) ? res.data.data : []
    list.value = rows.map((item) => {
      const videoUrl = normalizeMediaUrl(item?.video_url || item?.output || '')
      return {
        ...item,
        video_url: videoUrl,
        output: videoUrl || item?.output || '',
        water_url: normalizeMediaUrl(item?.water_url || ''),
        cover_url: normalizeMediaUrl(item?.cover_url || ''),
      }
    })
  } catch {
    list.value = []
  } finally {
    listLoading.value = false
  }
}

async function submit() {
  if (videoMaintenance.value) {
    showToast({ type: 'fail', message: `${t('imageApps.aiVideo')}${t('imageApps.maintenance')}` })
    return
  }
  if (!form.prompt.trim()) return showToast({ type: 'fail', message: t('sora.videoDescRequired') })
  if (!isLogin.value) return requireLogin()
  if (mode.value === 'i2v' && !uploadedImageUrl.value) return showToast({ type: 'fail', message: '请先上传图片' })

  creating.value = true
  try {
    const payload = {
      prompt: form.prompt,
      model: 'doubao-seedance-2-0-fast-260128',
      aspect_ratio: form.aspectRatio,
      duration: String(form.duration),
      quality: form.quality,
      images: mode.value === 'i2v' ? [uploadedImageUrl.value] : [],
    }
    const res = await httpPost('/api/sora2/generate', payload)
    const taskId = res?.data?.task_id
    if (!taskId) throw new Error(t('sora.generateSubmitNoTaskId'))
    progressStartAt.value = Date.now()
    progressRate.value = 0
    etaSeconds.value = progressTotalSeconds
    view.value = 'progress'
    showToast({ type: 'success', message: res?.data?.message || t('sora.generateSubmitSuccessDefault'), duration: 3500 })
    await refreshList()
    await pollProgress(taskId)
  } catch (e) {
    showToast({ type: 'fail', message: e?.message || t('sora.generateFailDefault') })
  } finally {
    creating.value = false
  }
}

function regenerate(item) {
  if (!item) return
  form.prompt = String(item.prompt || '').trim()
  mode.value = 't2v'
  clearUpload()
  view.value = 'form'
}
function download(item) {
  const url = item?.video_url
  if (!url) return
  const a = document.createElement('a')
  a.href = url
  a.download = `video_${item?.id || Date.now()}.mp4`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}
async function share(item) {
  const url = item?.video_url
  if (!url) return
  try {
    await navigator.clipboard.writeText(url)
    showToast({ type: 'success', message: '链接已复制' })
  } catch {
    showToast({ type: 'fail', message: '复制失败' })
  }
}

function requireLogin() {
  showToast({ type: 'fail', message: '请先登录' })
  try {
    router.push({ name: 'login' })
  } catch {}
}

onMounted(() => {
  void loadAppFlags()
  checkSession()
    .then(() => {
      isLogin.value = true
      void refreshList()
    })
    .catch(() => {
      isLogin.value = false
    })
})
watch(view, (v) => {
  if (v !== 'progress') stopPolling()
})
onUnmounted(() => {
  stopPolling()
  clearUpload()
})
</script>

<style scoped>
.sv{
  --ai-bg:#f4f5f7;--ai-surface:#fff;--ai-text:#111827;--ai-muted:#6b7280;--ai-line:rgba(17,24,39,.08);--ai-accent:#0f766e;
  min-height:100vh;min-height:100dvh;background:radial-gradient(ellipse 80% 45% at 0% 0%,rgba(37,99,235,.07),transparent 55%),radial-gradient(ellipse 70% 40% at 100% 0%,rgba(15,118,110,.05),transparent 50%),var(--ai-bg);
  color:var(--ai-text);padding:18px 16px 24px;box-sizing:border-box;
  font-family:'Plus Jakarta Sans','PingFang SC','Hiragino Sans GB','Microsoft YaHei',sans-serif;-webkit-font-smoothing:antialiased;
}
.sv__maintenance{display:flex;align-items:flex-start;gap:10px;margin-bottom:14px;padding:12px 14px;border-radius:12px;background:#fffbeb;border:1px solid #fcd34d;font-size:13px;line-height:1.5;color:#92400e}
.sv__nav{display:flex;align-items:center;justify-content:space-between;height:44px;margin-bottom:12px}
.sv__nav-btn{width:36px;height:36px;border:none;border-radius:12px;background:#fff;border:1px solid var(--ai-line);color:var(--ai-text);display:flex;align-items:center;justify-content:center;box-shadow:0 1px 2px rgba(17,24,39,.04)}
.sv__nav-btn--ghost{background:transparent;border-color:transparent;box-shadow:none}
.sv__nav-title{font-size:16px;font-weight:800;letter-spacing:-.02em}
.sv__brand{font-size:18px;font-weight:900;letter-spacing:.02em}
.sv__hero{margin-top:8px;border-radius:18px;padding:18px 16px 16px;background:var(--ai-surface);border:1px solid var(--ai-line);box-shadow:0 1px 3px rgba(17,24,39,.04),0 8px 24px rgba(17,24,39,.04)}
.sv__hero-title{font-size:22px;font-weight:800;letter-spacing:-.02em}
.sv__hero-sub{margin-top:6px;font-size:13px;color:var(--ai-muted);line-height:1.45}
.sv__ill{margin-top:14px;height:118px;border-radius:16px;background:radial-gradient(120px 120px at 50% 35%,rgba(37,99,235,.35),rgba(6,182,212,.18) 55%,transparent 70%),#f8fafc;border:1px solid var(--ai-line);position:relative;overflow:hidden}
.sv__ill-core{position:absolute;left:50%;top:52%;width:54px;height:54px;transform:translate(-50%,-50%);border-radius:18px;background:linear-gradient(135deg,#1d4ed8,#06b6d4);box-shadow:0 10px 30px rgba(37,99,235,.22)}
.sv__entry{margin-top:16px;display:grid;gap:12px}
.sv__entry-card{border:none;border-radius:16px;padding:14px;background:var(--ai-surface);border:1px solid var(--ai-line);display:flex;gap:12px;align-items:center;text-align:left;color:var(--ai-text);box-shadow:0 1px 3px rgba(17,24,39,.04)}
.sv__entry-card:active{transform:scale(.985)}
.sv__entry-icon{width:38px;height:38px;border-radius:14px;display:flex;align-items:center;justify-content:center;color:#fff}
.sv__entry-icon--t{background:linear-gradient(135deg,#2563eb,#4f46e5)}
.sv__entry-icon--i{background:linear-gradient(135deg,#06b6d4,#0f766e)}
.sv__entry-title{font-size:15px;font-weight:800}
.sv__entry-sub{margin-top:2px;font-size:12px;color:var(--ai-muted)}
.sv__mine{margin-top:18px}
.sv__mine-head{display:flex;align-items:center;justify-content:space-between;padding:0 2px}
.sv__mine-title{font-size:14px;font-weight:800;color:var(--ai-text)}
.sv__mine-more{border:none;background:transparent;color:var(--ai-accent);font-size:12px;font-weight:600;display:inline-flex;align-items:center;gap:4px}
.sv__thumbs{margin-top:12px;display:grid;grid-template-columns:repeat(2,1fr);gap:12px}
.sv__thumb{border:none;background:transparent;text-align:left;color:var(--ai-text);padding:0}
.sv__thumb-media{position:relative;width:100%;border-radius:14px;overflow:hidden;background:#e5e7eb;border:1px solid var(--ai-line);aspect-ratio:16/10}
.sv__thumb-media video{width:100%;height:100%;object-fit:cover}
.sv__thumb-skeleton{width:100%;height:100%;display:flex;flex-direction:column;gap:6px;align-items:center;justify-content:center;color:var(--ai-muted);background:#f3f4f6}
.sv__thumb-state{font-size:11px;color:var(--ai-muted)}
.sv__thumb-play{position:absolute;left:10px;bottom:10px;width:30px;height:30px;border-radius:999px;background:rgba(0,0,0,.45);color:#fff;display:flex;align-items:center;justify-content:center}
.sv__thumb-meta{margin-top:8px}
.sv__thumb-title{font-size:12px;font-weight:700;line-height:1.25;overflow:hidden;display:-webkit-box;-webkit-line-clamp:2;line-clamp:2;-webkit-box-orient:vertical}
.sv__thumb-sub{margin-top:4px;font-size:11px;color:var(--ai-muted)}
.sv__empty{margin-top:12px;border-radius:16px;padding:18px 14px;background:var(--ai-surface);border:1px solid var(--ai-line);text-align:center}
.sv__empty-title{font-size:13px;font-weight:800}
.sv__empty-sub{margin-top:4px;font-size:12px;color:var(--ai-muted)}
.sv__form{display:flex;flex-direction:column;gap:14px}
.sv__upload-card{width:100%;border:none;border-radius:16px;background:var(--ai-surface);border:1px solid var(--ai-line);overflow:hidden;padding:0}
.sv__upload-placeholder{height:170px;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:6px;color:var(--ai-muted);background:#f9fafb}
.sv__upload-tip{font-size:13px;font-weight:800;color:var(--ai-text)}
.sv__upload-sub{font-size:11px;color:var(--ai-muted)}
.sv__upload-preview{position:relative;height:170px}
.sv__upload-preview img{width:100%;height:100%;object-fit:cover}
.sv__upload-remove{position:absolute;top:10px;right:10px;width:30px;height:30px;border:none;border-radius:999px;background:rgba(0,0,0,.45);color:#fff;display:flex;align-items:center;justify-content:center}
.sv__field-title{padding:0 2px;font-size:13px;font-weight:800;color:var(--ai-text)}
.sv__field-sub{font-size:12px;font-weight:600;color:var(--ai-muted)}
.sv__textarea{margin-top:10px;border-radius:16px;background:var(--ai-surface);border:1px solid var(--ai-line);overflow:hidden}
.sv__textarea :deep(.van-field__body){align-items:flex-start}
.sv__textarea :deep(.van-field__control){color:var(--ai-text);-webkit-text-fill-color:var(--ai-text);caret-color:var(--ai-accent);font-weight:500}
.sv__textarea :deep(.van-field__control::placeholder){color:#9ca3af}
.sv__textarea :deep(.van-field__word-limit){color:var(--ai-muted)}
.sv__section{border-radius:16px;padding:14px 12px;background:var(--ai-surface);border:1px solid var(--ai-line)}
.sv__section-title{font-size:13px;font-weight:800;margin-bottom:12px}
.sv__row{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:8px 2px}
.sv__row-label{font-size:12px;font-weight:700;color:var(--ai-muted);flex-shrink:0}
.sv__chips{display:flex;gap:8px;flex-wrap:wrap;justify-content:flex-end}
.sv__chip{border:none;padding:8px 12px;border-radius:999px;background:#f3f4f6;color:#4b5563;font-size:12px;font-weight:700}
.sv__chip--active{background:#111827;color:#fff;box-shadow:0 6px 16px rgba(17,24,39,.16)}
.sv__cta-btn{border-radius:14px;height:46px;font-weight:800;background:#111827!important;border-color:#111827!important}
.sv__cta-sub{margin-top:8px;text-align:center;font-size:11px;color:var(--ai-muted)}
.sv__ring{display:flex;justify-content:center;padding-top:18px}
.sv__ring-circle{position:relative;width:170px;height:170px;border-radius:50%;display:flex;align-items:center;justify-content:center;transition:background .9s linear}
.sv__ring-circle::before{content:'';position:absolute;inset:10px;border-radius:50%;background:#fff;box-shadow:inset 0 0 0 1px var(--ai-line)}
.sv__rate{position:relative;z-index:1;font-size:28px;font-weight:900;color:var(--ai-text)}
.sv__p-title{margin-top:14px;text-align:center;font-size:16px;font-weight:800}
.sv__p-sub{margin-top:6px;text-align:center;font-size:12px;color:var(--ai-muted)}
.sv__steps{margin-top:18px;padding:14px;border-radius:16px;background:var(--ai-surface);border:1px solid var(--ai-line)}
.sv__step{display:flex;align-items:center;gap:10px}
.sv__step+.sv__step{margin-top:10px}
.sv__dot{width:20px;height:20px;border-radius:999px;background:#f3f4f6;border:1px solid var(--ai-line);display:flex;align-items:center;justify-content:center}
.sv__dot--on{background:rgba(15,118,110,.15);border-color:rgba(15,118,110,.35);color:var(--ai-accent)}
.sv__step-text{font-size:13px;font-weight:700;color:#374151}
.sv__p-foot{margin-top:16px;text-align:center;font-size:11px;color:var(--ai-muted)}
.sv__list{display:flex;flex-direction:column;min-height:0}
.sv__tabs{--van-tab-text-color:var(--ai-muted);--van-tab-active-text-color:var(--ai-text);--van-tabs-bottom-bar-color:#0f766e}
.sv__tabs :deep(.van-tabs__wrap){background:var(--ai-surface);border-radius:14px;border:1px solid var(--ai-line);overflow:hidden}
.sv__tabs :deep(.van-tab){color:var(--ai-muted)}
.sv__tabs :deep(.van-tab--active){color:var(--ai-text)}
.sv__tabs :deep(.van-tabs__line){background:var(--ai-accent)}
.sv__tabs :deep(.van-tabs__nav){background:transparent}
.sv__tabs :deep(.van-tab__text){color:inherit}
.sv__list-body{margin-top:12px}
.sv__center{padding:26px 0;display:flex;justify-content:center}
.sv__cards{display:grid;gap:12px}
.sv__card{border:none;border-radius:16px;background:var(--ai-surface);border:1px solid var(--ai-line);padding:12px;display:grid;grid-template-columns:110px 1fr auto;gap:12px;align-items:center;text-align:left;color:var(--ai-text);position:relative;cursor:pointer;box-shadow:0 1px 3px rgba(17,24,39,.04)}
.sv__card-delete{flex-shrink:0;width:36px;height:36px;border:none;border-radius:12px;background:#f3f4f6;color:#6b7280;display:flex;align-items:center;justify-content:center}
.sv__card-delete:active{background:#fef2f2;color:#dc2626}
.sv__card-media{position:relative;border-radius:14px;overflow:hidden;background:#e5e7eb;border:1px solid var(--ai-line);width:110px;height:74px}
.sv__card-media video{width:100%;height:100%;object-fit:cover}
.sv__card-placeholder{width:100%;height:100%;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:6px;background:#f3f4f6;color:var(--ai-muted)}
.sv__card-ph{font-size:11px;color:var(--ai-muted)}
.sv__card-play{position:absolute;left:8px;bottom:8px;width:26px;height:26px;border-radius:999px;background:rgba(0,0,0,.45);color:#fff;display:flex;align-items:center;justify-content:center}
.sv__card-title{font-size:13px;font-weight:800;line-height:1.25;overflow:hidden;display:-webkit-box;-webkit-line-clamp:2;line-clamp:2;-webkit-box-orient:vertical}
.sv__card-sub{margin-top:6px;font-size:11px;color:var(--ai-muted)}
.sv__sep{margin:0 6px;opacity:.7}
.sv__player{border-radius:16px;overflow:hidden;border:1px solid var(--ai-line);background:#111827;aspect-ratio:16/10}
.sv__player video{width:100%;height:100%;object-fit:contain;background:#000}
.sv__player-ph{width:100%;height:100%;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:8px;color:#9ca3af}
.sv__player-ph-text{font-size:12px;color:#9ca3af}
.sv__detail-body{margin-top:14px}
.sv__detail-title{font-size:16px;font-weight:800;line-height:1.25}
.sv__detail-time{margin-top:6px;font-size:11px;color:var(--ai-muted)}
.sv__detail-desc{margin-top:12px;padding:12px;border-radius:14px;background:var(--ai-surface);border:1px solid var(--ai-line);font-size:12px;line-height:1.6;color:#374151;white-space:pre-wrap}
.sv__meta{margin-top:12px;border-radius:14px;background:var(--ai-surface);border:1px solid var(--ai-line);overflow:hidden}
.sv__meta-row{display:flex;justify-content:space-between;padding:10px 12px}
.sv__meta-row+.sv__meta-row{border-top:1px solid var(--ai-line)}
.sv__k{font-size:12px;font-weight:700;color:var(--ai-muted)}
.sv__v{font-size:12px;font-weight:800;color:var(--ai-text)}
.sv__actions{margin-top:14px;display:grid;grid-template-columns:repeat(2,1fr);gap:10px}
.sv__btn{border-radius:14px;font-weight:800}
.sv__btn--delete{grid-column:1/-1}
</style>
