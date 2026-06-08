// 认证工具 — 统一 getToken/getPID，不再到处写 localStorage
export function useAuth() {
  const token = ref(localStorage.getItem('token') || '')
  const playerId = ref(localStorage.getItem('player_id') || '0')
  const refreshToken = ref(localStorage.getItem('refresh_token') || '')

  function setToken(t: string) { token.value = t; localStorage.setItem('token', t) }
  function setPlayerId(pid: string) { playerId.value = pid; localStorage.setItem('player_id', pid) }
  function setRefreshToken(rt: string) { refreshToken.value = rt; localStorage.setItem('refresh_token', rt) }
  function clearAuth() { token.value = ''; playerId.value = '0'; refreshToken.value = ''; ['token','player_id','refresh_token'].forEach(k => localStorage.removeItem(k)) }
  function isLoggedIn() { return !!token.value }

  return { token, playerId, refreshToken, setToken, setPlayerId, setRefreshToken, clearAuth, isLoggedIn }
}
