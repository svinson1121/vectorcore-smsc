import React, { useState, useEffect, useRef, useCallback } from 'react'
import {
  BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, Legend
} from 'recharts'
import { RefreshCw, Activity, XCircle } from 'lucide-react'
import StatCard from '../components/StatCard.jsx'
import Spinner from '../components/Spinner.jsx'
import { getPrometheusText, parsePrometheusText, sumMetric, getMetricSamples } from '../api/client.js'

const INTERFACES = ['sip3gpp', 'sipsimple', 'smpp', 'sgd']
const IFACE_COLORS = {
  sip3gpp:   'var(--accent)',
  sipsimple: 'var(--success)',
  smpp:      'var(--warning)',
  sgd:       'var(--danger)',
}

const MAX_HISTORY = 60

function formatTime(d) {
  return d.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

const ChartTooltip = ({ active, payload, label }) => {
  if (!active || !payload?.length) return null
  return (
    <div style={{
      background: 'var(--bg-elevated)', border: '1px solid var(--border)',
      borderRadius: 'var(--radius-sm)', padding: '8px 12px', fontSize: '0.75rem',
    }}>
      <div style={{ color: 'var(--text-muted)', marginBottom: 4 }}>{label}</div>
      {payload.map(p => (
        <div key={p.dataKey} style={{ color: p.color }}>
          {p.name}: <strong>{typeof p.value === 'number' ? p.value.toFixed(2) : p.value}</strong>
        </div>
      ))}
    </div>
  )
}

export default function Metrics() {
  const [metrics, setMetrics] = useState({})
  const [history, setHistory] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const prevRef = useRef({})
  const timerRef = useRef(null)
  const mountedRef = useRef(true)

  const fetchMetrics = useCallback(async () => {
    try {
      const text = await getPrometheusText()
      if (!mountedRef.current) return
      const m = parsePrometheusText(text)
      const now = new Date()

      setHistory(prev => {
        const point = { time: formatTime(now) }
        // Per-interface rates
        for (const iface of INTERFACES) {
          const keyIn  = `smsc_messages_in_total`
          const keyOut = `smsc_messages_out_total`
          const sampIn  = getMetricSamples(m, keyIn).find(s => s.labels.interface === iface)
          const sampOut = getMetricSamples(m, keyOut).find(s => s.labels.interface === iface)
          const prevIn  = prevRef.current[`in_${iface}`]  ?? null
          const prevOut = prevRef.current[`out_${iface}`] ?? null
          const valIn  = sampIn?.value  ?? 0
          const valOut = sampOut?.value ?? 0
          point[`in_${iface}`]  = prevIn  !== null ? Math.max(0, (valIn  - prevIn)  / 5) : 0
          point[`out_${iface}`] = prevOut !== null ? Math.max(0, (valOut - prevOut) / 5) : 0
          prevRef.current[`in_${iface}`]  = valIn
          prevRef.current[`out_${iface}`] = valOut
        }
        const next = [...prev, point]
        return next.length > MAX_HISTORY ? next.slice(next.length - MAX_HISTORY) : next
      })

      setMetrics(m)
      setError(null)
      setLoading(false)
    } catch (err) {
      if (!mountedRef.current) return
      setError(err.message)
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    mountedRef.current = true
    fetchMetrics()
    timerRef.current = setInterval(fetchMetrics, 5000)
    return () => { mountedRef.current = false; clearInterval(timerRef.current) }
  }, [fetchMetrics])

  if (loading) return <div className="loading-center"><Spinner size="lg" /><span>Loading metrics...</span></div>
  if (error && !Object.keys(metrics).length) return (
    <div className="error-state">
      <XCircle size={32} className="error-icon" />
      <div>{error}</div>
      <button className="btn btn-ghost mt-12" onClick={fetchMetrics}>Retry</button>
    </div>
  )

  const totalIn   = sumMetric(metrics, 'smsc_messages_in_total')
  const totalOut  = sumMetric(metrics, 'smsc_messages_out_total')
  const totalDR   = sumMetric(metrics, 'smsc_delivery_reports_total')
  const sfQueued  = sumMetric(metrics, 'smsc_store_forward_queued')
  const sfRetried = sumMetric(metrics, 'smsc_store_forward_retried_total')
  const sfExpired = sumMetric(metrics, 'smsc_store_forward_expired')
  const smppSess  = sumMetric(metrics, 'smsc_smpp_sessions_connected')
  const sipPeers  = sumMetric(metrics, 'smsc_sip_peers_connected')
  const diamPeers = sumMetric(metrics, 'smsc_diameter_peers_connected')

  // Per-interface totals for bar chart
  const ifaceData = INTERFACES.map(iface => {
    const inSamp  = getMetricSamples(metrics, 'smsc_messages_in_total').find(s => s.labels.interface === iface)
    const outSamp = getMetricSamples(metrics, 'smsc_messages_out_total').find(s => s.labels.interface === iface)
    return {
      interface: iface,
      in:  inSamp?.value  ?? 0,
      out: outSamp?.value ?? 0,
    }
  })

  // Delivery report breakdown
  const drSamples = getMetricSamples(metrics, 'smsc_delivery_reports_total')
  const drByStatus = {}
  for (const s of drSamples) {
    drByStatus[s.labels.status] = (drByStatus[s.labels.status] || 0) + s.value
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">Metrics</div>
          <div className="page-subtitle">Prometheus metrics — live view</div>
        </div>
        <button className="btn btn-ghost btn-sm" onClick={fetchMetrics}><RefreshCw size={13} /></button>
      </div>

      <div className="stats-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))' }}>
        <StatCard title="Messages In"   value={totalIn.toLocaleString()}  color="var(--accent)"   />
        <StatCard title="Messages Out"  value={totalOut.toLocaleString()} color="var(--success)"  />
        <StatCard title="Delivery Rpts" value={totalDR.toLocaleString()}  color="var(--info)"     />
        <StatCard title="Queued"         value={sfQueued}                  color="var(--warning)"  />
        <StatCard title="Retried"        value={sfRetried.toLocaleString()} color="var(--warning)" />
        <StatCard title="Expired"        value={sfExpired.toLocaleString()} color="var(--danger)"  />
        <StatCard title="SMPP Sessions" value={smppSess}  color="var(--warning)" />
        <StatCard title="SIP SIMPLE Peers" value={sipPeers}  color="var(--accent)"  />
        <StatCard title="Diameter Peers" value={diamPeers} color="var(--success)" />
      </div>

      <div className="chart-card mb-16">
        <div className="chart-title">
          <span>Inbound Rate per Interface (msg/s)</span>
          <Activity size={14} style={{ color: 'var(--text-muted)' }} />
        </div>
        <ResponsiveContainer width="100%" height={200}>
          <LineChart data={history} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border-subtle)" />
            <XAxis dataKey="time" tick={{ fontSize: 10, fill: 'var(--text-muted)' }}
              interval={Math.floor(history.length / 6) || 1} />
            <YAxis tick={{ fontSize: 10, fill: 'var(--text-muted)' }} width={36} allowDecimals={false} />
            <Tooltip content={<ChartTooltip />} />
            <Legend iconType="line" wrapperStyle={{ fontSize: '0.72rem' }} />
            {INTERFACES.map(iface => (
              <Line key={iface} type="monotone" dataKey={`in_${iface}`} name={iface}
                stroke={IFACE_COLORS[iface]} strokeWidth={1.5} dot={false} isAnimationActive={false} />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </div>

      <div className="chart-card mb-16">
        <div className="chart-title">
          <span>Outbound Rate per Interface (msg/s)</span>
          <Activity size={14} style={{ color: 'var(--text-muted)' }} />
        </div>
        <ResponsiveContainer width="100%" height={200}>
          <LineChart data={history} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border-subtle)" />
            <XAxis dataKey="time" tick={{ fontSize: 10, fill: 'var(--text-muted)' }}
              interval={Math.floor(history.length / 6) || 1} />
            <YAxis tick={{ fontSize: 10, fill: 'var(--text-muted)' }} width={36} allowDecimals={false} />
            <Tooltip content={<ChartTooltip />} />
            <Legend iconType="line" wrapperStyle={{ fontSize: '0.72rem' }} />
            {INTERFACES.map(iface => (
              <Line key={iface} type="monotone" dataKey={`out_${iface}`} name={iface}
                stroke={IFACE_COLORS[iface]} strokeWidth={1.5} strokeDasharray="4 2"
                dot={false} isAnimationActive={false} />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
        <div className="chart-card" style={{ marginBottom: 0 }}>
          <div className="chart-title">Total Messages by Interface</div>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={ifaceData} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border-subtle)" />
              <XAxis dataKey="interface" tick={{ fontSize: 10, fill: 'var(--text-muted)' }} />
              <YAxis tick={{ fontSize: 10, fill: 'var(--text-muted)' }} width={42} />
              <Tooltip content={<ChartTooltip />} />
              <Legend iconType="square" wrapperStyle={{ fontSize: '0.72rem' }} />
              <Bar dataKey="in"  name="inbound"  fill="var(--accent)"  radius={[2,2,0,0]} />
              <Bar dataKey="out" name="outbound" fill="var(--success)" radius={[2,2,0,0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>

        <div className="chart-card" style={{ marginBottom: 0 }}>
          <div className="chart-title">Delivery Reports by Status</div>
          {Object.keys(drByStatus).length === 0 ? (
            <div className="empty-state" style={{ padding: '40px 20px' }}>No delivery reports yet</div>
          ) : (
            <ResponsiveContainer width="100%" height={200}>
              <BarChart
                data={Object.entries(drByStatus).map(([status, value]) => ({ status, value }))}
                margin={{ top: 4, right: 8, left: 0, bottom: 0 }}
              >
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border-subtle)" />
                <XAxis dataKey="status" tick={{ fontSize: 10, fill: 'var(--text-muted)' }} />
                <YAxis tick={{ fontSize: 10, fill: 'var(--text-muted)' }} width={42} />
                <Tooltip content={<ChartTooltip />} />
                <Bar dataKey="value" name="count" fill="var(--accent)" radius={[2,2,0,0]} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>

    </div>
  )
}
