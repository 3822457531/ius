import {createRouter, createWebHistory} from "vue-router";
import { httpGet } from "@/utils/http";

/** 已指定客服的用户访问咨询页时，强制进入接线台 */
async function redirectCsAgentToDesk(to, from, next) {
  try {
    const res = await httpGet('/api/user/profile')
    const flag = res?.data?.is_cs
    if (flag === true || flag === 1 || flag === '1') {
      next({ path: '/mobile/cs-desk', replace: true })
      return
    }
  } catch {
    // 未登录或接口失败时仍进入普通咨询页
  }
  next()
}

const routes = [
    {
        name: 'home',
        path: '/',
        redirect: '/chat',
        meta: {title: '首页'},
        component: () => import('@/views/Home.vue'),
        children: [
            {
                name: 'chat',
                path: '/chat',
                meta: {title: '创作中心'},
                component: () => import('@/views/ChatPlus.vue'),
            },
            {
                name: 'image-mj',
                path: '/mj',
                meta: {title: 'MidJourney 绘画中心'},
                component: () => import('@/views/ImageMj.vue'),
            },
            {
                name: 'image-sd',
                path: '/sd',
                meta: {title: 'stable diffusion 绘画中心'},
                component: () => import('@/views/ImageSd.vue'),
            },
            {
                name: 'member',
                path: '/member',
                meta: {title: '会员充值中心'},
                component: () => import('@/views/Member.vue'),
            },
            {
                name: 'chat-role',
                path: '/apps',
                meta: {title: '应用中心'},
                component: () => import('@/views/ChatApps.vue'),
            },
            {
                name: 'images',
                path: '/images-wall',
                meta: {title: '作品展示'},
                component: () => import('@/views/ImagesWall.vue'),
            },
            {
                name: 'user-invitation',
                path: '/invite',
                meta: {title: '推广计划'},
                component: () => import('@/views/Invitation.vue'),
            },
            {
                name: 'powerLog',
                path: '/powerLog',
                meta: {title: '算力明细'},
                component: () => import('@/views/PowerLog.vue'),
            }
        ]
    },
    {
        name: 'chat-export',
        path: '/chat/export',
        meta: {title: '导出会话记录'},
        component: () => import('@/views/ChatExport.vue'),
    },
    {
        name: 'login',
        path: '/login',
        meta: {title: '用户登录'},
        component: () => import('@/views/Login.vue'),
    },
    {
        name: 'register',
        path: '/register',

        meta: {title: '用户注册'},
        component: () => import('@/views/Register.vue'),
    },
    {
        path: '/admin/login',
        name: 'admin-login',
        meta: {title: 'ChatPuls 控制台登录'},
        component: () => import('@/views/admin/Login.vue'),
    },
    {
        name: 'admin',
        path: '/admin',
        redirect: '/admin/dashboard',
        component: () => import("@/views/admin/Home.vue"),
        meta: {title: 'ChatPuls 管理后台'},
        children: [
            {
                path: '/admin/dashboard',
                name: 'admin-dashboard',
                meta: {title: '仪表盘'},
                component: () => import('@/views/admin/Dashboard.vue'),
            },
            {
                path: '/admin/system',
                name: 'admin-system',
                meta: {title: '系统设置'},
                component: () => import('@/views/admin/SysConfig.vue'),
            },
            {
                path: '/admin/apps',
                name: 'admin-apps',
                meta: {title: '应用管理'},
                component: () => import('@/views/admin/AppManage.vue'),
            },
            {
                path: '/admin/music',
                name: 'admin-music',
                meta: {title: '音乐管理'},
                component: () => import('@/views/admin/MusicManage.vue'),
            },
            {
                path: '/admin/user',
                name: 'admin-user',
                meta: {title: '用户管理'},
                component: () => import('@/views/admin/Users.vue'),
            },
            {
                path: '/admin/role',
                name: 'admin-role',
                meta: {title: '角色管理'},
                component: () => import('@/views/admin/Roles.vue'),
            },
            {
                path: '/admin/apikey',
                name: 'admin-apikey',
                meta: {title: 'API-KEY 管理'},
                component: () => import('@/views/admin/ApiKey.vue'),
            },
            {
                path: '/admin/chat/model',
                name: 'admin-chat-model',
                meta: {title: '语言模型'},
                component: () => import('@/views/admin/ChatModel.vue'),
            },
            {
                path: '/admin/product',
                name: 'admin-product',
                meta: {title: '充值产品'},
                component: () => import('@/views/admin/Product.vue'),
            },
            {
                path: '/admin/tradeBuy',
                name: 'admin-tradeBuy',
                meta: {title: '交易买单'},
                component: () => import('@/views/admin/TradeBuy.vue'),
            },
            {
                path: '/admin/tradeSell',
                name: 'admin-tradeSell',
                meta: {title: '交易卖单'},
                component: () => import('@/views/admin/TradeSell.vue'),
            },
            {
                path: '/admin/powerTradeOrder',
                name: 'admin-powerTradeOrder',
                meta: {title: '算力充值订单'},
                component: () => import('@/views/admin/PowerTradeOrder.vue'),
            },
            {
                path: '/admin/powerBuybackOrder',
                name: 'admin-powerBuybackOrder',
                meta: {title: '算力回购订单'},
                component: () => import('@/views/admin/PowerBuybackOrder.vue'),
            },

            {
                path: '/admin/mint',
                name: 'admin-mint',
                meta: {title: '矿机管理'},
                component: () => import('@/views/admin/Mint.vue'),
            },
            {
                path: '/admin/mintOrder',
                name: 'admin-mintOrder',
                meta: {title: '用户算力机'},
                component: () => import('@/views/admin/MintOrder.vue'),
            },
            {
                path: '/admin/order',
                name: 'admin-order',
                meta: {title: '充值订单'},
                component: () => import('@/views/admin/Order.vue'),
            },
            {
                path: '/admin/reward',
                name: 'admin-reward',
                meta: {title: '众筹管理'},
                component: () => import('@/views/admin/Reward.vue'),
            },
            {
                path: '/admin/loginLog',
                name: 'admin-loginLog',
                meta: {title: '登录日志'},
                component: () => import('@/views/admin/LoginLog.vue'),
            },
            {
                path: '/admin/functions',
                name: 'admin-functions',
                meta: {title: '函数管理'},
                component: () => import('@/views/admin/Functions.vue'),
            },
            {
                path: '/admin/chats',
                name: 'admin-chats',
                meta: {title: '对话管理'},
                component: () => import('@/views/admin/ChatList.vue'),
            },
            {
                path: '/admin/cs',
                name: 'admin-cs',
                meta: {title: '在线客服'},
                component: () => import('@/views/admin/CustomerService.vue'),
            },
            {
                path: '/admin/powerLog',
                name: 'admin-power-log',
                meta: {title: '算力日志'},
                component: () => import('@/views/admin/PowerLog.vue'),
            },
            {
                path: '/admin/coinLog',
                name: 'admin-coin-log',
                meta: {title: 'SOHA币产出日志'},
                component: () => import('@/views/admin/CoinLog.vue'),
            },
            {
                path: '/admin/q2Log',
                name: 'admin-q2-log',
                meta: {title: 'SOHA日志'},
                component: () => import('@/views/admin/Q2Log.vue'),
            },
            {
                path: '/admin/manger',
                name: 'admin-manger',
                meta: {title: '管理员'},
                component: () => import('@/views/admin/Manager.vue'),
            }
        ]
    },


    {
        name: 'mobile',
        path: '/mobile',
        meta: {title: ' v1.1.0'},
        component: () => import('@/views/mobile/Home.vue'),
        // 默认进入会话页，避免再落到列表页
        redirect: '/mobile/chat/session',
        children: [
            {
                path: '/mobile/chat',
                name: 'mobile-chat',
                component: () => import('@/views/mobile/ChatList.vue'),
            },
            {
                path: '/mobile/image',
                component: () => import('@/views/mobile/Image.vue'),
                children: [
                    {
                        path: '',
                        name: 'mobile-image-hub',
                        component: () => import('@/views/mobile/ImageHub.vue'),
                    },
                    {
                        // AI 图片：已对接 gpt-image-2（行业模版 + 隐藏模型选择）
                        path: 'mj',
                        name: 'mobile-image-mj',
                        meta: { hideTabbar: true },
                        component: () => import('@/views/mobile/ImageMjGpt.vue'),
                    },
                    {
                        // 兼容旧链接 /mobile/image/image2
                        path: 'image2',
                        name: 'mobile-image-mj-gpt',
                        meta: { hideTabbar: true },
                        component: () => import('@/views/mobile/ImageMjGpt.vue'),
                    },
                    {
                        path: 'music',
                        name: 'mobile-image-music',
                        meta: { hideTabbar: true },
                        component: () => import('@/views/mobile/MusicGenerate.vue'),
                    },
                    {
                        path: 'video',
                        name: 'mobile-image-video',
                        meta: { hideTabbar: true },
                        component: () => import('@/views/mobile/Sora.vue'),
                    },
                    {
                        path: 'drama',
                        name: 'mobile-image-drama',
                        meta: { hideTabbar: true, title: '文字故事' },
                        component: () => import('@/views/mobile/DramaStory.vue'),
                    },
                    {
                        path: 'logo',
                        name: 'mobile-image-logo',
                        meta: { hideTabbar: true },
                        component: () => import('@/views/mobile/ImageLogo.vue'),
                    },
                    {
                        path: 'ai-search',
                        name: 'mobile-ai-search',
                        meta: { hideTabbar: true, title: 'AI搜' },
                        component: () => import('@/views/mobile/AiSearch.vue'),
                    },
                ],
            },
            {
                // 兼容旧链接：搜索已迁入「应用」
                path: '/mobile/ai-search',
                redirect: '/mobile/image/ai-search',
            },
            {
                path: '/mobile/alliance',
                name: 'mobile-alliance',
                meta: { title: '联盟' },
                component: () => import('@/views/mobile/Alliance.vue'),
            },
            {
                path: '/mobile/profile',
                name: 'mobile-profile',
                component: () => import('@/views/mobile/Profile.vue'),
            },
            {
                path: '/mobile/img-wall',
                name: 'mobile-img-wall',
                component: () => import('@/views/mobile/ImgWall.vue'),
            },
            {
                path: '/mobile/invitation',
                name: 'mobile-invitation',
                component: () => import('@/views/mobile/Invitation.vue'),
            },
            {
                path: '/mobile/placement',
                name: 'mobile-placement',
                component: () => import('@/views/mobile/Placement.vue'),
            },
            {
                path: '/mobile/invest',
                name: 'mobile-invest',
                component: () => import('@/views/mobile/Invest.vue'),
            },
            {
                path: '/mobile/cp',
                name: 'mobile-cp',
                component: () => import('@/views/mobile/Cp.vue'),
            },
            {
                path: '/mobile/ing',
                name: 'mobile-ing',
                component: () => import('@/views/mobile/Ing.vue'),
            },
            {
                path: '/mobile/end',
                name: 'mobile-end',
                component: () => import('@/views/mobile/End.vue'),
            },
            {
                path: '/mobile/cent',
                name: 'mobile-cent',
                component: () => import('@/views/mobile/Cent.vue'),
            },
            {
                path: '/mobile/usdt/:id',
                name: 'mobile-usdt',
                component: () => import('@/views/mobile/Usdt.vue'),
            },
            {
                path: '/mobile/powerlog',
                name: 'mobile-powerlog',
                component: () => import('@/views/mobile/PowerLog.vue'),
            },
            {
                path: '/mobile/coinlog',
                name: 'mobile-coinlog',
                component: () => import('@/views/mobile/CoinLog.vue'),
            },

            {
                path: '/mobile/q2log',
                name: 'mobile-q2log',
                redirect: '/mobile/coinlog',
            },

            {
                path: '/mobile/syitems',
                name: 'mobile-syitems',
                component: () => import('@/views/mobile/Syitems.vue'),
            },
            {
                path: '/mobile/coin',
                name: 'mobile-coin',
                component: () => import('@/views/mobile/CoinItems.vue'),
            },
            {
                path: '/mobile/digital',
                name: 'mobile-digital',
                component: () => import('@/views/mobile/Digital.vue'),
            },
            {
                path: '/mobile/trade',
                name: 'mobile-trade',
                component: () => import('@/views/mobile/Trade.vue'),
            },
            {
                path: '/mobile/contact',
                name: 'mobile-contact',
                meta: { hideTabbar: true, title: '在线客服' },
                beforeEnter: redirectCsAgentToDesk,
                component: () => import('@/views/mobile/Contact.vue'),
            },
            {
                path: '/mobile/cs-desk',
                name: 'mobile-cs-desk',
                meta: { hideTabbar: true, title: '客服工作台' },
                component: () => import('@/views/mobile/CsDesk.vue'),
            },
            {
                path: '/mobile/powerpool',
                name: 'mobile-powerpool',
                component: () => import('@/views/mobile/PowerPool.vue'),
            },
            {
                path: '/mobile/c2c',
                name: 'mobile-c2c',
                component: () => import('@/views/mobile/c2c.vue'),
            },
            {
                path: '/mobile/powertrade',
                name: 'mobile-powertrade',
                component: () => import('@/views/mobile/PowerTrade.vue'),
            },
            {
                path: '/mobile/audit',
                name: 'mobile-audit',
                component: () => import('@/views/mobile/Audit.vue'),
            },
            {
                // 收进 Home 壳，切换 Tab 时不卸载底部框架，接近原生 App
                path: '/mobile/chat/session',
                name: 'mobile-chat-session',
                meta: { hideTabbar: true },
                component: () => import('@/views/mobile/ChatSession.vue'),
            },
            
        ]
    },
    {
        path: '/mobile/chat/export',
        name: 'mobile-chat-export',
        component: () => import('@/views/mobile/ChatExport.vue'),
    },

    {
        name: 'test',
        path: '/test',
        meta: {title: '测试页面'},
        component: () => import('@/views/Test.vue'),
    },
    {
        name: 'NotFound',
        path: '/:all(.*)',
        meta: {title: '页面没有找到'},
        component: () => import('@/views/404.vue'),
    },
]

// console.log(MY_VARIABLE)
const router = createRouter({
    history: createWebHistory(),
    routes: routes,
})

let prevRoute = null
// dynamic change the title when router change
router.beforeEach((to, from, next) => {
    if (to.meta.title) {
        document.title = `${to.meta.title} | ${process.env.VUE_APP_TITLE}`
    }
    prevRoute = from
    next()
})

export {router, prevRoute};