<template>
<Teleport to="body">
<div v-if="visible" class="modal-overlay" @click.self="visible=false">
<div class="wiki-modal" style="width:600px;max-width:95vw">
<div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">👥 好友</span><div class="top-bar-spacer"/><button class="modal-close" @click="visible=false">✕</button></div></header><div class="gold-divider"/>
<div class="wiki-body">
<div style="margin-bottom:12px"><input v-model="searchQuery" placeholder="搜索道友..." style="width:100%;padding:8px 12px;border:1px solid rgba(212,168,67,.15);border-radius:6px;background:rgba(255,255,255,.04);color:#fff;font-size:13px;outline:none" @keyup.enter="doSearch"/></div>
<div v-if="searchResults.length" style="margin-bottom:12px"><div v-for="u in searchResults" :key="u.id" style="display:flex;align-items:center;gap:8px;padding:8px;background:rgba(255,255,255,.03);border-radius:6px;margin-bottom:4px"><span style="flex:1;color:#fff;font-size:13px">{{ u.nickname }}</span><van-button size="small" type="primary" @click="addFriend(u.id)">加好友</van-button></div></div>
<h4 style="color:#d4a843;margin:12px 0 8px">我的好友</h4>
<div v-if="loading" style="text-align:center;padding:20px;color:rgba(255,255,255,.3)">加载中...</div>
<div v-else-if="friends.length===0" style="text-align:center;padding:20px;color:rgba(255,255,255,.3)">暂无好友</div>
<div v-else v-for="f in friends" :key="f.id" style="display:flex;align-items:center;gap:10px;padding:8px;border-bottom:1px solid rgba(255,255,255,.05)"><span style="font-size:20px;width:36px;height:36px;border-radius:50%;background:linear-gradient(135deg,#d4a843,#b8860b);display:flex;align-items:center;justify-content:center;color:#fff;font-weight:700;flex-shrink:0">{{ f.nickname[0] }}</span><span style="flex:1;color:#fff;font-size:14px">{{ f.nickname }}</span><span style="font-size:11px;color:rgba(255,255,255,.4)">锻体{{ f.realm_stage||'?' }}期</span><van-button size="mini" plain type="danger" @click="removeFriend(f.id)">删除</van-button></div>
<h4 v-if="requests.length" style="color:#d4a843;margin:16px 0 8px">好友申请</h4>
<div v-for="r in requests" :key="r.id" style="display:flex;align-items:center;gap:8px;padding:6px 0;border-bottom:1px solid rgba(255,255,255,.03)"><span style="flex:1;color:#fff;font-size:13px">{{ r.from_name }}</span><van-button size="small" type="primary" @click="acceptFriend(r.id)">接受</van-button><van-button size="small" plain type="danger" @click="rejectFriend(r.id)">拒绝</van-button></div>
</div>
</div>
</div>
</Teleport>
</template>
<script setup lang="ts">
const visible=ref(false),{token,playerId}=useAuth()
const searchQuery=ref(''),searchResults=ref<any[]>([]),friends=ref<any[]>([]),requests=ref<any[]>([]),loading=ref(false)
async function api(url:string,opts:any={}){const r=await fetch(url,{headers:{'Content-Type':'application/json',Authorization:'Bearer '+token.value},...opts});return r.json()}
async function load(){loading.value=true;try{const d=await api('/api/v1/player/'+playerId.value+'/friends');friends.value=d.data||[];const d2=await api('/api/v1/player/'+playerId.value+'/friends/pending');requests.value=d2.data||[]}catch{}finally{loading.value=false}}
async function doSearch(){if(!searchQuery.value.trim())return;const d=await api('/api/v1/player/'+playerId.value+'/friends/search?q='+encodeURIComponent(searchQuery.value));searchResults.value=d.data||[]}
async function addFriend(fid:number){await api('/api/v1/player/'+playerId.value+'/friends/add',{method:'POST',body:JSON.stringify({friend_id:fid})});searchQuery.value='';searchResults.value=[];load()}
async function removeFriend(fid:number){await api('/api/v1/player/'+playerId.value+'/friends/remove',{method:'POST',body:JSON.stringify({friend_id:fid})});load()}
async function acceptFriend(id:number){await api('/api/v1/player/'+playerId.value+'/friends/accept',{method:'POST',body:JSON.stringify({apply_id:id})});load()}
async function rejectFriend(id:number){await api('/api/v1/player/'+playerId.value+'/friends/remove',{method:'POST',body:JSON.stringify({friend_id:id})});load()}
watch(visible,v=>{if(v)load()})
defineExpose({open:(v:boolean)=>{visible.value=v}})
</script>