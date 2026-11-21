<template>
  <div class="container" v-loading="loading">
    <el-alert
      type="info"
      :closable="false"
      show-icon
      title="关闭应用后，用户端将显示「维护中」且无法调用对应功能；开启后立即恢复正常。"
      style="margin-bottom: 16px"
    />

    <el-table :data="appRows" border style="margin-bottom: 16px">
      <el-table-column prop="name" label="应用" width="140" />
      <el-table-column prop="desc" label="说明" min-width="220" />
      <el-table-column label="状态" width="120" align="center">
        <template #default="{ row }">
          <el-tag :type="system[row.field] ? 'success' : 'warning'" size="small">
            {{ system[row.field] ? '已开启' : '维护中' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="开关" width="120" align="center">
        <template #default="{ row }">
          <el-switch v-model="system[row.field]" />
        </template>
      </el-table-column>
    </el-table>

    <el-button type="primary" :loading="saving" @click="save">保存配置</el-button>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { httpGet, httpPost } from '@/utils/http'
import { ElMessage } from 'element-plus'

const loading = ref(false)
const saving = ref(false)
const system = reactive({})

const appRows = [
  { key: 'image', name: 'AI 图片', field: 'enabled_app_image', desc: 'Image Hub 文生图 / 图生图' },
  { key: 'music', name: 'AI 音乐', field: 'enabled_app_music', desc: 'Suno 音乐生成' },
  { key: 'video', name: 'AI 视频', field: 'enabled_app_video', desc: 'Sora2 文生视频 / 图生视频' },
  { key: 'ppt', name: 'AI PPT', field: 'enabled_app_ppt', desc: 'PPT 子系统入口' },
  { key: 'logo', name: 'AI Logo', field: 'enabled_app_logo', desc: '品牌 Logo 生成' },
  { key: 'drama', name: '文字故事 / 短剧', field: 'enabled_app_drama', desc: '口播脚本、TTS、配图与视频合成' },
]

function applyDefaults(data) {
  if (data.enabled_app_image === undefined || data.enabled_app_image === null) {
    data.enabled_app_image = true
  }
  if (data.enabled_app_music === undefined || data.enabled_app_music === null) {
    data.enabled_app_music = true
  }
  if (data.enabled_app_video === undefined || data.enabled_app_video === null) {
    data.enabled_app_video = false
  }
  if (data.enabled_app_ppt === undefined || data.enabled_app_ppt === null) {
    data.enabled_app_ppt = true
  }
  if (data.enabled_app_logo === undefined || data.enabled_app_logo === null) {
    data.enabled_app_logo = true
  }
  if (data.enabled_app_drama === undefined || data.enabled_app_drama === null) {
    data.enabled_app_drama = true
  }
}

function load() {
  loading.value = true
  httpGet('/api/admin/config/get?key=system')
    .then((res) => {
      Object.assign(system, res.data || {})
      applyDefaults(system)
    })
    .catch((e) => {
      ElMessage.error('加载配置失败: ' + e.message)
    })
    .finally(() => {
      loading.value = false
    })
}

function save() {
  saving.value = true
  httpPost('/api/admin/config/update', { key: 'system', config: system })
    .then(() => {
      ElMessage.success('保存成功')
    })
    .catch((e) => {
      ElMessage.error('保存失败: ' + e.message)
    })
    .finally(() => {
      saving.value = false
    })
}

onMounted(load)
</script>

<style scoped>
.container {
  padding: 12px;
}
</style>
