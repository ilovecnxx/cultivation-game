<template>
  <aside class="gh-panel gh-chat">
    <div class="panel-header">
      <h4>💬 社交</h4>
      <div class="panel-actions panel-actions-center">
        <button class="pa-btn" :class="{active:chatTab==='all'}" @click="chatTab='all'">📢 全部</button>
        <button class="pa-btn" :class="{active:chatTab==='world'}" @click="chatTab='world'">🌍 世界</button>
        <button class="pa-btn" :class="{active:chatTab==='sect'}" @click="chatTab='sect'">👥 宗门</button>
        <button class="pa-btn" :class="{active:chatTab==='private'}" @click="chatTab='private'">💬 私聊</button>
        <button class="pa-btn" :class="{active:chatTab==='friend'}" @click="chatTab='friend'">👫 好友</button>
        <button class="pa-btn" :class="{active:chatTab==='daoyou'}" @click="chatTab='daoyou'">💑 道友</button>
        <button class="pa-btn" :class="{active:chatTab==='master'}" @click="chatTab='master'">🎓 师徒</button>
      </div>
      <div class="panel-actions panel-actions-right">
        <button class="pa-btn" @click="logLocked=!logLocked" :title="logLocked?'解锁滚动':'锁定滚动'">{{ logLocked?'🔓':'🔒' }}</button>
      </div>
    </div>
    <div class="panel-body chat-body" ref="chatBody">
      <div v-for="(m,i) in filteredChat" :key="i" class="chat-msg">
        <span class="chat-time">{{ m.time }}</span>
        <span class="chat-name" :style="{color:m.color}">{{ m.name }}</span>
        <span class="chat-text">{{ m.text }}</span>
      </div>
      <div v-if="filteredChat.length===0" class="log-empty">{{ chatEmptyText }}</div>
    </div>
    <div class="chat-input-bar">
      <input v-model="chatInput" class="chat-input" :placeholder="chatPlaceholder" @keyup.enter="sendChat" />
      <button class="chat-send" @click="sendChat">发送</button>
    </div>
  </aside>
</template>

<script setup lang="ts">
const props = defineProps<{ playerName: string }>()

// 聊天状态
const chatTab = ref('all')
const chatMessages = reactive<any[]>([])
const chatInput = ref('')
const logLocked = ref(false)
let ws: WebSocket | null = null
let chatWs: WebSocket | null = null

// Token 工具
function getToken() { return localStorage.getItem('token') || '' }
function getPID() { return localStorage.getItem('player_id') || '0' }

const chatPlaceholder = computed(() => {
  const t: Record<string,string> = { all:'发送全服消息...', world:'世界频道发言...', sect:'宗门频道发言...', private:'输入私聊对象和内容...', friend:'给好友留言...', daoyou:'道侣悄悄话...', master:'向师父/徒弟发言...' }
  return t[chatTab.value] || '说点什么...'
})
const chatEmptyText = computed(() => {
  const t: Record<string,string> = { all:'暂无消息', world:'世界频道静默中...', sect:'宗门频道暂无消息', private:'暂无私聊消息', friend:'暂无好友消息', daoyou:'暂无道侣消息', master:'暂无师徒消息' }
  return t[chatTab.value] || ''
})
const filteredChat = computed(() => chatTab.value === 'all' ? chatMessages : chatMessages.filter((m: any) => m.channel === chatTab.value))

// WebSocket 连接
function connectWS() {
  const t = getToken(); if (!t) return
  // token 通过 Authorization 子协议传递（比 URL 参数更安全，不会出现在服务器日志中）
  try {
    ws = new WebSocket((location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/ws', ['authorization.' + t])
    ws.onopen = () => { ws?.send(JSON.stringify({ type: 'auth', token: t })) }
    ws.onmessage = (e) => { try { const m = JSON.parse(e.data); if (m.type === 'chat') handleServerChat(m) } catch {} }
    ws.onclose = () => { setTimeout(connectWS, 5000) }
    ws.onerror = () => { ws?.close() }
  } catch {
    // 降级：某些浏览器不支持子协议，回退到 URL 参数
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
  const t = getToken(); if (!t) return
  // token 代替 user_id 作为身份凭证（后端 Handler 需从 JWT 中解析 player_id）
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

// 加载历史
async function loadChatHistory() {
  try {
    const t = getToken(); if (!t) return
    const r = await fetch('/api/v1/chat/history?channel=world&limit=50', { headers: { Authorization: 'Bearer ' + t } })
    const d = await r.json()
    if (d.data && d.data.length) { d.data.reverse().forEach((m: any) => handleServerChat(m)) }
  } catch {}
}

// 发送消息
function sendChat() {
  const v = chatInput.value.trim(); if (!v) return
  const ch = chatTab.value === 'all' ? 'world' : chatTab.value
  chatInput.value = ''
  if (chatWs && chatWs.readyState === WebSocket.OPEN) {
    chatWs.send(JSON.stringify({ channel: ch, content: v }))
  } else if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'chat', channel: ch, text: v }))
  }
  // 乐观更新
  handleServerChat({ channel: ch, content: v, sender_name: props.playerName || '修仙者' })
}

onMounted(() => { loadChatHistory(); connectWS() })
</script>