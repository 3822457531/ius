<template>
  <div class="chat-list-page">
    <div v-if="isLogin" class="layout">
      <header class="page-head van-safe-area-top">
        <div class="page-brand">
          <span class="page-brand__mark">AI</span>
          <div>
            <h1 class="page-brand__title">{{ title }}</h1>
            <p class="page-brand__sub">{{ chats.length }} 个会话</p>
          </div>
        </div>
        <button type="button" class="page-clear" aria-label="clear" @click="clearAllChatHistory">
          <van-icon name="delete-o" size="18" />
        </button>
      </header>

      <button type="button" class="new-chat-btn" @click="showPicker = true">
        <van-icon name="plus" size="16" />
        <span>新建 AI 对话</span>
      </button>

      <van-search
        v-model="chatName"
        shape="round"
        placeholder="搜索会话标题"
        class="page-search"
        @input="search"
      />

      <div class="list-panel">
        <van-list
          v-model:error="error"
          v-model:loading="loading"
          :finished="finished"
          error-text="请求失败，点击重新加载"
          finished-text="没有更多了"
          @load="onLoad"
        >
          <div class="chat-list">
            <van-swipe-cell v-for="item in chats" :key="item.id" class="chat-item-wrapper">
              <div class="chat-item" @click="changeChat(item)">
                <div class="chat-item-body">
                  <div class="chat-title">{{ item.title }}</div>
                </div>
                <van-icon name="arrow" class="chat-arrow" size="14" />
              </div>
              <template #right>
                <van-button square text="修改" type="primary" class="action-btn edit-btn" @click.stop="editChat(item)"/>
                <van-button square text="删除" type="danger" class="action-btn delete-btn" @click.stop="removeChat(item)"/>
              </template>
            </van-swipe-cell>
          </div>
        </van-list>
      </div>

      <div class="bottom-spacing"></div>
    </div>

    <van-popup v-model:show="showPicker" position="bottom" round class="popup">
      <van-picker
        :columns="columns"
        title="选择AI模型"
        @cancel="showPicker = false"
        @confirm="newChat"
      >
        <template #option="item">
          <div class="picker-option">
            <van-image
              v-if="item.icon"
              :src="item.icon"
              fit="cover"
              round
            />
            <span v-if="item.text=='通用AI助手'">{{ item.text }}</span>
            <span v-else>{{ item.text }} ({{ item.power }})</span>
          </div>
        </template>
      </van-picker>
    </van-popup>

    <van-dialog v-model:show="showEditChat" title="修改对话标题" show-cancel-button class="dialog" @confirm="saveTitle">
      <van-field v-model="tmpChatTitle" label="" placeholder="请输入对话标题" class="field"/>
    </van-dialog>
  </div>
</template>

<script setup>
import {ref} from "vue";
import {httpGet, httpPost} from "@/utils/http";
import {showConfirmDialog, showFailToast, showSuccessToast} from "vant";
import {checkSession} from "@/action/session";
import {router} from "@/router";
import {setChatConfig} from "@/store/chat";
import {removeArrayItem} from "@/utils/libs";

const title = ref("对话列表")
const chatName = ref("")
const chats = ref([])
const allChats = ref([])
const loading = ref(false)
const finished = ref(false)
const error = ref(false)
const loginUser = ref(null)
const isLogin = ref(false)
const roles = ref([])
const models = ref([])
const showPicker = ref(false)
const columns = ref([roles.value, models.value])
const showEditChat = ref(false)
const item = ref({})
const tmpChatTitle = ref("")

checkSession().then((user) => {
  loginUser.value = user
  isLogin.value = true
  // 加载角色列表
  httpGet(`/api/role/list?user_id=${user.id}`).then((res) => {
    if (res.data) {
      const items = res.data
      for (let i = 0; i < items.length; i++) {
        roles.value.push({
          text: items[i].name,
          value: items[i].id,
          icon: items[i].icon,
          helloMsg: items[i].hello_msg
        })
      }
    }
  }).catch(() => {
    showFailToast("加载聊天角色失败")
  })

  // 加载模型
  httpGet('/api/model/list?enable=1').then(res => {
    if (res.data) {
      const items = res.data
      for (let i = 0; i < items.length; i++) {
        models.value.push({text: items[i].name, value: items[i].id,power:items[i].power})
      }
    }
  }).catch(e => {
    showFailToast("加载模型失败: " + e.message)
  })

}).catch(() => {
  router.push("/login")
})

const onLoad = () => {
  httpGet("/api/chat/list?user_id=" + loginUser.value.id).then((res) => {
    if (res.data) {
      chats.value = res.data;
      allChats.value = res.data;
      finished.value = true
    }
    loading.value = false;
  }).catch(() => {
    error.value = true
    showFailToast("加载会话列表失败")
  })
};

const getModelValue = (model_id) => {
  for (let i = 0; i < models.value.length; i++) {
    if (models.value[i].value === model_id) {
      return models.value[i].text
    }
  }
  return ""
}

const search = () => {
  if (chatName.value === '') {
    chats.value = allChats.value
    return
  }
  const items = [];
  for (let i = 0; i < allChats.value.length; i++) {
    if (allChats.value[i].title.toLowerCase().indexOf(chatName.value.toLowerCase()) !== -1) {
      items.push(allChats.value[i]);
    }
  }
  chats.value = items;
}

const clearAllChatHistory = () => {
  showConfirmDialog({
    title: '操作提示',
    message: '确定要删除所有的会话记录吗？'
  }).then(() => {
    httpGet("/api/chat/clear").then(() => {
      showSuccessToast('所有聊天记录已清空')
      chats.value = [];
    }).catch(e => {
      showFailToast("操作失败：" + e.message)
    })
  }).catch(() => {
    // on cancel
  })
}

const newChat = (item) => {
  showPicker.value = false
  const options = item.selectedOptions
  setChatConfig({
    role: {
      id: options[0].value,
      name: options[0].text,
      icon: options[0].icon,
      helloMsg: options[0].helloMsg
    },
    model: options[1].value,
    modelValue: getModelValue(options[1].value),
    title: '新建AI对话',
    chatId: 0
  })
  router.push('/mobile/chat/session')
}

const changeChat = (chat) => {
  let role = {}
  for (let i = 0; i < roles.value.length; i++) {
    if (roles.value[i].value === chat.role_id) {
      role = roles.value[i]
      break
    }
  }
  const chatIdValue = chat.chat_id || chat.id;
  setChatConfig({
    role: {
      id: chat.role_id,
      name: role.text,
      icon: role.icon
    },
    model: chat.model_id,
    modelValue: getModelValue(chat.model_id),
    title: chat.title,
    chatId: chatIdValue,
    helloMsg: chat.hello_msg,
  })
  router.push('/mobile/chat/session')
}

const editChat = (row) => {
  showEditChat.value = true
  item.value = row
  tmpChatTitle.value = row.title
}
const saveTitle = () => {
  httpPost('/api/chat/update', {id: item.value.id, title: tmpChatTitle.value}).then(() => {
    showSuccessToast("操作成功！");
    item.value.title = tmpChatTitle.value;
  }).catch(e => {
    showFailToast("操作失败：" + e.message);
  })
}

const removeChat = (item) => {
  httpGet('/api/chat/remove?chat_id=' + item.chat_id).then(() => {
    chats.value = removeArrayItem(chats.value, item, function (e1, e2) {
      return e1.id === e2.id
    })
  }).catch(e => {
    showFailToast('操作失败：' + e.message);
  })
}
</script>

<style lang="stylus" scoped>
@import "@/assets/css/mobile/chat-list.styl"
</style>
