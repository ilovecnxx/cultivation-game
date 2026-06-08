<template>
<Teleport to="body">
<div v-if="visible" class="modal-overlay" @click.self="visible=false">
<div class="wiki-modal" style="width:700px;max-width:95vw">
<div class="gold-divider"/>
<header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">🎒 背包</span><div class="top-bar-spacer"/><button class="modal-close" @click="visible=false">✕</button></div></header>
<div class="gold-divider"/>
<div class="wiki-body">
<div v-if="items.length===0" style="text-align:center;padding:40px;color:rgba(255,255,255,.3)">背包空空如也</div>
<div v-else style="display:grid;grid-template-columns:repeat(auto-fill,minmax(200px,1fr));gap:8px">
<div v-for="item in items" :key="item.id" class="pill-card" style="cursor:pointer" @click="selected=selected?.id===item.id?null:item">
<div class="pc-top"><span class="pc-icon-lg">{{ item.icon||'📦' }}</span><div class="pc-badges"><span class="pc-tier-badge">{{ item.quality_name||'普通' }}</span><span class="pc-cat-badge">×{{ item.quantity }}</span></div></div>
<div class="pc-name-lg">{{ item.name }}</div>
<div class="pc-desc-lg">{{ item.desc||item.category||'' }}</div>
<div class="pc-actions-lg" v-if="item.category==='消耗品'||item.type==='consumable'"><van-button size="small" type="primary" @click.stop="useItem(item)">使用</van-button></div>
</div>
</div>
<div v-if="selected" style="margin-top:16px;padding:12px;background:rgba(255,255,255,.05);border-radius:8px">
<h4 style="color:#d4a843;margin:0 0 8px">{{ selected.name }}</h4>
<p style="color:rgba(255,255,255,.6);font-size:13px;margin:0" v-if="selected.desc">{{ selected.desc }}</p>
</div>
</div>
</div>
</div>
</Teleport>
</template>
<script setup lang="ts">
const visible=ref(false),items=ref<any[]>([]),selected=ref<any>(null)
const {token,playerId}=useAuth()
async function load(){try{const r=await fetch('/api/v1/player/'+playerId.value+'/inventory',{headers:{Authorization:'Bearer '+token.value}});const d=await r.json();items.value=d.data||[]}catch{}}
async function useItem(item:any){await fetch('/api/v1/player/'+playerId.value+'/items/use',{method:'POST',headers:{'Content-Type':'application/json',Authorization:'Bearer '+token.value},body:JSON.stringify({item_id:item.id})});load()}
watch(visible,v=>{if(v)load()})
defineExpose({open:(v:boolean)=>visible.value=v})
</script>