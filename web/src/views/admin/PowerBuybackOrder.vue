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
            <div>用户: {{ scope.row.sellerName || '未知' }}</div>
            <div>回购方: {{ scope.row.buyerName || '系统回购' }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="listedPower" label="回购算力"/>
        <el-table-column prop="price" label="单价"/>
        <el-table-column prop="valueUsd" label="回购总额">
          <template #default="scope">
            ${{ scope.row.valueUsd.toFixed(2) }}
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
            <span>{{ dateFormat(scope.row.createdAt) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="完成时间" v-if="items.some(item => item.finishTime)">
          <template #default="scope">
            <span v-if="scope.row.finishTime">{{ dateFormat(scope.row.finishTime) }}</span>
            <span v-else style="color: #999;">未完成</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="150" fixed="right">
          <template #default="scope">
            <!-- 状态1：待管理员打款 - 显示已打款按钮 -->
            <el-button 
              v-if="scope.row.status === 1"
              type="primary" 
              size="small"
              @click="handleConfirmPayment(scope.row)"
            >
              已打款
            </el-button>
            <!-- 状态2：已完成 -->
            <span v-else-if="scope.row.status === 2" style="color: #67C23A;">回购完成</span>
            <span v-else style="color: #909399;">未知状态</span>
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
import {ElMessage, ElMessageBox} from "element-plus";
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
  {value: 1, label: "待管理员打款"},
  {value: 2, label: "已完成"}
])

const getStatusType = (status) => {
  switch(status) {
    case 1: return "warning"  // 待管理员打款
    case 2: return "success"   // 已完成
    default: return "info"
  }
}

const getStatusText = (status) => {
  switch(status) {
    case 1: return "待管理员打款"
    case 2: return "已完成"
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
    
    httpPost('/api/admin/trade/listBuyback', params)
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

// 确认已打款
const handleConfirmPayment = (row) => {
    ElMessageBox.confirm(
        `确认订单 ${row.orderNumber} 已打款？确认后将完成回购订单。`,
        '确认已打款',
        {
            confirmButtonText: '确认',
            cancelButtonText: '取消',
            type: 'warning',
        }
    ).then(() => {
        loading.value = true
        httpPost('/api/admin/trade/confirmBuybackPayment', {
            orderId: row.id
        })
        .then((res) => {
            ElMessage.success('确认已打款成功，回购订单已完成')
            fetchData() // 刷新列表
        })
        .catch(e => {
            ElMessage.error("确认已打款失败：" + (e.message || '未知错误'))
        })
        .finally(() => {
            loading.value = false
        })
    }).catch(() => {
        // 用户取消操作
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

