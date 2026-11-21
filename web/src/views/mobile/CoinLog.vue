<template>
  <div class="coin-page custom-scroll" v-loading="loading">
    <div class="w3-bg" aria-hidden="true">
      <div class="w3-bg__glow w3-bg__glow--a"></div>
      <div class="w3-bg__glow w3-bg__glow--b"></div>
      <div class="w3-bg__grid"></div>
    </div>

    <div class="layout">
      <header class="w3-header van-safe-area-top">
        <button type="button" class="w3-back" :aria-label="t('coinLog.title')" @click="router.back()">
          <van-icon name="arrow-left" size="18" />
        </button>
        <div class="w3-header__brand">
          <span class="w3-header__mark">$</span>
          <div>
            <div class="w3-header__title">{{ t('coinLog.title') }}</div>
            <div class="w3-header__sub">{{ t('coinLog.subtitle') }}</div>
          </div>
        </div>
        <div v-if="total > 0" class="w3-header__chip">{{ total }}</div>
        <div v-else class="w3-header__chip w3-header__chip--ghost">SOHA</div>
      </header>

      <section class="w3-card search-card">
        <van-search
          v-model="query.model"
          shape="round"
          clearable
          :placeholder="t('coinLog.searchPlaceholder')"
          @search="onSearch"
          @clear="onSearch"
        >
          <template #action>
            <button type="button" class="search-btn" @click="onSearch">
              {{ t('coinLog.searchBtn') }}
            </button>
          </template>
        </van-search>
      </section>

      <section v-if="items.length" class="list-stack">
        <article
          v-for="item in items"
          :key="item.id"
          class="w3-card flow-row"
          :class="item.mark === 1 ? 'is-in' : item.mark === 0 ? 'is-out' : 'is-flat'"
        >
          <div class="flow-row__icon">
            <van-icon :name="item.mark === 1 ? 'plus' : item.mark === 0 ? 'minus' : 'exchange'" />
          </div>
          <div class="flow-row__main">
            <div class="flow-row__title">{{ formatCoinLogModel(t, item.model) }}</div>
            <div class="flow-row__meta">
              <span>{{ t('coinLog.columnBalance') }} {{ item.balance }}</span>
              <span>{{ dateFormat(item.created_at, 'MM-dd HH:mm:ss') }}</span>
            </div>
          </div>
          <div class="flow-row__amount">
            <template v-if="item.mark === 1">+{{ item.amount }}</template>
            <template v-else-if="item.mark === 0">-{{ item.amount }}</template>
            <template v-else>{{ item.amount }}</template>
          </div>
        </article>

        <div v-if="pageCount > 1" class="w3-card pager">
          <div class="pager__info">
            {{ t('coinLog.pageInfo', { page, pageCount, total }) }}
          </div>
          <van-pagination
            v-model="page"
            mode="simple"
            :total-items="total"
            :items-per-page="pageSize"
            :prev-text="t('coinLog.prevPage')"
            :next-text="t('coinLog.nextPage')"
            @change="onPageChange"
          />
        </div>
      </section>

      <div v-else-if="!loading" class="w3-card empty-card">
        <van-empty image="search" :description="t('coinLog.empty')" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { showFailToast } from 'vant'
import { dateFormat } from '@/utils/libs'
import { httpPost } from '@/utils/http'
import { formatCoinLogModel } from '@/utils/coinLogI18n'

defineOptions({ name: 'CoinLog' })

const { t } = useI18n()
const router = useRouter()

const items = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const query = ref({
  model: '',
  date: [],
})

const pageCount = computed(() => Math.max(1, Math.ceil(Number(total.value || 0) / Number(pageSize.value || 20))))

onMounted(() => {
  fetchData()
})

const onSearch = () => {
  page.value = 1
  fetchData()
}

const onPageChange = () => {
  fetchData()
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

const fetchData = () => {
  loading.value = true
  httpPost('/api/coinLog/list', {
    model: query.value.model,
    date: query.value.date,
    page: page.value,
    page_size: pageSize.value,
  })
    .then((res) => {
      if (res.data) {
        items.value = res.data.items || []
        total.value = Number(res.data.total || 0)
        page.value = Number(res.data.page || page.value)
        pageSize.value = Number(res.data.page_size || pageSize.value)
      }
    })
    .catch((e) => {
      showFailToast(t('coinLog.loadFailPrefix') + (e.message || ''))
    })
    .finally(() => {
      loading.value = false
    })
}
</script>

<style lang="stylus" scoped>
@import "@/assets/css/custom-scroll.styl"

.coin-page
  --w3-bg: #070b14
  --w3-card: #121a2b
  --w3-line: rgba(255, 255, 255, 0.08)
  --w3-text: #f2f5fb
  --w3-muted: #8b93a7
  --w3-accent: #2ee59d
  --w3-accent-2: #2bb4ff
  --w3-up: #0ecb81
  --w3-down: #f6465d
  --w3-radius: 18px
  position relative
  min-height 100vh
  min-height 100dvh
  background var(--w3-bg)
  color var(--w3-text)
  font-family 'Outfit', 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif
  font-size 15px
  overflow-x hidden
  -webkit-font-smoothing antialiased

.w3-bg
  position fixed
  inset 0
  z-index 0
  pointer-events none
  background: radial-gradient(1200px 600px at 10% -10%, rgba(46, 229, 157, 0.12), transparent 55%), radial-gradient(900px 500px at 100% 0%, rgba(43, 180, 255, 0.1), transparent 50%), linear-gradient(180deg, #070b14 0%, #0a1020 50%, #070b14 100%)

  &__glow
    position absolute
    border-radius 50%
    filter blur(60px)
    opacity 0.45
    &--a
      width 220px
      height 220px
      top 12%
      left -40px
      background rgba(46, 229, 157, 0.25)
    &--b
      width 260px
      height 260px
      top 40%
      right -80px
      background rgba(43, 180, 255, 0.18)

  &__grid
    position absolute
    inset 0
    opacity 0.22
    background-image linear-gradient(rgba(255,255,255,0.03) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.03) 1px, transparent 1px)
    background-size 28px 28px
    mask-image linear-gradient(180deg, rgba(0,0,0,0.55), transparent 85%)

.layout
  position relative
  z-index 1
  max-width 720px
  width 100%
  margin 0 auto
  padding 14px 14px calc(28px + env(safe-area-inset-bottom, 0px))
  display flex
  flex-direction column
  gap 14px
  box-sizing border-box

.w3-header
  display flex
  align-items center
  gap 10px
  padding 4px 2px 2px

  &__brand
    flex 1
    min-width 0
    display flex
    align-items center
    gap 10px

  &__mark
    width 40px
    height 40px
    border-radius 14px
    display flex
    align-items center
    justify-content center
    font-size 16px
    font-weight 800
    color #04110c
    background: linear-gradient(145deg, #2ee59d 0%, #2bb4ff 100%)
    box-shadow 0 8px 20px rgba(46, 229, 157, 0.22)

  &__title
    font-size 17px
    font-weight 800
    letter-spacing (-0.02em)
    line-height 1.25

  &__sub
    margin-top 3px
    font-size 12px
    color var(--w3-muted)
    overflow hidden
    text-overflow ellipsis
    white-space nowrap

  &__chip
    flex-shrink 0
    min-width 34px
    height 28px
    padding 0 10px
    border-radius 999px
    display inline-flex
    align-items center
    justify-content center
    font-size 12px
    font-weight 700
    color var(--w3-accent)
    background rgba(46, 229, 157, 0.12)
    border 1px solid rgba(46, 229, 157, 0.28)
    &--ghost
      color var(--w3-muted)
      background rgba(255, 255, 255, 0.04)
      border-color var(--w3-line)

.w3-back
  flex-shrink 0
  width 36px
  height 36px
  padding 0
  border none
  border-radius 12px
  display flex
  align-items center
  justify-content center
  color var(--w3-text)
  background rgba(255, 255, 255, 0.05)
  border 1px solid var(--w3-line)
  cursor pointer
  -webkit-tap-highlight-color transparent
  &:active
    background rgba(255, 255, 255, 0.08)

.w3-card
  background var(--w3-card)
  border 1px solid var(--w3-line)
  border-radius var(--w3-radius)
  box-shadow 0 8px 24px rgba(0, 0, 0, 0.18)

.search-card
  padding 6px 8px
  overflow hidden
  :deep(.van-search)
    padding 4px 2px
    background transparent
  :deep(.van-search__content)
    background rgba(255, 255, 255, 0.05)
    border-radius 12px
  :deep(.van-field__control)
    color var(--w3-text)
  :deep(.van-field__control::placeholder)
    color var(--w3-muted)
  :deep(.van-icon)
    color var(--w3-muted)

.search-btn
  margin 0
  padding 0 8px
  border none
  background transparent
  color var(--w3-accent)
  font-size 14px
  font-weight 700
  cursor pointer

.list-stack
  display flex
  flex-direction column
  gap 10px

.flow-row
  display flex
  align-items center
  gap 12px
  padding 14px

  &__icon
    width 40px
    height 40px
    border-radius 12px
    display flex
    align-items center
    justify-content center
    flex-shrink 0
    color #04110c
    background rgba(46, 229, 157, 0.9)
    :deep(.van-icon)
      font-size 18px

  &.is-out &__icon
    color #fff
    background rgba(246, 70, 93, 0.9)
  &.is-flat &__icon
    color #fff
    background rgba(43, 180, 255, 0.9)

  &__main
    flex 1
    min-width 0

  &__title
    font-size 14px
    font-weight 700
    color var(--w3-text)
    line-height 1.35
    word-break break-word

  &__meta
    margin-top 6px
    display flex
    flex-wrap wrap
    gap 8px 12px
    font-size 11px
    color var(--w3-muted)

  &__amount
    flex-shrink 0
    font-size 16px
    font-weight 800
    font-variant-numeric tabular-nums
    color var(--w3-text)

  &.is-in &__amount
    color var(--w3-up)
  &.is-out &__amount
    color var(--w3-down)

.pager
  padding 12px 14px

  &__info
    margin-bottom 10px
    text-align center
    font-size 12px
    font-weight 600
    color var(--w3-muted)

  :deep(.van-pagination)
    justify-content center

  :deep(.van-pagination__item)
    color var(--w3-accent-2)
    background rgba(255, 255, 255, 0.05)

  :deep(.van-pagination__item--active),
  :deep(.van-pagination__item:active)
    color #04110c
    background var(--w3-accent)

.empty-card
  padding 28px 12px
  :deep(.van-empty__description)
    color var(--w3-muted)
</style>
