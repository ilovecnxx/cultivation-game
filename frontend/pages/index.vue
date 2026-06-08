<template>
  <div class="landing-view" :class="{ 'light-mode': !isDark }">
    <div class="gold-divider"><div class="gold-divider__light" /></div>
    <header class="top-bar">
      <div class="top-bar-inner">
        <span class="brand-logo">☯</span>
        <span class="brand-name">修仙世界</span>
        <div class="top-bar-spacer" />
        <div class="player-stats">
          <span class="online-badge"><span class="online-dot" /><span :class="{ flip: flipOnline }">{{ fmt(onlineCount) }}</span> 在线修士</span>
          <span class="registered-badge"><span :class="{ flip: flipRegistered }">{{ fmt(registeredCount) }}</span> 注册修士</span>
        </div>
        <button class="theme-toggle" @click="toggleTheme">{{ isDark ? '☀' : '🌙' }}</button>
      </div>
    </header>
    <div class="gold-divider"><div class="gold-divider__light" /></div>
    <main class="main-grid">
      <section class="left-col">
        <div class="brand-block">
          <div class="yin-yang" @click="toggleTheme" />
          <h1 class="game-title">修仙世界</h1>
          <p class="game-subtitle">修仙悟道 长生逍遥</p>
        </div>
        <div class="realm-progress">
          <div class="realm-track-bg" />
          <div v-for="(r, i) in realms" :key="r.name" class="realm-node" :style="{ animationDelay: i * 0.12 + 's' }" :class="{ completed: i < currentRealmIndex, active: i === currentRealmIndex }">
            <div class="realm-dot"><span v-if="i < currentRealmIndex" class="realm-check">✓</span><span v-else-if="i === currentRealmIndex" class="realm-pulse" /></div>
            <span class="realm-label">{{ r.name }}</span>
          </div>
        </div>
        <button class="enter-btn" @click="showAuth = true">踏入仙途</button>
        <p class="cultivation-quote">{{ currentQuote }}</p>
      </section>
    </main>
    <footer class="bottom-bar"><span class="copyright">© 2026 修仙世界 · 以文为本 以心入道</span></footer>
  <Teleport to="body">
    <div v-if="showAuth" class="auth-overlay" @click.self="showAuth = false">
      <div class="auth-card">
        <button class="auth-close" @click="showAuth = false">✕</button>
        <div class="auth-tabs">
          <span :class="{ on: authTab === 'login' }" @click="authTab = 'login'">登录</span>
          <span :class="{ on: authTab === 'register' }" @click="authTab = 'register'">注册</span>
        </div>
        <input v-model="auth.account" placeholder="账号" class="auth-inp" />
        <input v-model="auth.password" type="password" placeholder="密码" class="auth-inp" />
        <input v-if="authTab === 'register'" v-model="auth.nickname" placeholder="昵称" class="auth-inp" />
        <div v-if="authTab === 'register'" class="gender-pick">
          <span class="gp-label">性别</span>
          <button class="gp-btn" :class="{on:auth.gender==='male'}" @click="auth.gender='male'">♂ 男</button>
          <button class="gp-btn" :class="{on:auth.gender==='female'}" @click="auth.gender='female'">♀ 女</button>
        </div>
        <p v-if="authError" class="auth-err">{{ authError }}</p>
        <button class="auth-btn" :disabled="authLoading" @click="doAuth">{{ authLoading ? '...' : (authTab === 'login' ? '登 录' : '注 册') }}</button>
      </div>
    </div>
  </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
// router from nuxt auto-import
import { apiFetch } from '../src/core/api'
// use Nuxt routing
function goGame() { window.location.href = '/game' }

function getIsNight(): boolean {
  const now = new Date()
  const hour = now.getHours()
  return hour >= 19 || hour < 7  // 19:00-06:59 = night
}
const isDark = ref(getIsNight())
const MANUAL_KEY = 'theme-manual'
const MODE_KEY = 'theme-mode'
let manualTheme = localStorage.getItem(MANUAL_KEY) === '1'
// Restore saved mode
const savedMode = localStorage.getItem(MODE_KEY)
if (savedMode) isDark.value = savedMode === 'dark'

function toggleTheme() {
  isDark.value = !isDark.value
  manualTheme = true
  localStorage.setItem(MANUAL_KEY, '1')
  localStorage.setItem(MODE_KEY, isDark.value ? 'dark' : 'light')
}

// Auto-update theme every minute based on time
let themeTimer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  themeTimer = setInterval(() => {
    if (manualTheme) return
    const shouldBeDark = getIsNight()
    if (isDark.value !== shouldBeDark) isDark.value = shouldBeDark
  }, 60000)
})
onUnmounted(() => { if (themeTimer) clearInterval(themeTimer) })

const onlineCount = ref(0)
const registeredCount = ref(0)
let pollTimer: ReturnType<typeof setInterval> | null = null
async function fetchCounts() {
  try { const r = await fetch('/health'); const d = await r.json(); if (typeof d.online === 'number') onlineCount.value = d.online; if (typeof d.registered === 'number') registeredCount.value = d.registered } catch {}
}
function fmt(n: number): string { return n >= 10000 ? (n / 10000).toFixed(1) + '万' : n.toLocaleString() }
const flipOnline = ref(false); const flipRegistered = ref(false)
watch(onlineCount, () => { flipOnline.value = true; setTimeout(() => flipOnline.value = false, 600) })
watch(registeredCount, () => { flipRegistered.value = true; setTimeout(() => flipRegistered.value = false, 600) })
onMounted(() => {
  fetchCounts()
  pollTimer = setInterval(fetchCounts, 10000)
  themeTimer = setInterval(() => {
    if (manualTheme) return
    const shouldBeDark = getIsNight()
    if (isDark.value !== shouldBeDark) isDark.value = shouldBeDark
  }, 60000)
})
onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
  if (themeTimer) clearInterval(themeTimer)
})

const showAuth = ref(false)
const authTab = ref<'login' | 'register'>('login')
const authLoading = ref(false)
const authError = ref('')
const auth = reactive({ account: '', password: '', nickname: '', gender: 'male' })
async function doAuth() {
  authError.value = ''
  if (!auth.account.trim()) { authError.value = '请输入账号'; return }
  if (!auth.password || auth.password.length < 6) { authError.value = '密码至少6位'; return }
  if (authTab.value === 'register' && !auth.nickname.trim()) { authError.value = '请输入昵称'; return }
  authLoading.value = true
  try {
    const endpoint = authTab.value === 'login' ? '/auth/login' : '/auth/register'
    const body: any = { account: auth.account, password: auth.password }
    if (authTab.value === 'register') { body.nickname = auth.nickname; body.gender = auth.gender }
    const data = await apiFetch(endpoint, { method: 'POST', body: JSON.stringify(body) })
    if (data.access_token) { localStorage.setItem('token', data.access_token); localStorage.setItem('player_id', String(data.player_id ?? 0)); if (data.refresh_token) localStorage.setItem('refresh_token', data.refresh_token); goGame() }
  } catch (e: any) { authError.value = e.message || '操作失败' } finally { authLoading.value = false }
}

const currentRealmIndex = 2
const realms = [{ name: '练气' }, { name: '筑基' }, { name: '金丹' }, { name: '元婴' }, { name: '化神' }, { name: '炼虚' }, { name: '合体' }, { name: '大乘' }, { name: '渡劫' }]

// 修仙语录轮播
const quotes = [
  '大道无形，生育天地',
  '修仙之路，步步登天',
  '一粒金丹吞入腹，始知我命不由天',
  '千淘万漉虽辛苦，吹尽狂沙始到金',
  '仙路漫漫，吾将上下而求索',
  '天行健，君子以自强不息',
]
const quoteIndex = ref(0)
const currentQuote = computed(() => quotes[quoteIndex.value])
onMounted(() => setInterval(() => { quoteIndex.value = (quoteIndex.value + 1) % quotes.length }, 5000))

</script>

<style lang="scss" scoped>
.landing-view { height: 100vh; display: flex; flex-direction: column; background: #0d0d1a; color: #e8e0d0; font-family: 'Noto Sans SC','PingFang SC','Microsoft YaHei',sans-serif; overflow: hidden; transition: background 0.4s,color 0.4s; }
.landing-view.light-mode { background: #fff; color: #1a1a1a; }
.gold-divider { flex-shrink: 0; height: 2px; background: linear-gradient(90deg,#b8860b,#d4a843,#f0d878,#d4a843,#b8860b); position: relative; overflow: hidden; }
.gold-divider__light { position: absolute; top: 0; left: -60%; width: 60%; height: 100%; background: linear-gradient(90deg,transparent,rgba(255,255,255,.15),#ff6b6b,#ffd93d,#6bcb77,#4d96ff,#9b59b6,#ff6b6b,rgba(255,255,255,.15),transparent); animation: rainbow-run 3s linear infinite; }
@keyframes rainbow-run { to { left: 100%; } }
.top-bar { flex-shrink: 0; background: #000; }
.light-mode .top-bar { background: #fff; border-bottom: 1px solid rgba(0,0,0,.06); }
.top-bar-inner { padding: 0 28px; height: 56px; display: flex; align-items: center; gap: 10px; }
.brand-logo { font-size: 40px; line-height: 1; color: #fff; animation: logo-spin 8s linear infinite; }
@keyframes logo-spin { to { transform: rotate(360deg); } }
.brand-name { font-size: 30px; font-weight: 900; letter-spacing: 6px; color: #fff; }
.top-bar-spacer { flex: 1; }
.player-stats { display: flex; flex-direction: column; align-items: flex-end; gap: 2px; line-height: 1.2; }
.online-badge { display: flex; align-items: center; gap: 5px; font-size: 14px; font-weight: 500; color: #fff; }
.online-dot { width: 6px; height: 6px; border-radius: 50%; background: #4caf50; animation: dot-pulse 2s ease-in-out infinite; }
@keyframes dot-pulse { 0%,100%{opacity:.4} 50%{opacity:1} }
.registered-badge { font-size: 14px; font-weight: 500; color: #fff; }
.theme-toggle { width: 52px; height: 52px; border: 2px solid #d4a843; border-radius: 50%; background: #000; color: #fff; font-size: 24px; cursor: pointer; display: flex; align-items: center; justify-content: center; box-shadow: 0 0 10px rgba(212,168,67,.25); transition: all .3s; }
.theme-toggle:hover { border-color: #fff; box-shadow: 0 0 20px rgba(212,168,67,.4); transform: scale(1.08); } .theme-toggle { -webkit-text-fill-color: #fff; }
.flip { display: inline-block; animation: num-flip .5s ease-out; }
@keyframes num-flip { 0%{transform:translateY(-6px);opacity:.3} 50%{transform:translateY(2px);opacity:1} to{transform:translateY(0);opacity:1} }
.main-grid { flex: 1; display: grid; place-items: center; overflow: hidden; width: 100%; }
.left-col { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 28px; padding: 20px 0; width: 100%; text-align: center; }
.yin-yang { --yy-size: 226px; width: var(--yy-size); height: var(--yy-size); border-radius: 50%; margin: 0 auto; background: linear-gradient(to right,#fff 50%,#000 50%); position: relative; animation: spin-yy 8s linear infinite; cursor: pointer; transition: transform .4s cubic-bezier(.175,.885,.32,1.275); box-shadow: 0 0 30px rgba(255,255,255,.15),0 0 60px rgba(0,0,0,.4); &:active { transform: scale(1.15); box-shadow: 0 0 50px rgba(255,255,255,.3); } &::before,&::after { content: ''; position: absolute; border-radius: 50%; } &::before { width: 50%; height: 50%; top: 0; left: 25%; background: #fff; background-image: radial-gradient(circle at 50% 62%,#000 24%,transparent 24%); } &::after { width: 50%; height: 50%; top: 50%; left: 25%; background: #000; background-image: radial-gradient(circle at 50% 38%,#fff 24%,transparent 24%); } }
@keyframes spin-yy { to { transform: rotate(360deg); } }
.game-title { margin: 0; font-size: 80px; font-weight: 900; letter-spacing: 14px; line-height: 1.1; background: linear-gradient(135deg,#d4a843,#f0d878 30%,#d4a843 60%,#b8942e); background-size: 200% 200%; -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; animation: gold-shimmer 4s ease-in-out infinite; }
@keyframes gold-shimmer { 0%,to{background-position:0 50%} 50%{background-position:100% 50%} }
.game-subtitle { margin: 4px 0 0; font-size: 24px; font-weight: 500; color: rgba(232,224,208,.6); letter-spacing: 12px; }
.light-mode .game-subtitle { color: rgba(0,0,0,.55); }
.light-mode .brand-logo, .light-mode .brand-name, .light-mode .online-badge, .light-mode .registered-badge { color: #000; }
.light-mode .realm-dot { background: #fff !important; }
.light-mode .realm-label { color: rgba(0,0,0,.35) !important; }
.light-mode .realm-node.completed .realm-label, .light-mode .realm-node.active .realm-label { color: #000 !important; }
.landing-view.light-mode .theme-toggle { background: #000; color: #fff !important; border-color: #b8860b; }
.realm-progress { display: flex; align-items: flex-start; position: relative; width: 100%; max-width: 440px; padding: 0 8px; }
.realm-track-bg { position: absolute; top: 14px; left: 16px; right: 16px; height: 2px; background: rgba(232,224,208,.15); border-radius: 1px; }
.realm-node { display: flex; flex-direction: column; align-items: center; gap: 6px; flex: 1; cursor: pointer; opacity: 0; transform: translateX(-20px); animation: realm-in .5s ease-out forwards; }
@keyframes realm-in { to { opacity: 1; transform: translateX(0); } }
.realm-dot { width: 24px; height: 24px; border-radius: 50%; border: 2px solid rgba(232,224,208,.2); background: #0d0d1a; display: flex; align-items: center; justify-content: center; transition: all .4s; }
.realm-node.completed .realm-dot { border-color: #d4a843; background: #d4a843; }
.realm-node.active .realm-dot { border-color: #d4a843; background: #141428; box-shadow: 0 0 0 3px rgba(212,168,67,.25); }
.realm-check { color: #fff; font-size: 12px; font-weight: 700; }
.realm-pulse { width: 8px; height: 8px; border-radius: 50%; background: #d4a843; animation: realm-pulse 2s ease-in-out infinite; }
@keyframes realm-pulse { 0%,to{opacity:.4;transform:scale(.8)} 50%{opacity:1;transform:scale(1.2)} }
.realm-label { font-size: 13px; color: rgba(232,224,208,.35); }
.realm-node.completed .realm-label, .realm-node.active .realm-label { color: #d4a843; }
.enter-btn { margin-top: 8px; padding: 14px 48px; font-size: 20px; font-weight: 700; letter-spacing: 4px; color: #fff; background: linear-gradient(135deg,#d4a843,#f0d060); border: none; border-radius: 30px; cursor: pointer; box-shadow: 0 4px 20px rgba(212,168,67,.3); transition: all .3s; }
.enter-btn:hover { transform: translateY(-2px); box-shadow: 0 8px 30px rgba(212,168,67,.5); }
.cultivation-quote { margin: 0; font-size: 14px; color: rgba(212,168,67,.5); letter-spacing: 4px; font-style: italic; animation: quote-fade 5s ease-in-out infinite; }
@keyframes quote-fade { 0%,100%{opacity:.4} 50%{opacity:1} }
.light-mode .game-title { text-shadow: 0 2px 8px rgba(0,0,0,.15); }
.bottom-bar { flex-shrink: 0; background: rgba(13,13,26,.95); padding: 12px; text-align: center; border-top: 1px solid rgba(212,168,67,.12); }
.copyright { font-size: 12px; color: rgba(232,224,208,.35); letter-spacing: 2px; }

@media (max-width: 768px) {
  .yin-yang { --yy-size: 140px !important; }
  .game-title { font-size: 48px !important; letter-spacing: 8px !important; }
  .game-subtitle { font-size: 16px !important; letter-spacing: 6px !important; }
  .realm-progress { max-width: 340px !important; }
  .realm-dot { width: 18px !important; height: 18px !important; }
  .realm-label { font-size: 11px !important; }
  .enter-btn { padding: 12px 36px !important; font-size: 17px !important; }
  .brand-logo { font-size: 28px !important; }
  .brand-name { font-size: 22px !important; }
  .top-bar-inner { padding: 0 16px !important; height: 48px !important; }
  .theme-toggle { width: 40px !important; height: 40px !important; font-size: 18px !important; }
  .online-badge, .registered-badge { font-size: 12px !important; }
}

@media (max-width: 400px) {
  .yin-yang { --yy-size: 110px !important; }
  .game-title { font-size: 36px !important; letter-spacing: 6px !important; }
  .game-subtitle { font-size: 14px !important; letter-spacing: 4px !important; }
  .realm-progress { max-width: 290px !important; }
  .realm-label { font-size: 10px !important; }
  .enter-btn { padding: 10px 28px !important; font-size: 15px !important; }
}

</style>

<style lang="scss">

.auth-overlay {
  position: fixed; inset: 0; z-index: 2000;
  display: flex; align-items: center; justify-content: center;
  background: rgba(0,0,0,.5); backdrop-filter: blur(8px);
}
.auth-card {
  background: #111; border: 1px solid rgba(212,168,67,.15);
  border-radius: 16px; width: 400px; max-width: 90vw;
  padding: 48px 40px 36px; position: relative;
  box-shadow: 0 32px 64px rgba(0,0,0,.6), 0 0 40px rgba(212,168,67,.05);
  animation: auth-pop .3s ease;
}
@keyframes auth-pop { from { opacity: 0; transform: translateY(16px); } to { opacity: 1; transform: translateY(0); } }
.auth-close {
  position: absolute; top: 16px; right: 16px;
  background: none; border: none; color: rgba(255,255,255,.3); font-size: 22px; cursor: pointer;
  transition: color .2s;
  &:hover { color: #fff; }
}
.auth-tabs {
  display: flex; gap: 32px; margin-bottom: 36px;
  span {
    padding-bottom: 8px; font-size: 16px; color: rgba(255,255,255,.35); cursor: pointer;
    border-bottom: 2px solid transparent; transition: all .2s;
    &.on { color: #d4a843; border-bottom-color: #d4a843; font-weight: 600; }
  }
}
.auth-inp {
  display: block; width: 100%; padding: 14px 0; margin-bottom: 16px;
  border: none; border-bottom: 1px solid rgba(255,255,255,.1);
  background: transparent; color: #fff; font-size: 15px; outline: none;
  transition: border-color .2s;
  &::placeholder { color: rgba(255,255,255,.25); }
  &:focus { border-bottom-color: #d4a843; }
}
.auth-err { color: #ff453a; font-size: 13px; text-align: center; margin: 8px 0 0; }
.gender-pick { display: flex; align-items: center; gap: 10px; margin: 8px 0 16px; }
.gp-label { color: rgba(255,255,255,.4); font-size: 14px; }
.gp-btn { padding: 8px 20px; border: 1px solid rgba(212,168,67,.2); border-radius: 20px; background: transparent; color: rgba(255,255,255,.4); font-size: 14px; cursor: pointer; transition: all .2s; font-family: inherit; }
.gp-btn:hover { border-color: #d4a843; color: #d4a843; }
.gp-btn.on { background: rgba(212,168,67,.15); border-color: #d4a843; color: #d4a843; font-weight: 700; }
.auth-btn {
  display: block; width: 100%; margin-top: 32px; padding: 16px;
  border: none; background: linear-gradient(135deg,#d4a843,#c9952e);
  color: #fff; font-size: 17px; font-weight: 600; letter-spacing: 6px; cursor: pointer;
  transition: opacity .2s;
  &:hover { opacity: .9; }
  &:disabled { opacity: .4; }
}
</style>