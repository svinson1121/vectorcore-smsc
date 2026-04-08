import React from 'react'

const STATE_MAP = {
  // Message status
  QUEUED:     { cls: 'badge-queued',     label: 'QUEUED' },
  DISPATCHED: { cls: 'badge-dispatched', label: 'DISPATCHED' },
  DELIVERED:  { cls: 'badge-delivered',  label: 'DELIVERED' },
  FAILED:     { cls: 'badge-failed',     label: 'FAILED' },
  EXPIRED:    { cls: 'badge-expired',    label: 'EXPIRED' },
  // Interfaces
  sip3gpp:    { cls: 'badge-sip3gpp',   label: 'SIP/3GPP' },
  sipsimple:  { cls: 'badge-sipsimple', label: 'SIP Simple' },
  smpp:       { cls: 'badge-smpp',      label: 'SMPP' },
  sgd:        { cls: 'badge-info',      label: 'SGd' },
  // Generic state
  enabled:    { cls: 'badge-enabled',   label: 'Enabled' },
  disabled:   { cls: 'badge-disabled',  label: 'Disabled' },
  open:       { cls: 'badge-open',      label: 'open' },
  closed:     { cls: 'badge-closed',    label: 'closed' },
  connecting: { cls: 'badge-connecting', label: 'connecting' },
  // Boolean
  true:       { cls: 'badge-true',   label: 'yes' },
  false:      { cls: 'badge-false',  label: 'no' },
  // Transport
  udp: { cls: 'badge-udp', label: 'udp' },
  tcp: { cls: 'badge-tcp', label: 'tcp' },
  tls: { cls: 'badge-tls', label: 'tls' },
  // Bind types
  transceiver:  { cls: 'badge-info',     label: 'transceiver' },
  transmitter:  { cls: 'badge-info',     label: 'transmitter' },
  receiver:     { cls: 'badge-info',     label: 'receiver' },
  // Diameter apps
  sgd_app: { cls: 'badge-info', label: 'SGd' },
  sh:      { cls: 'badge-info', label: 'Sh' },
  s6c:     { cls: 'badge-info', label: 'S6c' },
}

export default function Badge({ state, label: labelOverride }) {
  if (state === undefined || state === null) return null
  const key = String(state).toLowerCase()
  const entry = STATE_MAP[key] || STATE_MAP[String(state)] || { cls: 'badge-disabled', label: String(state) }
  return (
    <span className={`badge ${entry.cls}`}>
      {labelOverride || entry.label}
    </span>
  )
}
