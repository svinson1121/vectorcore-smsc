import React, { useEffect, useState } from 'react'
import { Clock3, RefreshCw, Search, Trash2, XCircle } from 'lucide-react'
import Spinner from '../components/Spinner.jsx'
import Modal from '../components/Modal.jsx'
import { usePoller } from '../hooks/usePoller.js'
import { deleteQueueMessage, getQueueMessages, getStatus } from '../api/client.js'

export default function OAM() {
  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">OAM</div>
          <div className="page-subtitle">Operations, Administration &amp; Maintenance</div>
        </div>
      </div>
      <SystemTab />
    </div>
  )
}

function SystemTab() {
  const { data, error, loading, refresh } = usePoller(getStatus, 10000)
  const [queueOpen, setQueueOpen] = useState(false)

  if (loading && !data) return <div className="loading-center"><Spinner size="md" /></div>
  if (error && !data) return (
    <div className="error-state">
      <XCircle size={28} className="error-icon" />
      <div>{error}</div>
      <button className="btn btn-ghost mt-12" onClick={refresh}><RefreshCw size={13} /> Retry</button>
    </div>
  )

  const rows = data ? [
    { label: 'Version', value: data.version },
    { label: 'Uptime',  value: data.uptime },
  ] : []

  const queuedNow = (data?.message_counts?.queued || 0) + (data?.message_counts?.dispatched || 0)

  return (
    <div style={{ maxWidth: 480 }}>
      <div className="flex justify-end mb-12">
        <button className="btn btn-ghost btn-sm" onClick={refresh}><RefreshCw size={12} /></button>
      </div>
      <div className="table-container">
        <table>
          <tbody>
            {rows.map(r => (
              <tr key={r.label}>
                <td style={{ fontWeight: 600, width: 160 }}>{r.label}</td>
                <td className="mono">{r.value ?? '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="mt-16">
        <div className="text-muted" style={{ fontSize: 13 }}>
          Queue visibility includes queued and dispatched messages.
        </div>
        <div className="mt-12">
          <button className="btn btn-primary btn-sm" onClick={() => setQueueOpen(true)}>
            <Clock3 size={13} /> View Queue ({queuedNow})
          </button>
        </div>
      </div>
      {queueOpen ? <QueueModal onClose={() => setQueueOpen(false)} onQueueChange={refresh} /> : null}
    </div>
  )
}

function QueueModal({ onClose, onQueueChange }) {
  const [filters, setFilters] = useState({ srcMSISDN: '', dstMSISDN: '', originPeer: '' })
  const [appliedFilters, setAppliedFilters] = useState({ srcMSISDN: '', dstMSISDN: '', originPeer: '' })
  const [data, setData] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [deletingId, setDeletingId] = useState('')

  async function loadQueue(nextFilters = appliedFilters) {
    setLoading(true)
    try {
      const result = await getQueueMessages({ limit: 200, ...nextFilters })
      setData(result || [])
      setError(null)
    } catch (err) {
      setError(err.message || String(err))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadQueue(appliedFilters)
  }, [])

  function onSubmit(e) {
    e.preventDefault()
    const next = {
      srcMSISDN: filters.srcMSISDN.trim(),
      dstMSISDN: filters.dstMSISDN.trim(),
      originPeer: filters.originPeer.trim(),
    }
    setAppliedFilters(next)
    loadQueue(next)
  }

  function clearFilters() {
    const cleared = { srcMSISDN: '', dstMSISDN: '', originPeer: '' }
    setFilters(cleared)
    setAppliedFilters(cleared)
    loadQueue(cleared)
  }

  async function handleDelete(msg) {
    if (!msg?.id) return
    const confirmed = window.confirm(`Delete queued message ${msg.id}?`)
    if (!confirmed) return

    setDeletingId(msg.id)
    try {
      await deleteQueueMessage(msg.id)
      await loadQueue(appliedFilters)
      onQueueChange?.()
    } catch (err) {
      setError(err.message || String(err))
    } finally {
      setDeletingId('')
    }
  }

  return (
    <Modal title="Queued Messages" onClose={onClose} size="lg">
      <div className="modal-body">
        <form className="queue-filters" onSubmit={onSubmit}>
          <label className="form-group">
            <span className="form-label">Source MSISDN</span>
            <input value={filters.srcMSISDN} onChange={(e) => setFilters((s) => ({ ...s, srcMSISDN: e.target.value }))} />
          </label>
          <label className="form-group">
            <span className="form-label">Destination MSISDN</span>
            <input value={filters.dstMSISDN} onChange={(e) => setFilters((s) => ({ ...s, dstMSISDN: e.target.value }))} />
          </label>
          <label className="form-group">
            <span className="form-label">Ingress Peer</span>
            <input value={filters.originPeer} onChange={(e) => setFilters((s) => ({ ...s, originPeer: e.target.value }))} />
          </label>
          <div className="queue-filter-actions">
            <button type="submit" className="btn btn-primary btn-sm"><Search size={13} /> Search</button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => loadQueue(appliedFilters)}><RefreshCw size={13} /> Refresh</button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={clearFilters}>Clear</button>
          </div>
        </form>

        {error ? (
          <div className="error-state" style={{ minHeight: 140 }}>
            <XCircle size={28} className="error-icon" />
            <div>{error}</div>
          </div>
        ) : loading ? (
          <div className="loading-center" style={{ minHeight: 180 }}><Spinner size="md" /></div>
        ) : (
          <div className="table-container queue-table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Status</th>
                  <th>Source</th>
                  <th>Destination</th>
                  <th>Ingress Peer</th>
                  <th>Retry</th>
                  <th>Next Retry</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                {data.length ? data.map((msg) => (
                  <tr key={msg.id}>
                    <td><span className={`queue-status queue-status-${String(msg.status || '').toLowerCase()}`}>{msg.status}</span></td>
                    <td className="mono">{msg.src_msisdn || '—'}</td>
                    <td className="mono">{msg.dst_msisdn || '—'}</td>
                    <td className="mono">{msg.origin_peer || '—'}</td>
                    <td className="mono">{msg.retry_count ?? 0}</td>
                    <td className="mono">{formatTimestamp(msg.next_retry_at)}</td>
                    <td>
                      <button
                        type="button"
                        className="btn btn-ghost btn-sm"
                        onClick={() => handleDelete(msg)}
                        disabled={deletingId === msg.id}
                        title="Delete queued message"
                      >
                        <Trash2 size={13} />
                        {deletingId === msg.id ? 'Deleting...' : 'Delete'}
                      </button>
                    </td>
                  </tr>
                )) : (
                  <tr>
                    <td colSpan="7" className="table-empty">No queued or dispatched messages match the current filters.</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </Modal>
  )
}

function formatTimestamp(value) {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleString()
}
