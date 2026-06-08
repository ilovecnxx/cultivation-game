<template>
  <Teleport to="body">
    <div v-if="show" class="modal-overlay" @click.self="$emit('close')">
      <div class="map-modal">
        <div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">🗺️ 修仙世界 · 地图</span><div class="top-bar-spacer"/><button class="modal-close" @click="$emit('close')">✕</button></div></header><div class="gold-divider"/>
        <div class="map-body-wrap">
          <div class="map-legend">
            <span class="ml-item"><span class="ml-dot unlocked" /> 可进入</span>
            <span class="ml-item"><span class="ml-dot locked" /> 境界未达</span>
          </div>
          <div v-for="region in mapRegions" :key="region.name" class="map-region" :class="{locked:playerRealmId<region.minRealm}">
            <div class="mr-header">
              <span class="mr-icon">{{ region.icon }}</span><span class="mr-name">{{ region.name }}</span>
              <span class="mr-realm">{{ realmNames[region.minRealm] }}·{{ realmNames[region.maxRealm] }}</span>
              <span v-if="playerRealmId<region.minRealm" class="mr-lock">🔒 需{{ realmNames[region.minRealm] }}期</span>
            </div>
            <div class="mr-locations">
              <div v-for="loc in region.locations" :key="loc.key" class="mr-loc" :class="{locked:playerRealmId<loc.minRealm,current:currentLoc===loc.key}" @click="activeLoc=activeLoc===loc.key?null:loc.key">
                <span class="mrl-icon">{{ loc.icon }}</span><span class="mrl-name">{{ loc.name }}</span>
                <span v-if="playerRealmId<loc.minRealm" class="mrl-lock">🔒</span>
                <div v-if="activeLoc===loc.key" class="mrl-detail">
                  <p>{{ loc.desc }}</p>
                  <div class="mrl-info"><span>最低境界：{{ realmNames[loc.minRealm] }}{{ loc.minStage }}期</span><span>怪物：{{ loc.monsters }}</span></div>
                  <button v-if="playerRealmId>=loc.minRealm&&currentLoc!==loc.key" class="map-enter-btn" @click.stop="emit('enter',loc)">进入</button>
                  <button v-if="playerRealmId>=loc.minRealm&&playerHp>0" class="map-fight-btn" @click.stop="emit('fight',loc)">⚔️ 挑战</button>
                  <span v-else-if="currentLoc===loc.key" class="map-current-badge">📍 当前所在地</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
defineProps<{ show: boolean; playerRealmId: number; playerHp: number; currentLoc: string }>()
const emit = defineEmits(['close','enter','fight'])
import { mapRegions, realmNames } from '@/composables/useGameData'
const activeLoc = ref('')
</script>