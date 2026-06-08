<template>
<Teleport to="body">
<div v-if="visible" class="modal-overlay" @click.self="visible=false">
<div class="wiki-modal" style="width:750px;max-width:98vw">
<div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">⚔️ 武器库</span><div class="top-bar-spacer"/><button class="modal-close" @click="visible=false">✕</button></div></header><div class="gold-divider"/>
<div class="wiki-body">
  <div style="display:flex;flex-wrap:wrap;justify-content:center;gap:4px">
    <WeaponCard v-for="eq in weapons" :key="eq.id" :equip="eq" @equip="doEquip(eq)" />
  </div>
</div>
</div>
</div>
</Teleport>
</template>
<script setup lang="ts">
import { generateEquip, realmNames, tierInfo } from '@/composables/useEquipmentData'
import type { Equipment } from '@/composables/useEquipmentData'

const visible=ref(false),weapons=ref<Equipment[]>([])
const {token,playerId}=useAuth()

// 初始化时就生成 50 把演示武器
for(let realm=1; realm<=10; realm++){
  for(let t=0; t<5; t++){
    const tk = tierInfo[t]?.key || 'human'
    weapons.value.push(generateEquip(realm, tk, 'weapon'))
  }
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

defineExpose({open:(v:boolean)=>{visible.value=v}})
</script>