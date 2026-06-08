<template>
<Teleport to="body">
<div v-if="visible" class="modal-overlay" @click.self="visible=false">
<div class="wiki-modal" style="width:500px;max-width:95vw">
<div class="gold-divider"/>
<header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">🏆 排行榜</span><div class="top-bar-spacer"/><button class="modal-close" @click="visible=false">✕</button></div></header>
<div class="gold-divider"/>
<div class="wiki-tabs">
<button v-for="t in tabs" :key="t.key" class="modal-tab" :class="{active:tab===t.key}" @click="tab=t.key;load()">{{ t.label }}</button>
</div>
<div class="wiki-body">
<div v-if="loading" style="text-align:center;padding:30px;color:rgba(255,255,255,.3)">加载中...</div>
<table v-else class="wiki-table"><thead><tr><th>排名</th><th>昵称</th><th>境界</th><th>数值</th></tr></thead><tbody>
<tr v-for="(r,i) in list" :key="r.player_id" :class="{highlight:i<3}">
<td class="tc"><b :style="{color:['#ffd700','#c0c0c0','#cd7f32'][i]||'#fff'}">{{ i+1 }}</b></td>
<td>{{ r.nickname }}</td><td>{{ r.realm_name||'-' }}</td><td><b>{{ fmt(r.score) }}</b></td>
</tr></tbody></table>
</div>
</div>
</div>
</Teleport>
</template>
<script setup lang="ts">
const visible=ref(false),tab=ref('combat'),list=ref<any[]>([]),loading=ref(false)
const tabs=[{key:'combat',label:'战力'},{key:'realm',label:'境界'},{key:'wealth',label:'财富'}]
const {token}=useAuth()
async function load(){loading.value=true;try{const r=await fetch('/api/v1/ranking/'+tab.value+'/top',{headers:{Authorization:'Bearer '+token.value}});const d=await r.json();list.value=d.data||[]}catch{}finally{loading.value=false}}
function fmt(n:number):string{return n>=10000?(n/10000).toFixed(1)+'万':n.toLocaleString()}
defineExpose({open:(v:boolean)=>visible.value=v})
</script>
