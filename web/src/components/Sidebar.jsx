import React from 'react'
import { NavLink } from 'react-router-dom'
import { LayoutDashboard, Network, GitBranch, BarChart2, Settings } from 'lucide-react'

const NAV_ITEMS = [
  { to: '/dashboard', label: 'Dashboard', icon: <LayoutDashboard size={16} /> },
  { to: '/peers',     label: 'Peers',     icon: <Network size={16} /> },
  { to: '/routing',   label: 'Routing',   icon: <GitBranch size={16} /> },
  { to: '/metrics',   label: 'Metrics',   icon: <BarChart2 size={16} /> },
  { to: '/oam',       label: 'OAM',       icon: <Settings size={16} /> },
]

export default function Sidebar() {
  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <div className="sidebar-logo">VectorCore</div>
        <div className="sidebar-logo-sub">SMS Center</div>
      </div>
      <nav className="sidebar-nav" aria-label="Primary navigation">
        {NAV_ITEMS.map(({ to, label, icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}
          >
            {icon}
            {label}
          </NavLink>
        ))}
      </nav>
      <div className="sidebar-footer" />
    </aside>
  )
}
