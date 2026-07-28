<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const meta = ref({
  title: '情侣小程序管理台',
  appName: '-',
  version: '-',
  runMode: '-'
})
const dashboard = ref(null)
const couples = ref([])
const initialLoading = ref(true)
const loading = ref(false)
const errorMessage = ref('')
const refreshedAt = ref('')
const unbindingId = ref('')

let timer = null

const overview = computed(() => dashboard.value?.overview ?? {})
const runtime = computed(() => dashboard.value?.runtime ?? {})
const metrics = computed(() => dashboard.value?.metrics ?? [])
const recentUsers = computed(() => dashboard.value?.recentUsers ?? [])
const recentCouples = computed(() => couples.value.length ? couples.value : (dashboard.value?.recentCouples ?? []))
const errorLogs = computed(() => dashboard.value?.errorLogs ?? [])

const statCards = computed(() => [
  { label: '注册用户', value: overview.value.totalUsers, note: '平台累计用户数', tone: 'ink' },
  { label: '已配对情侣', value: overview.value.pairedCouples, note: '可在下方执行解绑', tone: 'green' },
  { label: '待确认分享码', value: overview.value.pendingPairCodes, note: '已生成但未输入确认', tone: 'amber' },
  { label: '24h 新用户', value: overview.value.newUsers24h, note: '最近一天新增', tone: 'blue' },
  { label: '7d 新配对', value: overview.value.newCouples7d, note: '最近七天完成配对', tone: 'rose' },
  { label: '错误日志', value: runtime.value.errorTotal, note: '累计错误数量', tone: 'red' }
])

const cpuChart = computed(() => buildChart(metrics.value, item => item.cpuPercent ?? 0))
const memoryChart = computed(() => buildChart(metrics.value, item => item.memoryPercent ?? 0))

async function requestData(url, options = {}) {
  const response = await fetch(url, {
    cache: 'no-store',
    credentials: 'same-origin',
    ...options
  })

  if (!response.ok) {
    throw new Error(`请求失败：${response.status}`)
  }

  const payload = await response.json()
  return payload.data ?? payload
}

async function refresh() {
  loading.value = true
  errorMessage.value = ''

  try {
    const [nextMeta, nextDashboard, nextCouples] = await Promise.all([
      requestData('/admin/api/meta'),
      requestData('/admin/api/dashboard'),
      requestData('/admin/api/couples?limit=100')
    ])

    meta.value = nextMeta
    dashboard.value = nextDashboard
    couples.value = nextCouples
    refreshedAt.value = formatDateTime(new Date())
    document.title = nextMeta.title || '情侣小程序管理台'
  } catch (error) {
    errorMessage.value = error?.message || String(error)
  } finally {
    loading.value = false
    initialLoading.value = false
  }
}

async function unbindCouple(couple) {
  if (!couple?.id || !couple.userBId) return
  const confirmed = window.confirm(`确认解除这对用户的绑定吗？\n\n情侣 ID：${couple.id}\n用户：${couple.userAId} / ${couple.userBId}`)
  if (!confirmed) return

  unbindingId.value = couple.id
  errorMessage.value = ''
  try {
    await requestData(`/admin/api/couples/${encodeURIComponent(couple.id)}/unbind`, {
      method: 'POST'
    })
    await refresh()
  } catch (error) {
    errorMessage.value = error?.message || String(error)
  } finally {
    unbindingId.value = ''
  }
}

function startTimer() {
  stopTimer()
  timer = window.setInterval(refresh, 10000)
}

function stopTimer() {
  if (!timer) return
  window.clearInterval(timer)
  timer = null
}

function buildChart(points, getter) {
  if (!points.length) return ''
  const width = 420
  const height = 120
  const padding = 10
  const values = points.map(getter)
  const upperBound = Math.max(100, ...values, 1)
  const stepX = values.length > 1 ? (width - padding * 2) / (values.length - 1) : 0
  return values
    .map((value, index) => {
      const x = padding + stepX * index
      const y = height - padding - ((height - padding * 2) * value) / upperBound
      return `${x},${y}`
    })
    .join(' ')
}

function formatNumber(value) {
  return new Intl.NumberFormat('zh-CN').format(Number(value || 0))
}

function formatPercent(value) {
  return `${Number(value || 0).toFixed(1)}%`
}

function formatMemory(value) {
  return `${Number(value || 0).toFixed(1)} MB`
}

function formatDateTime(value) {
  if (!value) return '-'
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

function formatUptime(seconds) {
  const total = Number(seconds || 0)
  const days = Math.floor(total / 86400)
  const hours = Math.floor((total % 86400) / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  if (days > 0) return `${days}天 ${hours}小时`
  if (hours > 0) return `${hours}小时 ${minutes}分钟`
  return `${minutes}分钟`
}

function pairStatus(couple) {
  return couple.userBId ? '已绑定' : '待确认'
}

function pad(value) {
  return String(value).padStart(2, '0')
}

onMounted(async () => {
  await refresh()
  startTimer()
})

onBeforeUnmount(() => {
  stopTimer()
})
</script>

<template>
  <main class="page-shell">
    <section class="hero-panel panel">
      <div>
        <p class="eyebrow">Couple Mini Ops</p>
        <h1>{{ meta.title }}</h1>
        <p class="hero-copy">
          查看小程序运行状态、用户增长和情侣绑定关系。危险操作集中在表格内，执行解绑前会再次确认。
        </p>
      </div>
      <div class="hero-actions">
        <div class="meta-stack">
          <span>{{ meta.appName }}</span>
          <strong>{{ meta.version }} · {{ meta.runMode }}</strong>
          <small>最后刷新：{{ refreshedAt || '-' }}</small>
        </div>
        <button class="refresh-btn" :disabled="loading" @click="refresh">
          {{ loading ? '刷新中...' : '立即刷新' }}
        </button>
      </div>
    </section>

    <div v-if="errorMessage" class="error-banner">
      {{ errorMessage }}
    </div>

    <section class="stats-grid">
      <article v-for="card in statCards" :key="card.label" class="stat-card panel" :class="`tone-${card.tone}`">
        <span>{{ card.label }}</span>
        <strong>{{ formatNumber(card.value) }}</strong>
        <p>{{ card.note }}</p>
      </article>
    </section>

    <section class="ops-grid">
      <article class="panel runtime-card">
        <div class="section-head">
          <div>
            <p class="eyebrow">Runtime</p>
            <h2>服务健康</h2>
          </div>
          <span class="health-pill">在线</span>
        </div>
        <div class="runtime-list">
          <div><span>运行时长</span><strong>{{ formatUptime(runtime.uptimeSeconds) }}</strong></div>
          <div><span>请求总数</span><strong>{{ formatNumber(runtime.requestTotal) }}</strong></div>
          <div><span>Goroutines</span><strong>{{ formatNumber(runtime.goroutines) }}</strong></div>
          <div><span>进程内存</span><strong>{{ formatMemory(runtime.processMemoryMB) }}</strong></div>
        </div>
      </article>

      <article class="panel chart-card">
        <div class="section-head">
          <div>
            <p class="eyebrow">Resources</p>
            <h2>资源趋势</h2>
          </div>
          <span>{{ formatPercent(runtime.lastCpuPercent) }} / {{ formatPercent(runtime.lastMemoryPercent) }}</span>
        </div>
        <svg viewBox="0 0 420 120" preserveAspectRatio="none">
          <polyline v-if="cpuChart" :points="cpuChart" class="line-cpu" />
          <polyline v-if="memoryChart" :points="memoryChart" class="line-memory" />
        </svg>
        <div class="legend">
          <span><i class="cpu-dot"></i>CPU</span>
          <span><i class="memory-dot"></i>内存</span>
        </div>
      </article>
    </section>

    <section class="dual-grid">
      <article class="panel table-card">
        <div class="section-head">
          <div>
            <p class="eyebrow">Users</p>
            <h2>最近用户</h2>
          </div>
        </div>
        <div v-if="recentUsers.length" class="table-shell">
          <table>
            <thead>
              <tr>
                <th>昵称</th>
                <th>用户 ID</th>
                <th>微信号</th>
                <th>创建时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="user in recentUsers" :key="user.id">
                <td>{{ user.nickname || '-' }}</td>
                <td>{{ user.id }}</td>
                <td>{{ user.wxid || '-' }}</td>
                <td>{{ formatDateTime(user.createdAt) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="empty-box">暂无用户数据</div>
      </article>

      <article class="panel table-card">
        <div class="section-head">
          <div>
            <p class="eyebrow">Couples</p>
            <h2>最近情侣配对</h2>
          </div>
          <span class="danger-note">支持管理员解绑</span>
        </div>
        <div v-if="recentCouples.length" class="table-shell">
          <table>
            <thead>
              <tr>
                <th>情侣 ID</th>
                <th>用户组合</th>
                <th>恋爱日</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="couple in recentCouples" :key="couple.id">
                <td>{{ couple.id }}</td>
                <td>{{ couple.userAId }} / {{ couple.userBId || '待确认' }}</td>
                <td>{{ couple.loveDate || '-' }}</td>
                <td>
                  <span class="status-chip" :class="{ pending: !couple.userBId }">
                    {{ pairStatus(couple) }}
                  </span>
                </td>
                <td>
                  <button
                    class="danger-btn"
                    :disabled="!couple.userBId || unbindingId === couple.id"
                    @click="unbindCouple(couple)"
                  >
                    {{ unbindingId === couple.id ? '解绑中...' : '解除绑定' }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="empty-box">暂无情侣数据</div>
      </article>
    </section>

    <section class="panel log-card">
      <div class="section-head">
        <div>
          <p class="eyebrow">Logs</p>
          <h2>最近错误日志</h2>
        </div>
      </div>
      <div v-if="errorLogs.length" class="log-list">
        <article v-for="(item, index) in errorLogs" :key="`${item.time}-${index}`" class="log-item">
          <span>{{ (item.level || 'info').toUpperCase() }} · {{ item.time || '-' }}</span>
          <strong>{{ item.message || 'No message' }}</strong>
          <p v-if="item.error">{{ item.error }}</p>
        </article>
      </div>
      <div v-else class="empty-box">最近没有 error / warn 日志</div>
    </section>

    <div v-if="initialLoading" class="loading-mask">
      <div class="loading-card">
        <strong>管理台加载中</strong>
        <p>正在读取后端状态和情侣关系...</p>
      </div>
    </div>
  </main>
</template>

<style>
:root {
  color-scheme: light;
  --paper: #f5efe5;
  --panel: rgba(255, 252, 246, 0.94);
  --ink: #211b16;
  --muted: #74695e;
  --line: rgba(33, 27, 22, 0.1);
  --green: #1f7a4d;
  --amber: #b76418;
  --blue: #145f72;
  --rose: #bb4669;
  --red: #b42318;
  --shadow: 0 24px 70px rgba(77, 49, 24, 0.13);
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  min-height: 100vh;
  color: var(--ink);
  font-family: "LXGW WenKai", "Noto Serif SC", "Microsoft YaHei", sans-serif;
  background:
    radial-gradient(circle at 6% 0%, rgba(183, 100, 24, 0.2), transparent 28%),
    radial-gradient(circle at 90% 10%, rgba(20, 95, 114, 0.14), transparent 24%),
    linear-gradient(180deg, #fbf6ee 0%, var(--paper) 100%);
}

button {
  font: inherit;
}

.page-shell {
  width: min(1480px, calc(100vw - 32px));
  margin: 24px auto 56px;
}

.panel {
  border: 1px solid var(--line);
  border-radius: 28px;
  background: var(--panel);
  box-shadow: var(--shadow);
}

.hero-panel {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  padding: 30px;
}

.eyebrow {
  margin: 0 0 8px;
  color: var(--amber);
  font-size: 12px;
  font-weight: 900;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

h1,
h2,
p {
  margin-top: 0;
}

h1 {
  margin-bottom: 12px;
  font-size: clamp(34px, 4vw, 56px);
  line-height: 1.02;
  letter-spacing: -0.04em;
}

h2 {
  margin-bottom: 0;
  font-size: 22px;
}

.hero-copy,
.stat-card p,
.empty-box,
.log-item p {
  color: var(--muted);
  line-height: 1.7;
}

.hero-actions {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.meta-stack {
  display: grid;
  gap: 5px;
  min-width: 220px;
  padding: 14px 16px;
  border-radius: 18px;
  background: rgba(33, 27, 22, 0.05);
}

.meta-stack span,
.meta-stack small,
th {
  color: var(--muted);
  font-size: 12px;
}

.refresh-btn,
.danger-btn {
  border: 0;
  border-radius: 999px;
  cursor: pointer;
  font-weight: 900;
}

.refresh-btn {
  padding: 12px 18px;
  color: #fff;
  background: #211b16;
}

.refresh-btn:disabled,
.danger-btn:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.error-banner {
  margin: 16px 0;
  padding: 14px 18px;
  border: 1px solid rgba(180, 35, 24, 0.24);
  border-radius: 18px;
  color: var(--red);
  background: rgba(180, 35, 24, 0.08);
}

.stats-grid,
.ops-grid,
.dual-grid {
  display: grid;
  gap: 14px;
  margin-top: 14px;
}

.stats-grid {
  grid-template-columns: repeat(6, minmax(0, 1fr));
}

.ops-grid,
.dual-grid {
  grid-template-columns: 1fr 1.35fr;
}

.stat-card,
.runtime-card,
.chart-card,
.table-card,
.log-card {
  padding: 20px;
}

.stat-card span {
  color: var(--muted);
  font-size: 13px;
  font-weight: 900;
}

.stat-card strong {
  display: block;
  margin: 18px 0 8px;
  font-size: 38px;
  line-height: 1;
}

.tone-green strong { color: var(--green); }
.tone-amber strong { color: var(--amber); }
.tone-blue strong { color: var(--blue); }
.tone-rose strong { color: var(--rose); }
.tone-red strong { color: var(--red); }

.section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}

.health-pill,
.danger-note,
.status-chip {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  padding: 7px 11px;
  font-size: 12px;
  font-weight: 900;
}

.health-pill,
.status-chip {
  color: var(--green);
  background: rgba(31, 122, 77, 0.11);
}

.status-chip.pending {
  color: var(--amber);
  background: rgba(183, 100, 24, 0.11);
}

.danger-note {
  color: var(--red);
  background: rgba(180, 35, 24, 0.08);
}

.runtime-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.runtime-list div {
  padding: 14px;
  border-radius: 18px;
  background: rgba(33, 27, 22, 0.04);
}

.runtime-list span {
  display: block;
  color: var(--muted);
  font-size: 12px;
}

.runtime-list strong {
  display: block;
  margin-top: 7px;
  font-size: 22px;
}

svg {
  width: 100%;
  height: 170px;
  border-radius: 20px;
  background: rgba(33, 27, 22, 0.04);
}

.line-cpu,
.line-memory {
  fill: none;
  stroke-width: 4;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.line-cpu { stroke: var(--amber); }
.line-memory { stroke: var(--blue); }

.legend {
  display: flex;
  gap: 12px;
  margin-top: 10px;
  color: var(--muted);
  font-size: 13px;
}

.legend i {
  display: inline-block;
  width: 9px;
  height: 9px;
  margin-right: 6px;
  border-radius: 999px;
}

.cpu-dot { background: var(--amber); }
.memory-dot { background: var(--blue); }

.table-shell {
  overflow: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th,
td {
  padding: 13px 10px;
  border-bottom: 1px solid var(--line);
  text-align: left;
  vertical-align: middle;
}

td {
  font-size: 14px;
}

.danger-btn {
  padding: 9px 13px;
  color: #fff;
  background: linear-gradient(135deg, #d33f49, #a91e24);
}

.empty-box {
  padding: 32px 18px;
  border: 1px dashed var(--line);
  border-radius: 20px;
  text-align: center;
  background: rgba(33, 27, 22, 0.03);
}

.log-card {
  margin-top: 14px;
}

.log-list {
  display: grid;
  gap: 10px;
  max-height: 440px;
  overflow: auto;
}

.log-item {
  padding: 14px;
  border: 1px solid var(--line);
  border-radius: 18px;
  background: rgba(33, 27, 22, 0.03);
}

.log-item span {
  color: var(--red);
  font-size: 12px;
  font-weight: 900;
}

.log-item strong {
  display: block;
  margin-top: 6px;
}

.loading-mask {
  position: fixed;
  inset: 0;
  display: grid;
  place-items: center;
  background: rgba(245, 239, 229, 0.72);
  backdrop-filter: blur(8px);
}

.loading-card {
  width: min(360px, calc(100vw - 32px));
  padding: 24px;
  border: 1px solid var(--line);
  border-radius: 24px;
  background: var(--panel);
  box-shadow: var(--shadow);
}

@media (max-width: 1180px) {
  .stats-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .ops-grid,
  .dual-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .page-shell {
    width: min(100vw - 20px, 100%);
    margin-top: 10px;
  }

  .hero-panel,
  .hero-actions {
    flex-direction: column;
  }

  .stats-grid,
  .runtime-list {
    grid-template-columns: 1fr;
  }
}
</style>
