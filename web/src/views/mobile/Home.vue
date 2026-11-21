<template>
  <van-config-provider :theme="getMobileTheme()">
    <div class="mobile-home">
      <!--
        不要用 <transition mode="out-in"> 包裹 keep-alive，
        当前 Vue 版本下会触发 parentComponent.ctx.deactivate is not a function
      -->
      <router-view v-slot="{ Component }">
        <keep-alive :include="keepAlivePages">
          <component :is="Component" />
        </keep-alive>
      </router-view>

      <van-tabbar v-if="!hideTabbar" route v-model="active" @change="onChange" class="mobile-home__tabbar">
        <van-tabbar-item to="/mobile/chat/session" name="home" icon="chat-o">
          {{ t('home.tabChat') }}
        </van-tabbar-item>
        <van-tabbar-item to="/mobile/image" name="image" icon="photo-o">
          {{ t('home.tabApps') }}
        </van-tabbar-item>
        <van-tabbar-item to="/mobile/alliance" name="alliance" class="mobile-home__tab-ai">
          <span>{{ t('home.tabAlliance') }}</span>
          <template #icon="props">
            <span class="mobile-home__ai-fab" :class="{ 'is-active': props.active }">
              <van-icon name="friends-o" size="13" />
            </span>
          </template>
        </van-tabbar-item>
        <van-tabbar-item to="/mobile/img-wall" name="market" icon="chart-trending-o">
          {{ t('home.tabMarket') }}
        </van-tabbar-item>
        <van-tabbar-item to="/mobile/invitation" name="cent" icon="user-o">
          {{ t('home.tabMine') }}
        </van-tabbar-item>
      </van-tabbar>

    </div>
  </van-config-provider>

</template>

<script setup>
import {computed, ref, watch} from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import {getMobileTheme} from "@/store/system";
import {useRouter} from "vue-router";
import {isMobile} from "@/utils/libs";
import {checkSession} from "@/action/session";

const { t } = useI18n();

const router = useRouter()
const route = useRoute()
const hideTabbar = computed(() => route.matched.some((r) => r.meta?.hideTabbar))

// Tab 主页面缓存，切换时保留状态，接近原生 App 体验
const keepAlivePages = ['Invitation', 'Image', 'ImgWall', 'Profile', 'Alliance']

const tabNameByPath = [
  { prefix: '/mobile/chat', name: 'home' },
  { prefix: '/mobile/image', name: 'image' },
  { prefix: '/mobile/alliance', name: 'alliance' },
  { prefix: '/mobile/img-wall', name: 'market' },
  { prefix: '/mobile/invitation', name: 'cent' },
  { prefix: '/mobile/profile', name: 'cent' },
]

const active = ref('home')
const syncActiveTab = () => {
  const path = route.path || ''
  const hit = tabNameByPath.find((item) => path === item.prefix || path.startsWith(item.prefix + '/'))
  if (hit) active.value = hit.name
}
syncActiveTab()
watch(() => route.path, syncActiveTab)

checkSession().then(() => {
  if (!isMobile()) {
    router.replace('/invitation')
  }
}).catch(() => {
  router.push('/login')
})

const onChange = () => {}

</script>

<style lang="stylus">
@import '@/assets/iconfont/iconfont.css';
@import "@/assets/css/custom-scroll.styl"

.mobile-home
  position relative
  width 100%
  max-width 100%
  min-height 100vh
  min-height 100dvh
  background linear-gradient(135deg, #4facfe 0%, #00f2fe 50%, #43e97b 100%)
  overflow-x hidden
  overflow-y hidden
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

  .container
    position relative
    z-index 1
    .van-nav-bar
      position fixed
      width 100%
      background rgba(255,255,255,0.95)
      backdrop-filter blur(10px)
      border-bottom 1px solid rgba(255,255,255,0.8)

    .content
      padding 46px 10px 60px 10px

  :deep(.van-tabbar),
  :deep(.mobile-home__tabbar)
    position fixed !important
    left 0
    right 0
    bottom 0
    z-index 5000 !important
    pointer-events auto !important
    background rgba(255,255,255,0.96)
    backdrop-filter blur(10px)
    box-shadow 0 -4px 16px rgba(0,0,0,0.08)

  :deep(.van-tabbar-item)
    color #7f8c8d
    font-weight 600
    &.van-tabbar-item--active
      color #4facfe

  :deep(.mobile-home__tab-ai)
    // 不覆盖高度与下边距，与其余 tab 完全同一套图标盒度量，保证文字基线对齐
    .van-tabbar-item__icon
      display flex
      align-items center
      justify-content center
    .van-tabbar-item__text
      font-weight 600
      color #7f8c8d
    &.van-tabbar-item--active .van-tabbar-item__text
      color #4facfe

  // 联盟图标：与其它 tab 同尺寸、同基线，仅用淡色圆形做轻微区分
  .mobile-home__ai-fab
    width 22px
    height 22px
    border-radius 50%
    display flex
    align-items center
    justify-content center
    background rgba(79, 172, 254, 0.12)
    color #7f8c8d
    transition background 0.18s ease, color 0.18s ease
    &.is-active
      background #4facfe
      color #fff

// 黑色主题
.van-theme-dark body
  background #1c1c1e

.van-toast--fail
  background #fef0f0
  color #f56c6c

.van-toast--success
  background #D6FBCC
  color #07C160

.van-nav-bar
  position fixed
  width 100%
</style>
