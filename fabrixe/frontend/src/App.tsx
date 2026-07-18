import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider, useAuthContext } from './context/AuthContext'
import Layout from './components/layout/Layout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import SystemManagement from './pages/SystemManagement'
import Deployment from './pages/Deployment'
import Security from './pages/Security'
import Communication from './pages/Communication'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, loading } = useAuthContext()
  if (loading) {
    return (
      <div className="h-full flex items-center justify-center bg-slate-950">
        <div className="flex flex-col items-center gap-4">
          <div className="w-12 h-12 border-4 border-fabrixe-600 border-t-transparent rounded-full animate-spin" />
          <span className="text-slate-400 text-sm">Loading Fabrixe…</span>
        </div>
      </div>
    )
  }
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route
            path="/"
            element={
              <PrivateRoute>
                <Layout />
              </PrivateRoute>
            }
          >
            <Route index element={<Dashboard />} />
            <Route path="system" element={<SystemManagement />} />
            <Route path="deployment" element={<Deployment />} />
            <Route path="security" element={<Security />} />
            <Route path="communication" element={<Communication />} />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
