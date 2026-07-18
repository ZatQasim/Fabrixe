import { Bell, LogOut, User, ChevronDown } from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthContext } from '../../context/AuthContext'
import { clsx } from 'clsx'

export default function Header() {
  const { user, logout } = useAuthContext()
  const navigate = useNavigate()
  const [menuOpen, setMenuOpen] = useState(false)

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  const roleColor: Record<string, string> = {
    administrator: 'text-fabrixe-400',
    operator: 'text-amber-400',
    viewer: 'text-slate-400',
  }

  return (
    <header className="flex items-center justify-between h-14 px-6 bg-slate-900 border-b border-slate-800 shrink-0">
      <div className="flex items-center gap-2">
        {/* Breadcrumb or title could go here */}
      </div>

      <div className="flex items-center gap-3">
        {/* Notifications placeholder */}
        <button className="relative p-2 rounded-lg text-slate-500 hover:text-slate-300 hover:bg-slate-800 transition-colors">
          <Bell className="w-4 h-4" />
        </button>

        {/* User menu */}
        <div className="relative">
          <button
            onClick={() => setMenuOpen(o => !o)}
            className="flex items-center gap-2 px-3 py-1.5 rounded-lg hover:bg-slate-800 transition-colors"
          >
            <div className="w-7 h-7 rounded-full bg-fabrixe-600/30 border border-fabrixe-500/40 flex items-center justify-center">
              <User className="w-3.5 h-3.5 text-fabrixe-400" />
            </div>
            <div className="text-left hidden sm:block">
              <p className="text-sm font-medium text-slate-200 leading-none">{user?.full_name || user?.username}</p>
              <p className={clsx('text-xs leading-none mt-0.5', roleColor[user?.role ?? ''] ?? 'text-slate-500')}>
                {user?.role}
              </p>
            </div>
            <ChevronDown className="w-3.5 h-3.5 text-slate-500" />
          </button>

          {menuOpen && (
            <div className="absolute right-0 mt-1 w-52 bg-slate-800 border border-slate-700 rounded-xl shadow-xl z-50 py-1 animate-slide-in">
              <div className="px-3 py-2 border-b border-slate-700">
                <p className="text-xs text-slate-400">{user?.email}</p>
              </div>
              <button
                onClick={handleLogout}
                className="flex items-center gap-2 w-full px-3 py-2.5 text-sm text-red-400 hover:bg-red-500/10 transition-colors"
              >
                <LogOut className="w-4 h-4" />
                Sign out
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Close menu on outside click */}
      {menuOpen && (
        <div className="fixed inset-0 z-40" onClick={() => setMenuOpen(false)} />
      )}
    </header>
  )
}
