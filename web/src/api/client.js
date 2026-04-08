const BASE = '/api/v1'

async function request(method, path, body) {
  const opts = { method, headers: {} }
  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(body)
  }
  const res = await fetch(`${BASE}${path}`, opts)
  if (res.status === 204) return null
  if (!res.ok) {
    let msg = `HTTP ${res.status}`
    try {
      const data = await res.json()
      msg = data.detail || data.message || data.error || msg
    } catch { /* ignore */ }
    throw new Error(msg)
  }
  const text = await res.text()
  if (!text) return null
  return JSON.parse(text)
}

// ── Status ────────────────────────────────────────────────────────────────────
export const getStatus      = () => request('GET', '/status')
export const getStatusPeers = () => request('GET', '/status/peers')

// ── SMPP Server Accounts ──────────────────────────────────────────────────────
export const getSMPPAccounts    = ()       => request('GET',    '/smpp/server/accounts')
export const getSMPPAccount     = (id)     => request('GET',    `/smpp/server/accounts/${id}`)
export const createSMPPAccount  = (data)   => request('POST',   '/smpp/server/accounts', data)
export const updateSMPPAccount  = (id, d)  => request('PUT',    `/smpp/server/accounts/${id}`, d)
export const deleteSMPPAccount  = (id)     => request('DELETE', `/smpp/server/accounts/${id}`)

// ── SMPP Clients ──────────────────────────────────────────────────────────────
export const getSMPPClients    = ()       => request('GET',    '/smpp/clients')
export const getSMPPClient     = (id)     => request('GET',    `/smpp/clients/${id}`)
export const createSMPPClient  = (data)   => request('POST',   '/smpp/clients', data)
export const updateSMPPClient  = (id, d)  => request('PUT',    `/smpp/clients/${id}`, d)
export const deleteSMPPClient  = (id)     => request('DELETE', `/smpp/clients/${id}`)

// ── SIP Peers ─────────────────────────────────────────────────────────────────
export const getSIPPeers    = ()       => request('GET',    '/sip/peers')
export const getSIPPeer     = (id)     => request('GET',    `/sip/peers/${id}`)
export const createSIPPeer  = (data)   => request('POST',   '/sip/peers', data)
export const updateSIPPeer  = (id, d)  => request('PUT',    `/sip/peers/${id}`, d)
export const deleteSIPPeer  = (id)     => request('DELETE', `/sip/peers/${id}`)

// ── Diameter Peers ────────────────────────────────────────────────────────────
export const getDiameterPeers    = ()       => request('GET',    '/diameter/peers')
export const getDiameterPeer     = (id)     => request('GET',    `/diameter/peers/${id}`)
export const createDiameterPeer  = (data)   => request('POST',   '/diameter/peers', data)
export const updateDiameterPeer  = (id, d)  => request('PUT',    `/diameter/peers/${id}`, d)
export const deleteDiameterPeer  = (id)     => request('DELETE', `/diameter/peers/${id}`)

// ── Routing Rules ─────────────────────────────────────────────────────────────
export const getRoutingRules    = ()       => request('GET',    '/routing/rules')
export const getRoutingRule     = (id)     => request('GET',    `/routing/rules/${id}`)
export const createRoutingRule  = (data)   => request('POST',   '/routing/rules', data)
export const updateRoutingRule  = (id, d)  => request('PUT',    `/routing/rules/${id}`, d)
export const deleteRoutingRule  = (id)     => request('DELETE', `/routing/rules/${id}`)

// ── SF Policies ───────────────────────────────────────────────────────────────
export const getSFPolicies    = ()       => request('GET',    '/routing/policies')
export const getSFPolicy      = (id)     => request('GET',    `/routing/policies/${id}`)
export const createSFPolicy   = (data)   => request('POST',   '/routing/policies', data)
export const updateSFPolicy   = (id, d)  => request('PUT',    `/routing/policies/${id}`, d)
export const deleteSFPolicy   = (id)     => request('DELETE', `/routing/policies/${id}`)

// ── Subscribers ───────────────────────────────────────────────────────────────
export const getSubscribers    = ()       => request('GET',    '/subscribers')
export const getSubscriber     = (id)     => request('GET',    `/subscribers/${id}`)
export const createSubscriber  = (data)   => request('POST',   '/subscribers', data)
export const updateSubscriber  = (id, d)  => request('PUT',    `/subscribers/${id}`, d)
export const deleteSubscriber  = (id)     => request('DELETE', `/subscribers/${id}`)

// ── Messages (read-only) ──────────────────────────────────────────────────────
export const getMessages       = (limit) => request('GET', `/messages?limit=${limit || 100}`)
export const getMessage        = (id)    => request('GET', `/messages/${id}`)
export const getQueueMessages  = ({ limit, srcMSISDN, dstMSISDN, originPeer } = {}) => {
  const params = new URLSearchParams()
  params.set('limit', String(limit || 100))
  if (srcMSISDN) params.set('src_msisdn', srcMSISDN)
  if (dstMSISDN) params.set('dst_msisdn', dstMSISDN)
  if (originPeer) params.set('origin_peer', originPeer)
  return request('GET', `/messages/queue?${params.toString()}`)
}
export const deleteQueueMessage = (id) => request('DELETE', `/messages/queue/${id}`)

// ── Delivery Reports (read-only) ──────────────────────────────────────────────
export const getDeliveryReports = (limit) => request('GET', `/delivery-reports?limit=${limit || 100}`)
export const getDeliveryReport  = (id)    => request('GET', `/delivery-reports/${id}`)

// ── SGd MME Mappings ──────────────────────────────────────────────────────────
export const getSGDMMEMappings    = ()       => request('GET',    '/sgd/mme-mappings')
export const getSGDMMEMapping     = (id)     => request('GET',    `/sgd/mme-mappings/${id}`)
export const createSGDMMEMapping  = (data)   => request('POST',   '/sgd/mme-mappings', data)
export const updateSGDMMEMapping  = (id, d)  => request('PUT',    `/sgd/mme-mappings/${id}`, d)
export const deleteSGDMMEMapping  = (id)     => request('DELETE', `/sgd/mme-mappings/${id}`)

// ── Raw Prometheus ────────────────────────────────────────────────────────────
export async function getPrometheusText() {
  const res = await fetch('/metrics')
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.text()
}

// ── Prometheus text parser ────────────────────────────────────────────────────
export function parsePrometheusText(text) {
  const metrics = {}
  if (!text) return metrics
  const lines = text.split('\n')
  const helpMap = {}
  const typeMap = {}
  for (const raw of lines) {
    const line = raw.trim()
    if (!line) continue
    if (line.startsWith('# HELP ')) {
      const rest = line.slice(7)
      const sp = rest.indexOf(' ')
      helpMap[rest.slice(0, sp)] = rest.slice(sp + 1)
      continue
    }
    if (line.startsWith('# TYPE ')) {
      const parts = line.slice(7).split(' ')
      typeMap[parts[0]] = parts[1]
      continue
    }
    const braceOpen = line.indexOf('{')
    const spaceIdx = line.lastIndexOf(' ')
    let name, labelsStr, value
    if (braceOpen !== -1) {
      const braceClose = line.indexOf('}')
      name = line.slice(0, braceOpen)
      labelsStr = line.slice(braceOpen + 1, braceClose)
      value = parseFloat(line.slice(braceClose + 1).trim().split(' ')[0])
    } else {
      name = line.slice(0, spaceIdx)
      labelsStr = ''
      value = parseFloat(line.slice(spaceIdx + 1).split(' ')[0])
    }
    const labels = {}
    if (labelsStr) {
      const re = /(\w+)="([^"]*)"/g
      let m
      while ((m = re.exec(labelsStr)) !== null) labels[m[1]] = m[2]
    }
    if (!metrics[name]) {
      metrics[name] = { name, help: helpMap[name] || '', type: typeMap[name] || 'untyped', samples: [] }
    }
    metrics[name].samples.push({ labels, value })
  }
  return metrics
}

export function sumMetric(metrics, name) {
  const m = metrics[name]
  if (!m) return 0
  return m.samples.reduce((acc, s) => acc + (isNaN(s.value) ? 0 : s.value), 0)
}

export function getMetricSamples(metrics, name) {
  return metrics[name]?.samples || []
}

export function sumMetricByLabel(metrics, name, labelKey, labelValue) {
  const m = metrics[name]
  if (!m) return 0
  return m.samples
    .filter(s => s.labels[labelKey] === labelValue)
    .reduce((acc, s) => acc + (isNaN(s.value) ? 0 : s.value), 0)
}
