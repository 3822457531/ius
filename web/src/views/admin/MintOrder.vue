<template>
  <div class="container mint-order" v-loading="loading">
    <div class="handle-box">
      <el-input v-model="query.username" placeholder="用户名" class="handle-input mr10" clearable />
      <el-input v-model="query.order_no" placeholder="订单号" class="handle-input mr10" clearable />
      <el-select v-model="query.status" placeholder="状态" style="width: 120px">
        <el-option
          v-for="item in statusOptions"
          :key="item.value"
          :label="item.label"
          :value="item.value"
        />
      </el-select>
      <el-button type="primary" :icon="Search" @click="fetchData">搜索</el-button>
    </div>

    <el-row>
      <el-table :data="items" :row-key="row => row.id" table-layout="auto">
        <el-table-column prop="order_no" label="订单号" min-width="160" />
        <el-table-column prop="username" label="用户" width="120" />
        <el-table-column prop="subject" label="算力机名称" min-width="120" />
        <el-table-column prop="power" label="投入算力" width="100" />
        <el-table-column prop="js_coin" label="已产出 SOHA" width="120" />
        <el-table-column prop="fee" label="产出系数" width="100" />
        <el-table-column label="状态" width="100">
          <template #default="scope">
            <el-tag v-if="scope.row.status === 8" type="success">运行中</el-tag>
            <el-tag v-else-if="scope.row.status === 9" type="info">已关闭</el-tag>
            <el-tag v-else type="warning">{{ scope.row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="激活时间" width="170">
          <template #default="scope">
            <span>{{ dateFormat(scope.row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="scope">
            <el-popconfirm
              v-if="scope.row.status === 8"
              title="关闭后该算力机将不再产出 SOHA，确定继续？"
              :width="280"
              @confirm="closeOrder(scope.row)"
            >
              <template #reference>
                <el-button size="small" type="warning">关闭</el-button>
              </template>
            </el-popconfirm>
            <span v-else class="closed-text">—</span>
          </template>
        </el-table-column>
      </el-table>
    </el-row>

    <div class="pagination">
      <el-pagination
        v-if="total > 0"
        background
        layout="total, prev, pager, next"
        :hide-on-single-page="true"
        v-model:current-page="page"
        v-model:page-size="pageSize"
        @current-change="fetchData"
        :total="total"
      />
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { httpGet, httpPost } from '@/utils/http'
import { ElMessage } from 'element-plus'
import { dateFormat } from '@/utils/libs'
import { Search } from '@element-plus/icons-vue'

const items = ref([])
const query = ref({ username: '', order_no: '', status: -1 })
const total = ref(0)
const page = ref(1)
const pageSize = ref(15)
const loading = ref(true)

const statusOptions = [
  { label: '全部', value: -1 },
  { label: '运行中', value: 8 },
  { label: '已关闭', value: 9 },
]

const fetchData = () => {
  loading.value = true
  httpPost('/api/admin/mintOrder/list', {
    username: query.value.username,
    order_no: query.value.order_no,
    status: query.value.status,
    page: page.value,
    page_size: pageSize.value,
  }).then((res) => {
    items.value = res.data.items || []
    total.value = res.data.total || 0
    loading.value = false
  }).catch((e) => {
    loading.value = false
    ElMessage.error('获取数据失败：' + e.message)
  })
}

const closeOrder = (row) => {
  httpGet('/api/admin/mintOrder/close?id=' + row.id).then(() => {
    ElMessage.success('已关闭，该算力机不再产出 SOHA')
    fetchData()
  }).catch((e) => {
    ElMessage.error('关闭失败：' + e.message)
  })
}

onMounted(() => {
  fetchData()
})
</script>

<style lang="stylus" scoped>
.mint-order {
  .handle-box {
    margin-bottom: 12px;
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    align-items: center;
  }

  .handle-input {
    width: 160px;
  }

  .closed-text {
    color: #909399;
  }

  .pagination {
    margin-top: 16px;
    display: flex;
    justify-content: flex-end;
  }
}
</style>
