<template>
<Teleport to="body">
<div v-if="visible" class="modal-overlay" @click.self="visible=false">
<div class="wiki-modal" style="width:650px;max-width:95vw">
<div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">⚒️ 锻造</span><div class="top-bar-spacer"/><button class="modal-close" @click="visible=false">✕</button></div></header><div class="gold-divider"/>
<div class="wiki-body">
<div style="margin-bottom:12px;display:flex;gap:6px;flex-wrap:wrap;align-items:center;padding:8px;background:rgba(255,255,255,.03);border-radius:8px">
<span style="color:#d4a843;font-size:12px;font-weight:700">品阶:</span>
<button v-for="t in weaponTiers" :key="t.key" class="modal-tab" :class="{active:selectedTier===t.key}" @click="selectedTier=t.key" :disabled="playerRealm<t.minRealm" :style="{opacity:playerRealm<t.minRealm?0.3:1}">{{ t.name }}</button>
<span style="color:#d4a843;font-size:12px;font-weight:700;margin-left:8px">类型:</span>
<button v-for="(w,i) in weaponTypes" :key="w.name" class="modal-tab" :class="{active:selectedType===i}" @click="selectedType=i">{{ w.icon }} {{ w.name }}</button>
</div>
<div v-if="result" style="padding:20px;text-align:center;background:rgba(0,0,0,.2);border-radius:8px;margin-bottom:12px">
<div style="font-size:48px;margin-bottom:8px">{{ result.icon }}</div>
<div style="font-size:22px;font-weight:900;color:#fff;margin-bottom:4px">{{ result.name }}</div>
<div style="font-size:14px;color:#d4a843;margin-bottom:4px">⚔️ 攻击力 +{{ result.attack }} · {{ result.element }}系</div>
<div v-if="result.subStats.length" style="display:flex;gap:6px;justify-content:center;flex-wrap:wrap;margin-top:8px">
<span v-for="s in result.subStats" style="font-size:11px;padding:2px 8px;border-radius:4px;background:rgba(212,168,67,.1);color:#d4a843">{{ s.name }} {{ s.display }}</span>
</div>
<div style="margin-top:16px;display:flex;gap:8px;justify-content:center">
<van-button type="primary" round size="small" @click="equipWeapon(result)">装备</van-button>
<van-button plain round size="small" @click="result=null">放弃</van-button>
</div>
</div>
<div v-else style="text-align:center;padding:40px">
<div style="font-size:48px;margin-bottom:16px">🔨</div>
<div style="font-size:14px;color:#d4a843;margin-bottom:24px">选择品阶和类型，锻造你的武器</div>
<van-button type="primary" round @click="forge">开始锻造</van-button>
<p style="margin-top:16px;font-size:11px;color:rgba(255,255,255,.3)"> {{ weaponTiers.find(t=>t.key===selectedTier)?.name }} · {{ weaponTypes[selectedType]?.name }} · 消耗灵石</p>
</div>
</div>
</div>
</div>
</Teleport>
</template>
<script setup lang="ts">
import { weaponTiers, weaponTypes, generateWeapon, type Weapon } from '@/composables/useWeaponSystem'
const visible=ref(false),selectedTier=ref('human'),selectedType=ref(0),result=ref<Weapon|null>(null),playerRealm=ref(1)
const {token,playerId}=useAuth()
async function loadRealm(){try{const r=await fetch('/api/v1/player/'+playerId.value,{headers:{Authorization:'Bearer '+token.value}});const d=await r.json();if(d.data)playerRealm.value=d.data.realmId||1}catch{}}
function forge(){result.value=generateWeapon(playerRealm.value,selectedType.value,selectedTier.value)}
async function equipWeapon(w:Weapon){try{await fetch('/api/v1/player/'+playerId.value+'/equipment/craft',{method:'POST',headers:{'Content-Type':'application/json',Authorization:'Bearer '+token.value},body:JSON.stringify({slot:'weapon',name:w.name,attack:w.attack})});visible.value=false}catch{}}
watch(visible,v=>{if(v)loadRealm()})
defineExpose({open:(v:boolean)=>{visible.value=v;result.value=null}})
</script>