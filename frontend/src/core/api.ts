// ============================================================
// 统一 API 请求工具
// 封装 fetch，自动携带 token、处理 401 跳转、错误提示
// ============================================================

import router from '@/router'

/** localStorage token 键名 */
const TOKEN_KEY = 'token'

/** 获取认证头 */
function getAuthHeaders(): Record<string, string> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  return headers
}

/** token 过期，跳转登录页 */
function redirectToLogin(): never {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem('player_id')
  localStorage.removeItem('refresh_token')
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
    ...getAuthHeaders(),
    ...(options.headers as Record<string, string> || {}),
  }

  let res: Response
  try {
    res = await fetch(url, { ...options, headers })
  } catch {
    throw new Error('网络连接失败，请检查服务器是否运行')
  }

  // 401 → 登录过期
  if (res.status === 401) {
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
