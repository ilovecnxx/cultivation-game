// ============================================================
// 统一 API 请求工具
// 封装 fetch，自动携带 token、处理 401 跳转、错误提示
//
// 安全说明：
// - Access Token 仅保存在内存中（页面刷新需要重新登录或使用 Refresh Token）
// - Refresh Token 保存在 sessionStorage（关闭标签页即清除）
// - 生产环境建议迁移到 httpOnly Cookie + CSRF Token 方案
// ============================================================

import router from '@/router'

/** 内存中的 Access Token（页面刷新后需要重新获取） */
let inMemoryAccessToken: string | null = null

/** sessionStorage 中 Refresh Token 的键名 */
const REFRESH_TOKEN_KEY = 'refresh_token'

/**
 * 设置 Access Token（仅保存在内存中）。
 * 页面刷新或关闭标签页后失效，有效防止 XSS 窃取。
 */
export function setAccessToken(token: string | null): void {
  inMemoryAccessToken = token
}

/**
 * 获取当前的 Access Token。
 * 优先从内存返回，如果没有则尝试从 sessionStorage 的 Refresh Token 刷新。
 */
export function getAccessToken(): string | null {
  return inMemoryAccessToken
}

/**
 * 检查是否已登录（内存中有 Access Token 或 sessionStorage 中有 Refresh Token）。
 */
export function isLoggedIn(): boolean {
  return !!inMemoryAccessToken || !!sessionStorage.getItem(REFRESH_TOKEN_KEY)
}

/**
 * 设置 Refresh Token（保存在 sessionStorage，关闭标签页即清除）。
 */
export function setRefreshToken(token: string | null): void {
  if (token) {
    sessionStorage.setItem(REFRESH_TOKEN_KEY, token)
  } else {
    sessionStorage.removeItem(REFRESH_TOKEN_KEY)
  }
}

/**
 * 获取 Refresh Token。
 */
export function getRefreshToken(): string | null {
  return sessionStorage.getItem(REFRESH_TOKEN_KEY)
}

/** 清除所有认证信息 */
export function clearAuth(): void {
  inMemoryAccessToken = null
  sessionStorage.removeItem(REFRESH_TOKEN_KEY)
  localStorage.removeItem('player_id')
}

/** token 过期，跳转登录页 */
function redirectToLogin(): never {
  clearAuth()
  router.push('/login')
  throw new Error('登录已过期，请重新登录')
}

// 简单 Toast 实现（不依赖 eventBus，避免耦合）
let toastTimer: ReturnType<typeof setTimeout> | null = null

/**
 * 显示 Toast 提示
 */
export function showToast(message: string, type: 'info' | 'success' | 'warning' | 'error' = 'info', duration = 3500) {
  // 移除已有 toast
  const existing = document.querySelector('.app-toast-global')
  if (existing) existing.remove()
  if (toastTimer) clearTimeout(toastTimer)

  const el = document.createElement('div')
  el.className = `app-toast-global toast-${type}`
  el.textContent = message
  document.body.appendChild(el)

  // 触发动画
  requestAnimationFrame(() => el.classList.add('show'))

  toastTimer = setTimeout(() => {
    el.classList.remove('show')
    setTimeout(() => el.remove(), 300)
    toastTimer = null
  }, duration)
}

/**
 * 统一 API 请求
 *
 * @param url 完整 URL 路径（如 /api/v1/player/1 或 /auth/login）
 * @param options fetch 选项
 * @returns 解析后的 JSON 数据
 */
export async function apiFetch<T = any>(url: string, options: RequestInit = {}): Promise<T> {
  // 合并 headers
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
  }

  const token = getAccessToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  let res: Response
  try {
    res = await fetch(url, { ...options, headers })
  } catch {
    throw new Error('网络连接失败，请检查服务器是否运行')
  }

  // 401 → 尝试用 Refresh Token 刷新
  if (res.status === 401) {
    const refreshed = await tryRefreshToken()
    if (refreshed) {
      // 重试原请求（更新 token）
      headers['Authorization'] = `Bearer ${getAccessToken()}`
      res = await fetch(url, { ...options, headers })
      if (res.ok) {
        return await res.json() as T
      }
    }
    redirectToLogin()
  }

  // 尝试解析 JSON
  let body: any
  try {
    body = await res.json()
  } catch {
    body = {}
  }

  if (!res.ok) {
    throw new Error(body.error || body.message || body.msg || `请求失败 (${res.status})`)
  }

  return body as T
}

/**
 * 尝试用 Refresh Token 刷新 Access Token。
 * @returns 是否刷新成功
 */
async function tryRefreshToken(): Promise<boolean> {
  const refreshToken = getRefreshToken()
  if (!refreshToken) return false

  try {
    const res = await fetch('/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
    if (!res.ok) return false

    const data = await res.json()
    if (data.access_token) {
      setAccessToken(data.access_token)
      // 如果返回了新的 Refresh Token，轮换存储
      if (data.refresh_token) {
        setRefreshToken(data.refresh_token)
      }
      return true
    }
    return false
  } catch {
    return false
  }
}

/**
 * API 请求包装：自动显示 Toast 错误
 */
export async function apiFetchWithToast<T = any>(url: string, options: RequestInit = {}): Promise<T | null> {
  try {
    return await apiFetch<T>(url, options)
  } catch (e: any) {
    showToast(e.message || '未知错误', 'error')
    return null
  }
}
