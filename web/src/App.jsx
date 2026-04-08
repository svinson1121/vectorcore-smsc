import React from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import Layout from './components/Layout.jsx'
import Dashboard from './pages/Dashboard.jsx'
import Peers from './pages/Peers.jsx'
import Routing from './pages/Routing.jsx'
import Metrics from './pages/Metrics.jsx'
import OAM from './pages/OAM.jsx'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<Dashboard />} />
        <Route path="peers" element={<Peers />} />
        <Route path="routing" element={<Routing />} />
        <Route path="metrics" element={<Metrics />} />
        <Route path="oam" element={<OAM />} />
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Route>
    </Routes>
  )
}
