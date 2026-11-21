import axios from 'axios'
import {getAdminToken, getSessionId, getUserToken} from "@/store/session";

axios.defaults.timeout = 30000

/** 生产环境 HTTPS 页面不能直连 http://IP:5678（混合内容会被浏览器拦截），改走同源 /api/ */
export function resolveApiBaseURL() {
    const envHost = (process.env.VUE_APP_API_HOST || '').trim()
    if (!envHost) return ''
    if (typeof window !== 'undefined'
        && window.location.protocol === 'https:'
        && /^http:\/\//i.test(envHost)) {
        return ''
    }
    return envHost.replace(/\/$/, '')
}

/**
 * HTTPS 页面禁止发起 ws://（浏览器会抛 SecurityError）。
 * 若环境变量是空或 ws://，在 https 下强制改走同源 wss://。
 */
export function resolveWsHost() {
    let host = (process.env.VUE_APP_WS_HOST || '').trim().replace(/\/$/, '')
    if (typeof window === 'undefined') {
        return host
    }
    const isHttps = window.location.protocol === 'https:'
    if (isHttps && (!host || /^ws:\/\//i.test(host))) {
        return 'wss://' + window.location.host
    }
    if (!host) {
        return (isHttps ? 'wss://' : 'ws://') + window.location.host
    }
    return host
}

/** 长耗时接口超时（毫秒），避免短剧生图/合成等被全局 30s 截断 */
export const HTTP_TIMEOUT = {
  default: 30000,
  adminPowerRollback: 600000,  // 算力时间点回滚 / 音乐羊毛回滚 10 分钟
  teachingScript: 180000,      // 口播脚本 3 分钟
  teachingTts: 300000,         // TTS 5 分钟
  teachingImagePrompts: 600000, // 生图提示词 10 分钟
  teachingImage: 300000,       // 单张配图 5 分钟（与后端 image client 300s 对齐）
  teachingRenderVideo: 1800000, // 合成视频 30 分钟
}
axios.defaults.baseURL = resolveApiBaseURL()
axios.defaults.withCredentials = true;
axios.defaults.headers.post['Content-Type'] = 'application/json'

// HTTP拦截器
function applyTeachingTimeout(config) {
    const url = config.url || ''
    if (!url.includes('/api/teaching/')) {
        return config
    }
    if (config.timeout != null && config.timeout !== 30000) {
        return config
    }
    if (url.includes('/images')) {
        config.timeout = HTTP_TIMEOUT.teachingImage
    } else if (url.includes('/record/render-video')) {
        config.timeout = HTTP_TIMEOUT.teachingRenderVideo
    } else if (url.includes('/image-prompts')) {
        config.timeout = HTTP_TIMEOUT.teachingImagePrompts
    } else if (url.includes('/tts')) {
        config.timeout = HTTP_TIMEOUT.teachingTts
    } else if (url.includes('/script')) {
        config.timeout = HTTP_TIMEOUT.teachingScript
    } else {
        config.timeout = HTTP_TIMEOUT.teachingImage
    }
    return config
}

function applyAdminLongRunningTimeout(config) {
    const url = config.url || ''
    const isPowerRollback =
        url.includes('/api/admin/user/previewPowerPointRollback') ||
        url.includes('/api/admin/user/executePowerPointRollback') ||
        url.includes('/api/admin/user/previewMusicRefundRollback') ||
        url.includes('/api/admin/user/rollbackMusicRefundExploit')
    if (!isPowerRollback) {
        return config
    }
    if (config.timeout == null || config.timeout === 30000) {
        config.timeout = HTTP_TIMEOUT.adminPowerRollback
    }
    return config
}

axios.interceptors.request.use(
    config => {
        applyTeachingTimeout(config)
        applyAdminLongRunningTimeout(config)
        // set token
        config.headers['Chat-Token'] = getSessionId();
        config.headers['Authorization'] = getUserToken();
        config.headers['Admin-Authorization'] = getAdminToken();
        return config
    }, error => {
        return Promise.reject(error)
    })
axios.interceptors.response.use(
    response => {
        let data = response.data;
        if (data.code === 0) {
            return response
        } else {
            return Promise.reject(response.data)
        }
    }, error => {
        return Promise.reject(error)
    })


// send a http get request
export function httpGet(url, params = {}, options = {}) {
    return new Promise((resolve, reject) => {
        axios.get(url, {
            params: params,
            ...options,
        }).then(response => {
            resolve(response.data)
        }).catch(err => {
            reject(err)
        })
    })
}


// send a http post request
export function httpPost(url, data = {}, options = {}) {
    return new Promise((resolve, reject) => {
        axios.post(url, data, options).then(response => {
            resolve(response.data)
        }).catch(err => {
            reject(err)
        })
    })
}
