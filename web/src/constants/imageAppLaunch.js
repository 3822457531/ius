/** Image 父级 provide：算力中心进入子应用时的全屏加载过渡 */
export const IMAGE_APP_LAUNCH_KEY = Symbol('imageAppLaunch')
/** 与上图同款加载层，在打开内嵌 iframe（如 PPT）前展示 */
export const IMAGE_EMBED_LAUNCH_KEY = Symbol('imageEmbedLaunch')
/** 图片/音乐返回算力中心：白屏盖住 → 路由回 hub → 白屏下滑收起 */
export const IMAGE_RETURN_HUB_KEY = Symbol('imageReturnHub')
