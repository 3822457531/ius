<template>
  <div class="app-background chat-layout-root">
    <div
      class="mobile-chat chat-layout"
      v-loading="loading"
      :element-loading-text="t('chat.loadingSession')"
    >
      <div class="chat-nav-wrap" ref="navBarRef">
        <header class="chat-topbar van-safe-area-top">
          <button type="button" class="chat-topbar__btn" :aria-label="t('chat.historyTitle')" @click="showHistory = true">
            <van-icon name="wap-nav" size="20" />
          </button>
          <button type="button" class="chat-topbar__model" @click="openModelPicker">
            <span v-if="webSearchEnabled" class="chat-topbar__web-pill">{{ t('chat.webSearchBadge') }}</span>
            <span class="chat-topbar__model-name">{{ displayModelName || t('chat.navTitle') }}</span>
            <van-icon name="arrow-down" size="12" />
          </button>
          <div class="chat-topbar__right">
            <button type="button" class="chat-topbar__btn" :aria-label="t('chat.historyNew')" @click="createNewChat">
              <van-icon name="edit" size="18" />
            </button>
            <button type="button" class="chat-topbar__btn" aria-label="profile" @click="goMyPage">
              <van-icon name="user-o" size="18" />
            </button>
          </div>
        </header>
      </div>

      <van-share-sheet
          v-model:show="showShare"
          :title="t('chat.shareTitle')"
          :options="shareOptions"
          @select="shareChat"
      />

      <!-- 对话历史抽屉 -->
      <van-popup
        v-model:show="showHistory"
        position="left"
        :style="{ width: '78%', maxWidth: '320px', height: '100%' }"
        class="history-popup"
      >
        <div class="history-container">
          <div class="history-header-panel van-safe-area-top">
            <div class="history-brand-row">
              <div class="history-brand">
                <span class="history-brand__mark">AI</span>
                <span class="history-brand__title">{{ t('chat.navTitle') }}</span>
              </div>
              <button type="button" class="history-close-btn" @click="showHistory = false">
                <van-icon name="cross" />
              </button>
            </div>
            <button type="button" class="history-new-btn" @click="createNewChat">
              <van-icon name="plus" />
              <span>{{ t('chat.historyNew') }}</span>
            </button>
            <van-search
              v-model="chatName"
              shape="round"
              :placeholder="t('chat.historySearchPlaceholder')"
              class="history-search"
              @update:model-value="searchHistory"
            />
          </div>

          <div class="history-content">
            <van-empty
              v-if="historyFinished && !historyLoading && historyChats.length === 0"
              class="history-empty"
              image="search"
              :description="t('chat.historyEmpty')"
            />
            <van-list
              v-else
              v-model:error="historyError"
              v-model:loading="historyLoading"
              :finished="historyFinished"
              :error-text="t('chat.listErrorText')"
              :finished-text="t('chat.listFinishedText')"
              @load="loadHistory"
            >
              <div class="history-list">
                <van-swipe-cell
                  v-for="item in historyChats"
                  :key="getChatItemId(item)"
                  class="history-item-wrapper"
                >
                  <div
                    class="history-item"
                    :class="{ 'is-active': isCurrentChat(item) }"
                    @click="switchChat(item)"
                  >
                    <div class="history-item-body">
                      <div class="history-chat-title van-ellipsis">
                        {{ item.title || t('chat.untitledChat') }}
                      </div>
                      <div v-if="getHistorySubtitle(item)" class="history-chat-subtitle van-ellipsis">
                        {{ getHistorySubtitle(item) }}
                      </div>
                    </div>
                    <van-icon
                      name="delete-o"
                      class="history-delete-icon"
                      @click.stop="removeHistoryChat(item)"
                    />
                  </div>
                  <template #right>
                    <van-button
                      square
                      type="danger"
                      class="history-swipe-delete"
                      :text="t('chat.deleteBtn')"
                      @click="removeHistoryChat(item)"
                    />
                  </template>
                </van-swipe-cell>
              </div>
            </van-list>
          </div>

          <div class="history-footer">
            <button type="button" class="history-footer-btn" @click="goMyPage">
              <van-icon name="user-o" size="16" />
              <span>{{ t('chat.profileBtn') }}</span>
            </button>
          </div>
        </div>
      </van-popup>

      <!-- 空状态：切换模型 -->
      <van-popup
        v-model:show="showModelPicker"
        position="bottom"
        round
        class="model-picker-popup"
      >
        <div class="model-picker-panel">
          <div class="model-picker-header">
            <span class="model-picker-title">{{ t('chat.pickModelTitle') }}</span>
            <van-icon name="cross" class="model-picker-close" @click="showModelPicker = false" />
          </div>
          <div class="model-picker-list">
            <div
              v-for="item in models"
              :key="item.value"
              class="model-picker-item"
              :class="{ active: item.value === currentModel }"
              @click="onPickModel(item)"
            >
              <div class="model-picker-item-main">
                <span class="model-picker-name">{{ item.text }}</span>
                <span v-if="item.power" class="model-picker-power">({{ item.power }})</span>
              </div>
              <van-icon v-if="item.value === currentModel" name="success" color="#0f766e" />
            </div>
            <div v-if="models.length === 0" class="model-picker-empty">{{ t('chat.noModelsAvailable') }}</div>
          </div>
        </div>
      </van-popup>

      <div class="chat-list-wrapper">
        <div id="message-list-box" :class="{ 'loading': loading }" class="message-list-box">
          <van-list
            v-model:error="error"
            :finished="finished"
            :error-text="t('chat.listErrorText')"
            @load="onLoad"
          >
              <div v-if="chatData.length === 0 && !loading" class="chat-empty-wrap">
                <div class="chat-empty-card">
                  <div class="chat-empty-mark" aria-hidden="true">AI</div>
                  <h2 class="chat-empty-title">{{ t('chat.emptyWelcomeTitle') }}</h2>
                  <p class="chat-empty-desc">{{ emptyWelcomeDesc }}</p>
                  <p v-if="emptyModelLabel" class="chat-empty-model">{{ emptyModelLabel }}</p>
                  <div class="chat-empty-tpl">
                    <div class="chat-empty-tpl__head">
                      <div class="chat-empty-tpl__label">{{ t('chat.promptTplTitle') }}</div>
                      <div class="chat-empty-tpl__hint">{{ t('chat.promptTplHint') }}</div>
                    </div>
                    <div class="chat-empty-tpl__grid">
                      <button
                        v-for="item in chatPromptTemplates"
                        :key="item.key"
                        type="button"
                        class="chat-empty-tpl__card"
                        :class="'chat-empty-tpl__card--' + item.tone"
                        @click="applyChatPromptTpl(item.key)"
                      >
                        <span class="chat-empty-tpl__icon" aria-hidden="true">
                          <van-icon :name="item.icon" size="18" />
                        </span>
                        <span class="chat-empty-tpl__text">
                          <span class="chat-empty-tpl__name">{{ item.name }}</span>
                          <span class="chat-empty-tpl__desc">{{ item.desc }}</span>
                        </span>
                      </button>
                    </div>
                  </div>
                </div>
              </div>
              <van-cell v-for="item in chatData" :key="item.id" :border="false" class="message-line">
                <chat-prompt
                    v-if="item.type === 'prompt'"
                    :content="item.content"
                    :created-at="dateFormat(item['created_at'])"
                    :icon="item.icon"
                    :model="model"
                    :tokens="item['tokens']"/>
                <chat-reply v-else-if="item.type === 'reply'"
                            :content="item.content"
                            :created-at="dateFormat(item['created_at'])"
                            :icon="item.icon"
                            :org-content="item.orgContent"
                            :thinking="!!item.thinking"
                            :tokens="item['tokens']"
                            :used-web-search="!!item.webSearch"/>
              </van-cell>
          </van-list>
        </div>
      </div>

      <div class="chat-box-wrapper" ref="bottomBarRef">
          <div v-if="models.length" class="model-selector">
            <div class="model-scroll">
              <div
                v-for="item in models"
                :key="item.value"
                class="model-chip"
                :class="{ active: item.value === currentModel }"
                @click="selectModel(item)"
              >
                <span class="model-name">{{ item.text }}</span>
                <span class="model-power" v-if="item.power">({{ item.power }})</span>
              </div>
            </div>
          </div>

          <div class="composer">
            <div class="composer__web-search" :class="{ 'composer__web-search--on': webSearchEnabled }">
              <button
                type="button"
                class="web-search-chip"
                :class="{ active: webSearchEnabled }"
                @click="webSearchEnabled = !webSearchEnabled"
                :aria-pressed="webSearchEnabled"
              >
                <span v-if="webSearchEnabled" class="web-search-chip__pulse" aria-hidden="true"></span>
                <van-icon :name="webSearchEnabled ? 'globe-o' : 'search'" size="15" />
                <span class="web-search-chip__label">{{ webSearchEnabled ? t('chat.webSearchActive') : t('chat.webSearchToggle') }}</span>
              </button>
            </div>
            <div class="composer__shell">
              <textarea
                  ref="promptInputRef"
                  v-model="prompt"
                  class="composer__textarea"
                  rows="1"
                  @input="autoResize"
                  @keydown.enter.exact.prevent="sendMessage"
                  :placeholder="t('chat.inputPlaceholder')"
              ></textarea>
              <button
                type="button"
                class="composer__send"
                :class="{ 'composer__send--stop': showStopGenerate }"
                :disabled="!showStopGenerate && !prompt.trim()"
                :aria-label="showStopGenerate ? t('chat.stopBtn') : t('chat.sendBtn')"
                @click="onComposerAction"
              >
                <span v-if="showStopGenerate" class="composer__send-square" aria-hidden="true"></span>
                <van-icon v-else name="upgrade" size="18" />
              </button>
            </div>
          </div>
      </div>

      <button id="copy-link-btn" style="display: none;" :data-clipboard-text="url">
        {{ t('chat.copyLinkBtn') }}
      </button>

      <van-overlay :show="showMic" z-index="100">
        <div class="mic-wrapper">
          <div class="image">
            <van-image
                width="100"
                height="100"
                src="/images/mic.gif"
            />
          </div>
          <van-button type="success" @click="stopVoice">
            {{ t('chat.voiceDoneBtn') }}
          </van-button>
        </div>
      </van-overlay>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { showConfirmDialog, showImagePreview, showNotify, showSuccessToast, showToast, showFailToast } from "vant";
import { onBeforeRouteLeave, useRouter } from "vue-router";
import { dateFormat, processContent, randString, renderInputText, UUID } from "@/utils/libs";
import { getChatConfig, setChatConfig, removeChatConfig } from "@/store/chat";
import { httpGet, resolveWsHost } from "@/utils/http";
import hl from "highlight.js";
import 'highlight.js/styles/a11y-dark.css';
import ChatPrompt from "@/components/mobile/ChatPrompt.vue";
import ChatReply from "@/components/mobile/ChatReply.vue";
import { getSessionId, getUserToken } from "@/store/session";
import { checkSession } from "@/action/session";
import Clipboard from "clipboard";

const { t } = useI18n();

const winHeight = ref(0);
const navBarRef = ref(null);
const bottomBarRef = ref(null);
const router = useRouter();

// 读取本地会话配置；为空时在登录后兜底创建一份
let chatConfig = getChatConfig();

const role = chatConfig ? chatConfig.role : {};
const model = chatConfig ? chatConfig.model : '';
const modelValue = chatConfig ? chatConfig.modelValue : '';
const title = ref(chatConfig ? chatConfig.title : '');
// 当前会话 ID，可能在新建会话时由后端/前端生成
const chatId = ref(chatConfig ? chatConfig.chatId : 0);
const loginUser = ref(null);
const sessionChecked = ref(false); // 标记 checkSession 是否已完成
const showMic = ref(false);
const showHistory = ref(false);
const showModelPicker = ref(false);
const promptInputRef = ref(null);

const url = ref(location.protocol + '//' + location.host + '/mobile/chat/export?chat_id=' + chatId.value);

const chatData = ref([]);
const loading = ref(false); // 控制历史加载状态
const finished = ref(true); // 初始设为 true，等待 checkSession 完成后再加载
const error = ref(false);

const prompt = ref('');
const showStopGenerate = ref(false); // 停止生成
const showReGenerate = ref(false); // 重新生成
const webSearchEnabled = ref(false)
const previousText = ref(''); // 上一次提问
const lineBuffer = ref(''); // 输出缓冲行
const socket = ref(null);
const activelyClose = ref(false); // 主动关闭
const canSend = ref(true);
const heartbeatHandle = ref(null);

// 历史会话相关
const chatName = ref("");
const historyChats = ref([]);
const allHistoryChats = ref([]);
const historyLoading = ref(false);
const historyFinished = ref(false);
const historyError = ref(false);

// 模型列表
const models = ref([]);
const currentModel = ref(model);
const currentModelLabel = ref(modelValue || '');

const emptyStateIcon = computed(() => {
  const cfg = getChatConfig();
  return (cfg && cfg.role && cfg.role.icon) || (role && role.icon) || '/images/gpt-icon.png';
});

const emptyWelcomeDesc = computed(() => {
  const cfg = getChatConfig();
  const hello = (cfg && cfg.helloMsg) || (cfg && cfg.role && cfg.role.helloMsg) || (role && role.helloMsg);
  if (hello) return hello;
  return t('chat.emptyWelcomeDesc');
});

// 随 currentModel / models 变化，不依赖 localStorage（非响应式）
const displayModelName = computed(() => {
  const hit = models.value.find((m) => m.value === currentModel.value);
  if (hit) return hit.text;
  if (currentModelLabel.value) return currentModelLabel.value;
  return modelValue || '';
});

const emptyModelLabel = computed(() => {
  const name = displayModelName.value;
  if (!name) return '';
  return t('chat.emptyCurrentModel', { model: name });
});

const emptyTips = computed(() => [
  { id: 'ask', icon: 'chat-o', text: t('chat.emptyTipAsk') },
  { id: 'model', icon: 'setting-o', text: t('chat.emptyTipModel') },
  { id: 'history', icon: 'clock-o', text: t('chat.emptyTipHistory') },
]);

const chatPromptTplMeta = {
  douyin: { icon: 'video-o', tone: 'coral' },
  brand: { icon: 'star-o', tone: 'indigo' },
  ppt: { icon: 'notes-o', tone: 'sky' },
  write: { icon: 'edit', tone: 'teal' },
  summarize: { icon: 'orders-o', tone: 'slate' },
  polish: { icon: 'comment-o', tone: 'amber' },
  plan: { icon: 'calendar-o', tone: 'green' },
  ideas: { icon: 'fire-o', tone: 'purple' },
}
const chatPromptTplKeys = Object.keys(chatPromptTplMeta)
const chatPromptTemplates = computed(() =>
  chatPromptTplKeys.map((key) => ({
    key,
    icon: chatPromptTplMeta[key].icon,
    tone: chatPromptTplMeta[key].tone,
    name: t(`chat.promptTpl.${key}.name`),
    desc: t(`chat.promptTpl.${key}.desc`),
  }))
)

const applyChatPromptTpl = (key) => {
  prompt.value = t(`chat.promptTpl.${key}.prompt`)
  showToast(t('chat.promptTplFilled'))
  nextTick(() => {
    autoResize()
    focusPromptInput()
  })
}

const onEmptyTipClick = (tip) => {
  if (tip.id === 'ask') {
    focusPromptInput();
    return;
  }
  if (tip.id === 'model') {
    openModelPicker();
    return;
  }
  if (tip.id === 'history') {
    showHistory.value = true;
  }
};

const focusPromptInput = () => {
  nextTick(() => {
    const el = promptInputRef.value;
    if (!el) return;
    el.focus();
    el.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
  });
};

const openModelPicker = () => {
  if (!models.value.length) {
    loadModels();
  }
  showModelPicker.value = true;
};

const onPickModel = (item) => {
  switchModelInPlace(item);
  showModelPicker.value = false;
};

const switchModelInPlace = (item) => {
  if (currentModel.value === item.value) return;

  const currentConfig = getChatConfig();
  const currentRole = currentConfig && currentConfig.role ? currentConfig.role : role;

  if (!currentRole || !currentRole.id) {
    showFailToast(t('chat.roleConfigMissing'));
    return;
  }

  currentModel.value = item.value;
  currentModelLabel.value = item.text;
  setChatConfig({
    role: {
      id: currentRole.id,
      name: currentRole.name,
      icon: currentRole.icon,
      helloMsg: currentRole.helloMsg || currentRole.hello_msg,
    },
    model: item.value,
    modelValue: item.text,
    title: (currentConfig && currentConfig.title) || title.value,
    chatId: chatId.value || (currentConfig ? currentConfig.chatId : 0),
    userId: loginUser.value ? loginUser.value.id : (currentConfig ? currentConfig.userId : undefined),
    helloMsg: currentConfig ? currentConfig.helloMsg : undefined,
  });

  closeChatSocket();
  const finalChatId = chatId.value || (currentConfig ? currentConfig.chatId : 0);
  connect(finalChatId, currentRole.id);
};

const closeChatSocket = () => {
  if (socket.value !== null) {
    activelyClose.value = true;
    clearTimeout(heartbeatHandle.value);
    socket.value.close();
    socket.value = null;
  }
};

const syncFromChatConfig = () => {
  chatConfig = getChatConfig();
  if (!chatConfig) return;
  chatId.value = chatConfig.chatId || 0;
  title.value = chatConfig.title || t('chat.newChatTitle');
  currentModel.value = chatConfig.model || '';
  currentModelLabel.value = chatConfig.modelValue || getModelValue(chatConfig.model) || '';
  url.value = `${location.protocol}//${location.host}/mobile/chat/export?chat_id=${chatId.value}`;
};

/** 不整页刷新，原地切换/新建会话 */
const restartSession = () => {
  closeChatSocket();
  syncFromChatConfig();
  canSend.value = true;
  showStopGenerate.value = false;
  showReGenerate.value = false;
  previousText.value = '';
  lineBuffer.value = '';
  finished.value = false;
  loading.value = false;
  onLoad();
};

// 跳转到「我的」页面
const goMyPage = () => {
  router.push('/mobile/invitation');
};

// 计算中间消息区域高度（当前布局已用 flex 固定，此函数保留供兼容或滚动等逻辑使用）
const calcWinHeight = () => {
  if (!navBarRef.value || !bottomBarRef.value) return;
  const viewHeight = window.innerHeight || document.documentElement.clientHeight || document.body.offsetHeight;
  const navEl = navBarRef.value.$el || navBarRef.value;
  const bottomEl = bottomBarRef.value.$el || bottomBarRef.value;
  const navHeight = (navEl && navEl.offsetHeight) || 0;
  const bottomHeight = (bottomEl && bottomEl.offsetHeight) || 0;
  winHeight.value = viewHeight - navHeight - bottomHeight;
};

// Ensure winHeight updates dynamically
watch(() => [navBarRef.value, bottomBarRef.value, models.value.length], () => {
  nextTick(() => calcWinHeight());
});

const handleResize = () => {
  calcWinHeight();
};

onMounted(() => {
  nextTick(() => {
    calcWinHeight();
    // 不在 onMounted 中直接调用 onLoad()，因为此时 loginUser.value 可能还未设置
    // checkSession() 的回调中会处理配置初始化和历史记录加载
  });

  window.addEventListener('resize', handleResize);

  const clipboard = new Clipboard(".content-mobile,.copy-code-mobile,#copy-link-btn");
  clipboard.on('success', (e) => {
    e.clearSelection();
    showNotify({ type: 'success', message: t('chat.copySuccess'), duration: 1000 });
  });
  clipboard.on('error', () => {
    showNotify({ type: 'danger', message: t('chat.copyFail'), duration: 2000 });
  });
});

onUnmounted(() => {
  window.removeEventListener('resize', handleResize);
  if (socket.value !== null) {
    activelyClose.value = true;
    clearTimeout(heartbeatHandle.value);
    socket.value.close();
  }
});

const bootstrapChatConfig = async (user) => {
  // 1) 优先使用已有会话列表的第一条
  try {
    const listRes = await httpGet(`/api/chat/list?user_id=${user.id}`);
    if (listRes.data && listRes.data.length > 0) {
      const first = listRes.data[0];
      const targetChatId = first.chat_id || first.id;
      const config = {
        role: {
          id: first.role_id,
          name: first.role_name,
          icon: first.icon,
          helloMsg: first.hello_msg,
        },
        model: first.model_id,
        modelValue: first.model_name || '',
        title: first.title || t('chat.newChatTitle'),
        chatId: targetChatId,
        userId: user.id,
        helloMsg: first.hello_msg,
      };
      setChatConfig(config);
      chatConfig = config;
      // 同步 currentModel
      currentModel.value = config.model;
      restartSession();
      loadHistory();
      return true;
    }
  } catch (e) {
    // ignore, fallback to模型/角色创建
  }

  // 2) 没有会话，则取可用的角色与模型生成一份默认配置
  try {
    const [roleRes, modelRes] = await Promise.all([
      httpGet(`/api/role/list?user_id=${user.id}`),
      httpGet('/api/model/list?enable=1'),
    ]);
    const roles = roleRes.data || [];
    const modelsList = modelRes.data || [];
    if (roles.length > 0 && modelsList.length > 0) {
      const pickedRole = roles[0];
      const pickedModel = modelsList[0];
      const config = {
        role: {
          id: pickedRole.id,
          name: pickedRole.name,
          icon: pickedRole.icon,
          helloMsg: pickedRole.hello_msg,
        },
        model: pickedModel.id,
        modelValue: pickedModel.name,
        title: t('chat.newChatTitle'),
        chatId: 0,
        userId: user.id,
        helloMsg: pickedRole.hello_msg,
      };
      setChatConfig(config);
      chatConfig = config;
      // 同步 currentModel
      currentModel.value = config.model;
      restartSession();
      loadHistory();
      return true;
    }
  } catch (e) {
    // ignore
  }
  // 如果无法创建，也保持在会话页，避免落回列表路由
  return false;
};

checkSession().then(user => {
  // 确保用户信息被正确设置
  if (!user || !user.id) {
    sessionChecked.value = true; // 标记为已完成（即使失败）
    router.push('/login');
    return;
  }
  
  loginUser.value = user;
  sessionChecked.value = true; // 标记 checkSession 已完成

  // 加载模型列表（不依赖 chatConfig，只要有用户就可以加载）
  loadModels();

  // 重新读取最新的配置
  chatConfig = getChatConfig();

  // 如果本地会话属于其他用户，清空旧会话配置并保持在会话页
  if (chatConfig && chatConfig.userId && chatConfig.userId !== user.id) {
    removeChatConfig();
    chatConfig = null;
    bootstrapChatConfig(user);
    return;
  }

  // 如果缺少 userId 字段，为当前用户补齐，避免下一次登录串号
  if (chatConfig && !chatConfig.userId) {
    chatConfig = { ...chatConfig, userId: user.id };
    setChatConfig(chatConfig);
  }

  // 新用户/无配置时，尝试自动补全配置并重新进入本页
  if (!chatConfig || !chatConfig.model || !chatConfig.role || !chatConfig.role.id) {
    bootstrapChatConfig(user);
    return;
  }

  // 配置存在时，确保 loginUser.value 已设置后再加载
  // 同步 currentModel 和 chatConfig.model
  if (chatConfig && chatConfig.model) {
    currentModel.value = chatConfig.model;
  }
  
  // 将 finished 设为 false，允许 van-list 触发加载
  finished.value = false;
  // 加载当前会话的消息历史
  onLoad();
  // 加载会话历史列表
  loadHistory();
}).catch((err) => {
  console.error('checkSession failed:', err);
  sessionChecked.value = true; // 标记为已完成（即使失败）
  router.push('/login');
});

const latexPlugin = require('markdown-it-latex2img');
const mathjaxPlugin = require('markdown-it-mathjax');
const md = require('markdown-it')({
  breaks: true,
  html: true,
  linkify: true,
  typographer: true,
  highlight: function (str, lang) {
    const codeIndex = parseInt(Date.now()) + Math.floor(Math.random() * 10000000);
    const copyBtn = `<span class="copy-code-mobile" data-clipboard-action="copy" data-clipboard-target="#copy-target-${codeIndex}">${t('chat.copyCodeBtn')}</span>
<textarea style="position: absolute;top: -9999px;left: -9999px;z-index: -9999;" id="copy-target-${codeIndex}">${str.replace(/<\/textarea>/g, '</textarea>')}</textarea>`;
    if (lang && hl.getLanguage(lang)) {
      const langHtml = `<span class="lang-name">${lang}</span>`;
      const preCode = hl.highlight(lang, str, true).value;
      return `<pre class="code-container"><code class="language-${lang} hljs">${preCode}</code>${copyBtn} ${langHtml}</pre>`;
    }
    const preCode = md.utils.escapeHtml(str);
    return `<pre class="code-container"><code class="language-${lang} hljs">${preCode}</code>${copyBtn}</pre>`;
  }
});
md.use(latexPlugin);
md.use(mathjaxPlugin);

const onLoad = async () => {
  // 如果 checkSession 还没完成，等待它完成
  if (!sessionChecked.value) {
    // 等待一小段时间后重试，或者直接返回（因为 checkSession 的回调会调用 onLoad）
    return;
  }
  
  if (loading.value) return;
  loading.value = true;
  error.value = false;
  finished.value = false;
  chatData.value = [];

  // 确保 chatConfig 存在且包含必要的 model 和 role
  let currentConfig = getChatConfig();
  if (!currentConfig || !currentConfig.model || !currentConfig.role || !currentConfig.role.id) {
    // 如果没有配置，尝试加载模型和角色
    if (loginUser.value) {
      try {
        const [roleRes, modelRes] = await Promise.all([
          httpGet(`/api/role/list?user_id=${loginUser.value.id}`),
          httpGet('/api/model/list?enable=1'),
        ]);
        const roles = roleRes.data || [];
        const modelsList = modelRes.data || [];
        if (roles.length > 0 && modelsList.length > 0) {
          const pickedRole = roles[0];
          const pickedModel = modelsList[0];
          currentConfig = {
            role: {
              id: pickedRole.id,
              name: pickedRole.name,
              icon: pickedRole.icon,
              helloMsg: pickedRole.hello_msg,
            },
            model: pickedModel.id,
            modelValue: pickedModel.name,
            title: t('chat.newChatTitle'),
            chatId: chatId.value || 0,
            userId: loginUser.value.id,
            helloMsg: pickedRole.hello_msg,
          };
          setChatConfig(currentConfig);
          chatConfig = currentConfig;
          // 同步 currentModel
          currentModel.value = currentConfig.model;
        } else {
          showFailToast(t('chat.noModelsOrRoles'));
          loading.value = false;
          return;
        }
      } catch (e) {
        showFailToast(t('chat.loadModelsOrRolesFailPrefix') + (e.message || ''));
        loading.value = false;
        return;
      }
    } else {
      showFailToast(t('chat.userNotLogin'));
      loading.value = false;
      return;
    }
  }

  const finalRoleId = currentConfig.role.id;
  const finalChatId = chatId.value || currentConfig.chatId || 0;
  
  // 同步 currentModel 和 chatConfig.model
  if (currentConfig.model && currentModel.value !== currentConfig.model) {
    currentModel.value = currentConfig.model;
  }

  httpGet('/api/chat/history?chat_id=' + finalChatId).then(res => {
    finished.value = true;
    const data = res.data;
    if (data && data.length > 0) {
      for (let i = 0; i < data.length; i++) {
        if (data[i].type === "prompt") {
          chatData.value.push(data[i]);
          continue;
        }
        data[i].orgContent = data[i].content;
        data[i].content = md.render(processContent(data[i].content));
        chatData.value.push(data[i]);
      }
      nextTick(() => {
        hl.configure({ ignoreUnescapedHTML: true });
        const blocks = document.querySelector("#message-list-box").querySelectorAll('pre code');
        blocks.forEach((block) => {
          hl.highlightElement(block);
        });
        scrollListBox();
      });
    }
    connect(finalChatId, finalRoleId);
  }).catch(() => {
    error.value = true;
    loading.value = false;
  });
};

// 加载模型列表
const loadModels = () => {
  httpGet('/api/model/list?enable=1').then(res => {
    if (res.data) {
      const items = res.data;
      models.value = [];
      for (let i = 0; i < items.length; i++) {
        models.value.push({ text: items[i].name, value: items[i].id, power: items[i].power });
      }
      
      // 如果当前没有选中的模型，自动选中第一个模型
      if (!currentModel.value && models.value.length > 0) {
        const firstModel = models.value[0];
        currentModel.value = firstModel.value;
        
        // 更新 chatConfig，确保模型配置存在
        const currentConfig = getChatConfig();
        if (currentConfig) {
          // 如果配置中没有模型，更新配置
          if (!currentConfig.model || currentConfig.model !== firstModel.value) {
            setChatConfig({
              ...currentConfig,
              model: firstModel.value,
              modelValue: firstModel.text,
            });
          }
        } else if (loginUser.value) {
          // 如果没有配置，创建一个默认配置
          // 这里需要等待角色加载，所以先不创建完整配置
          // 但至少确保模型被选中
        }
      }
      if (currentModel.value) {
        const name = getModelValue(currentModel.value);
        if (name) currentModelLabel.value = name;
      }
    }
  }).catch(e => {
    showFailToast(t('chat.loadModelsFailPrefix') + e.message);
  });
};

// 获取模型展示名称
const getModelValue = (model_id) => {
  for (let i = 0; i < models.value.length; i++) {
    if (models.value[i].value === model_id) {
      return models.value[i].text;
    }
  }
  return "";
};

// 加载历史会话列表
const loadHistory = () => {
  if (!loginUser.value) return;
  historyLoading.value = true;
  httpGet("/api/chat/list?user_id=" + loginUser.value.id).then((res) => {
        if (res.data) {
      historyChats.value = res.data;
      allHistoryChats.value = res.data;
      historyFinished.value = true;
    }
    historyLoading.value = false;
  }).catch(() => {
    historyError.value = true;
    historyLoading.value = false;
    showFailToast(t('chat.loadHistoryFail'));
  });
};

const getChatItemId = (chat) => chat.chat_id || chat.id;

const isCurrentChat = (chat) => String(chatId.value) === String(getChatItemId(chat));

const getHistorySubtitle = (chat) => {
  if (chat.model_name) return chat.model_name;
  if (chat.model_id) return getModelValue(chat.model_id);
  return '';
};

// 打开侧栏时刷新列表
watch(showHistory, (visible) => {
  if (visible && loginUser.value) {
    loadHistory();
  }
});

// 删除历史会话
const removeHistoryChat = (chat) => {
  const cid = getChatItemId(chat);
  showConfirmDialog({
    title: t('chat.deleteConfirmTitle'),
    message: t('chat.deleteConfirmMessage'),
  }).then(() => {
    httpGet('/api/chat/remove?chat_id=' + cid).then(() => {
      showSuccessToast(t('chat.deleteSuccess'));
      const keepItem = (item) => String(getChatItemId(item)) !== String(cid);
      historyChats.value = historyChats.value.filter(keepItem);
      allHistoryChats.value = allHistoryChats.value.filter(keepItem);
      if (isCurrentChat(chat)) {
        createNewChat();
      }
    }).catch((e) => {
      showFailToast(t('chat.deleteFailPrefix') + (e.message || ''));
    });
  }).catch(() => {});
};

// 搜索历史会话
const searchHistory = () => {
  if (chatName.value === "") {
    historyChats.value = allHistoryChats.value;
    return;
  }
  const items = [];
  for (let i = 0; i < allHistoryChats.value.length; i++) {
    if (allHistoryChats.value[i].title.toLowerCase().indexOf(chatName.value.toLowerCase()) !== -1) {
      items.push(allHistoryChats.value[i]);
    }
  }
  historyChats.value = items;
};

// 切换会话
const switchChat = (chat) => {
  showHistory.value = false;
  const targetChatId = chat.chat_id || chat.id;
  setChatConfig({
    role: {
      id: chat.role_id,
      name: chat.role_name || role.name,
      icon: chat.icon || role.icon
    },
    model: chat.model_id,
    modelValue: getModelValue(chat.model_id) || modelValue,
    title: chat.title,
    chatId: targetChatId,
    helloMsg: chat.hello_msg,
    userId: loginUser.value ? loginUser.value.id : (chatConfig ? chatConfig.userId : undefined),
  });
  restartSession();
};

// 在历史侧边栏里点击「新建会话」
const createNewChat = () => {
  showHistory.value = false;
  const currentConfig = getChatConfig();
  const currentRole = (currentConfig && currentConfig.role) || role;
  const targetModel = currentModel.value || (currentConfig && currentConfig.model) || model;
  setChatConfig({
    role: {
      id: currentRole.id,
      name: currentRole.name,
      icon: currentRole.icon,
      helloMsg: currentRole.helloMsg || currentRole.hello_msg
    },
    model: targetModel,
    modelValue: getModelValue(targetModel) || (currentConfig && currentConfig.modelValue) || modelValue,
    title: t('chat.newChatTitle'),
    chatId: 0,
    userId: loginUser.value ? loginUser.value.id : (currentConfig ? currentConfig.userId : undefined),
  });
  restartSession();
};

// 选择模型：无消息时原地切换，有历史则新建会话并刷新
const selectModel = (item) => {
  if (currentModel.value === item.value) return;

  if (chatData.value.length === 0) {
    switchModelInPlace(item);
    return;
  }

  // 确保 role 存在
  const currentConfig = getChatConfig();
  const currentRole = currentConfig && currentConfig.role ? currentConfig.role : role;

  if (!currentRole || !currentRole.id) {
    showFailToast(t('chat.roleConfigMissing'));
    return;
  }

  currentModel.value = item.value;
  setChatConfig({
    role: {
      id: currentRole.id,
      name: currentRole.name,
      icon: currentRole.icon,
      helloMsg: currentRole.helloMsg || currentRole.hello_msg
    },
    model: item.value,
    modelValue: item.text,
    title: t('chat.newChatTitle'),
    chatId: 0,
    userId: loginUser.value ? loginUser.value.id : (currentConfig ? currentConfig.userId : undefined),
  });
  restartSession();
};

onBeforeRouteLeave(() => {
  closeChatSocket();
});

const connect = function (chat_id, role_id) {
  // 从最新的 chatConfig 读取配置，确保使用最新的值
  const currentConfig = getChatConfig();
  if (!currentConfig || !currentConfig.model || !currentConfig.role || !currentConfig.role.id) {
    showFailToast(t('chat.configMissing'));
    loading.value = false;
    return;
  }

  const currentModel = currentConfig.model;
  const currentRole = currentConfig.role;
  const currentRoleId = currentRole.id;

  let isNewChat = false;
  if (!chat_id) {
    isNewChat = true;
    chat_id = UUID();
  }

  // 更新当前会话 ID，并持久化到本地，保证"新建会话"真正生成一个新的会话
  chatId.value = chat_id;
  setChatConfig({
    role: {
      id: currentRole.id,
      name: currentRole.name,
      icon: currentRole.icon,
      helloMsg: currentRole.helloMsg,
    },
    model: currentModel,
    modelValue: currentConfig.modelValue || '',
    title: currentConfig.title || title.value,
    chatId: chat_id,
    userId: loginUser.value ? loginUser.value.id : (currentConfig ? currentConfig.userId : undefined),
    helloMsg: currentConfig ? currentConfig.helloMsg : undefined,
  });
  url.value = location.protocol + '//' + location.host + '/mobile/chat/export?chat_id=' + chatId.value;

  const _sessionId = getSessionId();
  const host = resolveWsHost();

  const sendHeartbeat = () => {
    if (socket.value !== null) {
      new Promise((resolve) => {
        socket.value.send(JSON.stringify({ type: "heartbeat", content: "ping" }));
        resolve("success");
      }).then(() => {
        heartbeatHandle.value = setTimeout(() => sendHeartbeat(), 5000);
      });
    }
  };

  const _socket = new WebSocket(host + `/api/chat/new?session_id=${_sessionId}&role_id=${currentRoleId}&chat_id=${chat_id}&model_id=${currentModel}&token=${getUserToken()}`);
  _socket.addEventListener('open', () => {
    loading.value = false;
    previousText.value = '';
    canSend.value = true;
    activelyClose.value = false;

    sendHeartbeat();
  });

  _socket.addEventListener('message', event => {
    if (event.data instanceof Blob) {
      const reader = new FileReader();
      reader.readAsText(event.data, "UTF-8");
      reader.onload = () => {
        const data = JSON.parse(String(reader.result));
        if (data.type === 'start') {
          const last = chatData.value[chatData.value.length - 1];
          if (last && last.type === 'reply' && last.thinking) {
            clearThinkingState(last);
          } else {
            chatData.value.push({
              type: "reply",
              id: randString(32),
              icon: currentRole.icon,
              content: ""
            });
          }
        } else if (data.type === 'end') {
          enableInput();
          lineBuffer.value = '';
        } else {
          lineBuffer.value += data.content;
          const reply = chatData.value[chatData.value.length - 1];
          clearThinkingState(reply);
          reply['orgContent'] = lineBuffer.value;
          reply['content'] = md.render(processContent(lineBuffer.value));

          nextTick(() => {
            hl.configure({ ignoreUnescapedHTML: true });
            const lines = document.querySelectorAll('.message-line');
            const blocks = lines[lines.length - 1].querySelectorAll('pre code');
            blocks.forEach((block) => {
              hl.highlightElement(block);
            });
            scrollListBox();

            const items = document.querySelectorAll('.message-line');
            const imgs = items[items.length - 1].querySelectorAll('img');
            for (let i = 0; i < imgs.length; i++) {
              if (!imgs[i].src) continue;
              imgs[i].addEventListener('click', (e) => {
                e.stopPropagation();
                showImagePreview([imgs[i].src]);
              });
            }
          });
        }
      };
    }
  });

  _socket.addEventListener('close', () => {
    if (activelyClose.value || socket.value === null) return;
    canSend.value = true;
    checkSession().then(() => {
      connect(chat_id, role_id);
    }).catch(() => {
      loading.value = true;
      setTimeout(() => connect(chat_id, role_id), 3000);
    });
  });

  socket.value = _socket;
};

const disableInput = (force) => {
  canSend.value = false;
  showReGenerate.value = false;
  showStopGenerate.value = !force;
};

const enableInput = () => {
  canSend.value = true;
  showReGenerate.value = false;
  showStopGenerate.value = false;
};

const scrollListBox = () => {
  const listBox = document.getElementById('message-list-box');
  if (listBox) {
    listBox.scrollTo(0, listBox.scrollHeight + 46);
  }
};

const getReplyIcon = () => {
  const currentConfig = getChatConfig();
  return (currentConfig && currentConfig.role && currentConfig.role.icon)
    || (role && role.icon)
    || '/images/gpt-icon.png';
};

const pushThinkingReply = () => {
  chatData.value.push({
    type: "reply",
    id: randString(32),
    icon: getReplyIcon(),
    content: "",
    orgContent: "",
    thinking: true,
    webSearch: webSearchEnabled.value,
  });
  nextTick(() => scrollListBox());
};

const removeThinkingReply = () => {
  const last = chatData.value[chatData.value.length - 1];
  if (last && last.type === 'reply' && last.thinking) {
    chatData.value.pop();
  }
};

const clearThinkingState = (reply) => {
  if (reply && reply.thinking) {
    reply.thinking = false;
  }
};

const sendMessage = async () => {
  if (canSend.value === false) {
    showToast(t('chat.aiAnswering'));
    return;
  }

  if (prompt.value.trim().length === 0) {
    showToast(t('chat.emptyPrompt'));
    return false;
  }

  // 检查模型是否已选中
  const currentConfig = getChatConfig();
  if (!currentConfig || !currentConfig.model) {
    showFailToast(t('chat.modelRequired'));
    return false;
  }

  // 检查 socket 是否已连接
  if (!socket.value || socket.value.readyState !== WebSocket.OPEN) {
    showToast(t('chat.connectingAI'));
    
    // 确保配置存在
    if (!currentConfig.role || !currentConfig.role.id) {
      showFailToast(t('chat.roleConfigMissing'));
      return false;
    }
    
    // 重新连接 socket
    const finalChatId = chatId.value || currentConfig.chatId || 0;
    const finalRoleId = currentConfig.role.id;
    
    // 等待连接建立
    loading.value = true;
    connect(finalChatId, finalRoleId);
    
    // 等待 socket 连接成功（最多等待3秒）
    let retryCount = 0;
    const maxRetries = 30; // 3秒 = 30 * 100ms
    while (retryCount < maxRetries) {
      await new Promise(resolve => setTimeout(resolve, 100));
      if (socket.value && socket.value.readyState === WebSocket.OPEN) {
        break;
      }
      retryCount++;
    }
    
    if (!socket.value || socket.value.readyState !== WebSocket.OPEN) {
      showFailToast(t('chat.connectAIFail'));
      loading.value = false;
      return false;
    }
    
    loading.value = false;
  }

  chatData.value.push({
    type: "prompt",
    id: randString(32),
    icon: loginUser.value.avatar,
    content: renderInputText(prompt.value),
    created_at: new Date().getTime(),
  });

  nextTick(() => {
    scrollListBox();
  });

  disableInput(false);
  pushThinkingReply();
  try {
    socket.value.send(JSON.stringify({ type: "chat", content: prompt.value, web_search: webSearchEnabled.value }));
    previousText.value = prompt.value;
    prompt.value = '';
    return true;
  } catch (e) {
    console.error(t('chat.sendFailLogPrefix'), e);
    removeThinkingReply();
    showFailToast(t('chat.sendFailToast'));
    enableInput();
    return false;
  }
};

const onComposerAction = () => {
  if (showStopGenerate.value) {
    stopGenerate();
    return;
  }
  sendMessage();
};

const stopGenerate = () => {
  showStopGenerate.value = false;
  httpGet("/api/chat/stop?session_id=" + getSessionId()).then(() => {
    removeThinkingReply();
    enableInput();
  });
};

const reGenerate = () => {
  disableInput(false);
  const text = t('chat.regeneratePrefix') + previousText.value;
  chatData.value.push({
    type: "prompt",
    id: randString(32),
    icon: loginUser.value.avatar,
    content: renderInputText(text)
  });
  nextTick(() => scrollListBox());
  pushThinkingReply();
  socket.value.send(JSON.stringify({ type: "chat", content: previousText.value }));
};

const showShare = ref(false);
const shareOptions = [
  { name: t('chat.shareOptionWechat'), icon: 'wechat' },
  { name: t('chat.shareOptionLink'), icon: 'link' },
];
const shareChat = (option) => {
  showShare.value = false;
  if (option.icon === "wechat") {
    showToast({ message: t('chat.shareWechatTip'), duration: 5000 });
    router.push({
      path: "/mobile/chat/export",
      query: { title: title.value, chat_id: chatId.value, role: role.name, model: modelValue }
    });
  } else if (option.icon === "link") {
    document.getElementById('copy-link-btn').click();
  }
};

const autoResize = () => {
  const textarea = promptInputRef.value || document.querySelector('.composer__textarea');
  if (!textarea) return;
  textarea.style.height = 'auto';
  const next = Math.min(textarea.scrollHeight, 150);
  textarea.style.height = next + 'px';
};

// eslint-disable-next-line no-undef
const recognition = new (window.webkitSpeechRecognition || window.SpeechRecognition)();
recognition.onresult = function (event) {
  prompt.value = event.results[0][0].transcript;
};

recognition.onerror = function (event) {
  showMic.value = false;
  recognition.stop();
  showNotify({ type: 'danger', message: t('chat.asrErrorPrefix') + event.error });
};

recognition.onend = function () {
  console.log(t('chat.asrEndLog'));
};
const inputVoice = () => {
  showMic.value = true;
  recognition.start();
};

const stopVoice = () => {
  showMic.value = false;
  recognition.stop();
};
</script>

<style lang="stylus">
@import "@/assets/css/mobile/chat-session.styl"

/* 整页 flex 布局，保证底部输入框始终在视口内，无需下拉即可看到 */
.chat-layout-root
  height 100vh
  height 100dvh
  min-height 100vh
  overflow hidden
  display flex
  flex-direction column
  background #f4f5f7

.chat-layout
  display flex
  flex-direction column
  flex 1
  min-height 0
  overflow hidden

.chat-nav-wrap
  flex-shrink 0

.chat-list-wrapper
  flex 1
  min-height 0
  overflow hidden
  display flex
  flex-direction column

.message-list-box
  flex 1
  min-height 0
  overflow-y auto
  overflow-x hidden
  -webkit-overflow-scrolling touch

.message-list-box.loading
  min-height 120px

.chat-box-wrapper
  flex-shrink 0
  padding-bottom constant(safe-area-inset-bottom)
  padding-bottom env(safe-area-inset-bottom)


</style>