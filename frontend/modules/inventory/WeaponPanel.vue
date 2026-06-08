<template>
<Teleport to="body">
<div v-if="visible" class="modal-overlay" @click.self="visible=false">
<div class="wiki-modal" style="width:750px;max-width:98vw">
<div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">⚔️ 武器库</span><div class="top-bar-spacer"/><button class="modal-close" @click="visible=false">✕</button></div></header><div class="gold-divider"/>
<div class="wiki-body">
  <div v-if="loading" style="text-align:center;padding:40px;color:rgba(255,255,255,.3)">加载中...</div>
  <div v-else style="display:flex;flex-wrap:wrap;justify-content:center;gap:4px">
    <WeaponCard
      v-for="eq in weapons"
      :key="eq.id"
      :equip="eq"
      :show-actions="true"
      @equip="doEquip(eq)"
      @detail="selectedWeapon=eq"
    />
  </div>
  <div v-if="!loading && weapons.length===0" style="text-align:center;padding:40px;color:rgba(255,255,255,.3)">
    <div style="font-size:48px;margin-bottom:16px">🔨</div>
    暂无武器 — 去锻造一把吧
    <div style="margin-top:12px"><van-button type="primary" round @click="visible=false;$emit('openForge')">前往锻造</van-button></div>
  </div>
</div>
</div>
</div>
</Teleport>
</template>
<script setup lang="ts">
import { generateEquip, realmNames, tierInfo } from '@/composables/useEquipmentData'
import type { Equipment } from '@/composables/useEquipmentData'

const visible=ref(false),loading=ref(false),weapons=ref<Equipment[]>([]),selectedWeapon=ref<Equipment|null>(null)
const {token,playerId}=useAuth()

async function load(){
  loading.value=true
  // 展示所有境界+品阶的武器卡片
  weapons.value = []
  for(let realm=1; realm<=10; realm++){
    for(let t=0; t<5; t++){
      const tk = tierInfo[t]?.key || 'human'
      weapons.value.push(generateEquip(realm, tk, 'weapon'))
    }
  }
  loading.value=false
}

async function doEquip(eq:Equipment){
  try{
    await fetch('/api/v1/player/'+playerId.value+'/equipment/craft',{
      method:'POST',headers:{'Content-Type':'application/json',Authorization:'Bearer '+token.value},
      body:JSON.stringify({slot:'weapon',tier:eq.realm})
    })
    visible.value=false
  }catch{}
}

watch(visible,v=>{if(v)load()})
defineExpose({open:(v:boolean)=>{visible.value=v}})
</script>