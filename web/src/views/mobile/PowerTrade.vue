<template>
  <div class="power-trade-page">
    <van-nav-bar
      title="我的算力订单"
      left-arrow
      fixed
      placeholder
      @click-left="router.back()"
    />

    <div class="content">


      <!-- <section class="chart-card">
        <PriceChart />
      </section> -->

      <section class="orders-card">
        <div class="section-header">
          <h3>订单列表</h3>
          <van-button
            size="small"
            type="default"
            plain
            :loading="loading"
            @click="loadOrders"
          >
            刷新
          </van-button>
        </div>

        <van-tabs v-model:active="activeTab" shrink>
          <van-tab
            v-for="tab in tabs"
            :key="tab.name"
            :name="tab.name"
            :title="tab.title"
          >
            <van-skeleton v-if="loading" title :row="3" animated />
            <van-empty
              v-else-if="getOrdersByTab(tab.name).length === 0"
              description="暂无订单"
            />
            <div v-else class="order-list">
              <div
                v-for="order in getOrdersByTab(tab.name)"
                :key="order.id"
                class="order-card"
              >
                <div class="order-header">
                  <div class="title">
                    <span class="order-no">#{{ formatOrderNumber(order.orderNumber) }}</span>
                    <van-tag type="primary" size="small" plain>
                      {{ roleLabel(order.role) }}
                    </van-tag>
                  </div>
                  <van-tag
                    class="status-tag"
                    size="small"
                    :type="statusTypeMap[order.status] || 'default'"
                  >
                    {{ statusText(order.status) }}
                  </van-tag>
                </div>

                <div class="order-body">
                  <div class="row">
                    <span>算力数量</span>
                    <strong>{{ formatNumber(order.listedPower) }}</strong>
                  </div>
                  <div class="row">
                    <span>订单价值</span>
                    <strong>$ {{ formatNumber(order.valueUsd) }}</strong>
                  </div>
                  <div class="row">
                    <span>对手方</span>
                    <strong>{{ order.counterpartyName || '待匹配' }}</strong>
                  </div>
                  <!-- 收款地址 -->
                  <div class="row address-row" v-if="order.role === 'buyer' || order.paymentAddress">
                    <span>{{ paymentTypeLabel(order.paymentType) }}</span>
                    <div class="address-value">
                      <strong class="address-text">{{ formatAddress(order.paymentAddress || order.ethAddress) }}</strong>
                      <van-button
                        size="mini"
                        type="primary"
                        plain
                        :disabled="!(order.paymentAddress || order.ethAddress)"
                        @click="copyPaymentAddress(order.paymentAddress || order.ethAddress)"
                      >
                        复制
                      </van-button>
                    </div>
                  </div>

                  <div class="row">
                    <span>时间进度</span>
                    <strong>{{ formatTime(orderLatestTime(order)) }}</strong>
                  </div>
                  <div class="row proof-row" v-if="order.proofUrl">
                    <span>付款凭证</span>
                    <van-button
                      size="mini"
                      type="primary"
                      plain
                      @click="previewProof(order.proofUrl)"
                    >
                      查看
                    </van-button>
                  </div>
                </div>

                <div class="order-actions" v-if="hasOrderActions(order)">

                  <van-button
                    v-if="canCancel(order)"
                    size="small"
                    type="warning"
                    plain
                    @click="cancelOrder(order)"
                  >
                    取消订单
                  </van-button>
                  
                  <van-button
                    v-if="canMarkPaid(order)"
                    size="small"
                    type="primary"
                    plain
                    @click="openPayDialog(order)"
                  >
                    我已付款
                  </van-button>
                  <van-button
                    v-if="canConfirmReceipt(order)"
                    size="small"
                    type="success"
                    @click="confirmReceipt(order)"
                  >
                    确认收款
                  </van-button>
                  <van-button
                    v-if="canAppeal(order)"
                    size="small"
                    type="danger"
                    plain
                    @click="startAppeal(order)"
                  >
                    申诉
                  </van-button>
                </div>

                <div class="timeline">
                  <div class="row">
                    <span>创建</span>
                    <strong>{{ formatTime(order.createdAt) }}</strong>
                  </div>
                  <div class="row" v-if="order.matchTime">
                    <span>撮合</span>
                    <strong>{{ formatTime(order.matchTime) }}</strong>
                  </div>
                  <div class="row" v-if="order.payTime">
                    <span>付款</span>
                    <strong>{{ formatTime(order.payTime) }}</strong>
                  </div>
                  <div class="row" v-if="order.finishTime">
                    <span>完成</span>
                    <strong>{{ formatTime(order.finishTime) }}</strong>
                  </div>
                </div>
              </div>
            </div>
          </van-tab>
        </van-tabs>
      </section>
    </div>

    <van-dialog
      v-model:show="showPayDialog"
      title="上传付款凭证"
      show-cancel-button
      confirm-button-text="完成"
      @cancel="handleDialogClose"
      @confirm="handleDialogClose"
    >
      <div class="upload-content">
        <van-uploader
          v-model="uploadImages"
          :max-count="1"
          :after-read="afterRead"
          :before-read="beforeRead"
          accept="image/*"
          :disabled="proofUploading"
        >
          <template #preview-cover="{ file }">
            <div class="preview-cover" v-if="file.status === 'uploading'">
              <van-loading type="spinner" size="20" />
              <span class="upload-text">{{ file.message }}</span>
            </div>
          </template>
        </van-uploader>
        <p class="upload-hint">
          图片上传完成后将自动提交“已付款”，请确保凭证清晰可辨。
        </p>
      </div>
    </van-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showFailToast, showSuccessToast, showConfirmDialog, showImagePreview } from 'vant'
import { httpGet, httpPost } from '@/utils/http'
import Compressor from 'compressorjs'
import PriceChart from '@/src/components/PriceChart.vue'

defineOptions({
  name: 'PowerTradeView'
})

const router = useRouter()
const tabs = [
  { name: 'processing', title: '交易中' },
  { name: 'completed', title: '已完成' },
  { name: 'cancelled', title: '已取消' },
  { name: 'all', title: '全部' }
]

const activeTab = ref('processing')
const loading = ref(false)
const orders = ref([])
const showPayDialog = ref(false)
const uploadImages = ref([])
const currentOrder = ref(null)
const proofUploading = ref(false)

const statusTypeMap = {
  1: 'warning',
  2: 'primary',
  3: 'primary',
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
      return '待确认'
    case 4:
      return '已完成'
    case 5:
      return '已取消'
    case 6:
      return '申诉中'
    default:
      return '未知状态'
  }
}

const formatOrderNumber = (value) => {
  if (!value) return '--'
  const str = String(value)
  if (str.length <= 12) return str
  return `${str.slice(0, 4)}...${str.slice(-4)}`
}

const roleLabel = (role) => (role === 'buyer' ? '我是买家' : '我是卖家')

const paymentTypeLabel = (type) => {
  switch ((type || 'bsc').toLowerCase()) {
    case 'alipay':
      return '支付宝账号'
    case 'bsc':
    default:
      return 'BSC地址'
  }
}

const formatNumber = (value) => {
  const num = Number(value)
  if (Number.isNaN(num)) return '0.00'
  return num.toFixed(2)
}

const formatAddress = (address) => {
  const value = address || ''
  if (value.length <= 18) return value
  return `${value.slice(0, 10)}...${value.slice(-6)}`
}

const formatTime = (ts) => {
  if (!ts) return '--'
  return new Date(ts * 1000).toLocaleString()
}

const orderLatestTime = (order) => {
  if (order.finishTime) return order.finishTime
  if (order.payTime) return order.payTime
  if (order.matchTime) return order.matchTime
  return order.createdAt
}

const canCancel = (order) => {
  if (order.status === 1) return true
  return order.role === 'buyer' && order.status === 2
}
const canMarkPaid = order => order.role === 'buyer' && order.status === 2
const canConfirmReceipt = order => order.role === 'seller' && order.status === 3
const canAppeal = order => order.role === 'seller' && order.status === 3
const hasOrderActions = order =>
  canCancel(order) || canMarkPaid(order) || canConfirmReceipt(order) || canAppeal(order)

const openPayDialog = (order) => {
  currentOrder.value = order
  uploadImages.value = []
  showPayDialog.value = true
}

const handleDialogClose = () => {
  showPayDialog.value = false
  currentOrder.value = null
  uploadImages.value = []
}

const previewProof = (url) => {
  if (!url) return
  showImagePreview([url])
}


const cancelOrder = (order) => {
  showConfirmDialog({
    title: '确认取消订单',
    message: '确认取消订单吗？',
  })
    .then(() => httpPost('/api/trade/powerTrade/cancel', { orderId: order.id }))
    .then(() => {
      showSuccessToast('取消订单成功')
      loadOrders()
    })
    .catch(e => {
      showFailToast(e.message || '操作失败')
    })
}

const markOrderPaid = (orderId, proofUrl) => {
  return httpPost('/api/trade/powerTrade/c2c/pay', {
    orderId,
    proofUrl
  })
    .then(() => {
      showSuccessToast('已标记付款，等待卖家确认')
      handleDialogClose()
      loadOrders()
    })
    .catch(e => {
      showFailToast(e.message || '操作失败')
    })
}

const beforeRead = (file) => {
  const isImage = file.type?.includes('image')
  if (!isImage) {
    showFailToast('请上传图片格式文件')
    return false
  }
  if (file.size > 10 * 1024 * 1024) {
    showFailToast('图片大小不能超过10M')
    return false
  }
  return true
}

const afterRead = (file) => {
  if (!currentOrder.value) {
    showFailToast('订单信息异常，请重新打开')
    return
  }
  file.status = 'uploading'
  file.message = '上传中...'
  proofUploading.value = true

  new Compressor(file.file, {
    quality: 0.6,
    success(result) {
      const formData = new FormData()
      formData.append('file', result, result.name || `proof_${Date.now()}.jpg`)

      httpPost('/api/upload', formData)
        .then(res => {
          const url = res.data?.url
          if (!url) {
            throw new Error('上传结果异常')
          }
          file.status = 'done'
          file.message = '上传成功'
          return markOrderPaid(currentOrder.value.id, url)
        })
        .catch(err => {
          file.status = 'failed'
          file.message = '上传失败'
          showFailToast(err.message || '上传失败')
        })
        .finally(() => {
          proofUploading.value = false
        })
    },
    error(err) {
      file.status = 'failed'
      file.message = '处理失败'
      proofUploading.value = false
      showFailToast(err.message || '图片处理失败')
    }
  })
}

const confirmReceipt = (order) => {
  showConfirmDialog({
    title: '确认收款',
    message: '请确认已收到买家的付款。',
  })
    .then(() => httpPost('/api/trade/powerTrade/c2c/confirm', { orderId: order.id }))
    .then(() => {
      showSuccessToast('确认收款成功')
      loadOrders()
    })
    .catch(e => {
      if (e?.message) {
        showFailToast(e.message)
      }
    })
}

const startAppeal = () => {
  showFailToast('申诉还在完善中')
}

const copyPaymentAddress = async (address) => {
  const text = address?.trim()
  if (!text) {
    showFailToast('暂无地址可复制')
    return
  }

  const fallbackCopy = () => {
    if (typeof document === 'undefined') return false
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.setAttribute('readonly', '')
    textarea.style.position = 'absolute'
    textarea.style.left = '-9999px'
    document.body.appendChild(textarea)
    textarea.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(textarea)
    return ok
  }

  try {
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
    } else if (!fallbackCopy()) {
      throw new Error('fallback failed')
    }
    showSuccessToast('地址已复制')
  } catch (err) {
    const success = fallbackCopy()
    if (success) {
      showSuccessToast('地址已复制')
    } else {
      showFailToast('复制失败，请手动复制')
    }
  }
}

const processingStatuses = [1, 2, 3]
const cancelledStatuses = [5, 6]

const getOrdersByTab = (name) => {
  const list = orders.value || []
  switch (name) {
    case 'processing':
      return list.filter(item => processingStatuses.includes(item.status))
    case 'completed':
      return list.filter(item => item.status === 4)
    case 'cancelled':
      return list.filter(item => cancelledStatuses.includes(item.status))
    default:
      return list
  }
}

const loadOrders = () => {
  loading.value = true
  httpGet('/api/trade/powerTrade/c2c/mine')
    .then(res => {
      orders.value = Array.isArray(res.data) ? res.data : []
    })
    .catch(e => {
      showFailToast(e.message || '获取订单失败')
    })
    .finally(() => {
      loading.value = false
    })
}

onMounted(() => {
  loadOrders()
})
</script>

<style scoped lang="scss">
.power-trade-page {
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

  .chart-card {
    padding: 0;

    :deep(.chart-container) {
      border-radius: 12px;
      overflow: hidden;
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

    .order-card {
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

        .title {
          display: flex;
            gap: 8px;
          align-items: center;
            flex-wrap: wrap;
          }

          .order-no {
            display: inline-block;
            max-width: 150px;
            font-family: 'JetBrains Mono', monospace;
            font-size: 13px;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .order-no {
          font-weight: 600;
        }

          .status-tag {
            margin-left: auto;
            flex-shrink: 0;
          }
      }

      .order-body,
      .timeline {
        display: flex;
        flex-direction: column;
        gap: 8px;
        font-size: 14px;

        .row {
          display: flex;
          justify-content: space-between;
          color: #606266;

          strong {
            color: #303133;
          }

          .van-button {
            min-width: 72px;
          }
        }

        .address-row {
          align-items: center;

          span {
            flex-shrink: 0;
          }

          .address-value {
            display: flex;
            align-items: center;
            gap: 8px;
            max-width: 70%;
          }

          .address-text {
            display: inline-block;
            max-width: 100%;
            font-family: 'JetBrains Mono', monospace;
            white-space: normal;
            word-break: break-all;
          }
        }

        .proof-row {
          align-items: center;
        }
      }

      .timeline {
        margin-top: 10px;
        padding-top: 10px;
        border-top: 1px dashed #ebeef5;
        font-size: 12px;
      }

      .order-actions {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        margin: 8px 0 4px;
      }
    }
  }
}

.upload-content {
  padding: 12px 0 4px;
  text-align: center;

  .van-uploader {
    display: inline-flex;

    .van-uploader__wrapper {
      width: 180px;
      height: 180px;
    }

    .van-uploader__upload,
    .van-uploader__preview {
      width: 180px;
      height: 180px;
    }
  }

  .preview-cover {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    color: #fff;
    background: rgba(0, 0, 0, 0.5);

    .upload-text {
      margin-top: 8px;
      font-size: 12px;
    }
  }

  .upload-hint {
    margin-top: 12px;
    font-size: 12px;
    color: #909399;
    line-height: 1.4;
  }
}
</style>
