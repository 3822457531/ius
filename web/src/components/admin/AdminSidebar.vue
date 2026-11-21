<template>
  <div class="sidebar">
    <div class="logo">
      <el-image :src="logo"/>
      <span class="text" v-show="!sidebar.collapse">{{ title }} - {{ version }}</span>
    </div>

    <el-menu
        class="sidebar-el-menu"
        :default-active="onRoutes"
        :collapse="sidebar.collapse"
        background-color="#324157"
        text-color="#bfcbd9"
        active-text-color="#20a0ff"
        unique-opened
        router
    >
      <template v-for="item in items">
        <template v-if="item.subs">
          <el-sub-menu :index="item.index" :key="item.index">
            <template #title>
              <i :class="'iconfont icon-'+item.icon"></i>
              <span>{{ item.title }}</span>
            </template>
            <template v-for="subItem in item.subs">
              <el-sub-menu
                  v-if="subItem.subs"
                  :index="subItem.index"
                  :key="subItem.index"
              >
                <template #title>{{ subItem.title }}</template>
                <el-menu-item v-for="(threeItem, i) in subItem.subs" :key="i" :index="threeItem.index">
                  {{ threeItem.title }}
                </el-menu-item>
              </el-sub-menu>
              <el-menu-item v-else :index="subItem.index">
                <i v-if="subItem.icon" :class="'iconfont icon-'+subItem.icon"></i>
                {{ subItem.title }}
              </el-menu-item>
            </template>
          </el-sub-menu>
        </template>
        <template v-else>
          <el-menu-item :index="item.index" :key="item.index">
            <i :class="'iconfont icon-'+item.icon"></i>
            <template #title>
              <span class="menu-title-row">
                {{ item.title }}
                <em v-if="item.index === '/admin/cs' && csUnread > 0" class="cs-side-badge">
                  {{ csUnread > 99 ? '99+' : csUnread }}
                </em>
              </span>
            </template>
          </el-menu-item>
        </template>
      </template>
    </el-menu>
  </div>
</template>

<script setup>
import {computed, onMounted, onUnmounted, ref} from 'vue';
import {setMenuItems, useSidebarStore} from '@/store/sidebar';
import {useRoute} from 'vue-router';
import {httpGet} from "@/utils/http";
import {ElMessage} from "element-plus";
import {playCsNotifySound} from "@/utils/csNotify";

const title = ref('Chat-Plus-Admin')
const logo = ref('/images/logo.png')
const version = ref(process.env.VUE_APP_VERSION)
const csUnread = ref(0)
let csUnreadPrev = 0
let csUnreadTimer = null

const fetchCsUnread = async (playSound = false) => {
  try {
    const res = await httpGet('/api/admin/cs/unread')
    const n = Number(res.data?.unread || 0)
    if (playSound && n > csUnreadPrev) {
      playCsNotifySound()
    }
    csUnreadPrev = n
    csUnread.value = n
  } catch {
    // ignore
  }
}

onMounted(() => {
  fetchCsUnread(false)
  csUnreadTimer = setInterval(() => fetchCsUnread(true), 12000)
})

onUnmounted(() => {
  if (csUnreadTimer) {
    clearInterval(csUnreadTimer)
    csUnreadTimer = null
  }
})

// 加载系统配置
httpGet('/api/admin/config/get?key=system').then(res => {
  title.value = res.data['admin_title'];
}).catch(e => {
  ElMessage.error("加载系统配置失败: " + e.message)
})
const items = [
  {
    icon: 'home',
    index: '/admin/dashboard',
    title: '仪表盘',
  },

  {
    icon: 'user-fill',
    index: '/admin/user',
    title: '用户管理',
  },

  {
    icon: 'role',
    index: '/admin/role',
    title: '角色管理',
  },
  {
    icon: 'api-key',
    index: '/admin/apikey',
    title: 'API-KEY',
  },
  {
    icon: 'model',
    index: '/admin/chat/model',
    title: '语言模型',
  },
  {
    icon: 'recharge',
    index: '/admin/mint',
    title: '矿机管理',
  },
  {
    icon: 'order',
    index: '/admin/mintOrder',
    title: '用户算力机',
  },
  // {
  //   icon: 'recharge',
  //   index: '/admin/product',
  //   title: '充值产品',
  // },

  
  {
    icon: 'order',
    index: '/admin/order',
    title: '充值订单',
  },

  {
    icon: 'reward',
    index: '/admin/tradeBuy',
    title: 'c2c交易订单',
  },
  {
    icon: 'reward',
    index: '/admin/powerTradeOrder',
    title: '平台充值订单',
  },
  {
    icon: 'reward',
    index: '/admin/powerBuybackOrder',
    title: '算力回购订单',
  },

  // {
  //   icon: 'control',
  //   index: '/admin/q2Log',
  //   title: 'SOHA币消费日志',
  // },

  // {
  //   icon: 'control',
  //   index: '/admin/tradeSell',
  //   title: '交易卖单',
  // },
  // {
  //   icon: 'reward',
  //   index: '/admin/reward',
  //   title: '众筹管理',
  // },
  // {
  //   icon: 'control',
  //   index: '/admin/functions',
  //   title: '函数管理',
  // },
  {
    icon: 'prompt',
    index: '/admin/chats',
    title: '对话管理',
  },
  {
    icon: 'prompt',
    index: '/admin/cs',
    title: '在线客服',
  },
  {
    icon: 'role',
    index: '/admin/manger',
    title: '管理员',
  },
  {
    icon: 'config',
    index: '/admin/system',
    title: '系统设置',
  },
  {
    icon: 'control',
    index: '/admin/apps',
    title: '应用管理',
  },
  {
    icon: 'prompt',
    index: '/admin/music',
    title: '音乐管理',
  },
  {
    icon: 'log',
    index: '/admin/powerLog',
    title: '用户算力日志',
  },
  {
    icon: 'log',
    index: '/admin/coinLog',
    title: 'SOHA币产出日志',
  },
  {
    icon: 'log',
    index: '/admin/loginLog',
    title: '用户登录日志',
  },
  // {
  //   icon: 'menu',
  //   index: '1',
  //   title: '常用模板页面',
  //   subs: [
  //     {
  //       index: '/admin/demo/form',
  //       title: '表单页面',
  //     },
  //     {
  //       index: '/admin/demo/table',
  //       title: '常用表格',
  //     },
  //     {
  //       index: '/admin/demo/import',
  //       title: '导入Excel',
  //     },
  //     {
  //       index: '/admin/demo/editor',
  //       title: '富文本编辑器',
  //     },
  //   ],
  // },
];

const route = useRoute();
const onRoutes = computed(() => {
  return route.path;
});

const sidebar = useSidebarStore();
setMenuItems(items)
</script>

<style scoped lang="stylus">
.sidebar {
  display: block;
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  overflow-y: scroll;

  .logo {
    display flex
    width 219px
    background-color #324157
    padding 6px 15px;

    .el-image {
      width 30px;
      height 30px;
      padding-top 8px;
      border-radius 100%

      .el-image__inner {
        height 40px
      }
    }

    .text {
      color #ffffff
      font-weight bold
      padding 12px 0 12px 10px;
      transition: width 2s ease;
    }
  }

  ul {
    height: 100%;

    .el-menu-item, .el-sub-menu {
      .iconfont {
        font-size 16px;
        margin-right 5px;
      }
    }

    .el-menu-item.is-active {
      background-color rgb(40, 52, 70)
    }
  }

  .sidebar-el-menu:not(.el-menu--collapse) {
    width: 250px;
  }
}

.sidebar::-webkit-scrollbar {
  width: 0;
}

.menu-title-row
  display inline-flex
  align-items center
  gap 8px

.cs-side-badge
  display inline-flex
  align-items center
  justify-content center
  min-width 18px
  height 18px
  padding 0 5px
  border-radius 999px
  background #f04438
  color #fff
  font-size 11px
  font-style normal
  font-weight 700
  line-height 18px
</style>
