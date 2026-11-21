<template>
  <div class="power-pool-page">
    <van-nav-bar
      :title="t('powerPool.title')"
      left-arrow
      fixed
      placeholder
      @click-left="router.back()"
    >
      <template #right>
        <van-icon
          name="clock-o"
          size="18"
          class="recharge-records-ico"
          @click="toggleRechargeRecords"
        />
      </template>
    </van-nav-bar>

    <div class="content">
      <section class="stats-card">
        <!-- <div class="stat">
          <span class="label">可用算力</span>
          <span class="value">{{ powerBalance.toFixed(2) }}</span>
        </div> -->
        <div class="stat">
          <span class="label">{{ t('powerPool.statTotalPower') }}</span>
          <span class="value">{{ poolInfo.total_power.toFixed(2) }}</span>
        </div>

        <!-- <div class="stat">
          <span class="label">撮合池余额</span>
          <span class="value">${{ poolInfo.total_usdt.toFixed(2) }}</span>
        </div>  -->


        <!-- <div class="stat">
          <span class="label">平台总$</span>
          <span class="value">${{ poolInfo.total_usdt.toFixed(2) }}</span>
        </div> -->
   

      </section>

      <!-- 算力池充值（仅商户） -->
      <section class="form-card">
        <h3>{{ t('powerPool.rechargeTitle') }}</h3>
        <!-- <div v-if="!isMerchant" class="merchant-only-tip">
          {{ t('powerPool.merchantOnlyTip') }}
        </div> -->
        <van-field
          v-model="rechargePower"
          type="number"
          :label="t('powerPool.rechargeFieldLabel')"
          :placeholder="t('powerPool.rechargePlaceholder')"
          input-align="right"
        />
        <div class="tips">
          <p>{{ t('powerPool.rechargeTip1') }}</p>
          <p>
            {{ t('powerPool.rechargeTip2Prefix') }}{{ poolInfo.total_power.toFixed(2) }}
          </p>
          <!-- <p>{{ t('powerPool.rechargeTip3') }}</p> -->
        </div>
        <div class="preview" v-if="rechargePower && Number(rechargePower) > 0">
          <div>
            <span>{{ t('powerPool.rechargePayPrefix') }}</span>
            <strong>${{ rechargeUsdtAmount.toFixed(2) }}</strong>
          </div>
        </div>
        <van-button
          type="primary"
          block
          :loading="rechargeSubmitting"
          :disabled=" !rechargePower || Number(rechargePower) < RECHARGE_POWER_MIN"
          @click="handleRecharge"
        >
          {{ t('powerPool.rechargeConfirmBtn') }}
        </van-button>
      </section>


      <!-- <section class="stats-card">
        <div class="stat">
          <span class="label">{{ t('powerPool.statMyPower') }}</span>
          <span class="value">{{ powerBalance.toFixed(2) }}</span>
        </div>


      </section> -->

      <!-- 算力提现（5折，仅商户） -->
      <!-- <section class="form-card" v-if="poolInfo.buyback_usdt > 0">
      <h3>{{ t('powerPool.withdrawTitle') }}</h3>
      <div v-if="!isMerchant" class="merchant-only-tip">
        {{ t('powerPool.merchantOnlyTip') }}
      </div>
        <van-field
          v-model="withdrawUsdt"
          type="number"
        :label="t('powerPool.withdrawFieldLabel')"
        :placeholder="t('powerPool.withdrawPlaceholder')"
          input-align="right"
          :disabled="!isMerchant"
        />
        <div class="tips">
        <p>{{ t('powerPool.withdrawTip1') }}</p>
        <p>
        </p>
        <p>{{ t('powerPool.withdrawTip3') }}</p>
        </div>
        <div class="preview" v-if="withdrawUsdt && Number(withdrawUsdt) > 0">
          <div>
          <span>{{ t('powerPool.withdrawExpect') }}</span>
            <strong>${{ calculateWithdrawUsdt(Number(withdrawUsdt)).toFixed(2) }}</strong>
          </div>
        </div>
        <van-button
          type="success"
          block
          :loading="withdrawSubmitting"
          :disabled="!isMerchant || !withdrawUsdt || Number(withdrawUsdt) < WITHDRAW_POWER_MIN"
          @click="handleWithdraw"
        >
        {{ t('powerPool.withdrawBtn') }}
        </van-button>
      </section> -->

      
    </div>

    <!-- 最近充值记录弹窗（右上角 ico 点击打开） -->
    <van-popup
      v-model:show="showRechargeRecords"
      position="top"
      :style="{ width: '100%', height: '50vh', overflow: 'auto', padding: '12px 12px 0' }"
      :close-on-click-overlay="true"
    >
      <section class="records-card">
        <h3>最近5笔充值记录</h3>
        <div v-if="myRechargeRecords.length === 0" class="empty-records">
          暂无充值记录
        </div>
        <div v-else class="record-list">
          <div class="record-item" v-for="r in myRechargeRecords" :key="r.order_id">
            <div class="record-row">
              <span class="record-label">订单号</span>
              <span class="record-value mono">{{ r.order_number }}</span>
            </div>
            <div class="record-row">
              <span class="record-label">金额</span>
              <span class="record-value">{{ r.usdt_amount }} USDT</span>
            </div>
            <div class="record-row">
              <span class="record-label">状态</span>
              <span :class="getRechargeBadgeClass(r.status)">{{ r.status_text }}</span>
            </div>
            <div class="record-row" v-if="r.created_at">
              <span class="record-label">时间</span>
              <span class="record-value">{{ formatTime(r.created_at) }}</span>
            </div>
          </div>
        </div>
      </section>
    </van-popup>

    <!-- 充值订单弹窗 -->
    <van-dialog
      v-model:show="showRechargeDialog"
      :title="t('powerPool.dialogRechargeTitle')"
      :close-on-click-overlay="false"
      :show-confirm-button="false"
      width="90%"
    >
      <div class="recharge-dialog-content" v-if="rechargeOrderInfo">
        <div class="order-info">
          <div class="info-row">
            <span>{{ t('powerPool.dialogOrderNo') }}</span>
            <strong>{{ rechargeOrderInfo.order_number }}</strong>
          </div>
          <div class="info-row">
            <span>{{ t('powerPool.dialogOrderPower') }}</span>
            <strong>{{ rechargeOrderInfo.power_amount }}</strong>
          </div>
          <div class="info-row">
            <span>{{ t('powerPool.dialogOrderPay') }}</span>
            <strong>${{ rechargeOrderInfo.usdt_amount }}</strong>
          </div>
        </div>
        
        <div class="bsc-address-section">
          <div class="label">{{ t('powerPool.dialogBscLabel') }}</div>
          <div class="bsc-address">{{ rechargeOrderInfo.bsc_address }}</div>
          <van-button size="small" type="primary" @click="copyBscAddress">
            {{ t('powerPool.dialogCopyAddressBtn') }}
          </van-button>
        </div>

        <div class="qr-code-section" v-if="rechargeOrderInfo.bsc_address">
          <div class="label">{{ t('powerPool.dialogQrLabel') }}</div>
          <div class="qr-wrapper">
            <img :src="bscQrCode" alt="BSC QR" />
          </div>
        </div>

        <div class="tips-section">
          <!-- <p>{{ t('powerPool.dialogTip1') }}</p> -->
          <p class="important-tip">{{ t('powerPool.dialogTip1') }}</p>
          <p>
            {{ t('powerPool.dialogTip2Prefix') }}${{ rechargeOrderInfo.usdt_amount }}
          </p>
          <p>{{ t('powerPool.dialogTip3') }}</p>
          <p>
            {{ t('powerPool.dialogTip4Prefix') }}{{ getRechargeStatusText(rechargeOrderInfo.status) }}
          </p>
        </div>

        <div class="dialog-footer">
          <van-button block type="default" @click="showRechargeDialog = false">
            {{ t('powerPool.dialogCloseBtn') }}
          </van-button>
        </div>
      </div>
    </van-dialog>

    <!-- BSC 收款地址设置弹窗 -->
    <van-dialog
      v-model:show="showBscAddressDialog"
      title="设置BSC付款地址"
      show-cancel-button
      @confirm="saveBscAddress"
      @cancel="showBscAddressDialog = false"
    >
      <van-field
        v-model="bscAddressForm"
        label="BSC地址"
        placeholder="请输入您的BSC钱包付款地址（仅限BSC链 USDT）"
        clearable
      />
      <div style="margin-top: 8px; font-size: 12px; color: #969799;">
        地址通常以 0x 开头，长度为 42 位，请确认网络为 BSC（BEP20）。
      </div>
    </van-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { showSuccessToast, showFailToast, showConfirmDialog } from 'vant'
import { httpGet, httpPost } from '@/utils/http'
import QRCode from 'qrcode'

defineOptions({
  name: 'PowerPoolView'
})

const { t } = useI18n()

const router = useRouter()
const fixedPrice = 0.01

const powerBalance = ref(0)
const coinBalance = ref(0)
const isMerchant = ref(false) // 仅商户可买卖算力
const inputPower = ref('')
const submitting = ref(false)
const matchingId = ref(null)
const poolOrders = ref([])
const myOrders = ref([])
const myRechargeRecords = ref([])
const showRechargeRecords = ref(false)

// 算力池充值相关
const poolInfo = ref({
  total_power: 0,
  total_usdt: 0,
  buyback_usdt: 0
})
// 充值起步算力：5千
const RECHARGE_POWER_MIN = 5000
const rechargePower = ref('')
const rechargeSubmitting = ref(false)
const rechargeOrderId = ref(null)
const showRechargeDialog = ref(false)
const rechargeOrderInfo = ref(null)

// BSC 收款地址设置（在本页弹窗编辑，避免用户去个人中心）
const showBscAddressDialog = ref(false)
const bscAddressForm = ref('')

// 回购池提现（卖出算力）相关，起步1万算力
const WITHDRAW_POWER_MIN = 10000
const withdrawUsdt = ref('')
const withdrawSubmitting = ref(false)

// BSC二维码
const bscQrCode = ref('')

const destroyedPreview = computed(() => {
  const value = Number(inputPower.value) || 0
  return (value * 0.5).toFixed(2)
})

const listedPreview = computed(() => {
  const value = Number(inputPower.value) || 0
  return (value * 0.5).toFixed(2)
})

// 计算充值需要支付的$金额 ($100 = 10000 算力，即 $1 = 100 算力)
const rechargeUsdtAmount = computed(() => {
  const power = Number(rechargePower.value) || 0
  return power / 100.0
})

// 计算算力提现可获得的$（5折：0.005U/算力）
const calculateWithdrawUsdt = (power) => {
  if (!power || power <= 0) return 0
  return power * 0.005
}

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
      showFailToast(e.message || t('powerPool.loadProfileFail'))
      isMerchant.value = false
    })
}

// 加载算力池信息
const loadPoolInfo = () => {
  httpGet('/api/trade/powerPool/info')
    .then(res => {
      poolInfo.value = {
        total_power: res.data.total_power || 0,
        total_usdt: res.data.total_usdt || 0,
        buyback_usdt: res.data.buyback_usdt || 0
      }
    })
    .catch(e => {
      showFailToast(e.message || t('powerPool.loadPoolInfoFail'))
    })
}

// 处理算力池充值
const handleRecharge = () => {
  // if (!isMerchant.value) {
  //   showFailToast(t('powerPool.onlyMerchant'))
  //   return
  // }
  const power = Number(rechargePower.value)
  if (!power || power <= 0) {
    showFailToast(t('powerPool.rechargePowerInvalid'))
    return
  }
  if (power < RECHARGE_POWER_MIN) {
    showFailToast(t('powerPool.rechargePowerMin'))
    return
  }

  if (power > poolInfo.value.total_power) {
    showFailToast(t('powerPool.rechargePowerExceedPrefix') + poolInfo.value.total_power.toFixed(2))
    return
  }

  rechargeSubmitting.value = true
  httpPost('/api/trade/powerPool/recharge', {
    power_amount: power
  })
    .then(res => {
      showSuccessToast(t('powerPool.rechargeCreateSuccess'))
      rechargeOrderId.value = res.data.order_id
      rechargeOrderInfo.value = res.data
      showRechargeDialog.value = true
      rechargePower.value = ''
      
      // 生成BSC地址二维码
      if (res.data.bsc_address) {
        generateBscQrCode(res.data.bsc_address)
      }
      
      // 开始轮询订单状态
      startPollingRechargeOrder(res.data.order_id)
    })
    .catch(e => {
      const msg = e?.message || ''
      // 后端未设置 BSC 地址时的专用提示，引导用户在本页直接设置
      if (msg.includes('请先设置您的BSC地址')) {
        showFailToast(msg)
        bscAddressForm.value = ''
        showBscAddressDialog.value = true
        return
      }
      showFailToast(msg || t('powerPool.rechargeCreateFail'))
    })
    .finally(() => {
      rechargeSubmitting.value = false
    })
}

// 生成BSC地址二维码
const generateBscQrCode = (address) => {
  QRCode.toDataURL(address, { width: 200, height: 200, margin: 2 }, (error, url) => {
    if (error) {
      console.error(t('powerPool.qrGenerateFailPrefix'), error)
    } else {
      bscQrCode.value = url
    }
  })
}

// 保存 BSC 收款地址
const saveBscAddress = () => {
  const address = (bscAddressForm.value || '').trim()
  if (!address) {
    showFailToast('请输入BSC收款地址')
    return
  }
  // 简单校验：BSC 地址一般以 0x 开头，长度不少于 20
  if (!address.startsWith('0x') || address.length < 20) {
    showFailToast('请输入有效的BSC钱包地址')
    return
  }

  httpPost('/api/user/bindEthAddress', {
    eth_address: address
  })
    .then(() => {
      showSuccessToast('收款地址保存成功')
      showBscAddressDialog.value = false
    })
    .catch(e => {
      showFailToast('保存失败，' + (e?.message || '未知错误'))
    })
}

// 复制BSC地址
const copyBscAddress = () => {
  if (!rechargeOrderInfo.value || !rechargeOrderInfo.value.bsc_address) {
    return
  }
  
  if (navigator?.clipboard?.writeText) {
    navigator.clipboard.writeText(rechargeOrderInfo.value.bsc_address)
      .then(() => {
        showSuccessToast(t('powerPool.copyAddressSuccess'))
      })
      .catch(() => {
        showFailToast(t('powerPool.copyAddressFail'))
      })
  } else {
    // 降级方案
    const textArea = document.createElement('textarea')
    textArea.value = rechargeOrderInfo.value.bsc_address
    textArea.style.position = 'fixed'
    textArea.style.left = '-999999px'
    document.body.appendChild(textArea)
    textArea.focus()
    textArea.select()
    try {
      document.execCommand('copy')
      showSuccessToast(t('powerPool.copyAddressSuccess'))
    } catch (err) {
      showFailToast(t('powerPool.copyAddressFail'))
    }
    document.body.removeChild(textArea)
  }
}

// 获取充值订单状态文本
const getRechargeStatusText = (status) => {
  const statusMap = {
    1: t('powerPool.rechargeStatusPending'),
    2: t('powerPool.rechargeStatusPaid'),
    3: t('powerPool.rechargeStatusDone'),
    4: t('powerPool.rechargeStatusCanceled')
  }
  return statusMap[status] || t('powerPool.rechargeStatusUnknown')
}

// 状态徽标：待支付 vs 已支付（含已完成）
const getRechargeBadgeClass = (status) => {
  if (status === 1) return 'badge badge-pending'
  if (status === 2 || status === 3) return 'badge badge-paid'
  if (status === 4) return 'badge badge-canceled'
  return 'badge badge-unknown'
}

// 轮询充值订单状态
let rechargePollingTimer = null
const startPollingRechargeOrder = (orderId) => {
  // 清除之前的定时器
  if (rechargePollingTimer) {
    clearInterval(rechargePollingTimer)
  }

  // 每10秒查询一次订单状态
  rechargePollingTimer = setInterval(() => {
    httpGet(`/api/trade/powerPool/recharge/order?order_id=${orderId}`)
      .then(res => {
        const status = res.data.status
        if (status === 2 || status === 3) {
          // 已支付或已完成
          clearInterval(rechargePollingTimer)
          showSuccessToast(t('powerPool.rechargeSuccess'))
          showRechargeDialog.value = false
          refreshAll()
        } else if (status === 4) {
          // 已取消
          clearInterval(rechargePollingTimer)
          showFailToast(t('powerPool.rechargeOrderCanceled'))
          showRechargeDialog.value = false
        }
      })
      .catch(e => {
        console.error(t('powerPool.rechargeQueryFailPrefix'), e)
      })
  }, 10000) // 10秒查询一次

  // 30分钟后停止轮询
  setTimeout(() => {
    if (rechargePollingTimer) {
      clearInterval(rechargePollingTimer)
    }
  }, 30 * 60 * 1000)
}

// 处理算力提现（5折）
const handleWithdraw = () => {
  // if (!isMerchant.value) {
  //   showFailToast(t('powerPool.onlyMerchant'))
  //   return
  // }
  const power = Number(withdrawUsdt.value)
  if (!power || power <= 0) {
    showFailToast(t('powerPool.withdrawPowerInvalid'))
    return
  }
  if (power < WITHDRAW_POWER_MIN) {
    showFailToast(t('powerPool.withdrawPowerMin'))
    return
  }

  const usdt = calculateWithdrawUsdt(power)
  if (usdt > poolInfo.value.buyback_usdt) {
    showFailToast(t('powerPool.withdrawPoolNotEnoughPrefix') + poolInfo.value.buyback_usdt.toFixed(2))
    return
  }

  // 校验可用算力
  if (power > powerBalance.value) {
    showFailToast(
      t('powerPool.withdrawPowerNotEnoughPrefix') +
      power.toFixed(2) +
      `，${t('powerPool.statMyPower')}：` +
      powerBalance.value.toFixed(2)
    )
    return
  }

  withdrawSubmitting.value = true
  httpPost('/api/trade/powerPool/withdraw', {
    usdt_amount: usdt
  })
    .then(res => {
      showSuccessToast(t('powerPool.withdrawSubmitSuccess'))
      withdrawUsdt.value = ''
      refreshAll()
    })
    .catch(e => {
      showFailToast(e.message || t('powerPool.withdrawSubmitFail'))
    })
    .finally(() => {
      withdrawSubmitting.value = false
    })
}

const loadPoolOrders = () => {
  httpGet('/api/trade/powerPool/orders')
    .then(res => {
      poolOrders.value = Array.isArray(res.data) ? res.data : []
    })
    .catch(e => {
      showFailToast(e.message || t('powerPool.loadPoolOrdersFail'))
    })
}

const loadMyOrders = () => {
  httpGet('/api/trade/powerPool/mine')
    .then(res => {
      myOrders.value = Array.isArray(res.data) ? res.data : []
    })
    .catch(e => {
      showFailToast(e.message || t('powerPool.loadMyOrdersFail'))
    })
}

// 加载当前用户最近 5 笔充值记录
const loadMyRechargeRecords = () => {
  httpGet('/api/trade/powerPool/recharge/mine')
    .then(res => {
      myRechargeRecords.value = res?.data?.items && Array.isArray(res.data.items) ? res.data.items : []
    })
    .catch(e => {
      console.error('loadMyRechargeRecords error:', e)
    })
}

const refreshAll = () => {
  loadProfile()
  loadPoolInfo()
  loadPoolOrders()
  loadMyOrders()
  loadMyRechargeRecords()
}

const toggleRechargeRecords = () => {
  showRechargeRecords.value = !showRechargeRecords.value
  // 打开时刷新一次，确保是最新
  if (showRechargeRecords.value) {
    loadMyRechargeRecords()
  }
}

const mineOrder = () => {

  router.push('/mobile/powertrade');

}



const handleSubmit = () => {
  const value = Number(inputPower.value)
  if (!value || value <= 0) {
    showFailToast(t('powerPool.rechargePowerInvalid'))
    return
  }

  

  if (value > powerBalance.value) {
    showFailToast(t('powerPool.withdrawPowerNotEnoughPrefix') + value.toFixed(2))
    return
  }

  //最小1万算力,最多20万个算力挂卖
  if (value>200000 || value<10000){
    // 简单用两条提示拼起来，避免再加 key
    showFailToast('算力数量必须在1万到20万之间')
    return
  }

  if (value > powerBalance.value) {
    showFailToast(t('powerPool.withdrawPowerNotEnoughPrefix') + value.toFixed(2))
    return
  }


  showConfirmDialog({
    title: t('powerPool.confirmSellTitle'),
    message: `${t('powerPool.confirmSellMessagePrefix')}${destroyedPreview.value}` +
      ` 算力，${t('powerPool.confirmSellMessageMid')}${fixedPrice.toFixed(2)}` +
      `${t('powerPool.confirmSellMessageSuffix')}`
  }).then(() => submitOrder(value))
}

const submitOrder = (value) => {
  submitting.value = true
  httpPost('/api/trade/powerPool/submit', { power: value })
    .then(() => {
      showSuccessToast(t('powerPool.submitOrderSuccess'))
      inputPower.value = ''
      refreshAll()
    })
    .catch(e => {
      showFailToast(e.message || t('powerPool.submitOrderFail'))
    })
    .finally(() => {
      submitting.value = false
    })
}

const handleMatch = (order) => {
  if (!order || !order.id) {
    showFailToast(t('powerPool.orderInfoError'))
    return
  }

  showConfirmDialog({
    title: t('powerPool.matchConfirmTitle'),
    message: `${t('powerPool.matchConfirmMessagePrefix')}${order.orderNumber}?`
  })
    .then(() => executeMatch(order.id))
    .catch(() => {})
}

const executeMatch = (orderId) => {
  matchingId.value = orderId
  httpPost('/api/trade/powerPool/match', { id: orderId })
    .then(() => {
      showSuccessToast(t('powerPool.matchSuccess'))
      refreshAll()
    })
    .catch(e => {
      showFailToast(e.message || t('powerPool.matchFail'))
    })
    .finally(() => {
      matchingId.value = null
    })
}

onMounted(() => {
  refreshAll()
})

// 组件卸载时清除定时器
onUnmounted(() => {
  if (rechargePollingTimer) {
    clearInterval(rechargePollingTimer)
  }
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
    
    @media (max-width: 768px) {
      grid-template-columns: repeat(2, 1fr);
    }
    
    @media (max-width: 480px) {
      grid-template-columns: repeat(2, 1fr);
    }

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

    .merchant-only-tip {
      margin-bottom: 10px;
      padding: 8px 12px;
      font-size: 13px;
      color: #e6a23c;
      background: #fdf6ec;
      border-radius: 8px;
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

  .records-card {
    h3 {
      margin: 0 0 12px;
      font-size: 16px;
    }

    .empty-records {
      padding: 12px;
      background: #fff;
      border-radius: 12px;
      color: #909399;
      font-size: 13px;
    }

    .record-list {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .record-item {
      background: #fff;
      border-radius: 12px;
      padding: 12px;
      box-shadow: 0 6px 18px rgba(0, 0, 0, 0.04);
    }

    .record-row {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      font-size: 13px;
      color: #606266;
      margin-bottom: 8px;

      &:last-child {
        margin-bottom: 0;
      }
    }

    .record-label {
      color: #909399;
      flex-shrink: 0;
    }

    .record-value {
      color: #303133;
      font-weight: 600;
      text-align: right;
      word-break: break-all;
    }

    .mono {
      font-family: 'JetBrains Mono', monospace;
      font-size: 12px;
    }

    .badge {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      padding: 2px 10px;
      border-radius: 999px;
      font-size: 12px;
      font-weight: 700;
      line-height: 18px;
      word-break: keep-all;
      white-space: nowrap;
    }

    .badge-pending {
      color: #E6A23C;
      background: rgba(230, 162, 60, 0.12);
      border: 1px solid rgba(230, 162, 60, 0.35);
    }

    .badge-paid {
      color: #19A15B;
      background: rgba(25, 161, 91, 0.12);
      border: 1px solid rgba(25, 161, 91, 0.35);
    }

    .badge-canceled {
      color: #F56C6C;
      background: rgba(245, 108, 108, 0.12);
      border: 1px solid rgba(245, 108, 108, 0.35);
    }

    .badge-unknown {
      color: #909399;
      background: rgba(144, 147, 153, 0.12);
      border: 1px solid rgba(144, 147, 153, 0.35);
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

  .recharge-dialog-content {
    padding: 16px;

    .order-info {
      margin-bottom: 16px;
      padding: 12px;
      background: #f5f7fb;
      border-radius: 8px;

      .info-row {
        display: flex;
        justify-content: space-between;
        margin-bottom: 8px;
        font-size: 14px;

        &:last-child {
          margin-bottom: 0;
        }

        strong {
          color: #303133;
          font-weight: 600;
        }
      }
    }

    .bsc-address-section {
      margin-bottom: 16px;

      .label {
        font-size: 14px;
        color: #606266;
        margin-bottom: 8px;
      }

      .bsc-address {
        padding: 12px;
        background: #f5f7fb;
        border-radius: 8px;
        word-break: break-all;
        font-family: 'JetBrains Mono', monospace;
        font-size: 12px;
        margin-bottom: 8px;
        color: #303133;
      }
    }

    .qr-code-section {
      margin-bottom: 16px;
      text-align: center;

      .label {
        font-size: 14px;
        color: #606266;
        margin-bottom: 8px;
      }

      .qr-wrapper {
        display: inline-block;
        padding: 12px;
        background: #fff;
        border-radius: 8px;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);

        img {
          width: 200px;
          height: 200px;
          display: block;
        }
      }
    }

    .tips-section {
      margin-bottom: 16px;
      padding: 12px;
      background: #fff7e6;
      border-radius: 8px;
      border-left: 3px solid #ff9800;

      p {
        margin: 4px 0;
        font-size: 12px;
        color: #666;
        line-height: 1.5;

        &:first-child {
          margin-top: 0;
        }

        &:last-child {
          margin-bottom: 0;
        }
      }

      .important-tip {
        color: #ee0a24;
        font-weight: 700;
      }
    }

    .dialog-footer {
      margin-top: 16px;
    }
  }

  .recharge-records-ico {
    color: #4facfe;
    font-size: 18px;
  }
}
</style>

