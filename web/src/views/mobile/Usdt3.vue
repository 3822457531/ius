<template>
  <div>
    {{ title }}
  </div>
  <div>
    id 为{{ $route.params.id }}
    login 为{{ $loginUser.id }}
  </div>
</template>

<script setup>

import {ref} from "vue";

const title = ref('Test Page')

</script>

<style lang="stylus" scoped>
</style>