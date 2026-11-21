import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN'
import enUS from './locales/en-US'
import jaJP from './locales/ja-JP'
import krKR from './locales/kr-KR'
import { httpGet } from '@/utils/http'

function normalizeLocale(locale) {
  if (!locale) return null
  if (locale === 'ja-JP') return 'ja'
  if (locale === 'kr') return 'kr-KR'
  return locale
}

const rawStored = localStorage.getItem('locale')
const hasStoredLocale = rawStored !== null && rawStored !== ''
let storedLocale = normalizeLocale(rawStored) || 'en-US'

export const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: storedLocale,
  fallbackLocale: 'en-US',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
    ja: jaJP,
    // 兼容：如果内部归一化为 'ja-JP' 也能取到
    'ja-JP': jaJP,
    // 韩语
    'kr-KR': krKR,
  },
})

export const setLocale = (locale) => {
  const next = normalizeLocale(locale) || 'en-US'
  i18n.global.locale.value = next
  localStorage.setItem('locale', next)
}

/**
 * 首次访问（无本地语言偏好）时，根据 IP 所属国家自动选择语言：
 * 中国→zh-CN，日本→ja，韩国→kr-KR，其他→en-US。
 * 用户手动切换过语言后不再覆盖。
 */
export async function initLocaleByIP() {
  if (hasStoredLocale) {
    return i18n.global.locale.value
  }
  try {
    const res = await Promise.race([
      httpGet('/api/config/locale', {}, { timeout: 3000 }),
      new Promise((_, reject) => setTimeout(() => reject(new Error('locale timeout')), 3000)),
    ])
    const locale = normalizeLocale(res?.data?.locale)
    if (locale) {
      setLocale(locale)
      return locale
    }
  } catch (e) {
    // 接口失败时保持内存中的默认英文，不写入 localStorage，便于下次重试
  }
  return i18n.global.locale.value
}
