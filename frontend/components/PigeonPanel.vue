<template>
  <Teleport to="body">
    <div v-if="show" class="modal-overlay" @click.self="$emit('close')">
      <div class="pigeon-modal">
        <div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">🕊️ 飞鸽传书</span><div class="top-bar-spacer"/><button class="modal-close" @click="$emit('close')">✕</button></div></header><div class="gold-divider"/>
        <div class="pigeon-body">
          <div class="pig-left">
            <div class="pig-search-box"><input v-model="searchQuery" class="pig-search-inp" placeholder="搜索道友..." @keyup.enter="search" /></div>
            <div v-if="searchResults.length" class="pig-sr-results">
              <div v-for="u in searchResults" :key="u.id" class="pig-sr-row"><span class="pig-sr-name">{{ u.nickname }}</span><button @click="addFriend(u.id)">加好友</button></div>
            </div>
            <div class="pig-group">
              <div class="pig-group-head" @click="showFriends=!showFriends">👤 好友 {{ friends.length }}/50 <span class="pig-group-arrow" :class="{open:showFriends}">▾</span></div>
              <div v-if="showFriends">
                <div v-for="f in friends" :key="f.id" class="pig-friend" :class="{active:activePeer===f.id}" @click="openChat(f)">
                  <div class="pig-f-avatar">{{ f.nickname[0] }}</div>
                  <div class="pig-f-info"><div class="pig-f-name">{{ f.nickname }}<span class="pig-f-online" :class="{online:f.online}" /></div><div class="pig-f-realm">{{ f.realm_stage }}期</div></div>
                </div>
                <div v-if="friends.length===0" class="pig-empty-tip">暂无好友</div>
              </div>
            </div>
            <div class="pig-group">
              <div class="pig-group-head" @click="showRequests=!showRequests">📋 好友申请 <span v-if="pendingRequests.length" class="pig-badge">{{ pendingRequests.length }}</span><span class="pig-group-arrow" :class="{open:showRequests}">▾</span></div>
              <div v-if="showRequests">
                <div v-for="r in pendingRequests" :key="r.id" class="pig-request-row"><span class="pig-r-name">{{ r.from_name }}</span><div class="pig-r-btns"><button @click="acceptFriend(r.id)">接受</button><button class="pig-r-reject" @click="rejectFriend(r.id)">拒绝</button></div></div>
              </div>
            </div>
            <div class="pig-group"><div class="pig-group-head" @click="showDaolv=!showDaolv">💑 道侣 ▾</div><div v-if="showDaolv" class="pig-empty-tip">修仙路漫漫，寻一道侣</div></div>
            <div class="pig-group"><div class="pig-group-head" @click="showMaster=!showMaster">🎓 师徒 ▾</div><div v-if="showMaster" class="pig-empty-tip">拜师或收徒，传承衣钵</div></div>
          </div>
          <div class="pig-right">
            <div v-if="activePeer" class="pig-chat">
              <div class="pig-chat-msgs"><div v-for="m in privateMessages" :key="m.id" class="pig-msg-bubble" :class="{self:m.from_id===pid}"><div class="pig-msg-avatar">{{ m.from_id===pid ? myName[0] : activePeerName[0] }}</div><div class="pig-msg-bub-text">{{ m.text }}</div></div></div>
              <div class="pig-chat-send"><input v-model="privateInput" @keyup.enter="sendPrivate" placeholder="按回车发送" class="pig-send-inp" /><button @click="sendPrivate" class="pig-send-btn">发送</button></div>
            </div>
            <div v-else class="pig-empty-tip">选择一位好友开始聊天</div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
const props = defineProps<{ show: boolean; playerName: string; playerId: string }>()
defineEmits(['close'])
const { token, playerId: pid } = useAuth()
const searchQuery = ref('')
const searchResults = ref<any[]>([])
const showFriends = ref(true)
const showRequests = ref(true)
const showDaolv = ref(false)
const showMaster = ref(false)
const friends = ref<any[]>([])
const pendingRequests = ref<any[]>([])
const activePeer = ref(0)
const activePeerName = ref('')
const privateMessages = ref<any[]>([])
const privateInput = ref('')
const myName = computed(() => props.playerName)

async function api(path: string, opts: RequestInit = {}) {
  const r = await fetch(path, { headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token.value, ...opts.headers as any }, ...opts })
  return r.json()
}
async function loadFriends() { const d = await api('/api/v1/player/' + pid.value + '/friends'); friends.value = d.data || [] }
async function loadPending() { const d = await api('/api/v1/player/' + pid.value + '/friends/pending'); pendingRequests.value = d.data || [] }
async function search() { if (!searchQuery.value.trim()) return; const d = await api('/api/v1/player/' + pid.value + '/friends/search?q=' + encodeURIComponent(searchQuery.value)); searchResults.value = d.data || [] }
async function addFriend(fid: number) { await api('/api/v1/player/' + pid.value + '/friends/add', { method: 'POST', body: JSON.stringify({ friend_id: fid }) }); searchResults.value = []; loadFriends() }
async function acceptFriend(id: number) { await api('/api/v1/player/' + pid.value + '/friends/accept', { method: 'POST', body: JSON.stringify({ apply_id: id }) }); loadFriends(); loadPending() }
async function rejectFriend(id: number) { await api('/api/v1/player/' + pid.value + '/friends/remove', { method: 'POST', body: JSON.stringify({ friend_id: id }) }); loadPending() }
async function openChat(f: any) { activePeer.value = f.id; activePeerName.value = f.nickname; const d = await api('/api/v1/player/' + pid.value + '/messages?peer_id=' + f.id); privateMessages.value = (d.data || []).reverse() }
async function sendPrivate() { const v = privateInput.value.trim(); if (!v || !activePeer.value) return; const d = await api('/api/v1/player/' + pid.value + '/messages/send', { method: 'POST', body: JSON.stringify({ to_id: activePeer.value, text: v }) }); if (d && d.code === 0) { privateMessages.value.push({ id: Date.now(), from_id: parseInt(pid.value), to_id: activePeer.value, text: v }); privateInput.value = '' } }

watch(() => props.show, (v) => { if (v) { loadFriends(); loadPending() } })
</script>