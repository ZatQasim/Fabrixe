import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react'
import { auth } from '../lib/api'
import type { User } from '../types'

interface AuthContextValue {
  user: User | null
  loading: boolean
  isAuthenticated: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!auth.isAuthenticated()) {
      setLoading(false)
      return
    }
    auth.me()
      .then(u => setUser(u as User))
      .catch(() => auth.clearTokens())
      .finally(() => setLoading(false))
  }, [])

  const login = useCallback(async (username: string, password: string) => {
    const data = await auth.login(username, password)
    setUser(data.user as unknown as User)
  }, [])

  const logout = useCallback(async () => {
    const refresh = localStorage.getItem('fabrixe_refresh') ?? undefined
    await auth.logout(refresh).catch(() => {})
    setUser(null)
  }, [])

  return (
    <AuthContext.Provider value={{ user, loading, isAuthenticated: !!user, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuthContext(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuthContext must be used inside <AuthProvider>')
  return ctx
}
