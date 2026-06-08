<template>
  <Teleport to="body">
    <div v-if="show" class="modal-overlay" @click.self="$emit('close')">
      <div class="pigeon-modal">
        <div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">🕊️ 飞鸽传书</span><div class="top-bar-spacer"/><button class="modal-close" @click="$emit('close')">✕</button></div></header><div class="gold-divider"/>
        <div class="pigeon-body">
          <!-- 左侧联系人列表 -->
          <div class="pig-left">
            <div class="pig-search-box"><input v-model="searchQuery" class="pig-search-inp" placeholder="搜索..." @keyup.enter="doSearch"/></div>
            <div v-if="searchResults.length" class="pig-sr-results">
              <div v-for="u in searchResults" :key="u.id" class="pig-sr-row" @click="addFriend(u.id)"><span class="pig-sr-name">{{ u.nickname }}</span><van-button size="mini" type="primary">加好友</van-button></div>
            </div>
            <!-- 宗门分组 -->
            <div class="pig-group">
              <div class="pig-group-head" @click="showSect=!showSect">🏛️ 宗门 <span class="pig-group-arrow" :class="{open:showSect}">▾</span></div>
              <div v-if="showSect"><div v-if="sectList.length" class="pig-contact-list"><div v-for="s in sectList" :key="s.id" class="pig-contact" :class="{active:activeChat===s.id}" @click="openSectChat(s)"><span class="pig-contact-avatar">{{ s.name[0] }}</span><span class="pig-contact-name">{{ s.name }}</span></div></div><div v-else class="pig-empty-tip">未加入宗门</div></div>
            </div>
            <!-- 好友分组 -->
            <div class="pig-group"><div class="pig-group-head" @click="showFriends=!showFriends">👫 好友 {{ friends.length }}<span class="pig-group-arrow" :class="{open:showFriends}">▾</span></div>
              <div v-if="showFriends"><div v-if="friends.length" class="pig-contact-list"><div v-for="f in friends" :key="f.id" class="pig-contact" :class="{active:activeChat==='f_'+f.id}" @click="openFriendChat(f)"><span class="pig-contact-avatar" :class="{online:f.online}">{{ f.nickname[0] }}</span><span class="pig-contact-name">{{ f.nickname }}</span></div></div><div v-else class="pig-empty-tip">暂无好友</div></div>
            </div>
            <!-- 道侣分组 -->
            <div class="pig-group"><div class="pig-group-head" @click="showDaolv=!showDaolv">💑 道侣 <span class="pig-group-arrow" :class="{open:showDaolv}">▾</span></div>
              <div v-if="showDaolv"><div v-if="daolvPartner" class="pig-contact-list"><div class="pig-contact" :class="{active:activeChat==='daolv'}" @click="openDaolvChat"><span class="pig-contact-avatar">💑</span><span class="pig-contact-name">道侣</span></div></div><div v-else class="pig-empty-tip">修仙路漫漫，寻一道侣</div></div>
            </div>
            <!-- 师父分组 -->
            <div class="pig-group"><div class="pig-group-head" @click="showMaster=!showMaster">👴 师父 <span class="pig-group-arrow" :class="{open:showMaster}">▾</span></div>
              <div v-if="showMaster"><div v-if="masterName" class="pig-contact-list"><div class="pig-contact" :class="{active:activeChat==='master'}" @click="openMasterChat"><span class="pig-contact-avatar">👴</span><span class="pig-contact-name">{{ masterName }}</span></div></div><div v-else class="pig-empty-tip">尚未拜师</div></div>
            <!-- 徒儿分组 -->
            <div class="pig-group"><div class="pig-group-head" @click="showDisciple=!showDisciple">🧒 徒儿 <span class="pig-group-arrow" :class="{open:showDisciple}">▾</span></div>
              <div v-if="showDisciple"><div v-if="discipleList.length" class="pig-contact-list"><div v-for="d in discipleList" :key="d.id" class="pig-contact" :class="{active:activeChat==='d_'+d.id}" @click="openDiscipleChat(d)"><span class="pig-contact-avatar">{{ d.nickname[0] }}</span><span class="pig-contact-name">{{ d.nickname }}</span></div></div><div v-else class="pig-empty-tip">尚未收徒</div></div>
            </div>
            <!-- 世界频道 -->
            <div class="pig-group"><div class="pig-group-head" @click="activeChat='world';loadWorldHistory()">🌍 世界频道</div></div>
          </div>
          <!-- 右侧聊天区域 -->
          <div class="pig-right">
            <div v-if="activeChat" class="pig-chat">
              <div class="pig-chat-header">{{ chatTitle }}</div>
              <div class="pig-chat-msgs" ref="chatMsgs"><div v-for="m in currentMessages" :key="m.id||m.time" class="pig-msg-bubble" :class="{self:m.sender_id===myId||m.name===myName}"><div class="pig-msg-avatar">{{ (m.sender_name||m.name||'?')[0] }}</div><div class="pig-msg-content"><div class="pig-msg-bub-text">{{ m.content||m.text }}</div><div class="pig-msg-time">{{ m.time||'' }}</div></div></div></div>
              <div class="pig-chat-send"><input v-model="chatInput" @keyup.enter="sendMsg" placeholder="输入消息..." class="pig-send-inp"/><button @click="sendMsg" class="pig-send-btn">发送</button></div>
            </div>
            <div v-else class="pig-empty-tip" style="flex:1;display:flex;align-items:center;justify-content:center">选择一个联系人或频道开始聊天</div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
const props = defineProps<{ show: boolean; playerName: string }>()
defineEmits(['close'])
const {token,playerId}=useAuth()
const searchQuery=ref(''),searchResults=ref<any[]>([])
const showSect=ref(true),showFriends=ref(true),showDaolv=ref(true),showMaster=ref(true),showDisciple=ref(true)
const friends=ref<any[]>([]),sectList=ref<any[]>([])
const daolvPartner=ref<any>(null),masterName=ref(''),discipleList=ref<any[]>([])
const activeChat=ref(''),chatInput=ref(''),currentMessages=ref<any[]>([])
const chatMsgs=ref<HTMLElement|null>(null)
const myId=computed(()=>playerId.value)
const myName=computed(()=>props.playerName)
const chatTitle=computed(()=>{if(activeChat.value==='world')return'🌍 世界频道';if(activeChat.value==='daolv')return'💑 道侣';if(activeChat.value==='master')return'👴 师父';if(activeChat.value?.startsWith('d_')){const d=discipleList.value.find((x:any)=>'d_'+x.id===activeChat.value);return'🧒 徒儿·'+d?.nickname};const f=friends.value.find((x:any)=>'f_'+x.id===activeChat.value);if(f)return f.nickname;const s=sectList.value.find((x:any)=>x.id===activeChat.value);if(s)return s.name;return'聊天'})

async function api(url:string,opts:any={}){const r=await fetch(url,{headers:{'Content-Type':'application/json',Authorization:'Bearer '+token.value},...opts});return r.json()}
async function loadFriends(){try{const d=await api('/api/v1/player/'+playerId.value+'/friends');friends.value=d.data||[]}catch{}}
async function loadSect(){try{const d=await api('/api/v1/sect/my');sectList.value=d.data?[d.data]:[]}catch{}}
async function loadDaolv(){try{const d=await api('/api/v1/daolv/status/'+playerId.value);daolvPartner.value=d.data}catch{}}
async function loadMaster(){try{const d=await api('/api/v1/master/my-master');masterName.value=d.data?.master_name||''}catch{}}
async function loadDisciples(){try{const d=await api('/api/v1/master/my-students');discipleList.value=d.data||[]}catch{}}
async function doSearch(){if(!searchQuery.value.trim())return;const d=await api('/api/v1/player/'+playerId.value+'/friends/search?q='+encodeURIComponent(searchQuery.value));searchResults.value=d.data||[]}
async function addFriend(fid:number){await api('/api/v1/player/'+playerId.value+'/friends/add',{method:'POST',body:JSON.stringify({friend_id:fid})});searchQuery.value='';searchResults.value=[];loadFriends()}
async function loadWorldHistory(){try{const d=await api('/api/v1/chat/history?channel=world&limit=30');currentMessages.value=(d.data||[]).map((m:any)=>({...m,time:new Date(m.created_at).toLocaleTimeString('zh-CN',{hour12:false}).slice(0,5)}))}catch{}}
function openFriendChat(f:any){activeChat.value='f_'+f.id;currentMessages.value=[{text:'开始聊天吧~',sender_name:f.nickname,time:'',sender_id:f.id}]}
function openSectChat(s:any){activeChat.value=s.id;currentMessages.value=[{text:'宗门频道',sender_name:'系统',time:'',sender_id:'system'}]}
function openDaolvChat(){activeChat.value='daolv';currentMessages.value=[{text:'道侣悄悄话',sender_name:'系统',time:'',sender_id:'system'}]}
function openMasterChat(){activeChat.value='master';currentMessages.value=[{text:'师父传音',sender_name:'系统',time:'',sender_id:'system'}]}
function openDiscipleChat(d:any){activeChat.value='d_'+d.id;currentMessages.value=[{text:'与徒儿 '+d.nickname+' 的传音',sender_name:'系统',time:'',sender_id:'system'}]}
function sendMsg(){const v=chatInput.value.trim();if(!v)return;currentMessages.value.push({text:v,sender_name:myName.value,time:new Date().toLocaleTimeString('zh-CN',{hour12:false}).slice(0,5),sender_id:playerId.value});chatInput.value='';nextTick(()=>{if(chatMsgs.value)chatMsgs.value.scrollTop=chatMsgs.value.scrollHeight})}

watch(()=>props.show,v=>{if(v){loadFriends();loadSect();loadDaolv();loadMaster()}})
</script>

<style scoped>
.pigeon-modal{width:820px;max-width:96vw;background:linear-gradient(180deg,#12122a,#0d0d1a)!important;overflow:hidden!important}
.pigeon-body{display:flex;height:540px;max-height:72vh}
.pig-left{width:240px;min-width:180px;display:flex;flex-direction:column;overflow-y:auto;border-right:1px solid rgba(212,168,67,.1);background:rgba(0,0,0,.2)}
.pig-right{flex:1;display:flex;flex-direction:column;background:rgba(0,0,0,.08)}
.pig-search-box{padding:8px;background:rgba(0,0,0,.2)}
.pig-search-inp{width:100%;padding:6px 10px;border:1px solid rgba(255,255,255,.1);border-radius:16px;background:rgba(255,255,255,.05);color:#fff;font-size:12px;outline:none}
.pig-group{margin:1px 0}
.pig-group-head{display:flex;align-items:center;gap:4px;padding:9px 12px;font-size:12px;font-weight:600;color:rgba(255,255,255,.55);cursor:pointer;user-select:none;transition:all .15s}
.pig-group-head:hover{background:rgba(255,255,255,.03);color:#d4a843}
.pig-group-arrow{font-size:8px;margin-left:auto;transition:transform .2s}
.pig-group-arrow.open{transform:rotate(0)}.pig-group-arrow:not(.open){transform:rotate(-90deg)}
.pig-contact-list{display:flex;flex-direction:column}
.pig-contact{display:flex;align-items:center;gap:8px;padding:6px 12px;cursor:pointer;transition:all .12s}
.pig-contact:hover{background:rgba(212,168,67,.05)}.pig-contact.active{background:linear-gradient(90deg,rgba(212,168,67,.12),rgba(212,168,67,.04));border-right:2px solid #d4a843}
.pig-contact-avatar{width:28px;height:28px;border-radius:50%;background:linear-gradient(135deg,#d4a843,#b8860b);display:flex;align-items:center;justify-content:center;font-size:12px;font-weight:700;color:#fff;flex-shrink:0}.pig-contact-avatar.online{box-shadow:0 0 6px rgba(76,175,80,.5)}
.pig-contact-name{font-size:13px;color:#fff;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.pig-empty-tip{text-align:center;padding:12px;font-size:11px;color:rgba(255,255,255,.12);font-style:italic}
.pig-sr-results{margin:4px 0}
.pig-sr-row{display:flex;align-items:center;gap:6px;padding:6px 12px;font-size:12px;border-bottom:1px solid rgba(255,255,255,.03);cursor:pointer}.pig-sr-name{color:#fff;flex:1}
.pig-chat{display:flex;flex-direction:column;height:100%}
.pig-chat-header{flex-shrink:0;padding:8px 14px;font-size:14px;font-weight:700;color:#d4a843;border-bottom:1px solid rgba(212,168,67,.1);background:rgba(0,0,0,.2)}
.pig-chat-msgs{flex:1;overflow-y:auto;padding:10px 14px;display:flex;flex-direction:column;gap:8px}
.pig-msg-bubble{display:flex;gap:6px;max-width:75%}.pig-msg-bubble.self{align-self:flex-end;flex-direction:row-reverse}
.pig-msg-avatar{width:26px;height:26px;border-radius:50%;background:rgba(255,255,255,.1);display:flex;align-items:center;justify-content:center;font-size:11px;font-weight:700;color:rgba(255,255,255,.6);flex-shrink:0;align-self:flex-end}
.pig-msg-content{display:flex;flex-direction:column}
.pig-msg-bub-text{padding:6px 10px;border-radius:10px;font-size:12px;line-height:1.4;word-break:break-word}.pig-msg-bubble:not(.self) .pig-msg-bub-text{background:rgba(255,255,255,.08);color:#ddd;border-bottom-left-radius:2px}.pig-msg-bubble.self .pig-msg-bub-text{background:linear-gradient(135deg,rgba(212,168,67,.25),rgba(212,168,67,.12));color:#fff;border-bottom-right-radius:2px}
.pig-msg-time{font-size:9px;color:rgba(255,255,255,.2);margin-top:2px}.pig-msg-bubble.self .pig-msg-time{text-align:right}
.pig-chat-send{display:flex;gap:6px;padding:8px 10px;border-top:1px solid rgba(255,255,255,.06)}
.pig-send-inp{flex:1;padding:6px 12px;border:1px solid rgba(255,255,255,.08);border-radius:16px;background:rgba(255,255,255,.04);color:#fff;font-size:12px;outline:none}
.pig-send-btn{padding:6px 16px;border:none;border-radius:16px;background:linear-gradient(135deg,#d4a843,#b8860b);color:#fff;font-size:12px;font-weight:600;cursor:pointer}
</style>