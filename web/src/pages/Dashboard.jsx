import React, { useState, useEffect, useRef, useCallback } from 'react'
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer
} from 'recharts'
import { Activity, MessageSquare, Clock, XCircle, RefreshCw } from 'lucide-react'
import StatCard from '../components/StatCard.jsx'
import Badge from '../components/Badge.jsx'
import Spinner from '../components/Spinner.jsx'
import {
  getStatus, getMessages, getPrometheusText, parsePrometheusText,
  sumMetric
} from '../api/client.js'

const MAX_HISTORY = 60

function formatUptime(s) {
  if (s == null || isNaN(s)) return '—'
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = Math.floor(s % 60)
  if (d > 0) return `${d}d ${h}h ${m}m`
  if (h > 0) return `${h}h ${m}m ${sec}s`
  if (m > 0) return `${m}m ${sec}s`
  return `${sec}s`
}

function formatTime(date) {
  return date.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function fmtTs(ts) {
  if (!ts) return '—'
  try { return new Date(ts).toLocaleString() } catch { return ts }
}

const CustomTooltip = ({ active, payload, label }) => {
  if (!active || !payload?.length) return null
  return (
    <div style={{
      background: 'var(--bg-elevated)', border: '1px solid var(--border)',
      borderRadius: 'var(--radius-sm)', padding: '8px 12px', fontSize: '0.75rem',
    }}>
      <div style={{ color: 'var(--text-muted)', marginBottom: 4 }}>{label}</div>
      {payload.map(p => (
        <div key={p.dataKey} style={{ color: p.color }}>
          {p.name}: <strong>{p.value}</strong> <span style={{ opacity: 0.7 }}>msg/s</span>
        </div>
      ))}
    </div>
  )
}

export default function Dashboard() {
  const [status, setStatus] = useState(null)
  const [messages, setMessages] = useState([])
  const [metrics, setMetrics] = useState({})
  const [msgHistory, setMsgHistory] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const prevTotalRef = useRef(null)
  const timerRef = useRef(null)
  const mountedRef = useRef(true)

  const fetchAll = useCallback(async () => {
    try {
      const [s, msgs, promText] = await Promise.all([
        getStatus(), getMessages(20), getPrometheusText()
      ])
      if (!mountedRef.current) return

      const m = parsePrometheusText(promText)
      const totalIn = sumMetric(m, 'smsc_messages_in_total')

      const now = new Date()
      setMsgHistory(prev => {
        const rate = prevTotalRef.current !== null
          ? Math.max(0, (totalIn - prevTotalRef.current) / 5) : 0
        prevTotalRef.current = totalIn
        const next = [...prev, { time: formatTime(now), rate: parseFloat(rate.toFixed(2)) }]
        const trimmed = next.length > MAX_HISTORY ? next.slice(next.length - MAX_HISTORY) : next
        return trimmed.map((p, i, arr) => {
          const window = arr.slice(Math.max(0, i - 11), i + 1)
          const avg = window.reduce((s, x) => s + x.rate, 0) / window.length
          return { ...p, avg1m: parseFloat(avg.toFixed(2)) }
        })
      })

      setStatus(s)
      setMessages(Array.isArray(msgs) ? msgs : [])
      setMetrics(m)
      setError(null)
      setLoading(false)
    } catch (err) {
      if (!mountedRef.current) return
      setError(err.message || 'Failed to load data')
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    mountedRef.current = true
    fetchAll()
    timerRef.current = setInterval(fetchAll, 5000)
    return () => { mountedRef.current = false; clearInterval(timerRef.current) }
  }, [fetchAll])

  if (loading) return <div className="loading-center"><Spinner size="lg" /><span>Loading dashboard...</span></div>
  if (error && !status) return (
    <div className="error-state">
      <XCircle size={32} className="error-icon" />
      <div>{error}</div>
      <button className="btn btn-ghost mt-12" onClick={fetchAll}>Retry</button>
    </div>
  )

  const totalIn  = sumMetric(metrics, 'smsc_messages_in_total')
  const totalOut = sumMetric(metrics, 'smsc_messages_out_total')

  const sfQueued = sumMetric(metrics, 'smsc_store_forward_queued')
  const sfFailed = sumMetric(metrics, 'smsc_store_forward_expired')

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">Dashboard</div>
          <div className="page-subtitle">
            VectorCore SMSC — real-time overview
            {status?.uptime && <span style={{ marginLeft: 8, color: 'var(--text-muted)' }}>· up {formatUptime(status.uptime_sec)}</span>}
          </div>
        </div>
        <button className="btn btn-ghost btn-sm" onClick={fetchAll}><RefreshCw size={13} /></button>
      </div>

      <div className="stats-grid">
        <StatCard
          title="Messages In"
          value={totalIn.toLocaleString()}
          icon={<MessageSquare size={18} />}
          color="var(--accent)"
          subtitle="cumulative since start"
        />
        <StatCard
          title="Messages Out"
          value={totalOut.toLocaleString()}
          icon={<Activity size={18} />}
          color="var(--success)"
          subtitle="cumulative since start"
        />
        <StatCard
          title="Queued"
          value={sfQueued}
          icon={<Clock size={18} />}
          color="var(--warning)"
          subtitle={`${sfFailed} failed/expired`}
        />
        <StatCard
          title="Uptime"
          value={formatUptime(status?.uptime_sec)}
          icon={<Clock size={18} />}
          color="var(--warning)"
          subtitle={status?.started_at ? `since ${new Date(status.started_at).toLocaleString()}` : undefined}
        />
      </div>

      <div className="chart-card mb-16">
        <div className="chart-title">
          <span>Inbound Message Rate (msg/s)</span>
          <span style={{ display: 'flex', alignItems: 'center', gap: 12, fontSize: '0.72rem', fontWeight: 400 }}>
            <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
              <span style={{ display: 'inline-block', width: 16, height: 2, background: 'var(--accent)' }} />
              <span style={{ color: 'var(--text-muted)' }}>live</span>
            </span>
            <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
              <span style={{ display: 'inline-block', width: 16, height: 2, borderTop: '2px dashed var(--warning)' }} />
              <span style={{ color: 'var(--text-muted)' }}>1m avg</span>
            </span>
            <Activity size={14} style={{ color: 'var(--text-muted)' }} />
          </span>
        </div>
        <ResponsiveContainer width="100%" height={200}>
          <LineChart data={msgHistory} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border-subtle)" />
            <XAxis dataKey="time" tick={{ fontSize: 10, fill: 'var(--text-muted)' }}
              interval={Math.floor(msgHistory.length / 6) || 1} />
            <YAxis tick={{ fontSize: 10, fill: 'var(--text-muted)' }} width={36} allowDecimals={false} />
            <Tooltip content={<CustomTooltip />} />
            <Line type="monotone" dataKey="rate" name="msg/s" stroke="var(--accent)"
              strokeWidth={1.5} dot={false} isAnimationActive={false} />
            <Line type="monotone" dataKey="avg1m" name="1m avg" stroke="var(--warning)"
              strokeWidth={1.5} strokeDasharray="4 2" dot={false} isAnimationActive={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>

      <div className="section-title">Recent Messages</div>
      {messages.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon"><MessageSquare size={32} /></div>
          <div>No messages yet</div>
        </div>
      ) : (
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>Status</th>
                <th>From</th>
                <th>To</th>
                <th>Ingress</th>
                <th>Egress</th>
                <th>Retries</th>
                <th>Submitted</th>
              </tr>
            </thead>
            <tbody>
              {messages.map(msg => (
                <tr key={msg.id}>
                  <td><Badge state={msg.status} /></td>
                  <td className="mono" style={{ fontSize: '0.8rem' }}>{msg.src_msisdn || '—'}</td>
                  <td className="mono" style={{ fontSize: '0.8rem' }}>{msg.dst_msisdn || '—'}</td>
                  <td><Badge state={msg.origin_iface} /></td>
                  <td><Badge state={msg.egress_iface || undefined} /></td>
                  <td className="mono" style={{ color: msg.retry_count > 0 ? 'var(--warning)' : 'var(--text-muted)' }}>
                    {msg.retry_count}
                  </td>
                  <td className="text-muted" style={{ fontSize: '0.78rem', whiteSpace: 'nowrap' }}>
                    {fmtTs(msg.submitted_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
