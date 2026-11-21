<template>
    <div class="power-pool-page">
      <van-nav-bar
        :title="t('c2c.title')"
        left-arrow
        fixed
        placeholder
        @click-left="router.back()"
      />
  
      <div class="content">
        <section class="stats-card">
          <div class="stat">
            <span class="label">{{ t('c2c.availablePower') }}</span>
            <span class="value">{{ powerBalance.toFixed(2) }}</span>
          </div>
  <!-- 
          <div class="stat">
            <span class="label">可用SOHA</span>
            <span class="value">{{ coinBalance.toFixed(2) }}</span>
          </div>
           -->
          <div class="stat">
            <span class="label">撮合价格</span>
            <span class="value">${{ fixedPrice.toFixed(2) }}</span>
          </div>
        </section>
  
        <section class="form-card">
          <h3>{{ t('c2c.submitTitle') }}</h3>
          <van-field
            v-model="inputPower"
            type="number"
            :label="t('c2c.powerLabel')"
            :placeholder="t('c2c.powerPlaceholder')"
            input-align="right"
          />
          <div class="tips">
            {{ t('c2c.tips', { destroyed: destroyedPreview, listed: listedPreview }) }}
          </div>
          <div class="preview">
            <div>
              <span>{{ t('c2c.previewDestroyed') }}</span>
              <strong>{{ destroyedPreview }}</strong>
            </div>
            <div>
              <span>{{ t('c2c.previewListed') }}</span>
              <strong>{{ listedPreview }}</strong>
            </div>
            <div>
              <span>{{ t('c2c.previewPrice') }}</span>
              <strong>${{ fixedPrice.toFixed(2) }}</strong>
            </div>
          </div>
          <van-button
            type="primary"
            block
            :loading="submitting"
            @click="handleSubmit"
          >
            {{ t('c2c.submitButton') }}
          </van-button>
        </section>
  
        <section class="orders-card">
          <div class="section-header">
            <h3>{{ t('c2c.poolTitle') }}</h3>
            <van-button size="small" type="primary" icon="replay" @click="refreshAll">
              {{ t('c2c.refresh') }}
            </van-button>
            <van-button size="small" type="success" icon="orders-o" @click="mineOrder">
              {{ t('c2c.myOrdersButton') }}
            </van-button>
          </div>
          <van-empty v-if="poolOrders.length === 0" :description="t('c2c.emptyPool')" />
          <div v-else class="order-list">
            <div
              v-for="item in poolOrders"
              :key="item.id"
              class="order-item"
            >
              <div class="order-header">
                <div class="order-title">
                  <span class="order-no">#{{ formatOrderNumber(item.orderNumber) }}</span>
                  <span class="order-user">{{ item.username || t('c2c.anonymous') }}</span>
                </div>
                <van-tag class="status-tag" type="warning" round>{{ t('c2c.statusPending') }}</van-tag>
              </div>
              <div class="order-body">
                <div class="row">
                  <span>{{ t('c2c.poolPower') }}</span>
                  <strong>{{ item.listedPower.toFixed(2) }}</strong>
                </div>
              <div class="row">
                <span>{{ t('c2c.poolValue') }}</span>
                <strong>${{ getOrderValue(item).toFixed(2) }}</strong>
              </div>
                <div class="row">
                  <span>{{ t('c2c.poolDestroyed') }}</span>
                  <strong>{{ item.destroyedPower.toFixed(2) }}</strong>
                </div>
                <div class="row">
                  <span>{{ t('c2c.poolCreatedAt') }}</span>
                  <strong>{{ formatTime(item.createdAt) }}</strong>
                </div>
              </div>
              <van-button
                size="small"
                block
                type="primary"
                plain
                :loading="matchingId === item.id"
                @click="handleMatch(item)"
              >
                {{ t('c2c.matchButton') }}
              </van-button>
            </div>
          </div>
        </section>
  
        <section class="orders-card">
          <div class="section-header">
            <h3>{{ t('c2c.mineTitle') }}</h3>
          </div>
          <van-empty v-if="myOrders.length === 0" :description="t('c2c.emptyMine')" />
          <div v-else class="order-list">
            <div
              v-for="item in myOrders"
              :key="item.id"
              class="order-item"
            >
              <div class="order-header">
                <div class="order-title">
                  <span class="order-no">#{{ formatOrderNumber(item.orderNumber) }}</span>
                  <span class="order-user">数量 {{ item.listedPower.toFixed(2) }}</span>
                </div>
                <van-tag
                  class="status-tag"
                  :type="statusMap[item.status] || 'primary'"
                >
                  {{ statusText(item.status) }}
                </van-tag>
              </div>
              <div class="order-body">
                <div class="row">
                  <span>{{ t('c2c.poolDestroyed') }}</span>
                  <strong>{{ item.destroyedPower.toFixed(2) }}</strong>
                </div>
              <div class="row">
                <span>{{ t('c2c.poolValue') }}</span>
                <strong>${{ getOrderValue(item).toFixed(2) }}</strong>
              </div>
                <div class="row">
                  <span>{{ t('c2c.poolCreatedAt') }}</span>
                  <strong>{{ formatTime(item.createdAt) }}</strong>
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  </template>
  
  <script setup>
  import { ref, computed, onMounted } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { useRouter } from 'vue-router'
  import { showSuccessToast, showFailToast, showConfirmDialog } from 'vant'
  import { httpGet, httpPost } from '@/utils/http'
  
  defineOptions({
    name: 'PowerPoolView'
  })
  
  const { t } = useI18n()
  const router = useRouter()
  const fixedPrice = 0.01
  
  const powerBalance = ref(0)
  const coinBalance = ref(0)
  const isMerchant = ref(false) // 仅商户可挂卖
  const inputPower = ref('')
  const submitting = ref(false)
  const matchingId = ref(null)
  const poolOrders = ref([])
  const myOrders = ref([])
  
  const destroyedPreview = computed(() => {
    const value = Number(inputPower.value) || 0
    return (value * 0.5).toFixed(2)
  })
  
  const listedPreview = computed(() => {
    const value = Number(inputPower.value) || 0
    return (value * 0.5).toFixed(2)
  })
  
  const statusMap = {
    1: 'warning',
    2: 'primary',
    3: 'success',
    4: 'success',
    5: 'danger',
    6: 'danger'
  }
  
  const statusText = (status) => {
    switch (status) {
      case 1:
        return '待撮合'
      case 2:
        return '撮合中'
      case 3:
        return '已支付'
      case 4:
        return '已完成'
      case 5:
        return '已取消'
      case 6:
        return '申诉中'
      default:
        return '未知'
    }
  }
  
  const formatOrderNumber = (value) => {
    if (!value) return '--'
    const str = String(value)
    if (str.length <= 12) return str
    return `${str.slice(0, 4)}...${str.slice(-4)}`
  }
  
  const formatTime = (ts) => {
    if (!ts) return '--'
    return new Date(ts * 1000).toLocaleString()
  }
  
  const getOrderValue = (order) => {
    if (!order) return 0
    const value = Number(order.valueUsd)
    if (!Number.isNaN(value) && value > 0) {
      return value
    }
    const listed = Number(order.listedPower ?? order.number ?? 0)
    return listed * fixedPrice
  }
  
  const loadProfile = () => {
    httpGet('/api/user/profile')
      .then(res => {
        powerBalance.value = res.data.power || 0
        coinBalance.value = res.data.coin || 0
        isMerchant.value = !!res.data.vip
      })
      .catch(e => {
        showFailToast(e.message || '获取用户信息失败')
        isMerchant.value = false
      })
  }
  
  const loadPoolOrders = () => {
    httpGet('/api/trade/powerPool/orders')
      .then(res => {
        poolOrders.value = Array.isArray(res.data) ? res.data : []
      })
      .catch(e => {
        showFailToast(e.message || '获取订单池失败')
      })
  }
  
  const loadMyOrders = () => {
    httpGet('/api/trade/powerPool/mine')
      .then(res => {
        myOrders.value = Array.isArray(res.data) ? res.data : []
      })
      .catch(e => {
        showFailToast(e.message || '获取我的挂单失败')
      })
  }
  
  const refreshAll = () => {
    loadProfile()
    loadPoolOrders()
    loadMyOrders()
  }
  
  const mineOrder = () => {
  
    router.push('/mobile/powertrade');
  
  }
  
  
  
  const handleSubmit = () => {
    const value = Number(inputPower.value)
    if (!isMerchant.value) {
      showFailToast(t('powerPool.onlyMerchant'))
      return
    }
    if (!value || value <= 0) {
      showFailToast('请输入有效的算力数量')
      return
    }
  
    
  
    if (value > powerBalance.value) {
      showFailToast('算力不足，无法挂卖')
      return
    }
  
    //最小1万算力,最多20万个算力挂卖
    if (value>200000 || value<10000){
      showFailToast('算力数量必须在1万到20万之间')
      return
    }
  
    if (value > powerBalance.value) {
      showFailToast('算力不足，无法挂卖')
      return
    }
  
  
    showConfirmDialog({
      title: '确认挂卖',
      message: `系统将预销毁 ${destroyedPreview.value} 算力，并以每算力 $${fixedPrice.toFixed(2)} 的价格挂售剩余算力，是否继续？`
    }).then(() => submitOrder(value))
  }
  
  const submitOrder = (value) => {
    submitting.value = true
    httpPost('/api/trade/powerTrade/c2c/submit', { power: value })
      .then(() => {
        showSuccessToast('挂单已创建')
        inputPower.value = ''
        refreshAll()
      })
      .catch(e => {
        showFailToast(e.message || '提交失败')
      })
      .finally(() => {
        submitting.value = false
      })
  }
  
  const handleMatch = (order) => {
    if (!order || !order.id) {
      showFailToast('订单信息异常，无法撮合')
      return
    }
  
    showConfirmDialog({
      title: '确认撮合',
      message: `是否撮合订单 #${order.orderNumber}?`
    })
      .then(() => executeMatch(order.id))
      .catch(() => {})
  }
  
  const executeMatch = (orderId) => {
    matchingId.value = orderId
    httpPost('/api/trade/powerPool/match', { id: orderId })
      .then(() => {
        showSuccessToast('撮合成功')
        // 撮合成功后跳转到我的算力订单列表
        router.push('/mobile/powertrade')
      })
      .catch(e => {
        showFailToast(e.message || '撮合失败')
      })
      .finally(() => {
        matchingId.value = null
      })
  }
  
  onMounted(() => {
    refreshAll()
  })
  </script>
  
  <style scoped lang="scss">
  .power-pool-page {
    min-height: 100vh;
    background-color: #f5f7fb;
  
    .content {
      padding: 16px;
    }
  
    section {
      background: #fff;
      border-radius: 12px;
      padding: 16px;
      margin-bottom: 16px;
      box-shadow: 0 6px 18px rgba(0, 0, 0, 0.04);
    }
  
    .stats-card {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 12px;
  
      .stat {
        display: flex;
        flex-direction: column;
        gap: 6px;
  
        .label {
          font-size: 12px;
          color: #909399;
        }
  
        .value {
          font-size: 18px;
          font-weight: 600;
          color: #303133;
        }
      }
    }
  
    .form-card {
      h3 {
        margin: 0 0 12px;
        font-size: 16px;
      }
  
      .tips {
        margin: 8px 0;
        font-size: 12px;
        color: #909399;
        line-height: 1.4;
      }
  
      .preview {
        display: flex;
        justify-content: space-between;
        margin: 12px 0;
        padding: 12px;
        background: #f5f7fb;
        border-radius: 10px;
  
        div {
          display: flex;
          flex-direction: column;
          font-size: 12px;
  
          strong {
            font-size: 16px;
            color: #333;
            margin-top: 4px;
          }
        }
      }
    }
  
    .orders-card {
      .section-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 12px;
  
        h3 {
          margin: 0;
          font-size: 16px;
        }
      }
  
      .order-list {
        display: flex;
        flex-direction: column;
        gap: 12px;
      }
  
      .order-item {
        border: 1px solid #f0f2f5;
        border-radius: 12px;
        padding: 12px;
  
        .order-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          flex-wrap: wrap;
          gap: 8px;
          margin-bottom: 12px;
  
          .order-title {
            display: flex;
            flex-wrap: wrap;
            gap: 6px;
            align-items: center;
          }
  
          .order-no {
            display: inline-block;
            max-width: 150px;
            font-family: 'JetBrains Mono', monospace;
            font-size: 13px;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            font-weight: 600;
            margin-right: 6px;
          }
  
          .order-user {
            font-size: 12px;
            color: #909399;
          }
  
          .status-tag {
            margin-left: auto;
            flex-shrink: 0;
          }
        }
  
        .order-body {
          display: flex;
          flex-direction: column;
          gap: 8px;
          margin-bottom: 12px;
  
          .row {
            display: flex;
            justify-content: space-between;
            font-size: 14px;
            color: #606266;
  
            strong {
              color: #303133;
            }
          }
        }
      }
    }
  }
  </style>
  
  