<template>
  <div class="container coin-log" v-loading="loading">
    <div class="handle-box">
      <el-input v-model="query.username" placeholder="用户名" class="handle-input mr10" clearable></el-input>
      <el-input v-model="query.order_no" placeholder="算力机订单号" class="handle-input mr10" clearable></el-input>
      <el-input v-model="query.model" placeholder="类型" class="handle-input mr10" clearable></el-input>
      <el-date-picker
          v-model="query.date"
          type="daterange"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          format="YYYY-MM-DD"
          value-format="YYYY-MM-DD"
          style="margin: 0 10px;width: 200px; position: relative;top:3px;"
      />
      <el-button type="primary" :icon="Search" @click="search">搜索</el-button>
      <el-button v-if="totalCoin > 0">SOHA产出合计：{{ totalCoin }}</el-button>
    </div>

    <el-row v-if="items.length > 0">
      <el-table :data="items" :row-key="row => row.id" table-layout="auto" border>
        <el-table-column prop="username" label="用户"/>
        <el-table-column prop="order_no" label="算力机订单号"/>
        <el-table-column prop="model" label="类型"/>
        <el-table-column label="SOHA数量">
          <template #default="scope">
            <el-text type="success" v-if="scope.row.mark === 1">+{{ scope.row.amount }}</el-text>
            <el-text type="danger" v-else>-{{ scope.row.amount }}</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="balance" label="钱包余额"/>
        <el-table-column label="发生时间">
          <template #default="scope">
            <span>{{ dateFormat(scope.row['created_at']) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注"/>
      </el-table>

      <div class="pagination">
        <el-pagination v-if="total > 0" background
                       layout="total,prev, pager, next"
                       :hide-on-single-page="true"
                       v-model:current-page="page"
                       v-model:page-size="pageSize"
                       @current-change="fetchData()"
                       :total="total"/>
      </div>
    </el-row>
    <el-empty :image-size="100" v-else/>
  </div>
</template>

<script setup>
import {onMounted, ref} from "vue";
import {httpPost} from "@/utils/http";
import {ElMessage} from "element-plus";
import {dateFormat} from "@/utils/libs";
import {Search} from "@element-plus/icons-vue";

const items = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const totalCoin = ref(0)
const query = ref({
  username: "",
  order_no: "",
  model: "",
  date: [],
})

onMounted(() => {
  fetchData()
})

const search = () => {
  page.value = 1
  fetchData()
}

const fetchData = () => {
  loading.value = true
  httpPost('/api/admin/coinLog/list', {
    username: query.value.username,
    order_no: query.value.order_no,
    model: query.value.model,
    date: query.value.date,
    page: page.value,
    page_size: pageSize.value
  }).then((res) => {
    const data = res.data.data
    if (data) {
      items.value = data.items
      total.value = data.total
      page.value = data.page
      pageSize.value = data.page_size
    }
    totalCoin.value = res.data.stat
    loading.value = false
  }).catch(e => {
    loading.value = false
    ElMessage.error("获取数据失败：" + e.message);
  })
}
</script>

<style lang="stylus" scoped>
.coin-log {
  .handle-box {
    .handle-input {
      max-width 150px;
      margin-right 10px;
    }
  }
}
</style>
