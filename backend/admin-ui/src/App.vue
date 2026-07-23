<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const meta = ref({
  title: 'Couple Mini Admin',
  appName: '-',
  version: '-',
  runMode: '-'
})

const dashboard = ref(null)
const initialLoading = ref(true)
const loading = ref(false)
const errorMessage = ref('')
const refreshedAt = ref('')

let timer = null

const statCards = [
  { key: 'totalUsers', label: '使用人数', note: '平台注册总用户数', tone: 'warm' },
  { key: 'pairedCouples', label: '匹配情侣对数', note: '已经完成双向确认的情侣关系', tone: 'cool' },
  { key: 'pendingPairCodes', label: '待匹配邀请码', note: '已创建但仍待确认的邀请码', tone: 'plain' },
  { key: 'newUsers24h', label: '24h 新用户', note: '最近 24 小时新增用户', tone: 'plain' },
  { key: 'newCouples7d', label: '7d 新配对', note: '最近 7 天新增情侣关系', tone: 'plain' },
  { key: 'openTasks', label: '未完成任务', note: '状态不是 done 的任务总数', tone: 'plain' },
  { key: 'activeGoals', label: '活跃目标', note: '当前处于 active 状态的目标', tone: 'plain' },
  { key: 'scheduledTasks', label: '周期任务', note: '提醒与周期计划总数', tone: 'plain' },
  { key: 'totalMoments', label: '纪念时刻', note: '沉淀中的 moments 内容量', tone: 'plain' },
  { key: 'totalOrders', label: '点单记录', note: '历史吃什么决策记录', tone: 'plain' },
  { key: 'enabledDishes', label: '启用菜品', note: '当前可参与抽选的菜品', tone: 'plain' }
]

const overview = computed(() => dashboard.value?.overview ?? {})
const runtime = computed(() => dashboard.value?.runtime ?? {})
const metrics = computed(() => dashboard.value?.metrics ?? [])
const recentUsers = computed(() => dashboard.value?.recentUsers ?? [])
const recentCouples = computed(() => dashboard.value?.recentCouples ?? [])
const errorLogs = computed(() => dashboard.value?.errorLogs ?? [])

const cpuChart = computed(() => buildChart(metrics.value, item => item.cpuPercent ?? 0))
const memoryChart = computed(() => buildChart(metrics.value, item => item.memoryPercent ?? 0))

const logStats = computed(() => {
  const total = errorLogs.value.length
  const errorCount = errorLogs.value.filter(item => (item.level || '').toLowerCase() === 'error').length
  const warnCount = errorLogs.value.filter(item => (item.level || '').toLowerCase() === 'warn').length
  return { total, errorCount, warnCount }
})

const latestMetricTime = computed(() => {
  const lastPoint = metrics.value[metrics.value.length - 1]
  return lastPoint?.timestamp ? formatDateTime(lastPoint.timestamp) : '-'
})

const opsSummaries = computed(() => [
  { label: '待匹配邀请码', value: `${formatNumber(overview.value.pendingPairCodes)} 个` },
  { label: '未完成任务', value: `${formatNumber(overview.value.openTasks)} 项` },
  { label: '最近错误日志', value: `${formatNumber(logStats.value.total)} 条` },
  { label: '活跃目标', value: `${formatNumber(overview.value.activeGoals)} 个` }
])

const systemAlerts = computed(() => {
  const alerts = []
  const cpu = Number(runtime.value.lastCpuPercent || 0)
  const memory = Number(runtime.value.lastMemoryPercent || 0)
  const pendingCodes = Number(overview.value.pendingPairCodes || 0)
  const openTasks = Number(overview.value.openTasks || 0)

  if (cpu >= 80) {
    alerts.push({
      level: 'critical',
      title: 'CPU 负载偏高',
      detail: `当前 CPU 使用率 ${formatPercent(cpu)}，建议排查高频接口或循环任务。`
    })
  }

  if (memory >= 80) {
    alerts.push({
      level: 'warning',
      title: '内存占用持续偏高',
      detail: `当前内存使用率 ${formatPercent(memory)}，建议结合错误日志检查是否有异常堆积。`
    })
  }

  if (pendingCodes >= 20) {
    alerts.push({
      level: 'warning',
      title: '待确认邀请码较多',
      detail: `当前仍有 ${formatNumber(pendingCodes)} 个邀请码待匹配，建议观察配对流程是否卡住。`
    })
  }

  if (openTasks >= 30) {
    alerts.push({
      level: 'info',
      title: '未完成任务积压',
      detail: `当前未完成任务 ${formatNumber(openTasks)} 项，可以关注是否存在长期未流转任务。`
    })
  }

  if (logStats.value.errorCount > 0) {
    alerts.push({
      level: 'critical',
      title: '最近出现错误日志',
      detail: `最近日志中捕获到 ${formatNumber(logStats.value.errorCount)} 条 error，建议优先查看下方错误日志列表。`
    })
  } else if (logStats.value.warnCount > 0) {
    alerts.push({
      level: 'info',
      title: '最近存在警告日志',
      detail: `最近日志中出现 ${formatNumber(logStats.value.warnCount)} 条 warn，可作为巡检参考。`
    })
  }

  if (!alerts.length) {
    alerts.push({
      level: 'healthy',
      title: '系统状态稳定',
      detail: '当前没有明显异常指标，服务、配对和基础资源都处于健康范围。'
    })
  }

  return alerts.slice(0, 4)
})

async function requestData(url) {
  const response = await fetch(url, {
    cache: 'no-store',
    credentials: 'same-origin'
  })

  if (!response.ok) {
    throw new Error(`请求失败: ${response.status}`)
  }

  const payload = await response.json()
  return payload.data ?? payload
}

async function refresh() {
  loading.value = true
  errorMessage.value = ''

  try {
    const [nextMeta, nextDashboard] = await Promise.all([
      requestData('/admin/api/meta'),
      requestData('/admin/api/dashboard')
    ])

    meta.value = nextMeta
    dashboard.value = nextDashboard
    refreshedAt.value = formatDateTime(new Date())
    document.title = nextMeta.title || 'Couple Mini Admin'
  } catch (error) {
    errorMessage.value = error?.message || String(error)
  } finally {
    loading.value = false
    initialLoading.value = false
  }
}

function startTimer() {
  stopTimer()
  timer = window.setInterval(refresh, 10000)
}

function stopTimer() {
  if (timer) {
    window.clearInterval(timer)
    timer = null
  }
}

function buildChart(points, getter) {
  if (!points.length) {
    return { line: '', area: '', guides: [] }
  }

  const width = 560
  const height = 220
  const padding = 20
  const values = points.map(getter)
  const upperBound = Math.max(100, ...values, 1)
  const stepX = values.length > 1 ? (width - padding * 2) / (values.length - 1) : 0

  const line = values
    .map((value, index) => {
      const x = padding + stepX * index
      const y = height - padding - ((height - padding * 2) * value) / upperBound
      return `${x},${y}`
    })
    .join(' ')

  const area = `${padding},${height - padding} ${line} ${width - padding},${height - padding}`
  const guides = [0, 25, 50, 75, 100].map(mark => ({
    mark,
    y: height - padding - ((height - padding * 2) * mark) / upperBound
  }))

  return { line, area, guides }
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

  if (days > 0) return `${days}天 ${hours}小时 ${minutes}分钟`
  if (hours > 0) return `${hours}小时 ${minutes}分钟`
  return `${minutes}分钟`
}

function pad(value) {
  return String(value).padStart(2, '0')
}

function logLevelClass(level) {
  switch ((level || '').toLowerCase()) {
    case 'error':
      return 'pill-error'
    case 'warn':
      return 'pill-warn'
    default:
      return 'pill-info'
  }
}

function alertClass(level) {
  switch (level) {
    case 'critical':
      return 'alert-critical'
    case 'warning':
      return 'alert-warning'
    case 'healthy':
      return 'alert-healthy'
    default:
      return 'alert-info'
  }
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
  <div class="page-shell">
    <section class="hero-grid">
      <div class="panel hero-panel">
        <div class="hero-badge">Vue Admin</div>
        <h1>{{ meta.title }}</h1>
        <p class="hero-copy">
          面向情侣小程序的运营管理台。这里把业务概览、配对进度、系统资源曲线和错误日志放在同一页，
          方便你上线后快速巡检、查错和观察增长。
        </p>

        <div class="hero-meta">
          <div class="meta-chip">
            <span>应用</span>
            <strong>{{ meta.appName }}</strong>
          </div>
          <div class="meta-chip">
            <span>版本</span>
            <strong>{{ meta.version }}</strong>
          </div>
          <div class="meta-chip">
            <span>模式</span>
            <strong>{{ meta.runMode }}</strong>
          </div>
          <div class="meta-chip">
            <span>刷新时间</span>
            <strong>{{ refreshedAt || '-' }}</strong>
          </div>
        </div>
      </div>

      <div class="panel runtime-panel">
        <div class="runtime-header">
          <div>
            <p class="eyebrow">Runtime</p>
            <h2>服务健康快照</h2>
            <p class="section-copy">把运行时状态、请求量和错误累计先看一眼，再往下钻日志。</p>
          </div>
          <button class="refresh-btn" :disabled="loading" @click="refresh">
            {{ loading ? '刷新中...' : '立即刷新' }}
          </button>
        </div>

        <div class="runtime-grid">
          <div class="runtime-card">
            <span>服务运行时长</span>
            <strong>{{ formatUptime(runtime.uptimeSeconds) }}</strong>
          </div>
          <div class="runtime-card">
            <span>总请求数</span>
            <strong>{{ formatNumber(runtime.requestTotal) }}</strong>
          </div>
          <div class="runtime-card">
            <span>错误总数</span>
            <strong>{{ formatNumber(runtime.errorTotal) }}</strong>
          </div>
          <div class="runtime-card">
            <span>Goroutines</span>
            <strong>{{ formatNumber(runtime.goroutines) }}</strong>
          </div>
        </div>
      </div>
    </section>

    <section class="section-block">
      <div class="section-head">
        <div>
          <h2>巡检提醒</h2>
          <p>把最值得你优先看的风险点和健康信号提前提炼出来，减少翻日志的时间。</p>
        </div>
        <div class="metric-chip">最近采样 {{ latestMetricTime }}</div>
      </div>

      <div class="alert-grid">
        <article
          v-for="item in systemAlerts"
          :key="item.title"
          class="panel alert-card"
          :class="alertClass(item.level)"
        >
          <span class="alert-level">{{ item.level }}</span>
          <strong>{{ item.title }}</strong>
          <p>{{ item.detail }}</p>
        </article>
      </div>
    </section>

    <section class="section-block">
      <div class="section-head">
        <div>
          <h2>核心业务指标</h2>
          <p>从新增、配对、任务和内容沉淀几个方向看整体业务状态。</p>
        </div>
      </div>

      <div class="stats-grid">
        <article
          v-for="card in statCards"
          :key="card.key"
          class="panel stat-card"
          :class="`tone-${card.tone}`"
        >
          <span class="stat-label">{{ card.label }}</span>
          <strong class="stat-value">{{ formatNumber(overview[card.key]) }}</strong>
          <p class="stat-note">{{ card.note }}</p>
        </article>
      </div>
    </section>

    <section class="section-block">
      <div class="section-head">
        <div>
          <h2>系统资源与趋势</h2>
          <p>CPU、内存和运维摘要适合用来观察线上抖动、峰值与持续积压。</p>
        </div>
      </div>

      <div class="chart-layout">
        <div class="panel chart-card">
          <div class="chart-top">
            <div>
              <span class="chart-label">CPU 使用率</span>
              <strong>{{ formatPercent(runtime.lastCpuPercent) }}</strong>
            </div>
            <span class="chart-pill chart-pill-warm">实时曲线</span>
          </div>

          <div class="chart-box">
            <svg viewBox="0 0 560 220" preserveAspectRatio="none" aria-label="CPU usage chart">
              <line
                v-for="guide in cpuChart.guides"
                :key="`cpu-${guide.mark}`"
                x1="20"
                :y1="guide.y"
                x2="540"
                :y2="guide.y"
                class="guide-line"
              />
              <polyline v-if="cpuChart.area" :points="cpuChart.area" class="area-warm" />
              <polyline v-if="cpuChart.line" :points="cpuChart.line" class="line-warm" />
            </svg>
          </div>
        </div>

        <div class="panel chart-card">
          <div class="chart-top">
            <div>
              <span class="chart-label">内存使用率</span>
              <strong>{{ formatPercent(runtime.lastMemoryPercent) }}</strong>
            </div>
            <span class="chart-pill chart-pill-cool">实时曲线</span>
          </div>

          <div class="chart-box">
            <svg viewBox="0 0 560 220" preserveAspectRatio="none" aria-label="Memory usage chart">
              <line
                v-for="guide in memoryChart.guides"
                :key="`memory-${guide.mark}`"
                x1="20"
                :y1="guide.y"
                x2="540"
                :y2="guide.y"
                class="guide-line"
              />
              <polyline v-if="memoryChart.area" :points="memoryChart.area" class="area-cool" />
              <polyline v-if="memoryChart.line" :points="memoryChart.line" class="line-cool" />
            </svg>
          </div>
        </div>

        <div class="panel insights-card">
          <div class="chart-top">
            <div>
              <span class="chart-label">运维摘要</span>
              <strong>{{ formatMemory(runtime.processMemoryMB) }}</strong>
            </div>
            <span class="chart-pill chart-pill-neutral">重点巡检</span>
          </div>

          <div class="insight-list">
            <div v-for="item in opsSummaries" :key="item.label" class="insight-item">
              <span>{{ item.label }}</span>
              <strong>{{ item.value }}</strong>
            </div>
          </div>

          <p class="insight-note">
            建议优先关注错误总数、待匹配邀请码和未完成任务量，这三项最容易帮助你快速定位业务堵点。
          </p>
        </div>
      </div>
    </section>

    <section class="section-block dual-layout">
      <div class="panel table-card">
        <div class="section-head compact">
          <div>
            <h2>最近用户</h2>
            <p>观察最近注册与活跃开通节奏，判断增长是否正常。</p>
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
      </div>

      <div class="panel table-card">
        <div class="section-head compact">
          <div>
            <h2>最近情侣配对</h2>
            <p>快速判断最近匹配是否顺畅，也能看出待确认邀请码是否变多。</p>
          </div>
        </div>

        <div v-if="recentCouples.length" class="table-shell">
          <table>
            <thead>
              <tr>
                <th>情侣 ID</th>
                <th>用户组合</th>
                <th>恋爱日</th>
                <th>状态</th>
                <th>创建时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="couple in recentCouples" :key="couple.id">
                <td>{{ couple.id }}</td>
                <td>{{ couple.userAId }} / {{ couple.userBId || '待确认' }}</td>
                <td>{{ couple.loveDate || '-' }}</td>
                <td>{{ couple.userBId ? '已配对' : '待确认' }}</td>
                <td>{{ formatDateTime(couple.createdAt) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="empty-box">暂无情侣数据</div>
      </div>
    </section>

    <section class="section-block">
      <div class="panel log-card">
        <div class="section-head compact">
          <div>
            <h2>错误日志捕获</h2>
            <p>最近的 error 和 warn 会展示在这里，适合配合 Request ID 做线上排查。</p>
          </div>
          <div class="log-summary">
            <span class="summary-pill summary-pill-error">Error {{ formatNumber(logStats.errorCount) }}</span>
            <span class="summary-pill summary-pill-warn">Warn {{ formatNumber(logStats.warnCount) }}</span>
          </div>
        </div>

        <div v-if="errorMessage" class="empty-box danger-box">
          管理台数据加载失败：{{ errorMessage }}
        </div>

        <div v-else-if="errorLogs.length" class="log-list">
          <article v-for="(item, index) in errorLogs" :key="`${item.time}-${index}`" class="log-item">
            <div class="log-head">
              <span class="level-pill" :class="logLevelClass(item.level)">
                {{ (item.level || 'info').toUpperCase() }}
              </span>
              <span>{{ item.time || '-' }}</span>
              <span v-if="item.requestId">Request ID: {{ item.requestId }}</span>
              <span v-if="item.path">Path: {{ item.path }}</span>
            </div>
            <strong class="log-title">{{ item.message || 'No message' }}</strong>
            <p v-if="item.error" class="log-error">Error: {{ item.error }}</p>
          </article>
        </div>

        <div v-else class="empty-box">最近没有捕获到 error / warn 日志</div>
      </div>
    </section>

    <div v-if="initialLoading" class="loading-mask">
      <div class="loading-card">
        <strong>管理台加载中</strong>
        <p>正在拉取后台概览、系统指标和日志数据...</p>
      </div>
    </div>
  </div>
</template>

<style>
:root {
  color-scheme: light;
  --bg: #f4efe7;
  --panel: rgba(255, 252, 247, 0.93);
  --panel-strong: rgba(255, 251, 245, 0.98);
  --ink: #1f1c19;
  --muted: #6f665d;
  --line: rgba(35, 29, 24, 0.08);
  --warm: #c75d2c;
  --warm-soft: rgba(199, 93, 44, 0.14);
  --cool: #0e6b73;
  --cool-soft: rgba(14, 107, 115, 0.12);
  --neutral-soft: rgba(29, 28, 25, 0.06);
  --danger: #b42318;
  --danger-soft: rgba(180, 35, 24, 0.09);
  --success: #1f7a4d;
  --success-soft: rgba(31, 122, 77, 0.09);
  --warning: #a85b18;
  --warning-soft: rgba(168, 91, 24, 0.1);
  --shadow: 0 24px 60px rgba(63, 39, 20, 0.12);
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  min-height: 100vh;
  color: var(--ink);
  font-family: "Avenir Next", "PingFang SC", "Microsoft YaHei", sans-serif;
  background:
    radial-gradient(circle at top left, rgba(199, 93, 44, 0.18), transparent 24%),
    radial-gradient(circle at top right, rgba(14, 107, 115, 0.14), transparent 20%),
    linear-gradient(180deg, #faf4ec 0%, var(--bg) 100%);
}

#app {
  min-height: 100vh;
}

.page-shell {
  position: relative;
  width: min(1480px, calc(100vw - 32px));
  margin: 22px auto 52px;
}

.panel {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 28px;
  box-shadow: var(--shadow);
  backdrop-filter: blur(12px);
}

.hero-grid {
  display: grid;
  grid-template-columns: 1.35fr 1fr;
  gap: 18px;
  margin-bottom: 18px;
}

.hero-panel {
  position: relative;
  overflow: hidden;
  padding: 30px;
}

.hero-panel::after {
  content: "";
  position: absolute;
  right: -46px;
  bottom: -64px;
  width: 220px;
  height: 220px;
  border-radius: 999px;
  background: linear-gradient(135deg, rgba(199, 93, 44, 0.16), rgba(14, 107, 115, 0.08));
}

.hero-badge,
.chart-pill,
.level-pill,
.summary-pill,
.metric-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  font-weight: 700;
}

.hero-badge {
  padding: 8px 14px;
  background: var(--warm-soft);
  color: var(--warm);
  font-size: 13px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.hero-panel h1,
.runtime-header h2,
.section-head h2 {
  margin: 0;
}

.hero-panel h1 {
  margin-top: 14px;
  font-size: clamp(32px, 3.8vw, 50px);
  line-height: 1.06;
  letter-spacing: -0.04em;
}

.hero-copy,
.section-head p,
.section-copy,
.insight-note,
.empty-box,
.table-shell table,
.log-error,
.alert-card p {
  color: var(--muted);
}

.hero-copy {
  max-width: 60ch;
  margin: 12px 0 0;
  font-size: 15px;
  line-height: 1.8;
}

.hero-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 22px;
}

.meta-chip {
  display: inline-flex;
  flex-direction: column;
  gap: 4px;
  min-width: 124px;
  padding: 12px 14px;
  border-radius: 18px;
  background: rgba(29, 28, 25, 0.04);
}

.meta-chip span,
.eyebrow,
.stat-label,
.chart-label,
.runtime-card span,
.insight-item span,
th,
.alert-level {
  font-size: 12px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--muted);
}

.meta-chip strong {
  font-size: 14px;
}

.runtime-panel {
  display: grid;
  gap: 18px;
  padding: 24px;
}

.runtime-header,
.section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.runtime-header h2,
.section-head h2 {
  font-size: 22px;
}

.section-copy {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.6;
}

.runtime-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.runtime-card,
.insight-item,
.log-item,
.alert-card {
  border-radius: 18px;
  border: 1px solid var(--line);
  background: rgba(29, 28, 25, 0.03);
}

.runtime-card {
  padding: 16px;
}

.runtime-card strong {
  display: block;
  margin-top: 8px;
  font-size: 22px;
}

.refresh-btn {
  appearance: none;
  border: 0;
  border-radius: 999px;
  background: linear-gradient(135deg, #221d1a, #45342b);
  color: #fff;
  padding: 11px 16px;
  font-weight: 700;
  cursor: pointer;
}

.refresh-btn:disabled {
  cursor: wait;
  opacity: 0.72;
}

.section-block {
  margin-bottom: 18px;
}

.metric-chip {
  padding: 9px 14px;
  background: rgba(29, 28, 25, 0.05);
  color: #4c433c;
  font-size: 12px;
}

.alert-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.alert-card {
  min-height: 156px;
  padding: 18px;
}

.alert-card strong {
  display: block;
  margin-top: 12px;
  font-size: 20px;
  line-height: 1.25;
}

.alert-card p {
  margin: 10px 0 0;
  font-size: 13px;
  line-height: 1.7;
}

.alert-critical {
  background: linear-gradient(180deg, rgba(180, 35, 24, 0.11), rgba(255, 252, 247, 0.94));
}

.alert-warning {
  background: linear-gradient(180deg, rgba(168, 91, 24, 0.1), rgba(255, 252, 247, 0.94));
}

.alert-info {
  background: linear-gradient(180deg, rgba(14, 107, 115, 0.08), rgba(255, 252, 247, 0.94));
}

.alert-healthy {
  background: linear-gradient(180deg, rgba(31, 122, 77, 0.08), rgba(255, 252, 247, 0.94));
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 12px;
}

.stat-card {
  min-height: 150px;
  padding: 18px;
}

.tone-warm {
  background: linear-gradient(160deg, rgba(199, 93, 44, 0.13), rgba(255, 252, 247, 0.88));
}

.tone-cool {
  background: linear-gradient(160deg, rgba(14, 107, 115, 0.12), rgba(255, 252, 247, 0.88));
}

.tone-plain {
  background: rgba(255, 252, 247, 0.88);
}

.stat-value {
  display: block;
  margin: 18px 0 10px;
  font-size: 34px;
  line-height: 1;
  letter-spacing: -0.04em;
}

.stat-note {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--muted);
}

.chart-layout {
  display: grid;
  grid-template-columns: 1.2fr 1.2fr 0.9fr;
  gap: 12px;
}

.chart-card,
.insights-card,
.table-card,
.log-card {
  padding: 18px;
}

.chart-top {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.chart-top strong {
  display: block;
  margin-top: 8px;
  font-size: 28px;
  letter-spacing: -0.03em;
}

.chart-pill {
  padding: 6px 12px;
  font-size: 12px;
}

.chart-pill-warm {
  background: var(--warm-soft);
  color: var(--warm);
}

.chart-pill-cool {
  background: var(--cool-soft);
  color: var(--cool);
}

.chart-pill-neutral {
  background: var(--neutral-soft);
  color: #40352f;
}

.chart-box {
  height: 190px;
  padding: 8px;
  border-radius: 20px;
  border: 1px solid var(--line);
  background: linear-gradient(180deg, rgba(29, 28, 25, 0.03), rgba(29, 28, 25, 0.01));
}

svg {
  display: block;
  width: 100%;
  height: 100%;
}

.guide-line {
  stroke: rgba(29, 28, 25, 0.08);
  stroke-dasharray: 4 6;
}

.line-warm,
.line-cool {
  fill: none;
  stroke-width: 3.5;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.line-warm {
  stroke: var(--warm);
}

.line-cool {
  stroke: var(--cool);
}

.area-warm {
  fill: rgba(199, 93, 44, 0.16);
  stroke: none;
}

.area-cool {
  fill: rgba(14, 107, 115, 0.14);
  stroke: none;
}

.insight-list {
  display: grid;
  gap: 10px;
}

.insight-item {
  padding: 14px;
}

.insight-item strong {
  display: block;
  margin-top: 6px;
  font-size: 20px;
  color: var(--ink);
}

.insight-note {
  margin: 14px 0 0;
  font-size: 12px;
  line-height: 1.6;
}

.dual-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.compact {
  margin-bottom: 12px;
}

.table-shell {
  overflow: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th,
td {
  padding: 12px 10px;
  border-bottom: 1px solid var(--line);
  text-align: left;
  vertical-align: top;
}

th {
  font-weight: 700;
}

td {
  font-size: 14px;
}

.log-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.summary-pill {
  padding: 8px 12px;
  font-size: 12px;
}

.summary-pill-error {
  background: var(--danger-soft);
  color: var(--danger);
}

.summary-pill-warn {
  background: var(--warning-soft);
  color: var(--warning);
}

.log-list {
  display: grid;
  gap: 10px;
  max-height: 480px;
  overflow: auto;
  padding-right: 4px;
}

.log-item {
  padding: 14px;
}

.log-head {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin-bottom: 8px;
  color: var(--muted);
  font-size: 12px;
}

.level-pill {
  padding: 5px 10px;
  font-size: 11px;
}

.pill-error {
  background: rgba(180, 35, 24, 0.16);
  color: var(--danger);
}

.pill-warn {
  background: rgba(174, 90, 25, 0.16);
  color: var(--warning);
}

.pill-info {
  background: rgba(14, 107, 115, 0.12);
  color: var(--cool);
}

.log-title {
  font-size: 15px;
}

.log-error {
  margin: 8px 0 0;
  font-size: 12px;
  line-height: 1.6;
}

.empty-box {
  padding: 30px 18px;
  border-radius: 20px;
  border: 1px dashed var(--line);
  text-align: center;
  background: rgba(29, 28, 25, 0.03);
}

.danger-box {
  border-color: rgba(180, 35, 24, 0.18);
  color: var(--danger);
  background: rgba(180, 35, 24, 0.05);
}

.loading-mask {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(244, 239, 231, 0.7);
  backdrop-filter: blur(6px);
}

.loading-card {
  width: min(360px, calc(100vw - 32px));
  padding: 24px;
  border-radius: 24px;
  border: 1px solid var(--line);
  background: var(--panel-strong);
  box-shadow: var(--shadow);
}

.loading-card strong {
  display: block;
  font-size: 22px;
}

.loading-card p {
  margin: 10px 0 0;
  color: var(--muted);
  line-height: 1.7;
}

@media (max-width: 1260px) {
  .alert-grid,
  .stats-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }

  .chart-layout {
    grid-template-columns: 1fr 1fr;
  }

  .insights-card {
    grid-column: 1 / -1;
  }
}

@media (max-width: 920px) {
  .page-shell {
    width: min(100vw - 20px, 100%);
    margin-top: 10px;
  }

  .hero-grid,
  .dual-layout,
  .chart-layout,
  .stats-grid,
  .runtime-grid,
  .alert-grid {
    grid-template-columns: 1fr;
  }

  .hero-panel,
  .runtime-panel,
  .alert-card,
  .stat-card,
  .chart-card,
  .insights-card,
  .table-card,
  .log-card {
    padding: 16px;
  }
}
</style>
