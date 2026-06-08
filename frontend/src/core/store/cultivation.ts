// ============================================================
// 修炼系统 Store
// ============================================================
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiFetch } from '@/core/api'

export interface CultivationState {
  isMeditating: boolean
  expPerHour: number
  currentExp: number
  maxExp: number
  techniqueName: string
  pillBonuses: string[]
}

export const useCultivationStore = defineStore('cultivation', () => {
  const state = ref<CultivationState>({
    isMeditating: false,
    expPerHour: 0,
    currentExp: 0,
    maxExp: 0,
    techniqueName: '',
    pillBonuses: [],
  })
  const loading = ref(false)

  async function startMeditation(techniqueId: number, duration: number): Promise<void> {
    loading.value = true
    try {
      const res = await apiFetch<{ code: number; data: CultivationState }>('/api/v1/cultivate', {
        method: 'POST',
        body: JSON.stringify({ technique_id: techniqueId, duration }),
      })
      if (res.code === 0) {
        state.value = res.data
      }
    } finally {
      loading.value = false
    }
  }

  async function breakthrough(assistElixirs: number[]): Promise<void> {
    loading.value = true
    try {
      await apiFetch('/api/v1/breakthrough', {
        method: 'POST',
        body: JSON.stringify({ assist_elixirs: assistElixirs }),
      })
    } finally {
      loading.value = false
    }
  }

  return { state, loading, startMeditation, breakthrough }
})
