import React, { useState, useCallback } from 'react'
import { Plus, Trash2, Edit3, RefreshCw, XCircle, Wifi, WifiOff } from 'lucide-react'
import Badge from '../components/Badge.jsx'
import Spinner from '../components/Spinner.jsx'
import Modal from '../components/Modal.jsx'
import { useToast } from '../components/Toast.jsx'
import { usePoller } from '../hooks/usePoller.js'
import {
  getSMPPAccounts, createSMPPAccount, updateSMPPAccount, deleteSMPPAccount,
  getSMPPClients,  createSMPPClient,  updateSMPPClient,  deleteSMPPClient,
  getSIPPeers,     createSIPPeer,     updateSIPPeer,     deleteSIPPeer,
  getDiameterPeers, createDiameterPeer, updateDiameterPeer, deleteDiameterPeer,
  getSGDMMEMappings, createSGDMMEMapping, updateSGDMMEMapping, deleteSGDMMEMapping,
  getRoutingRules,
  getStatusPeers,
} from '../api/client.js'

const smppTONOptions = [
  { value: '0', label: '0 - Unknown' },
  { value: '1', label: '1 - International' },
  { value: '2', label: '2 - National' },
  { value: '3', label: '3 - Network Specific' },
  { value: '4', label: '4 - Subscriber Number' },
  { value: '5', label: '5 - Alphanumeric' },
  { value: '6', label: '6 - Abbreviated' },
]

const smppNPIOptions = [
  { value: '0', label: '0 - Unknown' },
  { value: '1', label: '1 - ISDN / E.164' },
  { value: '3', label: '3 - Data / X.121' },
  { value: '4', label: '4 - Telex / F.69' },
  { value: '6', label: '6 - Land Mobile / E.212' },
  { value: '8', label: '8 - National' },
  { value: '9', label: '9 - Private' },
  { value: '10', label: '10 - ERMES' },
  { value: '13', label: '13 - Internet' },
  { value: '18', label: '18 - WAP Client ID' },
]

function optionalSMPPAddressValue(value) {
  return value === 'auto' ? null : Number(value)
}

function smppTONNPILabel(ton, npi) {
  return `${ton == null ? 'auto' : ton}/${npi == null ? 'auto' : npi}`
}

export default function Peers() {
  const [tab, setTab] = useState('connected')
  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">Peers</div>
          <div className="page-subtitle">SMPP accounts & clients, SIP peers, Diameter peers</div>
        </div>
      </div>
      <div className="tabs">
        {[
          { id: 'connected',     label: 'Connected' },
          { id: 'smpp-accounts', label: 'SMPP Accounts' },
          { id: 'smpp-clients',  label: 'SMPP Clients' },
          { id: 'sip',           label: 'SIP SIMPLE Peers' },
          { id: 'diameter',      label: 'Diameter Peers' },
          { id: 'sgd-mappings',  label: 'S6c to SGd MME Mappings' },
        ].map(t => (
          <button key={t.id} className={`tab-btn${tab === t.id ? ' active' : ''}`} onClick={() => setTab(t.id)}>
            {t.label}
          </button>
        ))}
      </div>
      {tab === 'connected'     && <ConnectedTab />}
      {tab === 'smpp-accounts' && <SMPPAccountsTab />}
      {tab === 'smpp-clients'  && <SMPPClientsTab />}
      {tab === 'sip'           && <SIPPeersTab />}
      {tab === 'diameter'      && <DiameterPeersTab />}
      {tab === 'sgd-mappings'  && <SGDMMEMappingsTab />}
    </div>
  )
}

/* ── Connected Peers ─────────────────────────────────────────────────────────── */
const STATE_COLOR = {
  BOUND:        'var(--green)',
  OPEN:         'var(--green)',
  REGISTERED:   'var(--green)',
  CONNECTING:   '#f59e0b',
  WAIT_CEA:     '#f59e0b',
  DISCONNECTED: 'var(--red)',
  CLOSED:       'var(--red)',
  CLOSING:      'var(--red)',
  EXPIRED:      'var(--text-muted)',
}

function StateDot({ state }) {
  const color = STATE_COLOR[state] || 'var(--text-muted)'
  const isUp = state === 'BOUND' || state === 'OPEN' || state === 'REGISTERED'
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}>
      <span style={{
        width: 8, height: 8, borderRadius: '50%',
        background: color, display: 'inline-block', flexShrink: 0,
      }} />
      <span style={{ color }}>{state}</span>
      {isUp ? <Wifi size={12} style={{ color }} /> : <WifiOff size={12} style={{ color: 'var(--text-muted)' }} />}
    </span>
  )
}

function ConnectedSection({ title, subtitle, peers, emptyText, columns, renderRow }) {
  return (
    <div className="chart-card" style={{ marginBottom: 16 }}>
      <div className="flex justify-between mb-12" style={{ alignItems: 'baseline' }}>
        <div>
          <div style={{ fontWeight: 600, marginBottom: 2 }}>{title}</div>
          {subtitle ? <div className="text-muted text-sm">{subtitle}</div> : null}
        </div>
        <span className="text-muted text-sm">{peers.length}</span>
      </div>

      {peers.length === 0 ? (
        <div className="empty-state" style={{ padding: '28px 20px' }}>
          <WifiOff size={24} style={{ color: 'var(--text-muted)', marginBottom: 8 }} />
          <div className="text-muted text-sm">{emptyText}</div>
        </div>
      ) : (
        <div className="table-container" style={{ marginBottom: 0 }}>
          <table>
            <thead>
              <tr>{columns.map((label) => <th key={label}>{label}</th>)}</tr>
            </thead>
            <tbody>
              {peers.map((peer, index) => renderRow(peer, index))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function ConnectedTab() {
  const { data, error, loading, refresh } = usePoller(getStatusPeers, 5000)
  const peers = Array.isArray(data) ? data : []
  const smppPeers = peers.filter(p => p.type === 'smpp_server' || p.type === 'smpp_client')
  const sipSimplePeers = peers.filter(p => p.type === 'sip_simple')
  const sipImsPeers = peers.filter(p => p.type === 'sip_ims')
  const diameterPeers = (() => {
    const appOrder = ['sh', 'sgd', 's6c']
    const appLabel = { sh: 'Sh', sgd: 'SGd', s6c: 'S6c' }
    const inferApps = (peer) => {
      const apps = new Set()
      const raw = String(peer.application || '').trim()
      if (raw) {
        raw.split(/[\s,]+/).forEach(part => {
          const normalized = part.toLowerCase()
          if (appLabel[normalized]) apps.add(normalized)
        })
      }
      if (peer.type === 'diameter_sh') apps.add('sh')
      if (peer.type === 'diameter_sgd') apps.add('sgd')
      return apps
    }

    const merged = new Map()
    peers
      .filter(p => String(p.type || '').startsWith('diameter_'))
      .forEach((peer) => {
        const key = peer.name || ''
        const current = merged.get(key)
        if (!current) {
          merged.set(key, {
            ...peer,
            applications: inferApps(peer),
          })
          return
        }

        inferApps(peer).forEach(app => current.applications.add(app))
        if (!current.connected_at && peer.connected_at) current.connected_at = peer.connected_at
        if (current.state !== 'OPEN' && peer.state === 'OPEN') current.state = peer.state
      })

    return Array.from(merged.values()).map((peer) => {
      const orderedApps = appOrder.filter(app => peer.applications.has(app)).map(app => appLabel[app])
      return {
        ...peer,
        application: orderedApps.join(' '),
      }
    })
  })()

  if (loading && !data) return <div className="loading-center"><Spinner size="md" /></div>
  if (error && !data) return (
    <div className="error-state">
      <XCircle size={28} className="error-icon" />
      <div>{error}</div>
      <button className="btn btn-ghost mt-12" onClick={refresh}><RefreshCw size={13} /> Retry</button>
    </div>
  )

  return (
    <div>
      <div className="flex justify-between mb-12">
        <span className="text-muted text-sm">{peers.length} peer{peers.length !== 1 ? 's' : ''}</span>
        <button className="btn btn-ghost btn-sm" onClick={refresh}><RefreshCw size={12} /></button>
      </div>

      {peers.length === 0 ? (
        <div className="empty-state">
          <WifiOff size={28} style={{ color: 'var(--text-muted)', marginBottom: 8 }} />
          <div style={{ fontWeight: 600, marginBottom: 4 }}>No active connections</div>
          <div className="text-muted text-sm">SMPP sessions, Diameter peers, and IMS registrations will appear here.</div>
        </div>
      ) : (
        <div>
          {smppPeers.length > 0 && (
            <ConnectedSection
              title="SMPP"
              subtitle="Server-side ESME binds and outbound SMPP client sessions"
              peers={smppPeers}
              emptyText="No SMPP connections are active."
              columns={['Name', 'Mode', 'Transport', 'State', 'System ID', 'Bind Type', 'Remote', 'Since']}
              renderRow={(p, i) => (
                <tr key={`${p.type}-${p.name}-${i}`}>
                  <td style={{ fontWeight: 600 }} className="mono">{p.name || '—'}</td>
                  <td><span className="text-muted" style={{ fontSize: '0.8rem' }}>{p.type === 'smpp_client' ? 'client' : 'server'}</span></td>
                  <td><Badge state={p.transport || 'tcp'} /></td>
                  <td><StateDot state={p.state} /></td>
                  <td className="mono" style={{ fontSize: '0.8rem' }}>{p.system_id || '—'}</td>
                  <td className="mono" style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>{p.bind_type || '—'}</td>
                  <td className="mono" style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>{p.remote_addr || '—'}</td>
                  <td className="mono text-muted" style={{ fontSize: '0.75rem' }}>{p.connected_at ? new Date(p.connected_at).toLocaleTimeString() : '—'}</td>
                </tr>
              )}
            />
          )}

          {sipSimplePeers.length > 0 && (
            <ConnectedSection
              title="SIP SIMPLE"
              subtitle="Inter-site SIMPLE peers"
              peers={sipSimplePeers}
              emptyText="No persistent SIP SIMPLE sessions are tracked. Delivery uses per-message SIP transactions."
              columns={['Name', 'State', 'Remote', 'Since']}
              renderRow={(p, i) => (
                <tr key={`${p.type}-${p.name}-${i}`}>
                  <td style={{ fontWeight: 600 }} className="mono">{p.name || '—'}</td>
                  <td><StateDot state={p.state} /></td>
                  <td className="mono" style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>{p.remote_addr || '—'}</td>
                  <td className="mono text-muted" style={{ fontSize: '0.75rem' }}>{p.connected_at ? new Date(p.connected_at).toLocaleTimeString() : '—'}</td>
                </tr>
              )}
            />
          )}

          {sipImsPeers.length > 0 && (
            <ConnectedSection
              title="IMS (SIP 3GPP)"
              subtitle="IMS registrations and active serving S-CSCF bindings"
              peers={sipImsPeers}
              emptyText="No IMS registrations are active."
              columns={['MSISDN', 'State', 'SIP AOR', 'S-CSCF', 'Contact', 'Since', 'Expiry']}
              renderRow={(p, i) => (
                <tr key={`${p.type}-${p.name}-${i}`}>
                  <td style={{ fontWeight: 600 }} className="mono">{p.name || '—'}</td>
                  <td><StateDot state={p.state} /></td>
                  <td className="mono" style={{ fontSize: '0.8rem' }}>{p.system_id || '—'}</td>
                  <td className="mono" style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>{p.application || '—'}</td>
                  <td className="mono" style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>{p.remote_addr || '—'}</td>
                  <td className="mono text-muted" style={{ fontSize: '0.75rem' }}>{p.connected_at ? new Date(p.connected_at).toLocaleTimeString() : '—'}</td>
                  <td className="mono text-muted" style={{ fontSize: '0.75rem' }}>{p.expiry_at ? new Date(p.expiry_at).toLocaleString() : '—'}</td>
                </tr>
              )}
            />
          )}

          {diameterPeers.length > 0 && (
            <ConnectedSection
              title="Diameter"
              subtitle="peer sessions"
              peers={diameterPeers}
              emptyText="No Diameter peers are active."
              columns={['Name', 'State', 'Application', 'Since']}
              renderRow={(p, i) => (
                <tr key={`${p.type}-${p.name}-${i}`}>
                  <td style={{ fontWeight: 600 }} className="mono">{p.name || '—'}</td>
                  <td><StateDot state={p.state} /></td>
                  <td className="mono" style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>{p.application || '—'}</td>
                  <td className="mono text-muted" style={{ fontSize: '0.75rem' }}>{p.connected_at ? new Date(p.connected_at).toLocaleTimeString() : '—'}</td>
                </tr>
              )}
            />
          )}
        </div>
      )}
    </div>
  )
}

/* ── Shared helpers ─────────────────────────────────────────────────────────── */
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

function TabHeader({ count, noun, onRefresh, onAdd, addLabel }) {
  return (
    <div className="flex justify-between mb-12">
      <span className="text-muted text-sm">{count} {noun}{count !== 1 ? 's' : ''}</span>
      <div className="flex gap-8">
        <button className="btn btn-ghost btn-sm" onClick={onRefresh}><RefreshCw size={12} /></button>
        <button className="btn btn-primary btn-sm" onClick={onAdd}><Plus size={12} /> {addLabel}</button>
      </div>
    </div>
  )
}

function ErrorState({ error, onRefresh }) {
  return (
    <div className="error-state">
      <XCircle size={28} className="error-icon" />
      <div>{error}</div>
      <button className="btn btn-ghost mt-12" onClick={onRefresh}><RefreshCw size={13} /> Retry</button>
    </div>
  )
}

function EnabledStatus({ enabled }) {
  return <Badge state={enabled ? 'enabled' : 'disabled'} />
}

function buildRoutingPeerUsage(rules) {
  const usage = {}
  ;(Array.isArray(rules) ? rules : []).forEach((rule) => {
    if (rule.match_src_peer) {
      usage[rule.match_src_peer] = (usage[rule.match_src_peer] || 0) + 1
    }
    if (rule.egress_peer) {
      usage[rule.egress_peer] = (usage[rule.egress_peer] || 0) + 1
    }
  })
  return usage
}

/* ── SMPP Server Accounts ───────────────────────────────────────────────────── */
const SMPP_ACC_DEFAULTS = {
  name: '', system_id: '', password: '', bind_type: 'transceiver', allowed_ip: '',
  throughput_limit: 0, enabled: true,
}

function SMPPAccountsTab() {
  const toast = useToast()
  const { data, error, loading, refresh } = usePoller(getSMPPAccounts)
  const { data: rulesData } = usePoller(getRoutingRules)
  const [showModal, setShowModal] = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [busy, setBusy] = useState({})

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    setBusy(p => ({ ...p, [deleteTarget.id]: true }))
    try {
      await deleteSMPPAccount(deleteTarget.id)
      toast.success('Account deleted', deleteTarget.system_id)
      setDeleteTarget(null); refresh()
    } catch (err) { toast.error('Delete failed', err.message) }
    finally { setBusy(p => { const n = {...p}; delete n[deleteTarget.id]; return n }) }
  }, [deleteTarget, toast, refresh])

  const list = Array.isArray(data) ? data : []
  const routingPeerUsage = buildRoutingPeerUsage(rulesData)
  if (loading) return <div className="loading-center"><Spinner size="md" /></div>
  if (error && !data) return <ErrorState error={error} onRefresh={refresh} />

  return (
    <div>
      <TabHeader count={list.length} noun="account" onRefresh={refresh}
        onAdd={() => { setEditTarget(null); setShowModal(true) }} addLabel="Add Account" />
      {list.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon" style={{ fontSize: 28 }}>◎</div>
          <div style={{ fontWeight: 600, marginBottom: 4 }}>No SMPP server accounts</div>
          <div className="text-muted text-sm">Add accounts to allow ESMEs to bind to this SMSC.</div>
          <button className="btn btn-primary btn-sm mt-12" onClick={() => { setEditTarget(null); setShowModal(true) }}>
            <Plus size={12} /> Add Account
          </button>
        </div>
      ) : (
        <div className="table-container">
          <table>
            <thead><tr>
              <th>Name</th><th>System ID</th><th>Bind Type</th><th>Allowed IP</th>
              <th>Throughput</th><th>Enabled</th><th>Actions</th>
            </tr></thead>
            <tbody>
              {list.map(acc => (
                <tr key={acc.id}>
                  <td style={{ fontWeight: 600 }}>{acc.name || acc.system_id}</td>
                  <td style={{ fontWeight: 600, fontFamily: 'var(--font-mono)' }}>{acc.system_id}</td>
                  <td><Badge state={acc.bind_type} /></td>
                  <td className="mono text-muted" style={{ fontSize: '0.78rem' }}>{acc.allowed_ip || 'any'}</td>
                  <td className="mono text-muted">{acc.throughput_limit > 0 ? `${acc.throughput_limit} msg/s` : '∞'}</td>
                  <td><EnabledStatus enabled={acc.enabled} /></td>
                  <td>
                    <div className="flex gap-6">
                      <button className="btn-icon" onClick={() => { setEditTarget(acc); setShowModal(true) }}><Edit3 size={13} /></button>
                      <button
                        className="btn-icon danger"
                        disabled={busy[acc.id] || !!routingPeerUsage[acc.system_id]}
                        title={routingPeerUsage[acc.system_id] ? 'Peer is still used by one or more routing rules' : 'Delete account'}
                        onClick={() => setDeleteTarget(acc)}
                      ><Trash2 size={13} /></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {showModal && <SMPPAccountModal initial={editTarget} onClose={() => setShowModal(false)} onSaved={() => { setShowModal(false); refresh() }} />}
      {deleteTarget && <ConfirmDeleteModal label={`SMPP account "${deleteTarget.name || deleteTarget.system_id}"`} onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} loading={!!busy[deleteTarget?.id]} />}
    </div>
  )
}

function SMPPAccountModal({ initial, onClose, onSaved }) {
  const toast = useToast()
  const [form, setForm] = useState(initial ? { ...SMPP_ACC_DEFAULTS, ...initial, password: '' } : { ...SMPP_ACC_DEFAULTS })
  const [submitting, setSubmitting] = useState(false)
  const set = useCallback((k, v) => setForm(p => ({ ...p, [k]: v })), [])

  const handleSubmit = useCallback(async (e) => {
    e.preventDefault()
    if (!form.name.trim()) { toast.error('Validation', 'Name is required.'); return }
    if (!form.system_id.trim()) { toast.error('Validation', 'System ID is required.'); return }
    if (!initial && !form.password) { toast.error('Validation', 'Password is required.'); return }
    setSubmitting(true)
    try {
      // Only send the fields the API expects; extra store fields (id, created_at, etc.)
      // cause Huma to reject the request with a validation error.
      const payload = {
        name: form.name,
        system_id: form.system_id,
        bind_type: form.bind_type,
        allowed_ip: form.allowed_ip || '',
        throughput_limit: Number(form.throughput_limit),
        enabled: form.enabled,
      }
      if (form.password) payload.password = form.password
      if (initial) {
        await updateSMPPAccount(initial.id, payload)
        toast.success('Account updated', form.name)
      } else {
        await createSMPPAccount(payload)
        toast.success('Account created', form.name)
      }
      onSaved()
    } catch (err) { toast.error('Save failed', err.message) }
    finally { setSubmitting(false) }
  }, [form, initial, toast, onSaved])

  return (
    <Modal title={initial ? 'Edit SMPP Account' : 'Add SMPP Account'} onClose={onClose}>
      <form onSubmit={handleSubmit}>
        <div className="modal-body">
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Name *</label>
              <input className="input" value={form.name} onChange={e => set('name', e.target.value)} required />
            </div>
            <div className="form-group">
              <label className="form-label">System ID *</label>
              <input className="input mono" value={form.system_id} onChange={e => set('system_id', e.target.value)} disabled={!!initial} required />
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">{initial ? 'New Password' : 'Password *'}</label>
              <input className="input" type="password" value={form.password} onChange={e => set('password', e.target.value)} placeholder={initial ? 'Leave blank to keep current' : ''} />
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Bind Type</label>
              <select className="select" value={form.bind_type} onChange={e => set('bind_type', e.target.value)}>
                <option value="transceiver">transceiver</option>
                <option value="transmitter">transmitter</option>
                <option value="receiver">receiver</option>
              </select>
            </div>
            <div className="form-group">
              <label className="form-label">Allowed IP</label>
              <input className="input mono" value={form.allowed_ip} onChange={e => set('allowed_ip', e.target.value)} placeholder="any" />
              <span className="form-hint">Leave blank to allow any source IP</span>
            </div>
          </div>
          <div className="form-group">
            <label className="form-label">Throughput Limit (msg/s)</label>
            <input className="input mono" type="number" min={0} value={form.throughput_limit} onChange={e => set('throughput_limit', e.target.value)} />
            <span className="form-hint">0 = unlimited</span>
          </div>
          <label className="checkbox-wrap">
            <input type="checkbox" checked={form.enabled} onChange={e => set('enabled', e.target.checked)} />
            <span>Enabled</span>
          </label>
        </div>
        <div className="modal-footer">
          <button type="button" className="btn btn-ghost" onClick={onClose}>Cancel</button>
          <button type="submit" className="btn btn-primary" disabled={submitting}>
            {submitting ? <Spinner size="sm" /> : null}
            {initial ? 'Save Changes' : 'Add Account'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

/* ── SMPP Clients ───────────────────────────────────────────────────────────── */
const SMPP_CLI_DEFAULTS = {
  name: '', host: '', port: 2775, system_id: '', password: '',
  transport: 'tcp', verify_server_cert: false,
  bind_type: 'transceiver', reconnect_interval: '10s', throughput_limit: 0, enabled: true,
  source_addr_ton: 'auto', source_addr_npi: 'auto',
  dest_addr_ton: 'auto', dest_addr_npi: 'auto',
}

function normalizeReconnectInterval(value) {
  if (typeof value === 'string' && value.trim()) return value
  if (typeof value === 'number' && Number.isFinite(value) && value >= 0) return `${value}ns`
  return '10s'
}

function SMPPClientsTab() {
  const toast = useToast()
  const { data, error, loading, refresh } = usePoller(getSMPPClients)
  const { data: rulesData } = usePoller(getRoutingRules)
  const [showModal, setShowModal] = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [busy, setBusy] = useState({})

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    setBusy(p => ({ ...p, [deleteTarget.id]: true }))
    try {
      await deleteSMPPClient(deleteTarget.id)
      toast.success('Client deleted', deleteTarget.name)
      setDeleteTarget(null); refresh()
    } catch (err) { toast.error('Delete failed', err.message) }
    finally { setBusy(p => { const n = {...p}; delete n[deleteTarget.id]; return n }) }
  }, [deleteTarget, toast, refresh])

  const list = Array.isArray(data) ? data : []
  const routingPeerUsage = buildRoutingPeerUsage(rulesData)
  if (loading) return <div className="loading-center"><Spinner size="md" /></div>
  if (error && !data) return <ErrorState error={error} onRefresh={refresh} />

  return (
    <div>
      <TabHeader count={list.length} noun="client" onRefresh={refresh}
        onAdd={() => { setEditTarget(null); setShowModal(true) }} addLabel="Add Client" />
      {list.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon" style={{ fontSize: 28 }}>◎</div>
          <div style={{ fontWeight: 600, marginBottom: 4 }}>No SMPP outbound clients</div>
          <div className="text-muted text-sm">Add clients to connect to downstream SMSCs.</div>
          <button className="btn btn-primary btn-sm mt-12" onClick={() => { setEditTarget(null); setShowModal(true) }}>
            <Plus size={12} /> Add Client
          </button>
        </div>
      ) : (
        <div className="table-container">
          <table>
	            <thead><tr>
	              <th>Name</th><th>Host</th><th>Transport</th><th>Verify</th><th>System ID</th>
	              <th>Bind Type</th><th>TON/NPI</th><th>Reconnect</th><th>Enabled</th><th>Actions</th>
	            </tr></thead>
            <tbody>
              {list.map(c => (
                <tr key={c.id}>
                  <td style={{ fontWeight: 600, fontFamily: 'var(--font-mono)' }}>{c.name}</td>
                  <td className="mono text-muted" style={{ fontSize: '0.78rem' }}>{c.host}:{c.port}</td>
                  <td><Badge state={c.transport || 'tcp'} /></td>
                  <td><Badge state={String(!!c.verify_server_cert)} /></td>
	                  <td className="mono" style={{ fontSize: '0.8rem' }}>{c.system_id}</td>
	                  <td><Badge state={c.bind_type} /></td>
	                  <td className="mono text-muted" style={{ fontSize: '0.75rem' }}>
	                    src {smppTONNPILabel(c.source_addr_ton, c.source_addr_npi)} / dst {smppTONNPILabel(c.dest_addr_ton, c.dest_addr_npi)}
	                  </td>
	                  <td className="mono text-muted">{c.reconnect_interval || '10s'}</td>
                  <td><EnabledStatus enabled={c.enabled} /></td>
                  <td>
                    <div className="flex gap-6">
                      <button className="btn-icon" onClick={() => { setEditTarget(c); setShowModal(true) }}><Edit3 size={13} /></button>
                      <button
                        className="btn-icon danger"
                        disabled={busy[c.id] || !!routingPeerUsage[c.name]}
                        title={routingPeerUsage[c.name] ? 'Peer is still used by one or more routing rules' : 'Delete client'}
                        onClick={() => setDeleteTarget(c)}
                      ><Trash2 size={13} /></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {showModal && <SMPPClientModal initial={editTarget} onClose={() => setShowModal(false)} onSaved={() => { setShowModal(false); refresh() }} />}
      {deleteTarget && <ConfirmDeleteModal label={`SMPP client "${deleteTarget.name}"`} onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} loading={!!busy[deleteTarget?.id]} />}
    </div>
  )
}

function SMPPClientModal({ initial, onClose, onSaved }) {
  const toast = useToast()
  const [form, setForm] = useState(() => {
    const base = initial ? { ...SMPP_CLI_DEFAULTS, ...initial } : { ...SMPP_CLI_DEFAULTS }
	    base.reconnect_interval = normalizeReconnectInterval(base.reconnect_interval)
	    base.transport = base.transport || 'tcp'
	    base.verify_server_cert = !!base.verify_server_cert
	    base.source_addr_ton = base.source_addr_ton == null ? 'auto' : String(base.source_addr_ton)
	    base.source_addr_npi = base.source_addr_npi == null ? 'auto' : String(base.source_addr_npi)
	    base.dest_addr_ton = base.dest_addr_ton == null ? 'auto' : String(base.dest_addr_ton)
	    base.dest_addr_npi = base.dest_addr_npi == null ? 'auto' : String(base.dest_addr_npi)
	    return base
	  })
  const [submitting, setSubmitting] = useState(false)
  const set = useCallback((k, v) => setForm(p => ({ ...p, [k]: v })), [])
  const setTransport = useCallback((nextTransport) => {
    setForm((prev) => {
      const currentTransport = prev.transport || 'tcp'
      const currentPort = Number(prev.port)
      const nextPort = currentPort === (currentTransport === 'tls' ? 3550 : 2775)
        ? (nextTransport === 'tls' ? 3550 : 2775)
        : currentPort
      return { ...prev, transport: nextTransport, port: nextPort }
    })
  }, [])

  const handleSubmit = useCallback(async (e) => {
    e.preventDefault()
    if (!form.name.trim()) { toast.error('Validation', 'Name is required.'); return }
    if (!form.host.trim()) { toast.error('Validation', 'Host is required.'); return }
    setSubmitting(true)
    try {
      const payload = {
        name: form.name,
        host: form.host,
        port: Number(form.port),
        transport: form.transport || 'tcp',
        verify_server_cert: !!form.verify_server_cert,
        system_id: form.system_id,
	        bind_type: form.bind_type,
	        reconnect_interval: normalizeReconnectInterval(form.reconnect_interval),
	        throughput_limit: Number(form.throughput_limit),
	        source_addr_ton: optionalSMPPAddressValue(form.source_addr_ton),
	        source_addr_npi: optionalSMPPAddressValue(form.source_addr_npi),
	        dest_addr_ton: optionalSMPPAddressValue(form.dest_addr_ton),
	        dest_addr_npi: optionalSMPPAddressValue(form.dest_addr_npi),
	        enabled: form.enabled,
	      }
      if (!initial || form.password) {
        payload.password = form.password
      }
      if (initial) {
        await updateSMPPClient(initial.id, payload)
        toast.success('Client updated', form.name)
      } else {
        await createSMPPClient(payload)
        toast.success('Client created', form.name)
      }
      onSaved()
    } catch (err) { toast.error('Save failed', err.message) }
    finally { setSubmitting(false) }
  }, [form, initial, toast, onSaved])

  return (
    <Modal title={initial ? 'Edit SMPP Client' : 'Add SMPP Client'} onClose={onClose}>
      <form onSubmit={handleSubmit}>
        <div className="modal-body">
          <div className="form-group">
            <label className="form-label">Name *</label>
            <input className="input" value={form.name} onChange={e => set('name', e.target.value)} disabled={!!initial} required />
            <span className="form-hint">Used in routing rules to identify this peer</span>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Host *</label>
              <input className="input mono" value={form.host} onChange={e => set('host', e.target.value)} required />
            </div>
            <div className="form-group">
              <label className="form-label">Port</label>
              <input className="input mono" type="number" value={form.port} onChange={e => set('port', e.target.value)} />
              <span className="form-hint">{(form.transport || 'tcp') === 'tls' ? 'Default TLS port: 3550' : 'Default TCP port: 2775'}</span>
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">System ID *</label>
              <input className="input mono" value={form.system_id} onChange={e => set('system_id', e.target.value)} required />
            </div>
            <div className="form-group">
              <label className="form-label">Password *</label>
              <input className="input" type="password" value={form.password} onChange={e => set('password', e.target.value)} required={!initial} />
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Transport</label>
              <select className="select" value={form.transport || 'tcp'} onChange={e => setTransport(e.target.value)}>
                <option value="tcp">tcp</option>
                <option value="tls">tls</option>
              </select>
            </div>
            <div className="form-group">
              <label className="form-label">Bind Type</label>
              <select className="select" value={form.bind_type} onChange={e => set('bind_type', e.target.value)}>
                <option value="transceiver">transceiver</option>
                <option value="transmitter">transmitter</option>
                <option value="receiver">receiver</option>
              </select>
            </div>
          </div>
	          <div className="form-row">
	            <div className="form-group">
	              <label className="form-label">Reconnect Interval</label>
	              <input className="input mono" value={form.reconnect_interval} onChange={e => set('reconnect_interval', e.target.value)} placeholder="10s" />
	            </div>
            <div className="form-group">
              <label className="form-label">TLS Verification</label>
              <label className="checkbox-wrap" style={{ minHeight: 36 }}>
                <input type="checkbox" checked={!!form.verify_server_cert} onChange={e => set('verify_server_cert', e.target.checked)} />
                <span>Verify server certificate</span>
	              </label>
	            </div>
	          </div>
	          <div className="form-row">
	            <div className="form-group">
	              <label className="form-label">Source TON</label>
	              <select className="select" value={form.source_addr_ton} onChange={e => set('source_addr_ton', e.target.value)}>
	                <option value="auto">Auto</option>
	                {smppTONOptions.map(option => (
	                  <option key={option.value} value={option.value}>{option.label}</option>
	                ))}
	              </select>
	            </div>
	            <div className="form-group">
	              <label className="form-label">Source NPI</label>
	              <select className="select" value={form.source_addr_npi} onChange={e => set('source_addr_npi', e.target.value)}>
	                <option value="auto">Auto</option>
	                {smppNPIOptions.map(option => (
	                  <option key={option.value} value={option.value}>{option.label}</option>
	                ))}
	              </select>
	            </div>
	          </div>
	          <div className="form-row">
	            <div className="form-group">
	              <label className="form-label">Destination TON</label>
	              <select className="select" value={form.dest_addr_ton} onChange={e => set('dest_addr_ton', e.target.value)}>
	                <option value="auto">Auto</option>
	                {smppTONOptions.map(option => (
	                  <option key={option.value} value={option.value}>{option.label}</option>
	                ))}
	              </select>
	            </div>
	            <div className="form-group">
	              <label className="form-label">Destination NPI</label>
	              <select className="select" value={form.dest_addr_npi} onChange={e => set('dest_addr_npi', e.target.value)}>
	                <option value="auto">Auto</option>
	                {smppNPIOptions.map(option => (
	                  <option key={option.value} value={option.value}>{option.label}</option>
	                ))}
	              </select>
	            </div>
	          </div>
	          <div className="form-group">
	            <label className="form-label">Throughput Limit (msg/s)</label>
            <input className="input mono" type="number" min={0} value={form.throughput_limit} onChange={e => set('throughput_limit', e.target.value)} />
            <span className="form-hint">0 = unlimited</span>
          </div>
          <label className="checkbox-wrap">
            <input type="checkbox" checked={form.enabled} onChange={e => set('enabled', e.target.checked)} />
            <span>Enabled</span>
          </label>
        </div>
        <div className="modal-footer">
          <button type="button" className="btn btn-ghost" onClick={onClose}>Cancel</button>
          <button type="submit" className="btn btn-primary" disabled={submitting}>
            {submitting ? <Spinner size="sm" /> : null}
            {initial ? 'Save Changes' : 'Add Client'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

/* ── SIP Peers ──────────────────────────────────────────────────────────────── */
const SIP_DEFAULTS = {
  name: '', address: '', port: 5060, transport: 'udp',
  domain: '', auth_user: '', auth_pass: '', enabled: true,
}

function buildSIPPeerPayload(peer) {
  return {
    name: peer.name ?? '',
    address: peer.address ?? '',
    port: Number(peer.port),
    transport: peer.transport ?? 'udp',
    domain: peer.domain ?? '',
    auth_user: peer.auth_user ?? '',
    auth_pass: peer.auth_pass ?? '',
    enabled: peer.enabled ?? true,
  }
}

function SIPPeersTab() {
  const toast = useToast()
  const { data, error, loading, refresh } = usePoller(getSIPPeers)
  const { data: rulesData } = usePoller(getRoutingRules)
  const [showModal, setShowModal] = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [busy, setBusy] = useState({})

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    setBusy(p => ({ ...p, [deleteTarget.id]: true }))
    try {
      await deleteSIPPeer(deleteTarget.id)
      toast.success('SIP peer deleted', deleteTarget.name)
      setDeleteTarget(null); refresh()
    } catch (err) { toast.error('Delete failed', err.message) }
    finally { setBusy(p => { const n = {...p}; delete n[deleteTarget.id]; return n }) }
  }, [deleteTarget, toast, refresh])

  const list = Array.isArray(data) ? data : []
  const routingPeerUsage = buildRoutingPeerUsage(rulesData)
  if (loading) return <div className="loading-center"><Spinner size="md" /></div>
  if (error && !data) return <ErrorState error={error} onRefresh={refresh} />

  return (
    <div>
      <TabHeader count={list.length} noun="peer" onRefresh={refresh}
        onAdd={() => { setEditTarget(null); setShowModal(true) }} addLabel="Add Peer" />
      {list.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon" style={{ fontSize: 28 }}>◎</div>
          <div style={{ fontWeight: 600, marginBottom: 4 }}>No SIP SIMPLE peers</div>
          <div className="text-muted text-sm">Add peers for inter-site SIP SIMPLE federation.</div>
          <button className="btn btn-primary btn-sm mt-12" onClick={() => { setEditTarget(null); setShowModal(true) }}>
            <Plus size={12} /> Add Peer
          </button>
        </div>
      ) : (
        <div className="table-container">
          <table>
            <thead><tr>
              <th>Name</th><th>Address</th><th>Domain</th>
              <th>Transport</th><th>Enabled</th><th>Actions</th>
            </tr></thead>
            <tbody>
              {list.map(p => (
                <tr key={p.id}>
                  <td style={{ fontWeight: 600 }}>{p.name}</td>
                  <td className="mono text-muted" style={{ fontSize: '0.78rem' }}>{p.address}:{p.port}</td>
                  <td className="mono" style={{ fontSize: '0.8rem' }}>{p.domain}</td>
                  <td><Badge state={p.transport} /></td>
                  <td><EnabledStatus enabled={p.enabled} /></td>
                  <td>
                    <div className="flex gap-6">
                      <button className="btn-icon" onClick={() => { setEditTarget(p); setShowModal(true) }}><Edit3 size={13} /></button>
                      <button
                        className="btn-icon danger"
                        disabled={busy[p.id] || !!routingPeerUsage[p.name]}
                        title={routingPeerUsage[p.name] ? 'Peer is still used by one or more routing rules' : 'Delete peer'}
                        onClick={() => setDeleteTarget(p)}
                      ><Trash2 size={13} /></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {showModal && <SIPPeerModal initial={editTarget} onClose={() => setShowModal(false)} onSaved={() => { setShowModal(false); refresh() }} />}
      {deleteTarget && <ConfirmDeleteModal label={`SIP peer "${deleteTarget.name}"`} onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} loading={!!busy[deleteTarget?.id]} />}
    </div>
  )
}

function SIPPeerModal({ initial, onClose, onSaved }) {
  const toast = useToast()
  const [form, setForm] = useState(initial ? buildSIPPeerPayload(initial) : { ...SIP_DEFAULTS })
  const [submitting, setSubmitting] = useState(false)
  const set = useCallback((k, v) => setForm(p => ({ ...p, [k]: v })), [])

  const handleSubmit = useCallback(async (e) => {
    e.preventDefault()
    if (!form.name.trim()) { toast.error('Validation', 'Name is required.'); return }
    if (!form.address.trim()) { toast.error('Validation', 'Address is required.'); return }
    if (!form.domain.trim()) { toast.error('Validation', 'Domain is required.'); return }
    setSubmitting(true)
    try {
      const payload = buildSIPPeerPayload(form)
      if (initial) {
        await updateSIPPeer(initial.id, payload)
        toast.success('SIP peer updated', form.name)
      } else {
        await createSIPPeer(payload)
        toast.success('SIP peer created', form.name)
      }
      onSaved()
    } catch (err) { toast.error('Save failed', err.message) }
    finally { setSubmitting(false) }
  }, [form, initial, toast, onSaved])

  return (
    <Modal title={initial ? 'Edit SIP Peer' : 'Add SIP Peer'} onClose={onClose}>
      <form onSubmit={handleSubmit}>
        <div className="modal-body">
          <div className="form-group">
            <label className="form-label">Name *</label>
            <input className="input" value={form.name} onChange={e => set('name', e.target.value)} disabled={!!initial} required />
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Address *</label>
              <input className="input mono" value={form.address} onChange={e => set('address', e.target.value)} required />
            </div>
            <div className="form-group">
              <label className="form-label">Port</label>
              <input className="input mono" type="number" value={form.port} onChange={e => set('port', e.target.value)} />
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Domain *</label>
              <input className="input mono" value={form.domain} onChange={e => set('domain', e.target.value)} required />
            </div>
            <div className="form-group">
              <label className="form-label">Transport</label>
              <select className="select" value={form.transport} onChange={e => set('transport', e.target.value)}>
                <option value="udp">udp</option>
                <option value="tcp">tcp</option>
                <option value="tls">tls</option>
              </select>
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Auth User</label>
              <input className="input" value={form.auth_user} onChange={e => set('auth_user', e.target.value)} />
            </div>
            <div className="form-group">
              <label className="form-label">Auth Password</label>
              <input className="input" type="password" value={form.auth_pass} onChange={e => set('auth_pass', e.target.value)} />
            </div>
          </div>
          <label className="checkbox-wrap">
            <input type="checkbox" checked={form.enabled} onChange={e => set('enabled', e.target.checked)} />
            <span>Enabled</span>
          </label>
        </div>
        <div className="modal-footer">
          <button type="button" className="btn btn-ghost" onClick={onClose}>Cancel</button>
          <button type="submit" className="btn btn-primary" disabled={submitting}>
            {submitting ? <Spinner size="sm" /> : null}
            {initial ? 'Save Changes' : 'Add Peer'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

/* ── Diameter Peers ─────────────────────────────────────────────────────────── */
const DIAM_APP_OPTIONS = [
  { value: 'sgd', label: 'sgd — IP-SM-GW (TS 29.338)' },
  { value: 'sh',  label: 'sh — HSS User Data (TS 29.328)' },
  { value: 's6c', label: 's6c — HSS SM (TS 29.338)' },
]
const DIAM_DEFAULTS = {
  name: '', host: '', realm: '', port: 3868,
  transport: 'tcp', applications: ['sgd'], enabled: true,
}

function DiameterPeersTab() {
  const toast = useToast()
  const { data, error, loading, refresh } = usePoller(getDiameterPeers)
  const { data: rulesData } = usePoller(getRoutingRules)
  const [showModal, setShowModal] = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [busy, setBusy] = useState({})

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    setBusy(p => ({ ...p, [deleteTarget.id]: true }))
    try {
      await deleteDiameterPeer(deleteTarget.id)
      toast.success('Diameter peer deleted', deleteTarget.name)
      setDeleteTarget(null); refresh()
    } catch (err) { toast.error('Delete failed', err.message) }
    finally { setBusy(p => { const n = {...p}; delete n[deleteTarget.id]; return n }) }
  }, [deleteTarget, toast, refresh])

  const list = Array.isArray(data) ? data : []
  const routingPeerUsage = buildRoutingPeerUsage(rulesData)
  if (loading) return <div className="loading-center"><Spinner size="md" /></div>
  if (error && !data) return <ErrorState error={error} onRefresh={refresh} />

  return (
    <div>
      <TabHeader count={list.length} noun="peer" onRefresh={refresh}
        onAdd={() => { setEditTarget(null); setShowModal(true) }} addLabel="Add Peer" />
      {list.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon" style={{ fontSize: 28 }}>◎</div>
          <div style={{ fontWeight: 600, marginBottom: 4 }}>No Diameter peers</div>
          <div className="text-muted text-sm">Add SGd peers for IP-SM-GW, Sh peers for HSS lookup.</div>
          <button className="btn btn-primary btn-sm mt-12" onClick={() => { setEditTarget(null); setShowModal(true) }}>
            <Plus size={12} /> Add Peer
          </button>
        </div>
      ) : (
        <div className="table-container">
          <table>
            <thead><tr>
              <th>Name</th><th>Host</th><th>Realm</th>
              <th>App</th><th>Transport</th><th>Enabled</th><th>Actions</th>
            </tr></thead>
            <tbody>
              {list.map(p => (
                <tr key={p.id}>
                  <td style={{ fontWeight: 600 }}>{p.name}</td>
                  <td className="mono text-muted" style={{ fontSize: '0.78rem' }}>{p.host}:{p.port}</td>
                  <td className="mono" style={{ fontSize: '0.78rem' }}>{p.realm}</td>
                  <td>{(p.applications || []).map(a => <Badge key={a} state={a} style={{ marginRight: 3 }} />)}</td>
                  <td><Badge state={p.transport} /></td>
                  <td><EnabledStatus enabled={p.enabled} /></td>
                  <td>
                    <div className="flex gap-6">
                      <button className="btn-icon" onClick={() => { setEditTarget(p); setShowModal(true) }}><Edit3 size={13} /></button>
                      <button
                        className="btn-icon danger"
                        disabled={busy[p.id] || !!routingPeerUsage[p.name]}
                        title={routingPeerUsage[p.name] ? 'Peer is still used by one or more routing rules' : 'Delete peer'}
                        onClick={() => setDeleteTarget(p)}
                      ><Trash2 size={13} /></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {showModal && <DiameterPeerModal initial={editTarget} onClose={() => setShowModal(false)} onSaved={() => { setShowModal(false); refresh() }} />}
      {deleteTarget && <ConfirmDeleteModal label={`Diameter peer "${deleteTarget.name}"`} onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} loading={!!busy[deleteTarget?.id]} />}
    </div>
  )
}

/* ── SGd MME Mappings ────────────────────────────────────────────────────────── */

const SGD_MAPPING_DEFAULTS = { s6c_result: '', sgd_host: '', enabled: true }

function SGDMMEMappingsTab() {
  const toast = useToast()
  const { data, error, loading, refresh } = usePoller(getSGDMMEMappings)
  const [showModal, setShowModal] = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [busy, setBusy] = useState({})

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    setBusy(p => ({ ...p, [deleteTarget.id]: true }))
    try {
      await deleteSGDMMEMapping(deleteTarget.id)
      toast.success('Mapping deleted', deleteTarget.s6c_result)
      setDeleteTarget(null); refresh()
    } catch (err) { toast.error('Delete failed', err.message) }
    finally { setBusy(p => { const n = {...p}; delete n[deleteTarget.id]; return n }) }
  }, [deleteTarget, toast, refresh])

  const list = Array.isArray(data) ? data : []
  if (loading) return <div className="loading-center"><Spinner size="md" /></div>
  if (error && !data) return <ErrorState error={error} onRefresh={refresh} />

  return (
    <div>
      <TabHeader count={list.length} noun="mapping" onRefresh={refresh}
        onAdd={() => { setEditTarget(null); setShowModal(true) }} addLabel="Add Mapping" />
      <div className="text-muted text-sm" style={{ marginBottom: 12 }}>
        Translates the MME hostname returned by S6c (S6a FQDN) to the correct SGd FQDN for Diameter delivery.
        If no enabled match is found the original S6c hostname is used unchanged.
      </div>
      {list.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon" style={{ fontSize: 28 }}>↔</div>
          <div style={{ fontWeight: 600, marginBottom: 4 }}>No MME mappings configured</div>
          <div className="text-muted text-sm">Add a mapping when the S6a and SGd FQDNs of an MME differ.</div>
          <button className="btn btn-primary btn-sm mt-12" onClick={() => { setEditTarget(null); setShowModal(true) }}>
            <Plus size={12} /> Add Mapping
          </button>
        </div>
      ) : (
        <div className="table-container">
          <table>
            <thead><tr>
              <th>S6c Result (S6a FQDN)</th>
              <th>SGd Host (SGd FQDN)</th>
              <th>Enabled</th>
              <th>Actions</th>
            </tr></thead>
            <tbody>
              {list.map(m => (
                <tr key={m.id}>
                  <td className="mono" style={{ fontSize: '0.82rem' }}>{m.s6c_result}</td>
                  <td className="mono" style={{ fontSize: '0.82rem', color: 'var(--text-muted)' }}>{m.sgd_host}</td>
                  <td><EnabledStatus enabled={m.enabled} /></td>
                  <td>
                    <div className="flex gap-6">
                      <button className="btn-icon" onClick={() => { setEditTarget(m); setShowModal(true) }}><Edit3 size={13} /></button>
                      <button
                        className="btn-icon danger"
                        disabled={!!busy[m.id]}
                        title="Delete mapping"
                        onClick={() => setDeleteTarget(m)}
                      ><Trash2 size={13} /></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {showModal && <SGDMMEMappingModal initial={editTarget} onClose={() => setShowModal(false)} onSaved={() => { setShowModal(false); refresh() }} />}
      {deleteTarget && <ConfirmDeleteModal label={`mapping "${deleteTarget.s6c_result}"`} onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} loading={!!busy[deleteTarget?.id]} />}
    </div>
  )
}

function SGDMMEMappingModal({ initial, onClose, onSaved }) {
  const toast = useToast()
  const [form, setForm] = useState(() => initial ? { ...SGD_MAPPING_DEFAULTS, ...initial } : { ...SGD_MAPPING_DEFAULTS })
  const [submitting, setSubmitting] = useState(false)
  const set = useCallback((k, v) => setForm(p => ({ ...p, [k]: v })), [])

  const handleSubmit = useCallback(async (e) => {
    e.preventDefault()
    if (!form.s6c_result.trim()) { toast.error('Validation', 'S6c Result is required.'); return }
    if (!form.sgd_host.trim())   { toast.error('Validation', 'SGd Host is required.');   return }
    setSubmitting(true)
    try {
      const payload = { s6c_result: form.s6c_result.trim(), sgd_host: form.sgd_host.trim(), enabled: form.enabled }
      if (initial) {
        await updateSGDMMEMapping(initial.id, payload)
        toast.success('Mapping updated', form.s6c_result)
      } else {
        await createSGDMMEMapping(payload)
        toast.success('Mapping created', form.s6c_result)
      }
      onSaved()
    } catch (err) { toast.error('Save failed', err.message) }
    finally { setSubmitting(false) }
  }, [form, initial, toast, onSaved])

  return (
    <Modal title={initial ? 'Edit MME Mapping' : 'Add MME Mapping'} onClose={onClose}>
      <form onSubmit={handleSubmit}>
        <div className="modal-body">
          <div className="form-group">
            <label className="form-label">S6c Result (S6a FQDN) *</label>
            <input className="input mono" placeholder="mme1-s6a.realm.com"
              value={form.s6c_result} onChange={e => set('s6c_result', e.target.value)}
              disabled={!!initial} required />
            <div className="text-muted text-sm" style={{ marginTop: 4 }}>MME hostname as returned by S6c SRI-SM</div>
          </div>
          <div className="form-group">
            <label className="form-label">SGd Host (SGd FQDN) *</label>
            <input className="input mono" placeholder="mme1-sgd.realm.com"
              value={form.sgd_host} onChange={e => set('sgd_host', e.target.value)}
              required />
            <div className="text-muted text-sm" style={{ marginTop: 4 }}>MME hostname to use for Diameter SGd delivery</div>
          </div>
          <label className="checkbox-wrap">
            <input type="checkbox" checked={form.enabled} onChange={e => set('enabled', e.target.checked)} />
            <span>Enabled</span>
          </label>
        </div>
        <div className="modal-footer">
          <button type="button" className="btn btn-ghost" onClick={onClose}>Cancel</button>
          <button type="submit" className="btn btn-primary" disabled={submitting}>
            {submitting ? <Spinner size="sm" /> : null}
            {initial ? 'Save Changes' : 'Add Mapping'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

function DiameterPeerModal({ initial, onClose, onSaved }) {
  const toast = useToast()
  const [form, setForm] = useState(() => {
    const base = initial ? { ...DIAM_DEFAULTS, ...initial } : { ...DIAM_DEFAULTS }
    // Normalise: if old single-value 'application' came in without 'applications', convert it.
    if (!Array.isArray(base.applications) || base.applications.length === 0) {
      base.applications = base.application ? [base.application] : ['sgd']
    }
    return base
  })
  const [submitting, setSubmitting] = useState(false)
  const set = useCallback((k, v) => setForm(p => ({ ...p, [k]: v })), [])

  const toggleApp = useCallback((app) => {
    setForm(p => {
      const apps = Array.isArray(p.applications) ? p.applications : []
      return {
        ...p,
        applications: apps.includes(app) ? apps.filter(a => a !== app) : [...apps, app],
      }
    })
  }, [])

  const handleSubmit = useCallback(async (e) => {
    e.preventDefault()
    if (!form.name.trim()) { toast.error('Validation', 'Name is required.'); return }
    if (!form.host.trim()) { toast.error('Validation', 'Host is required.'); return }
    if (!form.realm.trim()) { toast.error('Validation', 'Realm is required.'); return }
    if (!Array.isArray(form.applications) || form.applications.length === 0) {
      toast.error('Validation', 'Select at least one application.')
      return
    }
    setSubmitting(true)
    try {
      const payload = {
        name: form.name,
        host: form.host,
        realm: form.realm,
        port: Number(form.port),
        transport: form.transport,
        applications: form.applications,
        enabled: form.enabled,
      }
      if (initial) {
        await updateDiameterPeer(initial.id, payload)
        toast.success('Diameter peer updated', form.name)
      } else {
        await createDiameterPeer(payload)
        toast.success('Diameter peer created', form.name)
      }
      onSaved()
    } catch (err) { toast.error('Save failed', err.message) }
    finally { setSubmitting(false) }
  }, [form, initial, toast, onSaved])

  const apps = Array.isArray(form.applications) ? form.applications : []

  return (
    <Modal title={initial ? 'Edit Diameter Peer' : 'Add Diameter Peer'} onClose={onClose}>
      <form onSubmit={handleSubmit}>
        <div className="modal-body">
          <div className="form-group">
            <label className="form-label">Name *</label>
            <input className="input" value={form.name} onChange={e => set('name', e.target.value)} disabled={!!initial} required />
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Host *</label>
              <input className="input mono" value={form.host} onChange={e => set('host', e.target.value)} required />
            </div>
            <div className="form-group">
              <label className="form-label">Port</label>
              <input className="input mono" type="number" value={form.port} onChange={e => set('port', e.target.value)} />
            </div>
          </div>
          <div className="form-group">
            <label className="form-label">Realm *</label>
            <input className="input mono" value={form.realm} onChange={e => set('realm', e.target.value)} required />
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Applications *</label>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6, marginTop: 4 }}>
                {DIAM_APP_OPTIONS.map(({ value, label }) => (
                  <label key={value} className="checkbox-wrap" style={{ fontWeight: 400 }}>
                    <input type="checkbox" checked={apps.includes(value)} onChange={() => toggleApp(value)} />
                    <span className="mono" style={{ fontSize: '0.82rem' }}>{label}</span>
                  </label>
                ))}
              </div>
            </div>
            <div className="form-group">
              <label className="form-label">Transport</label>
              <select className="select" value={form.transport} onChange={e => set('transport', e.target.value)}>
                <option value="tcp">tcp</option>
                <option value="sctp">sctp</option>
              </select>
            </div>
          </div>
          <label className="checkbox-wrap">
            <input type="checkbox" checked={form.enabled} onChange={e => set('enabled', e.target.checked)} />
            <span>Enabled</span>
          </label>
        </div>
        <div className="modal-footer">
          <button type="button" className="btn btn-ghost" onClick={onClose}>Cancel</button>
          <button type="submit" className="btn btn-primary" disabled={submitting}>
            {submitting ? <Spinner size="sm" /> : null}
            {initial ? 'Save Changes' : 'Add Peer'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
