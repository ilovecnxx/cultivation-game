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
  async function connectWebSocket(): Promise<void> {
    const token = getAccessToken()
    if (!token) throw new Error('No access token')

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws?token=${token}`

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
