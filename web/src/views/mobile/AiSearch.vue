<template>
  <div class="ai-search" :class="{ 'ai-search--embedded': embedded }">
    <div class="ai-search__bg" aria-hidden="true">
      <div class="ai-search__deco"></div>
    </div>

    <header v-show="!detailOpen" class="ai-search__hero van-safe-area-top">
      <div class="ai-search__brand">
        <button
          v-if="embedded"
          type="button"
          class="ai-search__close"
          aria-label="close"
          @click="closeToHub"
        >
          <van-icon name="cross" size="18" />
        </button>
        <div class="ai-search__logo" aria-hidden="true">
          <van-icon name="search" size="20" color="#fff" />
        </div>
        <div class="ai-search__brand-text">
          <h1>{{ t('imageApps.aiSearch') }}</h1>
          <p>{{ t('aiSearch.subtitle') }}</p>
        </div>
      </div>

      <form class="ai-search__bar" @submit.prevent="onSearch">
        <van-icon name="search" size="18" color="#8aa0b8" class="ai-search__bar-ico" />
        <input
          v-model="query"
          type="search"
          class="ai-search__input"
          :placeholder="t('aiSearch.placeholder')"
          :disabled="loading"
          enterkeyhint="search"
          autocomplete="off"
          @keydown.enter.prevent="onSearch"
        />
        <button type="submit" class="ai-search__go" :disabled="loading || !query.trim()">
          {{ loading ? t('aiSearch.searching') : t('aiSearch.search') }}
        </button>
      </form>
    </header>

    <div v-show="!detailOpen" class="ai-search__scroll">
      <section v-if="history.length" class="ai-search__block">
        <div class="ai-search__sec-head">
          <h3><i class="ai-search__bar-mark" />{{ t('aiSearch.history') }}</h3>
          <button type="button" class="ai-search__icon-btn" :aria-label="t('aiSearch.clearHistory')" @click="clearHistory">
            <van-icon name="delete-o" size="16" />
          </button>
        </div>
        <div class="ai-search__tags">
          <button
            v-for="row in visibleHistory"
            :key="row.keyword"
            type="button"
            class="ai-search__tag"
            :disabled="loading"
            @click="searchKeyword(row.keyword)"
          >
            {{ row.keyword }}
          </button>
          <button
            v-if="historyHasMore"
            type="button"
            class="ai-search__tag ai-search__tag--more"
            @click="historyExpanded = !historyExpanded"
          >
            {{ historyExpanded ? t('aiSearch.collapse') : t('aiSearch.more') }}
          </button>
        </div>
      </section>

      <p v-if="error && !detailOpen" class="ai-search__alert">{{ error }}</p>

      <section v-if="isLanding" class="ai-search__landing">
        <h2>{{ t('aiSearch.landingTitle') }}</h2>
        <p>{{ t('aiSearch.landingDesc') }}</p>
        <button type="button" class="ai-search__notice" @click="showNotice = true">
          {{ t('aiSearch.noticeBtn') }}
        </button>
      </section>

      <section v-if="hasResults" class="ai-search__block ai-search__results">
        <div class="ai-search__sec-head">
          <h3><i class="ai-search__bar-mark" />{{ t('aiSearch.resultsTitle') }}</h3>
        </div>

        <div class="ai-search__filters" role="tablist">
          <button
            v-for="f in filterOptions"
            :key="f.key"
            type="button"
            class="ai-search__filter"
            :class="{ 'is-active': activeFilter === f.key }"
            :disabled="loading"
            @click="setFilter(f.key)"
          >
            {{ f.label }}
          </button>
        </div>

        <p v-if="loading" class="ai-search__muted">{{ t('aiSearch.searchingNet') }}</p>
        <p v-else-if="!filteredTotal" class="ai-search__muted">{{ t('aiSearch.emptyFilter') }}</p>

        <ul v-else class="ai-search__list">
          <li v-for="(item, idx) in pageResults" :key="(item.url || '') + '-' + idx">
            <article class="ai-search__card">
              <div class="ai-search__card-meta">
                <div class="ai-search__card-site">
                  <img
                    class="ai-search__favicon"
                    :src="faviconOf(item)"
                    alt=""
                    loading="lazy"
                    @error="onFaviconError"
                  />
                  <span>{{ item.site_name || hostOf(item.url) }}</span>
                </div>
                <span v-if="item.date" class="ai-search__date">{{ formatDate(item.date) }}</span>
              </div>

              <h3 class="ai-search__card-title" @click="openResult(item)">{{ item.title }}</h3>
              <p class="ai-search__card-snippet">{{ item.snippet || item.summary }}</p>

              <div class="ai-search__card-foot">
                <a
                  class="ai-search__open"
                  :href="item.url"
                  target="_blank"
                  rel="noopener noreferrer"
                  @click.stop
                >{{ t('aiSearch.openLink') }}</a>
                <div class="ai-search__source">
                  <img
                    class="ai-search__favicon ai-search__favicon--sm"
                    :src="faviconOf(item)"
                    alt=""
                    loading="lazy"
                    @error="onFaviconError"
                  />
                  <span>{{ item.site_name || hostOf(item.url) }}</span>
                </div>
              </div>
            </article>
          </li>
        </ul>

        <div v-if="filteredTotal > PAGE_SIZE" class="ai-search__pager">
          <button type="button" class="ai-search__page-btn" :disabled="page <= 1 || loading" @click="goPage(page - 1)">
            {{ t('aiSearch.prevPage') }}
          </button>
          <span class="ai-search__page-info">{{ page }} / {{ totalPages }}</span>
          <button
            type="button"
            class="ai-search__page-btn"
            :disabled="page >= totalPages || loading"
            @click="goPage(page + 1)"
          >
            {{ t('aiSearch.nextPage') }}
          </button>
        </div>
      </section>
    </div>

    <!-- 页内详情：不盖住底部 Tabbar -->
    <section v-if="detailOpen" class="ai-search__detail">
      <div class="ai-search__detail-top van-safe-area-top">
        <button type="button" class="ai-search__back" @click="closeDetail">
          <van-icon name="arrow-left" size="18" />
          <span>{{ t('aiSearch.back') }}</span>
        </button>
        <span class="ai-search__detail-tip">{{ t('aiSearch.detailTip') }}</span>
      </div>

      <div class="ai-search__detail-scroll">
        <article class="ai-search__detail-card">
          <div class="ai-search__card-meta">
            <div class="ai-search__card-site">
              <img
                class="ai-search__favicon"
                :src="faviconOf(detailItem)"
                alt=""
                @error="onFaviconError"
              />
              <span>{{ detailItem?.site_name || hostOf(detailUrl) }}</span>
            </div>
            <span v-if="detailItem?.date" class="ai-search__date">{{ formatDate(detailItem.date) }}</span>
          </div>

          <h2 class="ai-search__detail-title">{{ detailTitle }}</h2>

          <div v-if="detailLoading" class="ai-search__loading">
            <van-loading type="spinner" color="#2f6bff" />
            <p>{{ t('aiSearch.loadingResource') }}</p>
          </div>

          <template v-else>
            <p v-if="detailError" class="ai-search__alert">{{ detailError }}</p>
            <div class="ai-search__detail-body" v-html="detailHtml"></div>
          </template>
        </article>
      </div>

      <div class="ai-search__detail-actions">
        <a
          v-if="detailUrl"
          class="ai-search__detail-open"
          :href="detailUrl"
          target="_blank"
          rel="noopener noreferrer"
        >{{ t('aiSearch.openLink') }}</a>
        <button type="button" class="ai-search__detail-close" @click="closeDetail">
          {{ t('aiSearch.backList') }}
        </button>
      </div>
    </section>

    <van-dialog v-model:show="showNotice" :title="t('aiSearch.noticeTitle')" confirm-button-text="OK">
      <p class="ai-search__dialog-body">{{ t('aiSearch.noticeBody') }}</p>
    </van-dialog>
  </div>
</template>

<script setup>
import { computed, inject, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { showToast } from 'vant'
import { IMAGE_RETURN_HUB_KEY } from '@/constants/imageAppLaunch'
import {
  aiSearchFetch,
  clearLocalHistory,
  loadLocalHistory,
  saveLocalHistory,
} from '@/utils/aiSearchApi'

defineOptions({ name: 'AiSearch' })

const HISTORY_COLLAPSED = 5
const PAGE_SIZE = 10
const FALLBACK_FAVICON =
  'data:image/svg+xml,' +
  encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32"><rect width="32" height="32" rx="8" fill="#dbe7ff"/><circle cx="14" cy="14" r="6" fill="none" stroke="#2f6bff" stroke-width="2"/><path d="M18.5 18.5 L24 24" stroke="#2f6bff" stroke-width="2" stroke-linecap="round"/></svg>'
  )

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const returnToHubNative = inject(IMAGE_RETURN_HUB_KEY, null)
const embedded = computed(() => !!route.meta?.hideTabbar)

async function closeToHub() {
  if (typeof returnToHubNative === 'function') {
    await returnToHubNative()
    return
  }
  router.push({ name: 'mobile-image-hub' })
}

const query = ref('')
const searchQuery = ref('')
const loading = ref(false)
const error = ref(null)
const allResults = ref([])
const activeFilter = ref('all')
const page = ref(1)
const history = ref(loadLocalHistory())
const historyExpanded = ref(false)
const showNotice = ref(false)

const detailOpen = ref(false)
const detailLoading = ref(false)
const detailError = ref(null)
const detailTitle = ref('')
const detailContent = ref('')
const detailUrl = ref('')
const detailItem = ref(null)

let searchAbort = null
let fetchAbort = null

const detailHtml = computed(() => formatDetailHtml(detailContent.value))

const filterOptions = computed(() => [
  { key: 'all', label: t('aiSearch.filterAll') },
  { key: 'quality', label: t('aiSearch.filterQuality') },
  { key: 'docs', label: t('aiSearch.filterDocs') },
  { key: 'apps', label: t('aiSearch.filterApps') },
])

const isLanding = computed(
  () => !loading.value && !allResults.value.length && !searchQuery.value && !error.value
)
const hasResults = computed(() => allResults.value.length > 0 || Boolean(searchQuery.value))
const historyHasMore = computed(() => history.value.length > HISTORY_COLLAPSED)
const visibleHistory = computed(() =>
  historyExpanded.value ? history.value : history.value.slice(0, HISTORY_COLLAPSED)
)

function textBlob(item) {
  return `${item?.title || ''} ${item?.site_name || ''} ${item?.url || ''} ${item?.snippet || ''}`.toLowerCase()
}

function isDocHit(item) {
  const s = textBlob(item)
  return /百科|wiki|wikipedia|pdf|\.doc|文档|知乎|baike|docs\.|文档|白皮书|report|论文/.test(s)
}

function isAppHit(item) {
  const s = textBlob(item)
  return /应用|app\b|apk|软件|商店|appstore|play\.google|microsoft\.com\/store|下载/.test(s)
}

const filteredResults = computed(() => {
  const list = allResults.value || []
  switch (activeFilter.value) {
    case 'quality':
      return list.filter((x) => Number(x.rerank_score) >= 1.5 || Number(x.authority) >= 3)
    case 'docs':
      return list.filter((x) => isDocHit(x))
    case 'apps':
      return list.filter((x) => isAppHit(x))
    default:
      return list
  }
})

const filteredTotal = computed(() => filteredResults.value.length)
const totalPages = computed(() => Math.max(1, Math.ceil(filteredTotal.value / PAGE_SIZE) || 1))
const pageResults = computed(() => {
  const start = (page.value - 1) * PAGE_SIZE
  return filteredResults.value.slice(start, start + PAGE_SIZE)
})

watch(activeFilter, () => {
  page.value = 1
})

function hostOf(url) {
  try {
    return new URL(url).hostname.replace(/^www\./, '')
  } catch {
    return ''
  }
}

function faviconOf(item) {
  if (item?.site_icon) return item.site_icon
  const host = hostOf(item?.url)
  if (!host) return FALLBACK_FAVICON
  return `https://www.google.com/s2/favicons?domain=${encodeURIComponent(host)}&sz=64`
}

function onFaviconError(e) {
  if (e?.target) e.target.src = FALLBACK_FAVICON
}

function formatDate(iso) {
  if (!iso) return ''
  try {
    const d = new Date(iso)
    if (Number.isNaN(d.getTime())) return iso
    const y = d.getFullYear()
    const m = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    return `${y}/${m}/${day}`
  } catch {
    return iso
  }
}

function clearHistory() {
  history.value = clearLocalHistory()
}

function searchKeyword(kw) {
  query.value = kw
  onSearch()
}

function setFilter(key) {
  if (activeFilter.value === key) return
  activeFilter.value = key
}

function goPage(p) {
  const next = Math.min(Math.max(1, p), totalPages.value)
  if (next === page.value) return
  page.value = next
  document.querySelector('.ai-search__results')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function scrubVendor(msg) {
  return String(msg || '')
    .replace(/DeepSeek\s*/gi, '')
    .replace(/\s{2,}/g, ' ')
    .trim()
}

async function onSearch() {
  const q = query.value.trim()
  if (!q || loading.value) return
  if (searchAbort) searchAbort.abort()
  searchAbort = new AbortController()
  loading.value = true
  error.value = null
  searchQuery.value = q
  allResults.value = []
  page.value = 1
  activeFilter.value = 'all'
  history.value = saveLocalHistory(undefined, q)

  try {
    const { data } = await aiSearchFetch('/search', {
      method: 'POST',
      body: { query: q, count: 50 },
      signal: searchAbort.signal,
      timeoutMs: 45000,
    })
    if (data?.code !== 0) {
      throw new Error(scrubVendor(data?.message) || t('aiSearch.searchFail'))
    }
    const list = data?.data?.results || []
    allResults.value = Array.isArray(list) ? list : []
    if (!allResults.value.length) error.value = t('aiSearch.noResult')
  } catch (err) {
    if (err?.name === 'AbortError') return
    error.value = scrubVendor(err?.message) || t('aiSearch.searchFail')
    showToast(error.value)
  } finally {
    loading.value = false
  }
}

function escapeHtml(s) {
  return String(s || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function formatDetailHtml(raw) {
  const text = String(raw || '').trim()
  if (!text) return `<p class="ai-search__detail-empty">${escapeHtml(t('aiSearch.empty'))}</p>`
  const blocks = text.split(/\n{2,}/)
  return blocks
    .map((block) => {
      const lines = block.split('\n').map((l) => escapeHtml(l.trim())).filter(Boolean)
      if (!lines.length) return ''
      if (lines.length === 1 && lines[0].startsWith('# ')) {
        return `<h3>${lines[0].slice(2)}</h3>`
      }
      return `<p>${lines.join('<br>')}</p>`
    })
    .join('')
}

async function openResult(item) {
  if (!item?.url) return
  detailOpen.value = true
  detailItem.value = item
  detailTitle.value = item.title || hostOf(item.url)
  detailUrl.value = item.url
  detailContent.value = item.summary || item.snippet || ''
  detailError.value = null
  detailLoading.value = true
  if (fetchAbort) fetchAbort.abort()
  fetchAbort = new AbortController()
  try {
    const { data } = await aiSearchFetch('/fetch', {
      method: 'POST',
      body: { url: item.url },
      signal: fetchAbort.signal,
      timeoutMs: 60000,
    })
    if (data?.code !== 0) {
      if (!detailContent.value) throw new Error(scrubVendor(data?.message) || t('aiSearch.loadFail'))
      detailError.value = scrubVendor(data?.message) || null
      return
    }
    detailTitle.value = data?.data?.title || item.title
    detailContent.value = data?.data?.content || item.summary || item.snippet || ''
    detailUrl.value = data?.data?.url || item.url
  } catch (err) {
    if (err?.name === 'AbortError') return
    if (!detailContent.value) detailContent.value = item.summary || item.snippet || ''
    detailError.value = scrubVendor(err?.message) || t('aiSearch.loadFail')
  } finally {
    detailLoading.value = false
  }
}

function closeDetail() {
  if (fetchAbort) fetchAbort.abort()
  detailOpen.value = false
  detailContent.value = ''
  detailError.value = null
  detailItem.value = null
}

onBeforeUnmount(() => {
  if (searchAbort) searchAbort.abort()
  if (fetchAbort) fetchAbort.abort()
})
</script>

<style scoped>
.ai-search {
  --ai-blue: #2f6bff;
  --ai-blue-deep: #1f57e8;
  --ai-text: #1f2a44;
  --ai-muted: #8b9bb2;
  --ai-card: #ffffff;
  --ai-tabbar-h: 50px;
  position: relative;
  z-index: 1; /* 低于底栏，避免结果列表挡住 Tab 点击 */
  /* 预留底栏高度，页面本身不要盖住「对话/应用/市场/我的」 */
  height: calc(100vh - var(--ai-tabbar-h));
  height: calc(100dvh - var(--ai-tabbar-h) - env(safe-area-inset-bottom, 0px));
  max-height: calc(100dvh - var(--ai-tabbar-h) - env(safe-area-inset-bottom, 0px));
  display: flex;
  flex-direction: column;
  color: var(--ai-text);
  background: linear-gradient(180deg, #eaf2ff 0%, #f5f8fc 42%, #f7f9fc 100%);
  overflow: hidden;
}
.ai-search--embedded {
  --ai-tabbar-h: 0px;
  height: 100%;
  max-height: none;
  min-height: 100dvh;
}
.ai-search__close {
  flex-shrink: 0;
  width: 34px;
  height: 34px;
  margin: 0;
  border: none;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.85);
  color: #1f2a44;
  box-shadow: 0 2px 8px rgba(31, 42, 68, 0.12);
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
}
.ai-search__close:active {
  opacity: 0.85;
}

.ai-search__bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
  overflow: hidden;
}
.ai-search__deco {
  position: absolute;
  top: -20px;
  right: -30px;
  width: 220px;
  height: 180px;
  background:
    radial-gradient(circle at 35% 40%, rgba(91, 140, 255, 0.35), transparent 55%),
    radial-gradient(circle at 70% 55%, rgba(160, 196, 255, 0.45), transparent 60%);
  filter: blur(2px);
}
.ai-search__deco::before,
.ai-search__deco::after {
  content: '';
  position: absolute;
  border-radius: 18px;
  background: linear-gradient(145deg, rgba(255, 255, 255, 0.75), rgba(210, 226, 255, 0.55));
  box-shadow: 0 12px 30px rgba(47, 107, 255, 0.12);
}
.ai-search__deco::before {
  width: 78px;
  height: 96px;
  top: 42px;
  right: 78px;
  transform: rotate(12deg);
}
.ai-search__deco::after {
  width: 54px;
  height: 54px;
  top: 70px;
  right: 48px;
  border-radius: 50%;
  border: 8px solid rgba(255, 255, 255, 0.7);
  background: transparent;
  box-shadow: 14px 14px 0 -6px rgba(255, 255, 255, 0.55);
}

.ai-search__hero {
  position: relative;
  z-index: 1;
  flex: 0 0 auto;
  padding: 14px 16px 8px;
}

.ai-search__brand {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}
.ai-search__logo {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  background: linear-gradient(160deg, #5b8cff, var(--ai-blue-deep));
  box-shadow: 0 8px 18px rgba(47, 107, 255, 0.28);
}
.ai-search__brand-text h1 {
  margin: 0;
  font-size: 22px;
  font-weight: 800;
  letter-spacing: 0.02em;
  line-height: 1.1;
}
.ai-search__brand-text p {
  margin: 3px 0 0;
  font-size: 12px;
  color: var(--ai-muted);
}

.ai-search__bar {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 48px;
  padding: 4px 4px 4px 14px;
  border-radius: 999px;
  background: #fff;
  box-shadow: 0 8px 24px rgba(47, 107, 255, 0.1);
  border: 1px solid rgba(47, 107, 255, 0.08);
}
.ai-search__bar-ico {
  flex: 0 0 auto;
}
.ai-search__input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: none;
  font-size: 15px;
  background: transparent;
  color: var(--ai-text);
}
.ai-search__go {
  flex: 0 0 auto;
  height: 40px;
  padding: 0 18px;
  border: 0;
  border-radius: 999px;
  font-size: 14px;
  font-weight: 700;
  color: #fff;
  background: linear-gradient(160deg, #5b8cff, var(--ai-blue));
  box-shadow: 0 6px 14px rgba(47, 107, 255, 0.28);
}
.ai-search__go:disabled {
  opacity: 0.55;
}

.ai-search__scroll {
  position: relative;
  z-index: 1;
  flex: 1;
  min-height: 0;
  overflow: auto;
  -webkit-overflow-scrolling: touch;
  padding: 6px 16px 16px;
}

.ai-search__block {
  margin-top: 10px;
}
.ai-search__sec-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}
.ai-search__sec-head h3 {
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 700;
  color: #243552;
}
.ai-search__bar-mark {
  width: 3px;
  height: 14px;
  border-radius: 2px;
  background: var(--ai-blue);
  display: inline-block;
}
.ai-search__icon-btn {
  border: 0;
  background: transparent;
  color: var(--ai-muted);
  padding: 4px;
}

.ai-search__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.ai-search__tag {
  border: 1px solid rgba(143, 163, 190, 0.28);
  background: rgba(255, 255, 255, 0.88);
  color: #4a5d78;
  border-radius: 10px;
  padding: 7px 12px;
  font-size: 13px;
}
.ai-search__tag--more {
  color: var(--ai-blue);
  border-color: rgba(47, 107, 255, 0.25);
}

.ai-search__filters {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 10px;
  -webkit-overflow-scrolling: touch;
}
.ai-search__filter {
  flex: 0 0 auto;
  border: 0;
  border-radius: 999px;
  padding: 7px 14px;
  font-size: 13px;
  color: #516379;
  background: #eef2f7;
}
.ai-search__filter.is-active {
  color: #fff;
  background: linear-gradient(160deg, #5b8cff, var(--ai-blue));
  box-shadow: 0 6px 14px rgba(47, 107, 255, 0.25);
}

.ai-search__list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.ai-search__card {
  background: var(--ai-card);
  border-radius: 16px;
  padding: 14px 14px 12px;
  box-shadow: 0 8px 22px rgba(36, 64, 120, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.8);
}
.ai-search__card-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
}
.ai-search__card-site,
.ai-search__source {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 12px;
  color: #6d7f97;
}
.ai-search__card-site span,
.ai-search__source span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ai-search__favicon {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  object-fit: cover;
  background: #eef3fb;
  flex: 0 0 auto;
}
.ai-search__favicon--sm {
  width: 14px;
  height: 14px;
}
.ai-search__date {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--ai-muted);
}
.ai-search__card-title {
  margin: 0 0 8px;
  font-size: 17px;
  line-height: 1.35;
  font-weight: 700;
  color: var(--ai-blue);
  cursor: pointer;
}
.ai-search__card-snippet {
  margin: 0;
  font-size: 13px;
  line-height: 1.55;
  color: #6a7b93;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.ai-search__card-foot {
  margin-top: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.ai-search__open {
  display: inline-flex;
  align-items: center;
  height: 28px;
  padding: 0 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  color: var(--ai-blue);
  background: #eaf0ff;
  text-decoration: none;
}

.ai-search__pager {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 14px;
}
.ai-search__page-btn {
  border: 0;
  border-radius: 999px;
  padding: 8px 14px;
  font-size: 13px;
  font-weight: 600;
  color: #fff;
  background: linear-gradient(160deg, #5b8cff, var(--ai-blue));
}
.ai-search__page-btn:disabled {
  opacity: 0.4;
}
.ai-search__page-info {
  font-size: 13px;
  color: var(--ai-muted);
}

.ai-search__landing {
  text-align: center;
  padding: 48px 8px 20px;
}
.ai-search__landing h2 {
  margin: 0 0 8px;
  font-size: 18px;
}
.ai-search__landing p {
  margin: 0 auto 14px;
  max-width: 280px;
  font-size: 13px;
  color: var(--ai-muted);
  line-height: 1.5;
}
.ai-search__notice {
  border: 0;
  background: transparent;
  color: var(--ai-blue);
  font-size: 13px;
  text-decoration: underline;
}

.ai-search__alert {
  color: #d14343;
  font-size: 13px;
  margin: 8px 0;
}
.ai-search__muted {
  color: var(--ai-muted);
  font-size: 13px;
  text-align: center;
  padding: 18px 0;
}
.ai-search__loading {
  display: grid;
  place-items: center;
  gap: 10px;
  padding: 40px 0;
  color: var(--ai-muted);
  font-size: 13px;
}

.ai-search__detail {
  position: relative;
  z-index: 1;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.ai-search__detail-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 16px 8px;
}
.ai-search__back {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border: 0;
  background: rgba(255, 255, 255, 0.9);
  color: var(--ai-blue);
  border-radius: 999px;
  padding: 8px 12px;
  font-size: 13px;
  font-weight: 600;
  box-shadow: 0 4px 12px rgba(47, 107, 255, 0.1);
}
.ai-search__detail-tip {
  font-size: 12px;
  color: var(--ai-muted);
}
.ai-search__detail-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 4px 16px 12px;
}
.ai-search__detail-card {
  background: #fff;
  border-radius: 18px;
  padding: 16px;
  box-shadow: 0 10px 28px rgba(36, 64, 120, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.9);
}
.ai-search__detail-title {
  margin: 4px 0 14px;
  font-size: 20px;
  line-height: 1.35;
  font-weight: 800;
  color: var(--ai-blue);
}
.ai-search__detail-body {
  color: #3d4f68;
  font-size: 14px;
  line-height: 1.75;
}
.ai-search__detail-body :deep(h3) {
  margin: 0 0 10px;
  font-size: 16px;
  color: #243552;
}
.ai-search__detail-body :deep(p) {
  margin: 0 0 12px;
}
.ai-search__detail-body :deep(.ai-search__detail-empty) {
  color: var(--ai-muted);
  text-align: center;
  padding: 24px 0;
}
.ai-search__detail-actions {
  display: flex;
  gap: 10px;
  padding: 8px 16px 10px;
}
.ai-search__detail-open,
.ai-search__detail-close {
  flex: 1;
  height: 44px;
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 700;
  text-decoration: none;
  border: 0;
}
.ai-search__detail-open {
  color: #fff;
  background: linear-gradient(160deg, #5b8cff, var(--ai-blue));
  box-shadow: 0 8px 18px rgba(47, 107, 255, 0.28);
}
.ai-search__detail-close {
  color: var(--ai-blue);
  background: #eaf0ff;
}

.ai-search__dialog-body {
  padding: 8px 16px 16px;
  font-size: 14px;
  line-height: 1.5;
  color: #4b657d;
}
</style>
