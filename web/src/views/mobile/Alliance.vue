<template>
  <div class="alliance custom-scroll">
    <div class="alliance__layout" :class="{ 'alliance__layout--locked': !allianceEnabled }">
      <!-- 会员身份卡 -->
      <section class="member-card" :class="'member-card--' + myTier.key">
        <div class="member-card__shine" aria-hidden="true" />
        <div class="member-card__orb member-card__orb--a" aria-hidden="true" />
        <div class="member-card__orb member-card__orb--b" aria-hidden="true" />

        <div class="member-card__top">
          <div class="member-card__brand">
            <span class="member-card__flag">CN</span>
            <div>
              <h1>{{ t('alliance.title') }}</h1>
              <p>{{ t('alliance.subtitle') }}</p>
            </div>
          </div>
          <div class="member-card__badge">
            <van-icon name="medal-o" size="18" />
            <span>{{ myTier.label }}</span>
          </div>
        </div>

        <div class="member-card__power">
          <span class="member-card__power-label">{{ t('alliance.powerInvested') }}</span>
          <div class="member-card__power-row">
            <strong>{{ formatNum(myPower) }}</strong>
            <em>{{ t('alliance.powerUnit') }}</em>
          </div>
        </div>

        <div class="member-card__progress">
          <div class="member-card__progress-head">
            <span>{{ t('alliance.currentTier') }} · {{ myTier.label }}</span>
            <span v-if="nextTier">{{ t('alliance.toNext', { name: nextTier.label }) }}</span>
            <span v-else>{{ t('alliance.topTier') }}</span>
          </div>
          <div class="member-card__bar">
            <i :style="{ width: progressPct + '%' }" />
          </div>
          <p v-if="nextTier" class="member-card__gap">
            {{ t('alliance.needMore', { n: formatNum(Math.max(0, nextTier.min - myPower)) }) }}
          </p>
        </div>
      </section>

      <!-- 等级阶梯 -->
      <section class="tier-rail glass">
        <div class="section-head">
          <span class="section-kicker">MEMBERSHIP</span>
          <h2>{{ t('alliance.tierLadder') }}</h2>
        </div>
        <div class="tier-rail__scroll">
          <button
            v-for="(tier, idx) in tiers"
            :key="tier.key"
            type="button"
            class="tier-chip"
            :class="[
              'tier-chip--' + tier.key,
              { 'is-active': tier.key === myTier.key, 'is-passed': isTierPassed(tier) },
            ]"
          >
            <span class="tier-chip__idx">{{ idx + 1 }}</span>
            <span class="tier-chip__name">{{ tier.label }}</span>
            <span class="tier-chip__range">{{ tier.range }}</span>
          </button>
        </div>
      </section>

      <!-- 分区切换 -->
      <div class="zone-tabs glass" role="tablist" :data-active="activeTab">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          class="zone-tabs__item"
          :class="{ active: activeTab === tab.key }"
          role="tab"
          :aria-selected="activeTab === tab.key"
          @click="activeTab = tab.key"
        >
          <van-icon :name="tab.icon" size="16" />
          <span>{{ tab.label }}</span>
        </button>
      </div>

      <!-- 上架规则条 -->
      <section class="rule-bar glass">
        <div class="rule-bar__icon">
          <van-icon name="shield-o" size="20" color="#fff" />
        </div>
        <div class="rule-bar__body">
          <strong>{{ t('alliance.listRuleTitle') }}</strong>
          <p>{{ t('alliance.listRuleShort') }}</p>
          <span class="rule-bar__fee">{{ t('alliance.feeTip') }}</span>
        </div>
        <button
          v-if="canList"
          type="button"
          class="rule-bar__cta"
          @click="openListDialog"
        >
          {{ t('alliance.listLink') }}
          <small>{{ myListingsCount }}/3</small>
        </button>
        <div v-else class="rule-bar__lock">
          <van-icon name="lock" size="14" />
          <span>{{ t('alliance.listNeedStar3') }}</span>
        </div>
      </section>

      <!-- 兑换区 -->
      <section v-if="activeTab === 'exchange'" class="zone-panel">
        <article v-for="(item, i) in exchangeList" :key="item.id" class="deal-card glass" :style="{ '--delay': i * 40 + 'ms' }">
          <div class="deal-card__accent deal-card__accent--blue" />
          <div class="deal-card__head">
            <div class="deal-card__avatar deal-card__avatar--ex">
              <van-icon name="exchange" size="22" color="#fff" />
            </div>
            <div class="deal-card__title">
              <h3>{{ item.name }}</h3>
              <a class="deal-card__phone" :href="'tel:' + item.phone">
                <van-icon name="phone-o" size="13" />
                {{ item.phone }}
              </a>
            </div>
            <button type="button" class="deal-card__link" @click="openLinkDialog(item)">
              <van-icon name="link-o" size="14" />
              {{ t('alliance.openLink') }}
            </button>
          </div>

          <div class="deal-card__metrics">
            <div class="metric">
              <span>{{ t('alliance.exchangeRate') }}</span>
              <strong>1万 → 700</strong>
              <em>{{ t('alliance.pointsUnit') }}</em>
            </div>
            <div class="metric">
              <span>{{ t('alliance.serviceFee') }}</span>
              <strong>1%</strong>
              <em>{{ t('alliance.perDeal') }}</em>
            </div>
          </div>

          <button type="button" class="deal-card__cta" @click="onExchange(item)">
            {{ t('alliance.doExchange') }}
          </button>
        </article>
        <p v-if="!exchangeList.length" class="alliance-empty">{{ t('alliance.empty') }}</p>
      </section>

      <!-- 让利区 -->
      <section v-else-if="activeTab === 'profit'" class="zone-panel">
        <article v-for="(item, i) in profitList" :key="item.id" class="deal-card glass" :style="{ '--delay': i * 40 + 'ms' }">
          <div class="deal-card__accent deal-card__accent--gold" />
          <div class="deal-card__head">
            <div class="deal-card__avatar deal-card__avatar--pf">
              <van-icon name="gift-o" size="22" color="#fff" />
            </div>
            <div class="deal-card__title">
              <h3>{{ item.name }}</h3>
              <a class="deal-card__phone" :href="'tel:' + item.phone">
                <van-icon name="phone-o" size="13" />
                {{ item.phone }}
              </a>
            </div>
            <button type="button" class="deal-card__link" @click="openLinkDialog(item)">
              <van-icon name="link-o" size="14" />
              {{ t('alliance.openLink') }}
            </button>
          </div>

          <div class="deal-card__metrics deal-card__metrics--3">
            <div class="metric">
              <span>{{ t('alliance.exchangeRate') }}</span>
              <strong>1万 → 700</strong>
            </div>
            <div class="metric metric--hot">
              <span>{{ t('alliance.profitShare') }}</span>
              <strong>2000×5</strong>
            </div>
            <div class="metric">
              <span>{{ t('alliance.merchantReceive') }}</span>
              <strong>8000</strong>
            </div>
          </div>

          <div class="deal-card__tip">
            <van-icon name="info-o" size="14" />
            {{ t('alliance.feeTip') }}
          </div>

          <button type="button" class="deal-card__cta deal-card__cta--gold" @click="onExchange(item)">
            {{ t('alliance.doExchange') }}
          </button>
        </article>
        <p v-if="!profitList.length" class="alliance-empty">{{ t('alliance.empty') }}</p>
      </section>

      <!-- 私域让利 -->
      <section v-else class="zone-panel">
        <article class="activate-card glass">
          <div class="activate-card__ribbon">{{ t('alliance.activateRibbon') }}</div>
          <h3>{{ t('alliance.activateTitle') }}</h3>
          <p class="activate-card__desc">{{ t('alliance.activateDesc') }}</p>

          <div class="activate-card__flow">
            <div class="flow-node">
              <span>{{ t('alliance.buyerGet') }}</span>
              <strong>5万</strong>
              <em>{{ t('alliance.pendingRelease') }}</em>
            </div>
            <div class="flow-arrow" aria-hidden="true">⇄</div>
            <div class="flow-node flow-node--seller">
              <span>{{ t('alliance.sellerGet') }}</span>
              <strong>1万</strong>
              <em>{{ t('alliance.pendingRelease') }}</em>
            </div>
          </div>

          <p class="activate-card__note">{{ t('alliance.activateNote') }}</p>
          <button type="button" class="deal-card__cta" @click="onActivate">
            {{ t('alliance.activateBtn') }}
          </button>
        </article>

        <article
          v-for="(item, i) in privateList"
          :key="item.id"
          class="product-card glass"
          :style="{ '--delay': (i + 1) * 40 + 'ms' }"
        >
          <div class="product-card__visual">
            <div class="product-card__icon">
              <van-icon name="shopping-cart-o" size="28" color="#fff" />
            </div>
            <div class="product-card__meta">
              <h3>{{ item.product }}</h3>
              <p class="product-card__price">
                <small>{{ t('alliance.marketPriceLabel') }}</small>
                <strong>¥{{ item.price }}</strong>
              </p>
            </div>
            <span class="product-card__gift">
              {{ t('alliance.giftPowerValue', { n: '1万', x: 5 }) }}
            </span>
          </div>

          <div class="product-card__form">
            <van-field v-model="item._address" :label="t('alliance.address')" :placeholder="t('alliance.addressPh')" />
            <van-field v-model="item._phone" type="tel" :label="t('alliance.phone')" :placeholder="t('alliance.phonePh')" />
            <van-field
              v-model="item._transferCode"
              :label="t('alliance.transferCode')"
              :placeholder="t('alliance.transferCodePh')"
            />
          </div>

          <div class="product-card__foot">
            <span class="product-card__fee">{{ t('alliance.feeTip') }}</span>
            <button type="button" class="deal-card__cta deal-card__cta--compact" @click="onSubscribe(item)">
              {{ t('alliance.subscribe') }}
            </button>
          </div>
        </article>
        <p v-if="!privateList.length" class="alliance-empty">{{ t('alliance.empty') }}</p>
      </section>
    </div>

    <!-- 暂未开放：遮罩挡住交互，背景保留虚化预览 -->
    <div v-if="!allianceEnabled" class="alliance__closed-mask" role="dialog" aria-modal="true">
      <div class="alliance__closed-card">
        <van-icon name="lock" size="28" color="#1a6cb5" />
        <p class="alliance__closed-title">{{ t('alliance.notOpen') }}</p>
        <p class="alliance__closed-desc">{{ t('alliance.notOpenHint') }}</p>
      </div>
    </div>

    <template v-if="allianceEnabled">
      <van-dialog
        v-model:show="showLinkDialog"
        :title="t('alliance.linkDialogTitle')"
        show-cancel-button
        :confirm-button-text="t('alliance.copyLink')"
        :cancel-button-text="t('alliance.close')"
        @confirm="copyLink"
      >
        <div class="dlg">
          <p class="dlg__name">{{ linkTarget?.name || linkTarget?.product }}</p>
          <p class="dlg__link">{{ linkTarget?.url }}</p>
        </div>
      </van-dialog>

      <van-dialog
        v-model:show="showListDialog"
        :title="t('alliance.listDialogTitle')"
        show-cancel-button
        :confirm-button-text="t('alliance.confirmList')"
        :cancel-button-text="t('alliance.close')"
        :before-close="beforeCloseList"
      >
        <div class="dlg dlg--form">
          <van-field v-model="listForm.name" :label="t('alliance.fieldName')" :placeholder="t('alliance.fieldNamePh')" />
          <van-field v-model="listForm.url" :label="t('alliance.fieldUrl')" :placeholder="t('alliance.fieldUrlPh')" />
          <van-field v-model="listForm.phone" type="tel" :label="t('alliance.fieldPhone')" :placeholder="t('alliance.fieldPhonePh')" />
          <div class="dlg__zones">
            <span>{{ t('alliance.fieldZone') }}</span>
            <div class="dlg__zone-row">
              <button
                v-for="z in listZones"
                :key="z.key"
                type="button"
                class="dlg__zone"
                :class="{ 'is-active': listForm.zone === z.key }"
                @click="listForm.zone = z.key"
              >
                {{ z.label }}
              </button>
            </div>
          </div>
        </div>
      </van-dialog>
    </template>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { showToast } from 'vant'

defineOptions({ name: 'Alliance' })

const { t } = useI18n()

/** 联盟功能开关：false 时展示虚化预览 +「暂未开放」，不可操作 */
const allianceEnabled = false

/** 演示：当前用户算力（后续接接口） */
const myPower = ref(120000)

const tiers = computed(() => [
  { key: 'normal', label: t('alliance.tierNormal'), min: 1000, max: 4999, range: '1000～4999' },
  { key: 'vip', label: t('alliance.tierVip'), min: 5000, max: 9999, range: '5000～9999' },
  { key: 'star1', label: t('alliance.tierStar1'), min: 10000, max: 49999, range: '1万～4.9万' },
  { key: 'star2', label: t('alliance.tierStar2'), min: 50000, max: 99999, range: '5万～9.9万' },
  { key: 'star3', label: t('alliance.tierStar3'), min: 100000, max: Infinity, range: t('alliance.tierStar3Range') },
])

const myTier = computed(() => {
  const p = Number(myPower.value) || 0
  const list = tiers.value
  for (let i = list.length - 1; i >= 0; i -= 1) {
    if (p >= list[i].min) return list[i]
  }
  return { key: 'none', label: t('alliance.tierNone'), min: 0, max: 999, range: '-' }
})

const nextTier = computed(() => {
  const list = tiers.value
  const idx = list.findIndex((x) => x.key === myTier.value.key)
  if (idx < 0 || idx >= list.length - 1) return null
  return list[idx + 1]
})

const progressPct = computed(() => {
  const p = Number(myPower.value) || 0
  const cur = myTier.value
  const nxt = nextTier.value
  if (!nxt) return 100
  const span = Math.max(1, nxt.min - cur.min)
  const done = Math.min(span, Math.max(0, p - cur.min))
  return Math.round((done / span) * 100)
})

function isTierPassed(tier) {
  return (Number(myPower.value) || 0) >= tier.min
}

const canList = computed(() => myTier.value.key === 'star3')

const tabs = computed(() => [
  { key: 'exchange', label: t('alliance.tabExchange'), icon: 'exchange' },
  { key: 'profit', label: t('alliance.tabProfit'), icon: 'gift-o' },
  { key: 'private', label: t('alliance.tabPrivate'), icon: 'shop-o' },
])

const activeTab = ref('exchange')

const exchangeList = ref([
  {
    id: 'ex1',
    name: '供应链兑换点',
    url: 'https://example.com/exchange/1',
    phone: '13564347668',
  },
  {
    id: 'ex2',
    name: '联盟积分站',
    url: 'https://example.com/exchange/2',
    phone: '13800138000',
  },
])

const profitList = ref([
  {
    id: 'pf1',
    name: '让利兑换商A',
    url: 'https://example.com/profit/1',
    phone: '13564347668',
  },
])

const privateList = ref([
  {
    id: 'pv1',
    product: '茶叶',
    price: 1000,
    url: 'https://example.com/private/tea',
    _address: '',
    _phone: '',
    _transferCode: '',
  },
])

const myListingsCount = ref(1)

const showLinkDialog = ref(false)
const linkTarget = ref(null)

function openLinkDialog(item) {
  linkTarget.value = item
  showLinkDialog.value = true
}

async function copyLink() {
  const url = linkTarget.value?.url
  if (!url) return
  try {
    await navigator.clipboard.writeText(url)
    showToast(t('alliance.copied'))
  } catch {
    showToast(url)
  }
}

function onExchange() {
  showToast(t('alliance.uiOnly'))
}

function onActivate() {
  showToast(t('alliance.uiOnly'))
}

function onSubscribe(item) {
  if (!item._address?.trim() || !item._phone?.trim()) {
    showToast(t('alliance.needAddressPhone'))
    return
  }
  if (!item._transferCode?.trim()) {
    showToast(t('alliance.needTransferCode'))
    return
  }
  showToast(t('alliance.uiOnly'))
}

const showListDialog = ref(false)
const listForm = reactive({
  name: '',
  url: '',
  phone: '',
  zone: 'exchange',
})

const listZones = computed(() => [
  { key: 'exchange', label: t('alliance.tabExchange') },
  { key: 'profit', label: t('alliance.tabProfit') },
  { key: 'private', label: t('alliance.tabPrivate') },
])

function openListDialog() {
  if (!canList.value) {
    showToast(t('alliance.listLocked'))
    return
  }
  if (myListingsCount.value >= 3) {
    showToast(t('alliance.listLimit'))
    return
  }
  listForm.name = ''
  listForm.url = ''
  listForm.phone = ''
  listForm.zone = activeTab.value === 'private' ? 'private' : activeTab.value
  showListDialog.value = true
}

function beforeCloseList(action) {
  if (action !== 'confirm') return true
  if (!listForm.name.trim() || !listForm.url.trim() || !listForm.phone.trim()) {
    showToast(t('alliance.listFormRequired'))
    return false
  }
  if (myListingsCount.value >= 3) {
    showToast(t('alliance.listLimit'))
    return false
  }
  myListingsCount.value += 1
  const row = {
    id: 'mine-' + Date.now(),
    name: listForm.name.trim(),
    product: listForm.name.trim(),
    price: 1000,
    url: listForm.url.trim(),
    phone: listForm.phone.trim(),
    _address: '',
    _phone: '',
    _transferCode: '',
  }
  if (listForm.zone === 'profit') profitList.value.unshift(row)
  else if (listForm.zone === 'private') privateList.value.unshift(row)
  else exchangeList.value.unshift(row)
  activeTab.value = listForm.zone
  showToast(t('alliance.listSuccess'))
  return true
}

function formatNum(n) {
  const v = Number(n) || 0
  return v.toLocaleString('zh-CN')
}
</script>

<style scoped lang="stylus">
.alliance
  position relative
  z-index 1
  min-height 100%
  overflow-x hidden
  overflow-y auto
  -webkit-overflow-scrolling touch
  color #0f2744

.alliance__layout
  max-width 480px
  margin 0 auto
  padding 14px 14px calc(78px + env(safe-area-inset-bottom, 0px))
  display flex
  flex-direction column
  gap 12px
  box-sizing border-box

  &--locked
    filter blur(7px)
    user-select none
    pointer-events none
    transform scale(1.01)

.alliance__closed-mask
  position absolute
  inset 0
  z-index 20
  display flex
  align-items center
  justify-content center
  padding 24px 20px calc(90px + env(safe-area-inset-bottom, 0px))
  box-sizing border-box
  background rgba(184, 227, 255, 0.28)
  backdrop-filter blur(2px)
  -webkit-backdrop-filter blur(2px)
  pointer-events auto
  touch-action none

.alliance__closed-card
  width 100%
  max-width 280px
  padding 28px 22px 24px
  text-align center
  border-radius 20px
  background rgba(255, 255, 255, 0.92)
  border 1px solid rgba(255, 255, 255, 0.95)
  box-shadow 0 16px 40px rgba(15, 39, 68, 0.16)

.alliance__closed-title
  margin 14px 0 0
  font-size 18px
  font-weight 700
  color #0f2744
  letter-spacing 0.04em

.alliance__closed-desc
  margin 8px 0 0
  font-size 13px
  line-height 1.5
  color rgba(15, 39, 68, 0.55)

.glass
  background rgba(255, 255, 255, 0.96)
  border 1px solid rgba(255, 255, 255, 0.9)
  border-radius 20px
  box-shadow 0 10px 28px rgba(15, 39, 68, 0.08), 0 2px 8px rgba(15, 39, 68, 0.04)
  backdrop-filter blur(12px)

/* —— 会员卡 —— */
.member-card
  position relative
  overflow hidden
  border-radius 22px
  padding 18px 16px 16px
  color #fff
  background linear-gradient(145deg, #0b1f3a 0%, #163a66 48%, #1f57a8 100%)
  box-shadow 0 16px 36px rgba(11, 31, 58, 0.35)
  animation alliance-rise 0.45s ease both

  &--star3
    background linear-gradient(145deg, #1a1208 0%, #3d2a12 40%, #8a6a2f 78%, #c9a227 100%)
  &--star2
    background linear-gradient(145deg, #1a1030 0%, #2f1f5a 50%, #5b3fa0 100%)
  &--star1
    background linear-gradient(145deg, #0c2a3a 0%, #14506a 50%, #1f8aaa 100%)
  &--vip
    background linear-gradient(145deg, #2a1a08 0%, #7a4b12 55%, #d4a017 100%)

.member-card__shine
  position absolute
  inset -40% -20% auto
  height 70%
  background linear-gradient(120deg, transparent 20%, rgba(255,255,255,0.18) 48%, transparent 72%)
  transform rotate(-8deg)
  animation alliance-shine 5.5s ease-in-out infinite
  pointer-events none

.member-card__orb
  position absolute
  border-radius 50%
  filter blur(2px)
  pointer-events none
  &--a
    width 120px
    height 120px
    right -30px
    top -40px
    background radial-gradient(circle, rgba(79,172,254,0.45), transparent 70%)
  &--b
    width 90px
    height 90px
    left -20px
    bottom -30px
    background radial-gradient(circle, rgba(67,233,123,0.28), transparent 70%)

.member-card__top
  position relative
  z-index 1
  display flex
  align-items flex-start
  justify-content space-between
  gap 10px
  margin-bottom 18px

.member-card__brand
  display flex
  align-items center
  gap 10px
  h1
    margin 0
    font-size 20px
    font-weight 800
    letter-spacing 0.03em
  p
    margin 4px 0 0
    font-size 11px
    opacity 0.78

.member-card__flag
  width 40px
  height 40px
  border-radius 12px
  display flex
  align-items center
  justify-content center
  font-size 12px
  font-weight 900
  letter-spacing 0.06em
  background rgba(255,255,255,0.16)
  border 1px solid rgba(255,255,255,0.28)
  box-shadow inset 0 1px 0 rgba(255,255,255,0.25)

.member-card__badge
  display inline-flex
  align-items center
  gap 5px
  padding 6px 10px
  border-radius 999px
  background rgba(255,255,255,0.14)
  border 1px solid rgba(255,255,255,0.22)
  font-size 12px
  font-weight 700
  backdrop-filter blur(8px)

.member-card__power
  position relative
  z-index 1
  margin-bottom 14px

.member-card__power-label
  font-size 11px
  opacity 0.72
  letter-spacing 0.08em

.member-card__power-row
  display flex
  align-items baseline
  gap 6px
  margin-top 4px
  strong
    font-size 34px
    font-weight 800
    letter-spacing 0.01em
    line-height 1
  em
    font-style normal
    font-size 13px
    opacity 0.8

.member-card__progress
  position relative
  z-index 1
  padding-top 4px

.member-card__progress-head
  display flex
  justify-content space-between
  gap 8px
  font-size 11px
  opacity 0.85
  margin-bottom 8px

.member-card__bar
  height 7px
  border-radius 999px
  background rgba(255,255,255,0.18)
  overflow hidden
  i
    display block
    height 100%
    border-radius inherit
    background linear-gradient(90deg, #7dd3fc, #fff, #fde68a)
    box-shadow 0 0 12px rgba(255,255,255,0.45)
    transition width 0.5s ease

.member-card__gap
  margin 8px 0 0
  font-size 11px
  opacity 0.78

/* —— 等级阶梯 —— */
.tier-rail
  padding 14px 12px 12px
  animation alliance-rise 0.5s ease 0.05s both

.section-head
  padding 0 4px 10px

.section-kicker
  display block
  font-size 10px
  font-weight 800
  letter-spacing 0.16em
  color #4facfe
  margin-bottom 2px

.section-head h2
  margin 0
  font-size 16px
  font-weight 800
  color #0f2744

.tier-rail__scroll
  display flex
  gap 8px
  overflow-x auto
  padding-bottom 4px
  -webkit-overflow-scrolling touch
  scrollbar-width none
  &::-webkit-scrollbar
    display none

.tier-chip
  flex 0 0 auto
  width 108px
  margin 0
  padding 12px 10px
  border none
  border-radius 14px
  background #f4f7fb
  border 1px solid transparent
  text-align left
  cursor default
  transition transform 0.2s ease, box-shadow 0.2s ease, border-color 0.2s ease
  &.is-passed
    background linear-gradient(180deg, #eef6ff, #f8fbff)
    border-color rgba(47, 107, 255, 0.18)
  &.is-active
    transform translateY(-2px)
    border-color rgba(47, 107, 255, 0.45)
    box-shadow 0 10px 20px rgba(47, 107, 255, 0.16)
    background #fff

.tier-chip__idx
  display inline-flex
  width 20px
  height 20px
  align-items center
  justify-content center
  border-radius 50%
  font-size 10px
  font-weight 800
  color #fff
  background #94a3b8
  margin-bottom 8px

.tier-chip.is-active .tier-chip__idx,
.tier-chip.is-passed .tier-chip__idx
  background linear-gradient(135deg, #2f6bff, #4facfe)

.tier-chip--star3.is-active .tier-chip__idx
  background linear-gradient(135deg, #b45309, #f59e0b)

.tier-chip__name
  display block
  font-size 12px
  font-weight 800
  color #0f2744
  margin-bottom 4px

.tier-chip__range
  display block
  font-size 10px
  color rgba(15, 39, 68, 0.5)
  line-height 1.3

/* —— 分区 Tab —— */
.zone-tabs
  display flex
  padding 4px
  position relative
  gap 0
  animation alliance-rise 0.5s ease 0.1s both

  &:before
    content ''
    position absolute
    width calc(33.333% - 4px)
    height calc(100% - 8px)
    top 4px
    left 4px
    border-radius 14px
    background #0f172a
    box-shadow 0 8px 18px rgba(15, 23, 42, 0.28)
    transition transform 0.28s cubic-bezier(0.4, 0, 0.2, 1)
    z-index 0

  &[data-active="profit"]:before
    transform translateX(100%)
  &[data-active="private"]:before
    transform translateX(200%)

.zone-tabs__item
  flex 1
  position relative
  z-index 1
  margin 0
  padding 11px 4px
  border none
  background transparent
  display flex
  flex-direction column
  align-items center
  gap 3px
  font-size 12px
  font-weight 700
  color #7b8794
  cursor pointer
  -webkit-tap-highlight-color transparent
  transition color 0.2s ease
  &.active
    color #fff

/* —— 规则条 —— */
.rule-bar
  display flex
  align-items center
  gap 10px
  padding 12px
  animation alliance-rise 0.5s ease 0.12s both

.rule-bar__icon
  flex-shrink 0
  width 40px
  height 40px
  border-radius 12px
  display flex
  align-items center
  justify-content center
  background linear-gradient(145deg, #2f6bff, #1f57e8)
  box-shadow 0 8px 16px rgba(47, 107, 255, 0.28)

.rule-bar__body
  flex 1
  min-width 0
  strong
    display block
    font-size 13px
    margin-bottom 2px
  p
    margin 0
    font-size 11px
    color rgba(15, 39, 68, 0.58)
    line-height 1.4

.rule-bar__fee
  display inline-block
  margin-top 4px
  font-size 11px
  font-weight 700
  color #2f6bff

.rule-bar__cta
  flex-shrink 0
  margin 0
  padding 10px 12px
  border none
  border-radius 12px
  background linear-gradient(160deg, #5b8cff, #2f6bff)
  color #fff
  font-size 12px
  font-weight 800
  display flex
  flex-direction column
  align-items center
  gap 2px
  box-shadow 0 8px 16px rgba(47, 107, 255, 0.28)
  cursor pointer
  small
    font-size 10px
    opacity 0.85
    font-weight 600

.rule-bar__lock
  flex-shrink 0
  max-width 88px
  display flex
  flex-direction column
  align-items center
  gap 4px
  font-size 10px
  font-weight 700
  color #c2410c
  text-align center
  line-height 1.25

/* —— 列表区 —— */
.zone-panel
  display flex
  flex-direction column
  gap 12px

.deal-card
  position relative
  overflow hidden
  padding 14px 14px 14px 16px
  animation alliance-rise 0.45s ease var(--delay, 0ms) both

.deal-card__accent
  position absolute
  left 0
  top 0
  bottom 0
  width 4px
  &--blue
    background linear-gradient(180deg, #4facfe, #2f6bff)
  &--gold
    background linear-gradient(180deg, #fbbf24, #d97706)

.deal-card__head
  display flex
  align-items center
  gap 10px
  margin-bottom 12px

.deal-card__avatar
  flex-shrink 0
  width 44px
  height 44px
  border-radius 14px
  display flex
  align-items center
  justify-content center
  &--ex
    background linear-gradient(145deg, #2f6bff, #4facfe)
    box-shadow 0 8px 16px rgba(47, 107, 255, 0.28)
  &--pf
    background linear-gradient(145deg, #d97706, #fbbf24)
    box-shadow 0 8px 16px rgba(217, 119, 6, 0.28)

.deal-card__title
  flex 1
  min-width 0
  h3
    margin 0 0 4px
    font-size 15px
    font-weight 800
    color #0f2744
    white-space nowrap
    overflow hidden
    text-overflow ellipsis

.deal-card__phone
  display inline-flex
  align-items center
  gap 3px
  font-size 12px
  color #2f6bff
  text-decoration none
  font-weight 600

.deal-card__link
  flex-shrink 0
  margin 0
  padding 7px 10px
  border none
  border-radius 10px
  background rgba(47, 107, 255, 0.08)
  color #2f6bff
  font-size 12px
  font-weight 700
  display inline-flex
  align-items center
  gap 3px
  cursor pointer

.deal-card__metrics
  display grid
  grid-template-columns 1fr 1fr
  gap 8px
  margin-bottom 12px
  &--3
    grid-template-columns repeat(3, 1fr)

.metric
  padding 10px 8px
  border-radius 12px
  background linear-gradient(180deg, #f5f8fc, #eef3f9)
  text-align center
  span
    display block
    font-size 10px
    color rgba(15, 39, 68, 0.5)
    margin-bottom 4px
  strong
    display block
    font-size 14px
    font-weight 800
    color #0f2744
    line-height 1.2
  em
    display block
    margin-top 2px
    font-style normal
    font-size 10px
    color rgba(15, 39, 68, 0.45)
  &--hot
    background linear-gradient(180deg, #fff7ed, #ffedd5)
    strong
      color #c2410c

.deal-card__tip
  display flex
  align-items center
  gap 5px
  margin 0 0 12px
  font-size 11px
  color rgba(15, 39, 68, 0.55)

.deal-card__cta
  width 100%
  margin 0
  padding 12px 14px
  border none
  border-radius 14px
  background linear-gradient(160deg, #5b8cff 0%, #2f6bff 55%, #1f57e8 100%)
  color #fff
  font-size 14px
  font-weight 800
  letter-spacing 0.02em
  box-shadow 0 10px 20px rgba(47, 107, 255, 0.3)
  cursor pointer
  -webkit-tap-highlight-color transparent
  transition transform 0.15s ease
  &:active
    transform scale(0.985)
  &--gold
    background linear-gradient(160deg, #fbbf24, #d97706)
    box-shadow 0 10px 20px rgba(217, 119, 6, 0.28)
  &--compact
    width auto
    padding 10px 18px
    font-size 13px

/* —— 激活卡 —— */
.activate-card
  position relative
  overflow hidden
  padding 16px
  animation alliance-rise 0.45s ease both

.activate-card__ribbon
  position absolute
  top 12px
  right -28px
  width 120px
  padding 4px 0
  text-align center
  font-size 10px
  font-weight 800
  color #fff
  background linear-gradient(90deg, #ef4444, #f97316)
  transform rotate(35deg)
  box-shadow 0 4px 10px rgba(239, 68, 68, 0.3)

.activate-card h3
  margin 0 0 8px
  font-size 16px
  font-weight 800
  padding-right 48px

.activate-card__desc
  margin 0 0 14px
  font-size 12px
  line-height 1.55
  color rgba(15, 39, 68, 0.68)

.activate-card__flow
  display flex
  align-items stretch
  gap 8px
  margin-bottom 12px

.flow-node
  flex 1
  padding 12px 8px
  border-radius 14px
  background linear-gradient(180deg, #eff6ff, #f8fbff)
  text-align center
  span
    display block
    font-size 11px
    color rgba(15, 39, 68, 0.55)
  strong
    display block
    margin 6px 0 2px
    font-size 20px
    font-weight 800
    color #2f6bff
  em
    font-style normal
    font-size 10px
    color rgba(15, 39, 68, 0.45)
  &--seller
    background linear-gradient(180deg, #fff7ed, #fffbeb)
    strong
      color #d97706

.flow-arrow
  align-self center
  font-size 16px
  color rgba(15, 39, 68, 0.35)
  font-weight 700

.activate-card__note
  margin 0 0 14px
  font-size 11px
  line-height 1.55
  color rgba(15, 39, 68, 0.55)

/* —— 私域商品 —— */
.product-card
  padding 14px
  animation alliance-rise 0.45s ease var(--delay, 0ms) both

.product-card__visual
  display flex
  align-items center
  gap 12px
  margin-bottom 12px

.product-card__icon
  width 52px
  height 52px
  border-radius 16px
  display flex
  align-items center
  justify-content center
  background linear-gradient(145deg, #0f2744, #2f6bff)
  box-shadow 0 10px 20px rgba(15, 39, 68, 0.22)

.product-card__meta
  flex 1
  min-width 0
  h3
    margin 0 0 4px
    font-size 16px
    font-weight 800

.product-card__price
  margin 0
  display flex
  align-items baseline
  gap 6px
  small
    font-size 11px
    color rgba(15, 39, 68, 0.5)
  strong
    font-size 20px
    color #ef4444
    font-weight 800

.product-card__gift
  flex-shrink 0
  max-width 78px
  padding 6px 8px
  border-radius 10px
  background #ecfdf5
  color #047857
  font-size 10px
  font-weight 800
  line-height 1.3
  text-align center

.product-card__form
  margin-bottom 12px
  border-radius 14px
  overflow hidden
  background #f5f8fc
  :deep(.van-cell)
    background transparent
    padding 10px 12px
  :deep(.van-field__label)
    width 72px
    color rgba(15, 39, 68, 0.55)

.product-card__foot
  display flex
  align-items center
  justify-content space-between
  gap 10px

.product-card__fee
  font-size 11px
  color rgba(15, 39, 68, 0.5)
  font-weight 600

.alliance-empty
  margin 28px 0
  text-align center
  font-size 13px
  color rgba(15, 39, 68, 0.4)

.dlg
  padding 8px 16px 16px
  &--form
    padding 4px 0 8px

.dlg__name
  margin 0 0 8px
  font-size 14px
  font-weight 700

.dlg__link
  margin 0
  padding 10px 12px
  border-radius 10px
  background #f5f8fc
  font-size 12px
  word-break break-all
  color #2f6bff

.dlg__zones
  padding 8px 16px 4px
  span
    display block
    margin-bottom 8px
    font-size 13px
    color rgba(15, 39, 68, 0.55)

.dlg__zone-row
  display flex
  flex-wrap wrap
  gap 8px

.dlg__zone
  margin 0
  padding 8px 10px
  border 1px solid #ebedf0
  border-radius 8px
  background #fff
  font-size 12px
  color #646566
  cursor pointer
  &.is-active
    border-color #2f6bff
    color #2f6bff
    background rgba(47, 107, 255, 0.08)

@keyframes alliance-rise
  from
    opacity 0
    transform translateY(12px)
  to
    opacity 1
    transform translateY(0)

@keyframes alliance-shine
  0%, 100%
    transform translateX(-10%) rotate(-8deg)
  50%
    transform translateX(18%) rotate(-8deg)

@media (prefers-reduced-motion: reduce)
  .member-card,
  .tier-rail,
  .zone-tabs,
  .rule-bar,
  .deal-card,
  .activate-card,
  .product-card
    animation none
  .member-card__shine
    animation none
</style>
