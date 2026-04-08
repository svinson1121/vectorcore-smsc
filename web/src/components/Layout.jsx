import React from 'react'
import { Outlet } from 'react-router-dom'
import Sidebar from './Sidebar.jsx'
import TopBar from './TopBar.jsx'

export default function Layout() {
  return (
    <div className="layout">
      <Sidebar />
      <div className="main-content">
        <TopBar />
        <main className="page">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
