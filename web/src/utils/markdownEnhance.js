/** Post-render tweaks for AI markdown HTML (links, tables, dash lines). */
const DASH_LINE = /^[-–—•*]\s+/

function enhanceDashParagraphs(root) {
  root.querySelectorAll('p').forEach((p) => {
    const text = (p.textContent || '').trim()
    if (!DASH_LINE.test(text)) {
      return
    }
    p.classList.add('md-dash-item')
  })
}

export function enhanceMarkdownRoot(root) {
  if (!root) {
    return
  }

  root.querySelectorAll('a[href^="http"]').forEach((a) => {
    a.setAttribute('target', '_blank')
    a.setAttribute('rel', 'noopener noreferrer')
    a.classList.add('md-external-link')
  })

  root.querySelectorAll('table').forEach((table) => {
    if (table.parentElement?.classList.contains('md-table-wrap')) {
      return
    }
    const wrap = document.createElement('div')
    wrap.className = 'md-table-wrap'
    table.parentNode.insertBefore(wrap, table)
    wrap.appendChild(table)
  })

  enhanceDashParagraphs(root)
}
