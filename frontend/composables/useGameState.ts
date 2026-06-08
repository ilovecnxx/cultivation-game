// 修仙世界 — 游戏核心状态和逻辑（从 game.vue 提取）
import { menus, descs, realmNames, realmCoefs, rootMults, qualityNames, qualityColors, pillQualityColors, rootNames, mapRegions, fmt, wikiRealms, wikiSpiritReqs, wikiRootBonuses, wikiQuality } from './useGameData'

export function useGameState() {
  // ====== 基础状态 ======
  const isDark = ref(true)
  const activeNav = ref('')
  function toggleTheme() { isDark.value = !isDark.value; localStorage.setItem('theme-mode', isDark.value ? 'dark' : 'light'); document.documentElement.className = isDark.value ? '' : 'light-mode' }
  const getToken = () => localStorage.getItem('token') || ''
  const getPID = () => localStorage.getItem('player_id') || '0'
  async function refreshToken() { try { const r = await fetch('/auth/refresh', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ refresh_token: localStorage.getItem('refresh_token') || '' }) }); const d = await r.json(); if (d.access_token) { localStorage.setItem('token', d.access_token); return d.access_token } } catch {} return null }

  // ====== UI 状态 ======
  const activeMenu = ref<any>(null)
  const showPigeon = ref(false)
  const modalVisible = ref(false)
  const activeSub = ref('')
  const modalDesc = ref('')
  const activeSubLabel = computed(() => activeMenu.value?.children?.find((s: any) => s.key === activeSub.value)?.label || '')
  function openMenu(m: any) { activeMenu.value = m; activeSub.value = m.children?.[0]?.key || m.key; modalDesc.value = descs[activeSub.value] || ''; if (m.key === 'pigeon') { showPigeon.value = true; return }
    if (!['map', 'profession', 'backpack'].includes(m.key)) modalVisible.value = true }

  // ====== 玩家状态 ======
  const player = reactive({
    name: '修仙者', gender: 'male', realmName: '锻体', realmId: 1, realmStage: 1, spiritName: '无灵根', qualityName: '无品', rootQuality: 0,
    level: 1, power: 0, hp: 100, maxHp: 100, mp: 50, maxMp: 50, attack: 10, defense: 5, speed: 100,
    critRate: 3, critDmg: 150, dodge: 2, hit: 95, cultBonus: 0, breakBonus: 0, mpRegen: 3, lifespan: 100,
    comprehension: 10, luck: 10, spiritSense: 100, spirit: 0, maxSpirit: 100, gold: 0, jade: 0,
    isMeditating: false, cultRate: 11, breakRate: 90,
  })
  const hpPct = computed(() => Math.min(100, Math.round((player.hp / player.maxHp) * 100)))
  const mpPct = computed(() => Math.min(100, Math.round((player.mp / player.maxMp) * 100)))
  const yySpeed = computed(() => player.isMeditating ? 2 : 3)
  const ageBracket = computed(() => player.level < 10 ? '幼年' : player.level < 30 ? '少年' : player.level < 60 ? '青年' : player.level < 80 ? '中年' : '老年')
  const ageDays = computed(() => player.level * 30)

  async function loadPlayer() { const pid = getPID(); if (!pid) return; try { const t = getToken(); const r = await fetch('/api/v1/player/' + pid, { headers: { Authorization: 'Bearer ' + t } }); const d = await r.json(); if (d.data) { Object.assign(player, d.data); if (d.data.spiritName) player.spiritName = d.data.spiritName } } catch {} }

  // ====== 修炼状态 ======
  const training = ref(false)
  const trainMult = ref(1)
  const logs = reactive<any[]>([])

  // ====== 丹药 ======
  const showPillPanel = ref(false)
  const showPillCraft = ref(false)
  const myPills = ref<any[]>([])
  const pillRecipes = ref<any[]>([])
  const pillCount = computed(() => myPills.value.reduce((s: number, p: any) => s + p.quantity, 0))
  const pillCat = ref('all')
  const pillQtys = reactive<Record<string, number>>({})
  const craftResult = ref<any>(null)
  const pillStats = reactive({ crafted: 0, failed: 0 })
  const pillCats = [{ key: 'all', label: '全部' }, { key: '恢复', label: '💚恢复' }, { key: '修炼', label: '🧘修炼' }, { key: '战斗', label: '⚔️战斗' }, { key: '防护', label: '🛡️防护' }, { key: '气运', label: '🍀气运' }, { key: '特殊', label: '🎲特殊' }]
  const filteredRecipes = computed(() => pillCat.value === 'all' ? pillRecipes.value : pillRecipes.value.filter((r: any) => r.category === pillCat.value))
  async function loadPills() { const pid = getPID(); if (!pid) return; try { const r = await fetch('/api/v1/player/' + pid + '/pills', { headers: { Authorization: 'Bearer ' + getToken() } }); const d = await r.json(); myPills.value = d.data || [] } catch {} }
  async function loadRecipes() { try { const r = await fetch('/api/v1/pills/recipes', { headers: { Authorization: 'Bearer ' + getToken() } }); const d = await r.json(); pillRecipes.value = d.data || [] } catch {} }
  function craftAgain() { const r = filteredRecipes.value.find((r: any) => r.pill_key === craftResult.value?.key); if (r) craftPill(r, pillQtys[r.pill_key] || 1) }
  async function craftPill(recipe: any, qty: number = 1) {
    const pid = getPID(); if (!pid) return; let sc = 0
    for (let i = 0; i < Math.min(qty, 20); i++) {
      try { const d = await apiPost('/api/v1/player/' + pid + '/pills/craft', { pill_key: recipe.pill_key }); if (d && d.data && d.data.success) { sc++; craftResult.value = d.data } else { craftResult.value = { success: false, key: recipe.pill_key } } } catch { craftResult.value = { success: false, key: recipe.pill_key } }
    }
    pillStats.crafted += sc; pillStats.failed += (qty - sc); loadPills(); loadRecipes()
  }
  async function usePill(pill: any) { const pid = getPID(); if (!pid) return; const d = await apiPost('/api/v1/player/' + pid + '/pills/use', { pill_id: pill.id }); if (d && d.data && d.data.success) { addLog('item', '💊 使用 ' + pill.name + '·' + pill.quality_name); loadPills(); loadPlayer() } }

  // ====== 地点 ======
  const activeLoc = ref('')
  const currentLoc = ref(localStorage.getItem('cur_loc') || 'qys')
  const currentLocInfo = computed(() => { for (const r of mapRegions) for (const l of r.locations) if (l.key === currentLoc.value) return l; return null })
  function enterLocation(loc: any) { currentLoc.value = loc.key; localStorage.setItem('cur_loc', loc.key); addLog('explore', '📍 前往 ' + loc.name); activeMenu.value = null }

  // ====== 百科 ======
  const showWiki = ref(false)
  const wikiTab = ref('realm')
  const wikiTabs = [{ key: 'realm', label: '境界体系' }, { key: 'root', label: '灵根体系' }, { key: 'attrs', label: '战斗属性' }, { key: 'innate', label: '先天属性' }, { key: 'equip', label: '装备体系' }]

  // ====== 背包 ======
  const bpItems = ref<any[]>([])
  async function loadBackpack() { const pid = getPID(); if (!pid) return; try { const r = await fetch('/api/v1/player/' + pid + '/inventory', { headers: { Authorization: 'Bearer ' + getToken() } }); const d = await r.json(); bpItems.value = d.data || [] } catch {} }

  // ====== 日志 ======
  const logFilter = ref('all')
  const logLocked = ref(false)
  const logBody = ref<HTMLElement | null>(null)
  const filteredLogs = computed(() => logFilter.value === 'all' ? logs : logs.filter((l: any) => l.type === logFilter.value))
  function addLog(type: string, text: string) { logs.push({ time: new Date().toLocaleTimeString('zh-CN', { hour12: false }).slice(0, 5), type, text }); if (logs.length > 500) logs.shift() }
  function loadLogs() { try { const d = localStorage.getItem('game_logs'); if (d) logs.push(...JSON.parse(d)) } catch {} }
  function saveLogs() { try { localStorage.setItem('game_logs', JSON.stringify(logs.slice(-200))) } catch {} }

  // ====== 在线统计 ======
  const onlineCount = ref(0)
  const registeredCount = ref(0)
  async function fetchStats() { try { const r = await fetch('/health'); const d = await r.json(); if (d.online !== undefined) onlineCount.value = d.online; if (d.registered !== undefined) registeredCount.value = d.registered } catch {} }

  // ====== 离线收益 ======
  function calcOfflineGains() { try { const last = localStorage.getItem('last_seen'); const now = Date.now(); if (last) { const diff = Math.min((now - parseInt(last)) / 1000, 86400 * 7); if (diff > 60) { const exp = Math.floor(diff * 5); addLog('system', '⏳ 离线 ' + Math.floor(diff / 3600) + ' 小时, 获得 ' + fmt(exp) + ' 修为') } } localStorage.setItem('last_seen', String(now)) } catch {} }

  // ====== 修炼函数 ======
  function toggleMeditation() { player.isMeditating = !player.isMeditating; addLog('system', player.isMeditating ? '🧘 开始闭关修炼' : '🚪 出关') }
  const smallBreakRate = computed(() => Math.min(95, Math.round(50 + (player.cultBonus || 0) * 0.5 + player.luck * 0.2)))
  function doBreakthrough() { addLog('system', '⚡ 尝试突破...'); player.breakRate = Math.min(95, player.breakRate + 5); saveLogs() }
  function toggleTraining() { training.value = !training.value; addLog('combat', training.value ? '⚔️ 开始历练' : '🛑 停止历练') }
  function startManual() { addLog('combat', '⚡ 手动历练: 获得 ' + fmt(trainMult.value * 10) + ' 经验'); saveLogs() }
  function showTooltip() {}
  const showCultTooltip = ref(false)
  const tooltipStyle = reactive({ top: '0px', left: '0px' })

  // ====== 战斗 / PVE ======
  const fightInProgress = ref(false)
  const fightResult = ref<any>(null)
  const pveReport = ref<any>(null)
  const pveRounds = ref(0)
  const encounterResult = ref<any>(null)
  const encounterType = ref('')
  const deathLog = ref<any[]>([])
  const reviveCountdown = ref(0)
  const isDead = computed(() => player.hp <= 0)
  const handleQuit = () => { localStorage.clear(); window.location.href = '/' }
  function startPve() { fightInProgress.value = true; pveReport.value = null; pveRounds.value = 0 }

  // ====== 职业 ======
  const showProfPanel = ref(false)
  const activeProf = ref(false)

  // ====== 百科数据 ======
  const realmCoef = computed(() => realmCoefs[player.realmId] || 1)
  const rootMult = computed(() => rootMults[player.rootQuality] || 1)
  const cultBaseVal = computed(() => (10 + realmCoef.value * (player.realmStage - 1)) * rootMult.value)

  // ====== 时间显示 ======
  const timeDisplay = ref('')
  const uptimeDisplay = ref('')
  function getTimeDisplay() { const n = new Date(); return n.toLocaleTimeString('zh-CN', { hour12: false }) }
  function getUptimeDisplay() { const diff = Math.floor((Date.now() - Date.now()) / 1000); const h = Math.floor(diff / 3600); const m = Math.floor((diff % 3600) / 60); return h + 'h' + m + 'm' }

  // ====== 好友 ======
  const friends = ref<any[]>([])
  const pendingRequests = ref<any[]>([])
  const searchResults = ref<any[]>([])
  const friendSearch = ref('')
  const activePeer = ref(0)
  const activePeerName = ref('')
  const privateInput = ref('')
  const privateMessages = ref<any[]>([])
  async function loadFriends() { const pid = getPID(); if (!pid) return; try { const r = await fetch('/api/v1/player/' + pid + '/friends', { headers: { Authorization: 'Bearer ' + getToken() } }); const d = await r.json(); friends.value = d.data || [] } catch {} }
  async function loadPending() { const pid = getPID(); if (!pid) return; try { const r = await fetch('/api/v1/player/' + pid + '/friends/pending', { headers: { Authorization: 'Bearer ' + getToken() } }); const d = await r.json(); pendingRequests.value = d.data || [] } catch {} }
  async function searchPlayers() { const pid = getPID(); if (!pid || !friendSearch.value.trim()) return; try { const r = await fetch('/api/v1/player/' + pid + '/friends/search?q=' + encodeURIComponent(friendSearch.value), { headers: { Authorization: 'Bearer ' + getToken() } }); const d = await r.json(); searchResults.value = d.data || [] } catch {} }
  async function removeFriend(fid: number) { const pid = getPID(); if (!pid) return; await apiPost('/api/v1/player/' + pid + '/friends/remove', { friend_id: fid }); loadFriends(); loadPending() }
  async function openChat(f: any) { activePeer.value = f.id; activePeerName.value = f.nickname; const pid = getPID(); if (!pid) return; try { const r = await fetch('/api/v1/player/' + pid + '/messages?peer_id=' + f.id, { headers: { Authorization: 'Bearer ' + getToken() } }); const d = await r.json(); privateMessages.value = (d.data || []).reverse() } catch {} }
  async function sendPrivate() { const pid = getPID(); const v = privateInput.value.trim(); if (!v || !activePeer.value || !pid) return; const d = await apiPost('/api/v1/player/' + pid + '/messages/send', { to_id: activePeer.value, text: v }); if (d && d.code === 0) { privateMessages.value.push({ from_id: parseInt(pid), to_id: activePeer.value, text: v, created_at: new Date().toISOString() }); privateInput.value = '' } }

  // ====== 装备 ======
  const playerEquips = ref<any[]>([])
  const equipCraftSlot = ref('')
  function getEquip(slot: string) { return playerEquips.value.find((e: any) => e.slot === slot) }
  async function loadEquips() { const pid = getPID(); if (!pid) return; try { const r = await fetch('/api/v1/player/' + pid + '/equipment', { headers: { Authorization: 'Bearer ' + getToken() } }); const d = await r.json(); playerEquips.value = d.data || [] } catch {} }

  // ====== WS ======
  let ws: WebSocket | null = null
  function connectWS() {
    const t = getToken(); if (!t) return
    try { ws = new WebSocket((location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/ws?token=' + t); ws.onclose = () => { setTimeout(connectWS, 5000) }; ws.onerror = () => { ws?.close() } } catch {}
  }

  // ====== Helper ======
  async function apiPost(url: string, body: any) { try { const r = await fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + getToken() }, body: JSON.stringify(body) }); if (r.status === 401) { const nt = await refreshToken(); if (nt) { const r2 = await fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + nt }, body: JSON.stringify(body) }); return await r2.json() } } return await r.json() } catch { return null } }

  // ====== 初始化 ======
  onMounted(() => { fetchStats(); setInterval(fetchStats, 30000); loadLogs(); loadPlayer(); loadPills(); loadRecipes(); loadBackpack(); loadFriends(); loadPending(); calcOfflineGains(); connectWS(); setTimeout(() => { addLog('system', '🌟 登录修仙世界') }, 500) })

  return {
    isDark, activeNav, toggleTheme, getToken, getPID, refreshToken,
    menus, descs, realmNames, realmCoefs, rootMults, qualityNames, qualityColors, pillQualityColors, rootNames, mapRegions, fmt,
    activeMenu, modalVisible, activeSub, modalDesc, activeSubLabel, openMenu, showPigeon,
    player, isDead, hpPct, mpPct, yySpeed, ageBracket, ageDays, loadPlayer,
    training, trainMult, logs,
    showPillPanel, showPillCraft, myPills, pillRecipes, pillCount, pillCat, pillQtys, craftResult, pillStats, pillCats, filteredRecipes, loadPills, loadRecipes, craftAgain, craftPill, usePill,
    activeLoc, currentLoc, currentLocInfo, enterLocation,
    showWiki, wikiTab, wikiTabs,
    bpItems, loadBackpack,
    addLog, loadLogs, saveLogs, logFilter, logLocked, logBody, filteredLogs,
    onlineCount, registeredCount, fetchStats,
    calcOfflineGains,
    toggleMeditation, smallBreakRate, doBreakthrough, toggleTraining, startManual, showTooltip, showCultTooltip, tooltipStyle,
    fightInProgress, fightResult, pveReport, pveRounds, encounterResult, encounterType, deathLog, reviveCountdown, isDead, handleQuit, startPve,
    showProfPanel, activeProf,
    wikiRealms, wikiRootBonuses, wikiSpiritReqs, wikiQuality, realmCoef, rootMult, cultBaseVal,
    timeDisplay, uptimeDisplay, getTimeDisplay, getUptimeDisplay,
    friends, pendingRequests, searchResults, friendSearch, activePeer, activePeerName, privateInput, privateMessages, loadFriends, loadPending, searchPlayers, removeFriend, openChat, sendPrivate,
    playerEquips, equipCraftSlot, getEquip, loadEquips,
    connectWS, apiPost,
  }
}
