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
        <el-table-column prop="orderNumber" label="订单号"/>
        <el-table-column label="用户信息">
          <template #default="scope">
            <div>买家: {{ scope.row.buyerName || '-' }}</div>
            <div>卖家: {{ scope.row.sellerName || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="listedPower" label="数量"/>
        <el-table-column prop="price" label="交易单价"/>

        <el-table-column prop="valueUsd" label="交易总额">
          <template #default="scope">
            ${{ Number(scope.row.valueUsd || 0).toFixed(2) }}
          </template>
        </el-table-column> 

        <el-table-column prop="proofUrl" label="交易凭证">
          <template #default="scope">
            <div v-if="scope.row.proofUrl">
              <el-image 
                :src="scope.row.proofUrl" 
                :preview-src-list="[scope.row.proofUrl]"
                fit="cover"
                style="width: 60px; height: 60px; cursor: pointer;"
                :preview-teleported="true"
              />
            </div>
            <span v-else style="color: #999;">暂无凭证</span>
          </template>
        </el-table-column> 


        <!-- <el-table-column label="交易凭证" prop="Image">
          <template #default="scope">
            <el-image :src="scope.row.Image" style="width: 45px; height: 45px; border-radius: 50%"/>
          </template>
        </el-table-column> -->

        <el-table-column label="交易状态">
          <template #default="scope">
            <el-tag :type="getStatusType(scope.row.status)">
              {{ getStatusText(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="ethAddress" label="收款地址"/>

        <el-table-column label="创建时间">
          <template #default="scope">
            <span>{{ dateFormat(scope.row.createdAt * 1000) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="180">
          <template #default="scope">
            <el-button 
              v-if="scope.row.Status === 1 || scope.row.Status === 2"
              size="small" 
              type="warning" 
              @click="cancelOrder(scope.row)">取消订单</el-button>
            <el-button 
              v-if="scope.row.Status === 3"
              size="small" 
              type="success" 
              @click="confirmComplete(scope.row)">确认完成</el-button>
            <!-- <el-popconfirm title="确定要删除当前记录吗?" @confirm="remove(scope.row)">
              <template #reference>
                <el-button size="small" type="danger">删除</el-button>
              </template>
            </el-popconfirm> -->
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
  {value: 1, label: "待撮合"},
  {value: 2, label: "撮合中"},
  {value: 3, label: "待确认"},
  {value: 4, label: "已完成"},
  {value: 5, label: "已取消"},
  {value: 6, label: "申诉中"}
])

const getStatusType = (status) => {
  switch(status) {
    case 1: return "info"
    case 2: return "warning" 
    case 3: return "primary"
    case 4: return "success"
    default: return "danger"
  }
}

const getStatusText = (status) => {
  switch(status) {
    case 1: return "待撮合"
    case 2: return "撮合中"
    case 3: return "待确认"
    case 4: return "已完成"
    case 5: return "已取消"
    case 6: return "申诉中"
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
    
    httpPost('/api/admin/trade/listPowerTrade', params)
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
  httpGet('/api/admin/trade/remove?id=' + row.id).then(() => {
    ElMessage.success("删除成功！")
    items.value = removeArrayItem(items.value, row, (v1, v2) => {
      return v1.id === v2.id
    })
  }).catch((e) => {
    ElMessage.error("删除失败：" + e.message)
  })
}

const cancelOrder = function (row) {
  httpGet('/api/admin/trade/cancel?orderId=' + row.id).then(() => {
    ElMessage.success("取消订单成功！")
    fetchData()
  }).catch((e) => {
    ElMessage.error("取消订单失败：" + e.message)
  })
}

const confirmComplete = function (row) {
  httpGet('/api/admin/trade/complete?orderId=' + row.id).then(() => {
    ElMessage.success("确认完成成功！")
    fetchData()
  }).catch((e) => {
    ElMessage.error("确认完成失败：" + e.message)
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