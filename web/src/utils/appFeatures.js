export const APP_KEYS = ['image', 'music', 'video', 'ppt', 'logo', 'drama']

const FLAG_MAP = {
  image: 'enabled_app_image',
  music: 'enabled_app_music',
  video: 'enabled_app_video',
  ppt: 'enabled_app_ppt',
  logo: 'enabled_app_logo',
  drama: 'enabled_app_drama',
}

/** 与后端 app_server 启动默认值一致 */
const DEFAULTS = {
  image: true,
  music: true,
  video: false,
  ppt: true,
  logo: true,
  drama: true,
}

export function parseAppFlags(system) {
  const flags = {}
  for (const key of APP_KEYS) {
    const field = FLAG_MAP[key]
    const val = system?.[field]
    flags[key] = val === undefined || val === null ? DEFAULTS[key] : !!val
  }
  return flags
}

export function isAppEnabled(system, key) {
  return parseAppFlags(system)[key]
}
