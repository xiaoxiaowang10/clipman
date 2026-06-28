let data = []
let filtered = []
let filterId = 0
let selectMode = false
let sel = new Set()
let activeDate = ''

const $ = (s, o) => (o || document).querySelector(s)
const q = $('#q')
const list = $('#list')
const toast = $('#toast')
const stats = $('#stats')
const count = $('#count')
const batchBar = $('#batchBar')
const selCount = $('#selCount')
const dateNav = $('#dateNav')

function toggleSettings(force) {
  const s = document.getElementById('settings')
  const b = document.getElementById('btnSettings')
  s.classList.toggle('open', force)
  b.classList.toggle('active', s.classList.contains('open'))
}

document.addEventListener('click', e => {
  if (!e.target.closest('#settings') && !e.target.closest('#btnSettings')) toggleSettings(false)
})

document.addEventListener('keydown', e => {
  if (e.key === 'Escape') toggleSettings(false)
})

function toggleTheme() {
  const html = document.documentElement
  const theme = html.getAttribute('data-theme')
  if (theme === 'dark' || (!theme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    html.setAttribute('data-theme', 'light')
    localStorage.setItem('theme', 'light')
  } else {
    html.setAttribute('data-theme', 'dark')
    localStorage.setItem('theme', 'dark')
  }
  updateTheme()
}

function updateTheme() {
  const html = document.documentElement
  const theme = html.getAttribute('data-theme')
  const isDark = theme === 'dark' || (!theme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  $('#btnTheme').textContent = isDark ? '☀️' : '🌙'
}

function esc(s) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

function highlight(s, terms) {
  let r = esc(s.slice(0, 500))
  for (const t of terms) {
    if (t.length < 2) continue
    r = r.replace(new RegExp('(' + t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + ')', 'gi'), '<em>$1</em>')
  }
  return r
}

function fmtDate(s) {
  const t = new Date(s.slice(0, 10) + 'T00:00:00')
  const n = new Date()
  const y = new Date(n)
  y.setDate(y.getDate() - 1)
  const f = d => d.toISOString().slice(0, 10)
  if (s === f(n)) return '今天'
  if (s === f(y)) return '昨天'
  return s.slice(5)
}

function setDate(d) {
  activeDate = d
  render(q.value)
}

function toggleSelect() {
  selectMode = !selectMode
  $('#btnSelect').textContent = selectMode ? '取消' : '选择'
  if (!selectMode) sel.clear()
  updateSel()
}

function toggleSel(idx, e) {
  e.stopPropagation()
  if (sel.has(idx)) sel.delete(idx)
  else sel.add(idx)
  updateSel()
}

function displayIdxs() {
  const display = activeDate
    ? filtered.filter(d => d.time.slice(0, 10) === activeDate)
    : filtered
  return display.map(d => d.idx)
}

function selectAll() {
  const d = displayIdxs()
  if (d.length === sel.size && d.every(i => sel.has(i))) {
    sel.clear()
  } else {
    d.forEach(i => sel.add(i))
  }
  updateSel()
}

function updateSel() {
  const d = displayIdxs()
  const allS = d.length && d.every(i => sel.has(i))
  $('#btnAll').textContent = allS ? '取消全选' : '全选'
  selCount.textContent = '已选 ' + sel.size + ' 项'
  batchBar.classList.toggle('show', sel.size > 0)
  render(q.value)
}

function render(qv) {
  const id = ++filterId
  const terms = qv.trim().toLowerCase().split(/\s+/).filter(Boolean)

  // 搜索过滤
  if (terms.length) {
    filtered = data.map((d, i) => ({ ...d, idx: i }))
      .filter(d => terms.every(t =>
        d.text.toLowerCase().includes(t) ||
        d.time.toLowerCase().includes(t)
      ))
  } else {
    filtered = data.map((d, i) => ({ ...d, idx: i }))
  }
  if (id !== filterId) return

  // 日期导航
  const dates = [...new Set(filtered.map(d => d.time.slice(0, 10)))].sort().reverse()
  dateNav.innerHTML =
    '<button class="btn btn-sm' + (activeDate === '' ? ' active' : '') + '" onclick="setDate(\'\')">全部</button>' +
    dates.map(d =>
      '<button class="btn btn-sm' + (activeDate === d ? ' active' : '') +
      '" onclick="setDate(\'' + d + '\')">' + fmtDate(d) + '</button>'
    ).join('')

  // 当前显示条目
  const display = activeDate
    ? filtered.filter(d => d.time.slice(0, 10) === activeDate)
    : filtered

  stats.textContent = data.length + ' 条记录'
  if (activeDate) {
    stats.textContent += '，显示 ' + display.length + ' 条'
    if (qv.trim()) stats.textContent += '，搜索到 ' + filtered.length + ' 条'
  }
  count.textContent = display.length + ' / ' + data.length

  if (!display.length) {
    list.innerHTML = '<div class="empty">暂无结果</div>'
    return
  }

  // 按日期分组
  const groups = {}
  for (const d of display) {
    const k = d.time.slice(0, 10)
    if (!groups[k]) groups[k] = []
    groups[k].push(d)
  }
  const order = activeDate ? [activeDate] : dates

  // 渲染列表
  list.innerHTML = order.flatMap(k => {
    const header = '<div class="date-head">' + fmtDate(k) + '</div>'
    const items = groups[k].map(d => {
      const long = d.text.length > 200
      const body =
        '<div class="body">' +
          '<div class="time">' + esc(d.time) + '</div>' +
          '<div class="text' + (long ? ' collapsed' : '') + '" id="t' + d.idx + '" ondblclick="event.stopPropagation();editEntry(' + d.idx + ')">' + highlight(d.text, terms) + '</div>' +
          (long ? '<div class="expand-btn" onclick="event.stopPropagation();expand(' + d.idx + ')">展开全部</div>' : '') +
        '</div>'
      const action = selectMode
        ? '<input type="checkbox" onclick="toggleSel(' + d.idx + ',event)" ' +
          (sel.has(d.idx) ? 'checked' : '') + '>'
        : '<span class="pin-btn' + (d.pinned ? ' pinned' : '') + '" onclick="event.stopPropagation();togglePin(' + d.idx + ')">' + (d.pinned ? '📌' : '📍') + '</span>' +
          '<span class="del-btn" onclick="event.stopPropagation();del(' + d.idx + ')">×</span>'
      return '<div class="item" onclick="copy(' + d.idx + ')">' + body + action + '</div>'
    })
    return [header, ...items]
  }).join('')
}

function showToast(msg) {
  toast.textContent = msg
  toast.classList.add('show')
  setTimeout(() => toast.classList.remove('show'), 1500)
}

function copy(idx) {
  if (selectMode) return
  navigator.clipboard.writeText(data[idx].text).then(() => showToast('已复制'))
}

async function del(idx) {
  await fetch('/api?id=' + idx, { method: 'DELETE' })
  await fetchData()
  showToast('已删除')
}

async function delSelected() {
  const ids = [...sel].join('&id=')
  if (!ids) return
  sel.clear()
  updateSel()
  const r = await fetch('/api?id=' + ids, { method: 'DELETE' })
  if (!r.ok) return showToast('删除失败')
  await fetchData()
  showToast('已删除 ' + ids.split('&id=').length + ' 项')
}

async function exportData() {
  const r = await fetch('/api/export')
  if (!r.ok) return showToast('导出失败')
  const blob = await r.blob()
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = 'clipman.jl'
  a.click()
  URL.revokeObjectURL(a.href)
  showToast('已导出')
}

async function importData(input) {
  const file = input.files[0]
  if (!file) return
  const fd = new FormData()
  fd.append('file', file)
  const r = await fetch('/api/import', { method: 'POST', body: fd })
  input.value = ''
  if (!r.ok) return showToast('导入失败')
  await fetchData()
  showToast('已导入')
}

async function togglePin(idx) {
  const r = await fetch('/api?id=' + idx, { method: 'PATCH' })
  if (!r.ok) return showToast('操作失败')
  await fetchData()
  showToast(data[idx].pinned ? '已取消置顶' : '已置顶')
}

function editEntry(idx) {
  const el = document.getElementById('t' + idx)
  if (!el || el.contentEditable === 'true') return
  const old = data[idx].text
  el.contentEditable = 'true'
  el.focus()
  el.classList.remove('collapsed')
  const save = async () => {
    el.contentEditable = 'false'
    const t = el.textContent.trim()
    if (t && t !== old) {
      const r = await fetch('/api?id=' + idx + '&text=' + encodeURIComponent(t), { method: 'PUT' })
      if (!r.ok) { el.textContent = old; return showToast('编辑失败') }
      await fetchData()
      showToast('已编辑')
    } else {
      el.textContent = old
    }
  }
  el.addEventListener('blur', save, { once: true })
  el.addEventListener('keydown', e => {
    if (e.key === 'Enter') { e.preventDefault(); el.blur() }
    if (e.key === 'Escape') { el.textContent = old; el.contentEditable = 'false'; el.classList.toggle('collapsed', old.length > 200) }
  })
}

function expand(idx) {
  const el = document.getElementById('t' + idx)
  if (!el) return
  el.classList.toggle('collapsed')
  el.nextElementSibling.textContent = el.classList.contains('collapsed') ? '展开全部' : '收起'
}

async function clearAll() {
  const r = await fetch('/api', { method: 'DELETE' })
  if (!r.ok) return showToast('清空失败')
  await fetchData()
  showToast('已清空')
}

async function toggleAutoStart() {
  const btn = $('#btnAutoStart')
  const on = btn.textContent === '开机启动'
  const r = await fetch('/api/autostart?enable=' + on)
  if (!r.ok) return showToast('设置失败')
  btn.textContent = on ? '已开机启动' : '开机启动'
  showToast(on ? '已开启开机启动' : '已关闭开机启动')
}

async function fetchData() {
  try {
    const r = await fetch('/api?q=' + encodeURIComponent(q.value))
    data = (await r.json()) || []
    render(q.value)
  } catch (e) {
    // 忽略轮询错误
  }
}

const savedTheme = localStorage.getItem('theme')
if (savedTheme) document.documentElement.setAttribute('data-theme', savedTheme)
updateTheme()
q.addEventListener('input', e => render(e.target.value))
setInterval(fetchData, 1000)
fetchData()
