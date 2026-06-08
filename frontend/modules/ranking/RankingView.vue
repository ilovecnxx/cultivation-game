<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="visible=false">
      <div class="wiki-modal" style="width:500px;max-width:95vw">
        <div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">🏆 排行榜</span><div class="top-bar-spacer"/><button class="modal-close" @click="visible=false">✕</button></div></header><div class="gold-divider"/>
        <div class="wiki-body">
          <van-tabs v-model:active="tab" color="#d4a843" @change="load">
            <van-tab title="战力" name="combat" /><van-tab title="境界" name="realm" /><van-tab title="财富" name="wealth" />
          </van-tabs>
          <div v-if="loading" style="text-align:center;padding:30px;color:rgba(255,255,255,.3)">加载中...</div>
          <div v-else v-for="(r,i) in list" :key="r.player_id" style="display:flex;align-items:center;gap:10px;padding:8px 0;border-bottom:1px solid rgba(255,255,255,.05)">
            <span style="font-size:18px;font-weight:700;width:28px;text-align:center" :style="{color:['#ffd700','#c0c0c0','#cd7f32'][i]||'#888'}">{{ i+1 }}</span>
            <span style="flex:1;font-size:14px;color:#fff">{{ r.nickname }}</span>
            <span style="font-size:12px;color:#d4a843">{{ r.realm_name||'' }}</span>
            <span style="font-size:14px;font-weight:700;color:#fff">{{ fmt(r.score) }}</span>
          </div>
        </div>
      </div>
    </div>
</template>
<script setup lang="ts">
const visible=ref(false),tab=ref('combat'),list=ref<any[]>([]),loading=ref(false)
const {token}=useAuth()
async function load(){loading.value=true;try{const r=await fetch('/api/v1/ranking/'+tab.value+'/top',{headers:{Authorization:'Bearer '+token.value}});const d=await r.json();list.value=d.data||[]}catch{}finally{loading.value=false}}
function fmt(n:number):string{return n>=10000?(n/10000).toFixed(1)+'万':n.toLocaleString()}
defineExpose({open:(v:boolean)=>visible.value=v})
</script>