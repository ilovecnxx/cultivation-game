<template>
  <div class="gh-root" :class="{ 'light-mode': !isDark }">
    <div class="gold-divider"><div class="gold-divider__light" /></div>
    <header class="top-bar">
      <div class="top-bar-inner">
        <span class="brand-logo">☯</span>
        <span class="brand-name">修仙世界</span>
        <nav class="main-nav">
          <div class="nav-item" @click="showWiki=true"><span class="nav-label">📖 百科</span></div>
          <div v-for="m in menus" :key="m.key" class="nav-item" @click="openMenu(m)"><span class="nav-label">{{ m.label }}</span></div>
        </nav>
        <div class="top-bar-spacer"></div>
        <div class="player-stats">
          <span class="online-badge"><span class="online-dot" />{{ fmt(onlineCount) }} 在线修士</span>
          <span class="registered-badge">{{ fmt(registeredCount) }} 注册修士</span>
        </div>
        <button class="theme-toggle" @click="toggleTheme">{{ isDark ? '☀' : '🌙' }}</button>
      </div>
    </header>
    <div class="gold-divider"><div class="gold-divider__light" /></div>
    <!-- 公告/世界信息栏 -->
    <div class="notice-bar">
      <span class="notice-icon">📢</span>
      <span class="notice-text">【宗门大战】每周六20:00开启 · 【灵脉争夺】修炼速度+20% · 【世界Boss】夔牛即将刷新 · 新道友不断涌入修仙世界</span>
    </div>
    <div class="gold-divider"><div class="gold-divider__light" /></div>
    <main class="gh-main">
      <!-- 右1/6: 侧边栏 -->
      <aside class="gh-sidebar">
        <div class="side-card side-profile">
          <div class="profile-top">
            <div class="profile-top-left">
              <div class="profile-name-row">
                <span class="side-avatar">{{ player.name?.[0] || '修' }}</span>
                <span class="side-name">{{ player.name || '修仙者' }}<span v-if="isDead" class="dead-skull">💀</span><span class="gender-tag" :style="{color:player.gender==='female'?'#ff6b6b':'#4d96ff'}">{{ player.gender==='female'?'♀':'♂' }}</span></span>
              </div>
              <div class="profile-tags">
                <span class="side-realm">{{ player.realmName }} {{ player.realmStage }}期</span>
                <span v-if="player.spiritName!=='无灵根'" class="side-spirit" :style="{color:qualityColors[player.rootQuality]||'#fff'}">{{ player.spiritName }} · {{ player.qualityName }}</span>
                <span v-else class="side-spirit" style="color:#666">灵根未觉醒</span>
              </div>
              <div class="profile-loc" v-if="currentLocInfo"><span class="pl-icon">{{ currentLocInfo.icon }}</span><span>{{ currentLocInfo.name }}</span></div>
            </div>
            <div class="profile-power-badge">
              <span class="pp-label">战力</span>
              <span class="pp-val">{{ fmt(player.power) }}</span>
            </div>
          </div>
          <div class="profile-bars">
            <div class="pbar-row"><span>❤️</span><div class="pbar-track"><div class="pbar-fill hp" :style="{width:hpPct+'%'}" /></div><span class="pbar-val">{{ player.hp }}/{{ player.maxHp }}</span></div>
            <div class="pbar-row"><span>💙</span><div class="pbar-track"><div class="pbar-fill mp" :style="{width:mpPct+'%'}" /></div><span class="pbar-val">{{ player.mp }}/{{ player.maxMp }}</span></div>
          </div>
          <div class="profile-attrs">
            <div class="pa-row"><span>🗡️ 攻击</span><span>{{ player.attack }}</span></div>
            <div class="pa-row"><span>🛡️ 防御</span><span>{{ player.defense }}</span></div>
            <div class="pa-row"><span>💨 速度</span><span>{{ player.speed }}</span></div>
            <div class="pa-row"><span>💥 暴击</span><span>{{ player.critRate }}%</span></div>
            <div class="pa-row"><span>💢 暴伤</span><span>{{ player.critDmg }}%</span></div>
            <div class="pa-row"><span>🎯 命中</span><span>{{ player.hit }}%</span></div>
            <div class="pa-row"><span>💨 闪避</span><span>{{ player.dodge }}%</span></div>
            <div class="pa-row"><span>🌀 修炼</span><span>+{{ player.cultBonus }}%</span></div>
            <div class="pa-row"><span>💧 回蓝</span><span>{{ player.mpRegen }}%/回合</span></div>
            <div class="pa-row"><span>⏳ 寿元</span><span>{{ player.lifespan }}年</span></div>
            <div class="pa-row"><span>🎂 年龄</span><span>{{ ageDays }}天({{ ageBracket }})</span></div>
            <div class="pa-row"><span>📖 悟性</span><span>{{ player.comprehension }}</span></div>
            <div class="pa-row"><span>🍀 气运</span><span>{{ player.luck }}</span></div>
            <div class="pa-row"><span>👁️ 神识</span><span>{{ player.spiritSense }}</span></div>
            <div class="pa-row"><span>💰 灵石</span><span>{{ player.gold }}</span></div>
            <div class="pa-row"><span>💎 仙玉</span><span>{{ player.jade }}</span></div>
          </div>
        </div>
        <div class="side-loc" v-if="currentLocInfo"><span class="sl-icon">{{ currentLocInfo.icon }}</span><span class="sl-name">{{ currentLocInfo.name }}</span><span class="sl-monsters">🐾 {{ currentLocInfo.monsters }}</span></div>
      <div class="side-card side-cultivation">
          <div ref="yyWrapRef" class="cult-yy-wrap" :class="{meditating:player.isMeditating}" @mouseenter="showTooltip" @mouseleave="showCultTooltip=false">
            <div class="cult-yy" :class="[player.gender==='female'?'female':'male']" :style="{animationDuration:yySpeed+'s'}" />
            <canvas ref="energyCanvas" v-if="player.isMeditating" class="energy-canvas" />
          </div>
          <div class="cult-info">
            <div class="cult-bar-wrap">
              <div class="cult-bar-label">修为</div>
              <div class="cult-bar"><div class="cult-fill" :style="{width:expPercent+'%'}" /></div>
            </div>
            <div class="cult-stats"><span>{{ player.spirit }}/{{ player.maxSpirit }}</span><span>+{{ player.cultRate.toFixed(1) }}/秒</span></div>
          </div>
          <div class="cult-btns">
            <button class="cult-btn primary" @click="toggleMeditation()">{{ player.isMeditating?'出关':'闭关' }}</button>
            <button class="cult-btn" :class="{danger:player.breakRate<60}" @click="doBreakthrough()">{{ player.realmStage<10?'突破 '+smallBreakRate+'%':'突破 '+player.breakRate+'%' }}</button>
            <button class="cult-btn train-btn" @click="toggleTraining()">{{ training?'历练中':'历练' }}</button>
          </div>
          <div class="train-panel" v-if="showTrainPanel">
            <div class="train-info">
              <span>👁️ 神识: {{ player.spiritSense }}</span>
              <span>倍率: <button v-for="m in [1,2,5,10]" :key="m" class="mult-btn" :class="{on:trainMult===m}" @click="trainMult=m" :disabled="training||isDead">×{{ m }}</button></span>
              <button class="manual-train-btn" @click="startManual()" :disabled="training||isDead||player.spiritSense<trainMult">⚡ 历练一次</button>
            </div>
            <label class="train-switch" @click.prevent="toggleTraining()"><input type="checkbox" :checked="training" :disabled="isDead||player.spiritSense<trainMult" tabindex="-1" /> 自动历练 (每3秒)</label>
            <div v-if="training" class="train-info" style="color:#6bcb77">预计: +{{ trainCultEst }}修为 +{{ trainGoldEst }}灵石/次 (消耗{{ trainMult }}神识)</div>
            <div v-if="isDead" class="dead-warn">💀 {{ player.gender==='female'?'香消玉殒':'道心破碎' }} · {{ reviveCountdown }}秒后复活</div>
            <!-- 战斗报告 -->
            <div v-if="combatReport" class="combat-report">
              <div class="cr-header">⚔️ 遭遇战斗 — {{ combatReport.result }}</div>
              <div v-for="r in combatReport.rounds" :key="r.round" class="cr-round">
                <span>第{{ r.round }}回合:</span>
                <span class="cr-dmg">你对敌人造成 {{ r.playerDmg }} 伤害</span>
                <span v-if="r.desc" class="cr-special">{{ r.desc }}</span>
                <span class="cr-dmg">敌人对你造成 {{ r.monsterDmg }} 伤害</span>
                <span class="cr-hp">❤️{{ r.playerHP }} 💙{{ r.playerMP>0?r.playerMP:'空蓝' }} | 👹{{ r.monsterHP>0?r.monsterHP:'击败！' }}</span>
              </div>
            </div>
          </div>
        </div>
        <div class="side-card side-equip">
          <div class="equip-slots">
            <div v-for="s in equipSlots" :key="s.key" class="eq-slot" :style="{borderColor:getEquip(s.key)?'#d4a843':'rgba(212,168,67,.25)'}" @click="craftEquip(s)" :title="getEquip(s.key)?getEquip(s.key).name:'点击打造'">
              <span>{{ getEquip(s.key)?.icon||s.icon }}</span>
              <span style="font-size:10px">{{ getEquip(s.key)?.name||s.name }}</span>
            </div>
          </div>
        </div>
      </aside>

      <div class="gh-center"></div>
    
      <!-- 日志面板 -->
      <aside class="gh-panel gh-log">
        <div class="panel-header">
          <h4>📜 日志</h4>
          <div class="panel-actions panel-actions-center">
            <button class="pa-btn" :class="{active:logFilter==='all'}" @click="logFilter='all'">全部</button>
            <button class="pa-btn" :class="{active:logFilter==='combat'}" @click="logFilter='combat'">⚔️ 战斗</button>
            <button class="pa-btn" :class="{active:logFilter==='cult'}" @click="logFilter='cult'">🧘 修炼</button>
            <button class="pa-btn" :class="{active:logFilter==='item'}" @click="logFilter='item'">🎒 物品</button>
            <button class="pa-btn" :class="{active:logFilter==='explore'}" @click="logFilter='explore'">📍 探索</button>
            <button class="pa-btn" :class="{active:logFilter==='quest'}" @click="logFilter='quest'">📜 任务</button>
            <button class="pa-btn" :class="{active:logFilter==='system'}" @click="logFilter='system'">⚙️ 系统</button>
          </div>
          <div class="panel-actions panel-actions-right">
            <button class="pa-btn" @click="logLocked=!logLocked" :title="logLocked?'解锁滚动':'锁定滚动'">{{ logLocked?'🔓':'🔒' }}</button>
            <button class="pa-btn" @click="logs.splice(0)" title="清屏">🗑️</button>
          </div>
        </div>
        <div class="panel-body" ref="logBody">
          <div v-for="(l,i) in filteredLogs" :key="i" class="log-entry" :class="l.type">{{ l.time }} {{ l.text }}</div>
          <div v-if="filteredLogs.length===0" class="log-empty">暂无日志</div>
        </div>
      </aside>

      
      <!-- 社交面板（ChatPanel 组件） -->
      <ChatPanel :player-name="player.name || '修仙者'" />

      


    </main>
    <footer class="bottom-bar">
      <div class="footer-left">
        <span class="footer-time">⏳ {{ timeDisplay }}</span>
        <span class="footer-divider">|</span>
        <span class="footer-uptime">🖥️ 已运行 {{ uptimeDisplay }}</span>
      </div>
      <div class="footer-right">
        <button class="footer-quit" @click="handleQuit">退隐江湖</button>
      </div>
    </footer>
    <Teleport to="body">
      <div v-if="modalVisible&&activeMenu?.key==='profession'" class="modal-overlay" @click.self="modalVisible=false">
        <div class="wiki-modal">
          <div class="wiki-header"><h2>{{ activeProf ? '🔥 丹师·炼丹' : '🎯 职业系统' }}</h2><button class="modal-close" @click="modalVisible=false;activeProf=false">✕</button></div>
          <div class="wiki-body" v-if="!activeProf">
            <p class="wiki-note" style="text-align:center;margin-bottom:16px">练气期可选择一门职业。神识越高，副职成功率越高。</p>
            <div class="prof-grid">
              <div v-for="p in profList" :key="p.key" class="prof-card" :class="{locked:player.realmId<2}" @click="openProfession(p)">
                <div class="pc-icon">{{ p.icon }}</div>
                <div class="pc-name">{{ p.name }}</div>
                <div class="pc-desc">{{ p.desc }}</div>
                <div class="pc-bonus">{{ p.bonus }}</div>
                <div class="pc-status">{{ player.realmId<2?'🔒 需练气期' : player.profession===p.key?('Lv.'+player.professionLevel) : '未选择' }}</div>
              </div>
            </div>
          </div>
          <div class="wiki-body" v-if="activeProf">
            <button class="pa-btn" @click="activeProf=false" style="margin-bottom:12px">← 返回职业列表</button>
            <div class="alchemy-hero">
              <div class="ah-furnace">🔥</div>
              <div class="ah-info">
                <div class="ah-title">{{ activeProf.icon }} {{ activeProf.name }} <span class="ah-lv">Lv.{{ player.professionLevel||1 }}</span></div>
                <div class="ah-exp-bar"><div class="ah-exp-fill" :style="{width:Math.min(100,((player.professionExp||0)%100))+'%'}" /></div>
                <div class="ah-exp-text">EXP {{ (player.professionExp||0)%100 }}/100</div>
                <div class="ah-stats">👁️ 神识 {{ player.spiritSense }} | 已炼 {{ pillStats.crafted }} | 失败 {{ pillStats.failed }}</div>
              </div>
            </div>
            <div class="pill-filter">
              <button v-for="c in pillCats" :key="c.key" class="pa-btn" :class="{active:pillCat===c.key}" @click="pillCat=c.key">{{ c.label }}</button>
            </div>
            <div class="pill-card-grid">
              <div v-for="r in filteredRecipes" :key="r.pill_key" class="pill-card" :class="{locked:player.realmId<r.req_realm}">
                <div class="pc-quality-strip" :style="{height:4+'px'}"></div>
                <div class="pc-top">
                  <span class="pc-icon-lg">{{ r.icon }}</span>
                  <div class="pc-badges">
                    <span class="pc-tier-badge">{{ '★'.repeat(r.tier) }}</span>
                    <span class="pc-cat-badge">{{ r.category }}</span>
                  </div>
                </div>
                <div class="pc-name-lg">{{ r.name }}</div>
                <div class="pc-desc-lg">{{ r.description }}</div>
                <div class="pc-stats-row">
                  <div class="pc-stat"><span class="pc-stat-label">成功率</span><span class="pc-stat-val green">{{ Math.min(95,r.base_success+Math.floor(player.spiritSense/50)+(player.profession=='dan'?player.professionLevel*2:0)) }}%</span></div>
                  <div class="pc-stat"><span class="pc-stat-label">神识消耗</span><span class="pc-stat-val">{{ r.ss_cost }}</span></div>
                </div>
                <div class="pc-quality-bar">
                  <div class="pc-qb-seg" v-for="(q,i) in [{c:'#888',v:Math.max(0,85-Math.floor(player.spiritSense/50)-(player.profession=='dan'?player.professionLevel*2:0))},{c:'#aaa',v:20},{c:'#6bcb77',v:Math.min(50,25+Math.floor(player.spiritSense/100))},{c:'#4d96ff',v:Math.min(30,15+Math.floor(player.spiritSense/150))},{c:'#ff6b9e',v:Math.min(25,10+Math.floor(player.spiritSense/100)+(player.profession=='dan'?player.professionLevel*2:0))}]" :key="i" :style="{flex:q.v,background:q.c}" :title="['劣质','普通','优良','精良','完美'][i]+' '+Math.round(q.v)+'%'" />
                </div>
                <div class="pc-actions-lg">
                  <button class="pc-btn-minus" @click="pillQtys[r.pill_key]=Math.max(1,(pillQtys[r.pill_key]||1)-1)">−</button>
                  <span class="pc-qty-display">{{ pillQtys[r.pill_key]||1 }}</span>
                  <button class="pc-btn-plus" @click="pillQtys[r.pill_key]=Math.min(99,(pillQtys[r.pill_key]||1)+1)">+</button>
                  <button class="pc-btn-craft" @click="craftPill(r,pillQtys[r.pill_key]||1)" :disabled="player.spiritSense<r.ss_cost*r.tier||player.realmId<r.req_realm">{{ player.realmId<r.req_realm?'🔒'+pillRecipes.find(x=>x.pill_key===r.pill_key)?.req_realm+'级解锁':'开炉炼制' }}</button>
                </div>
              </div>
            </div>
            <!-- 炼丹结果弹窗 -->
            <Teleport to="body">
              <div v-if="craftResult" class="modal-overlay" @click.self="craftResult=null">
                <div class="craft-modal-lg" :class="craftResult.success?'craft-ok':'craft-no'">
                  <div class="cml-top">{{ craftResult.success?'🔥 丹成！':'💥 炸炉！' }}</div>
                  <div v-if="craftResult.success" class="cml-pill-show">
                    <span class="cml-pill-icon-lg" :style="{animation:'em-bounce .3s ease-in-out infinite alternate'}">{{ filteredRecipes.find(r=>r.pill_key===craftResult.key)?.icon||'💊' }}</span>
                    <span class="cml-pill-name-lg" :style="{color:pillQualityColors[craftResult.quality]||'#ffd700',textShadow:'0 0 '+((craftResult.quality||0)*6+4)+'px '+pillQualityColors[craftResult.quality]||'#ffd700'}">{{ craftResult.name }} · {{ craftResult.quality_name }}</span>
                  </div>
                  <div class="cml-stats-grid">
                    <div class="cml-stat-box"><span class="cml-stat-v">{{ craftResult.effect||0 }}</span><span class="cml-stat-l">{{ craftResult.effect_type==='heal_hp'?'恢复HP%':'效果' }}</span></div>
                    <div class="cml-stat-box"><span class="cml-stat-v">+{{ craftResult.exp||0 }}</span><span class="cml-stat-l">经验</span></div>
                    <div class="cml-stat-box"><span class="cml-stat-v">−{{ craftResult.cost||0 }}</span><span class="cml-stat-l">神识</span></div>
                  </div>
                  <div v-if="!craftResult.success" class="cml-fail-reason">材料尽毁，神识消散...</div>
                  <div class="cml-btns">
                    <button class="cml-btn-retry" @click="craftResult=null;if(craftResult.success)craftAgain()">再炼一颗</button>
                    <button class="cml-btn-close" @click="craftResult=null">收下</button>
                  </div>
                </div>
              </div>
            </Teleport>
          </div>
        </div>
      </div>
      <div v-if="activeMenu?.key==='pigeon'" class="modal-overlay" @click.self="activeMenu=null">
        <div class="wiki-modal pigeon-modal">
          <div class="wiki-header"><h2>🕊️ 飞鸽传书</h2><button class="modal-close" @click="activeMenu=null">✕</button></div>
          <div class="pigeon-body">
            <div class="pig-left">
              <div class="pig-search-box"><input v-model="friendSearch" @keyup.enter="searchPlayers" placeholder="搜索道友..." class="pig-search-inp" /></div>
              <div class="pig-search-results" v-if="searchResults.length>0">
                <div v-for="r in searchResults" :key="r.id" class="pig-sr-row"><span class="pig-sr-name">{{ r.nickname }}</span><span class="pig-sr-tag">锻体{{ r.realm_stage }}期</span><button @click="addFriend(r.id)">添加</button></div>
              </div>
              <div class="pig-group"><div class="pig-group-head" @click="pigGroups.friends=!pigGroups.friends">👤 好友 <span class="pig-group-arrow" :class="{open:pigGroups.friends}">▾</span><span style="font-size:10px;color:rgba(255,255,255,.2);margin-left:auto">{{ friends.length }}/50</span></div>
                <div v-if="pigGroups.friends" class="pig-group-body">
                  <div v-for="f in friends" :key="f.id" class="pig-friend" :class="{active:activePeer===f.id}" @click="openChat(f)" @contextmenu.prevent="removeFriend(f.id)"><div class="pig-f-avatar">{{ f.nickname[0] }}</div><div class="pig-f-info"><div class="pig-f-name">{{ f.nickname }}<span class="pig-f-online" :class="{online:f.online}"></span></div><div class="pig-f-realm">锻体{{ f.realm_stage }}期</div></div></div>
                  <div v-if="friends.length===0" class="pig-empty-tip">暂无好友</div>
                </div></div>
              <div class="pig-group"><div class="pig-group-head" @click="pigGroups.requests=!pigGroups.requests">📋 好友申请 <span v-if="pendingRequests.length" class="pig-badge">{{ pendingRequests.length }}</span><span class="pig-group-arrow" :class="{open:pigGroups.requests}">▾</span></div>
                <div v-if="pigGroups.requests" class="pig-group-body">
                  <div v-for="r in pendingRequests" :key="r.id" class="pig-request-row"><span class="pig-r-name">{{ r.nickname }}</span><div class="pig-r-btns"><button @click="acceptFriend(r.id)">接受</button><button @click="removeFriend(r.id)" class="pig-r-reject">拒绝</button></div></div>
                  <div v-if="pendingRequests.length===0" class="pig-empty-tip">暂无申请</div>
                </div></div>
              <div class="pig-group"><div class="pig-group-head" @click="pigGroups.daolv=!pigGroups.daolv">💑 道侣 <span class="pig-group-arrow" :class="{open:pigGroups.daolv}">▾</span></div><div v-if="pigGroups.daolv" class="pig-group-body"><div class="pig-empty-tip">修仙路漫漫，寻一道侣</div></div></div>
              <div class="pig-group"><div class="pig-group-head" @click="pigGroups.master=!pigGroups.master">🎓 师徒 <span class="pig-group-arrow" :class="{open:pigGroups.master}">▾</span></div><div v-if="pigGroups.master" class="pig-group-body"><div class="pig-empty-tip">拜师或收徒，传承衣钵</div></div></div>
            </div>
            <div class="pig-right">
              <div v-if="activePeer" class="pig-chat">
                <div style="display:flex;align-items:center;gap:8px;padding:10px 14px;border-bottom:1px solid rgba(255,255,255,.06);background:rgba(0,0,0,.1)"><div style="width:34px;height:34px;border-radius:50%;background:linear-gradient(135deg,#4d96ff,#6bcb77);display:flex;align-items:center;justify-content:center;font-size:15px;font-weight:700;color:#fff">{{ activePeerName[0] }}</div><div style="font-size:14px;font-weight:700;color:#fff">{{ activePeerName }}</div></div>
                <div class="pig-chat-msgs"><div v-for="m in privateMessages" :key="m.id" class="pig-msg-bubble" :class="{self:m.from_id===playerId}"><div class="pig-msg-avatar">{{ m.from_id===playerId ? player.name[0] : activePeerName[0] }}</div><div class="pig-msg-content"><div class="pig-msg-bub-text">{{ m.text }}</div></div></div></div>
                <div class="pig-chat-send"><input v-model="privateInput" @keyup.enter="sendPrivate" placeholder="按回车发送" class="pig-send-inp" /><button @click="sendPrivate" class="pig-send-btn">发送</button></div>
              </div>
              <div v-else style="flex:1;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:12px;background:radial-gradient(ellipse at center,rgba(212,168,67,.03) 0%,transparent 70%)"><div style="font-size:80px;opacity:.15">🕊️</div><div style="font-size:22px;color:rgba(255,255,255,.3);font-weight:600;letter-spacing:4px">飞鸽传书</div><div style="font-size:13px;color:rgba(255,255,255,.15)">选择一个好友开始聊天</div></div>
            </div>
          </div>
        </div>
      </div>
      <div v-if="modalVisible&&activeMenu?.key!=='map'&&activeMenu?.key!=='profession'&&activeMenu?.key!=='pigeon'&&activeMenu?.key!=='backpack'" class="modal-overlay" @click.self="modalVisible=false">
        <div class="modal-card">
          <div class="modal-header"><h2>{{ activeMenu?.label }}</h2><button class="modal-close" @click="modalVisible=false">✕</button></div>
          <div v-if="activeMenu?.children" class="modal-tabs">
            <button v-for="sub in activeMenu.children" :key="sub.key" class="modal-tab" :class="{active:activeSub===sub.key}" @click="activeSub=sub.key;modalDesc=descs[sub.key]||''">{{ sub.label }}</button>
          </div>
          <div class="modal-body">
            <template v-if="activeMenu?.key==='backpack'">
              <div v-if="bpItems.length===0" style="text-align:center;padding:40px;color:#888">背包空空如也</div>
              <div v-for="i in bpItems" :key="i.id" style="display:flex;align-items:center;gap:8px;padding:6px 10px;border-bottom:1px solid rgba(255,255,255,.04);font-size:13px"><span style="font-size:22px;margin-right:8px">{{ i.icon }}</span><span style="color:#fff;flex:1">{{ i.name }}</span><span style="color:#888">×{{ i.quantity }}</span><button v-if="i.item_type==='pill'" style="padding:4px 14px;border:1px solid #d4a843;border-radius:14px;background:transparent;color:#d4a843;font-size:11px;cursor:pointer;font-family:inherit" @click="useBpPill(i)">使用</button></div>
            </template>
            <template v-else><p class="modal-placeholder">🏗️ {{ activeSubLabel }} 系统开发中...</p><p class="modal-desc">{{ modalDesc }}</p></template>
          </div>
        </div>
      </div>
    </Teleport>
    <!-- 地图弹窗 -->
    <Teleport to="body">
      <div v-if="activeMenu?.key==='map'" class="modal-overlay" @click.self="activeMenu=null">
        <div class="map-modal">
          <div class="wiki-header"><h2>🗺️ 修仙世界 · 地图</h2><button class="modal-close" @click="activeMenu=null">✕</button></div>
          <div class="map-body-wrap">
            <div class="map-legend">
              <span class="ml-item"><span class="ml-dot unlocked"></span> 可进入</span>
              <span class="ml-item"><span class="ml-dot locked"></span> 境界未达</span>
              <span class="ml-item">🧘 修炼 ⚔️ 战斗 🏚️ 副本 🌿 采集 🏪 交易 🔮 秘境</span>
            </div>
            <div v-for="region in mapRegions" :key="region.name" class="map-region" :class="{locked:player.realmId<region.minRealm}">
              <div class="mr-header">
                <span class="mr-icon">{{ region.icon }}</span>
                <span class="mr-name">{{ region.name }}</span>
                <span class="mr-realm">{{ realmNames[region.minRealm] }}·{{ realmNames[region.maxRealm] }}</span>
                <span v-if="player.realmId<region.minRealm" class="mr-lock">🔒 需{{ realmNames[region.minRealm] }}期</span>
              </div>
              <div class="mr-locations">
                <div v-for="loc in region.locations" :key="loc.key" class="mr-loc" :class="{locked:player.realmId<loc.minRealm,current:currentLoc===loc.key}" @click="activeLoc=activeLoc===loc.key?null:loc.key">
                  <span class="mrl-icon">{{ loc.icon }}</span>
                  <span class="mrl-name">{{ loc.name }}</span>
                  <span class="mrl-type">🐾 {{ loc.monsters }}</span>
                  <span v-if="player.realmId<loc.minRealm" class="mrl-lock">🔒</span>
                  <div v-if="activeLoc===loc.key" class="mrl-detail">
                    <p>{{ loc.desc }}</p>
                    <div class="mrl-info">
                      <span>最低境界：{{ realmNames[loc.minRealm] }}{{ loc.minStage }}期</span>
                      <span>怪物：{{ loc.monsters }}</span>
                      <span v-if="loc.monsters">怪物：{{ loc.monsters }}</span>
                      <span v-if="loc.boss">BOSS：{{ loc.boss }}</span>
                    </div>
                    <button v-if="player.realmId>=loc.minRealm&&currentLoc!==loc.key" class="map-enter-btn" @click.stop="enterLocation(loc)">进入</button>
                  <button v-if="player.realmId>=loc.minRealm&&player.hp>0" class="map-fight-btn" @click.stop="startPve(loc)">⚔️ 挑战</button>
                  <span v-else-if="currentLoc===loc.key" class="map-current-badge">📍 当前所在地</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
    <!-- 百科弹窗 -->
    <Teleport to="body">
      <div v-if="showWiki" class="modal-overlay" @click.self="showWiki=false">
        <div class="wiki-modal">
          <div class="wiki-header"><h2>📖 修仙百科</h2><button class="modal-close" @click="showWiki=false">✕</button></div>
          <div class="wiki-tabs">
            <button v-for="t in wikiTabs" :key="t.key" class="modal-tab" :class="{active:wikiTab===t.key}" @click="wikiTab=t.key">{{ t.label }}</button>
          </div>
          <div class="wiki-body">
            <!-- 境界体系 -->
            <div v-if="wikiTab==='realm'">
              <h3>境界体系</h3>
              <table class="wiki-table"><thead><tr><th>境界</th><th>系数</th><th>基础率</th><th>下品×5</th><th>中品×15</th><th>上品×40</th><th>极品×100</th><th>攻击</th><th>防御</th><th>生命</th><th>灵力</th><th>速度</th><th>暴击%</th><th>暴伤%</th><th>闪避%</th><th>回蓝%</th><th>寿元</th></tr></thead><tbody>
                <tr v-for="(r,i) in wikiRealms" :key="i" :class="{highlight:i+1===player.realmId}">
                  <td class="tc"><b>{{ r.name }}</b></td>
                  <td>{{ r.coef }}</td><td>{{ (r.brk*100).toFixed(r.brk>=1?0:r.brk>=0.01?1:r.brk>=0.001?2:4) }}%</td>
                  <td>{{ r.brk>=0.1 ? Math.min(95,Math.floor(r.brk*5+15))+'%' : r.brk>0 ? Math.floor(r.brk*5+15)+'%' : '≈0%' }}</td>
                  <td>{{ r.brk>=0.01 ? Math.min(95,Math.floor(r.brk*15+15))+'%' : r.brk>0 ? (r.brk*15+15).toFixed(r.brk*15>=1?0:2)+'%' : '≈0%' }}</td>
                  <td>{{ r.brk>=0.005 ? Math.min(95,Math.floor(r.brk*40+15))+'%' : r.brk>0 ? (r.brk*40+15).toFixed(r.brk*40>=1?0:2)+'%' : '≈0%' }}</td>
                  <td>{{ r.brk>=0.002 ? Math.min(95,Math.floor(r.brk*100+15))+'%' : r.brk>0 ? (r.brk*100+15).toFixed(r.brk*100>=1?0:3)+'%' : (r.brk*100+15).toFixed(4)+'%' }}</td>
                  <td>{{ r.atk }}</td><td>{{ r.def }}</td><td>{{ r.hp }}</td><td>{{ r.mp }}</td><td>{{ r.spd }}</td>
                  <td>{{ (r.cr/100).toFixed(0) }}%</td><td>{{ (r.cd/100).toFixed(0) }}%</td>
                  <td>{{ (r.dg/100).toFixed(0) }}%</td><td>{{ (r.mr/100).toFixed(0) }}%</td><td>{{ r.life }}年</td>
                </tr></tbody></table>
              <p class="wiki-note">※ 小期每级 +8% 固定属性 / +5% 百分比属性。速度只随境界变化。</p>
              <h4>修炼速度公式</h4>
              <p class="wiki-formula">每秒修为 = (10 + 境界系数 × 小期) × 灵根修炼倍率</p>
              <h4>修为需求表</h4>
              <table class="wiki-table"><thead><tr><th>境界</th><th v-for="s in 9">{{ s }}→{{ s+1 }}期</th></tr></thead><tbody>
                <tr v-for="(r,i) in wikiSpiritReqs" :key="i"><td class="tc"><b>{{ wikiRealms[i]?.name }}</b></td><td v-for="v in r">{{ fmt(v) }}</td></tr>
              </tbody></table>
              <p class="wiki-note">※ 突破率表中为少年期估算值。公式: 基础率 × 品质倍率 × 年龄倍率 + 剩余寿元 + 气运×0.08 + 悟性×0.03（上限95%下限2%）</p>
              <p class="wiki-note">年龄倍率：少年×1.5 / 青年×1.2 / 中年×0.8 / 老年×0.3。1天真=1寿元年。</p>
            </div>
            <!-- 灵根体系 -->
            <div v-if="wikiTab==='root'">
              <h3>灵根类型与属性加成</h3>
              <table class="wiki-table"><thead><tr><th>灵根</th><th>攻击</th><th>防御</th><th>生命</th><th>灵力</th><th>暴击</th><th>暴伤</th><th>闪避</th><th>回蓝</th></tr></thead><tbody>
                <tr v-for="(r,i) in wikiRootBonuses" :key="i" :class="{highlight:rootNames[i]===player.spiritName}">
                  <td class="tc"><b>{{ r.name }}</b></td>
                  <td>{{ r.atk>0?'+'+r.atk+'%':'—' }}</td><td>{{ r.def>0?'+'+r.def+'%':'—' }}</td>
                  <td>{{ r.hp>0?'+'+r.hp+'%':'—' }}</td><td>{{ r.mp>0?'+'+r.mp+'%':'—' }}</td>
                  <td>{{ r.cr>0?'+'+r.cr+'%':'—' }}</td><td>{{ r.cd>0?'+'+r.cd+'%':'—' }}</td>
                  <td>{{ r.dg>0?'+'+r.dg+'%':'—' }}</td><td>{{ r.mr>0?'+'+r.mr+'%':'—' }}</td>
                </tr></tbody></table>
              <h4>品质倍率</h4>
              <table class="wiki-table"><thead><tr><th>品质</th><th>修炼倍率</th><th>突破加成</th><th>属性加成倍率</th><th>概率</th></tr></thead><tbody>
                <tr v-for="q in wikiQuality" :key="q.name" :class="{highlight:qualityNames[player.rootQuality]===q.name}">
                  <td class="tc"><b>{{ q.name }}</b></td><td>×{{ q.speed }}</td><td>×{{ q.brk }}</td><td>×{{ q.attr }}</td><td>{{ q.chance }}%</td>
                </tr></tbody></table>
              <h4>灵根属性加成公式</h4>
              <p class="wiki-formula">最终属性 = 境界基础属性 × 灵根类型加成% × 品质倍率</p>
              <p class="wiki-note">※ 锻体期无灵根。练气期自动随机分配灵根类型+品质。灵根品质不可通过突破改变，仅极小概率奇遇可洗炼提升。</p>
            </div>
            <!-- 属性说明 -->
            <div v-if="wikiTab==='attrs'">
              <h3>战斗属性</h3>
              <table class="wiki-table"><thead><tr><th>属性</th><th>计算方式</th><th>作用</th></tr></thead><tbody>
                <tr><td>⚔️ 攻击</td><td>境界基础 × (1 + 灵根加成 × 品质倍率) × (1 + 0.08 × 小期)</td><td>决定物理伤害</td></tr>
                <tr><td>🛡️ 防御</td><td>同上（境界基础 + 灵根加成）</td><td>减免受到的伤害</td></tr>
                <tr><td>❤️ 生命</td><td>同上</td><td>归零则死亡/战斗失败</td></tr>
                <tr><td>💙 灵力</td><td>同上</td><td>释放技能消耗</td></tr>
                <tr><td>💨 速度</td><td>境界基础（仅随大境界变化）</td><td>决定战斗先手</td></tr>
                <tr><td>💥 暴击</td><td>境界基础%(×100) × (1 + 灵根加成%) × (1 + 0.05 × 小期) ÷ 100</td><td>概率造成暴伤</td></tr>
                <tr><td>💢 暴伤</td><td>同上公式</td><td>暴击时伤害倍率（基础150%）</td></tr>
                <tr><td>🎯 命中</td><td>同上</td><td>对抗闪避</td></tr>
                <tr><td>💨 闪避</td><td>同上</td><td>概率完全回避攻击</td></tr>
                <tr><td>🌀 修炼</td><td>基础0，装备/功法提供</td><td>直接加到 cultRate</td></tr>
                <tr><td>⚡ 突破加成</td><td>基础0，丹药提供</td><td>直接加到 breakRate</td></tr>
                <tr><td>💧 回蓝</td><td>境界基础% × (1 + 灵根加成%) × (1 + 0.05 × 小期) ÷ 100</td><td>打坐时每秒回蓝%</td></tr>
                <tr><td>⏳ 寿元</td><td>境界基础 × (1 + 0.08 × 小期)</td><td>角色寿命上限</td></tr>
              </tbody></table>
              <h4>战斗力公式</h4>
              <p class="wiki-formula">战力 = 攻击×1.5 + 防御×1.2 + 生命×0.15 + 灵力×0.1 + 速度×0.5 + 暴击%×3 + 暴伤%×0.15 + 闪避%×3 + 命中%×0.5 + 回蓝%×2</p>
              <h4>战斗公式（规划）</h4>
              <p class="wiki-note">伤害 = 攻击 × (1 - 防御/(防御+100)) × 随机(0.9~1.1)。暴击时 × 暴伤%。命中 vs 闪避判定。先手由速度决定。</p>
            </div>
            <!-- 先天属性 -->
            <div v-if="wikiTab==='innate'">
              <h3>📖 悟性</h3>
              <p class="wiki-formula">悟性 = 灵根品质基础(随机) + 大境界加成</p>
              <table class="wiki-table"><thead><tr><th>灵根品质</th><th>基础范围</th></tr></thead><tbody>
                <tr><td>无品</td><td>5 ~ 15</td></tr><tr><td>下品</td><td>15 ~ 35</td></tr><tr><td>中品</td><td>35 ~ 55</td></tr><tr><td>上品</td><td>55 ~ 75</td></tr><tr><td>极品</td><td>75 ~ 100</td></tr>
              </tbody></table>
              <p class="wiki-note">大境界加成：练气+5 → 筑基+10 → 金丹+20 → 元婴+35 → 化神+55 → 炼虚+80 → 合体+110 → 大乘+150 → 渡劫+200。功法数量额外+1/本。</p>
              <p class="wiki-note">作用：功法参悟速度。悟性越高，学习功法越快。</p>
              <h3>🍀 气运</h3>
              <p class="wiki-formula">每日气运 = 随机(0 ~ 品质上限) + 灵根类型固定加成</p>
              <table class="wiki-table"><thead><tr><th>品质</th><th>随机范围</th></tr></thead><tbody>
                <tr><td>无品</td><td>0 ~ 30</td></tr><tr><td>下品</td><td>0 ~ 50</td></tr><tr><td>中品</td><td>0 ~ 70</td></tr><tr><td>上品</td><td>0 ~ 85</td></tr><tr><td>极品</td><td>0 ~ 100</td></tr>
              </tbody></table>
              <p class="wiki-note">灵根类型加成：天+10, 地+5, 木+3, 土+3, 水+2。上限100。</p>
              <p class="wiki-note">作用：掉落率 = 基础 × (1 + 气运/200)。奇遇概率加成 = 气运/100。</p>
              <h3>👁️ 神识</h3>
              <p class="wiki-formula">神识 = 境界基础 × 灵根品质倍率</p>
              <table class="wiki-table"><thead><tr><th>境界</th><th v-for="r in wikiRealms.slice(0,5)">{{ r.name }}</th></tr></thead><tbody>
                <tr><td>基础值</td><td v-for="r in wikiRealms.slice(0,5)">{{ r.ss }}</td></tr>
                <tr><td>× 无品(0.7)</td><td v-for="r in wikiRealms.slice(0,5)">{{ Math.floor(r.ss*0.7) }}</td></tr>
                <tr><td>× 中品(1.3)</td><td v-for="r in wikiRealms.slice(0,5)">{{ Math.floor(r.ss*1.3) }}</td></tr>
                <tr><td>× 极品(2.0)</td><td v-for="r in wikiRealms.slice(0,5)">{{ Math.floor(r.ss*2.0) }}</td></tr>
              </tbody></table>
              <p class="wiki-note">品质倍率：无品×0.7, 下品×1.0, 中品×1.3, 上品×1.6, 极品×2.0</p>
              <p class="wiki-note">作用：①秘境发现概率 ②副职(炼丹/符箓/炼器)成功率=神识/50% ③优良品质概率=神识/200</p>
            </div>
            <!-- 装备体系 -->
            <div v-if="wikiTab==='equip'">
              <h3>装备体系</h3>
              <p class="wiki-note">10个装备部位×10个境界等级×5级品质。品质倍率：劣质×0.6 普通×1.0 优良×1.4 精良×1.8 完美×2.5</p>
              <table class="wiki-table"><thead><tr><th>部位</th><th>主属性</th><th>锻体</th><th>练气</th><th>筑基</th><th>金丹</th><th>元婴</th><th>化神</th><th>炼虚</th><th>合体</th><th>大乘</th><th>渡劫</th></tr></thead><tbody>
                <tr><td class="tc">🗡️ 武器</td><td>攻击·暴击</td><td>铁剑 15攻</td><td>灵剑 30攻</td><td>法宝剑 60攻</td><td>金丹剑 120攻</td><td>元婴剑 240攻</td><td>化神剑 480攻</td><td>炼虚剑 960攻</td><td>合体剑 1800攻</td><td>大乘剑 3500攻</td><td>渡劫剑 7000攻</td></tr>
                <tr><td class="tc">👘 法袍</td><td>防御·生命</td><td>布袍 10防30血</td><td>灵袍 20防60血</td><td>金丹袍 40防120血</td><td>元婴袍 150防500血</td><td>化神袍 300防1000血</td><td>炼虚袍 600防2000血</td><td>合体袍 1200防4000血</td><td>大乘袍 2500防8000血</td><td>渡劫袍 5000防16000血</td></tr>
                <tr><td class="tc">👑 发冠</td><td>灵力·回蓝</td><td>布冠 20灵</td><td>灵冠 40灵</td><td>金丹冠 80灵</td><td>元婴冠 150灵</td><td>化神冠 300灵</td><td>炼虚冠 600灵</td><td>合体冠 1200灵</td><td>大乘冠 2400灵</td><td>渡劫冠 4500灵</td></tr>
                <tr><td class="tc">🎗️ 腰带</td><td>生命</td><td>布带 40血</td><td>灵带 80血</td><td>金丹带 150血</td><td>元婴带 300血</td><td>化神带 600血</td><td>炼虚带 1200血</td><td>合体带 2400血</td><td>大乘带 4800血</td><td>渡劫带 9000血</td></tr>
                <tr><td class="tc">🛡️ 护腕</td><td>生命·暴击</td><td>铁腕 10血3暴</td><td>灵腕 20血6暴</td><td>金丹腕 40血10暴</td><td>元婴腕 80血15暴</td><td>化神腕 150血22暴</td><td>炼虚腕 300血30暴</td><td>合体腕 600血38暴</td><td>大乘腕 1200血48暴</td><td>渡劫腕 2500血58暴</td></tr>
                <tr><td class="tc">👢 云靴</td><td>速度·闪避</td><td>布靴 8速</td><td>灵靴 12速</td><td>金丹靴 16速</td><td>元婴靴 20速</td><td>化神靴 25速</td><td>炼虚靴 30速</td><td>合体靴 35速</td><td>大乘靴 40速</td><td>渡劫靴 48速</td></tr>
              </tbody></table>
              <p class="wiki-note">※ 表中为普通品质(×1.0)数值。完美品质=×2.5。共8个部位80件装备。</p>
<tr><td class="tc">📿 项链</td><td>生命·回蓝</td><td>石链 20血3回</td><td>灵链 40血5回</td><td>金丹链 80血7回</td><td>元婴链 150血9回</td><td>化神链 300血12回</td><td>炼虚链 600血15回</td><td>合体链 1200血18回</td><td>大乘链 2400血22回</td><td>渡劫链 4500血26回</td></tr>
                <tr><td class="tc">💍 戒指</td><td>攻击·暴伤</td><td>铁环 5攻5暴伤</td><td>灵环 10攻10暴伤</td><td>金丹环 20攻15暴伤</td><td>元婴环 40攻20暴伤</td><td>化神环 80攻28暴伤</td><td>炼虚环 160攻35暴伤</td><td>合体环 320攻45暴伤</td><td>大乘环 600攻55暴伤</td><td>渡劫环 1200攻65暴伤</td></tr>
              <h4>品质概率</h4>
              <table class="wiki-table"><thead><tr><th>品质</th><th>倍率</th><th>概率</th></tr></thead><tbody>
                <tr><td>劣质</td><td>×0.6</td><td>35%</td></tr>
                <tr><td>普通</td><td>×1.0</td><td>30%</td></tr>
                <tr><td>优良</td><td>×1.4</td><td>20%</td></tr>
                <tr><td>精良</td><td>×1.8</td><td>10%</td></tr>
                <tr><td>完美</td><td>×2.5</td><td>5%</td></tr>
              </tbody></table>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
        <!-- PVE 弹窗 -->
    <Teleport to="body">
      <div v-if="pveReport" class="modal-overlay" @click.self="pveReport=null;pveRounds=[]">
        <div class="death-modal" :style="{borderColor:pveReport.won?'#6bcb77':'#e53935'}">
          <div class="dm-header" :style="{color:pveReport.won?'#6bcb77':'#e53935'}">{{ pveReport.won?'⚔️ 胜利！':'💀 战败' }}</div>
          <div v-if="pveReport.monster" style="font-size:16px;color:#ffd700;margin:8px 0">👹 {{ pveReport.monster }}期怪物</div>
          <div v-if="pveReport.won" style="font-size:14px;color:#6bcb77;margin-bottom:8px">🎁 +{{ pveReport.cult }}修为 +{{ pveReport.gold }}灵石</div>
          <div class="dm-battle" style="max-height:200px;text-align:left">
            <div v-for="r in pveRounds" :key="r.round" class="cr-round"><span>第{{ r.round }}回合:</span><span class="cr-dmg">造成 {{ r.player_dmg }} 伤害</span><span v-if="r.desc" class="cr-special">{{ r.desc }}</span><span class="cr-dmg">受到 {{ r.monster_dmg }} 伤害</span><span class="cr-hp">❤️{{ r.player_hp }} | 👹{{ r.monster_hp>0?r.monster_hp:'击败' }}</span></div>
          </div>
          <button class="map-enter-btn" style="margin-top:12px;font-size:14px;padding:8px 24px" @click="pveReport=null;pveRounds=[]">关闭</button>
        </div>
      </div>
    </Teleport>
    <!-- 奇遇弹窗 -->
    <Teleport to="body">
      <div v-if="encounterResult" class="modal-overlay">
        <div class="encounter-modal">
          <div class="em-icon">{{ encounterResult.icon }}</div>
          <div class="em-title">{{ encounterResult.title }}</div>
          <div class="em-desc">历练中偶然发现一处隐秘洞府，获得了洗炼灵根的机会！</div>
          <div class="em-change"><span class="em-old">{{ encounterResult.old }}</span> → <span class="em-new">{{ encounterResult.new }}</span></div>
          <div class="em-note">属性已自动更新，修炼速度和战斗属性全面提升</div>
          <button class="map-enter-btn" style="margin-top:12px;font-size:14px;padding:8px 24px" @click="encounterResult=null;encounterType=''">收下</button>
        </div>
      </div>
    </Teleport>
    <!-- 死亡弹窗 -->
    <Teleport to="body">
      <div v-if="isDead" class="modal-overlay">
        <div class="death-modal">
          <div class="dm-header">{{ player.gender==='female'?'💀 香消玉殒':'💀 道心破碎' }}</div>
          <div class="dm-timer">复活倒计时: <b>{{ reviveCountdown }}</b> 秒</div>
          <div class="dm-info">复活后恢复 50% HP · 50% MP · 神识+10</div>
          <div class="dm-status">
            <span>❤️ {{ player.maxHp>0?Math.round(player.hp/player.maxHp*100):0 }}%</span>
            <span>💙 {{ player.maxMp>0?Math.round(player.mp/player.maxMp*100):0 }}%</span>
            <span>👁️ {{ player.spiritSense }}</span>
          </div>
          <div v-if="deathLog" class="dm-battle">
            <div class="dm-battle-title">⚔️ 最后一战</div>
            <div v-for="r in deathLog.rounds" :key="r.round" class="cr-round">
              <span>第{{ r.round }}回合:</span>
              <span class="cr-dmg">造成 {{ r.player_dmg }} 伤害</span>
              <span v-if="r.desc" class="cr-special">{{ r.desc }}</span>
              <span class="cr-dmg">受到 {{ r.monster_dmg }} 伤害</span>
              <span class="cr-hp">❤️{{ r.player_hp }} | 👹{{ r.monster_hp>0?r.monster_hp:'击败' }}</span>
            </div>
          </div>
          <p class="dm-note">死亡期间无法进行任何操作，但可以发送聊天消息</p>
        </div>
      </div>
    </Teleport>
    <Teleport to="body">
      <Transition name="tooltip">
        <div v-if="showCultTooltip" class="cult-tooltip" :style="tooltipStyle">
          <div class="ct-title">⚡ 修炼速度计算</div>
          <div class="ct-rows">
            <div class="ct-row"><span>基础速度</span><span>10</span></div>
            <div class="ct-row"><span>+ 境界系数 × 小期</span><span>{{ realmCoef }} × {{ player.realmStage }} = {{ realmCoef * player.realmStage }}</span></div>
            <div class="ct-row ct-sub"><span>境界基础</span><span>{{ cultBaseVal }}</span></div>
            <div v-if="hasRoot" class="ct-row ct-bonus"><span>{{ player.spiritName }} · {{ player.qualityName }}</span><span>×{{ rootMult.toFixed(1) }}</span></div>
            <div v-else class="ct-row ct-none"><span>灵根未觉醒</span><span>无加成</span></div>
            <div class="ct-divider"></div>
            <div class="ct-row ct-total"><span>最终速度</span><span>{{ player.cultRate.toFixed(1) }}/秒</span></div>
          </div>
          <div class="ct-formula">公式 = (10 + 境界系数 × 小期) × 灵根倍率</div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
const isDark=ref(true)
function toggleTheme(){isDark.value=!isDark.value;localStorage.setItem('theme-mode',isDark.value?'dark':'light');document.documentElement.className=isDark.value?'':'light-mode'}
onMounted(()=>{const m=localStorage.getItem('theme-mode');if(m)isDark.value=m==='dark';document.documentElement.className=isDark.value?'':'light-mode'})

interface MenuItem {key:string;label:string;children?:{key:string;label:string}[]}
const menus:MenuItem[]=[
  {key:'sect',label:'宗门',children:[{key:'my-sect',label:'我的宗门'},{key:'sect-list',label:'宗门列表'},{key:'sect-war',label:'宗门大战'}]},
  {key:'map',label:'地图',children:[{key:'world-map',label:'世界地图'},{key:'instance',label:'副本入口'},{key:'boss',label:'世界BOSS'}]},
  {key:'dongfu',label:'洞府',children:[{key:'my-dongfu',label:'我的洞府'},{key:'meditate',label:'闭关修炼'},{key:'alchemy',label:'炼丹房'}]},
  {key:'market',label:'坊市',children:[{key:'buy',label:'购买物品'},{key:'sell',label:'寄售物品'},{key:'my-shop',label:'我的摊位'}]},
  {key:'auction',label:'拍卖行',children:[{key:'auction-list',label:'竞拍中'},{key:'my-auction',label:'我的拍卖'},{key:'auction-record',label:'成交记录'}]},
  {key:'cultivation',label:'修炼',children:[{key:'meditate',label:'打坐修炼'},{key:'breakthrough',label:'突破境界'},{key:'tribulation',label:'渡劫'},{key:'technique',label:'功法'}]},
  {key:'profession',label:'职业',children:[{key:'dan',label:'丹师'},{key:'qi',label:'炼器师'},{key:'fu',label:'符师'},{key:'zhen',label:'阵法师'},{key:'zhi',label:'灵植师'}]},
  {key:'backpack',label:'背包',children:[{key:'items',label:'物品'},{key:'equipment',label:'装备'},{key:'pills',label:'丹药'}]},
  {key:'pet',label:'灵兽',children:[{key:'my-pets',label:'我的灵兽'},{key:'pet-hatch',label:'灵兽孵化'},{key:'pet-release',label:'放生'}]},
  {key:'artifact',label:'法宝',children:[{key:'my-artifact',label:'本命法宝'},{key:'artifact-enhance',label:'法宝淬炼'}]},
  {key:'rank',label:'排行',children:[{key:'power-rank',label:'战力榜'},{key:'realm-rank',label:'境界榜'},{key:'wealth-rank',label:'财富榜'}]},
  {key:'chat',label:'传音',children:[{key:'world-chat',label:'世界频道'},{key:'sect-chat',label:'宗门频道'},{key:'private-chat',label:'私聊'}]},
  {key:'pigeon',label:'飞鸽',children:[]},
  {key:'settings',label:'设置',children:[{key:'account',label:'账号设置'},{key:'display',label:'显示设置'},{key:'about',label:'关于游戏'}]},
]
const activeMenu=ref<MenuItem|null>(null),modalVisible=ref(false),activeSub=ref(''),modalDesc=ref('')
const activeSubLabel=computed(()=>activeMenu.value?.children?.find(s=>s.key===activeSub.value)?.label||'')
function openMenu(m:MenuItem){activeMenu.value=m;activeSub.value=m.children?.[0]?.key||m.key;modalDesc.value=descs[activeSub.value]||'';if(!['map','profession','pigeon','backpack'].includes(m.key))modalVisible.value=true}

const descs:Record<string,string>={'my-sect':'创建或加入宗门','world-map':'探索广阔修仙世界','my-dongfu':'洞府是修士的修炼根基','buy':'浏览坊市商品','auction-list':'参与竞拍稀有物品','meditate':'打坐修炼积累灵力','breakthrough':'灵力充盈即可突破','items':'查看背包物品','my-pets':'灵兽是修士的伙伴','my-artifact':'本命法宝与修士心血相连','power-rank':'全服战力排行榜','world-chat':'世界频道全服可见','account':'修改账号信息','friend-list':'添加道友为好友','master':'拜师学艺或收徒传道','daolv':'寻找道侣双修','tribulation':'突破需渡天劫','technique':'学习功法提升效率','equipment':'穿戴装备提升属性','pills':'使用丹药恢复','pet-hatch':'孵化灵兽蛋','pet-release':'放生灵兽','artifact-enhance':'淬炼法宝提升品阶','realm-rank':'全服境界排行榜','wealth-rank':'全服财富排行榜','sect-chat':'宗门内部频道','private-chat':'与道友私聊','display':'调整主题偏好','about':'修仙世界v1.0'}

const player=reactive({name:'修仙者',gender:'male',realmName:'锻体',realmId:1,realmStage:1,spiritName:'无灵根',qualityName:'无品',rootQuality:0,level:1,power:0,hp:100,maxHp:100,mp:50,maxMp:50,attack:10,defense:5,speed:100,critRate:3,critDmg:150,dodge:2,hit:95,cultBonus:0,breakBonus:0,mpRegen:3,lifespan:100,comprehension:10,luck:10,spiritSense:100,spirit:0,maxSpirit:100,gold:0,jade:0,isMeditating:false,cultRate:11,breakRate:90})
const realmNames:Record<number,string>={1:'锻体',2:'练气',3:'筑基',4:'金丹',5:'元婴',6:'化神',7:'炼虚',8:'合体',9:'大乘',10:'渡劫'}
// 境界系数（与后端 model.RealmCoefficient 一致）
const realmCoefs:Record<number,number>={1:1,2:2,3:3,4:5,5:8,6:12,7:18,8:25,9:35,10:50}
// 灵根修炼倍率（与后端 model.RootSpeedMultiplier 一致）
const rootMults:Record<number,number>={0:0.7,1:1.0,2:1.3,3:1.6,4:2.0}
const qualityNames:Record<number,string>={0:'无品',1:'下品',2:'中品',3:'上品',4:'极品'}
const qualityColors:Record<number,string>={0:'#888',1:'#aaa',2:'#6bcb77',3:'#4d96ff',4:'#ff6b6b'}
const pillQualityColors:Record<number,string>={0:'#888',1:'#aaa',2:'#6bcb77',3:'#4d96ff',4:'#ff6b9e'}
const showPillPanel=ref(false),showPillCraft=ref(false)
const myPills=ref<any[]>([]),pillRecipes=ref<any[]>([])
const pillCount=computed(()=>myPills.value.reduce((s:number,p:any)=>s+p.quantity,0))
const pillCat=ref('all'),pillQtys=reactive<Record<string,number>>({})
const craftResult=ref<any>(null),pillStats=reactive({crafted:0,failed:0})
const pillCats=[{key:'all',label:'全部'},{key:'恢复',label:'💚恢复'},{key:'修炼',label:'🧘修炼'},{key:'战斗',label:'⚔️战斗'},{key:'防护',label:'🛡️防护'},{key:'气运',label:'🍀气运'},{key:'特殊',label:'🎲特殊'}]
const filteredRecipes=computed(()=>pillCat.value==='all'?pillRecipes.value:pillRecipes.value.filter((r:any)=>r.category===pillCat.value))
async function loadPills(){const pid=getPID();if(!pid)return;try{const r=await fetch('/api/v1/player/'+pid+'/pills',{headers:{Authorization:'Bearer '+getToken()}});if(r.status===401){const nt=await refreshToken();if(nt){const r2=await fetch('/api/v1/player/'+pid+'/pills',{headers:{Authorization:'Bearer '+nt}});myPills.value=(await r2.json()).data||[]}}else{const d=await r.json();myPills.value=d.data||[]}}catch{}}
async function loadRecipes(){try{const r=await fetch('/api/v1/pills/recipes',{headers:{Authorization:'Bearer '+getToken()}});if(r.status===401){const nt=await refreshToken();if(nt){const r2=await fetch('/api/v1/pills/recipes',{headers:{Authorization:'Bearer '+nt}});pillRecipes.value=(await r2.json()).data||[]}}else{const d=await r.json();pillRecipes.value=d.data||[]}}catch{}}
async function craftPill(recipe:any,qty:number=1){
  const pid=getPID();if(!pid)return
  let successCount=0,lastResult=null
  for(let i=0;i<Math.min(qty,20);i++){
    const d=await apiPost('/api/v1/player/'+pid+'/pills/craft',{pill_key:recipe.pill_key})
    if(d&&d.data){
      if(d.data.success){successCount++;addLog('item','🧪 '+d.data.name+'·'+d.data.quality_name);pillStats.crafted++}
      else{pillStats.failed++;addLog('item','炼制失败')}
      lastResult={...d.data,key:recipe.pill_key,success:d.data.success,cost:recipe.ss_cost,exp:recipe.tier*10}
    }
  }
  if(lastResult)craftResult.value=lastResult
  loadPills()
}
function craftAgain(){const r=filteredRecipes.value.find((r:any)=>r.pill_key===craftResult.value?.key);if(r)craftPill(r,pillQtys[r.pill_key]||1)}
async function usePill(pill:any){const pid=getPID();if(!pid)return;const d=await apiPost('/api/v1/player/'+pid+'/pills/use',{pill_id:pill.id});if(d&&d.data.success){addLog('item','💊 使用 '+pill.name+'·'+pill.quality_name);loadPills();loadPlayer()}}
const rootNames:Record<number,string>={0:'无灵根',1:'金灵根',2:'木灵根',3:'水灵根',4:'火灵根',5:'土灵根',6:'地灵根',7:'天灵根'}
const yyWrapRef=ref<HTMLElement|null>(null)
// 百科
// 地图
const activeLoc=ref('')
interface MapLoc{key:string;icon:string;name:string;desc:string;minRealm:number;minStage:number;monsters:string}
interface MapRegion{name:string;icon:string;minRealm:number;maxRealm:number;locations:MapLoc[]}
const mapRegions:MapRegion[]=[
{name:'新手村',icon:'🌱',minRealm:1,maxRealm:2,locations:[
{key:'qys',icon:'⛰️',name:'青云山',desc:'灵气温和的修炼入门之地，常有散修往来。',minRealm:1,minStage:1,monsters:'野兔·山鸡·石魔'},
{key:'czl',icon:'🎋',name:'翠竹林',desc:'竹林幽静，竹叶青蛇出没，适合初期历练。',minRealm:1,minStage:3,monsters:'竹叶青·石魔·竹妖'},
{key:'ysm',icon:'🌲',name:'妖兽森林',desc:'密林深处妖兽横行，危机四伏。',minRealm:1,minStage:5,monsters:'妖狼·树妖·熊精'},
{key:'dsy',icon:'🏔️',name:'断魂崖',desc:'悬崖峭壁间厉鬼哭嚎，锻体圆满方可一试。',minRealm:1,minStage:8,monsters:'厉鬼·石像鬼·风妖'},
{key:'hfz',icon:'🏚️',name:'黑风寨',desc:'山贼盘踞之地，寨主实力不容小觑。',minRealm:2,minStage:1,monsters:'山贼·山贼头目·寨主'},
{key:'lqd',icon:'💧',name:'灵泉洞',desc:'洞中灵泉涌动，水属性妖兽聚集。',minRealm:2,minStage:3,monsters:'水怪·灵蛇·冰蛙'},
{key:'wszz',icon:'🌫️',name:'迷雾沼泽',desc:'终年浓雾不散，毒虫猛兽潜伏其中。',minRealm:2,minStage:6,monsters:'毒蛙·沼泽巨鳄·雾妖'},
{key:'byl',icon:'🌙',name:'半月岭',desc:'月圆之夜妖兽暴动，练气圆满方可挑战。',minRealm:2,minStage:8,monsters:'月狼·暗影豹·妖将'},
]},
{name:'中原大地',icon:'🌄',minRealm:3,maxRealm:4,locations:[
{key:'lxf',icon:'🌅',name:'落霞峰',desc:'中原修炼圣地，灵气充沛，妖兽温和。',minRealm:3,minStage:1,monsters:'金翅雕·灵狐·云鹤'},
{key:'lsg',icon:'🪨',name:'乱石岗',desc:'巨石嶙峋，土属性妖兽出没。',minRealm:3,minStage:3,monsters:'石巨人·土龙·岩蛇'},
{key:'ymd',icon:'🕳️',name:'幽冥洞',desc:'通往地底的幽深洞穴，阴气弥漫。',minRealm:3,minStage:5,monsters:'骷髅将军·幽魂·僵尸王'},
{key:'xshd',icon:'🩸',name:'血色荒地',desc:'上古战场遗址，煞气冲天。',minRealm:3,minStage:7,monsters:'血魔·食人花·怨灵'},
{key:'wjs',icon:'⚔️',name:'万剑山',desc:'无数断剑插于山体，剑气纵横。',minRealm:4,minStage:1,monsters:'剑灵·剑阵守卫·剑魂'},
{key:'cyg',icon:'🔥',name:'赤焰谷',desc:'地火喷涌，火属性妖兽的天堂。',minRealm:4,minStage:3,monsters:'火元素·炎魔·熔岩兽'},
{key:'lmfx',icon:'⚡',name:'雷鸣废墟',desc:'古战场遗迹，雷暴不止。',minRealm:4,minStage:6,monsters:'雷兽·雷灵·风暴之眼'},
{key:'lmzd',icon:'🐉',name:'龙眠之地',desc:'传说曾有真龙沉睡于此，威压犹存。',minRealm:4,minStage:8,monsters:'幼龙·龙守卫·半龙人'},
]},
{name:'东海',icon:'🌊',minRealm:5,maxRealm:6,locations:[
{key:'shj',icon:'🪸',name:'珊瑚礁',desc:'浅海珊瑚丛生，海妖出没。',minRealm:5,minStage:1,monsters:'海妖·鲨鱼精·水母精'},
{key:'xwhy',icon:'🌀',name:'漩涡海域',desc:'海流湍急，巨型海兽潜伏。',minRealm:5,minStage:3,monsters:'巨章鱼·海蛇·独角鲸'},
{key:'lgwq',icon:'🏯',name:'龙宫外围',desc:'东海龙宫外围防线，虾兵蟹将巡逻。',minRealm:5,minStage:5,monsters:'虾兵·蟹将·巡海夜叉'},
{key:'shlg',icon:'🌊',name:'深海裂谷',desc:'万丈海沟深处，远古海怪栖息。',minRealm:5,minStage:7,monsters:'深海巨兽·雷鳗·巨齿鲨'},
{key:'pljh',icon:'🏝️',name:'蓬莱近海',desc:'仙山蓬莱周边海域，灵气充盈。',minRealm:6,minStage:1,monsters:'仙鹤·灵龟·海麒麟'},
{key:'xd',icon:'🏝️',name:'仙岛',desc:'东海修仙圣地，蛟龙盘踞。',minRealm:6,minStage:3,monsters:'蛟龙·海兽·金翅大鹏'},
{key:'lgsc',icon:'🐉',name:'龙宫深处',desc:'龙族核心区域，守卫森严。',minRealm:6,minStage:5,monsters:'龙太子·龙卫·龟丞相'},
{key:'gxzy',icon:'🕳️',name:'归墟之眼',desc:'海中巨大漩涡通归墟，上古海怪蛰伏。',minRealm:6,minStage:8,monsters:'上古海怪·深渊之主·鲲鹏'},
]},
{name:'西域',icon:'🏜️',minRealm:7,maxRealm:8,locations:[
{key:'lszd',icon:'🏜️',name:'流沙之地',desc:'万里黄沙，沙虫潜伏地下。',minRealm:7,minStage:1,monsters:'沙虫·石蝎·沙匪'},
{key:'hyswq',icon:'🌋',name:'火焰山外围',desc:'火焰山外围区域，炙热难耐。',minRealm:7,minStage:3,monsters:'火蜥蜴·熔岩怪·火蚁'},
{key:'gcyj',icon:'🏛️',name:'古城遗迹',desc:'上古仙城废墟，机关重重。',minRealm:7,minStage:5,monsters:'木乃伊·守护石像·古虫'},
{key:'swsm',icon:'☠️',name:'死亡沙漠',desc:'沙漠深处，传说有古虫王沉睡。',minRealm:7,minStage:7,monsters:'沙暴之灵·古虫王·死亡之蝎'},
{key:'hyssc',icon:'🔥',name:'火焰山深处',desc:'火元素凝聚之地，神兽出没。',minRealm:8,minStage:1,monsters:'火麒麟·凤凰雏·烈焰魔'},
{key:'hasl',icon:'🌳',name:'黑暗森林',desc:'终年不见阳光，暗影生物横行。',minRealm:8,minStage:3,monsters:'暗影豹·树精王·魅魔'},
{key:'ygxd',icon:'🏛️',name:'远古神殿',desc:'上古神祇遗留，守护力量仍在。',minRealm:8,minStage:5,monsters:'神仆·守卫巨像·翼蛇'},
{key:'xklf',icon:'🌀',name:'虚空裂缝',desc:'空间不稳定区域，虚空生物涌出。',minRealm:8,minStage:8,monsters:'虚空兽·裂隙魔·空间之影'},
]},
{name:'北冥',icon:'❄️',minRealm:9,maxRealm:10,locations:[
{key:'ydby',icon:'🧊',name:'永冻冰原',desc:'万古冰封之地，极寒妖兽出没。',minRealm:9,minStage:1,monsters:'冰霜巨人·雪女·冰熊'},
{key:'jgxg',icon:'🌌',name:'极光峡谷',desc:'极光之下，冰属性圣兽栖息。',minRealm:9,minStage:3,monsters:'极光之灵·冰龙·雪凤凰'},
{key:'wykwq',icon:'🕳️',name:'万妖窟外围',desc:'万妖窟入口地带，妖气冲天。',minRealm:9,minStage:5,monsters:'妖将·妖帅·石魔皇'},
{key:'tjt',icon:'⚡',name:'天劫台',desc:'引动天劫之地，成功渡劫者飞升仙界。',minRealm:9,minStage:7,monsters:'天雷之灵·雷劫兽·雷公'},
{key:'wyksc',icon:'💀',name:'万妖窟深处',desc:'镇压上古大妖的禁地核心区。',minRealm:10,minStage:1,monsters:'妖皇·上古大妖·魔尊'},
{key:'fszl',icon:'🌈',name:'飞升之路',desc:'渡劫修士的最终试炼之路。',minRealm:10,minStage:3,monsters:'仙人残影·天道守卫·九天雷劫'},
{key:'hdxk',icon:'🌌',name:'混沌虚空',desc:'世界边缘的混沌之地。',minRealm:10,minStage:7,monsters:'混沌兽·虚空之主·灭世魔龙'},
{key:'xjzm',icon:'🚪',name:'仙界之门',desc:'传说中通往仙界的最后一道门。',minRealm:10,minStage:10,monsters:'？？？'},
]},
]
const currentLoc=ref(localStorage.getItem('cur_loc')||'qys')
const currentLocInfo=computed(()=>{for(const r of mapRegions)for(const l of r.locations)if(l.key===currentLoc.value)return l;return null})
function enterLocation(loc:MapLoc){currentLoc.value=loc.key;localStorage.setItem('cur_loc',loc.key);addLog('explore','📍 前往 '+loc.name);activeMenu.value=null}
const showWiki=ref(false)
const wikiTab=ref('realm')
const wikiTabs=[{key:'realm',label:'境界体系'},{key:'root',label:'灵根体系'},{key:'attrs',label:'战斗属性'},{key:'innate',label:'先天属性'},{key:'equip',label:'装备体系'}]
const wikiRealms=[{name:'锻体',coef:1,brk:70,atk:10,def:5,hp:100,mp:50,spd:100,cr:300,cd:15000,dg:200,mr:300,life:100,ss:100},{name:'练气',coef:2,brk:25,atk:25,def:12,hp:250,mp:125,spd:110,cr:500,cd:15500,dg:300,mr:400,life:150,ss:200},{name:'筑基',coef:3,brk:4,atk:60,def:30,hp:600,mp:300,spd:125,cr:800,cd:16000,dg:500,mr:500,life:200,ss:300},{name:'金丹',coef:4,brk:0.8,atk:140,def:70,hp:1400,mp:700,spd:140,cr:1200,cd:17000,dg:700,mr:600,life:300,ss:500},{name:'元婴',coef:5,brk:0.12,atk:300,def:150,hp:3000,mp:1500,spd:160,cr:1800,cd:18000,dg:1000,mr:800,life:500,ss:800},{name:'化神',coef:6,brk:0.015,atk:600,def:300,hp:6000,mp:3000,spd:185,cr:2500,cd:19000,dg:1300,mr:1000,life:800,ss:1200},{name:'炼虚',coef:7,brk:0.002,atk:1200,def:600,hp:12000,mp:6000,spd:210,cr:3200,cd:20000,dg:1600,mr:1200,life:1300,ss:1800},{name:'合体',coef:8,brk:0.0002,atk:2400,def:1200,hp:24000,mp:12000,spd:240,cr:4000,cd:21500,dg:2000,mr:1500,life:2000,ss:2500},{name:'大乘',coef:9,brk:0.00002,atk:4500,def:2250,hp:45000,mp:22500,spd:275,cr:5000,cd:23000,dg:2500,mr:1800,life:3500,ss:3500},{name:'渡劫',coef:10,brk:0.000002,atk:8000,def:4000,hp:80000,mp:40000,spd:310,cr:6000,cd:25000,dg:3000,mr:2200,life:5000,ss:5000}]
const wikiSpiritReqs=[[100,120,150,180,220,270,330,400,500],[600,700,850,1000,1200,1450,1750,2100,2600],[3000,3600,4300,5200,6300,7600,9200,11000,13500],[16000,19000,23000,28000,34000,41000,50000,60000,73000],[88000,105000,125000,150000,180000,215000,260000,310000,375000],[450000,540000,650000,780000,935000,1120000,1350000,1620000,1950000],[2340000,2810000,3370000,4050000,4860000,5830000,7000000,8400000,10080000],[12100000,14500000,17400000,20900000,25100000,30100000,36100000,43300000,52000000],[62400000,74900000,89900000,107900000,129500000,155400000,186500000,223800000,268600000],[322300000,386800000,464200000,557000000,668400000,802100000,962500000,1155000000,1386000000]]
const wikiRootBonuses=[{name:'金灵根',atk:12,def:0,hp:0,mp:0,cr:5,cd:0,dg:0,mr:0},{name:'木灵根',atk:0,def:0,hp:15,mp:0,cr:0,cd:0,dg:0,mr:10},{name:'水灵根',atk:0,def:0,hp:0,mp:12,cr:0,cd:0,dg:3,mr:15},{name:'火灵根',atk:8,def:0,hp:0,mp:0,cr:10,cd:10,dg:0,mr:0},{name:'土灵根',atk:0,def:15,hp:8,mp:0,cr:0,cd:0,dg:0,mr:0},{name:'地灵根',atk:8,def:8,hp:8,mp:8,cr:0,cd:0,dg:0,mr:0},{name:'天灵根',atk:12,def:10,hp:12,mp:10,cr:8,cd:0,dg:0,mr:8}]
const activeProf=ref<any>(null)
const profList=[{key:'dan',icon:'🔥',name:'丹师',desc:'炼制丹药：回血丹、聚气丹、突破丹',bonus:'炼丹成功率+'+Math.floor(player.spiritSense/50)+'%'},{key:'qi',icon:'⚒️',name:'炼器师',desc:'打造法宝、防具、武器',bonus:'炼器成功率+'+Math.floor(player.spiritSense/50)+'%'},{key:'fu',icon:'📝',name:'符师',desc:'制作符箓：攻击符、防御符',bonus:'制符成功率+'+Math.floor(player.spiritSense/50)+'%'},{key:'zhen',icon:'🌀',name:'阵法师',desc:'布置阵法：修炼加速、洞府守护',bonus:'阵法威力+'+Math.floor(player.spiritSense/50)+'%'},{key:'zhi',icon:'🌱',name:'灵植师',desc:'种植灵草，提供炼丹炼器材料',bonus:'产量+'+Math.floor(player.spiritSense/50)+'% 品质+'+(Math.floor(player.spiritSense/200))+'%'}]
function openProfession(p:any){activeProf.value=p;loadRecipes()}
const wikiQuality=[{name:'无品',speed:0.7,brk:1,attr:0.5,chance:8},{name:'下品',speed:1.0,brk:5,attr:1.0,chance:20},{name:'中品',speed:1.3,brk:15,attr:1.5,chance:40},{name:'上品',speed:1.6,brk:40,attr:2.0,chance:22},{name:'极品',speed:2.0,brk:100,attr:3.0,chance:8}]

const showCultTooltip=ref(false)
const tooltipStyle=ref({top:'0px',left:'0px'})
function showTooltip(){
  if(!yyWrapRef.value)return
  const r=yyWrapRef.value.getBoundingClientRect()
  tooltipStyle.value={top:r.top+r.height/2+'px',left:r.right+16+'px'}
  showCultTooltip.value=true
}
const realmCoef=computed(()=>realmCoefs[player.realmId]||1)
const rootMult=computed(()=>rootMults[player.rootQuality]||1.0)
const cultBaseVal=computed(()=>10+realmCoef.value*player.realmStage)
const hasRoot=computed(()=>player.spiritName!=='无灵根')
function getPID(){return localStorage.getItem('player_id')}
function getToken(){return localStorage.getItem('token')}
async function apiPost(path:string,body:any={}){try{const t=getToken();const r=await fetch(path,{method:'POST',headers:{'Content-Type':'application/json',Authorization:'Bearer '+t},body:JSON.stringify(body)});if(r.status===401){const newToken=await refreshToken();if(newToken){const r2=await fetch(path,{method:'POST',headers:{'Content-Type':'application/json',Authorization:'Bearer '+newToken},body:JSON.stringify(body)});return await r2.json()}};return await r.json()}catch{return null}}
async function refreshToken(){try{const rt=localStorage.getItem('refresh_token');if(!rt)return null;const r=await fetch('/auth/refresh',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({refresh_token:rt})});if(!r.ok)return null;const d=await r.json();if(d.access_token){localStorage.setItem('token',d.access_token);if(d.refresh_token)localStorage.setItem('refresh_token',d.refresh_token);return d.access_token}return null}catch{return null}}
async function loadPlayer(){
  try{
    const pid=getPID();if(!pid)return
    let r=await fetch('/api/v1/player/'+pid,{headers:{Authorization:'Bearer '+getToken()}})
    if(r.status===401){const nt=await refreshToken();if(nt)r=await fetch('/api/v1/player/'+pid,{headers:{Authorization:'Bearer '+nt}})}
    if(!r.ok)return;const d=await r.json()
    if(d.code===0&&d.data){
      const p=d.data.player||d.data
      player.name=p.name||'修仙者'
      player.gender=p.gender||'male'
      player.realmName=d.data.realm_name||'锻体'
      player.realmId=p.realm_id||1
      player.realmStage=p.realm_stage||1
      player.spiritName=rootNames[p.spirit_root]||'无灵根'
      player.qualityName=qualityNames[p.root_quality]||'无品'
      player.rootQuality=p.root_quality||0
      player.level=p.level||1
      player.hp=p.hp||100;player.maxHp=p.max_hp||100
      if(player.hp<=0&&!isDead.value){const saved=localStorage.getItem('dead_until');if(saved&&parseInt(saved)>Date.now()){isDead.value=true;reviveCountdown.value=Math.ceil((parseInt(saved)-Date.now())/1000);if(reviveTimer)clearInterval(reviveTimer);reviveTimer=setInterval(()=>{reviveCountdown.value--;if(reviveCountdown.value<=0){clearInterval(reviveTimer);player.hp=Math.ceil(player.maxHp*0.5);player.mp=Math.ceil(player.maxMp*0.5);player.spiritSense=Math.max(player.spiritSense,10);isDead.value=false;deathLog.value=null;localStorage.removeItem('dead_until');addLog('combat','✨ 道心重铸！已复活')}},1000)}}
      player.mp=p.mp||50;player.maxMp=p.max_mp||50
      player.attack=p.attack||10;player.defense=p.defense||5;player.speed=p.speed||10
      player.critRate=Math.round((p.crit_rate||300)/100)  // ×100 → %
      player.critDmg=Math.round((p.crit_dmg||15000)/100)
      player.dodge=Math.round((p.dodge||200)/100)
      player.hit=Math.round((p.hit||9500)/100)
      player.cultBonus=Math.round((p.cult_bonus||0)/100)
      player.breakBonus=Math.round((p.break_bonus||0)/100)
      player.mpRegen=Math.round((p.mp_regen||300)/100)
      player.lifespan=p.lifespan||100
      if(p.created_at&&!localStorage.getItem('created_at'))localStorage.setItem('created_at',p.created_at)
      player.comprehension=p.comprehension||10
      player.luck=p.luck||10
      player.spiritSense=p.spirit_sense||100
      const atk=p.attack||0,def=p.defense||0,spd=p.speed||0,hp=p.max_hp||100,mp=p.max_mp||50;const cr=Math.floor((p.crit_rate||300)/100),cd=Math.floor((p.crit_dmg||15000)/100),dg=Math.floor((p.dodge||200)/100),ht=Math.floor((p.hit||9500)/100),mr=Math.floor((p.mp_regen||300)/100);player.power=Math.floor(atk*1.5+def*1.2+hp*0.15+mp*0.1+spd*0.5+cr*3+cd*0.15+dg*3+ht*0.5+mr*2)
      player.spirit=p.spirit_power||0
      player.maxSpirit=p.max_spirit||100
      player.gold=p.gold||0;player.jade=p.jade||0
      player.cultRate=d.data.cult_rate||11
      player.breakRate=d.data.break_rate||90
    }
  }catch{}
}
const LOG_KEY='game_logs'
function loadLogs(){try{const d=localStorage.getItem(LOG_KEY);if(d){const items=JSON.parse(d);const cutoff=Date.now()-30*86400000;logs.push(...items.filter((i:any)=>i.ts>cutoff))}}catch{}}
function saveLogs(){try{const items=logs.slice(0,500);localStorage.setItem(LOG_KEY,JSON.stringify(items))}catch{}}
function addLog(type:string,text:string){const entry:any={ts:Date.now(),time:new Date().toLocaleTimeString('zh-CN',{hour12:false}).slice(0,8),text,type};logs.unshift(entry);saveLogs()}
let cultTimer=0
let syncTimer=0
let lastSyncTime=0  // 上次同步时间戳（模块级，供 beforeunload 使用）
function startAutoCultivate(){
  const pid=getPID()
  lastSyncTime=Date.now()
  // 前端实时计算修为+HP/MP恢复（每秒更新）
  cultTimer=window.setInterval(()=>{
    if(player.spirit<player.maxSpirit){
      const rate=player.cultRate
      player.spirit=Math.min(player.maxSpirit,player.spirit+Math.floor(rate))
    }
    // HP 每10秒回1%（每秒0.1%）
    if(player.hp<player.maxHp) player.hp=Math.min(player.maxHp,player.hp+Math.ceil(player.maxHp*0.001))
    // MP 按 mp_regen/10000 速率恢复（如500→5%/秒）
    if(player.mp<player.maxMp) player.mp=Math.min(player.maxMp,player.mp+Math.ceil(player.maxMp*player.mpRegen/10000))
  },1000)
  // 后端同步（每60秒校准+持久化，防止刷新丢进度）
  syncTimer=window.setInterval(async()=>{
    const elapsed=Math.floor((Date.now()-lastSyncTime)/1000)
    if(elapsed<5)return
    lastSyncTime=Date.now()
    localStorage.setItem(OFFLINE_KEY,String(lastSyncTime))
    const d=await apiPost('/api/v1/player/'+pid+'/cultivate/tick',{seconds:elapsed})
    if(d&&d.code===0&&d.data){
      player.spirit=d.data.spirit_power
      player.maxSpirit=d.data.max_spirit
      player.cultRate=d.data.rate
      if(d.data.hp!==undefined)player.hp=d.data.hp
      if(d.data.mp!==undefined)player.mp=d.data.mp
      if(d.data.is_full){addLog('cult','修为已满，准备突破！')}
    }
  },60000)
}
async function doBreakthrough(){
  if(isDead.value){addLog('system','💀 道心已碎，无法突破');return}
  const pid=getPID();if(!pid)return
  // 突破前先同步一次，确保后端修为最新
  await apiPost('/api/v1/player/'+pid+'/cultivate/tick',{seconds:3})
  const d=await apiPost('/api/v1/player/'+pid+'/breakthrough')
  if(!d)return
  if(d.code===0&&d.data){
    if(d.data.success){
      player.spirit=0
      player.maxSpirit=d.data.new_realm<=10?100*(d.data.new_realm||1):9999
      player.realmName=d.data.realm_name||player.realmName
      if(d.data.realm_stage)player.realmStage=d.data.realm_stage
      else player.realmStage=1
      addLog('cult','🎉 突破成功！当前境界：'+d.data.realm_name)
      // 刷新完整数据
      loadPlayer()
    }else{
      addLog('cult','突破失败，修为-20%')
      loadPlayer()
    }
  }else{
    addLog('cult','突破失败：'+(d.msg||'修为不足'))
  }
}
const OFFLINE_KEY='cult_offline_ms'
// calcOfflineGains: 基于"上次已知同步时间"计算离线收益，避免重复算已保存的修为
async function calcOfflineGains(){
  const pid=getPID();if(!pid)return
  const saved=localStorage.getItem(OFFLINE_KEY)
  if(!saved)return
  const elapsed=Math.floor((Date.now()-parseInt(saved))/1000)
  localStorage.removeItem(OFFLINE_KEY)
  if(elapsed>=3){
    const capped=Math.min(elapsed,28800)
    const d=await apiPost('/api/v1/player/'+pid+'/cultivate/tick',{seconds:capped})
    if(d&&d.code===0&&d.data&&d.data.gained>0){
      addLog('cult','⏳ 离线获得修为 +'+d.data.gained+' ('+Math.floor(elapsed/60)+'分)')
      loadPlayer()
    }
  }
  if(elapsed<28800){
    player.isMeditating=true
    if(training.value){training.value=false;clearInterval(trainTimer)}
    localStorage.setItem(OFFLINE_KEY,String(Date.now()))
    lastSyncTime=Date.now()
    startAutoCultivate()
  }else{
    addLog('cult','闭关已满8小时，自动出关')
  }
}
// 页面关闭/刷新：保存"上次成功同步时间"作为下次离线计算的起点
if(typeof window!=='undefined'){window.addEventListener('beforeunload',()=>{
  if(!player.isMeditating)return
  // 将 lastSyncTime 写入 OFFLINE_KEY，这样下次 reload 只算未同步的部分
  if(lastSyncTime>0)localStorage.setItem(OFFLINE_KEY,String(lastSyncTime))
})}
const showTrainPanel=ref(false)
const training=ref(false)
const trainMult=ref(1)
const encounterResult=ref<any>(null)
const encounterType=ref('')
const combatReport=ref<any>(null)
const isDead=ref(localStorage.getItem('dead_until')?parseInt(localStorage.getItem('dead_until')!)>Date.now():false)
const reviveCountdown=ref(isDead.value?Math.ceil((parseInt(localStorage.getItem('dead_until')||'0')-Date.now())/1000):0)
const deathLog=ref<any>(null) // 存储死亡时的战斗过程
let trainTimer=0
let reviveTimer:any=null
if(isDead.value&&reviveCountdown.value>0){
  reviveTimer=setInterval(()=>{reviveCountdown.value--;if(reviveCountdown.value<=0){clearInterval(reviveTimer);player.hp=Math.ceil(player.maxHp*0.5);player.mp=Math.ceil(player.maxMp*0.5);player.spiritSense=Math.max(player.spiritSense,10);isDead.value=false;deathLog.value=null;localStorage.removeItem('dead_until');addLog('combat','✨ 已复活！恢复50%状态')}},1000)
}
const trainCultEst=computed(()=>(player.realmId*10+Math.floor(player.realmId*10))*trainMult.value)
const trainGoldEst=computed(()=>(player.realmId*5+Math.floor(player.realmId*7))*trainMult.value)
function toggleTraining(){
  training.value=!training.value;showTrainPanel.value=true
  if(training.value){
    if(isDead.value){training.value=false;addLog('system','💀 道心已碎，无法历练');return}
    if(player.hp<player.maxHp*0.05){training.value=false;addLog('system','血量过低(<5%)，无法历练');return}
    if(player.spiritSense<trainMult.value){training.value=false;addLog('system','神识不足');return}
    if(player.isMeditating){player.isMeditating=false;clearInterval(cultTimer);clearInterval(syncTimer);addLog('cult','结束闭关')}
    addLog('explore','⚔️ 开始历练 ×'+trainMult.value)
    startTraining()
  }else{clearInterval(trainTimer);addLog('explore','结束历练')}
}
const pveReport=ref<any>(null),pveRounds=ref<any[]>([])
async function startPve(loc:any){const pid=getPID();if(!pid)return;activeMenu.value=null
  const d=await apiPost('/api/v1/player/'+pid+'/pve',{loc_key:loc.key})
  if(!d||d.code!==0)return;if(!d.data.success){addLog('system',d.data.reason||'战斗失败');return}
  player.spiritSense=d.data.sense_left;player.hp=d.data.hp;player.mp=d.data.mp
  pveReport.value={won:d.data.won,dead:d.data.dead,cult:d.data.cult_gain,gold:d.data.gold_gain,monster:d.data.monster}
  if(d.data.rounds){pveRounds.value=d.data.rounds;let ri=0;(function play(){if(ri>=pveRounds.value.length){addLog('combat','⚔️ 战斗结束 — '+(pveReport.value.won?'胜利！+'+pveReport.value.cult+'修为 +'+pveReport.value.gold+'灵石':'战败'));if(pveReport.value.dead)startRevive();return}const r=pveRounds.value[ri];let t='⚔️ 第'+r.round+'回合: 造成'+r.player_dmg+'伤害';if(r.desc)t+=' ('+r.desc+')';t+=' | 受到'+r.monster_dmg+'伤害 | ❤️'+r.player_hp+' 👹'+(r.monster_hp>0?r.monster_hp:'击败');addLog('combat',t);ri++;setTimeout(play,800)})()}}
function startRevive(){
  isDead.value=true;clearInterval(trainTimer);training.value=false
  const until=Date.now()+5000;localStorage.setItem('dead_until',String(until))
  reviveCountdown.value=5;deathLog.value=combatReport.value
  const deathTitle=player.gender==='female'?'💀 香消玉殒':'💀 道心破碎'
  addLog('combat',deathTitle+'！5秒后复活，恢复50%状态')
  const maleJokes=['刚才是不是有人飞升了？哦，看错了，是脸着地。','这位道友被怪物一个喷嚏喷死了...','修仙界快讯：又一位猛男体验了免费回城服务。','怪物：就这？我还没用力呢！','据目击者称，临终前大喊"我还没存档！！"','系统提示：该猛男已进入躺平状态。','男人至死是少年，但他现在真的是少年了...','兄弟，下次记得先存档再打架。']
  const femaleJokes=['仙女下凡...脸先着地了。','这位仙子被怪物吓得花容失色，香消玉殒。','修仙界快讯：一位仙女体验了免费回城服务。','怪物：对不起对不起，我不是故意的！','修真界八卦：又一位仙子去轮回重修了。','系统提示：该仙子已进入美容觉模式。','红颜薄命，但5秒后又是一条好汉！','下次记得带护花使者一起出门哦~']
  const jokes=player.gender==='female'?femaleJokes:maleJokes
  chatMessages.push({time:new Date().toLocaleTimeString('zh-CN',{hour12:false}).slice(0,5),name:'⚰️ 讣告',text:player.name+' '+deathTitle+' — '+jokes[Math.floor(Math.random()*jokes.length)],color:'#e53935',channel:'world'})
  reviveTimer=setInterval(()=>{
    reviveCountdown.value--
    if(reviveCountdown.value<=0){
      clearInterval(reviveTimer)
      player.hp=Math.ceil(player.maxHp*0.5);player.mp=Math.ceil(player.maxMp*0.5);player.spiritSense=Math.max(player.spiritSense,10)
      isDead.value=false;deathLog.value=null;combatReport.value=null
      localStorage.removeItem('dead_until')
      addLog('combat','✨ 道心重铸！已复活')
      // 同步后端
      const pid=getPID();if(pid)apiPost('/api/v1/player/'+pid+'/cultivate/tick',{seconds:1})
    }
  },1000)
}
async function doTrain(){
  const pid=getPID();if(!pid)return
  const d=await apiPost('/api/v1/player/'+pid+'/train',{multiplier:trainMult.value})
  if(!d||d.code!==0)return
  if(!d.data.success){training.value=false;clearInterval(trainTimer);addLog('system',d.data.reason||'历练失败');if(d.data.dead)startRevive();return}
  player.spiritSense=d.data.sense_left
  if(d.data.cult_gain>0){player.spirit+=d.data.cult_gain;if(player.spirit>player.maxSpirit)player.spirit=player.maxSpirit}
  if(d.data.gold_gain>0)player.gold+=d.data.gold_gain
  if(d.data.hp!==undefined)player.hp=d.data.hp
  if(d.data.mp!==undefined)player.mp=d.data.mp
  if(d.data.encounter){
    const qn=qualityNames[d.data.new_quality]||''
    const sn=d.data.spirit_name||''
    encounterType.value=d.data.encounter
    encounterResult.value={title:d.data.encounter,old:player.spiritName+'·'+player.qualityName,new:sn+'·'+qn,icon:d.data.encounter==='仙人点化'?'✨':'🍄'}
    addLog('explore','🌟 奇遇！'+d.data.encounter+'：灵根洗炼 '+encounterResult.value.old+' → '+encounterResult.value.new)
    if(d.data.encounter==='仙人点化'){chatMessages.push({time:new Date().toLocaleTimeString('zh-CN',{hour12:false}).slice(0,5),name:'📢 系统',text:player.name+' 获得仙人点化，灵根品质提升至 '+sn+'·'+qn+'！',color:'#ffd700',channel:'world'})}
    setTimeout(()=>{encounterResult.value=null;encounterType.value=''},5000)
  }
  if(d.data.dead){startRevive();return}
  let log='🔍 历练 ×'+trainMult.value+': +'+d.data.cult_gain+'修为 +'+d.data.gold_gain+'灵石'
  if(d.data.combat){
    log+=' ['+d.data.combat+']'
    combatReport.value=d.data.rounds?{rounds:d.data.rounds,result:d.data.combat,monster:d.data.monster_name}:null
    // 每秒播放一回合战斗日志
    if(d.data.rounds){
      let ri=0
      const playRound=()=>{
        if(ri>=d.data.rounds.length){addLog('combat','⚔️ 战斗结束 — '+d.data.combat);combatReport.value=null;return}
        const r=d.data.rounds[ri]
        let txt='⚔️ 第'+r.round+'回合: 造成'+r.player_dmg+'伤害'
        if(r.desc)txt+=' ('+r.desc+')'
        txt+=' | 受到'+r.monster_dmg+'伤害 | ❤️'+r.player_hp+' 👹'+(r.monster_hp>0?r.monster_hp:'击败')
        addLog('combat',txt)
        ri++;setTimeout(playRound,1000)
      }
      playRound()
    }
  }else{combatReport.value=null}
  addLog('explore',log)
  if(player.spiritSense<trainMult.value){training.value=false;clearInterval(trainTimer);addLog('system','神识耗尽，历练停止')}
}
async function doTrainOnce(){
  const pid=getPID();if(!pid)return
  const d=await apiPost('/api/v1/player/'+pid+'/train',{multiplier:trainMult.value})
  if(!d||d.code!==0)return
  if(!d.data.success){training.value=false;clearInterval(trainTimer);addLog('system',d.data.reason||'历练失败');if(d.data.dead)startRevive();return}
  player.spiritSense=d.data.sense_left
  if(d.data.cult_gain>0){player.spirit+=d.data.cult_gain;if(player.spirit>player.maxSpirit)player.spirit=player.maxSpirit}
  if(d.data.gold_gain>0)player.gold+=d.data.gold_gain
  if(d.data.hp!==undefined)player.hp=d.data.hp
  if(d.data.mp!==undefined)player.mp=d.data.mp
  let log='🔍 历练 ×'+trainMult.value+': +'+d.data.cult_gain+'修为 +'+d.data.gold_gain+'灵石'
  if(d.data.combat){
    log+=' ['+d.data.combat+']'
    combatReport.value=d.data.rounds?{rounds:d.data.rounds,result:d.data.combat}:null
    if(d.data.rounds){
      let ri=0
      const playRound=()=>{
        if(ri>=d.data.rounds.length){combatReport.value=null;addLog('combat','⚔️ 战斗结束 — '+d.data.combat);if(d.data.dead)startRevive();return}
        const r=d.data.rounds[ri]
        let txt='⚔️ 第'+r.round+'回合: 造成'+r.player_dmg+'伤害'
        if(r.desc)txt+=' ('+r.desc+')'
        txt+=' | 受到'+r.monster_dmg+'伤害 | ❤️'+r.player_hp+' 👹'+(r.monster_hp>0?r.monster_hp:'击败')
        addLog('combat',txt);ri++;setTimeout(playRound,1000)
      };playRound()
    }
  }else{combatReport.value=null;lastTrainTime=Date.now()}
  addLog('explore',log)
  if(player.spiritSense<trainMult.value){training.value=false;clearInterval(trainTimer);addLog('system','神识耗尽，历练停止')}
}
function scheduleNext(){
  lastTrainTime=Date.now()
  const doIt=async()=>{
    await doTrainOnce()
    if(!training.value||isDead.value||player.spiritSense<trainMult.value)return
    const elapsed=Date.now()-lastTrainTime
    const wait=Math.max(0,3000-elapsed)
    setTimeout(()=>scheduleNext(),wait)
  };doIt()
}
function startManual(){doTrainOnce()}
function startTraining(){doTrainOnce();trainTimer=window.setInterval(doTrainOnce,3000)}
function toggleMeditation(){
  if(isDead.value){addLog('system','💀 道心已碎，无法闭关');return}
  player.isMeditating=!player.isMeditating
  if(player.isMeditating){
    // 互斥：关闭历练
    if(training.value){training.value=false;clearInterval(trainTimer);addLog('explore','结束历练')}
    lastSyncTime=Date.now()
    localStorage.setItem(OFFLINE_KEY,String(lastSyncTime))
    addLog('cult','开始闭关修炼')
    startAutoCultivate()
  }else{
    localStorage.removeItem(OFFLINE_KEY)
    clearInterval(cultTimer)
    clearInterval(syncTimer)
    addLog('cult','结束闭关')
  }
}
watch(()=>player.isMeditating,(v)=>{setTimeout(()=>{if(v&&energyCanvas.value)startEnergyCanvas(energyCanvas.value);else stopEnergyCanvas()},50)})
const expPercent=computed(()=>Math.min(100,Math.round((player.spirit/player.maxSpirit)*100)))
const hpPct=computed(()=>Math.min(100,Math.round((player.hp/player.maxHp)*100)))
const mpPct=computed(()=>Math.min(100,Math.round((player.mp/player.maxMp)*100)))
const createdDate=computed(()=>{try{const p=localStorage.getItem('created_at');return p?new Date(p):new Date()}catch{return new Date()}})
const ageDays=computed(()=>Math.floor((Date.now()-createdDate.value.getTime())/86400000))
const ageBracket=computed(()=>{const r=ageDays.value/player.lifespan;if(r<0.25)return'少年';if(r<0.5)return'青年';if(r<0.75)return'中年';return'老年'})
const smallStageRates:Record<number,number>={0:50,1:65,2:80,3:90,4:99}
const smallBreakRate=computed(()=>smallStageRates[player.rootQuality]||50)
const yySpeed=computed(()=>Math.max(2,14-(player.level||1)))

const logFilter=ref('all')
const logLocked=ref(false)
const logs=reactive([])
const filteredLogs=computed(()=>logFilter.value==='all'?logs:logs.filter(l=>l.type===logFilter.value))
	const CHAT_KEY='game_chat'
	async function loadChatHistory(){
	  try{
	    const t=getToken();if(!t)return
	    const r=await fetch('/api/v1/chat/history?channel=world&limit=50',{headers:{Authorization:'Bearer '+t}})
	    const d=await r.json()
	    if(d.data&&d.data.length){d.data.reverse().forEach((m:any)=>{chatMessages.push({time:new Date(m.created_at).toLocaleTimeString('zh-CN',{hour12:false}).slice(0,5),name:m.sender_name||'未知',text:m.content||'',color:m.is_system?'#ffd700':'#d4a843',channel:m.channel||'world'})})}
	  }catch{}
	}
	function saveChat(){/* 聊天由后端MongoDB持久化 */}
const chatTab=ref('all')
	const chatMessages=reactive([])
const chatInput=ref('')
const filteredChat=computed(()=>chatTab.value==='all'?chatMessages:chatMessages.filter(m=>m.channel===chatTab.value))
const chatPlaceholder=computed(()=>{const t:Record<string,string>={all:'发送全服消息...',world:'世界频道发言...',sect:'宗门频道发言...',private:'输入私聊对象和内容...',friend:'给好友留言...',daoyou:'道侣悄悄话...',master:'向师父/徒弟发言...'};return t[chatTab.value]||'说点什么...'})
const chatEmptyText=computed(()=>{const t:Record<string,string>={all:'暂无消息',world:'世界频道静默中...',sect:'宗门频道暂无消息',private:'暂无私聊消息',friend:'暂无好友消息',daoyou:'暂无道侣消息',master:'暂无师徒消息'};return t[chatTab.value]||''})
const bpItems=ref<any[]>([])
async function loadBackpack(){const pid=getPID();if(!pid)return;try{const r=await fetch('/api/v1/player/'+pid+'/inventory',{headers:{Authorization:'Bearer '+getToken()}});const d=await r.json();bpItems.value=d.data||[]}catch{}}
async function useBpPill(pill:any){const pid=getPID();if(!pid)return;await apiPost('/api/v1/player/'+pid+'/pills/use',{pill_id:pill.id});addLog('item','💊 使用 '+pill.name);loadBackpack();loadPlayer()}
const playerId=computed(()=>parseInt(getPID()||'0'))
const pigGroups=reactive({friends:true,requests:true,daolv:false,master:false})
const friends=ref<any[]>([]),pendingRequests=ref<any[]>([]),searchResults=ref<any[]>([])
const friendSearch=ref(''),activePeer=ref(0),activePeerName=ref(''),privateInput=ref(''),privateMessages=ref<any[]>([])
async function loadFriends(){const pid=getPID();if(!pid)return;try{const r=await fetch('/api/v1/player/'+pid+'/friends',{headers:{Authorization:'Bearer '+getToken()}});const d=await r.json();if(d.data)friends.value=d.data}catch{}}
async function loadPending(){const pid=getPID();if(!pid)return;try{const r=await fetch('/api/v1/player/'+pid+'/friends/pending',{headers:{Authorization:'Bearer '+getToken()}});const d=await r.json();if(d.data)pendingRequests.value=d.data}catch{}}
async function searchPlayers(){const pid=getPID();if(!pid||!friendSearch.value.trim())return;try{const r=await fetch('/api/v1/player/'+pid+'/friends/search?q='+encodeURIComponent(friendSearch.value),{headers:{Authorization:'Bearer '+getToken()}});const d=await r.json();searchResults.value=d.data||[]}catch{}}
async function addFriend(fid:number){const pid=getPID();if(!pid)return;await apiPost('/api/v1/player/'+pid+'/friends/add',{friend_id:fid});addLog('social','好友申请已发送');searchResults.value=[];friendSearch.value=''}
async function acceptFriend(fid:number){const pid=getPID();if(!pid)return;await apiPost('/api/v1/player/'+pid+'/friends/accept',{friend_id:fid});loadFriends();loadPending();addLog('social','已添加好友')}
async function removeFriend(fid:number){const pid=getPID();if(!pid)return;await apiPost('/api/v1/player/'+pid+'/friends/remove',{friend_id:fid});loadFriends();loadPending()}
async function openChat(f:any){activePeer.value=f.id;activePeerName.value=f.nickname;const pid=getPID();if(!pid)return;try{const r=await fetch('/api/v1/player/'+pid+'/messages?peer_id='+f.id,{headers:{Authorization:'Bearer '+getToken()}});const d=await r.json();privateMessages.value=(d.data||[]).reverse()}catch{}}
async function sendPrivate(){const pid=getPID();const v=privateInput.value.trim();if(!v||!activePeer.value||!pid)return;const d=await apiPost('/api/v1/player/'+pid+'/messages/send',{to_id:activePeer.value,text:v});if(d&&d.code===0){privateMessages.value.push({from_id:parseInt(pid),to_id:activePeer.value,text:v,created_at:new Date().toISOString()});privateInput.value=''}}
const equipSlots=[{key:'weapon',icon:'🗡️',name:'武器'},{key:'crown',icon:'👑',name:'发冠'},{key:'robe',icon:'👘',name:'法袍'},{key:'bracer',icon:'🛡️',name:'护腕'},{key:'belt',icon:'🎗️',name:'腰带'},{key:'boots',icon:'👢',name:'云靴'},{key:'necklace',icon:'📿',name:'项链'},{key:'ring',icon:'💍',name:'戒指'},{key:'artifact',icon:'🔮',name:'法宝'},{key:'mount',icon:'🐉',name:'坐骑'}]
const playerEquips=ref<any[]>([]);const equipCraftSlot=ref('')
function getEquip(slot:string){return playerEquips.value.find((e:any)=>e.slot===slot)}
async function loadEquips(){const pid=getPID();if(!pid)return;try{const r=await fetch('/api/v1/player/'+pid+'/equipment',{headers:{Authorization:'Bearer '+getToken()}});const d=await r.json();playerEquips.value=d.data||[]}catch{}}
async function craftEquip(s:any){equipCraftSlot.value=s.key;const pid=getPID();if(!pid)return;await apiPost('/api/v1/player/'+pid+'/equipment/craft',{slot:s.key,tier:player.realmId});loadEquips();addLog('item','⚒️ 打造 '+s.name);equipCraftSlot.value=''}
	let ws:WebSocket|null=null
	let chatWs:WebSocket|null=null
	function connectWS(){
	  const t=getToken();if(!t)return
	  try{
	    ws=new WebSocket((location.protocol==='https:'?'wss://':'ws://')+location.host+'/ws?token='+t)
	    ws.onmessage=(e)=>{try{const m=JSON.parse(e.data);if(m.type==='chat'){handleServerChat(m)}}catch{}}
	    ws.onclose=()=>{setTimeout(connectWS,5000)}
	    ws.onerror=()=>{ws?.close()}
	  }catch{}
	  connectChatWS()
	}
	function connectChatWS(){
	  try{
	    chatWs=new WebSocket((location.protocol==='https:'?'wss://':'ws://')+location.host+'/api/v1/chat/ws?user_id='+(getPID()||''))
	    chatWs.onmessage=(e)=>{try{const m=JSON.parse(e.data);handleServerChat(m)}catch{}}
	    chatWs.onclose=()=>{setTimeout(connectChatWS,5000)}
	    chatWs.onerror=()=>{chatWs?.close()}
	  }catch{}
	}
	function handleServerChat(m:any){
	  chatMessages.push({time:new Date(m.created_at||Date.now()).toLocaleTimeString('zh-CN',{hour12:false}).slice(0,5),name:m.sender_name||m.name||'未知',text:m.content||m.text||'',color:m.is_system?'#ffd700':(m.color||'#d4a843'),channel:m.channel||'world'})
	  if(chatMessages.length>300)chatMessages.shift()
	}
	function sendChat(){
	  const v=chatInput.value.trim();if(!v)return
	  const ch=chatTab.value==='all'?'world':chatTab.value
	  chatInput.value=''
	  if(chatWs&&chatWs.readyState===WebSocket.OPEN){
	    chatWs.send(JSON.stringify({channel:ch,content:v,sender_name:player.name||'修仙者'}))
	  }else if(ws&&ws.readyState===WebSocket.OPEN){
	    ws.send(JSON.stringify({type:'chat',channel:ch,text:v}))
	  }
	  // 乐观更新
	  handleServerChat({channel:ch,content:v,sender_name:player.name||'修仙者'})
	}
const onlineCount=ref(0),registeredCount=ref(0)
async function fetchStats(){try{const r=await fetch('/health');const d=await r.json();if(d.online!==undefined)onlineCount.value=d.online;if(d.registered!==undefined)registeredCount.value=d.registered}catch{}}
onMounted(()=>{fetchStats();setInterval(fetchStats,30000);loadLogs();loadChatHistory();loadPlayer();loadPills();loadRecipes();loadBackpack();loadFriends();loadPending();calcOfflineGains();connectWS();setTimeout(()=>{addLog('system','🌟 登录修仙世界')},500)})
function fmt(n:number):string{return n>=10000?(n/10000).toFixed(1)+'万':n.toLocaleString()}
const currentRealmIndex=2
const realms=[{name:'练气'},{name:'筑基'},{name:'金丹'},{name:'元婴'},{name:'化神'},{name:'炼虚'},{name:'合体'},{name:'大乘'},{name:'渡劫'}]
const quotes=['大道无形，生育天地','修仙之路，步步登天','一粒金丹吞入腹，始知我命不由天','千淘万漉虽辛苦，吹尽狂沙始到金']
const quoteIndex=ref(0),currentQuote=computed(()=>quotes[quoteIndex.value])
let energyAnimId=0
function startEnergyCanvas(canvas){
  const ctx=canvas.getContext('2d')
  const W=()=>{canvas.width=canvas.offsetWidth;canvas.height=canvas.offsetHeight}
  W();const resize=new ResizeObserver(W);resize.observe(canvas)
  const colors=['#ff6b6b','#ffd93d','#6bcb77','#4d96ff','#d4a843','#ff6b9e','#64ffda','#b388ff','#ff9800','#e040fb']
  let particles=[]
  function spawn(){if(particles.length<25&&Math.random()<.4){const a=Math.random()*Math.PI*2;const d=90+Math.random()*40;particles.push({x:canvas.width/2+Math.cos(a)*d,y:canvas.height/2+Math.sin(a)*d,vx:Math.cos(a+Math.PI)*.8+Math.random()*.4-.2,vy:Math.sin(a+Math.PI)*.8+Math.random()*.4-.2,life:1,decay:.008+Math.random()*.02,color:colors[Math.floor(Math.random()*colors.length)],len:15+Math.random()*30,angle:a})}}
  function draw(){ctx.clearRect(0,0,canvas.width,canvas.height);spawn();particles=particles.filter(p=>p.life>0);particles.forEach(p=>{p.x+=p.vx;p.y+=p.vy;p.life-=p.decay;ctx.save();ctx.globalAlpha=p.life*.7;ctx.strokeStyle=p.color;ctx.lineWidth=1.5+Math.random();ctx.beginPath();ctx.moveTo(p.x,p.y);ctx.lineTo(p.x+Math.cos(p.angle)*p.len*p.life,p.y+Math.sin(p.angle)*p.len*p.life);ctx.stroke();ctx.restore()});energyAnimId=requestAnimationFrame(draw)}
  draw()
}
function stopEnergyCanvas(){cancelAnimationFrame(energyAnimId)}
const energyCanvas=ref(null)
const serverStartTime=Date.now()
function getLunarDate(d){
  // Lunar calendar data: encoded as bit flags for each year
  // Each entry: [year data, month data...]
  // 2026 lunar new year: Feb 17 (gregorian)
  const lunarInfo=[0x04bd8,0x04ae0,0x0a570,0x054d5,0x0d260,0x0d950,0x16554,0x056a0,0x09ad0,0x055d2,
    0x04ae0,0x0a5b6,0x0a4d0,0x0d250,0x1d255,0x0b540,0x0d6a0,0x0ada2,0x095b0,0x14977,
    0x04970,0x0a4b0,0x0b4b5,0x06a50,0x06d40,0x1ab54,0x02b60,0x09570,0x052f2,0x04970,
    0x06566,0x0d4a0,0x0ea50,0x06e95,0x05ad0,0x02b60,0x186e3,0x092e0,0x1c8d7,0x0c950,
    0x0d4a0,0x1d8a6,0x0b550,0x056a0,0x1a5b4,0x025d0,0x092d0,0x0d2b2,0x0a950,0x0b557,
    0x06ca0,0x0b550,0x15355,0x04da0,0x0a5b0,0x14573,0x052b0,0x0a9a8,0x0e950,0x06aa0,
    0x0aea6,0x0ab50,0x04b60,0x0aae4,0x0a570,0x05260,0x0f263,0x0d950,0x05b57,0x056a0,
    0x096d0,0x04dd5,0x04ad0,0x0a4d0,0x0d4d4,0x0d250,0x0d558,0x0b540,0x0b6a0,0x195a6,
    0x095b0,0x049b0,0x0a974,0x0a4b0,0x0b27a,0x06a50,0x06d40,0x0af46,0x0ab60,0x09570,
    0x04af5,0x04970,0x064b0,0x074a3,0x0ea50,0x06b58,0x05ac0,0x0ab60,0x096d5,0x092e0,
    0x0c960,0x0d954,0x0d4a0,0x0da50,0x07552,0x056a0,0x0abb7,0x025d0,0x092d0,0x0cab5,
    0x0a950,0x0b4a0,0x0baa4,0x0ad50,0x055d9,0x04ba0,0x0a5b0,0x15176,0x052b0,0x0a930,
    0x07954,0x06aa0,0x0ad50,0x05b52,0x04b60,0x0a6e6,0x0a4e0,0x0d260,0x0ea65,0x0d530,
    0x05aa0,0x076a3,0x096d0,0x04afb,0x04ad0,0x0a4d0,0x1d0b6,0x0d250,0x0d520,0x0dd45,
    0x0b5a0,0x056d0,0x055b2,0x049b0,0x0a577,0x0a4b0,0x0aa50,0x1b255,0x06d20,0x0ada0,
    0x14b63,0x09370,0x049f8,0x04970,0x064b0,0x168a6,0x0ea50,0x06aa0,0x1a6c4,0x0aae0,
    0x0a570,0x05260,0x0f263,0x0d950,0x05b57,0x056a0,0x096d0,0x04dd5,0x04ad0,0x0a4d0]
  const y=d.getFullYear();const m=d.getMonth()+1;const day=d.getDate()
  // Simplified: use known lunar new year dates for 2025-2027
  const lunarNY={2025:[1,29],2026:[2,17],2027:[2,6]}
  const ny=lunarNY[y]||lunarNY[2026];let days=(d-new Date(y,ny[0]-1,ny[1]))/86400000
  if(days<0){const pny=lunarNY[y-1]||[2,10];days=(d-new Date(y-1,pny[0]-1,pny[1]))/86400000}
  days=Math.floor(days)
  const lunarMonths=['正月','二月','三月','四月','五月','六月','七月','八月','九月','十月','冬月','腊月']
  const lunarDays=['初一','初二','初三','初四','初五','初六','初七','初八','初九','初十','十一','十二','十三','十四','十五','十六','十七','十八','十九','二十','廿一','廿二','廿三','廿四','廿五','廿六','廿七','廿八','廿九','三十','卅一']
  const mi=Math.floor(days/30);const di=days%30
  return `${y}年${lunarMonths[mi%12]}${lunarDays[Math.min(di,30)]}`
}
function getTimeDisplay(){
  const n=new Date()
  const h=n.getHours();const m=n.getMinutes()
  const shichen=['子时','丑时','寅时','卯时','辰时','巳时','午时','未时','申时','酉时','戌时','亥时']
  const sc=shichen[Math.floor(((h+1)%24)/2)]
  const ke=Math.floor(m/15)+1
  return `${getLunarDate(n)} ${sc}${ke}刻`
}
function getUptimeDisplay(){
  const elapsed=Math.floor((Date.now()-serverStartTime)/1000)
  const d=Math.floor(elapsed/86400);const h=Math.floor((elapsed%86400)/3600)
  const m=Math.floor((elapsed%3600)/60);const s=elapsed%60
  return `${d}天${h}时${m}分${s}秒`
}
const timeDisplay=ref(getTimeDisplay())
const uptimeDisplay=ref(getUptimeDisplay())
function handleQuit(){localStorage.clear();location.href='/#/'}
onMounted(()=>setInterval(()=>{timeDisplay.value=getTimeDisplay();uptimeDisplay.value=getUptimeDisplay()},1000))
</script>

<style lang="scss" scoped>
.gh-root{height:100vh;display:flex;flex-direction:column;background:#0d0d1a;color:#e8e0d0;font-family:'Noto Sans SC','PingFang SC','Microsoft YaHei',sans-serif;overflow:hidden}.gh-root.light-mode{background:#fff;color:#1a1a1a}
.gold-divider{flex-shrink:0;height:2px;background:linear-gradient(90deg,#b8860b,#d4a843,#f0d878,#d4a843,#b8860b);position:relative;overflow:hidden}.gold-divider__light{position:absolute;top:0;left:-60%;width:60%;height:100%;background:linear-gradient(90deg,transparent,rgba(255,255,255,.15),#ff6b6b,#ffd93d,#6bcb77,#4d96ff,#9b59b6,#ff6b6b,rgba(255,255,255,.15),transparent);animation:rainbow-run 3s linear infinite}@keyframes rainbow-run{to{left:100%}}
.top-bar{flex-shrink:0;background:#000}.light-mode .top-bar{background:#fff;border-bottom:1px solid rgba(0,0,0,.06)}.top-bar-inner{position:relative;padding:0 28px;height:56px;display:flex;align-items:center;gap:10px}.brand-logo{font-size:40px;line-height:1;color:#fff;animation:logo-spin 8s linear infinite}@keyframes logo-spin{to{transform:rotate(360deg)}}.brand-name{font-size:30px;letter-spacing:6px;font-weight:900;letter-spacing:4px;color:#fff}.main-nav{position:absolute;left:50%;transform:translateX(-50%);display:flex;align-items:center;gap:2px;flex-wrap:nowrap}.nav-item{padding:4px 7px;cursor:pointer;border-radius:4px;transition:all .2s;white-space:nowrap}.nav-item:hover{background:rgba(255,255,255,.08)}.nav-label{font-size:14px;font-weight:600;color:#fff;letter-spacing:1px}.light-mode .nav-label{color:rgba(0,0,0,.7)}.top-bar-spacer{flex:1}.light-mode .nav-item:hover{background:rgba(0,0,0,.05)}.top-bar-right{display:flex;align-items:center;gap:14px}.player-stats{display:flex;flex-direction:column;align-items:flex-end;gap:2px;line-height:1.2}.online-badge{display:flex;align-items:center;gap:5px;font-size:14px;font-weight:500;color:#fff}.online-dot{width:6px;height:6px;border-radius:50%;background:#4caf50;animation:dot-pulse 2s ease-in-out infinite}@keyframes dot-pulse{0%,to{opacity:.4}50%{opacity:1}}.registered-badge{font-size:14px;font-weight:500;color:#fff}.light-mode .brand-logo,.light-mode .brand-name,.light-mode .online-badge{color:#000}.light-mode .registered-badge{color:rgba(0,0,0,.4)}.theme-toggle{width:52px;height:52px;border:2px solid #d4a843;border-radius:50%;background:#000;color:#fff;font-size:24px;cursor:pointer;display:flex;align-items:center;justify-content:center;box-shadow:0 0 10px rgba(212,168,67,.25);transition:all .3s}.light-mode .theme-toggle{background:#000;color:#fff}.theme-toggle:hover{border-color:#fff;box-shadow:0 0 20px rgba(212,168,67,.4)}
.notice-bar{flex-shrink:0;display:flex;align-items:center;gap:8px;padding:6px 20px;background:rgba(0,0,0,.2);overflow:hidden}.gh-main{flex:1;display:flex;overflow:hidden;gap:0}.gh-panel{display:flex;flex-direction:column;border-right:1px solid rgba(212,168,67,.1);background:#1a1a2e}.light-mode .gh-panel{background:#ececee}.gh-log{width:41%;flex-shrink:0}.gh-chat{width:41%;flex-shrink:0}.gh-sidebar{width:18%;min-width:210px;border-right:none;flex-shrink:0;display:flex;flex-direction:column;gap:4px;padding:6px 4px;overflow-y:auto;border-right:2px solid rgba(212,168,67,.25);background:rgba(0,0,0,.15)}.light-mode .gh-sidebar{background:rgba(0,0,0,.02);border-right-color:rgba(0,0,0,.1)}.side-card{display:flex;flex-direction:column;justify-content:center;border:1px solid rgba(212,168,67,.15);border-radius:8px;padding:6px;overflow:hidden}
.light-mode .side-card{background:transparent;border-color:rgba(0,0,0,.08)}.side-card h4{margin:0 0 10px;font-size:13px;color:#d4a843;text-align:center}.panel-header{display:flex;align-items:center;justify-content:space-between;padding:8px 12px;border-bottom:1px solid rgba(212,168,67,.1);flex-shrink:0}.panel-header h4{margin:0;font-size:15px;color:#d4a843}.panel-actions{display:flex;gap:4px;flex-wrap:wrap}.pa-btn{padding:2px 8px;border:1px solid rgba(212,168,67,.2);border-radius:4px;background:transparent;color:rgba(255,255,255,.5);font-size:11px;cursor:pointer;transition:all .2s;font-family:inherit}.pa-btn:hover{color:#d4a843;border-color:#d4a843}.pa-btn.active{background:rgba(212,168,67,.15);color:#d4a843;border-color:#d4a843}.light-mode .pa-btn{color:rgba(0,0,0,.4);border-color:rgba(0,0,0,.1)}.panel-body{flex:1;overflow-y:auto;padding:8px 10px;font-size:14px;line-height:1.8}.log-entry{padding:2px 0;border-bottom:1px solid rgba(255,255,255,.03);color:rgba(255,255,255,.6)}.light-mode .log-entry{color:rgba(0,0,0,.5);border-bottom-color:rgba(0,0,0,.04)}.log-entry.combat{color:#ff6b6b}.log-entry.cult{color:#6bcb77}.log-entry.item{color:#4d96ff}.log-entry.social{color:#ffd93d}.log-entry.explore{color:#64ffda}.log-entry.quest{color:#b388ff}.log-entry.system{color:#fff;font-weight:700}.light-mode .log-entry.system{color:#000}.log-entry.important{color:#f0d878;font-weight:700;font-size:15px}.light-mode .log-entry.important{color:#b8860b}.log-empty{text-align:center;color:rgba(255,255,255,.2);padding:20px}.chat-body{display:flex;flex-direction:column;gap:2px}.chat-msg{font-size:14px;line-height:1.6}.chat-time{color:rgba(255,255,255,.2);margin-right:6px;font-size:11px}.chat-name{font-weight:700;margin-right:4px}.chat-text{color:rgba(255,255,255,.75)}.chat-online{font-size:11px;color:rgba(255,255,255,.3)}.chat-input-bar{display:flex;gap:6px;padding:8px 10px;border-top:1px solid rgba(212,168,67,.1);flex-shrink:0}.chat-input{flex:1;padding:10px 14px;border:1px solid rgba(212,168,67,.15);border-radius:6px;background:rgba(255,255,255,.04);color:#fff;font-size:14px;outline:none;font-family:inherit}.light-mode .chat-input{background:rgba(0,0,0,.03);color:#000;border-color:rgba(0,0,0,.1)}.chat-input::placeholder{color:rgba(255,255,255,.2)}.chat-send{padding:8px 18px;border:1px solid #d4a843;border-radius:6px;background:transparent;color:#d4a843;font-size:14px;cursor:pointer;font-family:inherit;transition:all .2s}.chat-send:hover{background:#d4a843;color:#fff}.map-body{display:flex;flex-direction:column;gap:6px}.map-btn{width:100%;padding:10px 12px;border:1px solid rgba(212,168,67,.2);border-radius:8px;background:rgba(212,168,67,.05);color:rgba(255,255,255,.7);font-size:13px;font-weight:500;cursor:pointer;transition:all .2s;font-family:inherit;text-align:left}.map-btn:hover{background:rgba(212,168,67,.15);color:#d4a843;border-color:#d4a843;transform:translateX(4px)}.light-mode .map-btn{background:rgba(0,0,0,.02);color:rgba(0,0,0,.5);border-color:rgba(0,0,0,.08)}.light-mode .map-btn:hover{background:rgba(184,134,11,.08);color:#b8860b}.notice-icon{flex-shrink:0;font-size:14px}.notice-text{font-size:12px;color:rgba(255,255,255,.5);white-space:nowrap;animation:notice-scroll 20s linear infinite}.light-mode .notice-bar{background:rgba(0,0,0,.03)}.light-mode .notice-text{color:rgba(0,0,0,.4)}@keyframes notice-scroll{0%{transform:translateX(100%)}100%{transform:translateX(-100%)}}
.side-profile{flex:0 0 auto;padding:0 8px 6px;overflow:visible;justify-content:flex-start}.profile-top{display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:4px}.profile-top-left{display:flex;align-items:center;gap:8px;min-width:0;flex:1}.side-avatar{width:40px;height:40px;border-radius:50%;background:linear-gradient(135deg,#d4a843,#b8860b);flex-shrink:0;display:flex;align-items:center;justify-content:center;font-size:20px;font-weight:700;color:#fff}.side-name{font-size:19px;font-weight:800;color:#fff;line-height:1.1}.light-mode .side-name{color:#000}.gender-tag{margin-left:2px;font-size:17px}.profile-tags{display:flex;gap:5px;font-size:10px;flex-wrap:wrap}.side-realm{color:#d4a843;font-weight:700}.side-spirit{color:#fff;font-weight:600;font-size:10px}.light-mode .side-spirit{color:#000}.profile-loc{font-size:10px;color:#6bcb77;display:flex;align-items:center;gap:2px}.pl-icon{font-size:13px}.profile-power-badge{text-align:center;min-width:50px;margin-top:-3px;cursor:help}.pp-val{font-size:24px;font-weight:900;color:#e53935;line-height:1}.profile-bars{margin:4px 0 3px}.pbar-row{display:flex;align-items:center;gap:6px;margin-bottom:4px;color:#fff;font-weight:600}.light-mode .pbar-row{color:#000}.pbar-row>span:first-child{font-size:14px;width:18px;text-align:center;flex-shrink:0}.pbar-val{width:62px;text-align:right;flex-shrink:0;font-size:11px;font-weight:700}.pbar-track{flex:1;height:8px;background:rgba(255,255,255,.08);border-radius:4px;overflow:hidden}.pbar-fill{height:100%;border-radius:4px;transition:width .6s}.pbar-fill.hp{background:linear-gradient(90deg,#e53935,#ef5350);box-shadow:0 0 4px rgba(229,57,53,.3)}.pbar-fill.mp{background:linear-gradient(90deg,#42a5f5,#64b5f6);box-shadow:0 0 4px rgba(66,165,245,.3)}.profile-attrs{display:grid;grid-template-columns:1fr 1fr;gap:0 6px}.pa-row{display:flex;justify-content:space-between;padding:1px 0;font-size:11px;color:#bbb;border-bottom:1px solid rgba(255,255,255,.04)}.light-mode .pa-row{color:rgba(0,0,0,.6);border-bottom-color:rgba(0,0,0,.06)}.pa-row span:first-child{color:#999}.light-mode .pa-row span:first-child{color:rgba(0,0,0,.4)}.pa-row span:last-child{font-weight:700;color:#fff}.light-mode .pa-row span:last-child{color:#000}
@keyframes gold-flow{0%{background-position:100% 0}100%{background-position:-100% 0}}
.side-cultivation{flex:0 0 auto;text-align:center;display:flex;flex-direction:column;align-items:center;gap:2px;padding:6px}.side-equip{flex:1}.equip-slots{display:grid;grid-template-columns:1fr 1fr;gap:4px}.eq-slot{font-weight:700;display:flex;flex-direction:row;align-items:center;gap:4px;justify-content:center;text-align:center;padding:8px;border:1.5px solid rgba(212,168,67,.25);border-radius:6px;font-size:22px;color:#fff}.eq-slot span{font-size:14px;font-weight:800;color:#fff}.cult-yy{width:100px;height:100px;border-radius:50%;flex-shrink:0;border:2px solid #d4a843;padding:3px;box-shadow:0 0 8px rgba(212,168,67,.3);position:relative}
.cult-info{width:100%}.cult-bar-wrap{display:flex;align-items:center;gap:6px;margin-bottom:4px}
.cult-bar::after{content:"";position:absolute;inset:0;background:linear-gradient(90deg,transparent,rgba(255,255,255,.15),transparent);animation:bar-shine 2s ease-in-out infinite}@keyframes bar-shine{0%{transform:translateX(-100%)}100%{transform:translateX(200%)}}.cult-yy.male{background:linear-gradient(to right,#fff 50%,#000 50%)}.cult-yy.male::before{content:'';position:absolute;width:50%;height:50%;top:0;left:25%;border-radius:50%;background:#fff;background-image:radial-gradient(circle at 50% 62%,#000 22%,transparent 22%)}.cult-yy.male::after{content:'';position:absolute;width:50%;height:50%;top:50%;left:25%;border-radius:50%;background:#000;background-image:radial-gradient(circle at 50% 38%,#fff 22%,transparent 22%)}.cult-yy.female{background:linear-gradient(to right,#ff6b6b 50%,#ffd700 50%)}.cult-yy.female::before{content:'';position:absolute;width:50%;height:50%;top:0;left:25%;border-radius:50%;background:#ff6b6b;background-image:radial-gradient(circle at 50% 62%,#ffd700 22%,transparent 22%)}.cult-yy.female::after{content:'';position:absolute;width:50%;height:50%;top:50%;left:25%;border-radius:50%;background:#ffd700;background-image:radial-gradient(circle at 50% 38%,#ff6b6b 22%,transparent 22%)}.cult-yy-wrap.meditating .cult-yy{animation:spin-yy 8s linear infinite}@keyframes rainbow-glow{0%{box-shadow:0 0 0 12px #d4a843,0 0 40px #d4a843,0 0 60px rgba(212,168,67,.4)}25%{box-shadow:0 0 0 12px #ff6b6b,0 0 40px #ff6b6b,0 0 60px rgba(255,107,107,.4)}50%{box-shadow:0 0 0 12px #6bcb77,0 0 40px #6bcb77,0 0 60px rgba(107,203,119,.4)}75%{box-shadow:0 0 0 12px #4d96ff,0 0 40px #4d96ff,0 0 60px rgba(77,150,255,.4)}100%{box-shadow:0 0 0 12px #d4a843,0 0 40px #d4a843,0 0 60px rgba(212,168,67,.4)}}.cult-info{width:100%}.cult-bar-wrap{display:flex;align-items:center;gap:6px;margin-bottom:4px}.cult-bar-label{font-size:11px;color:#d4a843;font-weight:600;flex-shrink:0}.cult-bar{height:7px;background:rgba(255,255,255,.06);border-radius:3px;overflow:hidden;margin-bottom:3px;position:relative}.cult-fill{height:100%;background:linear-gradient(90deg,#b8860b,#d4a843,#f0d878,#d4a843,#b8860b);background-size:200% 100%;animation:gold-flow 2s linear infinite,fill-pulse 2s ease-in-out infinite;border-radius:3px;transition:width .8s ease;box-shadow:0 0 8px rgba(212,168,67,.4)}@keyframes fill-pulse{0%,100%{box-shadow:0 0 6px rgba(212,168,67,.3)}50%{box-shadow:0 0 14px rgba(212,168,67,.6),0 0 20px rgba(240,216,120,.3)}}.cult-stats{display:flex;justify-content:space-between;font-size:11px;color:rgba(255,255,255,.7);margin-bottom:4px;font-weight:600}.light-mode .cult-stats{color:rgba(0,0,0,.6)}.cult-btn.danger{border-color:#e53935;color:#e53935}.cult-btn.danger:hover{box-shadow:0 4px 14px rgba(229,57,53,.3);background:rgba(229,57,53,.1)}.cult-realm-info{display:flex;gap:4px;justify-content:center;flex-wrap:wrap;margin:3px 0}.realm-tag{padding:1px 8px;border-radius:8px;background:rgba(212,168,67,.12);color:#d4a843;font-size:11px;font-weight:700}.root-tag{padding:1px 8px;border-radius:8px;background:rgba(255,255,255,.06);font-size:10px;font-weight:600}.light-mode .root-tag{background:rgba(0,0,0,.04)}.cult-btns{display:flex;gap:5px;width:100%;justify-content:center;flex-wrap:wrap}.cult-btn{padding:6px 12px;border:1.5px solid #d4a843;border-radius:8px;background:transparent;color:#d4a843;font-size:11px;font-weight:700;cursor:pointer;transition:all .25s ease;font-family:inherit}.cult-btn:hover{transform:translateY(-1px);box-shadow:0 4px 14px rgba(212,168,67,.3);background:rgba(212,168,67,.1)}.cult-btn:active{transform:translateY(0)}.train-btn{border-color:#64ffda;color:#64ffda}.train-btn:hover{box-shadow:0 4px 14px rgba(100,255,218,.3);background:rgba(100,255,218,.1)}.train-panel{width:100%;margin-top:6px;padding:6px 8px;background:rgba(100,255,218,.05);border:1px solid rgba(100,255,218,.15);border-radius:6px;font-size:10px}.train-info{display:flex;justify-content:space-between;align-items:center;color:rgba(255,255,255,.5);margin-bottom:3px}.train-switch{cursor:pointer;display:flex;align-items:center;gap:4px;color:#64ffda;margin-bottom:3px;font-size:10px}.train-switch input{accent-color:#64ffda;cursor:pointer}.mult-btn{padding:2px 6px;border:1px solid rgba(100,255,218,.2);border-radius:3px;background:transparent;color:rgba(100,255,218,.5);font-size:10px;cursor:pointer;font-family:inherit;transition:all .2s}.mult-btn:hover{border-color:#64ffda;color:#64ffda}.mult-btn.on{background:rgba(100,255,218,.15);border-color:#64ffda;color:#64ffda;font-weight:700}.mult-btn:disabled{opacity:.3;pointer-events:none}.manual-train-btn{padding:2px 8px;border:1px solid #ffd700;border-radius:3px;background:rgba(255,215,0,.1);color:#ffd700;font-size:10px;cursor:pointer;font-family:inherit;transition:all .2s;margin-left:auto}.manual-train-btn:hover{background:rgba(255,215,0,.2)}.manual-train-btn:disabled{opacity:.3;pointer-events:none}.dead-warn{text-align:center;color:#e53935;font-weight:700;font-size:11px;padding:4px 0}.death-modal{background:#1a1a2e;border:2px solid #e53935;border-radius:20px;width:420px;max-width:92vw;max-height:85vh;overflow-y:auto;padding:24px;box-shadow:0 0 40px rgba(229,57,53,.3);text-align:center;animation:modal-in .25s ease}.dm-header{font-size:28px;font-weight:900;color:#e53935;margin-bottom:12px}.dm-timer{font-size:18px;color:#ffd700;margin-bottom:8px}.dm-timer b{font-size:28px}.dm-info{font-size:12px;color:rgba(255,255,255,.5);margin-bottom:12px}.dm-status{display:flex;gap:12px;justify-content:center;margin-bottom:12px}.dm-status span{background:rgba(255,255,255,.06);padding:4px 12px;border-radius:6px;font-size:13px;color:#fff}.dm-battle{margin-top:12px;padding:8px;background:rgba(229,57,53,.06);border:1px solid rgba(229,57,53,.15);border-radius:8px;font-size:10px;text-align:left;max-height:180px;overflow-y:auto}.dm-battle-title{color:#e53935;font-weight:700;font-size:12px;margin-bottom:4px}.dm-note{margin-top:12px;font-size:11px;color:rgba(255,255,255,.3)}
.encounter-modal{background:#1a1a2e;border:2px solid #ffd700;border-radius:20px;width:400px;max-width:92vw;padding:28px;box-shadow:0 0 60px rgba(255,215,0,.3);text-align:center;animation:modal-in .25s ease}.em-icon{font-size:60px;margin-bottom:8px;animation:em-bounce .5s ease-in-out infinite alternate}@keyframes em-bounce{from{transform:translateY(0)}to{transform:translateY(-6px)}}.em-title{font-size:22px;font-weight:900;color:#ffd700;margin-bottom:8px}.em-desc{font-size:13px;color:rgba(255,255,255,.6);margin-bottom:12px;line-height:1.6}.em-change{font-size:18px;font-weight:700;margin:12px 0}.em-old{color:rgba(255,255,255,.4);text-decoration:line-through}.em-new{color:#6bcb77;font-size:22px;animation:em-glow 1s ease-in-out infinite alternate}@keyframes em-glow{from{text-shadow:0 0 10px rgba(107,203,119,.5)}to{text-shadow:0 0 20px rgba(107,203,119,.8),0 0 30px rgba(107,203,119,.4)}}.em-note{font-size:11px;color:rgba(255,255,255,.4)}.combat-report{margin-top:6px;padding:6px;background:rgba(255,107,107,.05);border:1px solid rgba(255,107,107,.2);border-radius:4px;font-size:10px;max-height:150px;overflow-y:auto}.cr-header{color:#ff6b6b;font-weight:700;margin-bottom:4px;font-size:11px}.cr-round{padding:2px 0;border-bottom:1px solid rgba(255,107,107,.1);color:rgba(255,255,255,.7);line-height:1.5}.cr-dmg{color:#ff6b6b;margin:0 4px}.cr-special{color:#ffd700;font-weight:700}.cr-hp{color:rgba(255,255,255,.5);font-size:9px}
.energy-canvas{position:absolute;inset:-50px;width:calc(100% + 100px);height:calc(100% + 100px);pointer-events:none;z-index:1}
.cult-yy-wrap{position:relative;width:100px;height:100px;flex-shrink:0}
.cult-tooltip{position:fixed;transform:translateY(-50%);z-index:5000;min-width:220px;background:rgba(10,10,30,.96);border:1px solid rgba(212,168,67,.4);border-radius:12px;padding:14px 16px;box-shadow:0 8px 32px rgba(0,0,0,.6);pointer-events:none}.light-mode .cult-tooltip{background:rgba(255,255,255,.98);border-color:rgba(184,134,11,.3);box-shadow:0 8px 32px rgba(0,0,0,.15)}.ct-title{font-size:14px;font-weight:700;color:#d4a843;margin-bottom:10px;text-align:center;letter-spacing:2px}.ct-rows{display:flex;flex-direction:column;gap:6px}.ct-row{display:flex;justify-content:space-between;font-size:12px;color:rgba(255,255,255,.7)}.light-mode .ct-row{color:rgba(0,0,0,.6)}.ct-row span:last-child{font-weight:600;color:#fff}.light-mode .ct-row span:last-child{color:#000}.ct-row.ct-sub{border-top:1px dashed rgba(212,168,67,.15);padding-top:6px;color:rgba(255,255,255,.5)}.ct-row.ct-sub span:last-child{color:#d4a843}.ct-row.ct-bonus{color:#6bcb77}.ct-row.ct-bonus span:last-child{color:#6bcb77}.ct-row.ct-none{color:rgba(255,255,255,.25)}.ct-row.ct-none span:last-child{color:rgba(255,255,255,.25)}.ct-divider{height:1px;background:linear-gradient(90deg,transparent,rgba(212,168,67,.3),transparent);margin:2px 0}.ct-row.ct-total{font-size:14px;font-weight:700;color:#d4a843}.ct-row.ct-total span:last-child{color:#f0d878;font-size:15px}.ct-formula{margin-top:8px;font-size:10px;color:rgba(255,255,255,.25);text-align:center}
.tooltip-enter-active{transition:opacity .2s}.tooltip-leave-active{transition:opacity .15s}.tooltip-enter-from,.tooltip-leave-to{opacity:0}
.gh-center{display:none;flex-direction:column;align-items:center;justify-content:center;gap:20px;padding:20px;text-align:center;overflow:hidden}.yin-yang{--yy-size:200px;width:var(--yy-size);height:var(--yy-size);border-radius:50%;margin:0 auto;background:linear-gradient(to right,#fff 50%,#000 50%);position:relative;animation:spin-yy 8s linear infinite;cursor:pointer;transition:transform .4s cubic-bezier(.175,.885,.32,1.275);box-shadow:0 0 30px rgba(255,255,255,.15),0 0 60px rgba(0,0,0,.4)}.yin-yang:active{transform:scale(1.15)}.yin-yang::before,.yin-yang::after{content:'';position:absolute;border-radius:50%}.yin-yang::before{width:50%;height:50%;top:0;left:25%;background:#fff;background-image:radial-gradient(circle at 50% 62%,#000 24%,transparent 24%)}.yin-yang::after{width:50%;height:50%;top:50%;left:25%;background:#000;background-image:radial-gradient(circle at 50% 38%,#fff 24%,transparent 24%)}@keyframes spin-yy{to{transform:rotate(360deg)}}.game-title{margin:0;font-size:72px;font-weight:900;letter-spacing:12px;line-height:1.1;background:linear-gradient(135deg,#d4a843,#f0d878 30%,#d4a843 60%,#b8942e);background-size:200% 200%;-webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text;animation:gold-shimmer 4s ease-in-out infinite}@keyframes gold-shimmer{0%,to{background-position:0 50%}50%{background-position:100% 50%}}.game-subtitle{margin:4px 0 0;font-size:22px;font-weight:500;color:rgba(232,224,208,.6);letter-spacing:10px}.light-mode .game-subtitle{color:rgba(0,0,0,.55)}.realm-progress{display:flex;align-items:flex-start;position:relative;width:100%;max-width:440px;padding:0 8px}.realm-track-bg{position:absolute;top:14px;left:16px;right:16px;height:2px;background:rgba(232,224,208,.15);border-radius:1px}.realm-node{display:flex;flex-direction:column;align-items:center;gap:6px;flex:1;cursor:pointer;opacity:0;transform:translateX(-20px);animation:realm-in .5s ease-out forwards}@keyframes realm-in{to{opacity:1;transform:translateX(0)}}.realm-dot{width:24px;height:24px;border-radius:50%;border:2px solid rgba(232,224,208,.2);background:#0d0d1a;display:flex;align-items:center;justify-content:center}.realm-node.completed .realm-dot{border-color:#d4a843;background:#d4a843}.realm-node.active .realm-dot{border-color:#d4a843;box-shadow:0 0 0 3px rgba(212,168,67,.25)}.realm-check{color:#fff;font-size:12px;font-weight:700}.realm-pulse{width:8px;height:8px;border-radius:50%;background:#d4a843;animation:realm-pulse 2s ease-in-out infinite}@keyframes realm-pulse{0%,to{opacity:.4;transform:scale(.8)}50%{opacity:1;transform:scale(1.2)}}.realm-label{font-size:13px;color:rgba(232,224,208,.35)}.realm-node.completed .realm-label,.realm-node.active .realm-label{color:#d4a843}.light-mode .realm-dot{background:#fff!important}.cultivation-quote{margin:0;font-size:14px;color:rgba(212,168,67,.5);letter-spacing:4px;font-style:italic;animation:quote-fade 5s ease-in-out infinite}@keyframes quote-fade{0%,to{opacity:.4}50%{opacity:1}}.bottom-bar{flex-shrink:0;background:rgba(13,13,26,.95);padding:8px 20px;display:flex;justify-content:space-between;align-items:center;border-top:1px solid rgba(212,168,67,.12)}.light-mode .bottom-bar{background:rgba(255,255,255,.95)}.footer-left{display:flex;align-items:center;gap:10px;font-size:12px;color:rgba(232,224,208,.45)}.light-mode .footer-left{color:#000}.footer-divider{color:rgba(212,168,67,.3)}.footer-uptime{font-size:13px}.footer-right{display:flex;align-items:center}.footer-quit{padding:4px 12px;border:1px solid rgba(229,57,53,.3);border-radius:4px;background:transparent;color:rgba(229,57,53,.6);font-size:12px;cursor:pointer;transition:all .2s;font-family:inherit}.footer-quit:hover{border-color:#e53935;color:#e53935;background:rgba(229,57,53,.08)}@media(max-width:768px){.yin-yang{--yy-size:120px!important}.game-title{font-size:40px!important}.main-nav{display:none}.brand-name{display:none}}
</style>

<style lang="scss">
.modal-overlay{position:fixed;inset:0;z-index:3000;display:flex;align-items:center;justify-content:center;background:rgba(0,0,0,.6)}.modal-card{background:#0d0d1a;border:1px solid rgba(212,168,67,.2);border-radius:16px;width:680px;max-width:92vw;max-height:85vh;overflow:hidden;display:flex;flex-direction:column;box-shadow:0 24px 80px rgba(0,0,0,.6);animation:modal-in .25s ease}.light-mode .modal-card{background:#fff;border-color:rgba(184,134,11,.15);box-shadow:0 24px 80px rgba(0,0,0,.1)}@keyframes modal-in{from{opacity:0;transform:translateY(12px)}to{opacity:1;transform:translateY(0)}}.modal-header{text-align:center;position:relative;padding:0;background:#000;border-bottom:2px solid;border-image:linear-gradient(90deg,#b8860b,#d4a843,#f0d878,#d4a843,#b8860b) 1;flex-shrink:0}.light-mode .modal-header{background:#fff;border-image:linear-gradient(90deg,#b8860b,#d4a843,#b8860b) 1}.modal-header h2{margin:0;padding:14px 24px;font-size:18px;color:#d4a843;letter-spacing:6px;font-weight:800}.light-mode .modal-header h2{color:#b8860b}.modal-close{position:absolute;top:50%;right:14px;transform:translateY(-50%);background:none;border:none;color:rgba(255,255,255,.25);font-size:20px;cursor:pointer;width:32px;height:32px;display:flex;align-items:center;justify-content:center;border-radius:50%;transition:all .2s}.modal-close:hover{color:#e53935;background:rgba(229,57,53,.1)}html.light-mode .modal-close{color:rgba(0,0,0,.2)}html.light-mode .modal-close:hover{color:#e53935}.modal-tabs{display:flex;justify-content:center;gap:6px;padding:10px 16px;flex-wrap:wrap;border-bottom:1px solid rgba(212,168,67,.08)}.modal-tab{padding:6px 14px;border:1.5px solid rgba(212,168,67,.15);border-radius:6px;background:transparent;color:rgba(255,255,255,.5);font-size:12px;cursor:pointer;transition:all .2s;font-family:inherit;font-weight:600}.modal-tab:hover{color:#d4a843;border-color:#d4a843;background:rgba(212,168,67,.05);transform:translateY(-1px)}.modal-tab.active{background:linear-gradient(135deg,rgba(212,168,67,.12),transparent);color:#d4a843;border-color:#d4a843;font-weight:700}html.light-mode .modal-tab{color:rgba(0,0,0,.35);border-color:rgba(184,134,11,.1)}html.light-mode .modal-tab:hover{color:#b8860b;border-color:#b8860b}html.light-mode .modal-tab.active{background:rgba(184,134,11,.08);color:#b8860b;border-color:#b8860b}.modal-body{flex:1;overflow-y:auto;padding:24px 28px}.modal-placeholder{text-align:center;font-size:44px;margin:12px 0 8px;opacity:.5}.modal-desc{text-align:center;font-size:13px;color:rgba(255,255,255,.35);line-height:2}.light-mode .modal-desc{color:rgba(0,0,0,.3)}.wiki-modal{background:#0d0d1a;border:1px solid rgba(212,168,67,.2);border-radius:16px;width:680px;max-width:92vw;max-height:85vh;overflow:hidden;display:flex;flex-direction:column;box-shadow:0 24px 80px rgba(0,0,0,.6)}.light-mode .wiki-modal{background:#fff;border-color:rgba(184,134,11,.15)}.wiki-header{text-align:center;position:relative;padding:0;background:#000;border-bottom:2px solid;border-image:linear-gradient(90deg,#b8860b,#d4a843,#f0d878,#d4a843,#b8860b) 1;flex-shrink:0}.light-mode .wiki-header{background:#fff;border-image:linear-gradient(90deg,#b8860b,#d4a843,#b8860b) 1}.wiki-header h2{margin:0;padding:14px 24px;font-size:18px;color:#d4a843;letter-spacing:6px;font-weight:800}.light-mode .wiki-header h2{color:#b8860b}.wiki-tabs{display:flex;justify-content:center;gap:6px;padding:8px 16px;border-bottom:1px solid rgba(212,168,67,.08)}.wiki-body{flex:1;overflow-y:auto;padding:16px 24px 24px}.light-mode .eq-slot,.light-mode .eq-slot span{color:#000!important}
.map-modal{background:#1a1a2e;border:1px solid rgba(212,168,67,.15);border-radius:20px;width:780px;max-width:95vw;max-height:88vh;overflow-y:auto;box-shadow:0 24px 64px rgba(0,0,0,.5);animation:modal-in .25s ease}html.light-mode .map-modal{background:#fff;border-color:rgba(0,0,0,.08)}.map-body-wrap{padding:16px 24px 28px}.map-legend{display:flex;gap:20px;align-items:center;flex-wrap:wrap;padding:8px 16px;margin-bottom:12px;background:rgba(255,255,255,.03);border-radius:8px;font-size:12px;color:rgba(255,255,255,.4)}.ml-item{display:flex;align-items:center;gap:4px}.ml-dot{width:10px;height:10px;border-radius:50%;display:inline-block}.ml-dot.unlocked{background:#4caf50;box-shadow:0 0 6px rgba(76,175,80,.4)}.ml-dot.locked{background:#666}.map-region{background:rgba(255,255,255,.03);border:1px solid rgba(212,168,67,.1);border-radius:12px;padding:12px 16px;margin-bottom:10px;transition:all .2s}.map-region.locked{opacity:.45;filter:grayscale(.6)}.mr-header{display:flex;align-items:center;gap:8px;margin-bottom:8px}.mr-icon{font-size:24px}.mr-name{font-size:17px;font-weight:800;color:#d4a843}.mr-realm{font-size:12px;color:rgba(255,255,255,.3);margin-left:auto}.mr-lock{font-size:11px;color:#e53935;font-weight:600}.mr-locations{display:grid;grid-template-columns:repeat(auto-fill,minmax(160px,1fr));gap:6px}.mr-loc{cursor:pointer;padding:8px 10px;background:rgba(255,255,255,.04);border:1px solid rgba(255,255,255,.06);border-radius:8px;transition:all .2s;position:relative}.mr-loc:hover{background:rgba(212,168,67,.08);border-color:rgba(212,168,67,.2)}.mr-loc.locked{opacity:.4;pointer-events:none}.mrl-icon{font-size:18px;margin-right:4px}.mrl-name{font-size:13px;font-weight:700;color:#fff}.mrl-type{display:block;font-size:10px;color:rgba(255,255,255,.35);margin-top:2px}.mrl-lock{position:absolute;top:6px;right:8px;font-size:12px}.mrl-detail{margin-top:8px;padding:8px;background:rgba(0,0,0,.3);border-radius:6px;font-size:11px;color:rgba(255,255,255,.6);line-height:1.6}.mrl-detail p{margin:0 0 4px;color:rgba(255,255,255,.8)}.mrl-info{display:flex;flex-wrap:wrap;gap:8px;margin-bottom:6px}.mrl-info span{background:rgba(212,168,67,.1);padding:2px 8px;border-radius:4px;font-size:10px;color:#d4a843}.map-enter-btn{width:100%;padding:6px;border:1px solid #4caf50;border-radius:6px;background:rgba(76,175,80,.1);color:#4caf50;font-size:12px;font-weight:700;cursor:pointer;transition:all .2s;font-family:inherit;margin-top:4px}.map-enter-btn:hover{background:#4caf50;color:#fff}
.map-current-badge{display:block;text-align:center;padding:6px;color:#4caf50;font-size:12px;font-weight:700;margin-top:4px}
.map-fight-btn{width:100%;padding:6px;border:1px solid #e53935;border-radius:6px;background:rgba(229,57,53,.1);color:#e53935;font-size:12px;font-weight:700;cursor:pointer;transition:all .2s;font-family:inherit;margin-top:4px}.map-fight-btn:hover{background:#e53935;color:#fff}
.mr-loc.current{border-color:rgba(76,175,80,.5);background:rgba(76,175,80,.12);box-shadow:0 0 8px rgba(76,175,80,.15)}
html.light-mode .map-modal{background:#fafafa}html.light-mode .map-region{background:rgba(0,0,0,.02);border-color:rgba(0,0,0,.06)}html.light-mode .mr-loc{background:rgba(0,0,0,.03)}html.light-mode .mr-name{color:#b8860b}html.light-mode .mrl-name{color:#000}
.wiki-btn:hover{background:rgba(212,168,67,.2);box-shadow:0 0 12px rgba(212,168,67,.3)}.wiki-modal{background:#1a1a2e;border:1px solid rgba(212,168,67,.15);border-radius:20px;width:960px;max-width:95vw;max-height:90vh;overflow-y:auto;box-shadow:0 24px 64px rgba(0,0,0,.5);animation:modal-in .25s ease}html.light-mode .wiki-modal{background:#fff;color:#1a1a1a;border-color:rgba(0,0,0,.08)}.wiki-header{text-align:center;position:relative;padding:24px 24px 0}.wiki-header h2{margin:0;font-size:24px;color:#d4a843;letter-spacing:6px}.wiki-tabs{display:flex;justify-content:center;gap:8px;padding:16px 24px 8px;flex-wrap:wrap;border-bottom:1px solid rgba(212,168,67,.1);margin-bottom:12px}.wiki-body{padding:0 32px 32px;color:rgba(255,255,255,.8);font-size:13px;line-height:1.8}.wiki-body h3{color:#d4a843;font-size:18px;margin:24px 0 12px;border-left:3px solid #d4a843;padding-left:10px}.wiki-body h4{color:#d4a843;font-size:15px;margin:16px 0 8px}
.prof-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(160px,1fr));gap:10px;margin:12px 0}.prof-card{background:rgba(255,255,255,.04);border:1px solid rgba(212,168,67,.15);border-radius:12px;padding:16px;text-align:center;transition:all .2s}.prof-card:hover{border-color:#d4a843;background:rgba(212,168,67,.08)}.prof-card.locked{opacity:.4;pointer-events:none}.pc-icon{font-size:40px;margin-bottom:8px}.pc-name{font-size:17px;font-weight:800;color:#d4a843;margin-bottom:6px}.pc-desc{font-size:12px;color:rgba(255,255,255,.6);line-height:1.5;margin-bottom:8px}.pc-bonus{font-size:11px;color:#6bcb77;margin-bottom:6px}.pc-status{font-size:11px;color:rgba(255,255,255,.3)}.pc-card{cursor:pointer}.prof-detail{margin-bottom:16px}.pill-craft-list{display:flex;flex-direction:column;gap:6px}.pill-craft-row{display:flex;align-items:center;gap:8px;padding:8px 10px;background:rgba(255,255,255,.03);border:1px solid rgba(255,255,255,.06);border-radius:8px;font-size:12px}.pcr-icon{font-size:22px;min-width:28px;text-align:center}.pcr-info{flex:1;color:rgba(255,255,255,.7)}.pcr-info b{color:#fff}.pcr-rate{color:#6bcb77;font-size:11px;min-width:70px;text-align:center}.pcr-cost{color:rgba(255,255,255,.4);font-size:11px;min-width:50px;text-align:center}
.alchemy-hero{display:flex;align-items:center;gap:16px;padding:16px 20px;background:linear-gradient(135deg,rgba(255,107,107,.05),rgba(212,168,67,.08));border:1px solid rgba(212,168,67,.15);border-radius:14px;margin-bottom:14px}.ah-furnace{font-size:52px;animation:em-bounce .5s ease-in-out infinite alternate}.ah-info{flex:1}.ah-title{font-size:20px;font-weight:900;color:#fff;margin-bottom:6px}.ah-lv{color:#d4a843;font-size:16px}.ah-exp-bar{height:6px;background:rgba(255,255,255,.06);border-radius:3px;overflow:hidden;margin-bottom:4px}.ah-exp-fill{height:100%;background:linear-gradient(90deg,#d4a843,#f0d878);border-radius:3px;transition:width .8s ease}.ah-exp-text{font-size:10px;color:rgba(255,255,255,.3);margin-bottom:2px}.ah-stats{font-size:11px;color:rgba(255,255,255,.5)}.pill-filter{display:flex;gap:4px;flex-wrap:wrap;margin-bottom:14px}.pill-card-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(240px,1fr));gap:10px;margin-bottom:20px}.pill-card{background:rgba(255,255,255,.03);border:1px solid rgba(255,255,255,.06);border-radius:12px;overflow:hidden;transition:all .25s ease;position:relative}.pill-card:hover{transform:translateY(-2px);border-color:rgba(212,168,67,.3);box-shadow:0 8px 24px rgba(0,0,0,.3)}.pill-card.locked{opacity:.35;pointer-events:none}.pc-quality-strip{position:absolute;top:0;left:0;right:0;background:linear-gradient(90deg,#888,#aaa,#6bcb77,#4d96ff,#ff6b9e)}.pc-top{display:flex;align-items:flex-start;justify-content:space-between;padding:10px 12px 0}.pc-icon-lg{font-size:32px}.pc-badges{display:flex;flex-direction:column;align-items:flex-end;gap:3px}.pc-tier-badge{color:#d4a843;font-size:10px;letter-spacing:1px}.pc-cat-badge{padding:1px 6px;border-radius:8px;background:rgba(255,255,255,.06);color:rgba(255,255,255,.4);font-size:9px}.pc-name-lg{padding:4px 12px 0;font-size:16px;font-weight:800;color:#fff}.pc-desc-lg{padding:2px 12px;font-size:11px;color:rgba(255,255,255,.4)}.pc-stats-row{display:flex;gap:16px;padding:8px 12px 4px}.pc-stat{display:flex;flex-direction:column}.pc-stat-label{font-size:9px;color:rgba(255,255,255,.3);margin-bottom:1px}.pc-stat-val{font-size:14px;font-weight:700;color:#fff}.pc-stat-val.green{color:#6bcb77}.pc-quality-bar{display:flex;height:8px;margin:4px 12px 0;border-radius:4px;overflow:hidden}.pc-qb-seg{cursor:help;transition:opacity .2s}.pc-qb-seg:hover{opacity:.8}.pc-actions-lg{display:flex;align-items:center;gap:6px;padding:8px 12px 12px}.pc-btn-minus,.pc-btn-plus{width:28px;height:28px;border:1px solid rgba(255,255,255,.1);border-radius:6px;background:rgba(255,255,255,.04);color:#fff;font-size:16px;cursor:pointer;font-family:inherit;display:flex;align-items:center;justify-content:center}.pc-btn-minus:hover,.pc-btn-plus:hover{background:rgba(255,255,255,.1)}.pc-qty-display{min-width:32px;text-align:center;font-size:16px;font-weight:700;color:#fff}.pc-btn-craft{flex:1;padding:8px 16px;border:none;border-radius:8px;background:linear-gradient(135deg,#d4a843,#b8860b);color:#fff;font-size:13px;font-weight:700;cursor:pointer;font-family:inherit;transition:all .2s;text-align:center}.pc-btn-craft:hover{transform:translateY(-1px);box-shadow:0 4px 16px rgba(212,168,67,.3)}.pc-btn-craft:disabled{background:rgba(255,255,255,.06);color:rgba(255,255,255,.2);transform:none;box-shadow:none;cursor:not-allowed}.craft-modal-lg{background:linear-gradient(180deg,#1a1a2e,#0d0d1a);border-radius:24px;width:420px;max-width:92vw;padding:32px 28px;text-align:center;animation:modal-in .3s ease}.craft-ok{border:2px solid #6bcb77;box-shadow:0 0 60px rgba(107,203,119,.15),0 0 120px rgba(107,203,119,.05)}.craft-no{border:2px solid #e53935;box-shadow:0 0 60px rgba(229,57,53,.15),0 0 120px rgba(229,57,53,.05)}.cml-top{font-size:26px;font-weight:900;margin-bottom:16px}.craft-ok .cml-top{color:#6bcb77}.craft-no .cml-top{color:#e53935}.cml-pill-show{display:flex;align-items:center;justify-content:center;gap:10px;margin-bottom:16px}.cml-pill-icon-lg{font-size:44px}.cml-pill-name-lg{font-size:22px;font-weight:900}.cml-stats-grid{display:flex;gap:12px;justify-content:center;margin-bottom:16px}.cml-stat-box{padding:10px 18px;background:rgba(255,255,255,.04);border-radius:10px;min-width:80px}.cml-stat-v{display:block;font-size:22px;font-weight:900;color:#d4a843}.cml-stat-l{display:block;font-size:10px;color:rgba(255,255,255,.3);margin-top:2px}.cml-fail-reason{font-size:15px;color:rgba(255,255,255,.4);margin-bottom:16px}.cml-btns{display:flex;gap:10px;justify-content:center}.cml-btn-retry{padding:10px 24px;border:1px solid #d4a843;border-radius:10px;background:rgba(212,168,67,.1);color:#d4a843;font-size:14px;font-weight:700;cursor:pointer;font-family:inherit;transition:all .2s}.cml-btn-retry:hover{background:#d4a843;color:#fff}.cml-btn-close{padding:10px 24px;border:1px solid rgba(255,255,255,.1);border-radius:10px;background:transparent;color:rgba(255,255,255,.5);font-size:14px;cursor:pointer;font-family:inherit;transition:all .2s}.cml-btn-close:hover{border-color:rgba(255,255,255,.3);color:#fff}.wiki-table{width:100%;border-collapse:collapse;margin:8px 0 16px;font-size:12px}.wiki-table th{background:rgba(212,168,67,.12);color:#d4a843;padding:6px 8px;text-align:center;font-weight:700;border:1px solid rgba(212,168,67,.1)}.wiki-table td{padding:4px 6px;text-align:center;border:1px solid rgba(255,255,255,.05);white-space:nowrap}.wiki-table td.tc{text-align:center}.wiki-table tr.highlight td{background:rgba(212,168,67,.1);color:#f0d878}.wiki-table tr:hover td{background:rgba(255,255,255,.04)}.wiki-note{color:rgba(255,255,255,.4);font-size:11px;margin:4px 0 12px}.wiki-formula{background:rgba(212,168,67,.08);border:1px solid rgba(212,168,67,.2);border-radius:8px;padding:8px 16px;color:#f0d878;font-size:13px;font-weight:600;text-align:center;margin:8px 0}html.light-mode .wiki-body{color:rgba(0,0,0,.7)}html.light-mode .wiki-table td{border-color:rgba(0,0,0,.05)}html.light-mode .wiki-table tr:hover td{background:rgba(0,0,0,.02)}html.light-mode .wiki-note{color:rgba(0,0,0,.4)}
.pigeon-modal{width:820px;max-width:96vw;background:linear-gradient(180deg,#12122a,#0d0d1a)!important;overflow:hidden!important}.pigeon-body{display:flex;height:540px;max-height:72vh}.pig-left{width:270px;min-width:210px;display:flex;flex-direction:column;overflow:hidden;border-right:1px solid rgba(212,168,67,.1);background:rgba(0,0,0,.2)}.pig-right{flex:1;display:flex;flex-direction:column;background:rgba(0,0,0,.08)}.pig-search-box{padding:12px;background:rgba(0,0,0,.2)}.pig-search-inp{width:100%;padding:8px 14px;border:1px solid rgba(255,255,255,.1);border-radius:20px;background:rgba(255,255,255,.05);color:#fff;font-size:12px;outline:none;font-family:inherit}.pig-search-inp:focus{border-color:#d4a843}.pig-sr-row{display:flex;align-items:center;gap:8px;padding:9px 12px;font-size:12px;border-bottom:1px solid rgba(255,255,255,.03)}.pig-sr-name{color:#fff;flex:1}.pig-sr-tag{font-size:10px;color:rgba(255,255,255,.25);background:rgba(255,255,255,.04);padding:1px 6px;border-radius:3px}.pig-sr-row button{padding:4px 12px;border:1px solid #d4a843;border-radius:12px;background:rgba(212,168,67,.1);color:#d4a843;font-size:11px;cursor:pointer;font-family:inherit}.pig-sr-row button:hover{background:#d4a843;color:#fff}.pig-group{margin:2px 0}.pig-group-head{display:flex;align-items:center;gap:6px;padding:11px 14px;font-size:13px;font-weight:600;color:rgba(255,255,255,.6);cursor:pointer;user-select:none;transition:all .15s;border-left:3px solid transparent}.pig-group-head:hover{background:rgba(255,255,255,.03);color:#d4a843;border-left-color:rgba(212,168,67,.3)}.pig-group-arrow{font-size:8px;transition:transform .2s;color:rgba(255,255,255,.25);margin-left:auto}.pig-group-arrow.open{transform:rotate(0deg)}.pig-group-arrow:not(.open){transform:rotate(-90deg)}.pig-badge{min-width:18px;height:18px;line-height:18px;text-align:center;border-radius:10px;background:#e53935;color:#fff;font-size:10px;font-weight:700;margin-left:auto;padding:0 5px}.pig-friend{display:flex;align-items:center;gap:10px;padding:8px 14px;cursor:pointer;transition:all .12s}.pig-friend:hover{background:rgba(212,168,67,.05)}.pig-friend.active{background:linear-gradient(90deg,rgba(212,168,67,.12),rgba(212,168,67,.04));border-right:3px solid #d4a843}.pig-f-avatar{width:36px;height:36px;border-radius:50%;background:linear-gradient(135deg,#d4a843,#b8860b);display:flex;align-items:center;justify-content:center;font-size:15px;font-weight:700;color:#fff;flex-shrink:0}.pig-f-info{flex:1}.pig-f-name{font-size:13px;color:#fff;display:flex;align-items:center;gap:6px}.pig-f-online{width:6px;height:6px;border-radius:50%;background:#666;flex-shrink:0}.pig-f-online.online{background:#4caf50;box-shadow:0 0 6px rgba(76,175,80,.5);animation:pig-dot-pulse 2s ease-in-out infinite}@keyframes pig-dot-pulse{0%{box-shadow:0 0 4px rgba(76,175,80,.3)}50%{box-shadow:0 0 8px rgba(76,175,80,.6)}}.pig-f-realm{font-size:10px;color:rgba(255,255,255,.25);margin-top:2px}.pig-empty-tip{text-align:center;padding:20px;font-size:12px;color:rgba(255,255,255,.15);font-style:italic}.pig-request-row{display:flex;align-items:center;gap:8px;padding:9px 14px}.pig-r-name{flex:1;font-size:12px;color:#fff}.pig-r-btns{display:flex;gap:4px}.pig-r-btns button{padding:4px 12px;border-radius:12px;font-size:11px;cursor:pointer;font-family:inherit;transition:all .15s;border:1px solid #6bcb77;background:rgba(107,203,119,.08);color:#6bcb77}.pig-r-btns button:hover{background:#6bcb77;color:#fff}.pig-r-btns .pig-r-reject{border-color:rgba(229,57,53,.3);background:rgba(229,57,53,.05);color:rgba(229,57,53,.6)}.pig-r-btns .pig-r-reject:hover{background:#e53935;color:#fff}.pig-chat-msgs{flex:1;overflow-y:auto;padding:12px 14px;display:flex;flex-direction:column;gap:10px}.pig-msg-bubble{display:flex;gap:8px;max-width:75%}.pig-msg-bubble.self{align-self:flex-end;flex-direction:row-reverse}.pig-msg-avatar{width:30px;height:30px;border-radius:50%;background:rgba(255,255,255,.1);display:flex;align-items:center;justify-content:center;font-size:12px;font-weight:700;color:rgba(255,255,255,.6);flex-shrink:0;align-self:flex-end}.pig-msg-bub-text{padding:8px 12px;border-radius:14px;font-size:13px;line-height:1.4;word-break:break-word}.pig-msg-bubble:not(.self) .pig-msg-bub-text{background:rgba(255,255,255,.08);color:#ddd;border-bottom-left-radius:4px}.pig-msg-bubble.self .pig-msg-bub-text{background:linear-gradient(135deg,rgba(212,168,67,.25),rgba(212,168,67,.12));color:#fff;border-bottom-right-radius:4px}.pig-chat-send{display:flex;gap:6px;padding:10px 14px;border-top:1px solid rgba(255,255,255,.06)}.pig-send-inp{flex:1;padding:8px 14px;border:1px solid rgba(255,255,255,.08);border-radius:20px;background:rgba(255,255,255,.04);color:#fff;font-size:13px;outline:none;font-family:inherit}.pig-send-btn{padding:8px 20px;border:none;border-radius:20px;background:linear-gradient(135deg,#d4a843,#b8860b);color:#fff;font-size:13px;font-weight:600;cursor:pointer;font-family:inherit}
</style>
