<template>
  <div class="gh-root" :class="{ 'light-mode': !isDark }">
    <div class="gold-divider"><div class="gold-divider__light" /></div>
    <header class="top-bar">
      <div class="top-bar-inner">
        <span class="brand-logo">☯</span>
        <span class="brand-name">修仙世界</span>
        <van-tabs class="main-nav-tabs" color="#d4a843" title-active-color="#d4a843" title-inactive-color="#8a8578" background="transparent" :border="false" @click-tab="handleTabClick">
          <van-tab title="百科" name="wiki" />
          
          <van-tab v-for="m in menus" :key="m.key" :title="m.label" :name="m.key"  />
        </van-tabs>
        <div class="top-bar-spacer"></div>
        <div class="player-stats">
          <span class="online-badge"><span class="online-dot" />{{ fmt(onlineCount) }} 在线</span>
          <span class="registered-badge">{{ fmt(registeredCount) }} 修士</span>
        </div>
        <van-button icon="exchange" size="small" round plain type="default" @click="toggleTheme" />
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
      <PlayerSidebar :player="player" :isDead="isDead" />


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
      <van-popup v-model:show="modalVisible" position="center" :style="{ width: 'fit-content', minWidth: '360px', maxWidth: '95vw', borderRadius: '8px', background: '#0d0d1a', border: '1px solid rgba(212,168,67,.2)' }" v-if="activeMenu?.key!=='map'&&activeMenu?.key!=='profession'&&activeMenu?.key!=='pigeon'&&activeMenu?.key!=='backpack'">
        <div class="modal-card">
          <div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">{{ activeMenu?.label }}</span><div class="top-bar-spacer"/><button class="modal-close" @click="modalVisible=false">✕</button></div></header><div class="gold-divider"/>
          <div v-if="activeMenu?.children" class="modal-tabs">
            <van-tabs v-model:active="activeSub" color="#d4a843">
              <van-tab v-for="sub in activeMenu.children" :key="sub.key" :title="sub.label" :name="sub.key" />
            </van-tabs>
          </div>
          <div class="modal-body">
            <p class="wiki-note">※ {{ activeSubLabel }} 功能即将上线，敬请期待！</p>
            <p class="modal-desc">{{ modalDesc }}</p>
          </div>
        </div>
      </van-popup>
    </Teleport>
    <!-- 地图弹窗 -->
    <Teleport to="body">
      <div v-if="activeMenu?.key==='map'" class="modal-overlay" @click.self="activeMenu=null">
        <div class="map-modal">
          <div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">🗺️ 修仙世界 · 地图</span><div class="top-bar-spacer"/><button class="modal-close" @click="activeMenu=null">✕</button></div></header><div class="gold-divider"/>
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
          <div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:18px">百科</span><div class="top-bar-spacer"/><button class="modal-close" @click="showWiki=false">✕</button></div></header><div class="gold-divider"/>
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
// useGameState auto-imported by Nuxt
const {
  isDark, toggleTheme, getToken, getPID, refreshToken, activeNav,
  menus, descs, realmNames, realmCoefs, rootMults, qualityNames, qualityColors, pillQualityColors, rootNames, mapRegions, fmt,
  activeMenu, modalVisible, activeSub, modalDesc, activeSubLabel, openMenu,
  player, isDead, hpPct, mpPct, yySpeed, ageBracket, ageDays, loadPlayer,
  training, trainMult, logs,
  showPillPanel, showPillCraft, myPills, pillRecipes, pillCount, pillCat, pillQtys, craftResult, pillStats, pillCats, filteredRecipes, loadPills, loadRecipes, craftAgain, craftPill, usePill,
  activeLoc, currentLoc, currentLocInfo, enterLocation,
  showWiki, wikiTab, wikiTabs,
  bpItems, loadBackpack,
  addLog, loadLogs, saveLogs,
  onlineCount, registeredCount, fetchStats,
  calcOfflineGains,
  toggleMeditation, smallBreakRate, doBreakthrough, toggleTraining, startManual, showTooltip, showCultTooltip, tooltipStyle,
  fightInProgress, fightResult, pveReport, pveRounds, encounterResult, encounterType, deathLog, reviveCountdown, handleQuit, startPve,
  showProfPanel, activeProf,
  wikiRealms, wikiRootBonuses, wikiSpiritReqs, wikiQuality, realmCoef, rootMult, cultBaseVal,
  timeDisplay, uptimeDisplay, getTimeDisplay, getUptimeDisplay,
  friends, pendingRequests, searchResults, friendSearch, activePeer, activePeerName, privateInput, privateMessages, loadFriends, loadPending, searchPlayers, removeFriend, openChat, sendPrivate,
  playerEquips, equipCraftSlot, getEquip, loadEquips,
  connectWS, apiPost, logFilter, logLocked, logBody, filteredLogs,
} = useGameState()


function handleTabClick({ name }: { name: string }) {
  if (name === "wiki") { showWiki.value = true; return }
  const m = menus.find((x: any) => x.key === name)
  if (m) openMenu(m)
}
</script>

<style lang="scss">
.gh-root{height:100vh;display:flex;flex-direction:column;background:#0d0d1a;color:#e8e0d0;font-family:'Noto Sans SC','PingFang SC','Microsoft YaHei',sans-serif;overflow:hidden}.gh-root.light-mode{background:#fff;color:#1a1a1a}
.gold-divider{flex-shrink:0;height:2px;background:linear-gradient(90deg,#b8860b,#d4a843,#f0d878,#d4a843,#b8860b);position:relative;overflow:hidden}.gold-divider__light{position:absolute;top:0;left:-60%;width:60%;height:100%;background:linear-gradient(90deg,transparent,rgba(255,255,255,.15),#ff6b6b,#ffd93d,#6bcb77,#4d96ff,#9b59b6,#ff6b6b,rgba(255,255,255,.15),transparent);animation:rainbow-run 3s linear infinite}@keyframes rainbow-run{to{left:100%}}
.top-bar{flex-shrink:0;background:#000}.light-mode .top-bar{background:#fff;border-bottom:1px solid rgba(0,0,0,.06)}.top-bar-inner{position:relative;padding:0 28px;height:56px;display:flex;align-items:center;gap:10px}.brand-logo{font-size:40px;line-height:1;color:#fff;animation:logo-spin 8s linear infinite}@keyframes logo-spin{to{transform:rotate(360deg)}}.brand-name{font-size:30px;letter-spacing:6px;font-weight:900;letter-spacing:4px;color:#fff}.main-nav{position:absolute;left:50%;transform:translateX(-50%);display:flex;align-items:center;gap:2px;flex-wrap:nowrap}.main-nav-tabs{position:absolute;left:50%;transform:translateX(-50%);z-index:1}.nav-item{padding:4px 7px;cursor:pointer;border-radius:4px;transition:all .2s;white-space:nowrap}.nav-item:hover{background:rgba(255,255,255,.08)}.nav-label{font-size:14px;font-weight:600;color:#fff;letter-spacing:1px}.light-mode .nav-label{color:rgba(0,0,0,.7)}.top-bar-spacer{flex:1}.light-mode .nav-item:hover{background:rgba(0,0,0,.05)}.top-bar-right{display:flex;align-items:center;gap:14px}.player-stats{display:flex;flex-direction:column;align-items:flex-end;gap:2px;line-height:1.2}.online-badge{display:flex;align-items:center;gap:5px;font-size:14px;font-weight:500;color:#fff}.online-dot{width:6px;height:6px;border-radius:50%;background:#4caf50;animation:dot-pulse 2s ease-in-out infinite}@keyframes dot-pulse{0%,to{opacity:.4}50%{opacity:1}}.registered-badge{font-size:14px;font-weight:500;color:#fff}.light-mode .brand-logo,.light-mode .brand-name,.light-mode .online-badge{color:#000}.light-mode .registered-badge{color:rgba(0,0,0,.4)}.theme-toggle{width:52px;height:52px;border:2px solid #d4a843;border-radius:50%;background:#000;color:#fff;font-size:24px;cursor:pointer;display:flex;align-items:center;justify-content:center;box-shadow:0 0 10px rgba(212,168,67,.25);transition:all .3s}.light-mode .theme-toggle{background:#000;color:#fff}.theme-toggle:hover{border-color:#fff;box-shadow:0 0 20px rgba(212,168,67,.4)}
.notice-bar{flex-shrink:0;display:flex;align-items:center;gap:8px;padding:6px 20px;background:rgba(0,0,0,.2);overflow:hidden}.gh-main{flex:1;display:flex;overflow:hidden;gap:0}.gh-panel{display:flex;flex-direction:column;border-right:1px solid rgba(212,168,67,.1);background:#1a1a2e}.light-mode .gh-panel{background:#f5f5f5}.gh-log{width:41%;flex-shrink:0}.gh-chat{width:41%;flex-shrink:0}.gh-sidebar{width:18%;min-width:210px;border-right:none;flex-shrink:0;display:flex;flex-direction:column;gap:4px;padding:6px 4px;overflow-y:auto;border-right:2px solid rgba(212,168,67,.25);background:rgba(0,0,0,.15)}.light-mode .gh-sidebar{background:rgba(0,0,0,.02);border-right-color:rgba(0,0,0,.1)}.side-card{display:flex;flex-direction:column;justify-content:center;border:1px solid rgba(212,168,67,.15);border-radius:8px;padding:6px;overflow:hidden}
.light-mode .side-card{background:transparent;border-color:rgba(0,0,0,.08)}.side-card h4{margin:0 0 10px;font-size:13px;color:#d4a843;text-align:center}.panel-header{display:flex;align-items:center;justify-content:space-between;padding:8px 12px;border-bottom:1px solid rgba(212,168,67,.1);flex-shrink:0}.panel-header h4{margin:0;font-size:15px;color:#d4a843}.panel-actions{display:flex;gap:4px;flex-wrap:wrap}.pa-btn{padding:2px 8px;border:1px solid rgba(212,168,67,.2);border-radius:4px;background:transparent;color:rgba(255,255,255,.5);font-size:11px;cursor:pointer;transition:all .2s;font-family:inherit}.pa-btn:hover{color:#d4a843;border-color:#d4a843}.pa-btn.active{background:rgba(212,168,67,.15);color:#d4a843;border-color:#d4a843}.light-mode .pa-btn{color:rgba(0,0,0,.4);border-color:rgba(0,0,0,.1)}.panel-body{flex:1;overflow-y:auto;padding:8px 10px;font-size:14px;line-height:1.8}.log-entry{padding:2px 0;border-bottom:1px solid rgba(255,255,255,.03);color:rgba(255,255,255,.6)}.light-mode .log-entry{color:rgba(0,0,0,.5);border-bottom-color:rgba(0,0,0,.04)}.log-entry.combat{color:#ff6b6b}.log-entry.cult{color:#6bcb77}.log-entry.item{color:#4d96ff}.log-entry.social{color:#ffd93d}.log-entry.explore{color:#64ffda}.log-entry.quest{color:#b388ff}.log-entry.system{color:#fff;font-weight:700}.light-mode .log-entry.system{color:#000}.log-entry.important{color:#f0d878;font-weight:700;font-size:15px}.light-mode .log-entry.important{color:#b8860b}.log-empty{text-align:center;color:rgba(255,255,255,.2);padding:20px}.chat-body{display:flex;flex-direction:column;gap:2px}.chat-msg{font-size:14px;line-height:1.6}.chat-time{color:rgba(255,255,255,.2);margin-right:6px;font-size:11px}.chat-name{font-weight:700;margin-right:4px}.chat-text{color:rgba(255,255,255,.75)}.chat-online{font-size:11px;color:rgba(255,255,255,.3)}.chat-input-bar{display:flex;gap:6px;padding:8px 10px;border-top:1px solid rgba(212,168,67,.1);flex-shrink:0}.chat-input{flex:1;padding:10px 14px;border:1px solid rgba(212,168,67,.15);border-radius:6px;background:rgba(255,255,255,.04);color:#fff;font-size:14px;outline:none;font-family:inherit}.light-mode .chat-input{background:rgba(0,0,0,.03);color:#000;border-color:rgba(0,0,0,.1)}.chat-input::placeholder{color:rgba(255,255,255,.2)}.chat-send{padding:8px 18px;border:1px solid #d4a843;border-radius:6px;background:transparent;color:#d4a843;font-size:14px;cursor:pointer;font-family:inherit;transition:all .2s}.chat-send:hover{background:#d4a843;color:#fff}.map-body{display:flex;flex-direction:column;gap:6px}.map-btn{width:100%;padding:10px 12px;border:1px solid rgba(212,168,67,.2);border-radius:8px;background:rgba(212,168,67,.05);color:rgba(255,255,255,.7);font-size:13px;font-weight:500;cursor:pointer;transition:all .2s;font-family:inherit;text-align:left}.map-btn:hover{background:rgba(212,168,67,.15);color:#d4a843;border-color:#d4a843;transform:translateX(4px)}.light-mode .map-btn{background:rgba(0,0,0,.02);color:rgba(0,0,0,.5);border-color:rgba(0,0,0,.08)}.light-mode .map-btn:hover{background:rgba(184,134,11,.08);color:#b8860b}.notice-icon{flex-shrink:0;font-size:14px}.notice-text{font-size:12px;color:rgba(255,255,255,.5);white-space:nowrap;animation:notice-scroll 20s linear infinite}.light-mode .notice-bar{background:rgba(0,0,0,.03)}.light-mode .notice-text{color:rgba(0,0,0,.4)}@keyframes notice-scroll{0%{transform:translateX(100%)}100%{transform:translateX(-100%)}}
.side-profile{flex:0 0 auto;padding:0 8px 6px;overflow:visible;justify-content:flex-start}.profile-top{display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:4px}.profile-top-left{display:flex;align-items:center;gap:8px;min-width:0;flex:1}.side-avatar{width:40px;height:40px;border-radius:50%;background:linear-gradient(135deg,#d4a843,#b8860b);flex-shrink:0;display:flex;align-items:center;justify-content:center;font-size:20px;font-weight:700;color:#fff}.side-name{font-size:19px;font-weight:800;color:#fff;line-height:1.1}.light-mode .side-name{color:#000}.gender-tag{margin-left:2px;font-size:17px}.profile-tags{display:flex;gap:5px;font-size:10px;flex-wrap:wrap}.side-realm{color:#d4a843;font-weight:700}.side-spirit{color:#fff;font-weight:600;font-size:10px}.light-mode .side-spirit{color:#000}.profile-loc{font-size:10px;color:#6bcb77;display:flex;align-items:center;gap:2px}.pl-icon{font-size:13px}.profile-power-badge{text-align:center;min-width:50px;margin-top:-3px;cursor:help}.pp-val{font-size:24px;font-weight:900;color:#e53935;line-height:1}.profile-bars{margin:4px 0 3px}.pbar-row{display:flex;align-items:center;gap:6px;margin-bottom:4px;color:#fff;font-weight:600}.light-mode .pbar-row{color:#000}.pbar-row>span:first-child{font-size:14px;width:18px;text-align:center;flex-shrink:0}.pbar-val{width:62px;text-align:right;flex-shrink:0;font-size:11px;font-weight:700}.pbar-track{flex:1;height:8px;background:rgba(255,255,255,.08);border-radius:4px;overflow:hidden}.pbar-fill{height:100%;border-radius:4px;transition:width .6s}.pbar-fill.hp{background:linear-gradient(90deg,#e53935,#ef5350);box-shadow:0 0 4px rgba(229,57,53,.3)}.pbar-fill.mp{background:linear-gradient(90deg,#42a5f5,#64b5f6);box-shadow:0 0 4px rgba(66,165,245,.3)}.profile-attrs{display:grid;grid-template-columns:1fr 1fr;gap:0 6px}.pa-row{display:flex;justify-content:space-between;padding:1px 0;font-size:11px;color:#bbb;border-bottom:1px solid rgba(255,255,255,.04)}.light-mode .pa-row{color:rgba(0,0,0,.6);border-bottom-color:rgba(0,0,0,.06)}.pa-row span:first-child{color:#999}.light-mode .pa-row span:first-child{color:rgba(0,0,0,.4)}.pa-row span:last-child{font-weight:700;color:#fff}.light-mode .pa-row span:last-child{color:#000}
@keyframes gold-flow{0%{background-position:100% 0}100%{background-position:-100% 0}}
.side-cultivation{flex:0 0 auto;text-align:center;display:flex;flex-direction:column;align-items:center;gap:2px;padding:6px}.side-equip{flex:1}.equip-slots{display:grid;grid-template-columns:1fr 1fr;gap:4px}.eq-slot{font-weight:700;display:flex;flex-direction:row;align-items:center;gap:4px;justify-content:center;text-align:center;padding:8px;border:1.5px solid rgba(212,168,67,.25);border-radius:6px;font-size:22px;color:#fff}.eq-slot span{font-size:14px;font-weight:800;color:#fff}.cult-yy{width:100px;height:100px;border-radius:50%;flex-shrink:0;border:2px solid #d4a843;padding:3px;box-shadow:0 0 8px rgba(212,168,67,.3);position:relative}
.cult-info{width:100%}.cult-bar-wrap{display:flex;align-items:center;gap:6px;margin-bottom:4px}
.cult-bar::after{content:"";position:absolute;inset:0;background:linear-gradient(90deg,transparent,rgba(255,255,255,.15),transparent);animation:bar-shine 2s ease-in-out infinite}@keyframes bar-shine{0%{transform:translateX(-100%)}100%{transform:translateX(200%)}}.cult-yy.male{background:linear-gradient(to right,#fff 50%,#000 50%)}.cult-yy.male::before{content:'';position:absolute;width:50%;height:50%;top:0;left:25%;border-radius:50%;background:#fff;background-image:radial-gradient(circle at 50% 62%,#000 22%,transparent 22%)}.cult-yy.male::after{content:'';position:absolute;width:50%;height:50%;top:50%;left:25%;border-radius:50%;background:#000;background-image:radial-gradient(circle at 50% 38%,#fff 22%,transparent 22%)}.cult-yy.female{background:linear-gradient(to right,#ff6b6b 50%,#ffd700 50%)}.cult-yy.female::before{content:'';position:absolute;width:50%;height:50%;top:0;left:25%;border-radius:50%;background:#ff6b6b;background-image:radial-gradient(circle at 50% 62%,#ffd700 22%,transparent 22%)}.cult-yy.female::after{content:'';position:absolute;width:50%;height:50%;top:50%;left:25%;border-radius:50%;background:#ffd700;background-image:radial-gradient(circle at 50% 38%,#ff6b6b 22%,transparent 22%)}.cult-yy-wrap.meditating .cult-yy{animation:spin-yy 8s linear infinite}@keyframes rainbow-glow{0%{box-shadow:0 0 0 12px #d4a843,0 0 40px #d4a843,0 0 60px rgba(212,168,67,.4)}25%{box-shadow:0 0 0 12px #ff6b6b,0 0 40px #ff6b6b,0 0 60px rgba(255,107,107,.4)}50%{box-shadow:0 0 0 12px #6bcb77,0 0 40px #6bcb77,0 0 60px rgba(107,203,119,.4)}75%{box-shadow:0 0 0 12px #4d96ff,0 0 40px #4d96ff,0 0 60px rgba(77,150,255,.4)}100%{box-shadow:0 0 0 12px #d4a843,0 0 40px #d4a843,0 0 60px rgba(212,168,67,.4)}}.cult-info{width:100%}.cult-bar-wrap{display:flex;align-items:center;gap:6px;margin-bottom:4px}.cult-bar-label{font-size:11px;color:#d4a843;font-weight:600;flex-shrink:0}.cult-bar{height:7px;background:rgba(255,255,255,.06);border-radius:3px;overflow:hidden;margin-bottom:3px;position:relative}.cult-fill{height:100%;background:linear-gradient(90deg,#b8860b,#d4a843,#f0d878,#d4a843,#b8860b);background-size:200% 100%;animation:gold-flow 2s linear infinite,fill-pulse 2s ease-in-out infinite;border-radius:3px;transition:width .8s ease;box-shadow:0 0 8px rgba(212,168,67,.4)}@keyframes fill-pulse{0%,100%{box-shadow:0 0 6px rgba(212,168,67,.3)}50%{box-shadow:0 0 14px rgba(212,168,67,.6),0 0 20px rgba(240,216,120,.3)}}.cult-stats{display:flex;justify-content:space-between;font-size:11px;color:rgba(255,255,255,.7);margin-bottom:4px;font-weight:600}.light-mode .cult-stats{color:rgba(0,0,0,.6)}.cult-btn.danger{border-color:#e53935;color:#e53935}.cult-btn.danger:hover{box-shadow:0 4px 14px rgba(229,57,53,.3);background:rgba(229,57,53,.1)}.cult-realm-info{display:flex;gap:4px;justify-content:center;flex-wrap:wrap;margin:3px 0}.realm-tag{padding:1px 8px;border-radius:8px;background:rgba(212,168,67,.12);color:#d4a843;font-size:11px;font-weight:700}.root-tag{padding:1px 8px;border-radius:8px;background:rgba(255,255,255,.06);font-size:10px;font-weight:600}.light-mode .root-tag{background:rgba(0,0,0,.04)}.cult-btns{display:flex;gap:5px;width:100%;justify-content:center;flex-wrap:wrap}.cult-btn{padding:6px 12px;border:1.5px solid #d4a843;border-radius:8px;background:transparent;color:#d4a843;font-size:11px;font-weight:700;cursor:pointer;transition:all .25s ease;font-family:inherit}.cult-btn:hover{transform:translateY(-1px);box-shadow:0 4px 14px rgba(212,168,67,.3);background:rgba(212,168,67,.1)}.cult-btn:active{transform:translateY(0)}.train-btn{border-color:#64ffda;color:#64ffda}.train-btn:hover{box-shadow:0 4px 14px rgba(100,255,218,.3);background:rgba(100,255,218,.1)}.train-panel{width:100%;margin-top:6px;padding:6px 8px;background:rgba(100,255,218,.05);border:1px solid rgba(100,255,218,.15);border-radius:6px;font-size:10px}.train-info{display:flex;justify-content:space-between;align-items:center;color:rgba(255,255,255,.5);margin-bottom:3px}.train-switch{cursor:pointer;display:flex;align-items:center;gap:4px;color:#64ffda;margin-bottom:3px;font-size:10px}.train-switch input{accent-color:#64ffda;cursor:pointer}.mult-btn{padding:2px 6px;border:1px solid rgba(100,255,218,.2);border-radius:3px;background:transparent;color:rgba(100,255,218,.5);font-size:10px;cursor:pointer;font-family:inherit;transition:all .2s}.mult-btn:hover{border-color:#64ffda;color:#64ffda}.mult-btn.on{background:rgba(100,255,218,.15);border-color:#64ffda;color:#64ffda;font-weight:700}.mult-btn:disabled{opacity:.3;pointer-events:none}.manual-train-btn{padding:2px 8px;border:1px solid #ffd700;border-radius:3px;background:rgba(255,215,0,.1);color:#ffd700;font-size:10px;cursor:pointer;font-family:inherit;transition:all .2s;margin-left:auto}.manual-train-btn:hover{background:rgba(255,215,0,.2)}.manual-train-btn:disabled{opacity:.3;pointer-events:none}.dead-warn{text-align:center;color:#e53935;font-weight:700;font-size:11px;padding:4px 0}.death-modal{background:#1a1a2e;border:2px solid #e53935;border-radius:20px;width:420px;max-width:92vw;max-height:85vh;overflow-y:auto;padding:24px;box-shadow:0 0 40px rgba(229,57,53,.3);text-align:center;animation:modal-in .25s ease}.dm-header{font-size:28px;font-weight:900;color:#e53935;margin-bottom:12px}.dm-timer{font-size:18px;color:#ffd700;margin-bottom:8px}.dm-timer b{font-size:28px}.dm-info{font-size:12px;color:rgba(255,255,255,.5);margin-bottom:12px}.dm-status{display:flex;gap:12px;justify-content:center;margin-bottom:12px}.dm-status span{background:rgba(255,255,255,.06);padding:4px 12px;border-radius:6px;font-size:13px;color:#fff}.dm-battle{margin-top:12px;padding:8px;background:rgba(229,57,53,.06);border:1px solid rgba(229,57,53,.15);border-radius:8px;font-size:10px;text-align:left;max-height:180px;overflow-y:auto}.dm-battle-title{color:#e53935;font-weight:700;font-size:12px;margin-bottom:4px}.dm-note{margin-top:12px;font-size:11px;color:rgba(255,255,255,.3)}
.encounter-modal{background:#1a1a2e;border:2px solid #ffd700;border-radius:20px;width:400px;max-width:92vw;padding:28px;box-shadow:0 0 60px rgba(255,215,0,.3);text-align:center;animation:modal-in .25s ease}.em-icon{font-size:60px;margin-bottom:8px;animation:em-bounce .5s ease-in-out infinite alternate}@keyframes em-bounce{from{transform:translateY(0)}to{transform:translateY(-6px)}}.em-title{font-size:22px;font-weight:900;color:#ffd700;margin-bottom:8px}.em-desc{font-size:13px;color:rgba(255,255,255,.6);margin-bottom:12px;line-height:1.6}.em-change{font-size:18px;font-weight:700;margin:12px 0}.em-old{color:rgba(255,255,255,.4);text-decoration:line-through}.em-new{color:#6bcb77;font-size:22px;animation:em-glow 1s ease-in-out infinite alternate}@keyframes em-glow{from{text-shadow:0 0 10px rgba(107,203,119,.5)}to{text-shadow:0 0 20px rgba(107,203,119,.8),0 0 30px rgba(107,203,119,.4)}}.em-note{font-size:11px;color:rgba(255,255,255,.4)}.combat-report{margin-top:6px;padding:6px;background:rgba(255,107,107,.05);border:1px solid rgba(255,107,107,.2);border-radius:4px;font-size:10px;max-height:150px;overflow-y:auto}.cr-header{color:#ff6b6b;font-weight:700;margin-bottom:4px;font-size:11px}.cr-round{padding:2px 0;border-bottom:1px solid rgba(255,107,107,.1);color:rgba(255,255,255,.7);line-height:1.5}.cr-dmg{color:#ff6b6b;margin:0 4px}.cr-special{color:#ffd700;font-weight:700}.cr-hp{color:rgba(255,255,255,.5);font-size:9px}
.energy-canvas{position:absolute;inset:-50px;width:calc(100% + 100px);height:calc(100% + 100px);pointer-events:none;z-index:1}
.cult-yy-wrap{position:relative;width:100px;height:100px;flex-shrink:0}
.cult-tooltip{position:fixed;transform:translateY(-50%);z-index:5000;min-width:220px;background:rgba(10,10,30,.96);border:1px solid rgba(212,168,67,.4);border-radius:8px;padding:14px 16px;box-shadow:0 8px 32px rgba(0,0,0,.6);pointer-events:none}.light-mode .cult-tooltip{background:rgba(255,255,255,.98);border-color:rgba(184,134,11,.3);box-shadow:0 8px 32px rgba(0,0,0,.15)}.ct-title{font-size:14px;font-weight:700;color:#d4a843;margin-bottom:10px;text-align:center;letter-spacing:2px}.ct-rows{display:flex;flex-direction:column;gap:6px}.ct-row{display:flex;justify-content:space-between;font-size:12px;color:rgba(255,255,255,.7)}.light-mode .ct-row{color:rgba(0,0,0,.6)}.ct-row span:last-child{font-weight:600;color:#fff}.light-mode .ct-row span:last-child{color:#000}.ct-row.ct-sub{border-top:1px dashed rgba(212,168,67,.15);padding-top:6px;color:rgba(255,255,255,.5)}.ct-row.ct-sub span:last-child{color:#d4a843}.ct-row.ct-bonus{color:#6bcb77}.ct-row.ct-bonus span:last-child{color:#6bcb77}.ct-row.ct-none{color:rgba(255,255,255,.25)}.ct-row.ct-none span:last-child{color:rgba(255,255,255,.25)}.ct-divider{height:1px;background:linear-gradient(90deg,transparent,rgba(212,168,67,.3),transparent);margin:2px 0}.ct-row.ct-total{font-size:14px;font-weight:700;color:#d4a843}.ct-row.ct-total span:last-child{color:#f0d878;font-size:15px}.ct-formula{margin-top:8px;font-size:10px;color:rgba(255,255,255,.25);text-align:center}
.tooltip-enter-active{transition:opacity .2s}.tooltip-leave-active{transition:opacity .15s}.tooltip-enter-from,.tooltip-leave-to{opacity:0}
.gh-center{display:none;flex-direction:column;align-items:center;justify-content:center;gap:20px;padding:20px;text-align:center;overflow:hidden}.yin-yang{--yy-size:200px;width:var(--yy-size);height:var(--yy-size);border-radius:50%;margin:0 auto;background:linear-gradient(to right,#fff 50%,#000 50%);position:relative;animation:spin-yy 8s linear infinite;cursor:pointer;transition:transform .4s cubic-bezier(.175,.885,.32,1.275);box-shadow:0 0 30px rgba(255,255,255,.15),0 0 60px rgba(0,0,0,.4)}.yin-yang:active{transform:scale(1.15)}.yin-yang::before,.yin-yang::after{content:'';position:absolute;border-radius:50%}.yin-yang::before{width:50%;height:50%;top:0;left:25%;background:#fff;background-image:radial-gradient(circle at 50% 62%,#000 24%,transparent 24%)}.yin-yang::after{width:50%;height:50%;top:50%;left:25%;background:#000;background-image:radial-gradient(circle at 50% 38%,#fff 24%,transparent 24%)}@keyframes spin-yy{to{transform:rotate(360deg)}}.game-title{margin:0;font-size:72px;font-weight:900;letter-spacing:12px;line-height:1.1;background:linear-gradient(135deg,#d4a843,#f0d878 30%,#d4a843 60%,#b8942e);background-size:200% 200%;-webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text;animation:gold-shimmer 4s ease-in-out infinite}@keyframes gold-shimmer{0%,to{background-position:0 50%}50%{background-position:100% 50%}}.game-subtitle{margin:4px 0 0;font-size:22px;font-weight:500;color:rgba(232,224,208,.6);letter-spacing:10px}.light-mode .game-subtitle{color:rgba(0,0,0,.55)}.realm-progress{display:flex;align-items:flex-start;position:relative;width:100%;max-width:440px;padding:0 8px}.realm-track-bg{position:absolute;top:14px;left:16px;right:16px;height:2px;background:rgba(232,224,208,.15);border-radius:1px}.realm-node{display:flex;flex-direction:column;align-items:center;gap:6px;flex:1;cursor:pointer;opacity:0;transform:translateX(-20px);animation:realm-in .5s ease-out forwards}@keyframes realm-in{to{opacity:1;transform:translateX(0)}}.realm-dot{width:24px;height:24px;border-radius:50%;border:2px solid rgba(232,224,208,.2);background:#0d0d1a;display:flex;align-items:center;justify-content:center}.realm-node.completed .realm-dot{border-color:#d4a843;background:#d4a843}.realm-node.active .realm-dot{border-color:#d4a843;box-shadow:0 0 0 3px rgba(212,168,67,.25)}.realm-check{color:#fff;font-size:12px;font-weight:700}.realm-pulse{width:8px;height:8px;border-radius:50%;background:#d4a843;animation:realm-pulse 2s ease-in-out infinite}@keyframes realm-pulse{0%,to{opacity:.4;transform:scale(.8)}50%{opacity:1;transform:scale(1.2)}}.realm-label{font-size:13px;color:rgba(232,224,208,.35)}.realm-node.completed .realm-label,.realm-node.active .realm-label{color:#d4a843}.light-mode .realm-dot{background:#fff!important}.cultivation-quote{margin:0;font-size:14px;color:rgba(212,168,67,.5);letter-spacing:4px;font-style:italic;animation:quote-fade 5s ease-in-out infinite}@keyframes quote-fade{0%,to{opacity:.4}50%{opacity:1}}.bottom-bar{flex-shrink:0;background:rgba(13,13,26,.95);padding:8px 20px;display:flex;justify-content:space-between;align-items:center;border-top:1px solid rgba(212,168,67,.12)}.light-mode .bottom-bar{background:rgba(255,255,255,.95)}.footer-left{display:flex;align-items:center;gap:10px;font-size:12px;color:rgba(232,224,208,.45)}.light-mode .footer-left{color:#000}.footer-divider{color:rgba(212,168,67,.3)}.footer-uptime{font-size:13px}.footer-right{display:flex;align-items:center}.footer-quit{padding:4px 12px;border:1px solid rgba(229,57,53,.3);border-radius:4px;background:transparent;color:rgba(229,57,53,.6);font-size:12px;cursor:pointer;transition:all .2s;font-family:inherit}.footer-quit:hover{border-color:#e53935;color:#e53935;background:rgba(229,57,53,.08)}@media(max-width:768px){.yin-yang{--yy-size:120px!important}.game-title{font-size:40px!important}.main-nav{display:none}.brand-name{display:none}}
.modal-overlay{position:fixed;inset:0;z-index:3000;display:flex;align-items:center;justify-content:center;background:rgba(0,0,0,.6)}.modal-card{background:#0d0d1a;border:1px solid rgba(212,168,67,.2);border-radius:8px;width:fit-content;min-width:360px;max-width:92vw;max-height:85vh;overflow:hidden;display:flex;flex-direction:column;box-shadow:0 24px 80px rgba(0,0,0,.6);animation:modal-in .25s ease}.light-mode .modal-card{background:#fff;border-color:rgba(184,134,11,.15);box-shadow:0 24px 80px rgba(0,0,0,.1)}@keyframes modal-in{from{opacity:0;transform:translateY(12px)}to{opacity:1;transform:translateY(0)}}.modal-close{position:absolute;top:50%;right:14px;transform:translateY(-50%);background:none;border:none;color:rgba(255,255,255,.25);font-size:20px;cursor:pointer;width:32px;height:32px;display:flex;align-items:center;justify-content:center;border-radius:50%;transition:all .2s}.modal-close:hover{color:#e53935;background:rgba(229,57,53,.1)}html.light-mode .modal-close{color:rgba(0,0,0,.2)}html.light-mode .modal-close:hover{color:#e53935}.modal-tabs{display:flex;justify-content:center;gap:6px;padding:10px 16px;flex-wrap:wrap;border-bottom:1px solid rgba(212,168,67,.08)}.modal-tab{padding:6px 14px;border:1.5px solid rgba(212,168,67,.15);border-radius:6px;background:transparent;color:rgba(255,255,255,.5);font-size:12px;cursor:pointer;transition:all .2s;font-family:inherit;font-weight:600}.modal-tab:hover{color:#d4a843;border-color:#d4a843;background:rgba(212,168,67,.05);transform:translateY(-1px)}.modal-tab.active{background:linear-gradient(135deg,rgba(212,168,67,.12),transparent);color:#d4a843;border-color:#d4a843;font-weight:700}html.light-mode .modal-tab{color:rgba(0,0,0,.35);border-color:rgba(184,134,11,.1)}html.light-mode .modal-tab:hover{color:#b8860b;border-color:#b8860b}html.light-mode .modal-tab.active{background:rgba(184,134,11,.08);color:#b8860b;border-color:#b8860b}.modal-body{flex:1;overflow-y:auto;padding:24px 28px}.modal-placeholder{text-align:center;font-size:44px;margin:12px 0 8px;opacity:.5}.modal-desc{text-align:center;font-size:13px;color:rgba(255,255,255,.35);line-height:2}.light-mode .modal-desc{color:rgba(0,0,0,.3)}.wiki-modal{background:#0d0d1a;border:1px solid rgba(212,168,67,.2);border-radius:8px;width:fit-content;max-width:95vw;max-height:85vh;overflow:hidden;display:flex;flex-direction:column;box-shadow:0 24px 80px rgba(0,0,0,.6)}.light-mode .wiki-modal{background:#fff;border-color:rgba(184,134,11,.15)}.wiki-tabs{display:flex;justify-content:center;gap:6px;padding:8px 16px;border-bottom:1px solid rgba(212,168,67,.08)}.wiki-body{flex:1;overflow-y:auto;padding:16px 24px 24px}.light-mode .eq-slot,.light-mode .eq-slot span{color:#000!important}
.map-modal{background:#1a1a2e;border:1px solid rgba(212,168,67,.15);border-radius:20px;width:780px;max-width:95vw;max-height:88vh;overflow-y:auto;box-shadow:0 24px 64px rgba(0,0,0,.5);animation:modal-in .25s ease}html.light-mode .map-modal{background:#fff;border-color:rgba(0,0,0,.08)}.map-body-wrap{padding:16px 24px 28px}.map-legend{display:flex;gap:20px;align-items:center;flex-wrap:wrap;padding:8px 16px;margin-bottom:12px;background:rgba(255,255,255,.03);border-radius:8px;font-size:12px;color:rgba(255,255,255,.4)}.ml-item{display:flex;align-items:center;gap:4px}.ml-dot{width:10px;height:10px;border-radius:50%;display:inline-block}.ml-dot.unlocked{background:#4caf50;box-shadow:0 0 6px rgba(76,175,80,.4)}.ml-dot.locked{background:#666}.map-region{background:rgba(255,255,255,.03);border:1px solid rgba(212,168,67,.1);border-radius:8px;padding:12px 16px;margin-bottom:10px;transition:all .2s}.map-region.locked{opacity:.45;filter:grayscale(.6)}.mr-header{display:flex;align-items:center;gap:8px;margin-bottom:8px}.mr-icon{font-size:24px}.mr-name{font-size:17px;font-weight:800;color:#d4a843}.mr-realm{font-size:12px;color:rgba(255,255,255,.3);margin-left:auto}.mr-lock{font-size:11px;color:#e53935;font-weight:600}.mr-locations{display:grid;grid-template-columns:repeat(auto-fill,minmax(160px,1fr));gap:6px}.mr-loc{cursor:pointer;padding:8px 10px;background:rgba(255,255,255,.04);border:1px solid rgba(255,255,255,.06);border-radius:8px;transition:all .2s;position:relative}.mr-loc:hover{background:rgba(212,168,67,.08);border-color:rgba(212,168,67,.2)}.mr-loc.locked{opacity:.4;pointer-events:none}.mrl-icon{font-size:18px;margin-right:4px}.mrl-name{font-size:13px;font-weight:700;color:#fff}.mrl-type{display:block;font-size:10px;color:rgba(255,255,255,.35);margin-top:2px}.mrl-lock{position:absolute;top:6px;right:8px;font-size:12px}.mrl-detail{margin-top:8px;padding:8px;background:rgba(0,0,0,.3);border-radius:6px;font-size:11px;color:rgba(255,255,255,.6);line-height:1.6}.mrl-detail p{margin:0 0 4px;color:rgba(255,255,255,.8)}.mrl-info{display:flex;flex-wrap:wrap;gap:8px;margin-bottom:6px}.mrl-info span{background:rgba(212,168,67,.1);padding:2px 8px;border-radius:4px;font-size:10px;color:#d4a843}.map-enter-btn{width:100%;padding:6px;border:1px solid #4caf50;border-radius:6px;background:rgba(76,175,80,.1);color:#4caf50;font-size:12px;font-weight:700;cursor:pointer;transition:all .2s;font-family:inherit;margin-top:4px}.map-enter-btn:hover{background:#4caf50;color:#fff}
.map-current-badge{display:block;text-align:center;padding:6px;color:#4caf50;font-size:12px;font-weight:700;margin-top:4px}
.map-fight-btn{width:100%;padding:6px;border:1px solid #e53935;border-radius:6px;background:rgba(229,57,53,.1);color:#e53935;font-size:12px;font-weight:700;cursor:pointer;transition:all .2s;font-family:inherit;margin-top:4px}.map-fight-btn:hover{background:#e53935;color:#fff}
.mr-loc.current{border-color:rgba(76,175,80,.5);background:rgba(76,175,80,.12);box-shadow:0 0 8px rgba(76,175,80,.15)}
html.light-mode .map-modal{background:#fafafa}html.light-mode .map-region{background:rgba(0,0,0,.02);border-color:rgba(0,0,0,.06)}html.light-mode .mr-loc{background:rgba(0,0,0,.03)}html.light-mode .mr-name{color:#b8860b}html.light-mode .mrl-name{color:#000}
.wiki-btn:hover{background:rgba(212,168,67,.2);box-shadow:0 0 12px rgba(212,168,67,.3)}.wiki-tabs{display:flex;justify-content:center;gap:8px;padding:4px 16px 8px;flex-wrap:wrap;border-bottom:1px solid rgba(212,168,67,.1);margin-bottom:12px}.wiki-body{padding:0 20px 20px;color:rgba(255,255,255,.8);font-size:13px;line-height:1.8;overflow-x:auto}.wiki-body h3{color:#d4a843;font-size:18px;margin:24px 0 12px;border-left:3px solid #d4a843;padding-left:10px}.wiki-body h4{color:#d4a843;font-size:15px;margin:16px 0 8px}
.prof-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(160px,1fr));gap:10px;margin:12px 0}.prof-card{background:rgba(255,255,255,.04);border:1px solid rgba(212,168,67,.15);border-radius:8px;padding:16px;text-align:center;transition:all .2s}.prof-card:hover{border-color:#d4a843;background:rgba(212,168,67,.08)}.prof-card.locked{opacity:.4;pointer-events:none}.pc-icon{font-size:40px;margin-bottom:8px}.pc-name{font-size:17px;font-weight:800;color:#d4a843;margin-bottom:6px}.pc-desc{font-size:12px;color:rgba(255,255,255,.6);line-height:1.5;margin-bottom:8px}.pc-bonus{font-size:11px;color:#6bcb77;margin-bottom:6px}.pc-status{font-size:11px;color:rgba(255,255,255,.3)}.pc-card{cursor:pointer}.prof-detail{margin-bottom:16px}.pill-craft-list{display:flex;flex-direction:column;gap:6px}.pill-craft-row{display:flex;align-items:center;gap:8px;padding:8px 10px;background:rgba(255,255,255,.03);border:1px solid rgba(255,255,255,.06);border-radius:8px;font-size:12px}.pcr-icon{font-size:22px;min-width:28px;text-align:center}.pcr-info{flex:1;color:rgba(255,255,255,.7)}.pcr-info b{color:#fff}.pcr-rate{color:#6bcb77;font-size:11px;min-width:70px;text-align:center}.pcr-cost{color:rgba(255,255,255,.4);font-size:11px;min-width:50px;text-align:center}
.alchemy-hero{display:flex;align-items:center;gap:16px;padding:16px 20px;background:linear-gradient(135deg,rgba(255,107,107,.05),rgba(212,168,67,.08));border:1px solid rgba(212,168,67,.15);border-radius:14px;margin-bottom:14px}.ah-furnace{font-size:52px;animation:em-bounce .5s ease-in-out infinite alternate}.ah-info{flex:1}.ah-title{font-size:20px;font-weight:900;color:#fff;margin-bottom:6px}.ah-lv{color:#d4a843;font-size:16px}.ah-exp-bar{height:6px;background:rgba(255,255,255,.06);border-radius:3px;overflow:hidden;margin-bottom:4px}.ah-exp-fill{height:100%;background:linear-gradient(90deg,#d4a843,#f0d878);border-radius:3px;transition:width .8s ease}.ah-exp-text{font-size:10px;color:rgba(255,255,255,.3);margin-bottom:2px}.ah-stats{font-size:11px;color:rgba(255,255,255,.5)}.pill-filter{display:flex;gap:4px;flex-wrap:wrap;margin-bottom:14px}.pill-card-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(240px,1fr));gap:10px;margin-bottom:20px}.pill-card{background:rgba(255,255,255,.03);border:1px solid rgba(255,255,255,.06);border-radius:8px;overflow:hidden;transition:all .25s ease;position:relative}.pill-card:hover{transform:translateY(-2px);border-color:rgba(212,168,67,.3);box-shadow:0 8px 24px rgba(0,0,0,.3)}.pill-card.locked{opacity:.35;pointer-events:none}.pc-quality-strip{position:absolute;top:0;left:0;right:0;background:linear-gradient(90deg,#888,#aaa,#6bcb77,#4d96ff,#ff6b9e)}.pc-top{display:flex;align-items:flex-start;justify-content:space-between;padding:10px 12px 0}.pc-icon-lg{font-size:32px}.pc-badges{display:flex;flex-direction:column;align-items:flex-end;gap:3px}.pc-tier-badge{color:#d4a843;font-size:10px;letter-spacing:1px}.pc-cat-badge{padding:1px 6px;border-radius:8px;background:rgba(255,255,255,.06);color:rgba(255,255,255,.4);font-size:9px}.pc-name-lg{padding:4px 12px 0;font-size:16px;font-weight:800;color:#fff}.pc-desc-lg{padding:2px 12px;font-size:11px;color:rgba(255,255,255,.4)}.pc-stats-row{display:flex;gap:16px;padding:8px 12px 4px}.pc-stat{display:flex;flex-direction:column}.pc-stat-label{font-size:9px;color:rgba(255,255,255,.3);margin-bottom:1px}.pc-stat-val{font-size:14px;font-weight:700;color:#fff}.pc-stat-val.green{color:#6bcb77}.pc-quality-bar{display:flex;height:8px;margin:4px 12px 0;border-radius:4px;overflow:hidden}.pc-qb-seg{cursor:help;transition:opacity .2s}.pc-qb-seg:hover{opacity:.8}.pc-actions-lg{display:flex;align-items:center;gap:6px;padding:8px 12px 12px}.pc-btn-minus,.pc-btn-plus{width:28px;height:28px;border:1px solid rgba(255,255,255,.1);border-radius:6px;background:rgba(255,255,255,.04);color:#fff;font-size:16px;cursor:pointer;font-family:inherit;display:flex;align-items:center;justify-content:center}.pc-btn-minus:hover,.pc-btn-plus:hover{background:rgba(255,255,255,.1)}.pc-qty-display{min-width:32px;text-align:center;font-size:16px;font-weight:700;color:#fff}.pc-btn-craft{flex:1;padding:8px 16px;border:none;border-radius:8px;background:linear-gradient(135deg,#d4a843,#b8860b);color:#fff;font-size:13px;font-weight:700;cursor:pointer;font-family:inherit;transition:all .2s;text-align:center}.pc-btn-craft:hover{transform:translateY(-1px);box-shadow:0 4px 16px rgba(212,168,67,.3)}.pc-btn-craft:disabled{background:rgba(255,255,255,.06);color:rgba(255,255,255,.2);transform:none;box-shadow:none;cursor:not-allowed}.craft-modal-lg{background:linear-gradient(180deg,#1a1a2e,#0d0d1a);border-radius:24px;width:420px;max-width:92vw;padding:32px 28px;text-align:center;animation:modal-in .3s ease}.craft-ok{border:2px solid #6bcb77;box-shadow:0 0 60px rgba(107,203,119,.15),0 0 120px rgba(107,203,119,.05)}.craft-no{border:2px solid #e53935;box-shadow:0 0 60px rgba(229,57,53,.15),0 0 120px rgba(229,57,53,.05)}.cml-top{font-size:26px;font-weight:900;margin-bottom:16px}.craft-ok .cml-top{color:#6bcb77}.craft-no .cml-top{color:#e53935}.cml-pill-show{display:flex;align-items:center;justify-content:center;gap:10px;margin-bottom:16px}.cml-pill-icon-lg{font-size:44px}.cml-pill-name-lg{font-size:22px;font-weight:900}.cml-stats-grid{display:flex;gap:12px;justify-content:center;margin-bottom:16px}.cml-stat-box{padding:10px 18px;background:rgba(255,255,255,.04);border-radius:10px;min-width:80px}.cml-stat-v{display:block;font-size:22px;font-weight:900;color:#d4a843}.cml-stat-l{display:block;font-size:10px;color:rgba(255,255,255,.3);margin-top:2px}.cml-fail-reason{font-size:15px;color:rgba(255,255,255,.4);margin-bottom:16px}.cml-btns{display:flex;gap:10px;justify-content:center}.cml-btn-retry{padding:10px 24px;border:1px solid #d4a843;border-radius:10px;background:rgba(212,168,67,.1);color:#d4a843;font-size:14px;font-weight:700;cursor:pointer;font-family:inherit;transition:all .2s}.cml-btn-retry:hover{background:#d4a843;color:#fff}.cml-btn-close{padding:10px 24px;border:1px solid rgba(255,255,255,.1);border-radius:10px;background:transparent;color:rgba(255,255,255,.5);font-size:14px;cursor:pointer;font-family:inherit;transition:all .2s}.cml-btn-close:hover{border-color:rgba(255,255,255,.3);color:#fff}.wiki-table{width:100%;border-collapse:collapse;margin:8px 0 16px;font-size:12px}.wiki-table th{background:rgba(212,168,67,.12);color:#d4a843;padding:6px 8px;text-align:center;font-weight:700;border:1px solid rgba(212,168,67,.1)}.wiki-table td{padding:4px 6px;text-align:center;border:1px solid rgba(255,255,255,.05);white-space:nowrap}.wiki-table td.tc{text-align:center}.wiki-table tr.highlight td{background:rgba(212,168,67,.1);color:#f0d878}.wiki-table tr:hover td{background:rgba(255,255,255,.04)}.wiki-note{color:rgba(255,255,255,.4);font-size:11px;margin:4px 0 12px}.wiki-formula{background:rgba(212,168,67,.08);border:1px solid rgba(212,168,67,.2);border-radius:8px;padding:8px 16px;color:#f0d878;font-size:13px;font-weight:600;text-align:center;margin:8px 0}html.light-mode .wiki-body{color:rgba(0,0,0,.7)}html.light-mode .wiki-table td{border-color:rgba(0,0,0,.05)}html.light-mode .wiki-table tr:hover td{background:rgba(0,0,0,.02)}html.light-mode .wiki-note{color:rgba(0,0,0,.4)}
.pigeon-modal{width:820px;max-width:96vw;background:linear-gradient(180deg,#12122a,#0d0d1a)!important;overflow:hidden!important}.pigeon-body{display:flex;height:540px;max-height:72vh}.pig-left{width:270px;min-width:210px;display:flex;flex-direction:column;overflow:hidden;border-right:1px solid rgba(212,168,67,.1);background:rgba(0,0,0,.2)}.pig-right{flex:1;display:flex;flex-direction:column;background:rgba(0,0,0,.08)}.pig-search-box{padding:12px;background:rgba(0,0,0,.2)}.pig-search-inp{width:100%;padding:8px 14px;border:1px solid rgba(255,255,255,.1);border-radius:20px;background:rgba(255,255,255,.05);color:#fff;font-size:12px;outline:none;font-family:inherit}.pig-search-inp:focus{border-color:#d4a843}.pig-sr-row{display:flex;align-items:center;gap:8px;padding:9px 12px;font-size:12px;border-bottom:1px solid rgba(255,255,255,.03)}.pig-sr-name{color:#fff;flex:1}.pig-sr-tag{font-size:10px;color:rgba(255,255,255,.25);background:rgba(255,255,255,.04);padding:1px 6px;border-radius:3px}.pig-sr-row button{padding:4px 12px;border:1px solid #d4a843;border-radius:8px;background:rgba(212,168,67,.1);color:#d4a843;font-size:11px;cursor:pointer;font-family:inherit}.pig-sr-row button:hover{background:#d4a843;color:#fff}.pig-group{margin:2px 0}.pig-group-head{display:flex;align-items:center;gap:6px;padding:11px 14px;font-size:13px;font-weight:600;color:rgba(255,255,255,.6);cursor:pointer;user-select:none;transition:all .15s;border-left:3px solid transparent}.pig-group-head:hover{background:rgba(255,255,255,.03);color:#d4a843;border-left-color:rgba(212,168,67,.3)}.pig-group-arrow{font-size:8px;transition:transform .2s;color:rgba(255,255,255,.25);margin-left:auto}.pig-group-arrow.open{transform:rotate(0deg)}.pig-group-arrow:not(.open){transform:rotate(-90deg)}.pig-badge{min-width:18px;height:18px;line-height:18px;text-align:center;border-radius:10px;background:#e53935;color:#fff;font-size:10px;font-weight:700;margin-left:auto;padding:0 5px}.pig-friend{display:flex;align-items:center;gap:10px;padding:8px 14px;cursor:pointer;transition:all .12s}.pig-friend:hover{background:rgba(212,168,67,.05)}.pig-friend.active{background:linear-gradient(90deg,rgba(212,168,67,.12),rgba(212,168,67,.04));border-right:3px solid #d4a843}.pig-f-avatar{width:36px;height:36px;border-radius:50%;background:linear-gradient(135deg,#d4a843,#b8860b);display:flex;align-items:center;justify-content:center;font-size:15px;font-weight:700;color:#fff;flex-shrink:0}.pig-f-info{flex:1}.pig-f-name{font-size:13px;color:#fff;display:flex;align-items:center;gap:6px}.pig-f-online{width:6px;height:6px;border-radius:50%;background:#666;flex-shrink:0}.pig-f-online.online{background:#4caf50;box-shadow:0 0 6px rgba(76,175,80,.5);animation:pig-dot-pulse 2s ease-in-out infinite}@keyframes pig-dot-pulse{0%{box-shadow:0 0 4px rgba(76,175,80,.3)}50%{box-shadow:0 0 8px rgba(76,175,80,.6)}}.pig-f-realm{font-size:10px;color:rgba(255,255,255,.25);margin-top:2px}.pig-empty-tip{text-align:center;padding:20px;font-size:12px;color:rgba(255,255,255,.15);font-style:italic}.pig-request-row{display:flex;align-items:center;gap:8px;padding:9px 14px}.pig-r-name{flex:1;font-size:12px;color:#fff}.pig-r-btns{display:flex;gap:4px}.pig-r-btns button{padding:4px 12px;border-radius:8px;font-size:11px;cursor:pointer;font-family:inherit;transition:all .15s;border:1px solid #6bcb77;background:rgba(107,203,119,.08);color:#6bcb77}.pig-r-btns button:hover{background:#6bcb77;color:#fff}.pig-r-btns .pig-r-reject{border-color:rgba(229,57,53,.3);background:rgba(229,57,53,.05);color:rgba(229,57,53,.6)}.pig-r-btns .pig-r-reject:hover{background:#e53935;color:#fff}.pig-chat-msgs{flex:1;overflow-y:auto;padding:12px 14px;display:flex;flex-direction:column;gap:10px}.pig-msg-bubble{display:flex;gap:8px;max-width:75%}.pig-msg-bubble.self{align-self:flex-end;flex-direction:row-reverse}.pig-msg-avatar{width:30px;height:30px;border-radius:50%;background:rgba(255,255,255,.1);display:flex;align-items:center;justify-content:center;font-size:12px;font-weight:700;color:rgba(255,255,255,.6);flex-shrink:0;align-self:flex-end}.pig-msg-bub-text{padding:8px 12px;border-radius:14px;font-size:13px;line-height:1.4;word-break:break-word}.pig-msg-bubble:not(.self) .pig-msg-bub-text{background:rgba(255,255,255,.08);color:#ddd;border-bottom-left-radius:4px}.pig-msg-bubble.self .pig-msg-bub-text{background:linear-gradient(135deg,rgba(212,168,67,.25),rgba(212,168,67,.12));color:#fff;border-bottom-right-radius:4px}.pig-chat-send{display:flex;gap:6px;padding:10px 14px;border-top:1px solid rgba(255,255,255,.06)}.pig-send-inp{flex:1;padding:8px 14px;border:1px solid rgba(255,255,255,.08);border-radius:20px;background:rgba(255,255,255,.04);color:#fff;font-size:13px;outline:none;font-family:inherit}.pig-send-btn{padding:8px 20px;border:none;border-radius:20px;background:linear-gradient(135deg,#d4a843,#b8860b);color:#fff;font-size:13px;font-weight:600;cursor:pointer;font-family:inherit}

.van-button--primary { --van-button-primary-background: linear-gradient(135deg,#8B6914,#d4a843) !important; }
.van-button--danger { --van-button-danger-background: linear-gradient(135deg,#8B2020,#ff4d4d) !important; }
.cult-btn-area { display:flex; gap:6px; flex-wrap:wrap; }
</style>

