/**
 * useAudio — Vue composable for easy audio access in components.
 * Provides reactive state and convenience methods wrapping AudioEngine.
 */

import { ref, computed, onMounted, onUnmounted, readonly } from 'vue'
import { AudioEngine } from '@/core/audio/AudioEngine'
import { SoundId } from '@/core/audio/soundDefinitions'
import type { MusicMode } from '@/core/audio/MusicGenerator'
import type { AudioState } from '@/core/audio/AudioEngine'

const engine = AudioEngine.getInstance()

/** Reactive audio state shared across all consumers */
const isInitialized = ref(false)
const sfxMuted = ref(false)
const musicMuted = ref(false)
const sfxVolume = ref(1)
const musicVolume = ref(1)
const masterVolume = ref(1)
const currentMusicMode = ref<string | null>(null)
const currentTrackName = ref('')

let unsub: (() => void) | null = null

function updateFromEngine(state: AudioState) {
  isInitialized.value = state.isInitialized
  sfxMuted.value = state.sfxMuted
  musicMuted.value = state.musicMuted
  sfxVolume.value = state.sfxVolume
  musicVolume.value = state.musicVolume
  masterVolume.value = state.masterVolume
  currentMusicMode.value = state.currentMusicMode as string | null
  currentTrackName.value = state.currentTrackName
}

export function useAudio() {
  // Subscribe to engine state changes
  onMounted(() => {
    if (!unsub) {
      // Sync initial state
      updateFromEngine(engine.getState())
      unsub = engine.subscribe(updateFromEngine)
    }
  })

  onUnmounted(() => {
    // Cleanup handled by Vue — the subscription persists as long as any
    // component using useAudio is alive. We manage this with a module-level
    // subscription to avoid duplicate listeners.
  })

  /** Play a sound effect by ID */
  function playSound(id: SoundId): void {
    engine.handleUserInteraction()
    engine.resume()
    engine.playSound(id)
  }

  /** Start background music for a mode */
  function playMusic(mode: MusicMode): void {
    engine.handleUserInteraction()
    engine.resume()
    engine.playMusic(mode)
  }

  /** Stop background music */
  function stopMusic(): void {
    engine.stopMusic()
  }

  /** Set volume by type */
  function setVolume(type: 'master' | 'sfx' | 'music', value: number): void {
    engine.setVolume(type, value)
  }

  /** Toggle mute for sfx or music */
  function toggleMute(type: 'sfx' | 'music'): void {
    engine.toggleMute(type)
  }

  /** Initialize audio engine (must be called from user gesture) */
  function initAudio(): void {
    engine.handleUserInteraction()
  }

  /** Get the raw engine instance for advanced usage */
  function getEngine(): AudioEngine {
    return engine
  }

  return {
    // Reactive state (readonly to prevent direct mutation)
    isInitialized: readonly(isInitialized),
    sfxMuted: readonly(sfxMuted),
    musicMuted: readonly(musicMuted),
    sfxVolume: readonly(sfxVolume),
    musicVolume: readonly(musicVolume),
    masterVolume: readonly(masterVolume),
    currentMusicMode: readonly(currentMusicMode),
    currentTrackName: readonly(currentTrackName),

    // Computed helpers
    isMuted: computed(() => ({
      sfx: sfxMuted.value,
      music: musicMuted.value,
    })),
    hasAudio: computed(() => isInitialized.value),

    // Methods
    playSound,
    playMusic,
    stopMusic,
    setVolume,
    toggleMute,
    initAudio,
    getEngine,
  }
}
