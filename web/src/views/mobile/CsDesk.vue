<template>
  <div class="cs-desk-page">
    <van-nav-bar
      title="客服工作台"
      left-arrow
      fixed
      placeholder
      @click-left="onBack"
    >
      <template #right>
        <span class="cs-desk-live" :class="{ on: wsConnected }">
          {{ wsConnected ? '已接入' : '连接中' }}
        </span>
      </template>
    </van-nav-bar>

    <div v-if="!current" class="cs-desk-list">
      <div class="cs-desk-tabs">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          :class="{ active: filter === tab.key }"
          @click="filter = tab.key"
        >
          {{ tab.label }}
          <em v-if="tab.key === 'unread' && unreadCount">{{ unreadCount > 99 ? '99+' : unreadCount }}</em>
        </button>
      </div>
      <van-search v-model="keyword" placeholder="搜索 UID / 用户名" clearable />
      <van-pull-refresh v-model="refreshing" @refresh="onRefresh">
        <div v-if="!filteredSessions.length" class="cs-desk-empty">暂无待接入会话</div>
        <button
          v-for="item in filteredSessions"
          :key="item.id"
          type="button"
          class="cs-desk-sess"
          @click="selectSession(item)"
        >
          <div class="cs-desk-sess__av">
            {{ avatarText(item) }}
            <i :class="{ on: item.user_online }" />
          </div>
          <div class="cs-desk-sess__main">
            <div class="cs-desk-sess__line">
              <strong>{{ displayName(item) }}</strong>
              <time>{{ formatTime(item.last_msg_at) }}</time>
            </div>
            <div class="cs-desk-sess__line">
              <p>{{ item.last_msg || '暂无消息' }}</p>
              <em v-if="item.admin_unread > 0">{{ item.admin_unread }}</em>
            </div>
          </div>
        </button>
      </van-pull-refresh>
    </div>

    <div v-else class="cs-desk-chat">
      <header class="cs-desk-chat__hd">
        <button type="button" class="cs-desk-chat__back" @click="backToList">
          <van-icon name="arrow-left" />
        </button>
        <div class="cs-desk-chat__info">
          <div class="cs-desk-chat__name">
            {{ displayName(current) }}
            <span class="cs-pill" :class="current.user_online ? 'ok' : 'off'">
              {{ current.user_online ? '在线' : '离线' }}
            </span>
          </div>
          <div class="cs-desk-chat__meta">UID {{ current.user_id }}</div>
        </div>
        <van-button size="mini" type="danger" plain @click="closeSession">结束</van-button>
      </header>

      <div ref="msgBox" class="cs-desk-chat__body">
        <div
          v-for="msg in messages"
          :key="msg.id || msg._key"
          class="cs-bubble"
          :class="'is-' + msg.role"
        >
          <div class="cs-bubble__text" :class="{ 'is-image': msg.msg_type === 'image' }">
            <img
              v-if="msg.msg_type === 'image'"
              :src="msg.content"
              class="cs-bubble__img"
              alt=""
              @click="previewImage(msg.content)"
            >
            <template v-else>{{ msg.content }}</template>
          </div>
          <div class="cs-bubble__time">{{ formatTime(msg.created_at) }}</div>
        </div>
      </div>

      <footer class="cs-desk-chat__ft">
        <div v-if="!current.user_online" class="cs-desk-chat__tip">
          用户离线：回复将作为留言，对方上线后可见
        </div>
        <div class="cs-desk-chat__editor">
          <textarea
            v-model="draft"
            rows="2"
            maxlength="2000"
            placeholder="输入回复…"
            @keydown.enter.exact.prevent="send"
          />
          <van-button
            type="primary"
            size="small"
            :disabled="!draft.trim() || sending"
            :loading="sending"
            @click="send"
          >
            发送
          </van-button>
        </div>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { showConfirmDialog, showFailToast, showImagePreview, showSuccessToast, showToast } from 'vant'
import { httpGet, httpPost, resolveWsHost } from '@/utils/http'
import { getUserToken } from '@/store/session'
import { playCsNotifySound } from '@/utils/csNotify'

defineOptions({ name: 'MobileCsDesk' })

const router = useRouter()
const tabs = [
  { key: 'all', label: '全部' },
  { key: 'unread', label: '未读' },
  { key: 'online', label: '在线' },
]

const sessions = ref([])
const current = ref(null)
const currentId = ref(0)
const messages = ref([])
const draft = ref('')
const msgBox = ref(null)
const wsConnected = ref(false)
const filter = ref('all')
const keyword = ref('')
const sending = ref(false)
const refreshing = ref(false)

let socket = null
let pingTimer = null
let reconnectTimer = null
let destroyed = false
let msgSeq = 0

const unreadCount = computed(() =>
  sessions.value.reduce((n, s) => n + (s.admin_unread > 0 ? 1 : 0), 0)
)

const filteredSessions = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return sessions.value.filter((s) => {
    if (filter.value === 'unread' && !(s.admin_unread > 0)) return false
    if (filter.value === 'online' && !s.user_online) return false
    if (!q) return true
    return String(s.user_id).includes(q)
      || String(s.username || '').toLowerCase().includes(q)
      || String(s.nickname || '').toLowerCase().includes(q)
  })
})

const displayName = (item) => item?.username || item?.nickname || (`用户#${item?.user_id || ''}`)
const avatarText = (item) => String(displayName(item)).slice(0, 1).toUpperCase()

const formatTime = (ts) => {
  if (!ts) return ''
  const d = new Date(Number(ts) * 1000)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n) => String(n).padStart(2, '0')
  const now = new Date()
  if (d.toDateString() === now.toDateString()) {
    return `${pad(d.getHours())}:${pad(d.getMinutes())}`
  }
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const sortSessions = (list) => {
  list.sort((a, b) => {
    const ua = a.admin_unread > 0 ? 1 : 0
    const ub = b.admin_unread > 0 ? 1 : 0
    if (ua !== ub) return ub - ua
    return (b.last_msg_at || 0) - (a.last_msg_at || 0)
  })
}

const scrollBottom = async () => {
  await nextTick()
  const el = msgBox.value
  if (el) el.scrollTop = el.scrollHeight
}

const upsertSession = (s) => {
  if (!s?.id) return
  const idx = sessions.value.findIndex((x) => x.id === s.id)
  if (idx >= 0) {
    sessions.value.splice(idx, 1, { ...sessions.value[idx], ...s })
  } else {
    sessions.value.unshift(s)
  }
  sortSessions(sessions.value)
  if (currentId.value === s.id) {
    current.value = { ...current.value, ...s }
  }
}

const loadSessions = async () => {
  const res = await httpPost('/api/cs/agent/sessions', { status: 0, page: 1, page_size: 100 })
  sessions.value = Array.isArray(res.data?.items) ? res.data.items : []
  sortSessions(sessions.value)
}

const onRefresh = async () => {
  try {
    await loadSessions()
  } catch (e) {
    showFailToast(e.message || '刷新失败')
  } finally {
    refreshing.value = false
  }
}

const selectSession = async (item) => {
  current.value = item
  currentId.value = item.id
  try {
    const res = await httpGet(`/api/cs/agent/messages?session_id=${item.id}`)
    messages.value = res.data?.messages || []
    if (res.data?.session) {
      current.value = res.data.session
      upsertSession({ ...res.data.session, admin_unread: 0 })
    } else {
      upsertSession({ ...item, admin_unread: 0 })
    }
    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'read', session_id: item.id }))
    }
    scrollBottom()
  } catch (e) {
    showFailToast(e.message || '加载消息失败')
  }
}

const backToList = () => {
  current.value = null
  currentId.value = 0
  messages.value = []
  draft.value = ''
  loadSessions().catch(() => {})
}

const closeSession = async () => {
  if (!currentId.value) return
  try {
    await showConfirmDialog({ title: '提示', message: '确定结束该会话？' })
    await httpGet(`/api/cs/agent/close?session_id=${currentId.value}`)
    sessions.value = sessions.value.filter((s) => s.id !== currentId.value)
    backToList()
    showSuccessToast('已结束会话')
  } catch (e) {
    if (e !== 'cancel') showFailToast(e.message || '操作失败')
  }
}

const previewImage = (url) => {
  if (!url) return
  showImagePreview([url])
}

const pushMsg = (msg) => {
  if (!msg) return
  if (msg.session_id && currentId.value && msg.session_id !== currentId.value) return
  if (msg.id && messages.value.some((m) => m.id === msg.id)) return
  if (!msg.id) msg._key = `t-${++msgSeq}`
  messages.value.push(msg)
  scrollBottom()
}

const handlePacket = (raw) => {
  let pkt
  try {
    pkt = JSON.parse(raw)
  } catch {
    return
  }
  if (pkt.type === 'init') {
    if (Array.isArray(pkt.sessions)) {
      sessions.value = pkt.sessions
      sortSessions(sessions.value)
    }
    return
  }
  if (pkt.type === 'user_presence') {
    sessions.value.forEach((s) => {
      if (s.user_id === pkt.user_id) s.user_online = !!pkt.user_online
    })
    if (current.value?.user_id === pkt.user_id) {
      current.value = { ...current.value, user_online: !!pkt.user_online }
    }
    return
  }
  if (pkt.type === 'message') {
    if (pkt.session) upsertSession(pkt.session)
    if (pkt.message) {
      if (pkt.message.role === 'user') {
        playCsNotifySound()
      }
      if (pkt.message.session_id === currentId.value) {
        pushMsg(pkt.message)
        upsertSession({ ...(pkt.session || {}), id: pkt.message.session_id, admin_unread: 0 })
      }
    }
    return
  }
  if (pkt.type === 'error' && pkt.error) showFailToast(pkt.error)
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
    showToast('请先登录')
    return
  }
  if (socket) {
    try { socket.close() } catch { /* ignore */ }
    socket = null
  }
  const host = resolveWsHost()
  const ws = new WebSocket(`${host}/api/cs/agent/ws?token=${encodeURIComponent(token)}`)
  socket = ws
  ws.onopen = () => {
    wsConnected.value = true
    clearTimers()
    pingTimer = setInterval(() => {
      if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'ping' }))
    }, 25000)
  }
  ws.onmessage = (ev) => handlePacket(ev.data)
  ws.onclose = () => {
    wsConnected.value = false
    clearTimers()
    if (!destroyed) reconnectTimer = setTimeout(connect, 3000)
  }
  ws.onerror = () => {}
}

const send = async () => {
  const content = draft.value.trim()
  if (!content || !currentId.value || sending.value) return
  sending.value = true
  const payload = { session_id: currentId.value, content }
  draft.value = ''
  try {
    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'message', ...payload }))
    } else {
      const res = await httpPost('/api/cs/agent/reply', payload)
      if (res.data?.message) pushMsg(res.data.message)
      if (res.data?.session) upsertSession(res.data.session)
    }
  } catch (e) {
    draft.value = content
    showFailToast(e.message || '发送失败')
  } finally {
    sending.value = false
  }
}

const onBack = () => {
  if (current.value) {
    backToList()
    return
  }
  router.back()
}

onMounted(async () => {
  try {
    const profile = await httpGet('/api/user/profile')
    if (!profile.data?.is_cs && profile.data?.is_cs !== 1 && profile.data?.is_cs !== '1') {
      showFailToast('无客服权限')
      router.replace('/mobile/invitation')
      return
    }
    await loadSessions()
    connect()
  } catch (e) {
    showFailToast(e.message || '加载失败')
  }
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

<style scoped lang="stylus">
.cs-desk-page
  min-height 100vh
  background #f5f7fb
  color #1f2d3d

.cs-desk-live
  font-size 12px
  color #98a2b3
  &.on
    color #067647

.cs-desk-tabs
  display flex
  gap 8px
  padding 10px 12px 0
  button
    border none
    background #fff
    border-radius 999px
    padding 6px 12px
    font-size 13px
    color #667085
    position relative
    &.active
      background #2f6bff
      color #fff
    em
      position absolute
      top -4px
      right -4px
      min-width 16px
      height 16px
      padding 0 4px
      border-radius 999px
      background #f04438
      color #fff
      font-size 10px
      line-height 16px
      font-style normal
      text-align center

.cs-desk-empty
  padding 48px 16px
  text-align center
  color #98a2b3
  font-size 14px

.cs-desk-sess
  width 100%
  display flex
  gap 12px
  padding 12px 14px
  border 0
  border-bottom 1px solid #eef2f6
  background #fff
  text-align left
  &__av
    position relative
    width 42px
    height 42px
    border-radius 50%
    background #dbe4ff
    color #2f6bff
    display flex
    align-items center
    justify-content center
    font-weight 700
    flex-shrink 0
    i
      position absolute
      right 0
      bottom 0
      width 10px
      height 10px
      border-radius 50%
      background #d0d5dd
      border 2px solid #fff
      &.on
        background #12b76a
  &__main
    flex 1
    min-width 0
  &__line
    display flex
    align-items center
    justify-content space-between
    gap 8px
    strong
      font-size 14px
      overflow hidden
      text-overflow ellipsis
      white-space nowrap
    time
      font-size 11px
      color #98a2b3
      flex-shrink 0
    p
      margin 4px 0 0
      font-size 12px
      color #667085
      overflow hidden
      text-overflow ellipsis
      white-space nowrap
    em
      min-width 18px
      height 18px
      padding 0 5px
      border-radius 999px
      background #f04438
      color #fff
      font-size 11px
      line-height 18px
      font-style normal
      text-align center
      flex-shrink 0

.cs-desk-chat
  display flex
  flex-direction column
  height calc(100vh - 46px)
  &__hd
    display flex
    align-items center
    gap 8px
    padding 10px 12px
    background #fff
    border-bottom 1px solid #eef2f6
  &__back
    border 0
    background transparent
    padding 4px
    font-size 18px
    color #344054
  &__info
    flex 1
    min-width 0
  &__name
    font-size 15px
    font-weight 600
    display flex
    align-items center
    gap 6px
  &__meta
    margin-top 2px
    font-size 11px
    color #98a2b3
  &__body
    flex 1
    overflow-y auto
    padding 12px
    -webkit-overflow-scrolling touch
  &__ft
    background #fff
    border-top 1px solid #eef2f6
    padding 10px 12px calc(10px + env(safe-area-inset-bottom))
  &__tip
    margin-bottom 8px
    font-size 12px
    color #b54708
    background #fffaeb
    border-radius 8px
    padding 6px 8px
  &__editor
    display flex
    gap 8px
    align-items flex-end
    textarea
      flex 1
      border 1px solid #e4e7ec
      border-radius 10px
      padding 8px 10px
      font-size 14px
      resize none
      outline none
      line-height 1.4

.cs-pill
  display inline-flex
  align-items center
  height 18px
  padding 0 6px
  border-radius 999px
  font-size 10px
  font-weight 600
  &.ok
    background #e8faf0
    color #067647
  &.off
    background #f2f4f7
    color #667085

.cs-bubble
  max-width 78%
  margin-bottom 12px
  &__text
    display inline-block
    max-width 100%
    padding 8px 12px
    border-radius 10px
    font-size 14px
    line-height 1.5
    white-space pre-wrap
    word-break break-word
    &.is-image
      padding 4px
      background #fff !important
      border 1px solid #e8ecf1
  &__img
    display block
    max-width 200px
    max-height 260px
    border-radius 8px
  &__time
    margin-top 4px
    font-size 11px
    color #98a2b3
  &.is-user
    margin-right auto
    .cs-bubble__text:not(.is-image)
      background #fff
      border 1px solid #e8ecf1
  &.is-admin
    margin-left auto
    text-align right
    .cs-bubble__text:not(.is-image)
      background #2f6bff
      color #fff
      text-align left
  &.is-system
    max-width 100%
    margin-left auto
    margin-right auto
    text-align center
    .cs-bubble__text
      background transparent
      color #98a2b3
      font-size 12px
</style>
