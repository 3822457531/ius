/** MJ 任务列表：解析接口 data、生成缩略图 URL（移动端 / 桌面共用）。 */

import { resolveApiBaseURL } from '@/utils/http'

export const LOGO_PROMPT_MARKER = '[AI-LOGO]'

export function isLogoMjJob(job) {
  return typeof job?.prompt === 'string' && job.prompt.includes(LOGO_PROMPT_MARKER)
}

export function isImageMjJob(job) {
  return !isLogoMjJob(job)
}

export function getMjApiBase() {
  return resolveApiBaseURL()
}



const ARRAY_KEYS = ['items', 'list', 'records', 'rows', 'jobs', 'data']



function pickArray(obj) {

  if (!obj || typeof obj !== 'object') return null

  for (const k of ARRAY_KEYS) {

    if (Array.isArray(obj[k])) return obj[k]

  }

  return null

}



/** 解析 { code, data }：data 为数组或分页对象；兼容 res 本身为数组、data 为 JSON 字符串 */

export function unwrapMjJobList(res) {

  if (Array.isArray(res)) return res

  if (!res || typeof res !== 'object') return []



  const raw = res.data

  if (raw === undefined || raw === null) {

    const top = pickArray(res)

    return top || []

  }

  if (Array.isArray(raw)) return raw

  if (typeof raw === 'string') {

    try {

      const parsed = JSON.parse(raw)

      if (Array.isArray(parsed)) return parsed

      if (parsed && typeof parsed === 'object') {

        const nested = pickArray(parsed)

        return nested || []

      }

    } catch {

      return []

    }

    return []

  }

  if (typeof raw === 'object') {

    const inner = pickArray(raw)

    if (inner) return inner

  }

  return []

}



/** 写入 thumb_url：代理/oss、绝对 URL 七牛参数、相对路径拼 API 根、仅 org_url */

export function applyThumbUrl(job) {

  if (!job) return

  const apiBase = getMjApiBase()

  const img = job.img_url

  const org = job.org_url

  if (img && job.use_proxy) {

    job.thumb_url = img + '?x-oss-process=image/quality,q_60&format=webp'

    return

  }

  if (img && /^https?:\/\//i.test(img)) {

    if (job.type === 'upscale' || job.type === 'swapFace') {

      job.thumb_url = img + '?imageView2/1/w/480/h/600/q/75'

    } else {

      job.thumb_url = img + '?imageView2/1/w/480/h/480/q/75'

    }

    return

  }

  if (img && img.startsWith('/')) {

    job.thumb_url = apiBase ? apiBase + img : img

    return

  }

  if (org && /^https?:\/\//i.test(org)) {

    job.thumb_url = org

    return

  }

  if (org && org.startsWith('/')) {

    job.thumb_url = apiBase ? apiBase + org : org

  }

}



/** 预览用绝对地址 */

export function resolveMjImageUrl(imgUrl, orgUrl) {

  const apiBase = getMjApiBase()

  let u = imgUrl || orgUrl || ''

  if (typeof u === 'string' && u.startsWith('/') && apiBase) u = apiBase + u

  return u

}


