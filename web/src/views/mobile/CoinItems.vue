<template>
  <div class="coin-items-page custom-scroll">
    <div class="gradient-bg"></div>

    <van-nav-bar
      :title="navTitle"
      class="custom-nav-bar"
      left-arrow
      @click-left="router.back()"
    />

    <div class="layout">
      <div class="glass-card section-card" v-if="haveOrder">
        <div class="subject-line" v-if="machineName">{{ machineName }}</div>

        <div class="table">
          <div class="table-header">
            <div class="col index">序号</div>
            <div class="col amount">产出</div>
            <div class="col balance">余额</div>
            <div class="col time">时间</div>
          </div>

          <div
            class="table-row"
            v-for="(item, index) in filteredOrders"
            :key="item.id || (item.created_at + '-' + index)"
          >
            <div class="col index">{{ index + 1 }}</div>
            <div class="col amount plus">+{{ item.amount }}</div>
            <div class="col balance">{{ formatBalance(item.balance) }}</div>
            <div class="col time">{{ dateFormat(item.created_at, 'MM-dd HH:mm:ss') }}</div>
          </div>
        </div>
      </div>

      <div class="glass-card empty-card" v-else-if="!loading">
        <van-empty
          image="https://fastly.jsdelivr.net/npm/@vant/assets/custom-empty-image.png"
          image-size="80"
          description="暂无记录"
        />
      </div>

      <div class="bottom-spacing"></div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import { showFailToast } from 'vant'
import { httpGet } from '@/utils/http'
import { checkSession } from '@/action/session'
import { useRoute, useRouter } from 'vue-router'
import { dateFormat } from '@/utils/libs'

defineOptions({ name: 'CoinItems' })

const router = useRouter()
const route = useRoute()
const loading = ref(false)
const orders = ref([])
const haveOrder = ref(false)
const productId = ref(route.query.productId)
const filterType = ref('all')

const typeMapping = {
  static: 11,
  recommend: 12,
}

const machineName = computed(() => {
  const name = route.query.title
  return name ? decodeURIComponent(String(name)) : ''
})

const navTitle = computed(() => machineName.value || '算力机明细')

const filteredOrders = computed(() => {
  if (filterType.value === 'all') return orders.value
  return orders.value.filter((order) => order.type === typeMapping[filterType.value])
})

const formatBalance = (n) => {
  const num = Number(n)
  if (!Number.isFinite(num)) return '--'
  return num
}

onMounted(() => {
  loading.value = true
  checkSession()
    .then(() => {
      const url = `/api/mint/getCoinItems?productId=${productId.value}`
      return httpGet(url)
        .then((res) => {
          orders.value = res.data || []
          haveOrder.value = orders.value.length > 0
        })
        .catch((e) => {
          showFailToast('获取收益详情失败：' + e.message)
        })
    })
    .catch(() => {
      router.push('/login')
    })
    .finally(() => {
      loading.value = false
    })
})
</script>

<style lang="stylus" scoped>
.coin-items-page
  position relative
  min-height 100vh
  min-height 100dvh
  background #f3f5f8
  overflow-x hidden

.gradient-bg
  position fixed
  inset 0
  background linear-gradient(135deg, rgba(102, 126, 234, 0.12) 0%, rgba(79, 172, 254, 0.1) 45%, rgba(0, 242, 254, 0.08) 100%)
  pointer-events none
  z-index 0

.custom-nav-bar
  position relative
  z-index 2
  background rgba(255, 255, 255, 0.92) !important
  :deep(.van-nav-bar__title)
    font-weight 700
    color #1f2937

.layout
  position relative
  z-index 1
  max-width 1120px
  width 100%
  margin 0 auto
  padding 14px 12px 70px
  box-sizing border-box

.glass-card
  background rgba(255, 255, 255, 0.96)
  border 1px solid rgba(79, 172, 254, 0.15)
  box-shadow 0 8px 24px rgba(0, 0, 0, 0.06)
  border-radius 20px
  overflow hidden

.section-card
  padding 14px 16px

.subject-line
  font-size 15px
  font-weight 700
  color #2c3e50
  margin-bottom 12px
  padding 10px 12px
  background rgba(255, 255, 255, 0.7)
  border 1px solid rgba(79, 172, 254, 0.15)
  border-radius 14px

.table
  width 100%
  overflow hidden
  border-radius 16px
  border 1px solid rgba(79, 172, 254, 0.12)
  background rgba(255, 255, 255, 0.85)

.table-header
  display grid
  grid-template-columns 0.45fr 0.85fr 0.95fr 1.15fr
  gap 6px
  padding 12px 10px
  background rgba(22, 119, 255, 0.08)
  border-bottom 1px solid rgba(79, 172, 254, 0.12)
  font-size 12px
  font-weight 700
  color #2c3e50

.table-row
  display grid
  grid-template-columns 0.45fr 0.85fr 0.95fr 1.15fr
  gap 6px
  padding 12px 10px
  border-bottom 1px solid rgba(79, 172, 254, 0.08)
  align-items center
  &:last-child
    border-bottom none
  &:nth-child(even)
    background rgba(22, 119, 255, 0.02)

.col
  min-width 0
  font-size 12px
  color #2c3e50

.index
  font-weight 600
  color #6b7280

.amount
  font-size 13px
  font-weight 700
  text-align right
  font-variant-numeric tabular-nums
  &.plus
    color #07c160

.balance
  font-size 13px
  font-weight 600
  color #1677ff
  text-align right
  font-variant-numeric tabular-nums
  word-break break-all

.time
  font-size 12px
  font-weight 500
  color #6b7280
  text-align right
  font-variant-numeric tabular-nums
  white-space nowrap

.empty-card
  padding 40px 20px
  display flex
  align-items center
  justify-content center
  min-height 300px

.bottom-spacing
  height 20px
</style>
