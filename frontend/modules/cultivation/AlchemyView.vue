<template>
<Teleport to="body">
<div v-if="visible" class="modal-overlay" @click.self="visible=false">
<div class="wiki-modal" style="width:700px;max-width:95vw">
<div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">🔥 炼丹</span><div class="top-bar-spacer"/><button class="modal-close" @click="visible=false">✕</button></div></header><div class="gold-divider"/>
<div class="wiki-body">
<div v-if="loading" style="text-align:center;padding:40px;color:rgba(255,255,255,.3)">丹炉预热中...</div>
<div v-else>
<div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(220px,1fr));gap:10px">
<div v-for="r in recipes" :key="r.pill_key||r.id" class="pill-card" @click="craftPill(r)">
<div class="pc-top"><span class="pc-icon-lg">💊</span><div class="pc-badges"><span class="pc-tier-badge">{{ r.quality_name||r.tier||'普通' }}</span></div></div>
<div class="pc-name-lg">{{ r.name }}</div>
<div class="pc-desc-lg">{{ r.effect||r.desc||'' }}</div>
<div class="pc-actions-lg"><van-button size="small" type="primary" @click.stop="craftPill(r)">炼制</van-button></div>
</div>
</div>
<div v-if="craftResult" style="padding:16px;border-radius:8px;text-align:center;margin-top:16px" :style="{background:craftResult.success?'rgba(107,203,119,.08)':'rgba(229,57,53,.08)',border:'1px solid '+(craftResult.success?'#6bcb77':'#e53935')}">
<div style="font-size:20px;font-weight:900;margin-bottom:6px" :style="{color:craftResult.success?'#6bcb77':'#e53935'}">{{ craftResult.success?'🎉 炼制成功！':'💥 炼制失败' }}</div>
<div v-if="craftResult.success" style="color:#d4a843;font-size:15px">获得 {{ craftResult.pill||'丹药' }}</div>
<van-button size="small" round style="margin-top:8px" @click="craftResult=null">继续</van-button>
</div>
</div>
</div>
</div>
</div>
</Teleport>
</template>
<script setup lang="ts">
const visible=ref(false),loading=ref(false),recipes=ref<any[]>([]),craftResult=ref<any>(null)
const {token,playerId}=useAuth()
async function load(){loading.value=true;try{const r=await fetch('/api/v1/pills/recipes',{headers:{Authorization:'Bearer '+token.value}});const d=await r.json();recipes.value=d.data||[]}catch{}finally{loading.value=false}}
async function craftPill(recipe:any){try{const r=await fetch('/api/v1/player/'+playerId.value+'/pills/craft',{method:'POST',headers:{'Content-Type':'application/json',Authorization:'Bearer '+token.value},body:JSON.stringify({pill_key:recipe.pill_key||recipe.key})});const d=await r.json();craftResult.value={success:!!d.data?.success,pill:recipe.name}}catch{craftResult.value={success:false}}}
watch(visible,v=>{if(v)load()})
defineExpose({open:(v:boolean)=>{visible.value=v}})
</script>