<template>
  <div class="placement-page">
    <div class="page-header">
      <van-icon name="arrow-left" size="20" class="back-btn" @click="onHeaderBack" />
      <div class="header-title">{{ t('placement.title') }}</div>
      <van-icon name="replay" size="18" class="refresh-btn" :class="{ spinning: loading }" @click="loadTree" />
    </div>

    <div class="search-wrap">
      <van-search
        v-model="keyword"
        :placeholder="t('placement.searchPlaceholder')"
        shape="round"
        background="transparent"
        clearable
      />
    </div>

    <div class="page-body" v-loading="loading">
      <van-empty v-if="!loading && !rootNode" :description="t('placement.empty')" />

      <template v-else-if="keyword.trim()">
        <div v-if="!searchGroups.length" class="empty-wrap">
          <van-empty :description="t('placement.noMatch')" />
        </div>
        <div v-for="group in searchGroups" :key="group.depth" class="layer-block">
          <div class="layer-title">{{ t('placement.layer', { n: group.depth }) }}</div>
          <div
            v-for="item in group.items"
            :key="item.node.user_id"
            class="search-item"
            :class="item.zone"
            @click="focusNode(item.node.user_id)"
          >
            <span class="zone-dot" :class="item.zone"></span>
            <div class="search-main">
              <div class="search-name">{{ displayName(item.node) }}</div>
              <div class="search-meta">ID {{ item.node.user_id }} · {{ formatCompact(item.node.recharge_amount || 0) }}</div>
            </div>
            <van-icon name="arrow" class="go" />
          </div>
        </div>
      </template>

      <template v-else-if="focus">
        <div class="crumb-bar" v-if="focusPath.length > 1">
          <button
            v-for="(p, idx) in focusPath"
            :key="p.user_id"
            type="button"
            class="crumb-item"
            :class="{ current: idx === focusPath.length - 1 }"
            @click="focusNode(p.user_id)"
          >
            <template v-if="idx > 0"><span class="sep">›</span></template>
            {{ displayName(p) }}
          </button>
        </div>

        <div class="tree-stage">
          <!-- 上级节点 -->
          <div class="node root-node">
            <div class="node-avatar">{{ initial(focus) }}</div>
            <div class="node-info">
              <div class="node-name">
                {{ displayName(focus) }}
                <span v-if="focus.user_id === rootNode.user_id" class="me">{{ t('placement.self') }}</span>
              </div>
              <div class="node-sub">ID {{ focus.user_id }} · {{ t('placement.recharge') }} {{ formatCompact(focus.recharge_amount || 0) }}</div>
              <div class="node-stats">
                <em class="lg">{{ t('placement.largeZone') }} {{ formatCompact(largePerfOf(focus)) }}</em>
                <em class="sm">{{ t('placement.smallZone') }} {{ formatCompact(smallPerfOf(focus)) }}</em>
              </div>
            </div>
          </div>

          <template v-if="largeChild || smallChild">
            <div class="connector">
              <svg viewBox="0 0 200 36" preserveAspectRatio="none" aria-hidden="true">
                <path d="M100 0 V10 M100 10 H40 V28 M100 10 H160 V28" fill="none" stroke="#cbd5e1" stroke-width="2" />
                <path d="M40 28 V36" fill="none" stroke="#16a34a" stroke-width="2.5" />
                <path d="M160 28 V36" fill="none" stroke="#ea580c" stroke-width="2.5" />
              </svg>
            </div>

            <div class="branches">
              <!-- 大区 -->
              <div class="branch">
                <div class="branch-tag large">{{ t('placement.largeZone') }}</div>
                <div
                  v-if="largeChild"
                  class="node child large"
                  @click="focusNode(largeChild.user_id)"
                >
                  <div class="node-avatar tint-lg">{{ initial(largeChild) }}</div>
                  <div class="node-name">{{ displayName(largeChild) }}</div>
                  <div class="node-sub">ID {{ largeChild.user_id }}</div>
                  <div class="node-sub">{{ t('placement.recharge') }} {{ formatCompact(largeChild.recharge_amount || 0) }}</div>
                  <div class="node-stats mini">
                    <em class="lg">{{ t('placement.largeShort') }} {{ formatCompact(largePerfOf(largeChild)) }}</em>
                    <em class="sm">{{ t('placement.smallShort') }} {{ formatCompact(smallPerfOf(largeChild)) }}</em>
                  </div>
                  <van-icon v-if="hasKids(largeChild)" name="arrow" class="enter" />
                </div>
                <div v-else class="node empty">{{ t('placement.noMember') }}</div>

                <template v-if="largeChild && hasKids(largeChild)">
                  <div class="connector mini">
                    <svg viewBox="0 0 100 24" preserveAspectRatio="none" aria-hidden="true">
                      <path d="M50 0 V8 M50 8 H20 V18 M50 8 H80 V18" fill="none" stroke="#e2e8f0" stroke-width="1.8" />
                      <path d="M20 18 V24" fill="none" stroke="#16a34a" stroke-width="2" />
                      <path d="M80 18 V24" fill="none" stroke="#ea580c" stroke-width="2" />
                    </svg>
                  </div>
                  <div class="sub-branches">
                    <button
                      v-if="largeOf(largeChild)"
                      type="button"
                      class="leaf large"
                      @click="focusNode(largeOf(largeChild).user_id)"
                    >
                      <b>{{ displayName(largeOf(largeChild)) }}</b>
                      <span>ID {{ largeOf(largeChild).user_id }}</span>
                    </button>
                    <div v-else class="leaf ghost">—</div>
                    <button
                      v-if="smallOf(largeChild)"
                      type="button"
                      class="leaf small"
                      @click="focusNode(smallOf(largeChild).user_id)"
                    >
                      <b>{{ displayName(smallOf(largeChild)) }}</b>
                      <span>ID {{ smallOf(largeChild).user_id }}</span>
                    </button>
                    <div v-else class="leaf ghost">—</div>
                  </div>
                </template>
              </div>

              <!-- 小区 -->
              <div class="branch">
                <div class="branch-tag small">{{ t('placement.smallZone') }}</div>
                <div
                  v-if="smallChild"
                  class="node child small"
                  @click="focusNode(smallChild.user_id)"
                >
                  <div class="node-avatar tint-sm">{{ initial(smallChild) }}</div>
                  <div class="node-name">{{ displayName(smallChild) }}</div>
                  <div class="node-sub">ID {{ smallChild.user_id }}</div>
                  <div class="node-sub">{{ t('placement.recharge') }} {{ formatCompact(smallChild.recharge_amount || 0) }}</div>
                  <div class="node-stats mini">
                    <em class="lg">{{ t('placement.largeShort') }} {{ formatCompact(largePerfOf(smallChild)) }}</em>
                    <em class="sm">{{ t('placement.smallShort') }} {{ formatCompact(smallPerfOf(smallChild)) }}</em>
                  </div>
                  <van-icon v-if="hasKids(smallChild)" name="arrow" class="enter" />
                </div>
                <div v-else class="node empty">{{ t('placement.noMember') }}</div>

                <template v-if="smallChild && hasKids(smallChild)">
                  <div class="connector mini">
                    <svg viewBox="0 0 100 24" preserveAspectRatio="none" aria-hidden="true">
                      <path d="M50 0 V8 M50 8 H20 V18 M50 8 H80 V18" fill="none" stroke="#e2e8f0" stroke-width="1.8" />
                      <path d="M20 18 V24" fill="none" stroke="#16a34a" stroke-width="2" />
                      <path d="M80 18 V24" fill="none" stroke="#ea580c" stroke-width="2" />
                    </svg>
                  </div>
                  <div class="sub-branches">
                    <button
                      v-if="largeOf(smallChild)"
                      type="button"
                      class="leaf large"
                      @click="focusNode(largeOf(smallChild).user_id)"
                    >
                      <b>{{ displayName(largeOf(smallChild)) }}</b>
                      <span>ID {{ largeOf(smallChild).user_id }}</span>
                    </button>
                    <div v-else class="leaf ghost">—</div>
                    <button
                      v-if="smallOf(smallChild)"
                      type="button"
                      class="leaf small"
                      @click="focusNode(smallOf(smallChild).user_id)"
                    >
                      <b>{{ displayName(smallOf(smallChild)) }}</b>
                      <span>ID {{ smallOf(smallChild).user_id }}</span>
                    </button>
                    <div v-else class="leaf ghost">—</div>
                  </div>
                </template>
              </div>
            </div>

            <div class="foot-hint">{{ t('placement.tapHint') }}</div>
          </template>

          <div v-else class="empty-wrap">
            <van-empty :image-size="64" :description="t('placement.noDownline')" />
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { httpGet } from '@/utils/http'
import { showToast } from 'vant'

const { t } = useI18n()
const router = useRouter()

const loading = ref(false)
const nodes = ref([])
const keyword = ref('')
const focusId = ref(null)

const nodeMap = computed(() => {
  const map = {}
  nodes.value.forEach((n) => { map[n.user_id] = n })
  return map
})

const rootNode = computed(() => {
  if (!nodes.value.length) return null
  return nodes.value.find((n) => n.side === 'self') || nodes.value[0]
})

const focus = computed(() => {
  if (!focusId.value) return rootNode.value
  return nodeMap.value[focusId.value] || rootNode.value
})

const parentOf = computed(() => {
  const map = {}
  Object.values(nodeMap.value).forEach((n) => {
    if (n.left_child_id) map[n.left_child_id] = n.user_id
    if (n.right_child_id) map[n.right_child_id] = n.user_id
  })
  return map
})

const focusPath = computed(() => {
  if (!focus.value || !rootNode.value) return []
  const path = []
  let cur = focus.value.user_id
  const guard = new Set()
  while (cur && !guard.has(cur)) {
    guard.add(cur)
    const n = nodeMap.value[cur]
    if (!n) break
    path.unshift(n)
    if (cur === rootNode.value.user_id) break
    cur = parentOf.value[cur]
  }
  return path
})

const isLeftLarge = (node) => Number(node?.left_perf || 0) >= Number(node?.right_perf || 0)

const largePerfOf = (node) => {
  if (!node) return 0
  return isLeftLarge(node) ? Number(node.left_perf || 0) : Number(node.right_perf || 0)
}

const smallPerfOf = (node) => {
  if (!node) return 0
  return isLeftLarge(node) ? Number(node.right_perf || 0) : Number(node.left_perf || 0)
}

const childByZone = (node, wantLarge) => {
  if (!node) return null
  const leftLarge = isLeftLarge(node)
  const id = wantLarge
    ? (leftLarge ? node.left_child_id : node.right_child_id)
    : (leftLarge ? node.right_child_id : node.left_child_id)
  return id ? nodeMap.value[id] || null : null
}

const largeOf = (node) => childByZone(node, true)
const smallOf = (node) => childByZone(node, false)
const hasKids = (node) => !!(largeOf(node) || smallOf(node))

const largeChild = computed(() => largeOf(focus.value))
const smallChild = computed(() => smallOf(focus.value))

const zoneOfChild = (parent, childId) => {
  if (!parent || !childId) return 'large'
  const leftLarge = isLeftLarge(parent)
  if (childId === parent.left_child_id) return leftLarge ? 'large' : 'small'
  if (childId === parent.right_child_id) return leftLarge ? 'small' : 'large'
  return 'large'
}

const depthOf = (userId) => {
  let d = 0
  let cur = userId
  const guard = new Set()
  while (cur && rootNode.value && cur !== rootNode.value.user_id && !guard.has(cur)) {
    guard.add(cur)
    cur = parentOf.value[cur]
    d += 1
    if (d > 64) break
  }
  return d
}

const searchGroups = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw || !rootNode.value) return []
  const matched = nodes.value.filter((n) => {
    const id = String(n.user_id)
    const username = (n.user_info?.username || '').toLowerCase()
    const nickname = (n.user_info?.nickname || '').toLowerCase()
    return id.includes(kw) || username.includes(kw) || nickname.includes(kw)
  })
  const groupsMap = {}
  matched.forEach((n) => {
    const depth = depthOf(n.user_id)
    if (!groupsMap[depth]) groupsMap[depth] = []
    let zone = 'self'
    if (n.user_id !== rootNode.value.user_id) {
      const pid = parentOf.value[n.user_id]
      const parent = pid ? nodeMap.value[pid] : null
      zone = parent ? zoneOfChild(parent, n.user_id) : 'large'
    }
    groupsMap[depth].push({ node: n, zone })
  })
  return Object.keys(groupsMap)
    .map(Number)
    .sort((a, b) => a - b)
    .map((depth) => ({ depth, items: groupsMap[depth] }))
})

const displayName = (node) => node?.user_info?.username || node?.user_info?.nickname || t('placement.unnamed')
const initial = (node) => (displayName(node) || '?').slice(0, 1).toUpperCase()

const formatNumber = (num) => {
  if (!num && num !== 0) return '0'
  return Number(num).toLocaleString('zh-CN', { minimumFractionDigits: 0, maximumFractionDigits: 2 })
}

const formatCompact = (num) => {
  const n = Number(num || 0)
  if (n >= 10000) return (n / 10000).toFixed(n >= 100000 ? 0 : 1) + '万'
  return formatNumber(n)
}

const focusNode = (id) => {
  keyword.value = ''
  focusId.value = id
}

const onHeaderBack = () => {
  if (focus.value && rootNode.value && focus.value.user_id !== rootNode.value.user_id) {
    focusId.value = parentOf.value[focus.value.user_id] || rootNode.value.user_id
    return
  }
  router.back()
}

const loadTree = async () => {
  loading.value = true
  try {
    const profileRes = await httpGet('/api/user/profile', {}, { timeout: 30000 })
    if (!profileRes.data?.allow_placement) {
      showToast(t('placement.notAllowed'))
      router.replace('/mobile/invitation')
      return
    }
    const res = await httpGet('/api/user/placement/tree', {}, { timeout: 90000 })
    const data = res.data || {}
    nodes.value = data.nodes || []
    focusId.value = (nodes.value.find((n) => n.side === 'self') || nodes.value[0])?.user_id || null
  } catch (e) {
    nodes.value = []
    focusId.value = null
    const msg = e?.message || e?.msg || t('placement.loadFail')
    showToast(msg)
    if (String(msg).includes('未开通') || String(msg).includes('权限')) {
      router.replace('/mobile/invitation')
    }
  } finally {
    loading.value = false
  }
}

watch(rootNode, (n) => {
  if (n && !focusId.value) focusId.value = n.user_id
})

onMounted(loadTree)
</script>

<style scoped lang="stylus">
.placement-page
  min-height 100vh
  background #f4f6f9
  padding-bottom 28px

.page-header
  display flex
  align-items center
  justify-content space-between
  padding 12px 16px
  background #fff
  position sticky
  top 0
  z-index 10

.header-title
  font-size 17px
  font-weight 600
  color #111827

.back-btn, .refresh-btn
  color #2563eb
  padding 4px

.refresh-btn.spinning
  animation spin .8s linear infinite

@keyframes spin
  to
    transform rotate(360deg)

.search-wrap
  background #fff
  padding 0 4px 4px
  :deep(.van-search)
    padding 4px 12px 8px
  :deep(.van-search__content)
    background #f3f4f6

.page-body
  padding 10px 12px 16px

.crumb-bar
  display flex
  flex-wrap wrap
  align-items center
  gap 2px
  margin-bottom 10px
  padding 0 2px

.crumb-item
  border none
  background transparent
  color #6b7280
  font-size 12px
  padding 0
  max-width 96px
  overflow hidden
  text-overflow ellipsis
  white-space nowrap
  &.current
    color #111827
    font-weight 600
  .sep
    margin 0 4px
    color #d1d5db

.tree-stage
  background #fff
  border-radius 16px
  padding 16px 12px 14px
  box-shadow 0 1px 3px rgba(0,0,0,.04)

.root-node
  display flex
  align-items center
  gap 12px
  margin 0 auto
  max-width 320px
  padding 12px 14px
  border-radius 14px
  background #f8fafc
  border 1px solid #e5e7eb

.node-avatar
  flex-shrink 0
  width 40px
  height 40px
  border-radius 12px
  background #2563eb
  color #fff
  display flex
  align-items center
  justify-content center
  font-weight 700
  font-size 15px
  &.tint-lg
    background #16a34a
  &.tint-sm
    background #ea580c

.node-info
  flex 1
  min-width 0

.node-name
  font-size 15px
  font-weight 700
  color #111827
  display flex
  align-items center
  gap 6px

.me
  font-size 10px
  font-weight 600
  padding 1px 6px
  border-radius 999px
  background #e0e7ff
  color #4338ca

.node-sub
  margin-top 2px
  font-size 11px
  color #9ca3af

.node-stats
  margin-top 8px
  display flex
  flex-wrap wrap
  gap 6px
  em
    font-style normal
    font-size 11px
    font-weight 600
    padding 2px 8px
    border-radius 999px
    &.lg
      background #dcfce7
      color #166534
    &.sm
      background #ffedd5
      color #9a3412
  &.mini
    justify-content center
    em
      font-size 10px
      padding 1px 6px

.connector
  height 36px
  margin 2px 0
  svg
    width 100%
    height 100%
    display block
  &.mini
    height 24px
    margin 0

.branches
  display grid
  grid-template-columns 1fr 1fr
  gap 10px

.branch-tag
  text-align center
  font-size 11px
  font-weight 700
  margin-bottom 6px
  &.large
    color #16a34a
  &.small
    color #ea580c

.node.child
  position relative
  text-align center
  padding 12px 8px 10px
  border-radius 12px
  background #fff
  border 1px solid #e5e7eb
  .node-avatar
    margin 0 auto 6px
  .node-name
    justify-content center
    font-size 13px
  &.large
    border-color #bbf7d0
    background #f0fdf4
  &.small
    border-color #fed7aa
    background #fff7ed
  &:active
    transform scale(.98)
  .enter
    position absolute
    right 8px
    top 8px
    color #9ca3af
    font-size 12px

.node.empty
  text-align center
  padding 28px 8px
  border-radius 12px
  border 1px dashed #e5e7eb
  color #9ca3af
  font-size 12px
  background #fafafa

.sub-branches
  display grid
  grid-template-columns 1fr 1fr
  gap 4px

.leaf
  border 1px solid #e5e7eb
  background #fff
  border-radius 8px
  padding 8px 4px
  text-align center
  b
    display block
    font-size 11px
    color #111827
    overflow hidden
    text-overflow ellipsis
    white-space nowrap
  span
    display block
    margin-top 2px
    font-size 9px
    color #9ca3af
  &.large
    border-color #bbf7d0
  &.small
    border-color #fed7aa
  &.ghost
    color #d1d5db
    border-style dashed
    background #fafafa
    font-size 12px
    padding 14px 4px
  &:active
    opacity .85

.foot-hint
  margin-top 12px
  text-align center
  font-size 11px
  color #9ca3af

.layer-block
  margin-bottom 12px

.layer-title
  font-size 12px
  font-weight 600
  color #6b7280
  margin 0 2px 8px

.search-item
  display flex
  align-items center
  gap 10px
  padding 12px
  margin-bottom 8px
  background #fff
  border-radius 12px
  border-left 3px solid #e5e7eb
  &.large
    border-left-color #16a34a
  &.small
    border-left-color #ea580c
  &.self
    border-left-color #6366f1
  .go
    color #d1d5db

.zone-dot
  width 8px
  height 8px
  border-radius 50%
  background #9ca3af
  &.large
    background #16a34a
  &.small
    background #ea580c
  &.self
    background #6366f1

.search-main
  flex 1
  min-width 0

.search-name
  font-size 14px
  font-weight 600
  color #111827

.search-meta
  margin-top 2px
  font-size 11px
  color #9ca3af

.empty-wrap
  padding 20px 0
</style>
