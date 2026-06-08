<template>
  <van-popup v-model:show="visible" position="bottom" round :style="{ height:'70%', background:'#1a1a2e' }" @click="visible=false">
    <div class="wiki-modal pigeon-modal">
      <div class="wiki-header">
        <h2>🕊️ 飞鸽传书</h2>
        <van-icon name="cross" size="24" @click="visible=false" />
      </div>
      <div class="pigeon-content">
        <div class="pig-search-box">
          <van-search v-model="friendSearch" placeholder="搜索道友..." shape="round" background="transparent" @search="search" />
        </div>
        <div v-if="searchResults.length" class="pig-search-results">
          <div v-for="u in searchResults" :key="u.id" class="pig-user-item" @click="addFriend(u.id)">
            <span>{{ u.nickname }}</span>
            <van-button size="mini" type="primary">加好友</van-button>
          </div>
        </div>
        <div class="pig-group">
          <div class="pig-group-head" @click="showFriends=!showFriends">👤 好友 {{ friends.length }}/50 ▾</div>
          <div v-if="showFriends" class="pig-group-body">
            <div v-for="f in friends" :key="f.id" class="pig-friend" :class="{active:activePeer===f.id}" @click="openChat(f)">
              <div class="pig-f-avatar">{{ f.nickname[0] }}</div>
              <div class="pig-f-info">
                <div class="pig-f-name">{{ f.nickname }}<span class="pig-f-online" :class="{online:f.online}" /></div>
              </div>
            </div>
            <div v-if="friends.length===0" class="pig-empty-tip">暂无好友</div>
          </div>
        </div>
        <div class="pig-group">
          <div class="pig-group-head" @click="showRequests=!showRequests">📋 好友申请 {{ pendingRequests.length }} ▾</div>
          <div v-if="showRequests" class="pig-group-body">
            <div v-for="r in pendingRequests" :key="r.id" class="pig-request">
              <span>{{ r.from_name }}</span>
              <van-button size="mini" type="primary" @click="acceptFriend(r.id)">接受</van-button>
              <van-button size="mini" plain type="danger" @click="rejectFriend(r.id)">拒绝</van-button>
            </div>
          </div>
        </div>
        <div class="pig-group">
          <div class="pig-group-head" @click="showDaolv=!showDaolv">💑 道侣 ▾</div>
          <div v-if="showDaolv" class="pig-group-body"><div class="pig-empty-tip">修仙路漫漫，寻一道侣</div></div>
        </div>
        <div class="pig-group">
          <div class="pig-group-head" @click="showMaster=!showMaster">🎓 师徒 ▾</div>
          <div v-if="showMaster" class="pig-group-body"><div class="pig-empty-tip">拜师或收徒，传承衣钵</div></div>
        </div>
      </div>
    </div>
  </van-popup>
</template>

<script setup lang="ts">
const visible = ref(false)
const friendSearch = ref('')
const searchResults = ref<any[]>([])
const showFriends = ref(true)
const showRequests = ref(true)
const showDaolv = ref(false)
const showMaster = ref(false)
const friends = ref<any[]>([])
const pendingRequests = ref<any[]>([])
const activePeer = ref(0)

function getToken() { return localStorage.getItem('token') || '' }
function getPID() { return localStorage.getItem('player_id') || '0' }

function open(v: boolean) { visible.value = v }

async function search() {
  const pid = getPID(); if (!pid || !friendSearch.value.trim()) return
  try {
    const r = await fetch('/api/v1/player/'+pid+'/friends/search?q='+encodeURIComponent(friendSearch.value), { headers: { Authorization: 'Bearer ' + getToken() } })
    const d = await r.json(); searchResults.value = d.data || []
  } catch {}
}
async function addFriend(fid: number) {
  const pid = getPID(); if (!pid) return
  await fetch('/api/v1/player/'+pid+'/friends/add', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + getToken() }, body: JSON.stringify({ friend_id: fid }) })
  searchResults.value = []
  loadFriends()
}
async function loadFriends() {
  const pid = getPID(); if (!pid) return
  try { const r = await fetch('/api/v1/player/'+pid+'/friends', { headers: { Authorization: 'Bearer ' + getToken() } }); const d = await r.json(); friends.value = d.data || [] } catch {}
}
async function loadPending() {
  const pid = getPID(); if (!pid) return
  try { const r = await fetch('/api/v1/player/'+pid+'/friends/pending', { headers: { Authorization: 'Bearer ' + getToken() } }); const d = await r.json(); pendingRequests.value = d.data || [] } catch {}
}
async function acceptFriend(id: number) {
  const pid = getPID(); if (!pid) return
  await fetch('/api/v1/player/'+pid+'/friends/accept', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + getToken() }, body: JSON.stringify({ apply_id: id }) })
  loadFriends(); loadPending()
}
async function rejectFriend(id: number) {
  const pid = getPID(); if (!pid) return
  await fetch('/api/v1/player/'+pid+'/friends/remove', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + getToken() }, body: JSON.stringify({ friend_id: id }) })
  loadPending()
}
function openChat(f: any) { activePeer.value = f.id }
onMounted(() => { loadFriends(); loadPending() })
defineExpose({ open, loadFriends })
</script>