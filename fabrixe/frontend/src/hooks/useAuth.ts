import { useState, useEffect, useCallback } from 'react'
import { auth } from '../lib/api'
import type { User } from '../types'

interface AuthState {
  user: User | null
  loading: boolean
  error: string | null
}

export function useAuth() {
  const [state, setState] = useState<AuthState>({
    user: null,
    loading: true,
    error: null,
  })

  useEffect(() => {
    if (!auth.isAuthenticated()) {
      setState({ user: null, loading: false, error: null })
      return
    }
    auth.me()
      .then(user => setState({ user: user as User, loading: false, error: null }))
      .catch(() => {
        auth.clearTokens()
        setState({ user: null, loading: false, error: null })
      })
  }, [])

  const login = useCallback(async (username: string, password: string): Promise<User> => {
    setState(s => ({ ...s, loading: true, error: null }))
    try {
      const data = await auth.login(username, password)
      const user = data.user as unknown as User
      setState({ user, loading: false, error: null })
      return user
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Login failed'
      setState(s => ({ ...s, loading: false, error: msg }))
      throw err
    }
  }, [])

  const logout = useCallback(async () => {
    const refresh = localStorage.getItem('fabrixe_refresh') ?? undefined
    await auth.logout(refresh)
    setState({ user: null, loading: false, error: null })
  }, [])

  return {
    user: state.user,
    loading: state.loading,
    error: state.error,
    isAuthenticated: !!state.user,
    login,
    logout,
  }
}
