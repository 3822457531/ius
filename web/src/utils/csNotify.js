/**
 * 客服新消息提示音（Web Audio，无需静态资源）
 */
let audioCtx = null

function getCtx() {
  if (typeof window === 'undefined') return null
  const AC = window.AudioContext || window.webkitAudioContext
  if (!AC) return null
  if (!audioCtx) audioCtx = new AC()
  if (audioCtx.state === 'suspended') {
    audioCtx.resume().catch(() => {})
  }
  return audioCtx
}

export function playCsNotifySound() {
  try {
    const ctx = getCtx()
    if (!ctx) return
    const now = ctx.currentTime
    const gain = ctx.createGain()
    gain.connect(ctx.destination)
    gain.gain.setValueAtTime(0.0001, now)
    gain.gain.exponentialRampToValueAtTime(0.22, now + 0.02)
    gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.32)

    const o1 = ctx.createOscillator()
    o1.type = 'sine'
    o1.frequency.setValueAtTime(880, now)
    o1.frequency.exponentialRampToValueAtTime(1320, now + 0.12)
    o1.connect(gain)
    o1.start(now)
    o1.stop(now + 0.34)

    const o2 = ctx.createOscillator()
    o2.type = 'sine'
    o2.frequency.setValueAtTime(1175, now + 0.08)
    o2.connect(gain)
    o2.start(now + 0.08)
    o2.stop(now + 0.34)
  } catch {
    // ignore autoplay / unsupported
  }
}
