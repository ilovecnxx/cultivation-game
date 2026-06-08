// ============================================================
// 战斗系统 Store
// ============================================================
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiFetch } from '@/core/api'

export interface BattleState {
  id: string
  state: string
  turnNumber: number
  playerHp: number
  playerMaxHp: number
  enemyHp: number
  enemyMaxHp: number
  logs: string[]
}

export const useCombatStore = defineStore('combat', () => {
  const currentBattle = ref<BattleState | null>(null)
  const loading = ref(false)

  async function startBattle(instanceId: number, monsterId: number): Promise<void> {
    loading.value = true
    try {
      const res = await apiFetch<{ code: number; data: BattleState }>('/api/v1/combat/start', {
        method: 'POST',
        body: JSON.stringify({ instance_id: instanceId, monster_id: monsterId }),
      })
      if (res.code === 0) {
        currentBattle.value = res.data
      }
    } finally {
      loading.value = false
    }
  }

  function clearBattle(): void {
    currentBattle.value = null
  }

  return { currentBattle, loading, startBattle, clearBattle }
})
