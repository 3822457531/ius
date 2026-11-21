<template>
  <div class="invest-page custom-scroll">
    <div class="gradient-bg"></div>
    
    <div class="layout">
      <div class="glass-card section-card" v-loading="ordersLoading">
        <div class="section-header">
          <div>
            <div class="title">
              {{ orderTab === 'exited' ? t('invest.titleExited') : t('invest.titlePending') }}
            </div>
            <!-- <div class="hint">{{ orderTab === 'exited' ? '已出局的算力' : '待产出（投入的算力）' }}</div> -->
          </div>
          <van-button size="mini" plain type="primary" @click="fetchOrders">
            {{ t('invest.refresh') }}
          </van-button>
        </div>
        <div v-if="orders.length" class="order-list">
          <div class="order-card" v-for="(order, index) in orders" :key="order.id">
            <div class="row row-order-no">
              <span class="order-no">
                <van-icon name="orders-o" class="order-no-icon" />
                <span class="order-no-num">{{ index + 1 }}</span>
              </span>
              <van-tag :type="Number(order.status) === 9 ? 'default' : 'success'" plain>
                {{ Number(order.status) === 9 ? t('invest.statusExited') : t('invest.statusInvesting') }}
              </van-tag>
            </div>
            <div class="row">
              <div class="name">
                <span>{{ t('invest.salesLabel') }} {{ formatNum(order.sales) }}</span>
                <span>{{ t('invest.powerLabel') }} {{ formatNum(order.power) }}</span>
              </div>
            </div>
            <!-- <div class="meta meta-sub">
              <span>投入 {{ formatNum(order.sales) }}</span>
              <span>预产 {{ formatNum(order.power) }}</span>
            </div> -->
            <!-- 待产出 tab 只显示待产，已产出 tab 只显示已产 -->
            <div class="pending-produced-row">
              <div v-if="orderTab === 'pending'" class="pp-block pp-pending">
                <span class="pp-label">{{ t('invest.pendingLabel') }}</span>
                <span class="pp-value">{{ formatNum(getPendingPower(order)) }}</span>
              </div>
              <div v-if="orderTab === 'exited'" class="pp-block pp-produced">
                <span class="pp-label">{{ t('invest.producedLabel') }}</span>
                <span class="pp-value">{{ formatNum(order.js_power) }}</span>
              </div>
            </div>
            <div class="meta meta-footer">
              <span>{{ t('invest.timePrefix') }}{{ formatTime(order.created_at) }}</span>
              <van-button
                size="mini"
                type="primary"
                plain
                :to="`/mobile/syitems?productId=${order.order_no}`"
              >
                {{ t('invest.incomeBtn') }}
              </van-button>
            </div>
          </div>
        </div>
        <van-empty
          v-else
          :description="orderTab === 'exited' ? t('invest.emptyExited') : t('invest.emptyPending')"
        />
      </div>

      <div class="bottom-spacing"></div>
    </div>

    <van-dialog
      v-model:show="customDialog"
      :title="t('invest.dialogTitle')"
      show-cancel-button
      @confirm="submitCustom"
      class="reinvest-dialog"
    >
      <div class="reinvest-dialog-content">
        <div class="reinvest-info">
          <div class="info-label">{{ t('invest.dialogAvailableLabel') }}</div>
          <div class="info-value">{{ profile.power || 0 }}</div>
        </div>
        <van-field
          v-model="customPower"
          type="number"
          :placeholder="t('invest.dialogPlaceholder')"
          input-align="right"
          clearable
        >
          <template #button>
            <div class="power-unit">算力</div>
          </template>
        </van-field>
      </div>
    </van-dialog>
  </div>
</template>

<script setup>
import {computed, onMounted, ref, watch} from 'vue'
import { useI18n } from 'vue-i18n'
import {useRoute, useRouter} from 'vue-router'
import {showFailToast, showNotify, showSuccessToast} from 'vant'
import {httpGet, httpPost} from '@/utils/http'
import {checkSession} from '@/action/session'
import {dateFormat} from '@/utils/libs'

const { t } = useI18n()

const router = useRouter()
const route = useRoute()
// pending=待产出(投入的算力)，exited=已产出(已出局的算力)
const orderTab = computed(() => (route.query.tab === 'exited' ? 'exited' : 'pending'))
const profile = ref({})
const products = ref([])
const orders = ref([])
const loginUser = ref(null)

const profileLoading = ref(false)
const productsLoading = ref(false)
const ordersLoading = ref(false)
const buyingId = ref(null)

const customDialog = ref(false)
const customPower = ref('')
const customLoading = ref(false)

const formatTime = (ts) => dateFormat(ts, 'MM-dd HH:mm')
const formatNum = (n) => {
  const v = Number(n)
  if (v === 0 || Number.isNaN(v)) return '0'
  return v >= 10000 ? v.toLocaleString() : String(Math.round(v))
}
// 同一订单的待产值：status=9 时为 0，否则为 now_power 或 预产-已解锁
const getPendingPower = (order) => {
  if (orderTab.value === 'exited') return 0
  if (order.now_power != null && !Number.isNaN(Number(order.now_power))) return Number(order.now_power)
  return Math.max(0, Number(order.power || 0) - Number(order.js_power || 0))
}

const fetchProfile = () => {
  profileLoading.value = true
  httpGet('/api/user/profile')
    .then((res) => {
      profile.value = res.data || {}
    })
    .catch((e) => {
      showFailToast(t('invest.errorFetchProfilePrefix') + e.message)
    })
    .finally(() => {
      profileLoading.value = false
    })
}

const fetchProducts = () => {
  productsLoading.value = true
  httpGet('/api/product/list')
    .then((res) => {
      products.value = res.data || []
    })
    .catch((e) => {
      showFailToast(t('invest.errorFetchProductsPrefix') + e.message)
    })
    .finally(() => {
      productsLoading.value = false
    })
}

const fetchOrders = () => {
  ordersLoading.value = true
  const type = orderTab.value
  httpGet('/api/order/getOrderList', { params: { type } })
    .then((res) => {
      orders.value = res.data || []
    })
    .catch((e) => {
      showFailToast(t('invest.errorFetchOrdersPrefix') + e.message)
    })
    .finally(() => {
      ordersLoading.value = false
    })
}

const openCustomDialog = () => {
  customPower.value = ''
  customDialog.value = true
}

const submitCustom = () => {
  const powerNum = Number(customPower.value)
  if (!powerNum || powerNum <= 0) {
    return showNotify({type: 'danger', message: t('invest.errorReinvestInvalid')})
  }
  if (Number(profile.value?.power || 0) < powerNum) {
    return showNotify({type: 'danger', message: t('invest.errorPowerNotEnough')})
  }
  if (!loginUser.value?.id) {
    return showNotify({type: 'danger', message: t('invest.loginRequired')})
  }

  customLoading.value = true
  httpPost('/api/payment/confirmJihuoEx', {
    power: powerNum,
    user_id: loginUser.value.id
  })
    .then(() => {
      showSuccessToast(t('invest.reinvestSuccess'))
      customDialog.value = false
      fetchProfile()
      fetchOrders()
    })
    .catch((e) => {
      showFailToast(t('invest.reinvestFailPrefix') + e.message)
    })
    .finally(() => {
      customLoading.value = false
    })
}

const confirmInvest = (product) => {
  if (!loginUser.value?.id) {
    return showNotify({type: 'danger', message: t('invest.loginRequired')})
  }
  buyingId.value = product.id
  httpPost('/api/payment/confirmJihuo', {
    product_id: product.id,
    user_id: loginUser.value.id
  })
    .then(() => {
      showSuccessToast(t('invest.reinvestSuccess'))
      fetchProfile()
      fetchOrders()
    })
    .catch((e) => {
      showFailToast(t('invest.reinvestFailPrefix') + e.message)
    })
    .finally(() => {
      buyingId.value = null
    })
}

watch(orderTab, () => {
  if (loginUser.value) fetchOrders()
})

onMounted(() => {
  checkSession()
    .then((user) => {
      loginUser.value = user
      fetchProfile()
      fetchProducts()
      fetchOrders()
    })
    .catch(() => {
      router.push('/login')
    })
})
</script>

<style lang="stylus" scoped>
@import "@/assets/css/custom-scroll.styl"

.invest-page
  position relative
  min-height 100vh
  background linear-gradient(135deg, #4facfe 0%, #00f2fe 50%, #43e97b 100%)
  color #2c3e50
  overflow hidden
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
  filter blur(0)
  z-index 0

.layout
  position relative
  z-index 1
  max-width 1120px
  margin 0 auto
  padding 18px 16px 80px
  display flex
  flex-direction column
  gap 16px

.glass-card
  position relative
  background rgba(255,255,255,0.95)
  border 1px solid rgba(255,255,255,0.8)
  box-shadow 0 8px 32px rgba(0,0,0,0.12), 0 2px 8px rgba(0,0,0,0.08)
  border-radius 20px
  padding 20px
  backdrop-filter blur(10px)
  overflow hidden
  transition all 0.3s ease
  &:before
    content ''
    position absolute
    inset 0
    border-radius inherit
    background linear-gradient(135deg, rgba(255,255,255,0.3), rgba(255,255,255,0))
    opacity 0
    transition opacity .25s ease
  &:after
    content ''
    position absolute
    inset -1px
    border-radius inherit
    background linear-gradient(120deg, rgba(79,172,254,0.2), rgba(67,233,123,0.2), rgba(255,159,67,0.2))
    filter blur(20px)
    opacity 0.4
    z-index -1
  &:hover
    transform translateY(-2px)
    box-shadow 0 12px 40px rgba(0,0,0,0.15), 0 4px 12px rgba(0,0,0,0.1)
    &:before
      opacity 1

.section-card
  .section-header
    display flex
    align-items center
    justify-content space-between
    gap 10px
    margin-bottom 16px
  .title
    font-size 20px
    font-weight 800
    color #2c3e50
    letter-spacing 0.4px
    position relative
    &::after
      content ''
      position absolute
      bottom -4px
      left 0
      width 30px
      height 3px
      background linear-gradient(90deg, #667eea 0%, #764ba2 100%)
      border-radius 2px
      opacity 0.6
  .hint
    font-size 13px
    color #7f8c8d
    letter-spacing 0.2px
    margin-top 4px

.order-list
  display flex
  flex-direction column
  gap 12px

.order-card
  padding 16px
  border-radius 16px
  background rgba(255,255,255,0.9)
  border 1px solid rgba(79,172,254,0.2)
  box-shadow 0 4px 16px rgba(0,0,0,0.1)
  display flex
  flex-direction column
  gap 12px
  transition all 0.3s ease
  &:hover
    transform translateY(-2px)
    box-shadow 0 6px 20px rgba(0,0,0,0.15)
    border-color rgba(79,172,254,0.3)

.row
  display flex
  align-items center
  justify-content space-between
  gap 8px

.row-order-no
  margin-bottom 6px
  padding-bottom 6px
  border-bottom 1px solid rgba(79, 172, 254, 0.1)

.name
  display flex
  flex-wrap wrap
  align-items center
  gap 8px 12px
  font-weight 700
  color #2c3e50
  font-size 16px
  letter-spacing 0.2px
  .order-no
    display inline-flex
    align-items center
    gap 4px
    padding 2px 8px
    border-radius 10px
    background linear-gradient(135deg, rgba(102, 126, 234, 0.12) 0%, rgba(118, 75, 162, 0.08) 100%)
    border 1px solid rgba(102, 126, 234, 0.2)
    font-size 13px
    color #667eea
    font-weight 700
    flex-shrink 0
    .order-no-icon
      font-size 14px
      opacity 0.9
    .order-no-num
      font-family 'DIN Alternate', 'Helvetica Neue', Arial, sans-serif
      min-width 1.2em
      text-align center

.meta
  display flex
  flex-wrap wrap
  gap 8px 16px
  font-size 13px
  color #7f8c8d
  font-weight 500

.meta-sub
  margin-top 2px

.meta-footer
  padding-top 4px
  border-top 1px solid rgba(79, 172, 254, 0.12)

// 同一订单的两个值：待产 / 已产，并排一目了然
.pending-produced-row
  display flex
  gap 12px
  margin 8px 0 4px

.pp-block
  flex 1
  min-width 0
  padding 12px 14px
  border-radius 12px
  display flex
  flex-direction column
  gap 4px
  .pp-label
    font-size 12px
    color #7f8c8d
    font-weight 500
  .pp-value
    font-size 18px
    font-weight 700
    font-family 'DIN Alternate', 'Helvetica Neue', Arial, sans-serif
    letter-spacing 0.5px

.pp-pending
  background linear-gradient(135deg, rgba(79, 172, 254, 0.12) 0%, rgba(0, 242, 254, 0.1) 100%)
  border 1px solid rgba(79, 172, 254, 0.25)
  .pp-value
    color #1989fa

.pp-produced
  background linear-gradient(135deg, rgba(67, 233, 123, 0.12) 0%, rgba(79, 172, 254, 0.08) 100%)
  border 1px solid rgba(67, 233, 123, 0.3)
  .pp-value
    color #07c160

.bottom-spacing
  height 70px

// 复投对话框样式
.reinvest-dialog
  :deep(.van-dialog)
    background rgba(255,255,255,0.98)
    border-radius 20px
    box-shadow 0 20px 60px rgba(0, 0, 0, 0.15), 0 8px 24px rgba(0, 0, 0, 0.1)
  
  :deep(.van-dialog__header)
    padding 24px 20px 20px
    font-size 19px
    font-weight 600
    color #2c3e50
    letter-spacing 0.3px
    border-bottom 1px solid rgba(0, 0, 0, 0.06)
  
  :deep(.van-dialog__content)
    padding 24px 20px 20px
    max-height 70vh
    overflow-y auto

.reinvest-dialog-content
  padding 4px 0
  
.reinvest-info
  display flex
  justify-content space-between
  align-items center
  padding 18px 20px
  background linear-gradient(135deg, #667eea 0%, #764ba2 100%)
  border-radius 16px
  margin-bottom 24px
  box-shadow 0 8px 24px rgba(102, 126, 234, 0.35), 0 2px 8px rgba(102, 126, 234, 0.2)
  .info-label
    font-size 15px
    color rgba(255, 255, 255, 0.95)
    font-weight 500
    letter-spacing 0.5px
  .info-value
    font-size 28px
    color #fff
    font-weight 700
    font-family 'DIN Alternate', 'Helvetica Neue', Arial, sans-serif
    text-shadow 0 2px 8px rgba(0, 0, 0, 0.15)
    letter-spacing -0.5px

:deep(.van-field)
  padding 14px 18px
  background #fff
  border-radius 14px
  margin-bottom 0
  border 1.5px solid #e4e7ed
  transition all 0.3s cubic-bezier(0.4, 0, 0.2, 1)
  box-shadow 0 2px 8px rgba(0, 0, 0, 0.04)
  &:focus-within
    border-color #667eea
    background #fff
    box-shadow 0 0 0 3px rgba(102, 126, 234, 0.12), 0 4px 12px rgba(102, 126, 234, 0.15)
    transform translateY(-1px)

:deep(.van-field__control)
  font-size 19px
  font-weight 600
  color #2c3e50
  text-align right
  letter-spacing 0.3px

:deep(.van-field__control::placeholder)
  color #c8c9cc
  font-weight normal
  font-size 15px

.power-unit
  padding 4px 10px
  font-size 13px
  color #667eea
  font-weight 600
  background linear-gradient(135deg, rgba(102, 126, 234, 0.1) 0%, rgba(118, 75, 162, 0.1) 100%)
  border-radius 6px
  margin-left 6px
  border 1px solid rgba(102, 126, 234, 0.2)
  transition all 0.2s ease
  &:hover
    background linear-gradient(135deg, rgba(102, 126, 234, 0.15) 0%, rgba(118, 75, 162, 0.15) 100%)
    border-color rgba(102, 126, 234, 0.3)

:deep(.van-empty)
  padding 40px 20px
  
:deep(.van-empty__description)
  color #7f8c8d
  font-size 14px

:deep(.van-button--mini)
  border-radius 12px
  font-weight 500
  transition all 0.3s ease
  &:active
    transform scale(0.98)

:deep(.van-tag)
  border-radius 10px
  font-weight 600
  padding 4px 12px

@media (max-width: 720px)
  .layout
    padding 14px 12px 70px
  .glass-card
    padding 16px
  .order-card
    padding 14px
</style>

