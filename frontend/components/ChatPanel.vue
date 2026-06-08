<template>
  <aside class="gh-panel gh-chat">
    <div class="panel-header">
      <h4>💬 社交</h4>
      <div class="panel-actions panel-actions-right">
        <van-button size="small" plain type="default" @click="logLocked=!logLocked">{{ logLocked?'🔓':'🔒' }}</van-button>
        <van-button size="small" plain type="default" @click="chatMessages.splice(0)">🗑️</van-button>
      </div>
    </div>
    <van-tabs v-model:active="chatTab" class="chat-vant-tabs" color="#d4a843" title-active-color="#d4a843" title-inactive-color="#8a8578" background="transparent" :border="false">
      <van-tab title="📢 全部" name="all" />
      <van-tab title="🌍 世界" name="world" />
      <van-tab title="👥 宗门" name="sect" />
      <van-tab title="💬 私聊" name="private" />
      <van-tab title="👫 好友" name="friend" />
      <van-tab title="💑 道友" name="daoyou" />
      <van-tab title="🎓 师徒" name="master" />
    </van-tabs>
    <div class="panel-body chat-body" ref="chatBody">
      <div v-for="(m,i) in filteredChat" :key="i" class="chat-msg">
        <span class="chat-time">{{ m.time }}</span>
        <span class="chat-name" :style="{color:m.color}">{{ m.name }}</span>
        <span class="chat-text">{{ m.text }}</span>
      </div>
      <div v-if="filteredChat.length===0" class="log-empty">{{ chatEmptyText }}</div>
    </div>
    <div class="chat-input-bar">
      <van-field v-model="chatInput" :placeholder="chatPlaceholder" border class="chat-input-vant" @keyup.enter="sendChat" />
      <van-button size="small" type="primary" @click="sendChat">发送</van-button>
    </div>
  </aside>
</template>

<script setup lang="ts">
const props = defineProps<{ playerName: string }>()
const chatTab = ref('all')
const chatMessages = reactive<any[]>([])
const chatInput = ref('')
const logLocked = ref(false)
let ws: WebSocket | null = null
let chatWs: WebSocket | null = null
const { token, playerId } = useAuth()
const chatPlaceholder = computed(() => {
  const t: Record<string,string> = { all:'发送全服消息...', world:'世界频道发言...', sect:'宗门频道发言...', private:'输入私聊对象和内容...', friend:'给好友留言...', daoyou:'道侣悄悄话...', master:'向师父/徒弟发言...' }
  return t[chatTab.value] || '说点什么...'
})
const chatEmptyText = computed(() => {
  const t: Record<string,string> = { all:'暂无消息', world:'世界频道静默中...', sect:'宗门频道暂无消息', private:'暂无私聊消息', friend:'暂无好友消息', daoyou:'暂无道侣消息', master:'暂无师徒消息' }
  return t[chatTab.value] || ''
})
const filteredChat = computed(() => chatTab.value === 'all' ? chatMessages : chatMessages.filter((m: any) => m.channel === chatTab.value))
function connectWS() {
  const t = token.value; if (!t) return
  try {
    ws = new WebSocket((location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/ws', ['authorization.' + t])
    ws.onopen = () => { ws?.send(JSON.stringify({ type: 'auth', token: t })) }
    ws.onmessage = (e) => { try { const m = JSON.parse(e.data); if (m.type === 'chat') handleServerChat(m) } catch {} }
    ws.onclose = () => { setTimeout(connectWS, 5000) }
    ws.onerror = () => { ws?.close() }
  } catch {
    try {
      ws = new WebSocket((location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/ws?token=' + t)
      ws.onmessage = (e) => { try { const m = JSON.parse(e.data); if (m.type === 'chat') handleServerChat(m) } catch {} }
      ws.onclose = () => { setTimeout(connectWS, 5000) }
      ws.onerror = () => { ws?.close() }
    } catch {}
  }
  connectChatWS()
}
function connectChatWS() {
  const t = token.value; if (!t) return
  try {
    chatWs = new WebSocket((location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/api/v1/chat/ws?token=' + t)
    chatWs.onmessage = (e) => { try { const m = JSON.parse(e.data); handleServerChat(m) } catch {} }
    chatWs.onclose = () => { setTimeout(connectChatWS, 5000) }
    chatWs.onerror = () => { chatWs?.close() }
  } catch {}
}
function handleServerChat(m: any) {
  chatMessages.push({ time: new Date(m.created_at || Date.now()).toLocaleTimeString('zh-CN', { hour12: false }).slice(0, 5), name: m.sender_name || m.name || '未知', text: m.content || m.text || '', color: m.is_system ? '#ffd700' : (m.color || '#d4a843'), channel: m.channel || 'world' })
  if (chatMessages.length > 300) chatMessages.shift()
}
async function loadChatHistory() {
  try {
    const t = token.value; if (!t) return
    const r = await fetch('/api/v1/chat/history?channel=world&limit=50', { headers: { Authorization: 'Bearer ' + t } })
    const d = await r.json()
    if (d.data && d.data.length) { d.data.reverse().forEach((m: any) => handleServerChat(m)) }
  } catch {}
}
function sendChat() {
  const v = chatInput.value.trim(); if (!v) return
  const ch = chatTab.value === 'all' ? 'world' : chatTab.value
  chatInput.value = ''
  if (chatWs && chatWs.readyState === WebSocket.OPEN) {
    chatWs.send(JSON.stringify({ channel: ch, content: v }))
  } else if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'chat', channel: ch, text: v }))
  }
  handleServerChat({ channel: ch, content: v, sender_name: props.playerName || '修仙者' })
}
onMounted(() => { loadChatHistory(); connectWS() })
</script>

<style scoped>
.chat-vant-tabs :deep(.van-tab) { font-size: 11px; }
.chat-input-vant { flex: 1; --van-field-background: rgba(0,0,0,.2); --van-field-border-color: rgba(212,168,67,.2); }
.chat-input-bar { display: flex; gap: 6px; align-items: center; padding: 6px; }
</style>