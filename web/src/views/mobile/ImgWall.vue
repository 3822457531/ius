<template>
  <div class="market-page custom-scroll">
    <div class="layout">
      <div class="top-buttons glass-card" :data-active="currentView">
        <div class="button-item" :class="{ active: currentView === 'chart' }" @click="currentView = 'chart'">
          <span>{{ t('market.tabChart') }}</span>
        </div>
        <div class="button-item" :class="{ active: currentView === 'mint' }" @click="currentView = 'mint'">
          <span>{{ t('market.tabMint') }}</span>
        </div>
        <div class="button-item" :class="{ active: currentView === 'swap' }" @click="currentView = 'swap'">
          <span>{{ t('market.tabSwap') }}</span>
        </div>
        <div class="button-item" :class="{ active: currentView === 'myMint' }" @click="currentView = 'myMint'">
          <span>{{ t('market.tabMyMint') }}</span>
        </div>
      </div>

      <!-- Chart -->
      <div v-if="currentView === 'chart'" class="chart-section">
        <div v-if="!marketOperable" class="market-readonly-tip">{{ marketReadonlyTip }}</div>

        <section class="glass-card market-hero">
          <div class="market-hero__content">
            <div class="market-hero__head">
              <div class="market-hero__token">
                <span class="market-hero__icon">S</span>
                <div class="market-hero__meta">
                  <span class="market-hero__name">SOHA</span>
                  <span class="market-hero__label">{{ t('market.guidePrice') }}</span>
                </div>
              </div>
              <div class="market-hero__live">
                <i class="live-dot"></i>{{ t('market.live') }}
              </div>
            </div>

            <div v-if="priceLoading" class="market-hero__loading">
              <van-loading type="spinner" size="20" color="#ffffff" />
              <span>{{ t('market.loadingQuote') }}</span>
            </div>
            <template v-else>
              <div class="market-hero__price">${{ formatPrice(currentPrice) }}</div>
              <div class="market-hero__change">
                <span class="change-pill" :class="priceChange >= 0 ? 'up' : 'down'">
                  {{ priceChange >= 0 ? '+' : '' }}{{ priceChange }}%
                </span>
                <span class="change-hint">{{ t('market.vsYesterday') }}</span>
              </div>
            </template>

            <div class="market-hero__stats">
              <div class="hero-metric hero-metric--output">
                <div class="hero-metric__icon"><van-icon name="chart-trending-o" /></div>
                <div class="hero-metric__body">
                  <span class="hero-metric__label">{{ t('market.totalOutput') }}</span>
                  <span class="hero-metric__value">
                    {{ formatVolume(volume) }}<em>SOHA</em>
                  </span>
                </div>
              </div>
              <div class="hero-metric hero-metric--burn">
                <div class="hero-metric__icon"><van-icon name="fire-o" /></div>
                <div class="hero-metric__body">
                  <span class="hero-metric__label">{{ t('market.burnTitle') }}</span>
                  <span class="hero-metric__value">
                    {{ formatVolume(destroyedVolume) }}<em>SOHA</em>
                  </span>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="glass-card price-chart" :class="{ 'market-blurred': !klineDisplayEnabled }">
          <div class="chart-title">
            <div class="title-content">
              <h3>{{ t('market.priceTrend') }}</h3>
              <div class="sub-title">{{ t('market.klineDays', { n: fullPriceHistory.length }) }}</div>
            </div>
            <div class="zoom-controls">
              <button type="button" class="zoom-btn" :disabled="!klineDisplayEnabled || priceLoading" @click="handleZoom('in')">
                <van-icon name="plus" size="14" />
              </button>
              <button type="button" class="zoom-btn" :disabled="!klineDisplayEnabled || priceLoading" @click="handleZoom('out')">
                <van-icon name="minus" size="14" />
              </button>
            </div>
          </div>
          <div class="chart-body">
            <div class="chart-container" ref="chartRef"></div>
            <div v-if="priceLoading" class="chart-placeholder chart-placeholder--overlay">
              <van-loading type="spinner" size="24" color="#1677ff" />
            </div>
            <div v-if="!klineDisplayEnabled && !priceLoading" class="market-blur-overlay">
              <span>{{ t('market.klineDisabled') }}</span>
            </div>
          </div>
        </section>

        <section class="glass-card price-table-section" :class="{ 'market-blurred': !klineDisplayEnabled }">
          <div class="table-title">
            <div class="title-content">
              <span class="table-title-text">{{ t('market.klineData') }}</span>
            </div>
            <span class="table-sub">{{ t('market.klineRecent', { n: klineTableRows.length }) }}</span>
          </div>
          <div class="price-table-wrapper">
            <table class="price-table">
              <thead>
                <tr>
                  <th class="col-date">{{ t('market.colDate') }}</th>
                  <th class="col-price">{{ t('market.colPrice') }}</th>
                  <th class="col-change">{{ t('market.colChange') }}</th>
                  <th class="col-vol">{{ t('market.colVol') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(row, index) in klineTableRows"
                  :key="row.timestamp"
                  :class="{ 'row--latest': index === 0 }"
                >
                  <td class="col-date">
                    <span v-if="index === 0" class="latest-tag">{{ t('market.latest') }}</span>
                    {{ row.Date }}
                  </td>
                  <td class="col-price price-cell">${{ formatPrice(row.Price) }}</td>
                  <td :class="['col-change', 'change-cell', row.ChangeRate >= 0 ? 'up' : 'down']">
                    {{ row.ChangeRate >= 0 ? '+' : '' }}{{ formatChange(row.ChangeRate) }}%
                  </td>
                  <td class="col-vol">{{ formatVolume(row.Volume) }}</td>
                </tr>
                <tr v-if="!priceLoading && klineTableRows.length === 0">
                  <td colspan="4" class="empty-cell">{{ t('market.klineEmpty') }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>

      <!-- Mint -->
      <div v-if="currentView === 'mint'" class="mint-section">
        <div v-if="!marketOperable" class="market-readonly-tip">{{ marketReadonlyTip }}</div>

        <section class="glass-card mint-hero">
          <div class="mint-hero__content">
            <div class="mint-hero__badge">
              <i class="live-dot"></i>
              SOHA {{ t('market.mintFarmBadge') }}
            </div>
            <h2 class="mint-hero__title">{{ t('market.mintTitle') }}</h2>
            <p class="mint-hero__desc">{{ t('market.mintDesc') }}</p>
            <div class="mint-hero__stats">
              <div class="mint-stat-pill mint-stat-pill--dark">
                <div class="mint-stat-pill__icon"><van-icon name="gold-coin-o" /></div>
                <div class="mint-stat-pill__text">
                  <span class="label">{{ t('market.availablePower') }}</span>
                  <span class="value">{{ formatMintPower(form.power) }}</span>
                </div>
              </div>
            </div>
          </div>
        </section>

        <div v-if="mints.length === 0" class="glass-card mint-empty-panel">
          <div class="mint-empty-panel__icon"><van-icon name="cluster-o" size="36" /></div>
          <h3 class="mint-empty-panel__title">{{ t('market.mintEmptyTitle') }}</h3>
          <p class="mint-empty-panel__sub">{{ t('market.mintEmptySub') }}</p>
        </div>

        <div v-else class="mint-rig-list">
          <article
            v-for="(item, index) in mints"
            :key="item.id"
            class="glass-card mint-rig"
            :class="'mint-rig--tone-' + (index % 3)"
          >
            <div class="mint-rig__accent"></div>
            <div class="mint-rig__top">
              <div class="mint-rig__visual">
                <div class="rig-tower">
                  <div class="rig-tower__fan rig-tower__fan--l"></div>
                  <div class="rig-tower__fan rig-tower__fan--r"></div>
                  <div class="rig-tower__body">
                    <div class="rig-tower__led"></div>
                    <div class="rig-tower__led rig-tower__led--delay"></div>
                    <div class="rig-tower__slot"></div>
                  </div>
                  <div class="rig-tower__base"></div>
                </div>
              </div>
              <div class="mint-rig__meta">
                <span class="mint-rig__index">#{{ Number(item.sort_num) > 0 ? item.sort_num : index + 1 }}</span>
                <h4 class="mint-rig__name">{{ mintDisplayName(item, index) }}</h4>
                <span class="mint-rig__tag">{{ t('market.mintTag') }}</span>
              </div>
            </div>
            <div class="mint-rig__metrics">
              <div class="mint-metric">
                <div class="mint-metric__label">
                  <van-icon name="fire-o" />
                  {{ t('market.investPower') }}
                </div>
                <span class="mint-metric__value mint-metric__value--power">{{ formatMintPower(item.power) }}</span>
              </div>
              <div class="mint-metric">
                <div class="mint-metric__label">
                  <van-icon name="orders-o" />
                  {{ t('market.subscribedLabel') }}
                </div>
                <span class="mint-metric__value">{{ t('market.subscribedCount', { n: item.sales || 0 }) }}</span>
              </div>
            </div>
            <button type="button" class="mint-activate-btn" @click="buyKj(item)">
              <van-icon name="play-circle-o" size="18" />
              <span>{{ t('market.startMint') }}</span>
            </button>
          </article>
        </div>
      </div>

      <!-- My mint -->
      <div v-if="currentView === 'myMint'" class="my-mint-section">
        <section class="glass-card my-mint-head">
          <h2 class="section-title">{{ t('market.myMintTitle') }}</h2>
          <div class="my-mint-summary">
            <div class="my-mint-summary__item">
              <span class="label">{{ t('market.running') }}</span>
              <span class="value">{{ myMintOrders.length }}<em v-if="t('market.unitMachine')">{{ t('market.unitMachine') }}</em></span>
            </div>
            <div class="my-mint-summary__item">
              <span class="label">{{ t('market.totalInvestPower') }}</span>
              <span class="value">{{ formatMintPower(myMintTotalPower) }}</span>
            </div>
            <div class="my-mint-summary__item">
              <span class="label">{{ t('market.totalProduced') }}</span>
              <span class="value value--soha">{{ formatMintCoin(myMintTotalCoin) }}</span>
            </div>
          </div>
          <div class="my-mint-ledger">
            <span class="my-mint-ledger__label">{{ t('market.sohaBalance') }}</span>
            <span class="my-mint-ledger__value">{{ formatMintCoin(form.coin) }}</span>
          </div>
        </section>

        <van-loading v-if="myMintLoading" class="my-mint-loading" size="24px" vertical color="#1989fa">{{ t('market.loading') }}</van-loading>

        <div v-else-if="myMintOrders.length === 0" class="glass-card my-mint-empty">
          <van-empty image-size="72" :description="t('market.myMintEmpty')" />
          <button type="button" class="my-mint-empty__btn" @click="currentView = 'mint'">
            {{ t('market.goActivateMint') }}
          </button>
        </div>

        <div v-else class="my-mint-list">
          <article v-for="(item, index) in myMintOrders" :key="item.id" class="glass-card my-mint-card">
            <div class="my-mint-card__head">
              <div class="my-mint-card__title-wrap">
                <span class="my-mint-card__index">#{{ index + 1 }}</span>
                <h4 class="my-mint-card__title">{{ item.subject || t('market.mintDefaultName') }}</h4>
              </div>
              <span class="my-mint-card__status">{{ t('market.running') }}</span>
            </div>
            <div class="my-mint-card__metrics">
              <div class="my-mint-metric">
                <span class="my-mint-metric__label">{{ t('market.investPower') }}</span>
                <span class="my-mint-metric__value">{{ formatMintPower(item.power) }}</span>
              </div>
              <div class="my-mint-metric">
                <span class="my-mint-metric__label">{{ t('market.producedSoha') }}</span>
                <span class="my-mint-metric__value my-mint-metric__value--coin">{{ formatMintCoin(item.js_coin) }}</span>
              </div>
              <div class="my-mint-metric">
                <span class="my-mint-metric__label">{{ t('market.activatedAt') }}</span>
                <span class="my-mint-metric__value my-mint-metric__value--time">{{ formatMintTime(item.created_at) }}</span>
              </div>
            </div>
            <button type="button" class="my-mint-card__detail-btn" @click="goMintCoinDetail(item)">
              <van-icon name="orders-o" size="16" />
              {{ t('market.mintDetail') }}
            </button>
          </article>
        </div>
      </div>

      <!-- Swap -->
      <div v-if="currentView === 'swap'" class="swap-section">
        <div v-if="!marketOperable" class="market-readonly-tip">{{ marketReadonlyTip }}</div>

        <section class="glass-card swap-container" :class="{ 'market-readonly': !marketOperable }">
          <div class="swap-header">
            <h2 class="section-title">{{ t('market.swapTitle') }}</h2>
            <p class="swap-header__desc">{{ t('market.swapDesc') }}</p>
          </div>

          <div class="token-input-container">
            <div class="token-input-header">
              <span>{{ t('market.pay') }}</span>
              <span class="balance">{{ t('market.sohaBalanceLabel', { n: form.coin }) }}</span>
            </div>
            <div class="token-input">
              <div class="token-selector-fixed">
                <div class="token-info">
                  <span class="token-icon-placeholder soha">S</span>
                  <span class="token-symbol">SOHA</span>
                </div>
              </div>
              <input
                v-model="fromAmount"
                type="number"
                placeholder="0.0"
                class="amount-input"
                :disabled="!marketOperable"
                @input="calculateToAmount"
                max="500000"
                min="0"
                step="0.01"
              >
            </div>
            <div class="token-input-footer">
              <span>{{ t('market.sohaAmountLabel', { n: fromAmount || '0' }) }}</span>
              <div class="quick-amounts">
                <span class="quick-btn" :class="{ disabled: !marketOperable }" @click="marketOperable && setMaxAmount()">{{ t('market.all') }}</span>
                <span class="quick-btn" :class="{ disabled: !marketOperable }" @click="marketOperable && setPercentAmount(50)">50%</span>
                <span class="quick-btn" :class="{ disabled: !marketOperable }" @click="marketOperable && setPercentAmount(25)">25%</span>
              </div>
            </div>
          </div>

          <div class="swap-arrow-container">
            <div class="swap-arrow"><van-icon name="exchange" /></div>
          </div>

          <div class="token-input-container">
            <div class="token-input-header">
              <span>{{ t('market.receive') }}</span>
              <span class="balance">{{ t('market.powerBalanceLabel', { n: availableBalance }) }}</span>
            </div>
            <div class="token-input">
              <div class="token-selector-fixed">
                <div class="token-info">
                  <span class="token-icon-placeholder power">{{ t('market.powerShort') }}</span>
                  <span class="token-symbol">{{ t('market.powerSymbol') }}</span>
                </div>
              </div>
              <input v-model="toAmount" type="number" placeholder="0.0" class="amount-input" readonly>
            </div>
            <div class="token-input-footer">
              <span>{{ t('market.powerValueLabel', { n: toAmount || '0' }) }}</span>
            </div>
          </div>

          <div v-if="fromAmount && toAmount" class="swap-info">
            <div class="info-header">{{ t('market.swapDetail') }}</div>
            <div class="info-row">
              <span>{{ t('market.currentSohaPrice') }}</span>
              <strong>${{ formatPrice(swapSohaPrice) }}</strong>
            </div>
            <div class="info-row">
              <span>{{ t('market.anchorPowerPrice') }}</span>
              <strong>${{ formatPrice(swapPowerPrice) }}</strong>
            </div>
            <div class="info-row">
              <span>{{ t('market.exchangeRate') }}</span>
              <strong>{{ t('market.rateFormula', { rate: exchangeRate }) }}</strong>
            </div>
            <div class="info-row">
              <span>{{ t('market.singleLimit') }}</span>
              <strong class="limit-text">{{ t('market.maxLimit') }}</strong>
            </div>
          </div>

          <div class="swap-button-container">
            <van-button
              v-if="!marketOperable"
              type="primary"
              block
              class="swap-button disabled-button"
              disabled
            >{{ t('market.marketClosedBtn') }}</van-button>
            <van-button
              v-else-if="!fromAmount || !toAmount"
              type="primary"
              block
              class="swap-button disabled-button"
              disabled
            >{{ t('market.enterAmountBtn') }}</van-button>
            <van-button
              v-else-if="Number(fromAmount) > Number(form.coin)"
              type="primary"
              block
              class="swap-button error-button"
              disabled
            >{{ t('market.insufficientBtn') }}</van-button>
            <van-button
              v-else
              type="primary"
              block
              class="swap-button"
              :loading="isSwapping"
              @click="handleSwap"
            >{{ isSwapping ? t('market.processing') : t('market.confirmSwap') }}</van-button>
          </div>
        </section>
      </div>

      <van-dialog
        v-model:show="showSwapConfirm"
        :title="t('market.confirmSwap')"
        show-cancel-button
        :confirm-button-text="t('market.confirmSwap')"
        :cancel-button-text="t('market.cancel')"
        teleport="body"
        :z-index="3000"
        class="swap-confirm-dialog"
        :before-close="beforeCloseSwapConfirm"
      >
        <div class="swap-confirm-dialog__body">
          <div class="detail-row">
            <span class="label">{{ t('market.exchangeRate') }}</span>
            <span class="value">{{ t('market.rateFormula', { rate: exchangeRate }) }}</span>
          </div>
          <div class="detail-row">
            <span class="label">{{ t('market.actualReceive') }}</span>
            <span class="value highlight">{{ t('market.receivePower', { n: toAmount }) }}</span>
          </div>
          <div class="detail-row">
            <span class="label">{{ t('market.singleLimit') }}</span>
            <span class="value limit-text">{{ t('market.maxLimit') }}</span>
          </div>
          <p class="risk-notice">{{ t('market.swapRiskTip') }}</p>
        </div>
      </van-dialog>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, onActivated, computed, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { showToast, showFailToast, showConfirmDialog, showSuccessToast, showDialog } from 'vant'
import * as echarts from 'echarts'
import { httpGet, httpPost } from '@/utils/http'
import { useRoute, useRouter } from 'vue-router'
import { checkSession } from '@/action/session'
import { dateFormat } from '@/utils/libs'
import { enrichSohaPriceHistoryChangeRates } from '@/utils/sohaPrice'

defineOptions({ name: 'ImgWall' })

const { t, locale } = useI18n()

const useWanUnit = () => {
  const loc = locale.value || ''
  return loc.startsWith('zh') || loc === 'ja' || loc.startsWith('kr')
}
const formatCompactNumber = (num) => {
  const n = Number(num) || 0
  if (useWanUnit() && n >= 10000) return (n / 10000).toFixed(2) + t('market.wan')
  if (n >= 1000) return (n / 1000).toFixed(2) + 'K'
  return n.toFixed(0)
}

const route = useRoute()
const router = useRouter()
const loginUser = ref(null)
const sohaMarketEnabled = ref(true)
const klineDisplayEnabled = ref(true)
const allowQ2 = ref(false)
const allowMint = ref(false)
const isVip = ref(false)
const currentView = ref('chart')
const chartRef = ref(null)
const chartInstance = ref(null)
const fullPriceHistory = ref([])
const priceHistory = ref([])
const priceLoading = ref(true)
const volume = ref(0)
const destroyedVolume = ref(0)
const mints = ref([])
const myMintOrders = ref([])
const myMintLoading = ref(false)
const mintReleaseRates = ref({})
const mintReleaseLabel = computed(() => t('market.daysRange'))

const swapPowerPrice = ref(0.01)
const swapSohaPrice = ref(0)
const form = ref({ username: '', coin: 0, power: 0 })
const availableBalance = ref(0)

const fromAmount = ref('')
const toAmount = ref('')
const isSwapping = ref(false)
const showSwapConfirm = ref(false)

const exchangeRate = computed(() => {
  const sohaPrice = swapSohaPrice.value || currentPrice.value || 0.01
  const anchor = swapPowerPrice.value || 0.01
  if (!sohaPrice || !anchor) return '0.00'
  return (sohaPrice / anchor).toFixed(4)
})

const marketOperable = computed(() => sohaMarketEnabled.value && allowQ2.value)

const marketReadonlyTip = computed(() => {
  if (!sohaMarketEnabled.value) {
    return t('market.marketNotOpen')
  }
  if (!allowQ2.value) {
    return t('market.noTradePermission')
  }
  return ''
})

const showMintSoldOutDialog = () => {
  showDialog({
    title: t('market.tip'),
    message: t('market.soldOut'),
  })
}

const showMintVipRequiredDialog = () => {
  showDialog({
    title: t('market.tip'),
    message: t('market.soldOut'),
  })
}

const currentPrice = computed(() => {
  if (!priceHistory.value?.length) return 0
  return priceHistory.value[priceHistory.value.length - 1].Price
})

const priceChange = computed(() => {
  if (!fullPriceHistory.value?.length) return 0
  const last = fullPriceHistory.value[fullPriceHistory.value.length - 1]
  return Number(formatChange(last.ChangeRate))
})

const klineTableRows = computed(() => {
  if (!fullPriceHistory.value.length) return []
  return [...fullPriceHistory.value].reverse().slice(0, 30)
})

const formatChange = (val) => {
  const num = Number(val)
  if (!num && num !== 0) return '0.00'
  return num.toFixed(2)
}

const formatVolume = (val) => {
  const num = Number(val)
  if (!num && num !== 0) return '0'
  return formatCompactNumber(num)
}

const formatMintPower = (val) => {
  const num = Number(val)
  if (!num && num !== 0) return '0'
  if (num >= 10000 && useWanUnit()) return (num / 10000).toFixed(2) + ' ' + t('market.wan')
  if (num >= 1000) return (num / 1000).toFixed(2) + 'K'
  return num.toLocaleString(locale.value || undefined, { maximumFractionDigits: 2 })
}

/** 展示 1号算力机～6号算力机；库名缺失时按序号兜底 */
const mintDisplayName = (item, index) => {
  const name = String(item?.name || '').trim()
  if (name) return name
  const n = Number(item?.sort_num)
  if (n >= 1 && n <= 6) return t('market.mintNumberName', { n })
  return t('market.mintNumberName', { n: Number(index) + 1 })
}

const formatMintCoin = (val) => {
  const num = Number(val)
  if (!num && num !== 0) return '0'
  if (num >= 10000 && useWanUnit()) return (num / 10000).toFixed(2) + ' ' + t('market.wan')
  if (num >= 1000) return (num / 1000).toFixed(2) + 'K'
  return num.toLocaleString(locale.value || undefined, { maximumFractionDigits: 4 })
}

const formatMintTime = (val) => {
  if (!val) return '-'
  const ts = Number(val)
  if (ts > 1e12) return dateFormat(ts)
  if (ts > 1e9) return dateFormat(ts * 1000)
  return dateFormat(val)
}

const myMintTotalPower = computed(() =>
  myMintOrders.value.reduce((sum, item) => sum + (Number(item.power) || 0), 0)
)

const myMintTotalCoin = computed(() =>
  myMintOrders.value.reduce((sum, item) => sum + (Number(item.js_coin) || 0), 0)
)

const parseDateString = (dateStr) => {
  const [month, day] = dateStr.split('/').map(Number)
  const now = new Date()
  const date = new Date(now.getFullYear(), month - 1, day)
  if (date > now) date.setFullYear(date.getFullYear() - 1)
  return date
}

const transformData = (data) => {
  return data.map((item, index, arr) => {
    const prevItem = index > 0 ? arr[index - 1] : item
    const openPrice = prevItem.Price
    const closePrice = item.Price
    return {
      value: [
        openPrice,
        closePrice,
        Math.min(openPrice, closePrice) * 0.99,
        Math.max(openPrice, closePrice) * 1.01,
      ],
      itemStyle: {
        color: closePrice >= openPrice ? '#07c160' : '#ee0a24',
        borderColor: closePrice >= openPrice ? '#07c160' : '#ee0a24',
      },
    }
  })
}

const disposeChart = () => {
  if (chartInstance.value) {
    chartInstance.value.dispose()
    chartInstance.value = null
  }
}

const initChart = () => {
  if (!chartRef.value || !fullPriceHistory.value.length) return
  disposeChart()
  const chart = echarts.init(chartRef.value)
  const totalDays = fullPriceHistory.value.length
  const startPercent = Math.max(0, (1 - 7 / totalDays) * 100)
  chart.setOption({
    animation: false,
    backgroundColor: 'transparent',
    grid: { left: '2%', right: '10%', bottom: '20%', top: '6%', containLabel: true },
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(255,255,255,0.96)',
      borderColor: 'rgba(31,42,55,0.12)',
      borderWidth: 1,
      padding: [10, 14],
      textStyle: { color: '#1f2a37', fontSize: 12 },
      extraCssText: 'border-radius:12px;box-shadow:0 8px 24px rgba(31,42,55,0.14);',
      formatter: (params) => {
        const idx = params?.[0]?.dataIndex
        const item = fullPriceHistory.value[idx]
        if (!item) return ''
        const color = item.ChangeRate >= 0 ? '#07c160' : '#ee0a24'
        const arrow = item.ChangeRate >= 0 ? '▲' : '▼'
        return `<div style="line-height:1.7">
          <div style="color:rgba(31,42,55,0.55);font-size:12px;margin-bottom:2px">${item.Date}</div>
          <div style="font-size:16px;font-weight:600;color:#1f2a37">$${formatPrice(item.Price)}</div>
          <div style="color:${color};font-size:13px;font-weight:500">${arrow} ${item.ChangeRate >= 0 ? '+' : ''}${formatChange(item.ChangeRate)}%</div>
        </div>`
      },
    },
    xAxis: {
      type: 'category',
      data: fullPriceHistory.value.map(i => i.Date),
      axisLine: { lineStyle: { color: 'rgba(31,42,55,0.14)' } },
      axisTick: { show: false },
      axisLabel: { fontSize: 11, color: 'rgba(31,42,55,0.55)', margin: 10 },
    },
    yAxis: {
      type: 'value',
      scale: true,
      position: 'right',
      splitLine: { lineStyle: { color: 'rgba(31,42,55,0.1)', type: 'dashed' } },
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        fontSize: 10,
        color: 'rgba(31,42,55,0.55)',
        formatter: (v) => Number(v).toFixed(6),
      },
    },
    dataZoom: [
      { type: 'inside', start: startPercent, end: 100 },
      {
        type: 'slider',
        bottom: 4,
        start: startPercent,
        end: 100,
        height: 22,
        borderColor: 'transparent',
        backgroundColor: 'rgba(22,119,255,0.08)',
        fillerColor: 'rgba(22,119,255,0.22)',
        handleStyle: { color: '#1677ff', borderColor: '#1677ff' },
        dataBackground: { lineStyle: { color: 'rgba(22,119,255,0.45)' }, areaStyle: { color: 'rgba(22,119,255,0.1)' } },
      },
    ],
    series: [{
      type: 'candlestick',
      barMaxWidth: 14,
      itemStyle: {
        color: '#07c160',
        color0: '#ee0a24',
        borderColor: '#07c160',
        borderColor0: '#ee0a24',
      },
      data: transformData(fullPriceHistory.value),
    }],
  })
  chartInstance.value = chart
  requestAnimationFrame(() => chartInstance.value?.resize())
}

const handleZoom = (type) => {
  if (!chartInstance.value) return
  const option = chartInstance.value.getOption()
  const dz = option.dataZoom[0]
  const range = dz.end - dz.start
  const center = (dz.start + dz.end) / 2
  const ratio = type === 'in' ? 0.7 : 1.3
  const newRange = Math.min(100, Math.max(10, range * ratio))
  chartInstance.value.dispatchAction({
    type: 'dataZoom',
    start: Math.max(0, center - newRange / 2),
    end: Math.min(100, center + newRange / 2),
  })
}

const initPriceData = async () => {
  priceLoading.value = true
  try {
    const res = await httpGet('/api/trade/priceHistory')
    if (res.code !== 0 || !Array.isArray(res.data)) return
    const sorted = res.data
      .map(item => ({
        ...item,
        timestamp: parseDateString(item.Date).getTime(),
      }))
      .sort((a, b) => a.timestamp - b.timestamp)
    fullPriceHistory.value = enrichSohaPriceHistoryChangeRates(sorted)
    const lastIndex = sorted.length - 1
    priceHistory.value = fullPriceHistory.value.slice(Math.max(0, lastIndex - 6))
    if (!swapSohaPrice.value && currentPrice.value) {
      swapSohaPrice.value = currentPrice.value
    }
  } catch (e) {
    console.error('获取K线失败:', e)
  } finally {
    priceLoading.value = false
    await nextTick()
    initChart()
  }
}

const loadMarketStats = async () => {
  try {
    const [sumRes, destroyRes] = await Promise.all([
      httpGet('/api/trade/sumCoin'),
      httpGet('/api/trade/destroyedCoin'),
    ])
    if (sumRes.code === 0 && sumRes.data?.total !== undefined) {
      volume.value = sumRes.data.total
    }
    if (destroyRes.code === 0 && destroyRes.data !== undefined && destroyRes.data !== null) {
      destroyedVolume.value = Number(destroyRes.data) || 0
    }
  } catch (e) {
    console.error('获取市场统计失败:', e)
  }
}

const formatPrice = (val) => {
  const num = Number(val)
  if (!num && num !== 0) return '0.00'
  return num >= 1 ? num.toFixed(4) : num.toFixed(6)
}

const loadMintData = async () => {
  try {
    const [listRes, configRes] = await Promise.all([
      httpGet('/api/mint/list'),
      httpGet('/api/mint/releaseConfig'),
    ])
    if (listRes.code === 0 && Array.isArray(listRes.data)) {
      mints.value = listRes.data
    }
    if (configRes.data) {
      mintReleaseRates.value = configRes.data
    }
  } catch (e) {
    console.error('获取算力机数据失败:', e)
  }
}

const loadMyMintOrders = async () => {
  if (!loginUser.value?.id) return
  myMintLoading.value = true
  try {
    const res = await httpGet('/api/mintOrder/getMintOrderList')
    if (res.code === 0 && Array.isArray(res.data)) {
      myMintOrders.value = res.data
    } else {
      myMintOrders.value = []
    }
  } catch (e) {
    console.error('获取我的算力机列表失败:', e)
    myMintOrders.value = []
  } finally {
    myMintLoading.value = false
  }
}

const goMintCoinDetail = (item) => {
  if (!item?.order_no) return
  router.push({
    path: '/mobile/coin',
    query: {
      productId: item.order_no,
      title: item.subject || t('market.mintDefaultName'),
    },
  })
}

const loadSwapInfo = async () => {
  try {
    const res = await httpGet('/api/trade/swapInfo')
    if (res.code === 0 && res.data) {
      swapPowerPrice.value = Number(res.data.swap_power_price) || 0.01
      swapSohaPrice.value = Number(res.data.q2_price) || 0
    }
  } catch (e) {
    console.error('获取Swap配置失败:', e)
  }
}

const loadUserProfile = async () => {
  try {
    const res = await httpGet('/api/user/profile')
    form.value = {
      username: res.data.username || '',
      coin: res.data.coin || 0,
      power: res.data.power || 0,
    }
    availableBalance.value = res.data.power || 0
    if (res.data.soha_market_enabled !== undefined) {
      sohaMarketEnabled.value = !!res.data.soha_market_enabled
    }
    if (res.data.soha_kline_enabled !== undefined) {
      klineDisplayEnabled.value = !!res.data.soha_kline_enabled
    }
    if (res.data.allow_q2 !== undefined) {
      allowQ2.value = !!res.data.allow_q2
    }
    if (res.data.allow_mint !== undefined) {
      allowMint.value = !!res.data.allow_mint
    }
    if (res.data.vip !== undefined) {
      isVip.value = !!res.data.vip
    }
  } catch (e) {
    showFailToast(t('market.profileFail'))
  }
}

const estimateDailyOutput = (item) => {
  const power = Number(item?.power) || 0
  const rates = mintReleaseRates.value || {}
  let rate = Number(rates.kj_zj_plus) || 0
  if (rate <= 0) rate = Number(item?.fee) || 0
  if (rate <= 0 || power <= 0) return '0'
  return (power * rate).toFixed(4)
}

const buyKj = (item) => {
  if (!loginUser.value?.id) {
    showFailToast(t('market.loginRequired'))
    return
  }
  if (!allowMint.value) {
    showMintSoldOutDialog()
    return
  }
  showConfirmDialog({
    title: t('market.buyConfirmTitle'),
    message: t('market.buyConfirmMsg', { name: item.name, power: item.power }),
  }).then(() => {
    httpPost('/api/mintOrder/buyKj', {
      mint_id: item.id,
      user_id: loginUser.value.id,
    }).then(() => {
      showSuccessToast(t('market.buySuccess'))
      loadUserProfile()
      loadMyMintOrders()
      loadMintData()
    }).catch(e => {
      showFailToast(t('market.buyFailPrefix') + e.message)
    })
  }).catch(() => {})
}

const calculateToAmount = () => {
  if (!fromAmount.value) {
    toAmount.value = ''
    return
  }
  const sohaAmount = Number(fromAmount.value)
  if (sohaAmount <= 0) {
    toAmount.value = ''
    return
  }
  if (sohaAmount > 500000) {
    showFailToast(t('market.swapMaxAdjust'))
    fromAmount.value = '500000'
    toAmount.value = (500000 * (Number(exchangeRate.value) || 0)).toFixed(2)
    return
  }
  toAmount.value = (sohaAmount * (Number(exchangeRate.value) || 0)).toFixed(2)
}

const setMaxAmount = () => {
  const maxAllowed = 500000
  const userBalance = Number(form.value.coin)
  const actualMax = Math.min(userBalance, maxAllowed)
  fromAmount.value = actualMax.toString()
  calculateToAmount()
}

const setPercentAmount = (percent) => {
  const maxAllowed = 500000
  const userBalance = Number(form.value.coin)
  const actualAmount = Math.min(userBalance * percent / 100, maxAllowed)
  fromAmount.value = actualAmount.toFixed(2)
  calculateToAmount()
}

const handleSwap = () => {
  if (!marketOperable.value) {
    showFailToast(marketReadonlyTip.value || t('market.notOperable'))
    return
  }
  if (!fromAmount.value || !toAmount.value) {
    showFailToast(t('market.swapAmountRequired'))
    return
  }
  if (Number(fromAmount.value) > 500000) {
    showFailToast(t('market.swapMaxLimit'))
    return
  }
  if (Number(fromAmount.value) > Number(form.value.coin)) {
    showFailToast(t('market.sohaInsufficient'))
    return
  }
  showSwapConfirm.value = true
}

const confirmSwap = async () => {
  showToast({ type: 'loading', message: t('market.processing'), forbidClick: true, duration: 2000 })
  try {
    isSwapping.value = true
    const response = await httpPost('/api/user/swapQ2ToPower', {
      q2_amount: Number(fromAmount.value),
      power_amount: Number(toAmount.value),
    })
    if (response.code === 0) {
      showSuccessToast(t('market.swapSuccess'))
      fromAmount.value = ''
      toAmount.value = ''
      await loadUserProfile()
    } else {
      showFailToast(t('market.swapFailPrefix') + response.message)
    }
  } catch (error) {
    showFailToast(t('market.swapFailPrefix') + error.message)
  } finally {
    isSwapping.value = false
  }
}

const beforeCloseSwapConfirm = (action) => {
  if (action !== 'confirm') return true
  if (isSwapping.value) return false
  confirmSwap()
  return true
}

watch(currentView, (view, oldView) => {
  if (oldView === 'chart' && view !== 'chart') {
    disposeChart()
  }
  if (view === 'chart') {
    nextTick(() => {
      nextTick(() => initChart())
    })
  }
  if (view === 'mint' && mints.value.length === 0) {
    loadMintData()
  }
  if (view === 'myMint') {
    loadMyMintOrders()
  }
})

onUnmounted(() => {
  disposeChart()
})

onMounted(() => {
  const view = route.query.view
  if (['chart', 'mint', 'swap', 'myMint'].includes(view)) {
    currentView.value = view
  }

  initPriceData()
  loadMarketStats()
  loadSwapInfo()
  loadMintData()
  loadUserProfile()

  checkSession().then(user => {
    loginUser.value = user
    if (user?.allow_q2 !== undefined) {
      allowQ2.value = !!user.allow_q2
    }
    if (currentView.value === 'myMint') {
      loadMyMintOrders()
    }
  }).catch(() => {})
})

const imgWallActivatedOnce = ref(false)
onActivated(() => {
  if (!imgWallActivatedOnce.value) {
    imgWallActivatedOnce.value = true
    return
  }
  // Tab 切回时静默刷新余额与行情，无整页闪烁
  loadUserProfile()
  loadMarketStats()
  loadSwapInfo()
})
</script>

<style lang="stylus" scoped>
@import "@/assets/css/custom-scroll.styl"

.market-page
  position relative
  min-height 100vh
  min-height 100dvh
  background #f3f5f8
  color #2c3e50
  overflow-x hidden
  overflow-y auto
  &:before
    content ''
    position fixed
    inset 0
    background radial-gradient(800px 800px at 15% 25%, rgba(255, 255, 255, 0.15), transparent 55%), radial-gradient(700px 700px at 85% 10%, rgba(255, 255, 255, 0.12), transparent 55%), radial-gradient(600px 600px at 70% 80%, rgba(255, 255, 255, 0.1), transparent 65%)
    pointer-events none
    z-index 0
  &:after
    content ''
    position fixed
    inset 0
    background linear-gradient(135deg, rgba(255,255,255,0.1) 0%, transparent 30%, rgba(255,255,255,0.08) 60%, transparent 100%)
    pointer-events none
    z-index 0

.gradient-bg
  position fixed
  inset 0
  background linear-gradient(135deg, #667eea 0%, #764ba2 25%, #f093fb 50%, #4facfe 75%, #00f2fe 100%)
  z-index 0

.layout
  position relative
  z-index 1
  max-width 1120px
  width 100%
  margin 0 auto
  padding 18px 16px calc(80px + env(safe-area-inset-bottom, 0px))
  display flex
  flex-direction column
  gap 16px
  box-sizing border-box

.glass-card
  position relative
  background rgba(255,255,255,0.98)
  border 1px solid rgba(79,172,254,0.15)
  box-shadow 0 8px 32px rgba(0,0,0,0.1), 0 2px 8px rgba(0,0,0,0.06)
  border-radius 24px
  padding 20px 16px
  backdrop-filter blur(12px)
  overflow hidden

.section-kicker
  display inline-block
  margin-bottom 4px
  font-size 11px
  font-weight 800
  letter-spacing 0.14em
  color #4facfe

.section-title
  margin 0 0 6px
  font-size 20px
  font-weight 800
  color #1f2937
  letter-spacing 0.02em
  line-height 1.25

.top-buttons
  display flex
  padding 4px !important
  border-radius 18px !important
  position relative
  gap 0

  .button-item
    flex 1
    text-align center
    padding 12px 0
    font-size 14px
    font-weight 700
    color #7b8794
    border-radius 14px
    position relative
    z-index 1
    min-width 0
    transition color 0.25s ease
    cursor pointer
    -webkit-tap-highlight-color transparent

    span
      display block
      white-space nowrap
      overflow hidden
      text-overflow ellipsis
      padding 0 2px

    &.active
      color #fff

  &:before
    content ''
    position absolute
    width calc(25% - 4px)
    height calc(100% - 8px)
    background #0f172a
    border-radius 14px
    transition all 0.3s cubic-bezier(0.4, 0, 0.2, 1)
    top 4px
    z-index 0
    box-shadow 0 6px 16px rgba(15, 23, 42, 0.3)

  &[data-active="chart"]:before
    left 4px
  &[data-active="mint"]:before
    left calc(25% + 1px)
  &[data-active="swap"]:before
    left calc(50% - 1px)
  &[data-active="myMint"]:before
    left calc(75% - 4px)

.chart-section
  display flex
  flex-direction column
  gap 14px

  .market-readonly-tip
    margin 0
    padding 10px 12px
    background #fff7e6
    color #d48806
    border 1px solid #ffe7ba
    border-radius 12px
    font-size 13px
    line-height 1.5

  .market-blurred
    filter blur(6px)
    user-select none
    pointer-events none

.market-hero
  padding 0 !important
  background linear-gradient(145deg, #1e3a5f 0%, #2b4a7a 45%, #3b6bb5 100%) !important
  border none !important
  box-shadow 0 10px 28px rgba(30, 58, 95, 0.28) !important
  overflow hidden

  &__content
    position relative
    z-index 1
    padding 20px 16px 16px
    color #fff

  &__head
    display flex
    align-items flex-start
    justify-content space-between
    margin-bottom 14px

  &__token
    display flex
    align-items center
    gap 12px

  &__icon
    width 46px
    height 46px
    border-radius 14px
    background #1677ff
    box-shadow 0 6px 16px rgba(0, 0, 0, 0.22)
    display flex
    align-items center
    justify-content center
    font-size 18px
    font-weight 700
    color #fff
    letter-spacing 0.02em

  &__meta
    display flex
    flex-direction column
    gap 2px

  &__name
    font-size 20px
    font-weight 700
    letter-spacing 0.04em
    color #fff

  &__label
    font-size 12px
    font-weight 500
    color rgba(255, 255, 255, 0.68)

  &__live
    display flex
    align-items center
    gap 5px
    font-size 11px
    font-weight 600
    padding 6px 12px
    background rgba(0, 0, 0, 0.22)
    border 1px solid rgba(255, 255, 255, 0.14)
    border-radius 999px
    color #fff

    .live-dot
      width 6px
      height 6px
      border-radius 50%
      background #07c160
      box-shadow 0 0 8px rgba(7, 193, 96, 0.7)
      animation live-pulse 1.8s ease-in-out infinite

  &__loading
    display flex
    align-items center
    gap 8px
    padding 12px 0 8px
    font-size 13px
    color rgba(255, 255, 255, 0.72)

  &__price
    font-size 36px
    font-weight 600
    letter-spacing -0.4px
    line-height 1.15
    margin-bottom 10px
    font-variant-numeric tabular-nums
    font-synthesis none
    -webkit-font-smoothing antialiased
    color #fff

  &__change
    display flex
    align-items center
    gap 10px
    margin-bottom 16px

    .change-pill
      display inline-flex
      align-items center
      gap 3px
      padding 5px 12px
      border-radius 999px
      font-size 13px
      font-weight 600
      font-variant-numeric tabular-nums
      border 1px solid transparent

      &.up
        background rgba(7, 193, 96, 0.2)
        color #7dffb3
        border-color rgba(7, 193, 96, 0.28)

      &.down
        background rgba(255, 77, 79, 0.2)
        color #ffb4b4
        border-color rgba(255, 77, 79, 0.28)

    .change-hint
      font-size 12px
      font-weight 500
      color rgba(255, 255, 255, 0.55)

  &__stats
    display grid
    grid-template-columns 1fr 1fr
    gap 10px
    padding-top 14px
    border-top 1px solid rgba(255, 255, 255, 0.12)

  .hero-metric
    min-width 0
    display flex
    align-items center
    gap 10px
    padding 12px 12px
    border-radius 14px
    background rgba(0, 0, 0, 0.22)
    border 1px solid rgba(255, 255, 255, 0.14)

    &__icon
      flex-shrink 0
      width 36px
      height 36px
      border-radius 50%
      display flex
      align-items center
      justify-content center
      background #1677ff
      color #fff
      font-size 18px
      box-shadow 0 4px 10px rgba(0, 0, 0, 0.18)

    &__body
      min-width 0
      flex 1
      display flex
      flex-direction column
      gap 3px

    &__label
      display block
      font-size 12px
      font-weight 500
      color rgba(255, 255, 255, 0.65)
      line-height 1.2
      white-space nowrap
      overflow hidden
      text-overflow ellipsis

    &__value
      display block
      font-size 16px
      font-weight 600
      color #fff
      line-height 1.25
      letter-spacing 0.01em
      font-variant-numeric tabular-nums
      font-synthesis none
      -webkit-font-smoothing antialiased
      white-space nowrap
      overflow hidden
      text-overflow ellipsis

      em
        margin-left 4px
        font-style normal
        font-size 11px
        font-weight 500
        color rgba(255, 255, 255, 0.5)

    &--burn &__icon
      background rgba(255, 255, 255, 0.16)
      color #fff
      box-shadow none

.price-chart
  .chart-body
    position relative

  .chart-placeholder
    height 260px
    padding 12px 4px
    display flex
    align-items center

    &--overlay
      position absolute
      inset 0
      z-index 2
      background #fff
      padding 12px 8px

  .chart-container
    height 260px

  .market-blur-overlay
    position absolute
    inset 0
    display flex
    align-items center
    justify-content center
    background rgba(255, 255, 255, 0.4)
    border-radius 12px
    pointer-events none
    z-index 2

    span
      padding 8px 16px
      background rgba(31, 41, 55, 0.72)
      color #fff
      border-radius 20px
      font-size 13px

  .chart-title
    display flex
    justify-content space-between
    align-items flex-start
    margin-bottom 12px

    .title-content
      h3
        margin 0 0 2px
        font-size 18px
        font-weight 800
        color #1f2937

      .sub-title
        font-size 12px
        color #9ca3af

    .zoom-controls
      display flex
      gap 6px
      padding-top 4px

      .zoom-btn
        width 32px
        height 32px
        border-radius 10px
        background rgba(22, 119, 255, 0.08)
        color #1677ff
        border 1px solid rgba(22, 119, 255, 0.16)
        display flex
        align-items center
        justify-content center
        cursor pointer
        -webkit-tap-highlight-color transparent

        &:disabled
          opacity 0.4
          cursor not-allowed

        &:active:not(:disabled)
          background rgba(22, 119, 255, 0.16)

.price-table-section
  .table-title
    display flex
    justify-content space-between
    align-items flex-start
    margin-bottom 12px

    .title-content
      display flex
      flex-direction column

    .table-title-text
      font-size 18px
      font-weight 800
      color #1f2937

    .table-sub
      font-size 12px
      font-weight 500
      color #9ca3af
      padding-top 6px

  .price-table-wrapper
    overflow-x auto
    max-height 320px
    overflow-y auto
    -webkit-overflow-scrolling touch
    border-radius 14px
    border 1px solid rgba(79, 172, 254, 0.12)

    &::-webkit-scrollbar
      width 3px
      height 3px

    &::-webkit-scrollbar-thumb
      background rgba(79, 172, 254, 0.35)
      border-radius 3px

  .price-table
    width 100%
    border-collapse collapse
    font-size 12px

    th, td
      padding 10px 6px
      text-align center
      border-bottom 1px solid #f3f4f6

    thead
      position sticky
      top 0
      z-index 1

    th
      color #7b8794
      font-weight 600
      font-size 11px
      background #f8fbff
      white-space nowrap

    tbody tr:last-child td
      border-bottom none

    tbody tr.row--latest
      background rgba(79, 172, 254, 0.08)

      td
        font-weight 600

    .col-date
      text-align left
      padding-left 10px
      white-space nowrap

    .col-price, .col-change, .col-vol
      font-variant-numeric tabular-nums

    .col-vol
      color #7b8794

    .latest-tag
      display inline-block
      margin-right 4px
      padding 1px 5px
      font-size 9px
      font-weight 700
      color #1677ff
      background rgba(22, 119, 255, 0.12)
      border-radius 4px
      vertical-align middle

    .price-cell
      color #1677ff
      font-weight 700

    .change-cell.up
      color #07c160
      font-weight 600

    .change-cell.down
      color #ff4d4f
      font-weight 600

    .empty-cell
      color #9ca3af
      padding 32px 16px
      font-size 13px

@keyframes live-pulse
  0%, 100%
    opacity 1
    transform scale(1)
  50%
    opacity 0.5
    transform scale(0.85)

.my-mint-section
  display flex
  flex-direction column
  gap 14px

  .section-title
    font-size 22px
    font-weight 700

.my-mint-head
  .my-mint-summary
    display flex
    gap 8px
    margin 14px 0 12px

    &__item
      flex 1
      min-width 0
      padding 12px 10px
      background linear-gradient(180deg, #ffffff 0%, #f7fbff 100%)
      border 1px solid rgba(79, 172, 254, 0.12)
      border-radius 14px
      text-align center

      .label
        display block
        font-size 13px
        font-weight 500
        color #7b8794
        margin-bottom 6px

      .value
        display block
        font-size 18px
        font-weight 600
        color #1f2937
        line-height 1.2
        letter-spacing 0.01em
        font-variant-numeric tabular-nums
        font-synthesis none
        -webkit-font-smoothing antialiased

        em
          font-style normal
          font-size 12px
          font-weight 500
          color #9ca3af
          margin-left 2px

        &--soha
          color #4facfe

  .my-mint-ledger
    padding 12px 14px
    background linear-gradient(135deg, rgba(79, 172, 254, 0.08), rgba(0, 242, 254, 0.06))
    border 1px solid rgba(79, 172, 254, 0.14)
    border-radius 14px

    &__label
      display block
      font-size 13px
      font-weight 500
      color #7b8794
      margin-bottom 4px

    &__value
      display block
      font-size 20px
      font-weight 600
      color #4facfe
      letter-spacing 0.01em
      font-variant-numeric tabular-nums
      font-synthesis none
      -webkit-font-smoothing antialiased

.my-mint-loading
  padding 48px 0
  text-align center
  color #7b8794

.my-mint-empty
  text-align center

  &__btn
    display block
    width calc(100% - 32px)
    max-width 240px
    margin 8px auto 0
    padding 11px 16px
    border none
    border-radius 999px
    font-size 14px
    font-weight 700
    color #fff
    background #1677ff
    box-shadow 0 6px 16px rgba(22, 119, 255, 0.28)
    cursor pointer
    -webkit-tap-highlight-color transparent

    &:active
      opacity 0.88

.my-mint-list
  display flex
  flex-direction column
  gap 12px

.my-mint-card
  &__head
    display flex
    align-items flex-start
    justify-content space-between
    gap 10px
    margin-bottom 12px

  &__title-wrap
    flex 1
    min-width 0

  &__index
    display inline-block
    margin-right 6px
    padding 2px 6px
    font-size 11px
    font-weight 600
    color #4facfe
    background rgba(79, 172, 254, 0.12)
    border-radius 6px
    vertical-align middle

  &__title
    display inline
    margin 0
    font-size 16px
    font-weight 600
    color #1f2937
    vertical-align middle
    word-break break-all

  &__status
    flex-shrink 0
    padding 3px 8px
    font-size 12px
    font-weight 600
    color #07c160
    background rgba(7, 193, 96, 0.1)
    border-radius 999px

  &__metrics
    display grid
    grid-template-columns repeat(3, 1fr)
    gap 8px
    margin-bottom 12px

  &__detail-btn
    display flex
    align-items center
    justify-content center
    gap 6px
    width 100%
    padding 10px
    border none
    border-radius 12px
    font-size 14px
    font-weight 600
    color #4facfe
    background rgba(79, 172, 254, 0.1)
    cursor pointer
    -webkit-tap-highlight-color transparent

    &:active
      background rgba(79, 172, 254, 0.16)

.my-mint-metric
  min-width 0
  padding 10px 8px
  background linear-gradient(180deg, #ffffff 0%, #f7fbff 100%)
  border 1px solid rgba(79, 172, 254, 0.1)
  border-radius 12px
  text-align center

  &__label
    display block
    font-size 12px
    font-weight 500
    color #7b8794
    margin-bottom 4px
    white-space nowrap
    overflow hidden
    text-overflow ellipsis

  &__value
    display block
    font-size 15px
    font-weight 600
    color #1f2937
    letter-spacing 0.01em
    font-variant-numeric tabular-nums
    font-synthesis none
    -webkit-font-smoothing antialiased
    word-break break-all

    &--coin
      color #4facfe

    &--time
      font-size 12px
      font-weight 500
      color #6b7280

.mint-section,
.swap-section
  .market-readonly-tip
    margin 0 0 12px
    padding 10px 12px
    background #fff7e6
    color #d48806
    border 1px solid #ffe7ba
    border-radius 12px
    font-size 13px
    line-height 1.5

.mint-section
  display flex
  flex-direction column
  gap 14px

.mint-hero
  &__content
    position relative
    z-index 1

  &__title
    margin 0 0 8px
    font-size 22px
    font-weight 800
    color #1f2937

  &__desc
    margin 0 0 14px
    font-size 13px
    line-height 1.5
    color #7b8794

  &__stats
    display flex
    gap 10px

.mint-stat-pill
  flex 1
  display flex
  align-items center
  gap 10px
  padding 12px 14px
  border-radius 14px
  background linear-gradient(135deg, rgba(79, 172, 254, 0.1), rgba(0, 242, 254, 0.08))
  border 1px solid rgba(79, 172, 254, 0.16)

  &__icon
    color #4facfe
    font-size 20px
    flex-shrink 0

  &__text
    display flex
    flex-direction column
    min-width 0

    .label
      font-size 11px
      color #7b8794

    .value
      font-size 15px
      font-weight 800
      color #1f2937
      white-space nowrap
      overflow hidden
      text-overflow ellipsis

.mint-empty-panel
  text-align center
  padding 40px 24px !important
  border-style dashed !important

  &__icon
    width 72px
    height 72px
    margin 0 auto 14px
    border-radius 50%
    display flex
    align-items center
    justify-content center
    background linear-gradient(135deg, rgba(79, 172, 254, 0.15), rgba(0, 242, 254, 0.12))
    color #4facfe

  &__title
    margin 0 0 6px
    font-size 16px
    font-weight 700
    color #1f2937

  &__sub
    margin 0
    font-size 13px
    color #9ca3af

.mint-rig-list
  display flex
  flex-direction column
  gap 14px

.mint-rig
  &__accent
    position absolute
    top 0
    left 0
    right 0
    height 3px
    background #1677ff

  &--tone-1 &__accent
    background #1677ff

  &--tone-2 &__accent
    background #1677ff

  &__top
    display flex
    gap 14px
    margin-bottom 14px

  &__visual
    flex-shrink 0
    width 56px
    height 64px
    display flex
    align-items flex-end
    justify-content center

  &__meta
    flex 1
    min-width 0
    padding-top 4px

  &__index
    display block
    font-size 10px
    font-weight 700
    letter-spacing 0.12em
    color #9ca3af
    margin-bottom 4px

  &__name
    margin 0 0 6px
    font-size 17px
    font-weight 800
    color #1f2937
    line-height 1.3

  &__tag
    display inline-block
    font-size 11px
    padding 2px 8px
    border-radius 999px
    color #4facfe
    background rgba(79, 172, 254, 0.12)
    font-weight 600

  &__metrics
    display grid
    grid-template-columns 1fr 1fr
    gap 10px
    margin-bottom 12px

.mint-metric
  padding 12px
  border-radius 14px
  background linear-gradient(180deg, #ffffff 0%, #f7fbff 100%)
  border 1px solid rgba(79, 172, 254, 0.12)

  &__label
    display flex
    align-items center
    gap 4px
    font-size 11px
    color #7b8794
    margin-bottom 6px

  &__value
    display block
    font-size 16px
    font-weight 800
    color #1f2937

  &__value--power
    color #d32f2f

.mint-activate-btn
  width 100%
  display flex
  align-items center
  justify-content center
  gap 8px
  padding 13px 16px
  border none
  border-radius 14px
  font-size 15px
  font-weight 700
  color #fff
  background #1677ff
  box-shadow 0 6px 16px rgba(22, 119, 255, 0.3)
  cursor pointer
  transition transform 0.15s ease, box-shadow 0.15s ease, opacity 0.15s ease

  &:active:not(:disabled)
    transform scale(0.98)

.rig-tower
  position relative
  width 44px

  &__fan
    position absolute
    top 0
    width 14px
    height 14px
    border-radius 50%
    border 2px solid rgba(71, 85, 105, 0.45)
    background rgba(100, 116, 139, 0.14)

    &::after
      content ''
      position absolute
      inset 3px
      border-radius 50%
      border 1px dashed rgba(100, 116, 139, 0.5)

    &--l
      left 4px
      animation rig-spin 3s linear infinite

    &--r
      right 4px
      animation rig-spin 3s linear infinite reverse

  &__body
    margin 16px auto 0
    width 36px
    height 28px
    border-radius 8px
    background linear-gradient(180deg, #334155 0%, #475569 100%)
    border 1px solid rgba(148, 163, 184, 0.35)
    display flex
    flex-direction column
    align-items center
    justify-content center
    gap 3px
    padding 4px 0
    box-shadow 0 6px 14px rgba(51, 65, 85, 0.22)

  &__led
    width 20px
    height 3px
    border-radius 2px
    background #94a3b8
    box-shadow 0 0 5px rgba(148, 163, 184, 0.45)
    animation mint-pulse 1.6s ease-in-out infinite

    &--delay
      animation-delay 0.8s
      width 14px
      opacity 0.7

  &__slot
    width 24px
    height 4px
    border-radius 2px
    background rgba(15, 23, 42, 0.22)

  &__base
    margin 4px auto 0
    width 40px
    height 6px
    border-radius 3px
    background linear-gradient(90deg, #64748b, #475569)

@keyframes mint-pulse
  0%, 100%
    opacity 1
  50%
    opacity 0.45

@keyframes rig-spin
  from
    transform rotate(0deg)
  to
    transform rotate(360deg)

.swap-section
  .market-readonly
    opacity 0.72
    pointer-events none

  .quick-btn.disabled
    opacity 0.45
    pointer-events none

.swap-container
  .swap-header
    margin-bottom 16px

    &__desc
      margin 0
      font-size 13px
      color #7b8794
      line-height 1.45

  .token-input-container
    background linear-gradient(180deg, #ffffff 0%, #f7fbff 100%)
    border 1px solid rgba(79, 172, 254, 0.12)
    border-radius 16px
    padding 14px
    margin-bottom 12px
    overflow hidden
    box-sizing border-box

  .token-input-header
    display flex
    justify-content space-between
    align-items center
    gap 10px
    margin-bottom 12px
    font-size 14px
    color #4b5563

    > span:first-child
      flex-shrink 0

    .balance
      min-width 0
      max-width 70%
      color #4facfe
      font-size 12px
      font-weight 600
      text-align right
      overflow hidden
      text-overflow ellipsis
      white-space nowrap
      font-variant-numeric tabular-nums

  .token-input
    display flex
    align-items center
    gap 10px
    min-width 0
    overflow hidden

  .token-selector-fixed
    padding 10px 12px
    background rgba(79, 172, 254, 0.1)
    border-radius 12px
    flex-shrink 0
    max-width 46%

  .token-info
    display flex
    align-items center
    gap 8px
    min-width 0

  .token-symbol
    font-weight 700
    color #1f2937
    white-space nowrap

  .token-icon-placeholder
    width 28px
    height 28px
    border-radius 50%
    display flex
    align-items center
    justify-content center
    font-size 11px
    font-weight bold
    color #fff
    flex-shrink 0

    &.soha
      background linear-gradient(135deg, #4facfe, #00c6fb)

    &.power
      background linear-gradient(135deg, #f093fb, #f5576c)

  .amount-input
    flex 1
    min-width 0
    width 0
    border none
    outline none
    background transparent
    font-size 20px
    font-weight 700
    text-align right
    color #1f2937
    padding 0 2px 0 0
    box-sizing border-box
    font-variant-numeric tabular-nums
    -moz-appearance textfield

    &::-webkit-outer-spin-button,
    &::-webkit-inner-spin-button
      -webkit-appearance none
      margin 0

  .token-input-footer
    display flex
    justify-content space-between
    align-items center
    gap 8px
    margin-top 10px
    font-size 12px
    color #9ca3af
    min-width 0

    > span
      min-width 0
      overflow hidden
      text-overflow ellipsis
      white-space nowrap

  .quick-amounts
    display flex
    gap 8px

    .quick-btn
      padding 4px 10px
      background rgba(79, 172, 254, 0.12)
      color #4facfe
      border-radius 8px
      cursor pointer
      font-weight 600

  .swap-arrow-container
    display flex
    justify-content center
    margin 4px 0 8px

    .swap-arrow
      width 40px
      height 40px
      background #fff
      border-radius 50%
      display flex
      align-items center
      justify-content center
      border 1px solid rgba(79, 172, 254, 0.2)
      color #4facfe
      box-shadow 0 4px 12px rgba(79, 172, 254, 0.12)

  .swap-info
    margin-top 12px
    padding 12px 14px
    background linear-gradient(180deg, #ffffff 0%, #f7fbff 100%)
    border 1px solid rgba(79, 172, 254, 0.12)
    border-radius 14px
    font-size 13px

    .info-header
      font-weight 700
      margin-bottom 8px
      color #1f2937

    .info-row
      display flex
      justify-content space-between
      padding 4px 0
      color #4b5563

      .limit-text
        color #ff976a
        font-weight 600

  .swap-button-container
    margin-top 18px

    .swap-button
      height 48px
      border-radius 24px
      font-weight 700
      background #1677ff
      border none

      &.disabled-button
        background #eef2f7
        color #9ca3af

      &.error-button
        background #ffebee
        color #ff4d4f

.swap-confirm-popup
  padding 20px
  height 100%
  display flex
  flex-direction column

  .popup-header h3
    margin 0 0 16px
    font-size 18px
    font-weight 800
    color #1f2937

  .swap-confirm-content
    flex 1

  .detail-row
    display flex
    justify-content space-between
    padding 10px 0
    border-bottom 1px solid #f0f0f0
    font-size 14px

    .highlight
      color #4facfe
      font-weight 700

  .risk-notice
    margin-top 16px
    font-size 13px
    color #9ca3af
    text-align center

  .confirm-buttons
    display flex
    gap 12px
    margin-top 16px

    .cancel-btn, .confirm-btn
      flex 1
      height 44px
      border-radius 22px


.mint-hero
  background linear-gradient(145deg, #1e3a5f 0%, #2b4a7a 45%, #3b6bb5 100%) !important
  border none !important
  box-shadow 0 10px 28px rgba(30, 58, 95, 0.28) !important
  color #fff
  overflow hidden

  &__badge
    display inline-flex
    align-items center
    gap 6px
    padding 4px 10px
    margin-bottom 12px
    border-radius 999px
    font-size 12px
    font-weight 700
    color #fff
    background rgba(0, 0, 0, 0.22)

    .live-dot
      width 6px
      height 6px
      border-radius 50%
      background #07c160
      box-shadow 0 0 8px rgba(7, 193, 96, 0.7)
      animation live-pulse 1.8s ease-in-out infinite

  &__title
    color #fff !important

  &__desc
    color rgba(255, 255, 255, 0.78) !important

.mint-stat-pill--dark
  background rgba(0, 0, 0, 0.22) !important
  border 1px solid rgba(255, 255, 255, 0.14) !important

  .mint-stat-pill__icon
    width 36px
    height 36px
    border-radius 50%
    display flex
    align-items center
    justify-content center
    background #1677ff
    color #fff !important

  .label
    color rgba(255, 255, 255, 0.65) !important

  .value
    color #fff !important

.mint-activate-btn
  background #1677ff !important
  box-shadow 0 6px 16px rgba(22, 119, 255, 0.3) !important

.swap-confirm-dialog
  :deep(.van-dialog__header)
    padding-top 20px
    font-weight 800
    font-size 18px

.swap-confirm-dialog__body
  padding 8px 20px 16px
  .detail-row
    display flex
    justify-content space-between
    align-items center
    gap 12px
    padding 12px 0
    border-bottom 1px solid #f0f0f0
    font-size 15px
    .label
      color #6b7280
      flex-shrink 0
    .value
      color #1f2937
      font-weight 600
      text-align right
      word-break break-all
    .highlight
      color #1989fa
      font-weight 700
    .limit-text
      color #ed6a0c
      font-weight 600
  .risk-notice
    margin 14px 0 0
    font-size 13px
    color #7f8c8d
    text-align center
    line-height 1.5

.market-page
  &:before, &:after
    display none

.my-mint-section
  display flex
  flex-direction column
  gap 14px

</style>
