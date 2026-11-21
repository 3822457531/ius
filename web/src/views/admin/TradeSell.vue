<template>
  <div class="container order" v-loading="loading">
    <div class="handle-box">
      <el-select v-model="query.status" placeholder="交易状态" style="width: 120px; margin-right: 10px;">
        <el-option
            v-for="item in tradeStatus"
            :key="item.value"
            :label="item.label"
            :value="item.value"
        />
      </el-select>
      <el-input v-model="query.order_no" placeholder="订单编号" class="handle-input mr10"></el-input>
      <el-date-picker
        v-model="dateRange"
        type="daterange"
        range-separator="至"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        style="width: 300px; margin-right: 10px;"
      />
      <el-button type="primary" :icon="Search" @click="fetchData">搜索</el-button>
    </div>

    <el-row>
      <el-table :data="items" :row-key="row => row.id" table-layout="auto">
        <el-table-column prop="order_no" label="订单号"/>
        <el-table-column label="用户信息">
          <template #default="scope">
            <div>买家: {{ scope.row.username }}</div>
            <div>卖家: {{ scope.row.target_username }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="amount" label="数量"/>
        <el-table-column prop="unit_price" label="交易单价"/>
        <el-table-column prop="total_price" label="交易总额">
          <template #default="scope">
            {{ (scope.row.Number * scope.row.Price).toFixed(2) }}
          </template>
        </el-table-column>
        <el-table-column label="交易状态">
          <template #default="scope">
            <el-tag :type="getStatusType(scope.row.status)">
              {{ getStatusText(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间">
          <template #default="scope">
            <span>{{ dateFormat(scope.row.created_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="180">
          <template #default="scope">
            <el-button 
              v-if="scope.row.status === 1"
              size="small" 
              type="primary" 
              @click="confirmMatch(scope.row)"></el-button>
            <el-button 
              v-if="scope.row.status === 2"
              size="small" 
              type="success" 
              @click="confirmPayment(scope.row)">确认收款</el-button>
            <el-popconfirm title="确定要删除当前记录吗?" @confirm="remove(scope.row)">
              <template #reference>
                <el-button size="small" type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-row>

    <div class="pagination">
      <el-pagination v-if="total > 0" background
                     layout="total,prev, pager, next"
                     :hide-on-single-page="true"
                     v-model:current-page="page"
                     v-model:page-size="pageSize"
                     @current-change="fetchData()"
                     :total="total"/>
    </div>
  </div>
</template>

<script setup>
import {onMounted, ref} from "vue";
import {httpGet, httpPost} from "@/utils/http";
import {ElMessage} from "element-plus";
import {dateFormat, removeArrayItem} from "@/utils/libs";
import {Search} from "@element-plus/icons-vue";

// 变量定义
const items = ref([])
const query = ref({
    order_no: "", 
    status: "",
    pay_time: [],
    page: 1,
    page_size: 15
})
const dateRange = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(15)
const loading = ref(true)
const tradeStatus = ref([
  {value: "", label: "全部状态"},
  {value: 1, label: "等待匹配"},
  {value: 2, label: "等待收款"},
  {value: 3, label: "交易完成"}
])

const getStatusType = (status) => {
  switch(status) {
    case 1: return "info"
    case 2: return "warning"
    case 3: return "success"
    default: return "danger"
  }
}

const getStatusText = (status) => {
  switch(status) {
    case 1: return "等待匹配"
    case 2: return "等待收款"
    case 3: return "交易完成"
    default: return "未知状态"
  }
}

onMounted(() => {
  fetchData()
})

// 获取数据
const fetchData = () => {
    // 处理查询参数
    const params = {
        order_no: query.value.order_no || "",
        status: query.value.status === "" ? null : Number(query.value.status),
        pay_time: [],
        page: Number(page.value) || 1,
        page_size: Number(pageSize.value) || 15
    }
    
    // 处理日期范围
    if (dateRange.value && dateRange.value.length === 2) {
        params.pay_time = dateRange.value.map(date => dateFormat(date, 'YYYY-MM-DD'))
    }

    loading.value = true
    console.log('Sending params:', params)
    
    httpPost('/api/admin/trade/listBuy', params)
        .then((res) => {
            if (res.data) {
                items.value = res.data.items
                total.value = res.data.total
                page.value = res.data.page
                pageSize.value = res.data.page_size
            }
        })
        .catch(e => {
            ElMessage.error("获取数据失败：" + e.message)
        })
        .finally(() => {
            loading.value = false
        })
}

const remove = function (row) {
  httpGet('/api/admin/order/remove?id=' + row.id).then(() => {
    ElMessage.success("删除成功！")
    items.value = removeArrayItem(items.value, row, (v1, v2) => {
      return v1.id === v2.id
    })
  }).catch((e) => {
    ElMessage.error("删除失败：" + e.message)
  })
}

const confirmMatch = function (row) {
  httpGet('/api/admin/order/match?id=' + row.id).then(() => {
    ElMessage.success("匹配成功！")
    fetchData()
  }).catch((e) => {
    ElMessage.error("匹配失败：" + e.message)
  })
}

const confirmPayment = function (row) {
  httpGet('/api/admin/order/payment?id=' + row.id).then(() => {
    ElMessage.success("确认收款成功！")
    fetchData()
  }).catch((e) => {
    ElMessage.error("确认收款失败：" + e.message)
  })
}

</script>

<style lang="stylus" scoped>
.order {

  .handle-box {
    .handle-input {
      max-width 150px;
      margin-right 10px;
    }
  }

  .opt-box {
    padding-bottom: 10px;
    display flex;
    justify-content flex-end

    .el-icon {
      margin-right: 5px;
    }
  }

  .el-select {
    width: 100%
  }

}
</style>