<template>
  <div class="mobile-image container">
    <van-tabs v-model:active="activeName" class="my-tab" animated sticky>




      <van-tab title="算力套餐" name="cent">
        <Profile/>
      </van-tab>

      <van-tab title="进行中算力" name="ing">
        <Ing/>
      </van-tab>
      <!-- <van-tab title="Stable-Diffusion" name="sd">
        <image-sd/>
      </van-tab> -->
      <van-tab title="已结束算力" name="end">
        <End/>
      </van-tab>
    </van-tabs>
  </div>
</template>

<script setup>
import {ref} from "vue";
import Profile from "@/views/mobile/Profile.vue";
import Ing from "@/views/mobile/Ing.vue";
import End from "@/views/mobile/End.vue";
// import ImageSd from "@/views/mobile/ImageSd.vue";

const activeName = ref("cent")
</script>

<style lang="stylus">
.mobile-image {
  .my-tab {
    .van-tab__panel {
      padding 1px
    }
  }
}
</style>