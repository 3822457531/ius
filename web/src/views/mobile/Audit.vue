<template>
  <div class="audit-page">
    <van-nav-bar
      :title="t('audit.navTitle')"
      left-arrow
      fixed
      placeholder
      @click-left="router.back()"
    />

    <div class="content">
      <!-- 筛选栏 -->
      <div class="filter-bar">
        <van-field
          v-model="query.order_no"
          :placeholder="t('audit.searchPlaceholder')"
          clearable
          @clear="handleSearch"
        >
          <template #button>
            <van-button size="small" type="primary" @click="handleSearch">
              {{ t('audit.searchBtn') }}
            </van-button>
          </template>
        </van-field>
        <van-dropdown-menu>
          <van-dropdown-item v-model="query.status" :options="statusOptions" @change="handleSearch" />
        </van-dropdown-menu>
      </div>

      <!-- 订单列表 -->
      <van-pull-refresh v-model="refreshing" @refresh="onRefresh">
        <van-list
          v-model:loading="loading"
          :finished="finished"
          :finished-text="t('audit.listFinishedText')"
          @load="loadOrders"
        >
          <van-skeleton v-if="loading && items.length === 0" title :row="3" animated />
          <van-empty
            v-else-if="items.length === 0"
            :description="t('audit.emptyDesc')"
          />
          <div v-else class="order-list">
            <div
              v-for="order in items"
              :key="order.id"
              class="order-card"
            >
              <div class="order-header">
                <div class="title">
                  <span class="order-no">
                    {{ t('audit.orderNoPrefix') }}{{ order.orderNumber }}
                  </span>
                  <van-tag
                    :type="getStatusType(order.status)"
                    size="small"
                  >
                    {{ getStatusText(order.status) }}
                  </van-tag>
                </div>
              </div>

              <div class="order-body">
                <div class="row">
                  <span>{{ t('audit.userLabel') }}</span>
                  <strong>{{ order.sellerName || t('audit.userUnknown') }}</strong>
                </div>
                <div class="row">
                  <span>{{ t('audit.withdrawPowerLabel') }}</span>
                  <strong>{{ formatNumber(order.listedPower) }}</strong>
                </div>
                <div class="row">
                  <span>{{ t('audit.withdrawAmountLabel') }}</span>
                  <strong class="price">$ {{ formatNumber(order.valueUsd) }}</strong>
                </div>
                <div class="row">
                  <span>{{ t('audit.priceLabel') }}</span>
                  <strong>$ {{ formatNumber(order.price) }}</strong>
                </div>
                <div class="row address-row">
                  <span>{{ t('audit.payAddressLabel') }}</span>
                  <div class="address-value">
                    <strong class="address-text">{{ formatAddress(order.ethAddress) }}</strong>
                    <van-button
                      size="mini"
                      type="primary"
                      plain
                      :disabled="!order.ethAddress"
                      @click="copyEthAddress(order.ethAddress)"
                    >
                      {{ t('audit.copyBtn') }}
                    </van-button>
                  </div>
                </div>
                <div class="row">
                  <span>{{ t('audit.createdAtLabel') }}</span>
                  <strong>{{ formatTime(order.createdAt) }}</strong>
                </div>
              </div>

              <div class="order-actions" v-if="order.status === 1">
                <van-button
                  type="primary"
                  block
                  @click="handleConfirm(order)"
                  :loading="confirming"
                >
                  {{ t('audit.confirmPaidBtn') }}
                </van-button>
              </div>
            </div>
          </div>
        </van-list>
      </van-pull-refresh>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { httpPost } from '@/utils/http'
import { ElMessage } from 'element-plus'
import { showConfirmDialog, showToast } from 'vant'

const { t } = useI18n()
const router = useRouter()

const items = ref([])
const loading = ref(false)
const finished = ref(false)
const refreshing = ref(false)
const confirming = ref(false)
const page = ref(1)
const pageSize = ref(15)
const total = ref(0)

const query = ref({
  order_no: '',
  status: '',
  page: 1,
  page_size: 15
})

const statusOptions = [
  { text: t('audit.statusAll'), value: '' },
  { text: t('audit.statusPending'), value: 1 },
  { text: t('audit.statusDone'), value: 2 }
]

const formatNumber = (num) => {
  if (!num) return '0.00'
  return Number(num).toFixed(2)
}

const formatAddress = (address) => {
  if (!address) return t('audit.addressNotSet')
  if (address.length <= 12) return address
  return `${address.slice(0, 6)}...${address.slice(-6)}`
}

const formatTime = (timestamp) => {
  if (!timestamp) return '-'
  const date = new Date(timestamp * 1000)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const getStatusType = (status) => {
  switch(status) {
    case 1: return 'warning'
    case 2: return 'success'
    default: return 'default'
  }
}

const getStatusText = (status) => {
  switch(status) {
    case 1: return t('audit.statusTextPending')
    case 2: return t('audit.statusTextDone')
    default: return t('audit.statusTextUnknown')
  }
}

const copyEthAddress = (address) => {
  if (!address) return
  navigator.clipboard.writeText(address).then(() => {
    showToast(t('audit.copySuccess'))
  }).catch(() => {
    showToast(t('audit.copyFail'))
  })
}

const loadOrders = async () => {
  if (loading.value || finished.value) return
  
  loading.value = true
  try {
    const params = {
      order_no: query.value.order_no || '',
      status: query.value.status === '' ? null : Number(query.value.status),
      page: page.value,
      page_size: pageSize.value
    }
    
    const res = await httpPost('/api/trade/audit/buyback/list', params)
    if (res.data) {
      const newItems = res.data.items || []
      if (page.value === 1) {
        items.value = newItems
      } else {
        items.value.push(...newItems)
      }
      total.value = res.data.total || 0
      page.value = res.data.page || page.value
      
      if (items.value.length >= total.value) {
        finished.value = true
      } else {
        page.value++
      }
    }
  } catch (e) {
    ElMessage.error(t('audit.loadOrdersFailPrefix') + (e.message || ''))
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

const onRefresh = () => {
  page.value = 1
  finished.value = false
  loadOrders()
}

const handleSearch = () => {
  page.value = 1
  finished.value = false
  items.value = []
  loadOrders()
}

const handleConfirm = (order) => {
  showConfirmDialog({
    title: t('audit.confirmDialogTitle'),
    message: `${t('audit.confirmDialogMessagePrefix')}${order.orderNumber}${t('audit.confirmDialogMessageMid')}`,
  }).then(async () => {
    confirming.value = true
    try {
      await httpPost('/api/trade/audit/buyback/confirm', {
        orderId: order.id
      })
      showToast(t('audit.confirmSuccess'))
      // 刷新列表
      page.value = 1
      finished.value = false
      items.value = []
      await loadOrders()
    } catch (e) {
      ElMessage.error(t('audit.confirmFailPrefix') + (e.message || ''))
    } finally {
      confirming.value = false
    }
  }).catch(() => {
    // 用户取消
  })
}

onMounted(() => {
  loadOrders()
})
</script>

<style lang="stylus" scoped>
.audit-page
  min-height 100vh
  background linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%)
  padding-bottom 20px

.content
  padding 16px

.filter-bar
  background #fff
  border-radius 12px
  padding 12px
  margin-bottom 16px
  box-shadow 0 2px 8px rgba(0, 0, 0, 0.1)

.order-list
  .order-card
    background #fff
    border-radius 12px
    padding 16px
    margin-bottom 16px
    box-shadow 0 2px 8px rgba(0, 0, 0, 0.1)

    .order-header
      display flex
      justify-content space-between
      align-items center
      margin-bottom 12px
      padding-bottom 12px
      border-bottom 1px solid #f0f0f0

      .title
        display flex
        align-items center
        gap 8px
        flex 1

        .order-no
          font-size 14px
          font-weight 600
          color #2c3e50

    .order-body
      .row
        display flex
        justify-content space-between
        align-items center
        padding 8px 0
        font-size 14px

        span
          color #666

        strong
          color #2c3e50
          font-weight 600

          &.price
            color #ee4d2d
            font-size 16px

        &.address-row
          .address-value
            display flex
            align-items center
            gap 8px
            flex 1
            justify-content flex-end

            .address-text
              font-size 12px
              max-width 120px
              overflow hidden
              text-overflow ellipsis
              white-space nowrap

    .order-actions
      margin-top 16px
      padding-top 16px
      border-top 1px solid #f0f0f0
</style>

