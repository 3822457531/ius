<template>
  <div class="invite-list" v-loading="loading">
    <!-- 排序选项 -->
    <div class="sort-controls" v-if="total > 0">
      <el-button 
        :type="sortBy === 'yeji' ? 'primary' : 'default'"
        size="small"
        @click="toggleSort">
        <el-icon v-if="sortBy === 'yeji'" style="margin-right: 4px;"><Sort /></el-icon>
        {{ sortBy === 'yeji' ? t('inviteList.sortByPerf') : t('inviteList.sortDefault') }}
      </el-button>
    </div>
    <el-row v-if="items.length > 0">
      <el-table
        :data="items"
        :row-key="row => row.id"
        table-layout="auto"
        border
        :row-class-name="tableRowClassName"
        class="mobile-table">
        <el-table-column label="下级" width="66" align="center">
          <template #default="scope">
            <el-button
              circle
              size="small"
              class="child-list-btn"
              @click="openChildDialog(scope.row)"
              :title="`查看 ${scope.row.username} 的直推`"
            >
              <el-icon><Plus /></el-icon>
            </el-button>
          </template>
        </el-table-column>
        <el-table-column :label="t('inviteList.colUsername')">
          <template #default="scope">
            <div class="username-cell">
              <span>{{ scope.row.username }}</span>
              <span v-if="scope.row.diamond_level > 0" class="diamond-badge">
                {{ scope.row.diamond_level }}{{ t('inviteList.diamondSuffix') }}
              </span>
              <span v-else-if="scope.row.is_jh === 1 && scope.row.member_tier_label" class="member-tier-badge">
                {{ scope.row.member_tier_label }}
              </span>
            </div>
          </template>
        </el-table-column>
      
      
        <el-table-column :label="t('inviteList.colCode')">
          <template #default="scope">
            <div class="code-cell">
              <span>{{ scope.row.next_code }}</span>
              <el-icon 
                class="copy-icon" 
                @click="copyCode(scope.row.next_code)"
                :title="t('inviteList.copyTitle')">
                <DocumentCopy />
              </el-icon>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="yeji" :label="t('inviteList.colPerf')">
          <template #header>
            <div class="yeji-header" @click="toggleSort">
              <span>{{ t('inviteList.colPerf') }}</span>
              <el-icon 
                v-if="sortBy === 'yeji'" 
                class="sort-icon active">
                <ArrowDown />
              </el-icon>
              <el-icon 
                v-else 
                class="sort-icon inactive">
                <Sort />
              </el-icon>
            </div>
          </template>
        </el-table-column>
        <!-- <el-table-column label="注册时间">
          <template #default="scope">
            <span>{{ dateFormat(scope.row['created_at'], 'MM-dd HH:mm:ss') }}</span>
          </template>
        </el-table-column> -->
      </el-table>
    </el-row>
    <el-empty :image-size="100" v-else class="mobile-empty"/>
    <div class="pagination">
      <el-pagination v-if="total > 0"
                     background
                     layout="total,prev, pager, next"
                     :hide-on-single-page="true"
                     v-model:current-page="page"
                     v-model:page-size="pageSize"
                     @current-change="fetchData"
                     :total="total"/>
    </div>

    <el-dialog
      v-model="childDialogVisible"
      :title="`【${childParentName || '-'}】的直推用户`"
      width="92%"
      top="6vh"
      class="child-list-dialog"
      :append-to-body="true"
      destroy-on-close
    >
      <div v-loading="childLoading" class="child-dialog-body">
        <el-table
          :data="childItems"
          table-layout="auto"
          border
          class="mobile-table child-table"
          empty-text="暂无数据"
        >
          <el-table-column prop="username" label="用户名" />
          <el-table-column label="邀请码">
            <template #default="scope">
              <div class="code-cell">
                <span>{{ scope.row.next_code }}</span>
                <el-icon
                  class="copy-icon"
                  @click="copyCode(scope.row.next_code)"
                  title="复制邀请码"
                >
                  <DocumentCopy />
                </el-icon>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="yeji" label="业绩" />
        </el-table>

        <div class="pagination child-pagination">
          <el-pagination
            v-if="childTotal > 0"
            background
            layout="prev, pager, next"
            :hide-on-single-page="true"
            v-model:current-page="childPage"
            v-model:page-size="childPageSize"
            @current-change="fetchChildData"
            :total="childTotal"
          />
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { httpPost } from "@/utils/http";
import { ElMessage } from "element-plus";
import { DocumentCopy, Sort, ArrowDown, Plus } from '@element-plus/icons-vue'
import Clipboard from 'clipboard'

const { t } = useI18n();

const items = ref([]);
const total = ref(0);
const page = ref(1);
const pageSize = ref(10);
const loading = ref(true);
const sortBy = ref(''); // 排序字段：'yeji' 或 ''（默认）
const childDialogVisible = ref(false);
const childLoading = ref(false);
const childParentUserId = ref(0);
const childParentName = ref('');
const childItems = ref([]);
const childTotal = ref(0);
const childPage = ref(1);
const childPageSize = ref(10);

// 切换排序
const toggleSort = () => {
  if (sortBy.value === 'yeji') {
    sortBy.value = ''; // 切换为默认排序
  } else {
    sortBy.value = 'yeji'; // 切换为按业绩排序
  }
  page.value = 1; // 重置到第一页
  fetchData();
};

// 获取数据
const fetchData = () => {
  loading.value = true;
  const params = { 
    page: page.value, 
    page_size: pageSize.value 
  };
  if (sortBy.value) {
    params.sort_by = sortBy.value;
  }
  httpPost('/api/invite/list', params)
    .then((res) => {
      if (res.data) {
        items.value = res.data.items;
        total.value = res.data.total;
      }
      loading.value = false;
    })
    .catch(e => {
      ElMessage.error(t('inviteList.getDataFailPrefix') + e.message);
      loading.value = false;
    });
};

// 动态设置行的类名
const tableRowClassName = ({ row }) => {
  return row.is_jh === 1 ? 'active-user' : '';
};

const fetchChildData = () => {
  if (!childParentUserId.value) return;
  childLoading.value = true;
  httpPost('/api/invite/list', {
    page: childPage.value,
    page_size: childPageSize.value,
    parent_user_id: childParentUserId.value,
  })
    .then((res) => {
      if (res.data) {
        childItems.value = res.data.items || [];
        childTotal.value = res.data.total || 0;
      } else {
        childItems.value = [];
        childTotal.value = 0;
      }
      childLoading.value = false;
    })
    .catch((e) => {
      ElMessage.error('获取下级列表失败: ' + e.message);
      childLoading.value = false;
    });
};

const openChildDialog = (row) => {
  childParentUserId.value = row.user_id || 0;
  childParentName.value = row.username || '';
  childPage.value = 1;
  childItems.value = [];
  childTotal.value = 0;
  childDialogVisible.value = true;
  fetchChildData();
};

// 复制邀请码
const copyCode = (code) => {
  const clipboard = new Clipboard('.copy-icon', {
    text: () => code
  });
  
  clipboard.on('success', () => {
    ElMessage.success(t('inviteList.copySuccess'));
    clipboard.destroy();
  });
  
  clipboard.on('error', () => {
    ElMessage.error(t('inviteList.copyFail'));
    clipboard.destroy();
  });
};

onMounted(() => {
  fetchData();
});
</script>

<style  lang="stylus">
.invite-list {
  .child-list-btn {
    width: 26px;
    height: 26px;
    min-height: 26px;
    padding: 0;
  }

  .sort-controls {
    margin-bottom: 16px;
    display: flex;
    justify-content: flex-end;
    
    .el-button {
      border-radius: 8px;
      font-size: 13px;
      padding: 8px 16px;
      transition: all 0.3s ease;
      
      &.el-button--primary {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        border: none;
        
        &:hover {
          opacity: 0.9;
          transform: translateY(-1px);
          box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
        }
      }
      
      &.el-button--default {
        background-color: rgba(255, 255, 255, 0.9);
        border-color: rgba(0, 0, 0, 0.1);
        color: #606266;
        
        &:hover {
          background-color: rgba(255, 255, 255, 1);
          border-color: #667eea;
          color: #667eea;
        }
      }
    }
  }
  
  .pagination {
    margin: 20px 0 0 0;
    display: flex;
    justify-content: center;
    width: 100%;
    max-width: 100%;
    position: relative;
    z-index: 9999;
    overflow-x: auto;
    overflow-y: visible;
    padding: 0 4px;
    -webkit-overflow-scrolling: touch;
    box-sizing: border-box;
    
    &::-webkit-scrollbar {
      height: 4px;
    }
    
    &::-webkit-scrollbar-track {
      background: rgba(0, 0, 0, 0.05);
      border-radius: 2px;
    }
    
    &::-webkit-scrollbar-thumb {
      background: rgba(102, 126, 234, 0.3);
      border-radius: 2px;
      
      &:hover {
        background: rgba(102, 126, 234, 0.5);
      }
    }
    
    :deep(.el-pagination) {
      pointer-events: auto;
      flex-wrap: nowrap;
      min-width: fit-content;
      width: 100%;
      max-width: 100%;
      justify-content: center;
      
      .el-pager {
        display: flex;
        flex-wrap: nowrap;
        gap: 4px;
      }
      
      .el-pager li {
        background-color: rgba(255, 255, 255, 0.9);
        color: #606266;
        border-radius: 6px;
        margin: 0;
        flex-shrink: 0;
        min-width: 32px;
        
        &.is-active {
          background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
          color: #fff;
        }
      }
      
      .btn-prev, .btn-next {
        background-color: rgba(255, 255, 255, 0.9);
        color: #606266;
        border-radius: 6px;
        flex-shrink: 0;
      }
      
      .el-pagination__total {
        color: #606266;
        flex-shrink: 0;
        margin-right: 8px;
      }
    }
  }
  
  .code-cell {
    display: flex;
    align-items: center;
    justify-content: space-between;
    
    .copy-icon {
      cursor: pointer;
      color: #909399;
      margin-left: 8px;
      font-size: 16px;
      transition: all 0.2s ease;
      
      &:hover {
        color: #667eea;
        transform: scale(1.1);
      }
    }
  }
  
  .username-cell {
    display: flex;
    align-items: center;
    gap: 8px;
    
    .diamond-badge {
      display: inline-block;
      padding: 2px 8px;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: #fff;
      border-radius: 12px;
      font-size: 11px;
      font-weight: 600;
      line-height: 1.4;
      white-space: nowrap;
    }

    .member-tier-badge {
      display: inline-block;
      padding: 2px 8px;
      background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
      color: #fff;
      border-radius: 12px;
      font-size: 11px;
      font-weight: 600;
      line-height: 1.4;
      white-space: nowrap;
    }
  }
  
  .yeji-header {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    cursor: pointer;
    user-select: none;
    transition: all 0.2s ease;
    
    &:hover {
      color: #667eea;
      
      .sort-icon {
        color: #667eea;
      }
    }
    
    .sort-icon {
      font-size: 14px;
      transition: all 0.2s ease;
      
      &.active {
        color: #667eea;
        opacity: 1;
        transform: scale(1.1);
      }
      
      &.inactive {
        color: #c0c4cc;
        opacity: 0.6;
      }
    }
  }
}

.child-dialog-body {
  min-height: 220px;
}

.child-pagination {
  margin-top: 14px !important;
}

.child-list-dialog {
  :deep(.el-dialog) {
    margin: 0 auto;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  :deep(.el-dialog__header) {
    flex-shrink: 0;
    padding-right: 36px;
  }

  :deep(.el-dialog__body) {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
  }
}

.mobile-table {
  background: transparent;
  border-radius: 12px;
  overflow: hidden;
  
  :deep(.el-table__header-wrapper) {
    .el-table__header {
      th {
        background: linear-gradient(135deg, rgba(102, 126, 234, 0.1) 0%, rgba(118, 75, 162, 0.1) 100%);
        border-color: rgba(0, 0, 0, 0.08);
        color: #2c3e50;
        font-weight: 600;
        font-size: 14px;
        padding: 12px 0;
      }
    }
  }
  
  :deep(.el-table__body-wrapper) {
    .el-table__body {
      tr {
        background-color: rgba(255, 255, 255, 0.6);
        border-color: rgba(0, 0, 0, 0.06);
        transition: all 0.2s ease;
        
        &:hover {
          background-color: rgba(255, 255, 255, 0.9);
          transform: translateX(2px);
        }
        
        td {
          border-color: rgba(0, 0, 0, 0.06);
          color: #2c3e50;
          font-size: 13px;
          padding: 14px 0;
        }
      }
    }
  }
  
  :deep(.el-table__border) {
    border-color: rgba(0, 0, 0, 0.08);
  }
  
  :deep(.el-table__inner-wrapper::before) {
    background-color: rgba(0, 0, 0, 0.08);
  }
}

.active-user {
  background: linear-gradient(135deg, rgba(123, 220, 181, 0.15) 0%, rgba(123, 220, 181, 0.08) 100%) !important;
  
  &:hover {
    background: linear-gradient(135deg, rgba(123, 220, 181, 0.25) 0%, rgba(123, 220, 181, 0.15) 100%) !important;
  }
}

.mobile-empty {
  padding: 40px 20px;
  
  :deep(.el-empty__description) {
    color: #909399;
    font-size: 14px;
  }
}
</style>
