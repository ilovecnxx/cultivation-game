// ============================================================
// 玩家状态 Store
// ============================================================
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiFetch, getAccessToken } from '@/core/api'
import { wsClient } from '@/core/network/WebSocketClient'

export interface PlayerInfo {
  id: number
  name: string
  nickname: string
  realmName: string
  realmStage: string
  realmLevel: number
  realmID: number
  spiritName: string
  qualityName: string
  rootQuality: number
  gender: string
  level: number
  power: number
  hp: number
  maxHp: number
  mp: number
  maxMp: number
  attack: number
  defense: number
  speed: number
  critRate: number
  critDmg: number
  hit: number
  dodge: number
  money: number
  bindMoney: number
  exp: number
  maxExp: number
}

// 辅助函数：获取短时效的 WebSocket 连接 Token
async function acquireWsToken(): Promise<string | null> {
  // 优先尝试通过 refresh 端点获取新 token（缩短 URL 泄露窗口）
  const refreshToken = sessionStorage.getItem('refresh_token')
  if (!refreshToken) return null

  try {
    const res = await fetch('/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
    if (!res.ok) return null
    const data = await res.json()
    return data.access_token || null
  } catch {
    return null
  }
}

export const usePlayerStore = defineStore('player', () => {
  // 状态
  const player = ref<PlayerInfo | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const onlineCount = ref(0)
  const registeredCount = ref(0)

  // 计算属性
  const hpPercent = computed(() => {
    if (!player.value) return 0
    return Math.round((player.value.hp / player.value.maxHp) * 100)
  })

  const mpPercent = computed(() => {
    if (!player.value) return 0
    return Math.round((player.value.mp / player.value.maxMp) * 100)
  })

  const isLoggedIn = computed(() => !!player.value)

  const realmDisplay = computed(() => {
    if (!player.value) return '凡人'
    return `${player.value.realmName} ${player.value.realmStage}`
  })

  // 动作
  async function fetchPlayerInfo(playerId: number): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const data = await apiFetch<{ code: number; data: PlayerInfo }>(`/api/v1/player/${playerId}`)
      if (data.code === 0) {
        player.value = data.data
      } else {
        error.value = data.data?.toString() || '获取玩家信息失败'
      }
    } catch (e: any) {
      error.value = e.message || '获取玩家信息失败'
    } finally {
      loading.value = false
    }
  }

  function setPlayerInfo(info: PlayerInfo): void {
    player.value = info
  }

  function updatePlayerStats(stats: Partial<PlayerInfo>): void {
    if (player.value) {
      Object.assign(player.value, stats)
    }
  }

  function setOnlineCount(count: number): void {
    onlineCount.value = count
  }

  function setRegisteredCount(count: number): void {
    registeredCount.value = count
  }

  function clearPlayer(): void {
    player.value = null
    loading.value = false
    error.value = null
  }

  // WebSocket 连接
  //
  // 安全说明：Token 出现在 WebSocket URL 查询参数中，可能被代理服务器/日志记录。
  // 生产环境建议：
  //   1. 后端增加 POST /auth/ws-ticket 接口，返回 30 秒有效期的单次连接凭证
  //   2. 前端用此短效凭证替代 Access Token 建立 WS 连接
  //   3. 凭证使用后立即失效，降低 URL 泄露风险
  async function connectWebSocket(): Promise<void> {
    const token = getAccessToken()
    if (!token) throw new Error('No access token')

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'

    // 使用专用 WS token 替代完整 Access Token（通过刷新获取短时效 token）
    // 这样即使 URL 被记录，泄露的 token 生命周期短
    const wsToken = await acquireWsToken()
    const wsUrl = `${protocol}//${window.location.host}/ws?token=${wsToken || token}`

    wsClient.setHandlers({
      onOpen: () => console.log('[WS] connected'),
      onClose: (code, reason) => console.log('[WS] closed', code, reason),
      onError: (err) => console.error('[WS] error', err),
      onMessage: (msgId, data) => {
        // 处理服务端推送消息
        console.debug('[WS] message', msgId, data)
      },
      onReconnect: (attempt) => console.log('[WS] reconnecting', attempt),
    })

    await wsClient.connect(wsUrl, token)
  }

  function disconnectWebSocket(): void {
    wsClient.disconnect()
  }

  return {
    // 状态
    player,
    loading,
    error,
    onlineCount,
    registeredCount,
    // 计算属性
    hpPercent,
    mpPercent,
    isLoggedIn,
    realmDisplay,
    // 动作
    fetchPlayerInfo,
    setPlayerInfo,
    updatePlayerStats,
    setOnlineCount,
    setRegisteredCount,
    clearPlayer,
    connectWebSocket,
    disconnectWebSocket,
  }
})
