import React, { useState, useEffect } from 'react'
import { Sun, Moon } from 'lucide-react'
import { useTheme } from '../theme.jsx'
import { getStatus } from '../api/client.js'

export default function TopBar() {
  const { theme, toggleTheme } = useTheme()
  const [status, setStatus] = useState(null)
  const [connected, setConnected] = useState(true)

  useEffect(() => {
    let mounted = true
    let timer

    async function fetchStatus() {
      try {
        const s = await getStatus()
        if (mounted) { setStatus(s); setConnected(true) }
      } catch {
        if (mounted) setConnected(false)
      }
      if (mounted) timer = setTimeout(fetchStatus, 10000)
    }

    fetchStatus()
    return () => { mounted = false; clearTimeout(timer) }
  }, [])

  return (
    <header className="topbar">
      <div className="topbar-identity">
        {status?.version && (
          <span className="topbar-identity-text mono">
            VectorCore SMSC {status.version}
            {status?.uptime && (
              <span style={{ color: 'var(--border)', marginLeft: 8, marginRight: 8 }}>·</span>
            )}
            {status?.uptime && <span>up {status.uptime}</span>}
          </span>
        )}
      </div>
      <div className="topbar-right">
        <div className="connection-indicator">
          <div className={`connection-dot ${connected ? 'connected' : 'error'}`} />
          <span>{connected ? 'Connected' : 'Disconnected'}</span>
        </div>
        <button
          className="btn-icon"
          onClick={toggleTheme}
          aria-label={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
        >
          {theme === 'dark' ? <Sun size={15} /> : <Moon size={15} />}
        </button>
      </div>
    </header>
  )
}
