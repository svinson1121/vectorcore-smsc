import React, { useState, useCallback } from 'react'
import { Plus, Trash2, Edit3, RefreshCw, XCircle, Users } from 'lucide-react'
import Badge from '../components/Badge.jsx'
import Spinner from '../components/Spinner.jsx'
import Modal from '../components/Modal.jsx'
import { useToast } from '../components/Toast.jsx'
import { usePoller } from '../hooks/usePoller.js'
import { getSubscribers, createSubscriber, updateSubscriber, deleteSubscriber } from '../api/client.js'

const SUB_DEFAULTS = {
  msisdn: '', imsi: '', ims_registered: false, lte_attached: false,
  mme_host: '', mwd_set: false,
}

export default function Subscribers() {
  const toast = useToast()
  const { data, error, loading, refresh } = usePoller(getSubscribers, 8000)
  const [showModal, setShowModal] = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [busy, setBusy] = useState({})
  const [filter, setFilter] = useState('')

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    setBusy(p => ({ ...p, [deleteTarget.id]: true }))
    try {
      await deleteSubscriber(deleteTarget.id)
      toast.success('Subscriber deleted', deleteTarget.msisdn)
      setDeleteTarget(null); refresh()
    } catch (err) { toast.error('Delete failed', err.message) }
    finally { setBusy(p => { const n = {...p}; delete n[deleteTarget.id]; return n }) }
  }, [deleteTarget, toast, refresh])

  const list = Array.isArray(data) ? data : []
  const filtered = filter
    ? list.filter(s =>
        s.msisdn?.includes(filter) ||
        s.imsi?.includes(filter) ||
        s.mme_host?.toLowerCase().includes(filter.toLowerCase())
      )
    : list

  if (loading) return <div className="loading-center"><Spinner size="lg" /><span>Loading subscribers...</span></div>
  if (error && !data) return (
    <div className="error-state">
      <XCircle size={32} className="error-icon" />
      <div>{error}</div>
      <button className="btn btn-ghost mt-12" onClick={refresh}>Retry</button>
    </div>
  )

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">Subscribers</div>
          <div className="page-subtitle">IMS registration table and LTE attach state</div>
        </div>
        <div className="flex gap-8">
          <button className="btn btn-ghost btn-sm" onClick={refresh}><RefreshCw size={13} /></button>
          <button className="btn btn-primary btn-sm" onClick={() => { setEditTarget(null); setShowModal(true) }}>
            <Plus size={12} /> Add Subscriber
          </button>
        </div>
      </div>

      <div className="flex gap-8 mb-16">
        <input
          className="input"
          style={{ maxWidth: 320 }}
          placeholder="Filter by MSISDN, IMSI, or MME host..."
          value={filter}
          onChange={e => setFilter(e.target.value)}
        />
        <span className="text-muted text-sm" style={{ alignSelf: 'center' }}>
          {filtered.length}{filter ? ` of ${list.length}` : ''} subscriber{filtered.length !== 1 ? 's' : ''}
        </span>
      </div>

      {list.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon"><Users size={36} /></div>
          <div style={{ fontWeight: 600, marginBottom: 4 }}>No subscribers</div>
          <div className="text-muted text-sm">Subscribers are populated from SIP REGISTER/NOTIFY and Diameter Sh lookups.</div>
          <button className="btn btn-primary btn-sm mt-12" onClick={() => { setEditTarget(null); setShowModal(true) }}>
            <Plus size={12} /> Add Manually
          </button>
        </div>
      ) : (
        <div className="table-container">
          <table>
            <thead><tr>
              <th>MSISDN</th>
              <th>IMSI</th>
              <th>IMS Reg</th>
              <th>LTE Attached</th>
              <th>MME Host</th>
              <th>MWD Set</th>
              <th>Actions</th>
            </tr></thead>
            <tbody>
              {filtered.map(sub => (
                <tr key={sub.id}>
                  <td style={{ fontWeight: 600, fontFamily: 'var(--font-mono)' }}>{sub.msisdn}</td>
                  <td className="mono text-muted" style={{ fontSize: '0.78rem' }}>{sub.imsi || '—'}</td>
                  <td>
                    <Badge state={sub.ims_registered ? 'true' : 'false'}
                      label={sub.ims_registered ? 'registered' : 'unregistered'} />
                  </td>
                  <td>
                    <Badge state={sub.lte_attached ? 'true' : 'false'}
                      label={sub.lte_attached ? 'attached' : 'detached'} />
                  </td>
                  <td className="mono text-muted" style={{ fontSize: '0.78rem' }}>{sub.mme_host || '—'}</td>
                  <td>
                    <Badge state={sub.mwd_set ? 'true' : 'false'}
                      label={sub.mwd_set ? 'set' : 'clear'} />
                  </td>
                  <td>
                    <div className="flex gap-6">
                      <button className="btn-icon" title="Edit" onClick={() => { setEditTarget(sub); setShowModal(true) }}><Edit3 size={13} /></button>
                      <button className="btn-icon danger" title="Delete" disabled={busy[sub.id]} onClick={() => setDeleteTarget(sub)}><Trash2 size={13} /></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showModal && (
        <SubscriberModal
          initial={editTarget}
          onClose={() => setShowModal(false)}
          onSaved={() => { setShowModal(false); refresh() }}
        />
      )}
      {deleteTarget && (
        <ConfirmDeleteModal
          label={`subscriber ${deleteTarget.msisdn}`}
          onClose={() => setDeleteTarget(null)}
          onConfirm={handleDelete}
          loading={!!busy[deleteTarget?.id]}
        />
      )}
    </div>
  )
}

function SubscriberModal({ initial, onClose, onSaved }) {
  const toast = useToast()
  const [form, setForm] = useState(initial ? { ...SUB_DEFAULTS, ...initial } : { ...SUB_DEFAULTS })
  const [submitting, setSubmitting] = useState(false)
  const set = useCallback((k, v) => setForm(p => ({ ...p, [k]: v })), [])

  const handleSubmit = useCallback(async (e) => {
    e.preventDefault()
    if (!form.msisdn.trim()) { toast.error('Validation', 'MSISDN is required.'); return }
    setSubmitting(true)
    try {
      if (initial) {
        await updateSubscriber(initial.id, form)
        toast.success('Subscriber updated', form.msisdn)
      } else {
        await createSubscriber(form)
        toast.success('Subscriber created', form.msisdn)
      }
      onSaved()
    } catch (err) { toast.error('Save failed', err.message) }
    finally { setSubmitting(false) }
  }, [form, initial, toast, onSaved])

  return (
    <Modal title={initial ? 'Edit Subscriber' : 'Add Subscriber'} onClose={onClose}>
      <form onSubmit={handleSubmit}>
        <div className="modal-body">
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">MSISDN *</label>
              <input className="input mono" value={form.msisdn} onChange={e => set('msisdn', e.target.value)} placeholder="+14155551234" disabled={!!initial} required />
            </div>
            <div className="form-group">
              <label className="form-label">IMSI</label>
              <input className="input mono" value={form.imsi} onChange={e => set('imsi', e.target.value)} placeholder="310260123456789" />
            </div>
          </div>
          <div className="form-group">
            <label className="form-label">MME Host</label>
            <input className="input mono" value={form.mme_host} onChange={e => set('mme_host', e.target.value)} placeholder="mme.epc.example.com" />
          </div>
          <div className="flex gap-16">
            <label className="checkbox-wrap">
              <input type="checkbox" checked={form.ims_registered} onChange={e => set('ims_registered', e.target.checked)} />
              <span>IMS Registered</span>
            </label>
            <label className="checkbox-wrap">
              <input type="checkbox" checked={form.lte_attached} onChange={e => set('lte_attached', e.target.checked)} />
              <span>LTE Attached</span>
            </label>
            <label className="checkbox-wrap">
              <input type="checkbox" checked={form.mwd_set} onChange={e => set('mwd_set', e.target.checked)} />
              <span>MWD Set</span>
            </label>
          </div>
        </div>
        <div className="modal-footer">
          <button type="button" className="btn btn-ghost" onClick={onClose}>Cancel</button>
          <button type="submit" className="btn btn-primary" disabled={submitting}>
            {submitting ? <Spinner size="sm" /> : null}
            {initial ? 'Save Changes' : 'Add Subscriber'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

function ConfirmDeleteModal({ label, onClose, onConfirm, loading }) {
  return (
    <Modal title="Confirm Delete" onClose={onClose}>
      <div className="modal-body">
        <p>Delete {label}?</p>
        <p className="text-muted text-sm" style={{ marginTop: 6 }}>This action cannot be undone.</p>
      </div>
      <div className="modal-footer">
        <button className="btn btn-ghost" onClick={onClose}>Cancel</button>
        <button className="btn btn-danger" onClick={onConfirm} disabled={loading}>
          {loading ? <Spinner size="sm" /> : <Trash2 size={13} />} Delete
        </button>
      </div>
    </Modal>
  )
}
