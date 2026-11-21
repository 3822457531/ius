<template>
  <div class="container user-list" v-loading="loading">
    <div class="handle-box">
      <el-input v-model="query.username" placeholder="账号" class="handle-input mr10"></el-input>
      <el-button type="primary" :icon="Search" @click="handleSearch">搜索</el-button>

      <el-button type="success" :icon="Plus" @click="addUser">新增用户</el-button>

      <!-- 更改为审核用户 -->
      <el-button type="primary" :icon="Plus" @click="auditUser">审核用户</el-button>
      <el-button type="primary" plain @click="setCs(true)">设为客服</el-button>
      <el-button type="info" plain @click="setCs(false)">取消客服</el-button>

      <el-button type="warning" @click="setAllowMint(true)">开通算力机抢购</el-button>
      <el-button type="info" @click="setAllowMint(false)">关闭算力机抢购</el-button>
      <el-button type="success" plain @click="setAllowPlacement(true)">开通大小区</el-button>
      <el-button type="info" plain @click="setAllowPlacement(false)">关闭大小区</el-button>
      <el-button type="danger" plain @click="rollbackSelectedMusicRefund">回滚音乐重复退款</el-button>
      <el-button type="danger" @click="openPointRollbackDialog">算力时间点回滚</el-button>
    </div>

    <el-row>
      <el-table :data="users.items" border class="table" :row-key="row => row.id"
                @selection-change="handleSelectionChange" table-layout="auto">
        <el-table-column type="selection" width="38"/>
        <el-table-column prop="mobile" label="账号">
          <template #default="scope">
            <span>{{ scope.row.username }}</span>
            <el-image v-if="scope.row.vip" :src="vipImg" style="height: 20px;position: relative; top:5px; left: 5px"/>
            <el-icon v-if="scope.row.is_auditor" :size="18" style="position: relative; top:3px; left: 5px; color: #667eea;">
              <View />
            </el-icon>
            <el-tag v-if="scope.row.is_cs" type="success" size="small" style="margin-left: 6px;">客服</el-tag>
          </template>
        </el-table-column>
        <!-- <el-table-column prop="nickname" label="昵称"/> -->
        <el-table-column prop="lev" label="钻石等级" width="110">
          <template #default="scope">
            <el-tag v-if="Number(scope.row.lev) > 0" type="warning">{{ scope.row.lev }}钻</el-tag>
            <el-tag v-else type="info">无</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="code" label="邀请码"/>
        <el-table-column prop="power" label="剩余算力"/>
        <el-table-column label="算力机" width="90">
          <template #default="scope">
            <el-tag v-if="scope.row.allow_mint" type="success" size="small">可抢购</el-tag>
            <el-tag v-else type="info" size="small">已售罄</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="大小区" width="90">
          <template #default="scope">
            <el-tag v-if="scope.row.allow_placement" type="success" size="small">已开通</el-tag>
            <el-tag v-else type="info" size="small">关闭</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="scope">
            <el-tag v-if="scope.row.status" type="success">正常</el-tag>
            <el-tag type="danger" v-else>停用</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="过期时间">
          <template #default="scope">
            <span v-if="scope.row['expired_time']">{{ scope.row['expired_time'] }}</span>
            <el-tag v-else>长期有效</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="注册时间">
          <template #default="scope">
            <span>{{ dateFormat(scope.row['created_at']) }}</span>
          </template>
        </el-table-column>

        <el-table-column fixed="right" label="操作" width="360">
          <template #default="scope">
            <el-button-group class="ml-4">
              <el-button size="small" type="primary" @click="userEdit(scope.row)">编辑</el-button>
              <el-button size="small" type="danger" @click="removeUser(scope.row)">删除</el-button>
              <el-button size="small" type="success" @click="resetPass(scope.row)">重置密码</el-button>
              <el-button size="small" type="warning" plain @click="rollbackMusicRefund(scope.row)">回滚羊毛</el-button>
              <el-button size="small" type="danger" plain @click="openPointRollbackDialog(scope.row)">时间点回滚</el-button>
            </el-button-group>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination v-if="users.total > 0"
                       background
                       layout="total, prev, pager, next"
                       v-model:current-page="users.page"
                       v-model:page-size="users.page_size"
                       :total="users.total"
                       @current-change="fetchUserList(users.page, users.page_size)"
        />

      </div>
    </el-row>

    <el-dialog
        v-model="showUserEditDialog"
        :title="title"
        :close-on-click-modal="false"
        width="50%"
    >
      <el-form :model="user" label-width="100px" ref="userEditFormRef" :rules="rules">
        <el-form-item label="账号：" prop="username">
          <el-input v-model="user.username" autocomplete="off"/>
        </el-form-item>
        <el-form-item v-if="add" label="密码：" prop="password">
          <el-input v-model="user.password" autocomplete="off"/>
        </el-form-item>
        <el-form-item label="剩余算力：" prop="power">
          <el-input v-model.number="user.power" autocomplete="off" placeholder="0"/>
        </el-form-item>

        <el-form-item label="钻石等级">
          <el-select v-model="user.lev" placeholder="选择钻石等级" style="width: 260px" @change="onLevChange">
            <el-option v-for="opt in levOptions" :key="opt.value" :label="opt.label" :value="opt.value"/>
          </el-select>
          <div style="font-size: 12px; color: #909399; margin-top: 5px;">
            管理员强制设置后，用户端将以此等级为准（不再自动回写覆盖）。
          </div>
        </el-form-item>

        <el-form-item label="有效期：" prop="expired_time">
          <el-date-picker
              v-model="user.expired_time"
              type="datetime"
              placeholder="选择日期"
              format="YYYY-MM-DD HH:mm:ss"
              value-format="YYYY-MM-DD HH:mm:ss"
              :disabled-date="disabledDate"
          />
        </el-form-item>

        <el-form-item label="聊天角色" prop="chat_roles">
          <el-select
              v-model="user.chat_roles"
              multiple
              :filterable="true"
              placeholder="选择聊天角色，多选"
          >
            <el-option
                v-for="item in roles"
                :key="item.key"
                :label="item.name"
                :value="item.key"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="模型权限" prop="chat_models">
          <el-select
              v-model="user.chat_models"
              multiple
              :filterable="true"
              placeholder="选择AI模型，多选"
          >
            <el-option
                v-for="item in models"
                :key="item.id"
                :label="item.name"
                :value="item.id"
            />
          </el-select>
        </el-form-item>


        <el-form-item label="启用状态">
          <el-switch v-model="user.status"/>
        </el-form-item>

        <el-form-item label="开通VIP">
          <el-switch v-model="user.vip"/>
        </el-form-item>

        <!-- 是否允许算力交易 -->
        <el-form-item label="允许算力交易">
          <el-switch v-model="user.allow_trade"/>
        </el-form-item>

                <!-- 是否允许SOHA交易 -->
        <el-form-item label="允许SOHA交易">
          <el-switch v-model="user.allow_q2"/>
          <div style="font-size: 12px; color: #909399; margin-top: 5px;">
            默认开启；需全局「开放SOHA交易市场」同时开启后可 Swap / C2C；关闭时用户仅可浏览行情
          </div>
        </el-form-item>

        <el-form-item label="允许抢购算力机">
          <el-switch v-model="user.allow_mint"/>
          <div style="font-size: 12px; color: #909399; margin-top: 5px;">
            默认开启；关闭后用户端提示已售罄
          </div>
        </el-form-item>

        <el-form-item label="允许查看大小区">
          <el-switch v-model="user.allow_placement"/>
          <div style="font-size: 12px; color: #909399; margin-top: 5px;">
            默认关闭；开启后手机端邀请页才显示「大小区」入口
          </div>
        </el-form-item>

        <!-- 是否审核用户 -->
        <el-form-item label="审核用户">
          <el-switch v-model="user.is_auditor"/>
          <div style="font-size: 12px; color: #909399; margin-top: 5px;">
            审核用户可以在手机端查看和审核算力提现订单
          </div>
        </el-form-item>

        <el-form-item label="指定客服">
          <el-switch v-model="user.is_cs"/>
          <div style="font-size: 12px; color: #909399; margin-top: 5px;">
            开启后，该用户在 H5 点击「客服」进入接线台接入用户，而不是作为普通用户咨询
          </div>
        </el-form-item>

      </el-form>

      <template #footer>
            <span class="dialog-footer">
              <el-button @click="showUserEditDialog = false">取消</el-button>
              <el-button type="primary" @click="saveUser">提交</el-button>
            </span>
      </template>
    </el-dialog>

    <el-dialog
        v-model="showResetPassDialog"
        title="重置密码"
        width="50%"
    >
      <el-form label-width="100px" ref="userEditFormRef">
        <el-form-item label="账户：">
          <el-input v-model="pass.username" autocomplete="off" readonly disabled/>
        </el-form-item>

        <el-form-item label="新密码：">
          <el-input v-model="pass.password" autocomplete="off"/>
        </el-form-item>
      </el-form>

      <template #footer>
            <span class="dialog-footer">
              <el-button type="primary" @click="doResetPass">提交</el-button>
            </span>
      </template>

    </el-dialog>

    <el-dialog
        v-model="showPointRollbackDialog"
        title="算力时间点回滚"
        :close-on-click-modal="false"
        width="640px"
    >
      <el-form label-width="110px">
        <el-form-item label="用户">
          <span>{{ pointRollback.user?.username }}（ID: {{ pointRollback.user?.id }}）</span>
        </el-form-item>
        <el-form-item label="回滚至">
          <el-date-picker
              v-model="pointRollback.rollbackAt"
              type="datetime"
              placeholder="选择年月日时分秒"
              format="YYYY-MM-DD HH:mm:ss"
              value-format="YYYY-MM-DD HH:mm:ss"
              style="width: 100%"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="pointRollback.previewLoading" @click="previewPointRollback">
            预览影响
          </el-button>
        </el-form-item>
      </el-form>

      <div v-if="pointRollback.preview" class="point-rollback-preview">
        <p>当前算力：<strong>{{ pointRollback.preview.current_power }}</strong></p>
        <p>回滚后算力：<strong>{{ pointRollback.preview.restored_power }}</strong></p>
        <p>软隐藏流水：本用户 {{ pointRollback.preview.primary_hidden_count }} 条，
          关联用户 {{ pointRollback.preview.paired_hidden_count }} 条，
          合计 {{ pointRollback.preview.total_hidden_count }} 条</p>
        <p v-if="pointRollback.preview.order_release_log_count > 0">
          订单同步：{{ pointRollback.preview.affected_order_count }} 笔订单将重算
          js_power / now_power（含签到、静态释放、团队奖、推荐奖等 {{ pointRollback.preview.order_release_log_count }} 条加速释放流水）
        </p>
        <div v-if="categorySummaryLines.length">
          <p>分类统计：</p>
          <ul>
            <li v-for="line in categorySummaryLines" :key="line">{{ line }}</li>
          </ul>
        </div>
        <div v-if="pointRollback.preview.affected_users?.length > 1">
          <p>受影响用户：</p>
          <el-table :data="pointRollback.preview.affected_users" size="small" border>
            <el-table-column prop="username" label="账号"/>
            <el-table-column prop="current_power" label="当前算力"/>
            <el-table-column prop="restored_power" label="回滚后"/>
            <el-table-column prop="hidden_log_count" label="隐藏条数"/>
          </el-table>
        </div>
        <p v-if="pointRollback.preview.cascade_order_count > 0">
          收款方级联：将同步删除 {{ pointRollback.preview.cascade_order_count }} 笔质押订单（收款后投入/激活一并回滚，订单物理删除）
        </p>
        <p class="hint">流水不会物理删除，仅软隐藏。转账收款方会级联回滚收款后流水与订单；下级投入/质押触发的上级「推荐奖励释放」「团队奖励释放」会自动配对回滚；test 回滚时刻后的团队奖等也会一并隐藏。</p>
      </div>

      <template #footer>
        <el-button @click="showPointRollbackDialog = false">取消</el-button>
        <el-button
            type="danger"
            :loading="pointRollback.executing"
            :disabled="!pointRollback.preview || pointRollback.preview.total_hidden_count === 0"
            @click="executePointRollback"
        >
          确认回滚
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import {computed, onMounted, reactive, ref} from "vue";
import {httpGet, httpPost} from "@/utils/http";
import {ElMessage, ElMessageBox} from "element-plus";
import {dateFormat, disabledDate, removeArrayItem} from "@/utils/libs";
import {Plus, Search, View} from "@element-plus/icons-vue";

// 变量定义
const users = ref({page: 1, page_size: 15, items: []})
const query = ref({username: '', page: 1, page_size: 15})

const title = ref('添加用户')
const vipImg = ref("/images/vip.png")
const add = ref(true)
const user = ref({chat_roles: [], chat_models: []})
const pass = ref({username: '', password: '', id: 0})
const roles = ref([])
const models = ref([])
const showUserEditDialog = ref(false)
const showResetPassDialog = ref(false)
const rules = reactive({
  username: [{required: true, message: '请输入账号', trigger: 'change',}],
  password: [{required: true, message: '请输入密码', trigger: 'change',}],
  calls: [
    {required: true, message: '请输入提问次数'},
    {type: 'number', message: '请输入有效数字'},
  ],
  chat_roles: [{required: true, message: '请选择聊天角色', trigger: 'change'}],
  chat_models: [{required: true, message: '请选择AI模型', trigger: 'change'}],
})
const loading = ref(true)

const userEditFormRef = ref(null)

const levOptions = [
  { label: '无', value: 0 },
  { label: '1钻', value: 1 },
  { label: '2钻', value: 2 },
  { label: '3钻', value: 3 },
  { label: '4钻', value: 4 },
  { label: '5钻', value: 5 },
  { label: '6钻', value: 6 },
  { label: '7钻', value: 7 },
  { label: '8钻', value: 8 },
  { label: '9钻', value: 9 },
]

const levToFloor = (lev) => {
  const n = Number(lev || 0)
  if (n <= 0) return 10
  if (n >= 9) return 100
  return n * 10 + 10
}

const onLevChange = (val) => {
  user.value.floor = levToFloor(val)
}

onMounted(() => {
  fetchUserList(users.value.page, users.value.page_size)
  // 获取角色列表
  httpGet('/api/admin/role/list').then((res) => {
    roles.value = res.data;
  }).catch(() => {
    ElMessage.error("获取聊天角色失败");
  })

  httpGet('/api/admin/model/list').then(res => {
    models.value = res.data
  }).catch(e => {
    ElMessage.error("获取模型失败：" + e.message)
  })
})

const fetchUserList = function (page, pageSize) {
  query.value.page = page
  query.value.page_size = pageSize
  httpGet('/api/admin/user/list', query.value).then((res) => {
    if (res.data) {
      // 初始化数据
      const arr = res.data.items;
      for (let i = 0; i < arr.length; i++) {
        arr[i].expired_time = dateFormat(arr[i].expired_time)
      }
      users.value.items = arr
      users.value.total = res.data.total
      users.value.page = res.data.page
      user.value.page_size = res.data.page_size
    }
    loading.value = false
  }).catch(() => {
    ElMessage.error('加载用户列表失败')
  })
}

const handleSearch = () => {
  fetchUserList(users.value.page, users.value.page_size)
}

// 删除用户
const removeUser = function (user) {
  ElMessageBox.confirm(
      '此操作将会永久删除用户信息和聊天记录，确认操作吗?',
      '警告',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
  ).then(() => {
    httpGet('/api/admin/user/remove', {id: user.id}).then(() => {
      ElMessage.success('操作成功！')
      users.value.items = removeArrayItem(users.value.items, user, function (v1, v2) {
        return v1.id === v2.id
      })
    }).catch((e) => {
      ElMessage.error('操作失败，' + e.message)
    })
  }).catch(() => {
    ElMessage.info('操作被取消')
  })

}

const userEdit = function (row) {
  user.value = row
  if (user.value && user.value.lev !== undefined) {
    user.value.floor = levToFloor(user.value.lev)
  }
  title.value = '编辑用户'
  showUserEditDialog.value = true
  add.value = false
}

const addUser = () => {
  user.value = {}
  title.value = '添加用户'
  showUserEditDialog.value = true
  add.value = true
  user.value.lev = 0
  user.value.floor = levToFloor(0)
  user.value.allow_q2 = true
  user.value.allow_mint = true
  user.value.allow_trade = true
}

const saveUser = function () {
  userEditFormRef.value.validate((valid) => {
    if (valid) {
      showUserEditDialog.value = false
      console.log(user.value)
      httpPost('/api/admin/user/save', user.value).then((res) => {
        ElMessage.success('操作成功！')
        if (add.value) {
          users.value.items.push(res.data)
        }
      }).catch((e) => {
        ElMessage.error('操作失败，' + e.message)
      })
    } else {
      return false
    }
  })
}

const handleSelectionChange = function (rows) {
  selectedUsers.value = rows
}

const selectedUsers = ref([])

// 批量设置为审核用户
const auditUser = () => {
  if (selectedUsers.value.length === 0) {
    ElMessage.warning('请先选择要设置为审核用户的用户')
    return
  }
  
  ElMessageBox.confirm(
    `确认将选中的 ${selectedUsers.value.length} 个用户设置为审核用户？`,
    '设置审核用户',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(() => {
    const userIds = selectedUsers.value.map(u => u.id)
    httpPost('/api/admin/user/setAuditor', { ids: userIds }).then(() => {
      ElMessage.success('设置成功！')
      fetchUserList(users.value.page, users.value.page_size)
      selectedUsers.value = []
    }).catch((e) => {
      ElMessage.error('设置失败，' + e.message)
    })
  }).catch(() => {
    // 用户取消操作
  })
}

const setCs = (allow) => {
  if (selectedUsers.value.length === 0) {
    ElMessage.warning('请先选择用户')
    return
  }
  const action = allow ? '设为客服' : '取消客服'
  ElMessageBox.confirm(
    `确认将选中的 ${selectedUsers.value.length} 个用户${action}？`,
    action,
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(() => {
    const userIds = selectedUsers.value.map(u => u.id)
    httpPost('/api/admin/user/setCs', { ids: userIds, allow }).then(() => {
      ElMessage.success('设置成功！')
      fetchUserList(users.value.page, users.value.page_size)
      selectedUsers.value = []
    }).catch((e) => {
      ElMessage.error('设置失败，' + e.message)
    })
  }).catch(() => {})
}

const setAllowMint = (allow) => {
  if (selectedUsers.value.length === 0) {
    ElMessage.warning('请先选择用户')
    return
  }
  const action = allow ? '开通' : '关闭'
  ElMessageBox.confirm(
    `确认${action}选中的 ${selectedUsers.value.length} 个用户的算力机抢购权限？`,
    `${action}算力机抢购`,
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(() => {
    const userIds = selectedUsers.value.map(u => u.id)
    httpPost('/api/admin/user/setAllowMint', { ids: userIds, allow }).then(() => {
      ElMessage.success('设置成功！')
      fetchUserList(users.value.page, users.value.page_size)
      selectedUsers.value = []
    }).catch((e) => {
      ElMessage.error('设置失败，' + e.message)
    })
  }).catch(() => {})
}

const setAllowPlacement = (allow) => {
  if (selectedUsers.value.length === 0) {
    ElMessage.warning('请先选择用户')
    return
  }
  const action = allow ? '开通' : '关闭'
  ElMessageBox.confirm(
    `确认${action}选中的 ${selectedUsers.value.length} 个用户的大小区显示？`,
    `${action}大小区`,
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(() => {
    const userIds = selectedUsers.value.map(u => u.id)
    httpPost('/api/admin/user/setAllowPlacement', { ids: userIds, allow }).then(() => {
      ElMessage.success('设置成功！')
      fetchUserList(users.value.page, users.value.page_size)
      selectedUsers.value = []
    }).catch((e) => {
      ElMessage.error('设置失败，' + e.message)
    })
  }).catch(() => {})
}

const buildMusicRollbackMessage = (preview) => {
  const lines = [
    `用户：${preview.username}`,
    `当前算力：${preview.current_power}`,
    `重复退款日志：${preview.duplicate_log_count} 条`,
    `需扣回算力：${preview.excess_power}`,
    `回滚后算力：${preview.corrected_power}`,
  ]
  if (preview.task_groups?.length) {
    lines.push('', '涉及任务（每任务保留最早一条退款）：')
    preview.task_groups.slice(0, 8).forEach((g) => {
      lines.push(`· ${g.task_id}：${g.refund_count} 次退款，删除 ${g.remove_log_count} 条，扣回 ${g.excess_amount}`)
    })
    if (preview.task_groups.length > 8) {
      lines.push(`… 另有 ${preview.task_groups.length - 8} 个任务`)
    }
  }
  lines.push('', '将软隐藏重复退款日志并重算余额，是否继续？')
  return lines.join('\n')
}

const rollbackMusicRefund = async (row) => {
  try {
    const previewRes = await httpGet('/api/admin/user/previewMusicRefundRollback', { user_id: row.id })
    const preview = previewRes.data
    if (!preview?.duplicate_log_count && !(preview?.excess_power > 0)) {
      ElMessage.info('该用户没有可回滚的音乐重复退款记录')
      return
    }
    await ElMessageBox.confirm(buildMusicRollbackMessage(preview), '回滚音乐重复退款', {
      confirmButtonText: '确认回滚',
      cancelButtonText: '取消',
      type: 'warning',
    })
    const res = await httpPost('/api/admin/user/rollbackMusicRefundExploit', { user_id: row.id })
    ElMessage.success(`回滚成功！算力 ${preview.current_power} → ${res.data.corrected_power}`)
    fetchUserList(users.value.page, users.value.page_size)
  } catch (e) {
    if (e !== 'cancel' && e?.message !== 'cancel') {
      ElMessage.error('回滚失败：' + (e?.message || e))
    }
  }
}

const rollbackSelectedMusicRefund = async () => {
  if (selectedUsers.value.length === 0) {
    ElMessage.warning('请先选择用户')
    return
  }
  if (selectedUsers.value.length > 1) {
    ElMessage.warning('请一次只回滚一个用户，或使用行内「回滚羊毛」按钮')
    return
  }
  await rollbackMusicRefund(selectedUsers.value[0])
}

const showPointRollbackDialog = ref(false)
const pointRollback = reactive({
  user: null,
  rollbackAt: '',
  preview: null,
  previewLoading: false,
  executing: false,
})

const categorySummaryLines = computed(() => {
  const summary = pointRollback.preview?.category_summary
  if (!summary) return []
  return Object.entries(summary).map(([k, v]) => `${k}：${v} 条`)
})

const openPointRollbackDialog = (row) => {
  const target = row || (selectedUsers.value.length === 1 ? selectedUsers.value[0] : null)
  if (!target) {
    ElMessage.warning('请先选择一名用户，或点击行内「时间点回滚」')
    return
  }
  pointRollback.user = target
  pointRollback.rollbackAt = ''
  pointRollback.preview = null
  showPointRollbackDialog.value = true
}

const previewPointRollback = async () => {
  if (!pointRollback.user?.id) {
    ElMessage.warning('请选择用户')
    return
  }
  if (!pointRollback.rollbackAt) {
    ElMessage.warning('请选择回滚时间')
    return
  }
  pointRollback.previewLoading = true
  try {
    const res = await httpGet('/api/admin/user/previewPowerPointRollback', {
      user_id: pointRollback.user.id,
      rollback_at: pointRollback.rollbackAt,
    })
    pointRollback.preview = res.data
    if (!pointRollback.preview?.total_hidden_count) {
      ElMessage.info('该时间点之后没有可回滚的算力流水')
    }
  } catch (e) {
    ElMessage.error('预览失败：' + (e?.message || e))
  } finally {
    pointRollback.previewLoading = false
  }
}

const executePointRollback = async () => {
  if (!pointRollback.preview?.total_hidden_count) {
    ElMessage.info('请先预览并确认有可回滚流水')
    return
  }
  try {
    await ElMessageBox.confirm(
      `确认将用户 ${pointRollback.user.username} 的算力回滚至 ${pointRollback.rollbackAt}？\n` +
      `算力 ${pointRollback.preview.current_power} → ${pointRollback.preview.restored_power}，` +
      `软隐藏 ${pointRollback.preview.total_hidden_count} 条流水（含转账关联用户）。\n` +
      `数据量大时可能需要数分钟，请勿关闭页面。`,
      '确认算力时间点回滚',
      { confirmButtonText: '确认回滚', cancelButtonText: '取消', type: 'warning' }
    )
    pointRollback.executing = true
    const res = await httpPost('/api/admin/user/executePowerPointRollback', {
      user_id: pointRollback.user.id,
      rollback_at: pointRollback.rollbackAt,
    })
    ElMessage.success(
      `回滚成功！算力 ${pointRollback.preview.current_power} → ${res.data.restored_power}`
    )
    showPointRollbackDialog.value = false
    fetchUserList(users.value.page, users.value.page_size)
  } catch (e) {
    if (e !== 'cancel' && e?.message !== 'cancel') {
      ElMessage.error('回滚失败：' + (e?.message || e))
    }
  } finally {
    pointRollback.executing = false
  }
}

const resetPass = (row) => {
  showResetPassDialog.value = true
  pass.value.id = row.id
  pass.value.username = row.username
}

const doResetPass = () => {
  httpPost('/api/admin/user/resetPass', pass.value).then(() => {
    ElMessage.success('操作成功！')
    showResetPassDialog.value = false
  }).catch((e) => {
    ElMessage.error('操作失败，' + e.message)
  })
}

</script>

<style lang="stylus" scoped>
.user-list {

  .handle-box {
    .handle-input {
      max-width 150px;
      margin-right 10px;
    }
  }

  .table {
    width 100%
  }

  .opt-box {
    padding-bottom: 10px;

    .el-icon {
      margin-right: 5px;
    }
  }

  .el-select {
    width: 100%
  }

}

.point-rollback-preview {
  margin-top 8px
  padding 12px
  background #f5f7fa
  border-radius 4px
  font-size 13px

  .hint {
    color #909399
    margin-top 8px
  }

  ul {
    margin 4px 0 8px 18px
    padding 0
  }
}
</style>