/**
 * 解析 Suno 歌词文本为行列表
 * @param {string} lyrics
 * @returns {{ id: number, text: string, type: 'section'|'sing'|'meta' }[]}
 */
export function parseLyricsLines(lyrics) {
  if (!lyrics || !String(lyrics).trim()) return []

  return String(lyrics)
    .split('\n')
    .map((line, index) => {
      const text = line.trim()
      if (!text) return null
      let type = 'sing'
      if (/^\[[^\]]+\]$/.test(text)) type = 'section'
      else if (/^\([^)]+\)$/.test(text)) type = 'meta'
      return { id: index, text, type }
    })
    .filter(Boolean)
}

/** 估算单行演唱时长（秒） */
function estimateLineDuration(text, type) {
  if (type === 'section') return 0
  if (type === 'meta') {
    const len = text.replace(/\s/g, '').length
    return Math.max(1.2, len * 0.22)
  }
  const chinese = (text.match(/[\u4e00-\u9fff]/g) || []).length
  const other = Math.max(text.replace(/\s/g, '').length - chinese, 0)
  // 中文约 2.8 字/秒，英文按字符略快
  return Math.max(2.2, chinese * 0.36 + other * 0.16)
}

/**
 * 根据总时长构建歌词时间轴（无 LRC 时的最佳估算）
 * @returns {{ id: number, type: string, start: number, end: number }[]}
 */
export function buildLyricTimeline(lines, duration) {
  if (!lines.length || !duration || duration <= 0) return []

  // 预留前奏/尾奏，避免一开播就滚歌词
  const intro = Math.min(Math.max(duration * 0.07, 4), 22)
  const outro = Math.min(Math.max(duration * 0.04, 2), 15)
  const vocalStart = intro
  const vocalEnd = Math.max(vocalStart + 1, duration - outro)
  const vocalDuration = vocalEnd - vocalStart

  const weighted = lines.map((line) => ({
    id: line.id,
    type: line.type,
    weight: estimateLineDuration(line.text, line.type),
  }))

  const totalWeight = weighted
    .filter((l) => l.type !== 'section')
    .reduce((sum, l) => sum + l.weight, 0)

  if (!totalWeight) return []

  const scale = vocalDuration / totalWeight
  const entries = []
  let cursor = vocalStart

  for (const line of weighted) {
    if (line.type === 'section') {
      entries.push({ id: line.id, type: line.type, start: cursor, end: cursor })
      continue
    }
    const dur = line.weight * scale
    entries.push({
      id: line.id,
      type: line.type,
      start: cursor,
      end: cursor + dur,
    })
    cursor += dur
  }

  // 段落标签对齐到其后第一句
  for (let i = 0; i < entries.length; i++) {
    if (entries[i].type !== 'section') continue
    const next = entries.slice(i + 1).find((e) => e.type !== 'section')
    if (next) {
      entries[i].start = Math.max(0, next.start - 0.35)
      entries[i].end = next.start
    }
  }

  return entries
}

/**
 * 根据时间轴查找当前应高亮的歌词行
 */
export function getActiveLyricIdFromTimeline(timeline, currentTime) {
  if (!timeline.length || currentTime < 0) return -1

  const singable = timeline.filter((e) => e.type !== 'section')
  if (!singable.length) return -1

  if (currentTime < singable[0].start) {
    return singable[0].id
  }

  const lead = 0.18
  let active = -1
  for (const entry of singable) {
    if (currentTime >= entry.start - lead && currentTime < entry.end) {
      active = entry.id
    }
  }

  if (active >= 0) return active

  if (currentTime >= singable[singable.length - 1].end) {
    return singable[singable.length - 1].id
  }

  // 落在两行间隙时取最近将开始的一句
  let nearest = singable[0].id
  let minGap = Infinity
  for (const entry of singable) {
    const gap = Math.abs(currentTime - entry.start)
    if (gap < minGap) {
      minGap = gap
      nearest = entry.id
    }
  }
  return nearest
}

/** @deprecated 使用 buildLyricTimeline + getActiveLyricIdFromTimeline */
export function getActiveLyricId(lines, currentTime, duration) {
  const timeline = buildLyricTimeline(lines, duration)
  return getActiveLyricIdFromTimeline(timeline, currentTime)
}

export function formatAudioTime(seconds) {
  if (!Number.isFinite(seconds) || seconds < 0) return '0:00'
  const total = Math.floor(seconds)
  const m = Math.floor(total / 60)
  const s = total % 60
  return `${m}:${String(s).padStart(2, '0')}`
}

/** 统一后端 MusicJob 字段（Go 默认 PascalCase） */
export function normalizeMusicJob(raw) {
  if (!raw) return null
  const progress = raw.Progress ?? raw.progress ?? 0
  return {
    Id: raw.Id ?? raw.id,
    TaskId: raw.TaskId ?? raw.task_id,
    Content: raw.Content ?? raw.content ?? '',
    AudioUrl: raw.AudioUrl ?? raw.audio_url ?? '',
    VideoUrl: raw.VideoUrl ?? raw.video_url ?? '',
    ImageUrl: raw.ImageUrl ?? raw.image_url ?? '',
    Lyrics: raw.Lyrics ?? raw.lyrics ?? '',
    Duration: Number(raw.Duration ?? raw.duration ?? 0) || 0,
    Progress: typeof progress === 'number' ? progress : parseInt(String(progress).replace('%', ''), 10) || 0,
    Status: raw.Status ?? raw.status ?? '',
    ErrMsg: raw.ErrMsg ?? raw.err_msg ?? '',
    CreatedAt: raw.CreatedAt ?? raw.created_at,
  }
}

export function normalizeMusicStatus(status) {
  const value = String(status || '').toLowerCase()
  if (value === 'success' || value === 'succeeded') return 'success'
  if (value === 'failed') return 'failed'
  return 'pending'
}

/** 任务是否已结束（成功或失败） */
export function isMusicJobDone(job) {
  const item = normalizeMusicJob(job)
  if (!item) return false
  const status = normalizeMusicStatus(item.Status)
  if (status === 'success' || status === 'failed') return true
  if (Number(item.Progress) >= 100 && item.AudioUrl) return true
  return false
}

export function getMusicJobProgress(job) {
  const item = normalizeMusicJob(job)
  if (!item) return 0
  return Number(item.Progress) || 0
}
