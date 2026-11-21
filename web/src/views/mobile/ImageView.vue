<template>
  <div class="mobile-image container">
    <van-tabs v-model:active="activeName" class="my-tab" animated sticky>
      <van-tab title="对话" name="dall2">
        <chat-list />
      </van-tab>

      <van-tab title="Ai绘画" name="mj">
        <image-mj/>
      </van-tab>

      <van-tab title="数字人视频" name="dall">
        <digital/>
      </van-tab>
    </van-tabs>
  </div>
</template>

<script setup>
import {ref} from "vue";
import ImageMj from "@/views/mobile/ImageMj.vue";
import Digital from "@/views/mobile/Digital.vue";
import ChatList from "@/views/mobile/ChatList.vue";

const activeName = ref("dall2")
</script>

<style lang="stylus" scoped>
.mobile-image {
  .my-tab {
    .van-tab__panel {
      padding: 10px;
    }
  }
}
</style> 