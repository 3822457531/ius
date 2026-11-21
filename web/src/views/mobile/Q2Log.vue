<template>
  <div class="power-log" v-loading="loading">
    <div class="inner">
      <h2>SOHA明细</h2>

      <div class="list-box" :style="{height: listBoxHeight + 'px'}">
        <div class="handle-box">
          <el-input v-model="query.model" placeholder="模型或者用户名" class="handle-input" clearable></el-input>
   
          <el-button type="primary" :icon="Search" @click="fetchData" class="search-button">搜索</el-button>
        </div>

        <el-row v-if="items.length > 0">
          <el-table :data="items" :row-key="row => row.id" table-layout="auto" border class="mobile-table">
            <el-table-column prop="model" label="模型" />
 
            <el-table-column label="SOHA">
              <template #default="scope">
                <div>
                  <el-text type="success" v-if="scope.row.mark === 1">+{{ scope.row.amount }}</el-text>
                  <el-text type="danger" v-if="scope.row.mark === 0">-{{ scope.row.amount }}</el-text>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="balance" label="余额" />


            <el-table-column label="时间">
              <template #default="scope">
                <span>{{ dateFormat(scope.row['created_at'],'MM-dd HH:mm:ss') }}</span>
              </template>
            </el-table-column>
          </el-table>

          <div class="pagination">
            <el-pagination v-if="total > 0" background
                           layout="total,prev, pager, next"
                           :hide-on-single-page="true"
                           v-model:current-page="page"
                           v-model:page-size="pageSize"
                           @current-change="fetchData()"
                           :total="total" />
          </div>
        </el-row>
        <el-empty :image-size="100" v-else />
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from "vue";
import { dateFormat } from "@/utils/libs";
import { Search } from "@element-plus/icons-vue";
import Clipboard from "clipboard";
import { ElMessage } from "element-plus";
import { httpPost } from "@/utils/http";

const items = ref([]);
const total = ref(0);
const page = ref(1);
const pageSize = ref(20);
const loading = ref(false);
const listBoxHeight = window.innerHeight - 117;
const query = ref({
  model: "",
  date: []
});
const tagColors = ref(["", "success", "primary", "danger", "info", "warning"]);

onMounted(() => {
  fetchData();
  const clipboard = new Clipboard('.copy-order-no');
  clipboard.on('success', () => {
    ElMessage.success("复制成功！");
  });

  clipboard.on('error', () => {
    ElMessage.error('复制失败！');
  });
});

const fetchData = () => {
  loading.value = true;
  httpPost('/api/q2Log/list', {
    model: query.value.model,
    date: query.value.date,
    page: page.value,
    page_size: pageSize.value
  }).then((res) => {
    if (res.data) {
      items.value = res.data.items;
      total.value = res.data.total;
      page.value = res.data.page;
      pageSize.value = res.data.page_size;
    }
    loading.value = false;
  }).catch(e => {
    loading.value = false;
    ElMessage.error("获取数据失败：" + e.message);
  });
};


</script>


<style lang="stylus" scoped>
.power-log {
  background-color: #ffffff;

  .inner {
    padding: 10px;

    h2 {
      font-size: 20px;
      margin-bottom: 10px;
    }

    .list-box {
      overflow-x: hidden;

      .handle-box {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
        padding: 10px 0;

        .el-input,
        .date-picker,
        .search-button {
          flex: 1;
          min-width: 100px;
        }

        .date-picker {
          max-width: 200px;
        }

        .search-button {
          flex: none;
        }
      }

      .mobile-table {
        .el-table__header,
        .el-table__body {
          display: block;
          overflow-x: auto;
          white-space: nowrap;
        }

        .el-table__header-wrapper,
        .el-table__body-wrapper {
          overflow-x: auto;
        }
      }

      .pagination {
        display: flex;
        justify-content: center;
        width: 100%;
        padding: 10px 0;
      }
    }
  }
}
</style>
