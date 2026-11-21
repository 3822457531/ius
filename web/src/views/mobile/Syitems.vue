<template>
  <div class="syitems-page custom-scroll">
    <div class="gradient-bg"></div>
    
    <van-nav-bar
      :title="title"
      class="custom-nav-bar"
      left-arrow
      @click-left="onClickBack"
    />
    
    <div class="layout">

<!--       
      <div class="glass-card filter-card">
        <van-dropdown-menu v-model="filterType" class="custom-dropdown">
          <van-dropdown-item v-model="filterType" :options="typeOptions" />
        </van-dropdown-menu>
      </div> -->

      <div class="glass-card section-card" v-if="haveOrder">
        <div class="subject-line" v-if="orderSubject">{{ orderSubject }}</div>

        <div class="table">
          <div class="table-header">
            <div class="col model">{{ t('syitems.columnType') }}</div>
            <div class="col amount">{{ t('syitems.columnAmount') }}</div>
            <div class="col balance">{{ t('syitems.columnBalance') }}</div>
            <div class="col time">{{ t('syitems.columnTime') }}</div>
          </div>

          <div class="table-row" v-for="(item, index) in filteredOrders" :key="item.id || (item.created_at + '-' + index)">
            <div class="col model">
              <!-- <div class="model-text">{{ item.model || typeText(item.type) }}</div> -->
              <div class="type-tag" :class="typeTagClass(item)">
                {{ typeText(item) }}
              </div>
            </div>

            <div class="col amount" :class="amountClass(item)">
              {{ formatAmount(item) }}
            </div>

            <div class="col balance">
              {{ formatNumber(item.balance) }}
            </div>

            <div class="col time">
              {{ dateFormat(item['created_at'], 'MM-dd HH:mm:ss') }}
            </div>
          </div>
        </div>
      </div>
      
      <div class="glass-card empty-card" v-else>
        <van-empty
          image="https://fastly.jsdelivr.net/npm/@vant/assets/custom-empty-image.png"
          image-size="80"
          :description="t('syitems.emptyDesc')"
        />
      </div>

      <div class="bottom-spacing"></div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { httpGet, showFailToast } from "@/utils/http";
import { checkSession } from "@/action/session";
import { useRoute, useRouter } from "vue-router";
import { dateFormat } from "@/utils/libs";

const { t } = useI18n();

const title = computed(() => t("syitems.title"));
const form = ref({
  username: "",
  nickname: "",
  mobile: "",
  avatar: "",
  calls: 0,
  tokens: 0,
});
const orders = ref([]);
const haveOrder = ref(false);
const route = useRoute();
const router = useRouter();
const productId = ref(route.query.productId);
const filterType = ref("all"); // 默认筛选类型为全部
const typeOptions = [
  { text: "全部", value: "all" },
  { text: "静态释放", value: "static" },
  { text: "推荐释放", value: "recommend" },
  { text: "等级分红", value: "fenhong" },
]; // 筛选类型选项


// 类型映射：静态释放=11；推荐释放=12；等级分红=12 且 model=等级分红奖励
const typeMapping = {
  static: 11,
  recommend: 12,
  fenhong: 11,
};



onMounted(() => {
  checkSession()
    .then((user) => {
      // 获取用户算力套餐列表
      const url = `/api/order/getOrderItems?productId=${productId.value}`;
      // 获取当前产品的收益列表
      httpGet(url)
        .then((res) => {
          orders.value = res.data;
          if (res.data.length != 0) {
            haveOrder.value = true;
          }
        })
        .catch((e) => {
          showFailToast(t("syitems.loadFailPrefix") + e.message);
        });
    })
    .catch(() => {
      router.push("/login");
    });
});

// 过滤后的订单数据
// const filteredOrders = computed(() => {
//   if (filterType.value === "all") {
//     return orders.value;
//   }
//   return orders.value.filter((order) => order.type === filterType.value);
// });


const filteredOrders = computed(() => {
  if (filterType.value === "all") {
    return orders.value;
  }

  const targetType = typeMapping[filterType.value];
  const isFenHong = (order) => (order?.model || order?.Model) === "等级分红奖励";

  if (filterType.value === "fenhong") {
    return orders.value.filter((order) => order.type === targetType && isFenHong(order));
  }
  if (filterType.value === "recommend") {
    return orders.value.filter((order) => order.type === targetType);
  }

  return orders.value.filter((order) => order.type === targetType);
});

const orderSubject = computed(() => {
  // 后端为该 order_no 返回的流水里，username 字段存的是订单 subject（套餐名/标题）
  return orders.value?.[0]?.username || "";
});

const isFenHong = (item) => (item?.model || item?.Model) === "等级分红奖励";

const typeText = (item) => {
  if (isFenHong(item)) return t("syitems.typeFenhong");
  if (item?.type === 11) return t("syitems.typeStatic");
  if (item?.type === 12) return t("syitems.typeRecommend");
  return t("syitems.typeUnknown");
};

const typeTagClass = (item) => {
  if (isFenHong(item)) return { fenhong: true };
  if (item?.type === 11) return { static: true };
  if (item?.type === 12) return { recommend: true };
  return {};
};

const formatNumber = (n) => {
  if (n === null || n === undefined) return "-";
  const num = Number(n);
  if (Number.isNaN(num)) return "-";
  return String(num);
};

const amountClass = (item) => {
  const mark = item?.mark;
  if (mark === 0) return "minus";
  if (mark === 1) return "plus";
  // 默认按正数展示
  return "plus";
};

const formatAmount = (item) => {
  const num = formatNumber(item?.amount);
  if (num === "-") return "-";
  const sign = item?.mark === 0 ? "-" : "+";
  return `${sign}${num}`;
};

const onClickBack = () => {
  router.back();
};
</script>

<style lang="stylus" scoped>
@import "@/assets/css/custom-scroll.styl"

.syitems-page
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

.custom-nav-bar
  position relative
  z-index 1
  background rgba(255,255,255,0.95)
  backdrop-filter blur(10px)
  border-bottom 1px solid rgba(255,255,255,0.8)
  box-shadow 0 2px 8px rgba(0,0,0,0.08)
  :deep(.van-nav-bar__title)
    color #2c3e50
    font-weight 700
    font-size 18px
    letter-spacing 0.3px

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

.filter-card
  padding 12px 20px

.custom-dropdown
  :deep(.van-dropdown-menu__bar)
    background transparent
    box-shadow none
    border-radius 12px
  :deep(.van-dropdown-menu__item)
    background rgba(255,255,255,0.9)
    border-radius 12px
    border 1px solid rgba(79,172,254,0.2)
    box-shadow 0 2px 8px rgba(0,0,0,0.08)
    transition all 0.3s ease
    &:active
      transform scale(0.98)
  :deep(.van-dropdown-menu__title)
    color #2c3e50
    font-weight 600
    font-size 15px
    padding 10px 16px

.section-card
  padding 14px 16px

.subject-line
  font-size 15px
  font-weight 800
  color #2c3e50
  letter-spacing 0.2px
  margin-bottom 12px
  padding 10px 12px
  background rgba(255,255,255,0.7)
  border 1px solid rgba(79,172,254,0.15)
  border-radius 14px

.table
  width 100%
  overflow hidden
  border-radius 16px
  border 1px solid rgba(79,172,254,0.12)
  background rgba(255,255,255,0.75)

.table-header
  display grid
  grid-template-columns 1.4fr 0.7fr 0.7fr 1fr
  gap 10px
  padding 12px 12px
  background rgba(102,126,234,0.12)
  border-bottom 1px solid rgba(79,172,254,0.12)
  font-size 13px
  font-weight 800
  color #2c3e50

.table-row
  display grid
  grid-template-columns 1.4fr 0.7fr 0.7fr 1fr
  gap 10px
  padding 12px 12px
  border-bottom 1px solid rgba(79,172,254,0.08)
  align-items center
  &:last-child
    border-bottom none

.col
  min-width 0

.model-text
  font-size 13px
  font-weight 800
  color #2c3e50
  line-height 1.2
  word-break break-all

.type-tag
  display inline-block
  margin-top 6px
  padding 2px 8px
  border-radius 8px
  font-size 12px
  font-weight 700
  &.static
    background rgba(79,172,254,0.15)
    color #4facfe
    border 1px solid rgba(79,172,254,0.25)
  &.recommend
    background rgba(67,233,123,0.15)
    color #43e97b
    border 1px solid rgba(67,233,123,0.25)
  &.fenhong
    background rgba(255,159,67,0.18)
    color #ff9f43
    border 1px solid rgba(255,159,67,0.28)

.amount
  font-size 14px
  font-weight 900
  text-align right
  &.plus
    color #2ecc71
  &.minus
    color #e74c3c

.balance
  font-size 14px
  font-weight 800
  color #2c3e50
  text-align right

.time
  font-size 13px
  font-weight 600
  color #7f8c8d
  text-align right

.empty-card
  padding 40px 20px
  display flex
  align-items center
  justify-content center
  min-height 300px

:deep(.van-empty)
  .van-empty__description
    color #7f8c8d
    font-size 14px

.bottom-spacing
  height 70px

@media (max-width: 720px)
  .layout
    padding 14px 12px 70px
  .glass-card
    padding 16px
  .table-header, .table-row
    grid-template-columns 1.4fr 0.7fr 0.7fr 1fr
    padding 10px 10px
    gap 8px
  .model-text
    font-size 12px
  .amount, .balance
    font-size 13px
  .time
    font-size 12px
</style>
