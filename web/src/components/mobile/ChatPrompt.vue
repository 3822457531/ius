<template>
  <div class="mobile-message-prompt">
    <div class="chat-item">
      <div ref="contentRef" :data-clipboard-text="content" class="content" v-html="content"></div>
    </div>
  </div>
</template>

<script setup>
import {onMounted, ref} from "vue";
import Clipboard from "clipboard";
import {showNotify} from "vant";

const props = defineProps({
  content: {
    type: String,
    default: '',
  },
  icon: {
    type: String,
    default: '/images/user-icon.png',
  }
});
const contentRef = ref(null)
onMounted(() => {
  const clipboard = new Clipboard(contentRef.value);
  clipboard.on('success', () => {
    showNotify({type: 'success', message: '复制成功', duration: 1000})
  })
  clipboard.on('error', () => {
    showNotify({type: 'danger', message: '复制失败', duration: 2000})
  })
})
</script>

<style lang="stylus">
.mobile-message-prompt
  display flex
  justify-content flex-end
  width 100%

  .chat-item
    position relative
    max-width min(86%, 560px)
    overflow hidden

    .content
      word-break break-word
      text-align left
      padding 10px 14px
      background #ececf1
      color #111827
      font-size 15px
      line-height 1.58
      border-radius 18px
      font-family 'Plus Jakarta Sans', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif

.van-theme-dark
  .mobile-message-prompt
    .chat-item
      .content
        background-color #2f3033
        color #e5e5e5
</style>
