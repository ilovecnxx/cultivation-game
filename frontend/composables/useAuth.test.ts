// useAuth composable 测试
import { describe, it, expect, beforeEach } from 'vitest'

// Mock localStorage
const store: Record<string, string> = {}
const mockLS = {
  getItem: (k: string) => store[k] || null,
  setItem: (k: string, v: string) => { store[k] = v },
  removeItem: (k: string) => { delete store[k] },
}
Object.defineProperty(globalThis, 'localStorage', { value: mockLS })

describe('useAuth', () => {
  beforeEach(() => {
    Object.keys(store).forEach(k => delete store[k])
    store.token = 'test-token-abc'
    store.player_id = '42'
    store.refresh_token = 'rt-xyz'
  })

  // 需要动态 import 因为 composable 依赖 localStorage
  it('should read token from localStorage', async () => {
    const { useAuth } = await import('./useAuth')
    const auth = useAuth()
    expect(auth.token.value).toBe('test-token-abc')
    expect(auth.playerId.value).toBe('42')
    expect(auth.isLoggedIn()).toBe(true)
  })

  it('should clear auth state', async () => {
    const { useAuth } = await import('./useAuth')
    const auth = useAuth()
    auth.clearAuth()
    expect(auth.token.value).toBe('')
    expect(auth.isLoggedIn()).toBe(false)
  })

  it('should set new token', async () => {
    const { useAuth } = await import('./useAuth')
    const auth = useAuth()
    auth.setToken('new-token')
    expect(auth.token.value).toBe('new-token')
    expect(store.token).toBe('new-token')
  })

  it('should return false when not logged in', async () => {
    store.token = ''
    const { useAuth } = await import('./useAuth')
    const auth = useAuth()
    expect(auth.isLoggedIn()).toBe(false)
  })
})