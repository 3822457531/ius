import { computed, onMounted, ref, watch } from 'vue'
import { showFailToast, showSuccessToast, showToast } from 'vant'
import { httpGet, httpPost, HTTP_TIMEOUT } from '@/utils/http'

function defaultNotify() {
  return {
    success: (text) => showSuccessToast(text),
    warning: (text) => showToast(text),
    error: (text) => showFailToast(text),
    info: (text) => showToast(text),
  }
}

const unwrap = (res) => {
  if (res == null) return null
  if (typeof res.code !== 'undefined' && res.code !== 0) {
    throw new Error(res.message || '请求失败')
  }
  return res.data != null ? res.data : res
}

const segKey = (v) => (v == null || v === '' ? '' : String(v))

const outlineWithBodyText = (outline) => (outline || []).filter((b) => (b.text || '').trim().length > 0)

const normalizeJsonArrayLike = (raw, isSingleRow) => {
  if (raw == null) return []
  if (Array.isArray(raw)) return raw.filter((x) => x && typeof x === 'object')
  if (typeof raw === 'object') {
    if (isSingleRow(raw)) return [raw]
    const vals = Object.values(raw).filter((x) => x && typeof x === 'object')
    if (vals.length) return vals
  }
  return []
}

const normalizeTeachingAssets = (raw) => {
  if (raw == null) return []
  if (Array.isArray(raw)) return raw.filter((a) => a && typeof a === 'object')
  if (typeof raw === 'object') {
    if (raw.segmentId != null && raw.imageUrl != null) return [raw]
    const vals = Object.values(raw).filter((x) => x && typeof x === 'object' && x.segmentId != null)
    if (vals.length) return vals
  }
  return []
}

const unwrapJsonValue = (raw) => {
  let v = raw
  for (let i = 0; i < 6; i++) {
    if (v == null) return null
    if (typeof v !== 'string') return v
    const t = v.trim()
    if (!t) return null
    if (t[0] !== '[' && t[0] !== '{') return v
    try {
      v = JSON.parse(t)
    } catch {
      return raw
    }
  }
  return v
}

const parseLoadedScript = (raw) => {
  if (raw == null) return null
  let s = raw
  if (typeof raw === 'string') {
    try {
      s = JSON.parse(raw)
    } catch {
      return null
    }
  }
  if (typeof s !== 'object' || s == null) return null
  let outline = Array.isArray(s.outline) ? s.outline : []
  outline = outline.map((b) => {
    if (!b || typeof b !== 'object') return { id: '', title: '', text: '' }
    const id = b.id ?? b.segmentId ?? b.segment_id ?? ''
    return {
      id,
      segmentId: b.segmentId ?? b.segment_id ?? id,
      title: typeof b.title === 'string' ? b.title : '',
      text: typeof b.text === 'string' ? b.text : '',
    }
  })
  const title = typeof s.title === 'string' ? s.title : ''
  if (!title && !outline.length) return null
  return { title, outline }
}

export function useTeachingStory(notifyFactory = defaultNotify) {
  const msg = notifyFactory()

  const step = ref(0)
  const topic = ref('')
  const loadingScript = ref(false)
  const scriptPayload = ref(null)
  const ttsLoading = ref(false)
  const ttsProgress = ref(0)
  const ttsItems = ref([])
  const imagePromptRows = ref([])
  const assetTasks = ref([])
  const recordId = ref(0)
  const recordsList = ref([])
  const recordsLoading = ref(false)
  const recordVideoUrl = ref('')
  const renderVideoLoading = ref(false)
  const nextLoading = ref(false)
  const promptsLoading = ref(false)

  let saveTimer = null
  let scriptOutlineSnapshot = ''

  const syncScriptOutlineSnapshot = () => {
    scriptOutlineSnapshot = JSON.stringify(scriptPayload.value?.outline || [])
  }

  const invalidateDownstreamFromScript = () => {
    ttsItems.value = []
    ttsProgress.value = 0
    imagePromptRows.value = []
    assetTasks.value = []
    recordVideoUrl.value = ''
  }

  const hasDownstreamFromScript = () =>
    ttsItems.value.length > 0 ||
    imagePromptRows.value.length > 0 ||
    assetTasks.value.some((t) => (t.previewImage || '').trim())

  const onScriptOutlineEdited = () => {
    const cur = JSON.stringify(scriptPayload.value?.outline || [])
    if (cur === scriptOutlineSnapshot) {
      scheduleSaveRecord()
      return
    }
    if (hasDownstreamFromScript()) {
      invalidateDownstreamFromScript()
      msg.info('脚本已修改，配音与分镜需重新生成')
    }
    scriptOutlineSnapshot = cur
    scheduleSaveRecord()
  }

  const removeScriptSegment = (idx) => {
    const outline = scriptPayload.value?.outline
    if (!outline?.length) return false
    if (outline.length <= 1) {
      msg.warning('至少保留一个段落')
      return false
    }
    outline.splice(idx, 1)
    onScriptOutlineEdited()
    return true
  }

  const apiBaseTrim = () => (process.env.VUE_APP_API_HOST || '').replace(/\/$/, '')

  const teachingMediaSrc = (url) => {
    if (!url || typeof url !== 'string') return ''
    const u = url.trim()
    if (!u) return ''
    const api = apiBaseTrim()
    if (u.startsWith('/') && !u.startsWith('//')) {
      return api ? `${api}${u}` : u
    }
    try {
      const abs = new URL(u)
      const needProxy =
        typeof window !== 'undefined' &&
        window.location?.protocol === 'https:' &&
        abs.protocol === 'http:'
      if (needProxy) {
        const enc = encodeURIComponent(abs.href)
        if (api) {
          try {
            const base = new URL(api.startsWith('//') ? `${window.location.protocol}${api}` : api)
            if (base.protocol === 'https:') {
              return `${api.replace(/\/$/, '')}/api/download?url=${enc}`
            }
          } catch {
            /* ignore */
          }
        }
        return `/api/download?url=${enc}`
      }
    } catch {
      return u
    }
    return u
  }

  const scheduleSaveRecord = () => {
    if (saveTimer) clearTimeout(saveTimer)
    saveTimer = setTimeout(() => {
      saveTimer = null
      saveRecord()
    }, 900)
  }

  const formatRecordTime = (unix) => {
    if (!unix) return ''
    return new Date(unix * 1000).toLocaleString()
  }

  const fetchRecords = async () => {
    recordsLoading.value = true
    try {
      const res = await httpGet('/api/teaching/record/list')
      const d = unwrap(res)
      recordsList.value = d.list || []
    } catch {
      recordsList.value = []
    } finally {
      recordsLoading.value = false
    }
  }

  const saveRecord = async () => {
    if (!topic.value.trim() && !recordId.value) return
    if (!scriptPayload.value && !recordId.value) return
    try {
      const assetsPayload = assetTasks.value
        .filter((t) => t.previewImage)
        .map((t) => ({ segmentId: t.segmentId, imageUrl: t.previewImage }))
      const res = await httpPost('/api/teaching/record/save', {
        id: recordId.value || 0,
        topic: topic.value.trim(),
        script: scriptPayload.value || {},
        tts_items: ttsItems.value || [],
        image_prompts: imagePromptRows.value || [],
        assets: assetsPayload,
        max_step: step.value,
      })
      const d = unwrap(res)
      if (d && d.id) recordId.value = d.id
      fetchRecords()
    } catch {
      /* silent */
    }
  }

  const parentSegmentIdForTts = (segmentId) => {
    const s = segKey(segmentId)
    if (!s) return ''
    const i = s.indexOf('__')
    if (i > 0) return s.slice(0, i)
    return s
  }

  const ttsRowForShot = (segmentId) => {
    const k = segKey(segmentId)
    const by = Object.fromEntries(ttsItems.value.map((t) => [segKey(t.segmentId), t]))
    return by[k] || by[segKey(parentSegmentIdForTts(k))] || {}
  }

  const hydrateAssets = (assetsRaw) => {
    const assetsArr = normalizeTeachingAssets(assetsRaw)
    const prompts = imagePromptRows.value
      .filter((r) => (r.prompt || '').trim())
      .map((r) => ({
        segmentId: r.segmentId || r.id,
        prompt: r.prompt,
      }))
    const byAsset = Object.fromEntries(assetsArr.map((a) => [segKey(a.segmentId), a]))
    const rowMeta = Object.fromEntries(imagePromptRows.value.map((r) => [segKey(r.segmentId || r.id), r]))
    assetTasks.value = prompts.map((p) => {
      const k = segKey(p.segmentId)
      const a = byAsset[k]
      const tts = ttsRowForShot(p.segmentId)
      const meta = rowMeta[k] || {}
      const done = !!(a && a.imageUrl)
      return {
        segmentId: p.segmentId,
        label: meta.title || p.segmentId,
        previewImage: a?.imageUrl || '',
        audioLabel: tts?.fileName || '',
        audioUrl: tts?.url || '',
        progress: done ? 100 : 0,
        status: done ? 'done' : 'pending',
        errorMsg: '',
      }
    })
  }

  const loadRecord = async (id) => {
    try {
      const res = await httpGet(`/api/teaching/record/${id}`)
      const d = unwrap(res)
      recordId.value = d.id
      topic.value = d.topic || ''
      scriptPayload.value = parseLoadedScript(unwrapJsonValue(d.script))
      syncScriptOutlineSnapshot()
      ttsItems.value = normalizeJsonArrayLike(
        unwrapJsonValue(d.tts_items),
        (o) => (o.segmentId != null && o.segmentId !== '') || o.url != null || o.fileName != null
      )
      imagePromptRows.value = normalizeJsonArrayLike(
        unwrapJsonValue(d.image_prompts),
        (o) => (o.prompt != null && String(o.prompt).trim() !== '') || o.segmentId != null || o.id != null
      )
      hydrateAssets(unwrapJsonValue(d.assets))
      const ms = typeof d.max_step === 'number' ? d.max_step : 0
      step.value = Math.min(Math.max(ms, 0), 3)
      recordVideoUrl.value = typeof d.video_url === 'string' ? d.video_url.trim() : ''
      msg.success('已加载记录')
    } catch (e) {
      msg.error(e.message || String(e))
    }
  }

  const newDraft = () => {
    recordId.value = 0
    recordVideoUrl.value = ''
    topic.value = ''
    scriptPayload.value = null
    scriptOutlineSnapshot = ''
    imagePromptRows.value = []
    ttsItems.value = []
    assetTasks.value = []
    step.value = 0
    ttsProgress.value = 0
    ttsLoading.value = false
  }

  const onGenerateScript = async () => {
    if (!topic.value.trim()) {
      msg.warning('请先填写故事主题')
      return
    }
    loadingScript.value = true
    try {
      const res = await httpPost('/api/teaching/script', { topic: topic.value.trim() }, { timeout: HTTP_TIMEOUT.teachingScript })
      const d = unwrap(res)
      scriptPayload.value = { title: d.title, outline: d.outline || [] }
      syncScriptOutlineSnapshot()
      imagePromptRows.value = []
      ttsItems.value = []
      assetTasks.value = []
      msg.success('故事脚本已生成')
      scheduleSaveRecord()
    } catch (e) {
      msg.error(e.message || String(e))
    } finally {
      loadingScript.value = false
    }
  }

  const regenScript = async () => {
    if (!topic.value.trim()) {
      msg.warning('请先填写故事主题')
      return
    }
    scriptPayload.value = null
    imagePromptRows.value = []
    ttsItems.value = []
    assetTasks.value = []
    await onGenerateScript()
  }

  const runTts = async () => {
    if (!scriptPayload.value?.outline?.length) return
    const segments = outlineWithBodyText(scriptPayload.value.outline).map((o) => ({
      id: o.id || o.segmentId,
      text: o.text || '',
    }))
    if (!segments.length) {
      msg.warning('请至少保留一节有正文的段落')
      return
    }
    ttsLoading.value = true
    ttsProgress.value = 5
    ttsItems.value = []
    try {
      const res = await httpPost('/api/teaching/tts', { segments }, { timeout: HTTP_TIMEOUT.teachingTts })
      const d = unwrap(res)
      ttsItems.value = (d.items || []).map((it) => ({
        segmentId: it.segmentId,
        fileName: it.fileName,
        url: it.url,
        durationSec: it.durationSec ?? 0,
      }))
      ttsProgress.value = 100
      msg.success('配音已完成')
      scheduleSaveRecord()
    } catch (e) {
      msg.error(e.message || String(e))
    } finally {
      ttsLoading.value = false
    }
  }

  const loadImagePrompts = async () => {
    if (!scriptPayload.value?.outline?.length) return false
    const bodyOutline = outlineWithBodyText(scriptPayload.value.outline)
    if (!bodyOutline.length) {
      msg.warning('请至少保留一节有正文的段落')
      return false
    }
    promptsLoading.value = true
    try {
      const res = await httpPost('/api/teaching/image-prompts', {
        topic: topic.value.trim(),
        outline: bodyOutline,
      }, { timeout: HTTP_TIMEOUT.teachingImagePrompts })
      const d = unwrap(res)
      imagePromptRows.value = d.prompts || []
      if (!imagePromptRows.value.length) {
        msg.warning('未返回分镜提示词')
        return false
      }
      scheduleSaveRecord()
      return true
    } catch (e) {
      msg.error(e.message || String(e))
      return false
    } finally {
      promptsLoading.value = false
    }
  }

  const patchTaskBySegment = (segmentId, partial) => {
    const idx = assetTasks.value.findIndex((t) => t.segmentId === segmentId)
    if (idx < 0) return
    const cur = assetTasks.value[idx]
    assetTasks.value.splice(idx, 1, { ...cur, ...partial })
  }

  const hasVisualStepData = () =>
    assetTasks.value.some((t) => (t.previewImage || '').trim())

  const runParallelAssets = async () => {
    const prompts = imagePromptRows.value
      .filter((r) => (r.prompt || '').trim())
      .map((r) => ({
        segmentId: r.segmentId || r.id,
        prompt: r.prompt,
      }))
    if (!prompts.length) {
      msg.warning('没有可生成的分镜配图')
      return
    }
    const rowMeta = Object.fromEntries(imagePromptRows.value.map((r) => [segKey(r.segmentId || r.id), r]))
    assetTasks.value = prompts.map((p) => {
      const k = segKey(p.segmentId)
      const tts = ttsRowForShot(p.segmentId)
      return {
        segmentId: p.segmentId,
        label: rowMeta[k]?.title || p.segmentId,
        previewImage: '',
        audioLabel: tts?.fileName || '',
        audioUrl: tts?.url || '',
        progress: 5,
        status: 'running',
        errorMsg: '',
      }
    })

    await Promise.all(
      prompts.map(async (p) => {
        try {
          const res = await httpPost('/api/teaching/images', { prompts: [p] }, { timeout: HTTP_TIMEOUT.teachingImage })
          const d = unwrap(res)
          const list = normalizeTeachingAssets(d.assets)
          const a = list[0]
          if (!a || !a.imageUrl) throw new Error('未返回配图地址')
          const sk = segKey(p.segmentId)
          const meta = rowMeta[sk] || {}
          const tts = ttsRowForShot(p.segmentId)
          patchTaskBySegment(p.segmentId, {
            label: meta.title || p.segmentId,
            previewImage: a.imageUrl,
            audioLabel: tts?.fileName || '',
            audioUrl: tts?.url || '',
            progress: 100,
            status: 'done',
            errorMsg: '',
          })
          scheduleSaveRecord()
        } catch (e) {
          patchTaskBySegment(p.segmentId, {
            status: 'error',
            errorMsg: e.message || String(e),
            progress: 0,
            previewImage: '',
          })
        }
      })
    )

    const allOk = assetTasks.value.length && assetTasks.value.every((t) => t.status === 'done')
    const anyOk = assetTasks.value.some((t) => t.status === 'done')
    if (allOk) msg.success('分镜画面已全部生成')
    else if (anyOk) msg.warning('部分分镜已生成，失败项可单独重试')
    else msg.error('分镜生成失败，请检查配图接口配置')
    scheduleSaveRecord()
  }

  const regenOneImage = async (segmentId) => {
    const row = imagePromptRows.value.find((r) => (r.segmentId || r.id) === segmentId)
    if (!row || !(row.prompt || '').trim()) {
      msg.warning('未找到对应提示词')
      return
    }
    const p = { segmentId: row.segmentId || row.id, prompt: row.prompt }
    const meta = row
    const tts = ttsRowForShot(p.segmentId)
    patchTaskBySegment(segmentId, {
      status: 'running',
      progress: 40,
      errorMsg: '',
      previewImage: '',
      label: meta.title || p.segmentId,
      audioLabel: tts?.fileName || '',
      audioUrl: tts?.url || '',
    })
    try {
      const res = await httpPost('/api/teaching/images', { prompts: [p] }, { timeout: HTTP_TIMEOUT.teachingImage })
      const d = unwrap(res)
      const list = normalizeTeachingAssets(d.assets)
      const a = list[0]
      if (!a || !a.imageUrl) throw new Error('未返回配图地址')
      patchTaskBySegment(segmentId, {
        previewImage: a.imageUrl,
        progress: 100,
        status: 'done',
        errorMsg: '',
      })
      msg.success('该分镜已更新')
      scheduleSaveRecord()
    } catch (e) {
      patchTaskBySegment(segmentId, {
        status: 'error',
        errorMsg: e.message || String(e),
        progress: 0,
      })
      msg.error(e.message || String(e))
    }
  }

  const onRenderVideo = async () => {
    if (!recordId.value) {
      msg.warning('请先保存或加载一条记录')
      return
    }
    renderVideoLoading.value = true
    try {
      const res = await httpPost('/api/teaching/record/render-video', { id: recordId.value }, { timeout: HTTP_TIMEOUT.teachingRenderVideo })
      const d = unwrap(res)
      recordVideoUrl.value = typeof d.video_url === 'string' ? d.video_url.trim() : ''
      msg.success('视频已合成')
      fetchRecords()
    } catch (e) {
      msg.error(e.message || String(e))
    } finally {
      renderVideoLoading.value = false
    }
  }

  const canNext = computed(() => {
    if (step.value === 0) return !!scriptPayload.value
    if (step.value === 1) return !ttsLoading.value && ttsItems.value.length > 0
    if (step.value === 2) return imagePromptRows.value.length > 0 && !promptsLoading.value
    if (step.value === 3) {
      return assetTasks.value.length > 0 && assetTasks.value.every((t) => t.status !== 'running')
    }
    return false
  })

  const canRenderVideo = computed(() => {
    if (!recordId.value || renderVideoLoading.value) return false
    const hasImg = assetTasks.value.some((t) => (t.previewImage || '').trim())
    const hasTts = Array.isArray(ttsItems.value) && ttsItems.value.some((t) => (t.url || '').trim())
    return hasImg && hasTts
  })

  watch(step, (s) => {
    if (s === 1 && scriptPayload.value && !ttsLoading.value && ttsItems.value.length === 0) {
      runTts()
    }
    if (s === 2 && imagePromptRows.value.length === 0 && !promptsLoading.value) {
      loadImagePrompts()
    }
  })

  const onPrev = () => {
    if (step.value <= 0) return
    step.value -= 1
  }

  const onNext = async () => {
    if (step.value === 0 && !scriptPayload.value) {
      msg.info('请先生成故事脚本')
      return
    }
    if (step.value === 1 && (ttsLoading.value || !ttsItems.value.length)) {
      msg.info('请等待配音完成')
      return
    }
    if (step.value === 2 && !imagePromptRows.value.length) {
      const ok = await loadImagePrompts()
      if (!ok) return
    }
    if (step.value === 3) {
      if (assetTasks.value.some((t) => t.status === 'running')) {
        msg.info('仍有分镜在生成中')
        return
      }
      if (!assetTasks.value.length) {
        msg.info('请确认已生成分镜提示词')
        return
      }
      msg.success('创作流程已完成')
      scheduleSaveRecord()
      return
    }
    if (step.value === 2) {
      nextLoading.value = true
      step.value = 3
      if (!hasVisualStepData()) {
        await runParallelAssets()
      }
      nextLoading.value = false
      return
    }
    step.value += 1
  }

  onMounted(() => {
    fetchRecords()
  })

  return {
    step,
    topic,
    loadingScript,
    scriptPayload,
    ttsLoading,
    ttsProgress,
    ttsItems,
    imagePromptRows,
    assetTasks,
    recordId,
    recordsList,
    recordsLoading,
    recordVideoUrl,
    renderVideoLoading,
    nextLoading,
    promptsLoading,
    canNext,
    canRenderVideo,
    teachingMediaSrc,
    formatRecordTime,
    fetchRecords,
    loadRecord,
    newDraft,
    onGenerateScript,
    regenScript,
    onScriptOutlineEdited,
    removeScriptSegment,
    runTts,
    loadImagePrompts,
    regenOneImage,
    onRenderVideo,
    onPrev,
    onNext,
  }
}
