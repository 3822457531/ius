<template>
  <div class="container" v-loading="loading">
    <el-alert
      type="warning"
      :closable="false"
      show-icon
      title="用于修复因超时误判为失败、但上游 Suno 实际已生成成功的音乐任务。修复后用户可在音乐应用中正常播放历史记录。"
      style="margin-bottom: 16px"
    />

    <el-descriptions :column="2" border style="margin-bottom: 16px">
      <el-descriptions-item label="待修复任务数">
        <el-tag type="warning">{{ failedCount }}</el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="上游查询">
        GET {ApiURL}/suno/fetch/{task_id}
      </el-descriptions-item>
    </el-descriptions>

    <el-button type="primary" :loading="repairing" @click="repairHistory">
      修复历史数据
    </el-button>
    <el-button plain :loading="loading" @click="loadStats">刷新统计</el-button>

    <el-divider v-if="lastResult" content-position="left">最近一次修复结果</el-divider>
    <el-descriptions v-if="lastResult" :column="3" border style="margin-bottom: 16px">
      <el-descriptions-item label="扫描总数">{{ lastResult.total }}</el-descriptions-item>
      <el-descriptions-item label="修复成功">
        <el-tag type="success">{{ lastResult.repaired }}</el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="跳过">{{ lastResult.skipped }}</el-descriptions-item>
    </el-descriptions>

    <el-table v-if="lastResult?.items?.length" :data="lastResult.items" border size="small">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="user_id" label="用户ID" width="90" />
      <el-table-column prop="task_id" label="Task ID" min-width="180" />
      <el-table-column label="结果" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'repaired' ? 'success' : 'info'" size="small">
            {{ row.status === 'repaired' ? '已修复' : '跳过' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="reason" label="说明" min-width="160" />
    </el-table>

    <el-alert
      v-if="lastResult?.errors?.length"
      type="error"
      :closable="false"
      show-icon
      style="margin-top: 16px"
      :title="`部分任务处理异常（${lastResult.errors.length} 条）`"
    >
      <div v-for="(msg, idx) in lastResult.errors" :key="idx">{{ msg }}</div>
    </el-alert>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { httpGet, httpPost } from '@/utils/http'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const repairing = ref(false)
const failedCount = ref(0)
const lastResult = ref(null)

async function loadStats() {
  loading.value = true
  try {
    const res = await httpGet('/api/admin/music/failed-stats')
    failedCount.value = res.data?.failed_count ?? 0
  } catch (e) {
    ElMessage.error('加载统计失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

async function repairHistory() {
  try {
    await ElMessageBox.confirm(
      `将扫描 ${failedCount.value} 条本地失败且无音频的记录，并从上游重新拉取。是否继续？`,
      '修复历史数据',
      { type: 'warning', confirmButtonText: '开始修复', cancelButtonText: '取消' }
    )
  } catch {
    return
  }

  repairing.value = true
  try {
    const res = await httpPost('/api/admin/music/repair-history', {})
    lastResult.value = res.data || null
    ElMessage.success(`修复完成：成功 ${lastResult.value?.repaired ?? 0} 条，跳过 ${lastResult.value?.skipped ?? 0} 条`)
    await loadStats()
  } catch (e) {
    ElMessage.error('修复失败: ' + e.message)
  } finally {
    repairing.value = false
  }
}

onMounted(loadStats)
</script>

<style scoped>
.container {
  padding: 12px;
}
</style>
