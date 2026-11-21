<template>
  <div class="cs-page">
    <van-nav-bar
      :title="t('contact.title')"
      left-arrow
      fixed
      placeholder
      @click-left="router.back()"
    >
      <template #right>
        <span class="cs-online" :class="{ 'is-on': adminOnline }">
          <i class="cs-online__dot" />
          {{ adminOnline ? t('contact.online') : t('contact.offline') }}
        </span>
      </template>
    </van-nav-bar>

    <div v-if="!adminOnline" class="cs-wait-banner">
      <van-icon name="clock-o" />
      <span>{{ offlineTip || t('contact.waitOnline') }}</span>
    </div>

    <div ref="listRef" class="cs-messages" @scroll="onScroll">
      <div v-if="loading" class="cs-empty">{{ t('contact.loading') }}</div>
      <div v-else-if="messages.length === 0" class="cs-empty">
        <van-icon name="service-o" />
        <p>{{ t('contact.emptyHint') }}</p>
      </div>
      <div
        v-for="msg in messages"
        :key="msg.id || msg._key"
        class="cs-msg"
        :class="'cs-msg--' + msg.role"
      >
        <div v-if="msg.role === 'system'" class="cs-msg__system">{{ msg.content }}</div>
        <template v-else>
          <div class="cs-msg__bubble" :class="{ 'is-image': msg.msg_type === 'image' }">
            <img
              v-if="msg.msg_type === 'image'"
              :src="msg.content"
              alt="img"
              class="cs-msg__img"
              @click="previewImage(msg.content)"
            />
            <template v-else>{{ msg.content }}</template>
          </div>
          <div class="cs-msg__time">{{ formatTime(msg.created_at) }}</div>
        </template>
      </div>
    </div>

    <div class="cs-composer">
      <button
        type="button"
        class="cs-img-btn"
        :disabled="uploading || !getUserToken()"
        @click="pickImage"
      >
        <van-icon :name="uploading ? 'loading' : 'photo-o'" />
      </button>
      <input
        ref="fileInput"
        type="file"
        accept="image/*"
        class="cs-file-input"
        @change="onFileChange"
      />
      <van-field
        v-model="draft"
        rows="1"
        autosize
        type="textarea"
        maxlength="2000"
        :placeholder="adminOnline ? t('contact.placeholder') : t('contact.placeholderLeave')"
        :border="false"
        @keydown.enter.exact.prevent="send"
      />
      <button
        type="button"
        class="cs-send"
        :disabled="!canSend"
        @click="send"
      >
        {{ t('contact.send') }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { showFailToast, showToast, showImagePreview } from 'vant'
import { httpGet, httpPost, resolveWsHost } from '@/utils/http'
import { getUserToken } from '@/store/session'
import { playCsNotifySound } from '@/utils/csNotify'

defineOptions({ name: 'Contact' })

const { t } = useI18n()
const router = useRouter()

const messages = ref([])
const draft = ref('')
const loading = ref(true)
const adminOnline = ref(false)
const offlineTip = ref('')
const listRef = ref(null)
const fileInput = ref(null)
const sending = ref(false)
const uploading = ref(false)
let socket = null
let pingTimer = null
let reconnectTimer = null
let destroyed = false
let stickBottom = true
let msgSeq = 0

const canSend = computed(() => {
  return !!draft.value.trim() && !sending.value && !uploading.value && !!getUserToken()
})

const previewImage = (url) => {
  if (!url) return
  showImagePreview([url])
}

const pickImage = () => {
  if (uploading.value || !getUserToken()) return
  fileInput.value?.click()
}

const deliverMessage = async (content, msgType = 'text') => {
  if (socket?.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify({ type: 'message', content, msg_type: msgType }))
    return
  }
  const res = await httpPost('/api/cs/send', { content, msg_type: msgType })
  if (res.data?.message) pushMessage(res.data.message)
  if (typeof res.data?.admin_online === 'boolean') {
    adminOnline.value = res.data.admin_online
  }
  if (!res.data?.admin_online) {
    offlineTip.value = t('contact.waitOnline')
  }
}

const onFileChange = async (e) => {
  const file = e.target?.files?.[0]
  e.target.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) {
    showFailToast(t('contact.imageTypeFail'))
    return
  }
  if (file.size > 8 * 1024 * 1024) {
    showFailToast(t('contact.imageTooLarge'))
    return
  }
  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', file)
    const res = await httpPost('/api/upload', formData)
    const url = res.data?.url || res.data?.URL
    if (!url) throw new Error('no url')
    await deliverMessage(url, 'image')
  } catch (err) {
    showFailToast(t('contact.imageFail') + (err?.message ? `: ${err.message}` : ''))
  } finally {
    uploading.value = false
  }
}

const formatTime = (ts) => {
  if (!ts) return ''
  const d = new Date(ts * 1000)
  const pad = (n) => String(n).padStart(2, '0')
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const scrollToBottom = async (force = false) => {
  await nextTick()
  const el = listRef.value
  if (!el) return
  if (force || stickBottom) {
    el.scrollTop = el.scrollHeight
  }
}

const onScroll = () => {
  const el = listRef.value
  if (!el) return
  stickBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80
}

const pushMessage = (msg) => {
  if (!msg) return
  if (msg.id && messages.value.some((m) => m.id === msg.id)) return
  if (!msg.id) {
    msg._key = `tmp-${++msgSeq}`
  }
  messages.value.push(msg)
  scrollToBottom()
}

const applyInit = (data) => {
  if (Array.isArray(data.messages)) {
    messages.value = data.messages
  }
  if (typeof data.admin_online === 'boolean') {
    adminOnline.value = data.admin_online
  }
  if (data.tip) {
    offlineTip.value = data.tip
  } else if (!data.admin_online) {
    offlineTip.value = t('contact.waitOnline')
  }
  loading.value = false
  scrollToBottom(true)
}

const handlePacket = (raw) => {
  let pkt
  try {
    pkt = JSON.parse(raw)
  } catch {
    return
  }
  switch (pkt.type) {
    case 'init':
      applyInit(pkt)
      break
    case 'status':
      adminOnline.value = !!pkt.admin_online
      if (!pkt.admin_online && pkt.content) {
        offlineTip.value = pkt.content
      }
      break
    case 'message':
      if (pkt.message) {
        if (pkt.message.role === 'admin') {
          playCsNotifySound()
        }
        pushMessage(pkt.message)
      }
      break
    case 'error':
      if (pkt.error) showFailToast(pkt.error)
      break
    default:
      break
  }
}

const clearTimers = () => {
  if (pingTimer) {
    clearInterval(pingTimer)
    pingTimer = null
  }
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
}

const connect = () => {
  const token = getUserToken()
  if (!token) {
    loading.value = false
    showToast(t('contact.needLogin'))
    router.push('/login')
    return
  }
  if (socket) {
    try { socket.close() } catch { /* ignore */ }
    socket = null
  }
  const host = resolveWsHost()
  const ws = new WebSocket(`${host}/api/cs/ws?token=${encodeURIComponent(token)}`)
  socket = ws
  ws.onopen = () => {
    clearTimers()
    pingTimer = setInterval(() => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'ping' }))
      }
    }, 25000)
  }
  ws.onmessage = (ev) => handlePacket(ev.data)
  ws.onclose = () => {
    clearTimers()
    if (destroyed) return
    reconnectTimer = setTimeout(connect, 3000)
  }
  ws.onerror = () => {
    // onclose will reconnect
  }
}

const loadHistoryFallback = async () => {
  try {
    const res = await httpGet('/api/cs/history')
    applyInit(res.data || {})
  } catch (e) {
    loading.value = false
    if (e?.code === 400 || String(e?.message || '').toLowerCase().includes('auth')) {
      showToast(t('contact.needLogin'))
      router.push('/login')
    }
  }
}

const send = async () => {
  const content = draft.value.trim()
  if (!content || !canSend.value) return
  sending.value = true
  draft.value = ''
  try {
    await deliverMessage(content, 'text')
  } catch {
    draft.value = content
    showFailToast(t('contact.sendFail'))
  } finally {
    sending.value = false
  }
}

onMounted(async () => {
  try {
    const res = await httpGet('/api/user/profile')
    const flag = res?.data?.is_cs
    if (flag === true || flag === 1 || flag === '1') {
      await router.replace('/mobile/cs-desk')
      return
    }
  } catch {
    // ignore, fall through to user chat
  }
  await loadHistoryFallback()
  connect()
})

onUnmounted(() => {
  destroyed = true
  clearTimers()
  if (socket) {
    try { socket.close() } catch { /* ignore */ }
    socket = null
  }
})
</script>

<style lang="stylus" scoped>
.cs-page
  position relative
  z-index 1
  display flex
  flex-direction column
  height 100vh
  height 100dvh
  box-sizing border-box
  background #f3f6fb
  color #1f2d3d

  :deep(.van-nav-bar)
    background rgba(255, 255, 255, 0.96)
    backdrop-filter blur(10px)

.cs-online
  display inline-flex
  align-items center
  gap 5px
  font-size 12px
  color #8a97a5
  font-weight 600

  &__dot
    width 7px
    height 7px
    border-radius 50%
    background #c0c8d2

  &.is-on
    color #07c160
    .cs-online__dot
      background #07c160
      box-shadow 0 0 0 3px rgba(7, 193, 96, 0.18)

.cs-wait-banner
  display flex
  align-items center
  gap 8px
  margin 8px 12px 0
  padding 10px 12px
  border-radius 12px
  background #fff7e8
  color #b76a00
  font-size 12px
  line-height 1.4
  border 1px solid rgba(255, 183, 77, 0.35)

.cs-messages
  flex 1
  overflow-y auto
  -webkit-overflow-scrolling touch
  padding 12px 12px 8px

.cs-empty
  text-align center
  padding 48px 20px
  color #8a97a5
  font-size 13px
  :deep(.van-icon)
    font-size 36px
    color #b7c3d0
    margin-bottom 8px
  p
    margin 0

.cs-msg
  display flex
  flex-direction column
  margin-bottom 12px
  max-width 82%

  &--user
    margin-left auto
    align-items flex-end
    .cs-msg__bubble
      background linear-gradient(135deg, #4facfe, #3f7dff)
      color #fff
      border-bottom-right-radius 6px

  &--admin
    margin-right auto
    align-items flex-start
    .cs-msg__bubble
      background #fff
      color #1f2d3d
      border 1px solid rgba(20, 60, 100, 0.06)
      border-bottom-left-radius 6px

  &--system
    max-width 100%
    margin 4px auto 12px
    align-items center

  &__bubble
    padding 10px 12px
    border-radius 16px
    font-size 14px
    line-height 1.5
    word-break break-word
    white-space pre-wrap
    box-shadow 0 4px 14px rgba(20, 60, 100, 0.06)
    &.is-image
      padding 4px
      background #fff !important
      color inherit

  &__img
    display block
    max-width 200px
    max-height 260px
    border-radius 12px
    object-fit cover
    cursor pointer

  &__time
    margin-top 4px
    font-size 11px
    color #95a1ad

  &__system
    padding 6px 12px
    border-radius 999px
    background rgba(148, 163, 184, 0.15)
    color #64748b
    font-size 12px
    text-align center

.cs-composer
  display flex
  align-items flex-end
  gap 8px
  padding 8px 10px calc(8px + env(safe-area-inset-bottom, 0px))
  background #fff
  border-top 1px solid rgba(20, 60, 100, 0.06)

  :deep(.van-field)
    flex 1
    background #f3f6fb
    border-radius 14px
    padding 6px 10px

.cs-file-input
  display none

.cs-img-btn
  flex-shrink 0
  width 36px
  height 36px
  border none
  border-radius 12px
  background #eef3fb
  color #3f7dff
  display flex
  align-items center
  justify-content center
  cursor pointer
  :deep(.van-icon)
    font-size 20px
  &:disabled
    opacity 0.45
  &:active:not(:disabled)
    transform scale(0.96)

.cs-send
  flex-shrink 0
  height 36px
  padding 0 14px
  border none
  border-radius 12px
  background linear-gradient(135deg, #4facfe, #3f7dff)
  color #fff
  font-size 13px
  font-weight 700
  cursor pointer
  &:disabled
    opacity 0.45
  &:active:not(:disabled)
    transform scale(0.97)
</style>
