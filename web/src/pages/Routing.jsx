import React, { useState, useCallback, useEffect } from 'react'
import { Plus, Trash2, Edit3, RefreshCw, XCircle, Power } from 'lucide-react'
import Badge from '../components/Badge.jsx'
import Spinner from '../components/Spinner.jsx'
import Modal from '../components/Modal.jsx'
import { useToast } from '../components/Toast.jsx'
import { usePoller } from '../hooks/usePoller.js'
import {
  getRoutingRules, createRoutingRule, updateRoutingRule, deleteRoutingRule,
  getSFPolicies, createSFPolicy, updateSFPolicy, deleteSFPolicy,
  getSMPPClients, getSMPPAccounts, getSIPPeers, getDiameterPeers, getStatusPeers,
} from '../api/client.js'

export default function Routing() {
  const [tab, setTab] = useState('rules')
  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">Routing</div>
          <div className="page-subtitle">Routing rules and store-and-forward policies</div>
        </div>
      </div>
      <div className="tabs">
        {[
          { id: 'rules',    label: 'Routing Rules' },
          { id: 'policies', label: 'Store and Forward Policies' },
        ].map(t => (
          <button key={t.id} className={`tab-btn${tab === t.id ? ' active' : ''}`} onClick={() => setTab(t.id)}>
            {t.label}
          </button>
        ))}
      </div>
      {tab === 'rules'    && <RoutingRulesTab />}
      {tab === 'policies' && <SFPoliciesTab />}
    </div>
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

/* ── Routing Rules ──────────────────────────────────────────────────────────── */
const RULE_DEFAULTS = {
  name: '', priority: 10,
  match_src_iface: '', match_src_peer: '', match_dst_prefix: '',
  match_msisdn_min: '', match_msisdn_max: '',
  egress_iface: 'smpp', egress_peer: '',
  sf_policy_id: '',
  enabled: true,
}

function buildRoutingRulePayload(rule) {
  return {
    name: rule.name ?? '',
    priority: Number(rule.priority),
    match_src_iface: rule.match_src_iface ?? '',
    match_src_peer: rule.match_src_peer ?? '',
    match_dst_prefix: rule.match_dst_prefix ?? '',
    match_msisdn_min: rule.match_msisdn_min ?? '',
    match_msisdn_max: rule.match_msisdn_max ?? '',
    egress_iface: rule.egress_iface ?? '',
    egress_peer: rule.egress_peer ?? '',
    sf_policy_id: rule.sf_policy_id ?? '',
    enabled: rule.enabled ?? true,
  }
}

function EnabledStatus({ enabled }) {
  return <Badge state={enabled ? 'enabled' : 'disabled'} />
}

function RoutingRulesTab() {
  const toast = useToast()
  const { data, error, loading, refresh } = usePoller(getRoutingRules)
  const [sfPolicyNames, setSFPolicyNames] = useState({})
  const [showModal, setShowModal] = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [busy, setBusy] = useState({})

  const handleToggle = useCallback(async (rule) => {
    setBusy(p => ({ ...p, [rule.id]: true }))
    try {
      await updateRoutingRule(rule.id, buildRoutingRulePayload({ ...rule, enabled: !rule.enabled }))
      toast.success(rule.enabled ? 'Rule disabled' : 'Rule enabled')
      refresh()
    } catch (err) { toast.error('Action failed', err.message) }
    finally { setBusy(p => ({ ...p, [rule.id]: false })) }
  }, [toast, refresh])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    setBusy(p => ({ ...p, [deleteTarget.id]: true }))
    try {
      await deleteRoutingRule(deleteTarget.id)
      toast.success('Rule deleted', deleteTarget.name)
      setDeleteTarget(null); refresh()
    } catch (err) { toast.error('Delete failed', err.message) }
    finally { setBusy(p => { const n = {...p}; delete n[deleteTarget.id]; return n }) }
  }, [deleteTarget, toast, refresh])

  const sorted = Array.isArray(data) ? [...data].sort((a, b) => a.priority - b.priority) : []

  useEffect(() => {
    getSFPolicies()
      .then((policies) => {
        const names = {}
        ;(policies || []).forEach((policy) => {
          names[policy.id] = policy.name
        })
        setSFPolicyNames(names)
      })
      .catch(() => setSFPolicyNames({}))
  }, [data])

  if (loading) return <div className="loading-center"><Spinner size="md" /></div>
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
        <span className="text-muted text-sm">{sorted.length} rule{sorted.length !== 1 ? 's' : ''}</span>
        <div className="flex gap-8">
          <button className="btn btn-ghost btn-sm" onClick={refresh}><RefreshCw size={12} /></button>
          <button className="btn btn-primary btn-sm" onClick={() => { setEditTarget(null); setShowModal(true) }}>
            <Plus size={12} /> Add Rule
          </button>
        </div>
      </div>

      {sorted.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon" style={{ fontSize: 28 }}>◎</div>
          <div style={{ fontWeight: 600, marginBottom: 4 }}>No routing rules configured</div>
          <div className="text-muted text-sm">Rules are evaluated in ascending priority order. First match wins.</div>
          <button className="btn btn-primary btn-sm mt-12" onClick={() => { setEditTarget(null); setShowModal(true) }}>
            <Plus size={12} /> Add First Rule
          </button>
        </div>
      ) : (
        <div className="table-container">
          <table>
            <thead><tr>
              <th>Pri</th><th>Name</th><th>Match Ingress</th><th>Match Dst Prefix</th>
              <th>Egress</th><th>Peer</th><th>SF Policy</th><th>Enabled</th><th>Actions</th>
            </tr></thead>
            <tbody>
              {sorted.map(rule => (
                <tr key={rule.id}>
                  <td className="mono" style={{ fontWeight: 700 }}>{rule.priority}</td>
                  <td style={{ fontWeight: 500 }}>{rule.name || '—'}</td>
                  <td>
                    <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                      {rule.match_src_iface && <Badge state={rule.match_src_iface} />}
                      {rule.match_src_peer && <span className="mono text-muted" style={{ fontSize: '0.75rem' }}>{rule.match_src_peer}</span>}
                      {!rule.match_src_iface && !rule.match_src_peer && <span className="text-muted">any</span>}
                    </div>
                  </td>
                  <td className="mono" style={{ fontSize: '0.8rem', color: rule.match_dst_prefix ? 'var(--text)' : 'var(--text-muted)' }}>
                    {rule.match_dst_prefix || '*'}
                  </td>
                  <td>
                    <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                      <Badge state={rule.egress_iface} />
                    </div>
                  </td>
                  <td className="mono text-muted" style={{ fontSize: '0.8rem' }}>
                    {rule.egress_peer || <span className="text-muted">—</span>}
                  </td>
                  <td className="text-muted" style={{ fontSize: '0.8rem' }}>
                    {rule.sf_policy_id ? (sfPolicyNames[rule.sf_policy_id] || 'Default Policy') : 'Default Policy'}
                  </td>
                  <td><EnabledStatus enabled={rule.enabled ?? true} /></td>
                  <td>
                    <div className="flex gap-6">
                      <button
                        className="btn-icon"
                        title={(rule.enabled ?? true) ? 'Disable rule' : 'Enable rule'}
                        aria-label={(rule.enabled ?? true) ? 'Disable rule' : 'Enable rule'}
                        disabled={busy[rule.id]}
                        onClick={() => handleToggle(rule)}
                      >
                        <Power size={13} />
                      </button>
                      <button className="btn-icon" onClick={() => { setEditTarget(rule); setShowModal(true) }}><Edit3 size={13} /></button>
                      <button className="btn-icon danger" disabled={busy[rule.id]} onClick={() => setDeleteTarget(rule)}><Trash2 size={13} /></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showModal && <RuleModal initial={editTarget} onClose={() => setShowModal(false)} onSaved={() => { setShowModal(false); refresh() }} />}
      {deleteTarget && <ConfirmDeleteModal label={`rule "${deleteTarget.name || 'priority ' + deleteTarget.priority}"`}
        onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} loading={!!busy[deleteTarget?.id]} />}
    </div>
  )
}

function RuleModal({ initial, onClose, onSaved }) {
  const toast = useToast()
  const [form, setForm] = useState(initial ? buildRoutingRulePayload(initial) : { ...RULE_DEFAULTS })
  const [submitting, setSubmitting] = useState(false)
  const [peers, setPeers] = useState({ smpp: [], sipsimple: [], sgd: [] })
  const [sfPolicies, setSFPolicies] = useState([])
  const set = useCallback((k, v) => setForm(p => ({ ...p, [k]: v })), [])

  useEffect(() => {
    Promise.all([
      getSMPPClients().catch(() => []),
      getSMPPAccounts().catch(() => []),
      getSIPPeers().catch(() => []),
      getDiameterPeers().catch(() => []),
      getStatusPeers().catch(() => []),
      getSFPolicies().catch(() => []),
    ]).then(([smppClients, smppAccounts, sipPeers, diamPeers, livePeers, policies]) => {
      // SMPP: outbound clients by name + inbound server accounts by system_id
      const smppClientNames = (smppClients || []).filter(c => c.enabled).map(c => ({ value: c.name, label: c.name }))
      const smppServerNames = (smppAccounts || []).filter(a => a.enabled).map(a => ({ value: a.system_id, label: `${a.system_id} (server)` }))

      // SGd: configured DB peers + live inbound peers from status
      const sgdDbNames = (diamPeers || [])
        .filter(p => p.enabled && Array.isArray(p.applications) && p.applications.includes('sgd'))
        .map(p => p.name)
      const sgdLiveNames = (livePeers || [])
        .filter(p => p.type === 'diameter_sgd')
        .map(p => p.name)
      const sgdAll = [...new Set([...sgdDbNames, ...sgdLiveNames])]

      setPeers({
        smpp: [...smppClientNames, ...smppServerNames],
        sipsimple: (sipPeers || []).filter(p => p.enabled).map(p => ({ value: p.name, label: p.name })),
        sgd: sgdAll.map(n => ({ value: n, label: n })),
      })
      setSFPolicies((policies || []).map(p => ({ value: p.id, label: p.name })))
    })
  }, [])

  const ingressPeerOptions = peers[form.match_src_iface] || []
  const useIngressPeerSelect = form.match_src_iface === 'smpp' || form.match_src_iface === 'sipsimple' || form.match_src_iface === 'sgd'

  const handleSubmit = useCallback(async (e) => {
    e.preventDefault()
    if (!form.egress_iface) { toast.error('Validation', 'Egress interface is required.'); return }
    setSubmitting(true)
    try {
      const payload = buildRoutingRulePayload(form)
      if (initial) {
        await updateRoutingRule(initial.id, payload)
        toast.success('Rule updated')
      } else {
        await createRoutingRule(payload)
        toast.success('Rule created')
      }
      onSaved()
    } catch (err) { toast.error('Save failed', err.message) }
    finally { setSubmitting(false) }
  }, [form, initial, toast, onSaved])

  return (
    <Modal title={initial ? 'Edit Routing Rule' : 'Add Routing Rule'} onClose={onClose} size="lg">
      <form onSubmit={handleSubmit}>
        <div className="modal-body">
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Name</label>
              <input className="input" value={form.name} onChange={e => set('name', e.target.value)} placeholder="optional label" />
            </div>
            <div className="form-group">
              <label className="form-label">Priority</label>
              <input className="input mono" type="number" min={1} value={form.priority} onChange={e => set('priority', e.target.value)} />
              <span className="form-hint">Lower = evaluated first</span>
            </div>
          </div>

          <div style={{ fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--text-muted)', marginBottom: 6 }}>
            Match Criteria (all non-empty fields must match)
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Ingress Interface</label>
              <select className="select" value={form.match_src_iface} onChange={e => set('match_src_iface', e.target.value)}>
                <option value="">any</option>
                <option value="sip3gpp">sip3gpp</option>
                <option value="sipsimple">sipsimple</option>
                <option value="smpp">smpp</option>
                <option value="sgd">sgd</option>
              </select>
            </div>
            <div className="form-group">
              <label className="form-label">Ingress Peer</label>
              {form.match_src_iface === '' ? (
                <input className="input" value="— any —" disabled />
              ) : form.match_src_iface === 'sip3gpp' ? (
                <input className="input" value="— IMS registry —" disabled />
              ) : useIngressPeerSelect ? (
                <select className="select" value={form.match_src_peer} onChange={e => set('match_src_peer', e.target.value)}>
                  <option value="">— any —</option>
                  {ingressPeerOptions.map(({ value, label }) => (
                    <option key={value} value={value}>{label}</option>
                  ))}
                </select>
              ) : (
                <input className="input mono" value={form.match_src_peer} onChange={e => set('match_src_peer', e.target.value)} placeholder="peer name or blank = any" />
              )}
              <span className="form-hint">
                {form.match_src_iface === 'smpp' && 'Select an SMPP client or server account'}
                {form.match_src_iface === 'sipsimple' && 'Select a SIP peer'}
                {form.match_src_iface === 'sgd' && 'Select a Diameter SGd peer'}
                {form.match_src_iface === '' && 'Any ingress interface matches any peer'}
                {form.match_src_iface === 'sip3gpp' && 'IMS ingress uses the registry lookup rather than a specific peer selection'}
              </span>
            </div>
          </div>
          <div className="form-group">
            <label className="form-label">Dest MSISDN Prefix</label>
            <input className="input mono" value={form.match_dst_prefix} onChange={e => set('match_dst_prefix', e.target.value)} placeholder="+1, +44, etc. — blank = any" />
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">MSISDN Range Min</label>
              <input className="input mono" value={form.match_msisdn_min} onChange={e => set('match_msisdn_min', e.target.value)} placeholder="+14150000000" />
            </div>
            <div className="form-group">
              <label className="form-label">MSISDN Range Max</label>
              <input className="input mono" value={form.match_msisdn_max} onChange={e => set('match_msisdn_max', e.target.value)} placeholder="+14159999999" />
            </div>
          </div>

          <div style={{ fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--text-muted)', marginBottom: 6, marginTop: 4 }}>
            Action
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Egress Interface *</label>
              <select className="select" value={form.egress_iface} onChange={e => { set('egress_iface', e.target.value); set('egress_peer', '') }}>
                <option value="sip3gpp">sip3gpp — S-CSCF (ISC)</option>
                <option value="sipsimple">sipsimple — inter-site SIP</option>
                <option value="smpp">smpp — SMPP</option>
                <option value="sgd">sgd — Diameter SGd (MME)</option>
              </select>
            </div>
            <div className="form-group">
              <label className="form-label">Egress Peer</label>
              {form.egress_iface === 'sip3gpp' ? (
                <input className="input" value="— uses IMS registry —" disabled />
              ) : (
                <select className="select" value={form.egress_peer} onChange={e => set('egress_peer', e.target.value)}>
                  <option value="">— default / any —</option>
                  {(peers[form.egress_iface] || []).map(({ value, label }) => (
                    <option key={value} value={value}>{label}</option>
                  ))}
                </select>
              )}
              <span className="form-hint">
                {form.egress_iface === 'smpp' && 'SMPP client name'}
                {form.egress_iface === 'sipsimple' && 'SIP peer name'}
                {form.egress_iface === 'sgd' && 'Diameter peer name'}
              </span>
            </div>
          </div>
          <div className="form-group">
            <label className="form-label">Store and Forward Policy</label>
            <select className="select" value={form.sf_policy_id || ''} onChange={e => set('sf_policy_id', e.target.value)}>
              <option value="">Default Policy</option>
              {sfPolicies.map(({ value, label }) => (
                <option key={value} value={value}>{label}</option>
              ))}
            </select>
            <span className="form-hint">If no specific policy is selected, the default retry behavior is used.</span>
          </div>

          <label className="checkbox-wrap">
            <input type="checkbox" checked={form.enabled ?? true} onChange={e => set('enabled', e.target.checked)} />
            <span>Enabled</span>
          </label>
        </div>
        <div className="modal-footer">
          <button type="button" className="btn btn-ghost" onClick={onClose}>Cancel</button>
          <button type="submit" className="btn btn-primary" disabled={submitting}>
            {submitting ? <Spinner size="sm" /> : null}
            {initial ? 'Save Changes' : 'Add Rule'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

/* ── SF Policies ────────────────────────────────────────────────────────────── */
const SF_DEFAULTS = {
  name: '', max_retries: 8, max_ttl: '48h0m0s',
  retry_schedule: '[30, 300, 1800, 3600, 3600, 3600, 3600, 3600]',
  vp_override: '',
}

function buildSFPolicyFormState(policy) {
  const asDurationString = (value, fallback = '') => {
    if (value === undefined || value === null || value === '') return fallback
    if (typeof value === 'string') return value
    if (typeof value === 'number' && Number.isFinite(value)) return `${value}ns`
    return String(value)
  }

  return {
    name: policy.name ?? '',
    max_retries: policy.max_retries ?? 8,
    max_ttl: asDurationString(policy.max_ttl, '48h0m0s'),
    retry_schedule: policy.retry_schedule
      ? (typeof policy.retry_schedule === 'string' ? policy.retry_schedule : JSON.stringify(policy.retry_schedule))
      : SF_DEFAULTS.retry_schedule,
    vp_override: asDurationString(policy.vp_override, ''),
  }
}

function buildSFPolicyPayload(form, retrySchedule) {
  return {
    name: form.name ?? '',
    max_retries: Number(form.max_retries),
    max_ttl: String(form.max_ttl ?? ''),
    retry_schedule: retrySchedule,
    vp_override: String(form.vp_override ?? ''),
  }
}

function SFPoliciesTab() {
  const toast = useToast()
  const { data, error, loading, refresh } = usePoller(getSFPolicies)
  const { data: rulesData } = usePoller(getRoutingRules)
  const [showModal, setShowModal] = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [busy, setBusy] = useState({})

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    setBusy(p => ({ ...p, [deleteTarget.id]: true }))
    try {
      await deleteSFPolicy(deleteTarget.id)
      toast.success('SF policy deleted', deleteTarget.name)
      setDeleteTarget(null); refresh()
    } catch (err) { toast.error('Delete failed', err.message) }
    finally { setBusy(p => { const n = {...p}; delete n[deleteTarget.id]; return n }) }
  }, [deleteTarget, toast, refresh])

  const list = Array.isArray(data) ? data : []
  const usageByPolicy = (() => {
    const counts = {}
    ;(Array.isArray(rulesData) ? rulesData : []).forEach((rule) => {
      if (!rule.sf_policy_id) return
      counts[rule.sf_policy_id] = (counts[rule.sf_policy_id] || 0) + 1
    })
    return counts
  })()

  if (loading) return <div className="loading-center"><Spinner size="md" /></div>
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
        <span className="text-muted text-sm">{list.length} polic{list.length !== 1 ? 'ies' : 'y'}</span>
        <div className="flex gap-8">
          <button className="btn btn-ghost btn-sm" onClick={refresh}><RefreshCw size={12} /></button>
          <button className="btn btn-primary btn-sm" onClick={() => { setEditTarget(null); setShowModal(true) }}>
            <Plus size={12} /> Add Policy
          </button>
        </div>
      </div>

      {list.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon" style={{ fontSize: 28 }}>◎</div>
          <div style={{ fontWeight: 600, marginBottom: 4 }}>No SF policies</div>
          <div className="text-muted text-sm">SF policies define retry schedules and max TTL for store-and-forward.</div>
          <button className="btn btn-primary btn-sm mt-12" onClick={() => { setEditTarget(null); setShowModal(true) }}>
            <Plus size={12} /> Add Policy
          </button>
        </div>
      ) : (
        <div className="table-container">
          <table>
            <thead><tr>
              <th>Name</th><th>Max Retries</th><th>Max TTL</th><th>Retry Schedule</th><th>Used By Rules</th><th>Actions</th>
            </tr></thead>
            <tbody>
              {list.map(pol => (
                <tr key={pol.id}>
                  <td style={{ fontWeight: 600 }}>{pol.name}</td>
                  <td className="mono">{pol.max_retries}</td>
                  <td className="mono text-muted">{pol.max_ttl}</td>
                  <td className="mono text-muted" style={{ fontSize: '0.75rem', maxWidth: 260, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {JSON.stringify(pol.retry_schedule)}
                  </td>
                  <td className="text-muted" style={{ fontSize: '0.8rem' }}>
                    {usageByPolicy[pol.id] || 0}
                  </td>
                  <td>
                    <div className="flex gap-6">
                      <button className="btn-icon" onClick={() => { setEditTarget(pol); setShowModal(true) }}><Edit3 size={13} /></button>
                      <button
                        className="btn-icon danger"
                        disabled={busy[pol.id] || !!usageByPolicy[pol.id]}
                        title={usageByPolicy[pol.id] ? 'Policy is still used by one or more routing rules' : 'Delete policy'}
                        onClick={() => setDeleteTarget(pol)}
                      ><Trash2 size={13} /></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showModal && <SFPolicyModal initial={editTarget} onClose={() => setShowModal(false)} onSaved={() => { setShowModal(false); refresh() }} />}
      {deleteTarget && <ConfirmDeleteModal label={`SF policy "${deleteTarget.name}"`}
        onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} loading={!!busy[deleteTarget?.id]} />}
    </div>
  )
}

function SFPolicyModal({ initial, onClose, onSaved }) {
  const toast = useToast()
  const [form, setForm] = useState(initial ? buildSFPolicyFormState(initial) : { ...SF_DEFAULTS })
  const [submitting, setSubmitting] = useState(false)
  const set = useCallback((k, v) => setForm(p => ({ ...p, [k]: v })), [])

  const handleSubmit = useCallback(async (e) => {
    e.preventDefault()
    if (!form.name.trim()) { toast.error('Validation', 'Name is required.'); return }
    let schedule
    try { schedule = JSON.parse(form.retry_schedule) } catch {
      toast.error('Validation', 'Retry schedule must be valid JSON array of seconds.')
      return
    }
    setSubmitting(true)
    try {
      const payload = buildSFPolicyPayload(form, schedule)
      if (initial) {
        await updateSFPolicy(initial.id, payload)
        toast.success('SF policy updated', form.name)
      } else {
        await createSFPolicy(payload)
        toast.success('SF policy created', form.name)
      }
      onSaved()
    } catch (err) { toast.error('Save failed', err.message) }
    finally { setSubmitting(false) }
  }, [form, initial, toast, onSaved])

  return (
    <Modal title={initial ? 'Edit SF Policy' : 'Add SF Policy'} onClose={onClose}>
      <form onSubmit={handleSubmit}>
        <div className="modal-body">
          <div className="form-group">
            <label className="form-label">Name *</label>
            <input className="input" value={form.name} onChange={e => set('name', e.target.value)} disabled={!!initial} required />
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Max Retries</label>
              <input className="input mono" type="number" min={0} value={form.max_retries} onChange={e => set('max_retries', e.target.value)} />
            </div>
            <div className="form-group">
              <label className="form-label">Max TTL</label>
              <input className="input mono" value={form.max_ttl} onChange={e => set('max_ttl', e.target.value)} placeholder="48h0m0s" />
              <span className="form-hint">Go duration format</span>
            </div>
          </div>
          <div className="form-group">
            <label className="form-label">Retry Schedule (seconds)</label>
            <input className="input mono" value={form.retry_schedule} onChange={e => set('retry_schedule', e.target.value)} placeholder="[30, 300, 1800, 3600]" />
            <span className="form-hint">JSON array of wait intervals between retries</span>
          </div>
        </div>
        <div className="modal-footer">
          <button type="button" className="btn btn-ghost" onClick={onClose}>Cancel</button>
          <button type="submit" className="btn btn-primary" disabled={submitting}>
            {submitting ? <Spinner size="sm" /> : null}
            {initial ? 'Save Changes' : 'Add Policy'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
