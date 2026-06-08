<template>
<Teleport to="body">
<div v-if="visible" class="modal-overlay" @click.self="visible=false">
<div class="wiki-modal" style="width:700px;max-width:95vw">
<div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">💰 交易行</span><div class="top-bar-spacer"/><button class="modal-close" @click="visible=false">✕</button></div></header><div class="gold-divider"/>
<div class="wiki-tabs"><button v-for="t in tabs" :key="t.key" class="modal-tab" :class="{active:tab===t.key}" @click="tab=t.key;load()">{{ t.label }}</button></div>
<div class="wiki-body">
<div v-if="loading" style="text-align:center;padding:30px;color:rgba(255,255,255,.3)">加载中...</div>
<div v-else-if="tab==='market'">
<table class="wiki-table"><thead><tr><th>物品</th><th>卖家</th><th>单价</th><th>数量</th><th>操作</th></tr></thead><tbody>
<tr v-for="l in listings" :key="l.id"><td>{{ l.item_name }}</td><td>{{ l.seller_name }}</td><td>{{ fmt(l.unit_price) }}</td><td>{{ l.quantity }}</td><td><van-button size="small" type="primary" @click="buyItem(l)">购买</van-button></td></tr></tbody></table>
<div v-if="listings.length===0" style="text-align:center;padding:30px;color:rgba(255,255,255,.3)">暂无挂单</div>
</div>
<div v-else-if="tab==='mine'">
<table class="wiki-table"><thead><tr><th>物品</th><th>单价</th><th>数量</th><th>状态</th><th>操作</th></tr></thead><tbody>
<tr v-for="l in myListings" :key="l.id"><td>{{ l.item_name }}</td><td>{{ fmt(l.unit_price) }}</td><td>{{ l.quantity }}</td><td :style="{color:l.status==='active'?'#6bcb77':'#888'}">{{ l.status==='active'?'销售中':'已售' }}</td><td><van-button v-if="l.status==='active'" size="small" plain type="danger" @click="cancelListing(l.id)">下架</van-button></td></tr></tbody></table>
<div v-if="myListings.length===0" style="text-align:center;padding:30px;color:rgba(255,255,255,.3)">暂无挂单</div>
</div>
</div>
</div>
</div>
</Teleport>
</template>
<script setup lang="ts">
const visible=ref(false),tab=ref('market'),listings=ref<any[]>([]),myListings=ref<any[]>([]),loading=ref(false)
const tabs=[{key:'market',label:'市场'},{key:'mine',label:'我的挂单'}]
const {token,playerId}=useAuth()
async function load(){loading.value=true;try{const r=await fetch('/api/v1/trade/listings',{headers:{Authorization:'Bearer '+token.value}});const d=await r.json();listings.value=d.data||[];const r2=await fetch('/api/v1/trade/my-listings',{headers:{Authorization:'Bearer '+token.value}});const d2=await r2.json();myListings.value=d2.data||[]}catch{}finally{loading.value=false}}
async function buyItem(l:any){await fetch('/api/v1/trade/buy',{method:'POST',headers:{'Content-Type':'application/json',Authorization:'Bearer '+token.value},body:JSON.stringify({listing_id:l.id})});load()}
async function cancelListing(id:number){await fetch('/api/v1/trade/cancel',{method:'POST',headers:{'Content-Type':'application/json',Authorization:'Bearer '+token.value},body:JSON.stringify({listing_id:id})});load()}
function fmt(n:number):string{return n>=10000?(n/10000).toFixed(1)+'万':n.toLocaleString()}
watch(visible,v=>{if(v)load()})
defineExpose({open:(v:boolean)=>{visible.value=v}})
</script>