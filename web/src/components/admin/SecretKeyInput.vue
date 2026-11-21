<template>
  <el-input
    :model-value="modelValue"
    @update:model-value="emit('update:modelValue', $event)"
    :type="visible ? 'text' : 'password'"
    :placeholder="placeholder"
    autocomplete="off"
  >
    <template #suffix>
      <el-icon
        class="secret-key-input__eye"
        :class="{ 'is-loading': loading }"
        @click="toggleVisible"
      >
        <View v-if="!visible" />
        <Hide v-else />
      </el-icon>
    </template>
  </el-input>
</template>

<script setup>
import { ref } from 'vue'
import { View, Hide } from '@element-plus/icons-vue'
import { httpGet } from '@/utils/http'
import { ElMessage } from 'element-plus'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '' },
  configKey: { type: String, required: true },
  field: { type: String, required: true },
})

const emit = defineEmits(['update:modelValue'])

const visible = ref(false)
const loading = ref(false)

function looksMasked(value) {
  const v = String(value || '').trim()
  if (!v) return true
  return /^[\*•·]+$/.test(v) || v.includes('****')
}

async function fetchSecretValue() {
  loading.value = true
  try {
    const res = await httpGet('/api/admin/config/secret', {
      key: props.configKey,
      field: props.field,
    })
    // 安全策略：接口只返回掩码 hint，不再下发明文
    const value = res.data?.hint || res.data?.value || ''
    if (res.data?.configured === false) {
      ElMessage.info('该密钥尚未配置')
    } else {
      ElMessage.success('已显示掩码（明文密钥不可通过接口回显）')
    }
    emit('update:modelValue', value)
    return value
  } catch (e) {
    ElMessage.error('读取密钥失败: ' + e.message)
    return props.modelValue
  } finally {
    loading.value = false
  }
}

async function toggleVisible() {
  if (!visible.value && looksMasked(props.modelValue)) {
    await fetchSecretValue()
  }
  visible.value = !visible.value
}
</script>

<style scoped>
.secret-key-input__eye {
  cursor: pointer;
  color: var(--el-text-color-secondary);
  font-size: 16px;
}

.secret-key-input__eye:hover {
  color: var(--el-color-primary);
}

.secret-key-input__eye.is-loading {
  pointer-events: none;
  opacity: 0.5;
}
</style>
