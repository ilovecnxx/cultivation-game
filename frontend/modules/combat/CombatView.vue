<template>
<Teleport to="body">
<div v-if="visible" class="modal-overlay" @click.self="visible=false">
<div class="wiki-modal" style="width:600px;max-width:95vw">
<div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">⚔️ 副本</span><div class="top-bar-spacer"/><button class="modal-close" @click="visible=false">✕</button></div></header><div class="gold-divider"/>
<div class="wiki-body">
<div v-if="loading" style="text-align:center;padding:30px;color:rgba(255,255,255,.3)">加载中...</div>
<div v-else style="display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:10px">
<div v-for="d in dungeons" :key="d.id" class="pill-card" :class="{locked:playerRealm<d.min_realm}">
<div class="pc-top"><span class="pc-icon-lg">{{ d.icon||'🏯' }}</span><div class="pc-badges"><span class="pc-tier-badge">{{ d.difficulty||'普通' }}</span><span class="pc-cat-badge">Lv.{{ d.min_realm||1 }}</span></div></div>
<div class="pc-name-lg">{{ d.name }}</div>
<div class="pc-desc-lg">{{ d.desc||'' }}</div>
<div class="pc-actions-lg"><van-button size="small" type="primary" @click.stop="startDungeon(d)">进入</van-button></div>
</div>
</div>
</div>
</div>
</div>
</Teleport>
</template>
<script setup lang="ts">
const visible=ref(false),loading=ref(false),dungeons=ref<any[]>([])
const {token,playerId}=useAuth()
const playerRealm=ref(1)
async function load(){loading.value=true;try{const r=await fetch('/api/v1/combat/dungeons',{headers:{Authorization:'Bearer '+token.value}});const d=await r.json();dungeons.value=d.data||[];const r2=await fetch('/api/v1/player/'+playerId.value,{headers:{Authorization:'Bearer '+token.value}});const d2=await r2.json();if(d2.data)playerRealm.value=d2.data.realmId||1}catch{}finally{loading.value=false}}
async function startDungeon(d:any){try{const r=await fetch('/api/v1/combat/dungeon/start',{method:'POST',headers:{'Content-Type':'application/json',Authorization:'Bearer '+token.value},body:JSON.stringify({dungeon_id:d.id})});const data=await r.json();if(data.data)alert('副本战斗完成!')}catch{}}
watch(visible,v=>{if(v)load()})
defineExpose({open:(v:boolean)=>{visible.value=v}})
</script>