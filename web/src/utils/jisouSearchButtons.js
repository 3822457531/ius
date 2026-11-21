/** 极搜筛选按钮与结果行展示（移植自 tg/lib/jisou-search-buttons） */

export const SUPPORTED_JISOU_FILTER_TYPES = new Set(['Video', 'Photo', 'GText'])
export const DEFAULT_JISOU_RESOURCE_FILTER_TYPE = 'Video'

const FILTER_TYPE_ICONS = {
  channel: '📢',
  Video: '🎬',
  Photo: '🖼️',
  Audio: '🎧',
  GText: '💬',
}

const JISOU_TG_TO_DISPLAY_ICON = {
  '🏞': '🖼️',
}

const FILTER_TYPE_LABELS = {
  channel: '频道',
  Video: '视频',
  Photo: '图片',
  Audio: '音频',
  GText: '文字',
}

export const JISOU_RESULT_ICONS = ['📄', '🎬', '🏞', '🎧', '💬', '📢']

export function jisouDisplayResultIcon(icon) {
  if (!icon) return null
  return JISOU_TG_TO_DISPLAY_ICON[icon] ?? icon
}

export function parseJisouCallbackFilterType(callback) {
  if (!callback) return null
  const match = String(callback).trim().match(/^\/s\s+\S+\s+\S+\s+(\S+)/)
  return match?.[1] ?? null
}

export function isSupportedJisouFilterCallback(callback) {
  const type = parseJisouCallbackFilterType(callback)
  return type != null && SUPPORTED_JISOU_FILTER_TYPES.has(type)
}

export function jisouFilterIconForType(type) {
  if (!type) return null
  return FILTER_TYPE_ICONS[type] ?? null
}

export function jisouFilterButtonLabel(btn) {
  const type = parseJisouCallbackFilterType(btn?.callback)
  if (type && FILTER_TYPE_ICONS[type]) return FILTER_TYPE_ICONS[type]
  return btn?.text || ''
}

export function jisouFilterButtonTitle(btn) {
  const type = parseJisouCallbackFilterType(btn?.callback)
  if (type && FILTER_TYPE_LABELS[type]) return FILTER_TYPE_LABELS[type]
  return btn?.text || ''
}

export function pickSupportedJisouFilterButtons(filters) {
  return (filters || []).filter((btn) => isSupportedJisouFilterCallback(btn.callback))
}

export function findJisouFilterButtonByType(filters, type) {
  return (
    pickSupportedJisouFilterButtons(filters).find(
      (btn) => parseJisouCallbackFilterType(btn.callback) === type
    ) ?? null
  )
}

export function parseJisouResultLeadingIcon(text) {
  let trimmed = String(text ?? '').trimStart()
  let icon = null
  let changed = true
  while (changed) {
    changed = false
    for (const candidate of JISOU_RESULT_ICONS) {
      if (trimmed.startsWith(candidate)) {
        icon = candidate
        trimmed = trimmed.slice(candidate.length).trimStart()
        changed = true
        break
      }
    }
  }
  return {
    icon,
    rest: trimmed.replace(/\s+\d+(?:\.\d+)?[kK万]?\s*$/, '').trim(),
  }
}

export function stripJisouResultLeadingIcons(text) {
  return parseJisouResultLeadingIcon(text).rest
}

export function formatJisouChannelRow(channel, activeFilterType) {
  const rawTitle = String(channel?.title ?? channel?.label ?? '').trim()
  const fromTitle = parseJisouResultLeadingIcon(rawTitle)
  const rawIcon =
    fromTitle.icon || channel?.contentIcon?.trim() || jisouFilterIconForType(activeFilterType) || null
  const title = stripJisouResultLeadingIcons(rawTitle)
  return { icon: jisouDisplayResultIcon(rawIcon), title: title || rawTitle }
}

export function normalizeJisouSearchButtons(raw) {
  const source = raw && typeof raw === 'object' ? raw : {}
  const pick = (key) => {
    const list = source[key]
    if (!Array.isArray(list)) return []
    return list
      .map((item) => {
        if (!item || typeof item !== 'object') return null
        const text = String(item.text ?? '').trim()
        if (!text) return null
        return {
          text,
          className: item.className != null ? String(item.className) : undefined,
          url: item.url != null ? String(item.url) : null,
          callback: item.callback != null ? String(item.callback) : null,
        }
      })
      .filter(Boolean)
  }
  return { filters: pick('filters'), actions: pick('actions') }
}

export function isJisouPromotedChannel(channel) {
  return !channel?.username
}
