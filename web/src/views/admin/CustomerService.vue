<template>
  <div class="cs-page">
    <div class="cs-desk">
      <aside class="cs-side">
        <div class="cs-side__hd">
          <div class="cs-side__title">
            <span>会话</span>
            <span class="cs-side__live" :class="{ on: wsConnected }">
              {{ wsConnected ? '已接入' : '连接中' }}
            </span>
          </div>
          <div class="cs-side__tabs">
            <button
              v-for="tab in tabs"
              :key="tab.key"
              type="button"
              :class="{ active: filter === tab.key }"
              @click="filter = tab.key"
            >
              {{ tab.label }}
              <b v-if="tab.key === 'unread' && unreadCount">{{ unreadCount }}</b>
            </button>
          </div>
          <el-input
            v-model="keyword"
            clearable
            size="small"
            placeholder="搜索 UID / 用户名"
          />
        </div>

        <div class="cs-side__list">
          <button
            v-for="item in filteredSessions"
            :key="item.id"
            type="button"
            class="cs-sess"
            :class="{ active: currentId === item.id }"
            @click="selectSession(item)"
          >
            <div class="cs-sess__av">
              {{ avatarText(item) }}
              <i :class="{ on: item.user_online }" />
            </div>
            <div class="cs-sess__main">
              <div class="cs-sess__line">
                <strong>{{ displayName(item) }}</strong>
                <time>{{ formatTime(item.last_msg_at) }}</time>
              </div>
              <div class="cs-sess__line">
                <p>{{ item.last_msg || '暂无消息' }}</p>
                <em v-if="item.admin_unread > 0">{{ item.admin_unread }}</em>
              </div>
            </div>
          </button>
          <div v-if="!filteredSessions.length" class="cs-side__empty">暂无会话</div>
        </div>
      </aside>

      <section class="cs-chat">
        <div v-if="!current" class="cs-chat__idle">
          <h3>客服工作台</h3>
          <p>从左侧选择会话开始回复。用户离线时也可留言。</p>
        </div>
        <template v-else>
          <header class="cs-chat__hd">
            <div>
              <div class="cs-chat__name">
                {{ displayName(current) }}
                <span class="cs-pill" :class="current.user_online ? 'ok' : 'off'">
                  {{ current.user_online ? '在线' : '离线' }}
                </span>
              </div>
              <div class="cs-chat__meta">
                UID {{ current.user_id }}
                <template v-if="current.nickname && current.nickname !== current.username">
                  · 昵称 {{ current.nickname }}
                </template>
              </div>
            </div>
            <div class="cs-chat__ops">
              <el-button size="small" text @click="loadSessions">刷新</el-button>
              <el-button size="small" type="danger" plain @click="closeSession">结束</el-button>
            </div>
          </header>

          <div ref="msgBox" class="cs-chat__body">
            <div
              v-for="msg in messages"
              :key="msg.id || msg._key"
              class="cs-bubble"
              :class="'is-' + msg.role"
            >
              <div class="cs-bubble__text" :class="{ 'is-image': msg.msg_type === 'image' }">
                <el-image
                  v-if="msg.msg_type === 'image'"
                  :src="msg.content"
                  fit="cover"
                  class="cs-bubble__img"
                  :preview-src-list="[msg.content]"
                  preview-teleported
                />
                <template v-else>{{ msg.content }}</template>
              </div>
              <div class="cs-bubble__time">{{ formatTime(msg.created_at) }}</div>
            </div>
          </div>

          <footer class="cs-chat__ft">
            <div v-if="!current.user_online" class="cs-chat__tip">
              用户离线：回复将作为留言，对方上线后可见
            </div>
            <div class="cs-chat__editor">
              <textarea
                v-model="draft"
                rows="3"
                maxlength="2000"
                placeholder="输入回复，Enter 发送，Shift+Enter 换行"
                @keydown.enter.exact.prevent="send"
              />
              <el-button
                type="primary"
                :disabled="!draft.trim() || sending"
                :loading="sending"
                @click="send"
              >
                发送
              </el-button>
            </div>
          </footer>
        </template>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { httpGet, httpPost, resolveWsHost } from '@/utils/http'
import { getAdminToken } from '@/store/session'
import { playCsNotifySound } from '@/utils/csNotify'

defineOptions({ name: 'CustomerService' })

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
  try {
    const res = await httpPost('/api/admin/cs/sessions', { status: 0, page: 1, page_size: 100 })
    sessions.value = Array.isArray(res.data?.items) ? res.data.items : []
    sortSessions(sessions.value)
  } catch (e) {
    ElMessage.error('加载会话失败: ' + (e.message || e))
  }
}

const selectSession = async (item) => {
  current.value = item
  currentId.value = item.id
  try {
    const res = await httpGet(`/api/admin/cs/messages?session_id=${item.id}`)
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
    ElMessage.error('加载消息失败: ' + (e.message || e))
  }
}

const closeSession = async () => {
  if (!currentId.value) return
  try {
    await ElMessageBox.confirm('确定结束该会话？', '提示', { type: 'warning' })
    await httpGet(`/api/admin/cs/close?session_id=${currentId.value}`)
    sessions.value = sessions.value.filter((s) => s.id !== currentId.value)
    current.value = null
    currentId.value = 0
    messages.value = []
    ElMessage.success('已结束会话')
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('操作失败')
  }
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
  if (pkt.type === 'error' && pkt.error) ElMessage.error(pkt.error)
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
  const token = getAdminToken()
  if (!token) {
    ElMessage.error('请先登录管理后台')
    return
  }
  if (socket) {
    try { socket.close() } catch { /* ignore */ }
    socket = null
  }
  const host = resolveWsHost()
  const ws = new WebSocket(`${host}/api/admin/cs/ws?token=${encodeURIComponent(token)}`)
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
      const res = await httpPost('/api/admin/cs/reply', payload)
      if (res.data?.message) pushMsg(res.data.message)
      if (res.data?.session) upsertSession(res.data.session)
    }
  } catch (e) {
    draft.value = content
    ElMessage.error('发送失败: ' + (e.message || e))
  } finally {
    sending.value = false
  }
}

onMounted(async () => {
  await loadSessions()
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

<style scoped lang="stylus">
/* 占满内容区，避免外层再滚动 */
.cs-page
  height calc(100vh - 130px)
  min-height 520px
  box-sizing border-box

.cs-desk
  height 100%
  display flex
  background #fff
  border 1px solid #e5eaf0
  border-radius 8px
  overflow hidden

.cs-side
  width 300px
  flex-shrink 0
  display flex
  flex-direction column
  border-right 1px solid #e5eaf0
  background #fafbfd

  &__hd
    padding 12px
    border-bottom 1px solid #eef1f5
    background #fff
    flex-shrink 0

  &__title
    display flex
    align-items center
    justify-content space-between
    margin-bottom 10px
    font-size 15px
    font-weight 700
    color #1f2d3d

  &__live
    font-size 12px
    font-weight 500
    color #98a2b3
    &.on
      color #067647

  &__tabs
    display flex
    gap 6px
    margin-bottom 10px
    button
      border 0
      background #eef2f7
      color #5b6b7c
      font-size 12px
      padding 4px 10px
      border-radius 999px
      cursor pointer
      &.active
        background #2f6bff
        color #fff
      b
        margin-left 4px
        font-weight 600

  &__list
    flex 1
    min-height 0
    overflow-y auto

  &__empty
    padding 40px 12px
    text-align center
    color #98a2b3
    font-size 13px

.cs-sess
  width 100%
  display flex
  gap 10px
  padding 12px
  border 0
  border-bottom 1px solid #f0f2f5
  background transparent
  text-align left
  cursor pointer
  &:hover, &.active
    background #eef4ff

  &__av
    position relative
    width 38px
    height 38px
    border-radius 10px
    background linear-gradient(135deg, #5b8cff, #3f7dff)
    color #fff
    font-weight 700
    display flex
    align-items center
    justify-content center
    flex-shrink 0
    i
      position absolute
      right -1px
      bottom -1px
      width 10px
      height 10px
      border-radius 50%
      background #c0c8d2
      border 2px solid #fff
      &.on
        background #17c964

  &__main
    flex 1
    min-width 0

  &__line
    display flex
    align-items center
    justify-content space-between
    gap 8px
    strong
      font-size 13px
      color #1f2d3d
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
      flex-shrink 0
      min-width 18px
      height 18px
      margin-top 4px
      padding 0 5px
      border-radius 9px
      background #f04438
      color #fff
      font-style normal
      font-size 11px
      line-height 18px
      text-align center

.cs-chat
  flex 1
  min-width 0
  min-height 0
  display flex
  flex-direction column
  background #fff

  &__idle
    margin auto
    text-align center
    color #667085
    h3
      margin 0 0 8px
      color #1f2d3d
    p
      margin 0
      font-size 13px

  &__hd
    flex-shrink 0
    display flex
    align-items center
    justify-content space-between
    gap 12px
    padding 12px 16px
    border-bottom 1px solid #eef1f5

  &__name
    display flex
    align-items center
    gap 8px
    font-size 15px
    font-weight 700
    color #1f2d3d

  &__meta
    margin-top 4px
    font-size 12px
    color #98a2b3

  &__ops
    display flex
    gap 6px

  &__body
    flex 1
    min-height 0
    overflow-x hidden
    overflow-y auto
    padding 16px
    background #f4f6fa

  &__ft
    flex-shrink 0
    border-top 1px solid #eef1f5
    padding 10px 14px 12px
    background #fff

  &__tip
    margin-bottom 8px
    padding 6px 10px
    border-radius 6px
    background #fff7e8
    color #b54708
    font-size 12px

  &__editor
    display flex
    gap 10px
    align-items flex-end
    textarea
      flex 1
      min-height 72px
      max-height 140px
      resize vertical
      padding 10px 12px
      border 1px solid #e5eaf0
      border-radius 8px
      background #f7f9fc
      font-size 14px
      line-height 1.5
      color #1f2d3d
      outline none
      font-family inherit
      &:focus
        border-color #2f6bff
        background #fff

.cs-pill
  display inline-flex
  align-items center
  height 20px
  padding 0 8px
  border-radius 999px
  font-size 11px
  font-weight 600
  &.ok
    background #e8faf0
    color #067647
  &.off
    background #f2f4f7
    color #667085

.cs-bubble
  max-width 62%
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
    box-sizing border-box
    &.is-image
      padding 4px
      background #fff !important
      border 1px solid #e8ecf1
      color inherit
  &__img
    display block
    max-width 220px
    max-height 280px
    border-radius 8px
    cursor pointer
  &__time
    margin-top 4px
    font-size 11px
    color #98a2b3
  &.is-user
    margin-right auto
    .cs-bubble__text:not(.is-image)
      background #fff
      border 1px solid #e8ecf1
      color #1f2d3d
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
      border 0
      color #98a2b3
      font-size 12px
      padding 0
</style>
