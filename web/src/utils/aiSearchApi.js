import { getUserToken } from '@/store/session'
import { resolveApiBaseURL } from '@/utils/http'

export const AI_SEARCH_API = '/api/ai-search'

function authHeaders(extra = {}) {
  const token = getUserToken()
  const headers = { ...extra }
  if (token) headers.Authorization = token
  return headers
}

function apiUrl(path) {
  const base = resolveApiBaseURL()
  const p = path.startsWith('/') ? path : `/${path}`
  return `${base}${AI_SEARCH_API}${p}`
}

/**
 * DeepSeek 网页搜索
 * @returns {{ res: Response, data: any }}
 */
export async function aiSearchFetch(path, options = {}) {
  const { method = 'GET', body, signal, timeoutMs = 45000 } = options
  const controller = new AbortController()
  const onAbort = () => controller.abort()
  if (signal) {
    if (signal.aborted) controller.abort()
    else signal.addEventListener('abort', onAbort, { once: true })
  }
  const timer = setTimeout(() => controller.abort(), timeoutMs)

  try {
    const res = await fetch(apiUrl(path), {
      method,
      headers: authHeaders(body != null ? { 'Content-Type': 'application/json' } : {}),
      body: body != null ? JSON.stringify(body) : undefined,
      signal: controller.signal,
    })
    const text = await res.text()
    let data = null
    try {
      data = text ? JSON.parse(text) : null
    } catch {
      data = { code: 1, message: text || 'invalid_json' }
    }
    return { res, data }
  } finally {
    clearTimeout(timer)
    if (signal) signal.removeEventListener('abort', onAbort)
  }
}

export function historyStorageKey(userId) {
  return `AI_SEARCH_HISTORY_${userId || 'guest'}`
}

export function loadLocalHistory(userId) {
  try {
    const raw = localStorage.getItem(historyStorageKey(userId))
    const list = raw ? JSON.parse(raw) : []
    return Array.isArray(list) ? list.filter((x) => x && x.keyword) : []
  } catch {
    return []
  }
}

export function saveLocalHistory(userId, keyword) {
  const q = String(keyword || '').trim()
  if (!q) return loadLocalHistory(userId)
  const prev = loadLocalHistory(userId).filter((x) => x.keyword !== q)
  const next = [{ keyword: q, searchedAt: new Date().toISOString() }, ...prev].slice(0, 30)
  localStorage.setItem(historyStorageKey(userId), JSON.stringify(next))
  return next
}

export function clearLocalHistory(userId) {
  localStorage.removeItem(historyStorageKey(userId))
  return []
}
