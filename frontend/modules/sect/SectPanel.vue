<template>
<Teleport to="body">
<div v-if="visible" class="modal-overlay" @click.self="visible=false">
<div class="wiki-modal" style="width:860px;max-width:98vw;max-height:90vh">
<div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">🏯 宗门</span><div class="top-bar-spacer"/><button class="modal-close" @click="visible=false">✕</button></div></header><div class="gold-divider"/>

<!-- ====== 加载中 ====== -->
<div v-if="loading" class="wiki-body" style="text-align:center;padding:60px">
  <p style="font-size:40px">⏳</p>
  <p style="color:#d4a843">加载宗门数据...</p>
</div>

<!-- ====== 无宗门状态 ====== -->
<div v-else-if="!mySect" class="wiki-body">
  <div class="wiki-tabs">
    <button class="modal-tab" :class="{active:noSectTab==='create'}" @click="noSectTab='create'">🏯 创建宗门</button>
    <button class="modal-tab" :class="{active:noSectTab==='join'}" @click="noSectTab='join';searchSects()">🔍 加入宗门</button>
  </div>

  <!-- 创建宗门 -->
  <div v-if="noSectTab==='create'" style="max-width:500px;margin:0 auto">
    <h3 style="color:#d4a843">创建新宗门</h3>
    <p class="wiki-note">※ 创建宗门需要消耗 {{ createCost }} 灵石</p>
    <div style="display:flex;flex-direction:column;gap:12px;margin:16px 0">
      <input v-model="createForm.name" placeholder="宗门名称（2-8字）" maxlength="8" class="sect-input" />
      <input v-model="createForm.description" placeholder="宗门简介" maxlength="100" class="sect-input" />
      <input v-model="createForm.notice" placeholder="宗门公告" maxlength="200" class="sect-input" />
    </div>
    <button class="sect-btn primary" @click="doCreateSect" :disabled="!createForm.name||creating">
      {{ creating?'创建中...':'🏯 创建宗门 ('+createCost+'灵石)' }}
    </button>
    <p v-if="createError" style="color:#e53935;text-align:center;margin-top:8px">{{ createError }}</p>
  </div>

  <!-- 加入宗门 -->
  <div v-else>
    <div style="display:flex;gap:8px;margin-bottom:16px">
      <input v-model="searchKeyword" placeholder="搜索宗门名称..." class="sect-input" style="flex:1" @keyup.enter="searchSects" />
      <button class="sect-btn primary" @click="searchSects">🔍 搜索</button>
    </div>
    <div v-if="searchResults.length" style="display:flex;flex-direction:column;gap:8px">
      <div v-for="s in searchResults" :key="s.id" class="sect-card-row">
        <div style="flex:1">
          <div style="font-size:16px;font-weight:700;color:#d4a843">{{ s.name }}</div>
          <div style="font-size:11px;color:rgba(255,255,255,.4)">Lv.{{ s.level }} · {{ s.member_count }}/{{ s.max_members }}人 · 宗主:{{ s.leader_name }}</div>
          <div v-if="s.description" style="font-size:11px;color:rgba(255,255,255,.3);margin-top:2px">{{ s.description }}</div>
        </div>
        <button class="sect-btn primary" @click="doJoinSect(s)" :disabled="joining">{{ joining?'申请中...':'📝 申请加入' }}</button>
      </div>
    </div>
    <p v-else-if="searched" style="text-align:center;color:rgba(255,255,255,.3)">未找到宗门</p>
  </div>
</div>

<!-- ====== 已加入宗门状态 ====== -->
<div v-else class="wiki-body">
  <!-- 宗门头 -->
  <div class="sect-header">
    <div class="sect-header-left">
      <span style="font-size:40px">🏯</span>
      <div>
        <div style="font-size:22px;font-weight:900;color:#d4a843">{{ mySect.name }}</div>
        <div style="font-size:12px;color:rgba(255,255,255,.5)">Lv.{{ mySect.level }} · {{ rankLabel }} · 贡献: {{ fmtNum(myMember?.contribution||0) }}</div>
      </div>
    </div>
    <div class="sect-header-right">
      <div class="sect-stat-mini"><span>💰</span><span>{{ fmtNum(mySect.funds||0) }}</span></div>
      <div class="sect-stat-mini"><span>👥</span><span>{{ mySect.member_count }}/{{ mySect.max_members }}</span></div>
      <div class="sect-stat-mini"><span>⭐</span><span>{{ mySect.reputation||0 }}</span></div>
    </div>
  </div>

  <!-- 经验条 -->
  <div style="margin:8px 0 4px;display:flex;align-items:center;gap:8px">
    <span style="font-size:10px;color:#d4a843;white-space:nowrap">宗门经验</span>
    <div style="flex:1;height:6px;background:rgba(255,255,255,.06);border-radius:3px;overflow:hidden">
      <div :style="{width:sectExpPct+'%',height:'100%',background:'linear-gradient(90deg,#b8860b,#d4a843,#f0d878)',borderRadius:'3px',transition:'width .8s'}"></div>
    </div>
    <span style="font-size:10px;color:rgba(255,255,255,.3)">{{ fmtNum(mySect.experience||0) }}/{{ fmtNum(sectExpNeed) }}</span>
  </div>

  <!-- 公告 -->
  <div v-if="mySect.notice" class="sect-notice">📢 {{ mySect.notice }}</div>

  <!-- Tabs -->
  <div class="wiki-tabs">
    <button v-for="t in sectTabs" :key="t.key" class="modal-tab" :class="{active:tab===t.key}" @click="switchTab(t.key)">{{ t.label }}</button>
  </div>

  <!-- ====== 宗门总览 ====== -->
  <div v-if="tab==='overview'">
    <div class="sect-stats-grid">
      <div class="sect-stat-card"><div class="ssc-val">{{ fmtNum(mySect.funds||0) }}</div><div class="ssc-label">💰 宗门资金</div></div>
      <div class="sect-stat-card"><div class="ssc-val">{{ mySect.member_count }}/{{ mySect.max_members }}</div><div class="ssc-label">👥 成员数量</div></div>
      <div class="sect-stat-card"><div class="ssc-val">{{ mySect.reputation||0 }}</div><div class="ssc-label">⭐ 宗门声望</div></div>
      <div class="sect-stat-card"><div class="ssc-val">Lv.{{ mySect.level }}</div><div class="ssc-label">🏯 宗门等级</div></div>
    </div>
    <div class="sect-info-box">
      <div class="sib-row"><span class="sib-label">宗主</span><span class="sib-val">{{ mySect.leader_name }}</span></div>
      <div class="sib-row"><span class="sib-label">简介</span><span class="sib-val">{{ mySect.description||'暂无简介' }}</span></div>
      <div class="sib-row"><span class="sib-label">公告</span><span class="sib-val">{{ mySect.notice||'暂无公告' }}</span></div>
      <div class="sib-row"><span class="sib-label">创建时间</span><span class="sib-val">{{ mySect.created_at?.slice(0,10) }}</span></div>
    </div>
    <!-- 贡献排行(Top 10) -->
    <h4 style="color:#d4a843;margin:16px 0 8px">🏆 贡献排行</h4>
    <div v-if="contributionRank.length" class="rank-list">
      <div v-for="(m,i) in contributionRank" :key="i" class="rank-row" :class="{highlight:m.user_id===playerId}">
        <span class="rank-num">{{ i+1 }}</span>
        <span style="flex:1;font-weight:600">{{ m.user_id===mySect.leader_id?'👑 ':'' }}{{ m.user_id }}</span>
        <span class="rank-contrib">{{ fmtNum(m.contribution||0) }} 贡献</span>
        <span class="rank-badge">{{ rankName(m.rank) }}</span>
      </div>
    </div>
    <p v-else class="wiki-note">暂无贡献数据</p>
  </div>

  <!-- ====== 成员列表 ====== -->
  <div v-if="tab==='members'">
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:12px">
      <span style="color:#d4a843;font-weight:700">全部成员 ({{ contributionRank.length }})</span>
      <span style="font-size:11px;color:rgba(255,255,255,.3)">👑宗主 · 🔶长老 · ⭐精英 · 👤弟子</span>
    </div>
    <div v-if="contributionRank.length" class="rank-list">
      <div v-for="(m,i) in contributionRank" :key="i" class="rank-row" :class="{highlight:m.user_id===playerId}">
        <span class="rank-num">{{ i+1 }}</span>
        <span style="flex:1;font-weight:600">{{ m.user_id===mySect.leader_id?'👑 ':'' }}{{ m.user_id }}</span>
        <span class="rank-contrib">{{ fmtNum(m.contribution||0) }} 贡献</span>
        <span class="rank-badge">{{ rankName(m.rank) }}</span>
        <template v-if="canManage && m.user_id!==playerId">
          <button v-if="canManage && m.rank!=='leader'" class="sect-btn-sm" @click="doKick(m.user_id)" title="踢出">🚫</button>
          <select v-if="myMember?.rank==='leader' && m.rank!=='leader'" class="sect-select-sm" @change="doSetRank(m.user_id, ($event.target as HTMLSelectElement).value)" :value="m.rank">
            <option value="elder">🔶 长老</option>
            <option value="elite">⭐ 精英</option>
            <option value="member">👤 弟子</option>
          </select>
          <button v-if="myMember?.rank==='leader'" class="sect-btn-sm primary" @click="doTransferLeader(m.user_id)" title="转让宗主">👑</button>
        </template>
      </div>
    </div>
  </div>

  <!-- ====== 宗门仓库 ====== -->
  <div v-if="tab==='warehouse'">
    <div style="display:flex;gap:8px;margin-bottom:16px">
      <button class="sect-btn" :class="{active:whTab==='list'}" @click="whTab='list'">📦 仓库物品</button>
      <button class="sect-btn" :class="{active:whTab==='donate'}" @click="whTab='donate'">📥 捐献物品</button>
      <button class="sect-btn" :class="{active:whTab==='funds'}" @click="whTab='funds'">💰 捐献资金</button>
    </div>

    <!-- 仓库物品列表 -->
    <div v-if="whTab==='list'">
      <div v-if="whItems.length" style="display:flex;flex-direction:column;gap:6px">
        <div v-for="item in whItems" :key="item.id" class="wh-item-row">
          <span style="font-size:24px">{{ item.item_icon||'📦' }}</span>
          <div style="flex:1">
            <div style="font-weight:700;color:#fff">{{ item.item_name }} <span style="font-size:10px;color:rgba(255,255,255,.3)">x{{ item.quantity }}</span></div>
            <div style="font-size:10px;color:rgba(255,255,255,.3)">{{ item.item_type }} · 捐献者: {{ item.donor_name }}</div>
          </div>
          <div style="text-align:right;margin-right:8px">
            <div style="font-size:13px;color:#d4a843">💰 {{ fmtNum(item.price_spirit) }}</div>
            <div style="font-size:11px;color:#6bcb77">⭐ {{ fmtNum(item.price_contribution) }}贡献</div>
          </div>
          <button class="sect-btn primary" @click="buyWith='spirit';buyItemTarget=item;showBuyConfirm=true" style="font-size:11px" :disabled="item.donor_id===playerId">💰买</button>
          <button class="sect-btn" @click="buyWith='contribution';buyItemTarget=item;showBuyConfirm=true" style="font-size:11px;color:#6bcb77;border-color:#6bcb77" :disabled="item.donor_id===playerId">⭐兑换</button>
        </div>
      </div>
      <p v-else class="wiki-note">仓库暂无物品，快去捐献吧！</p>

      <!-- 购买确认弹窗 -->
      <Teleport to="body">
        <div v-if="showBuyConfirm" class="modal-overlay" @click.self="showBuyConfirm=false" style="z-index:4000">
          <div class="wiki-modal" style="width:360px;max-width:92vw">
            <div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:14px">确认购买</span><div class="top-bar-spacer"/><button class="modal-close" @click="showBuyConfirm=false">✕</button></div></header><div class="gold-divider"/>
            <div class="wiki-body" style="text-align:center">
              <p style="font-size:32px">{{ buyItemTarget?.item_icon||'📦' }}</p>
              <p style="color:#fff;font-size:16px;font-weight:700">{{ buyItemTarget?.item_name }}</p>
              <p v-if="buyWith==='spirit'" style="color:#d4a843">消耗 💰 {{ fmtNum(buyItemTarget?.price_spirit||0) }} 灵石</p>
              <p v-else style="color:#6bcb77">消耗 ⭐ {{ fmtNum(buyItemTarget?.price_contribution||0) }} 贡献</p>
              <div style="display:flex;gap:8px;justify-content:center;margin-top:16px">
                <button class="sect-btn" @click="showBuyConfirm=false">取消</button>
                <button class="sect-btn primary" @click="doBuyItem">确认购买</button>
              </div>
            </div>
          </div>
        </div>
      </Teleport>
    </div>

    <!-- 捐献物品 -->
    <div v-else-if="whTab==='donate'" style="max-width:500px;margin:0 auto">
      <h3 style="color:#d4a843">捐献物品到仓库</h3>
      <p class="wiki-note">※ 捐献物品可获得与市场价等值的宗门贡献</p>
      <div style="display:flex;flex-direction:column;gap:10px;margin:12px 0">
        <select v-model="donateForm.itemType" class="sect-input">
          <option value="weapon">⚔️ 武器</option>
          <option value="robe">👘 防具</option>
          <option value="headgear">👑 头饰</option>
          <option value="boots">👢 靴子</option>
          <option value="necklace">📿 项链</option>
          <option value="ring">💍 戒指</option>
          <option value="pill">💊 丹药</option>
          <option value="material">🪨 材料</option>
          <option value="other">📦 其他</option>
        </select>
        <input v-model="donateForm.itemName" placeholder="物品名称" class="sect-input" />
        <input v-model="donateForm.itemIcon" placeholder="图标(emoji)" class="sect-input" />
        <input v-model.number="donateForm.quantity" type="number" placeholder="数量" class="sect-input" min="1" />
        <input v-model.number="donateForm.marketValue" type="number" placeholder="市场价值(灵石)" class="sect-input" />
      </div>
      <p style="text-align:center;color:#6bcb77;font-size:13px">你将获得: ⭐ {{ fmtNum(donateForm.marketValue * donateForm.quantity) }} 宗门贡献</p>
      <button class="sect-btn primary" style="width:100%;margin-top:8px" @click="doDonateItem" :disabled="!donateForm.itemName||!donateForm.marketValue">
        📥 捐献
      </button>
    </div>

    <!-- 捐献资金 -->
    <div v-else style="max-width:400px;margin:0 auto;text-align:center">
      <h3 style="color:#d4a843">捐献灵石</h3>
      <p class="wiki-note">※ 捐献灵石可获得10%的宗门贡献（例: 捐1000灵石 → 得100贡献）</p>
      <div style="display:flex;gap:8px;margin:12px 0">
        <input v-model.number="fundsAmount" type="number" placeholder="灵石数量" class="sect-input" style="flex:1" />
        <button class="sect-btn primary" @click="doDonateFunds" :disabled="!fundsAmount||fundsAmount<=0">💰 捐献</button>
      </div>
      <p style="color:#6bcb77">将获得 ⭐ {{ fmtNum(Math.floor(fundsAmount/10)) }} 贡献</p>
    </div>
  </div>

  <!-- ====== 功法阁 ====== -->
  <div v-if="tab==='technique'">
    <div style="display:flex;gap:8px;margin-bottom:16px">
      <button class="sect-btn" :class="{active:techTab==='available'}" @click="techTab='available';loadTechniques()">📚 功法列表</button>
      <button class="sect-btn" :class="{active:techTab==='my'}" @click="techTab='my';loadMyTechniques()">📖 我的功法</button>
    </div>

    <!-- 可兑换功法 -->
    <div v-if="techTab==='available'">
      <div style="display:flex;align-items:center;gap:8px;margin-bottom:12px">
        <span style="color:#d4a843;font-weight:700;font-size:13px">功法阁 Lv.{{ mySect.level }} · 你的境界: {{ realmNames[playerRealm] }}</span>
      </div>
      <div v-if="techniques.length" class="tech-grid">
        <div v-for="t in techniques" :key="t.id" class="tech-card" :class="{owned: myTechIds.has(t.id)}">
          <div class="tech-card-top">
            <span style="font-size:32px">{{ t.icon||'📖' }}</span>
            <span class="tech-cat-badge">{{ catLabel(t.category) }}</span>
          </div>
          <div class="tech-name">{{ t.name }}</div>
          <div class="tech-desc">{{ t.description }}</div>
          <div class="tech-info">
            <span>境界: {{ realmNames[t.realm_required] }}</span>
            <span>最大 {{ t.max_level }} 层</span>
          </div>
          <div class="tech-cost">⭐ {{ fmtNum(t.cost_contribute) }} 贡献</div>
          <button v-if="myTechIds.has(t.id)" class="sect-btn" disabled style="width:100%;opacity:.5">✅ 已拥有</button>
          <button v-else class="sect-btn primary" style="width:100%" @click="doExchangeTechnique(t)" :disabled="exchanging">
            {{ exchanging?'兑换中...':'📚 兑换功法' }}
          </button>
        </div>
      </div>
      <p v-else class="wiki-note">暂无可兑换的功法</p>
    </div>

    <!-- 我的功法 -->
    <div v-else>
      <div v-if="myTechniques.length" class="tech-grid">
        <div v-for="mt in myTechniques" :key="mt.id" class="tech-card owned">
          <div class="tech-card-top">
            <span style="font-size:32px">{{ getTechIcon(mt.technique_id) }}</span>
            <span class="tech-lv-badge">Lv.{{ mt.level }}</span>
          </div>
          <div class="tech-name">{{ getTechName(mt.technique_id) }}</div>
          <div class="tech-desc">{{ getTechDesc(mt.technique_id) }}</div>
          <div class="tech-level-bar">
            <div v-for="lv in getTechMaxLevel(mt.technique_id)" :key="lv" class="tl-dot" :class="{filled:lv<=mt.level}"></div>
          </div>
          <button v-if="mt.level < getTechMaxLevel(mt.technique_id)" class="sect-btn primary" style="width:100%;font-size:11px" @click="doUpgradeTechnique(mt)">
            ⬆️ 升级到 {{ mt.level+1 }}层 (⭐{{ fmtNum(getTechUpgradeCost(mt)) }})
          </button>
          <button v-else class="sect-btn" disabled style="width:100%;opacity:.5">✨ 已满级</button>
        </div>
      </div>
      <p v-else class="wiki-note">你还没有兑换任何功法</p>
    </div>
  </div>

  <!-- ====== 技能树 ====== -->
  <div v-if="tab==='skills'">
    <p class="wiki-note">※ 宗门技能提升全体成员属性。长老以上可升级宗门技能，成员消耗贡献学习。</p>
    <div v-if="sectSkills.length" class="skill-grid">
      <div v-for="sk in sectSkills" :key="sk.id" class="skill-card">
        <div style="font-size:28px">{{ skillIcon(sk.effect_type) }}</div>
        <div class="skill-name">{{ sk.name }}</div>
        <div class="skill-desc">{{ sk.description }}</div>
        <div class="skill-level-bar">
          <div v-for="lv in sk.max_level" :key="lv" class="tl-dot" :class="{filled:lv<=sk.level}"></div>
        </div>
        <div class="skill-effect">效果: {{ skillEffectText(sk) }}</div>
        <div style="display:flex;gap:4px;margin-top:4px">
          <button v-if="canManage && sk.level < sk.max_level" class="sect-btn primary" style="flex:1;font-size:10px" @click="doUpgradeSkill(sk)">
            ⬆️ 升级 (⭐{{ fmtNum(sk.cost_per_level * (sk.level+1)) }})
          </button>
          <button class="sect-btn" style="flex:1;font-size:10px;color:#6bcb77;border-color:#6bcb77" @click="doLearnSkill(sk)">
            📚 学习 (⭐{{ fmtNum(sk.cost_per_level) }})
          </button>
        </div>
      </div>
    </div>
    <p v-else class="wiki-note">暂无宗门技能</p>
  </div>

  <!-- ====== 科技树 ====== -->
  <div v-if="tab==='tech'">
    <p class="wiki-note">※ 科技树消耗宗门资金+个人贡献升级。长老以上可升级。</p>
    <div v-if="sectTechs.length" style="display:flex;flex-direction:column;gap:8px">
      <div v-for="t in sectTechs" :key="t.branch" class="tech-branch-row">
        <div style="display:flex;align-items:center;gap:10px;flex:1">
          <span style="font-size:24px">{{ techBranchIcon(t.branch) }}</span>
          <div>
            <div style="font-weight:700;color:#fff">{{ techBranchName(t.branch) }}</div>
            <div style="font-size:10px;color:rgba(255,255,255,.3)">{{ techBranchDesc(t.branch) }}</div>
          </div>
        </div>
        <div class="tech-branch-bar">
          <div v-for="lv in 10" :key="lv" class="tl-dot" :class="{filled:lv<=t.level}"></div>
        </div>
        <span style="font-size:13px;font-weight:700;color:#d4a843;min-width:40px;text-align:center">Lv.{{ t.level }}/10</span>
        <button v-if="canManage && t.level < 10" class="sect-btn primary" style="font-size:10px" @click="doUpgradeTech(t)">
          升级
        </button>
      </div>
    </div>
    <p v-else class="wiki-note">加载科技树中...</p>
  </div>

  <!-- ====== 宗门任务 ====== -->
  <div v-if="tab==='missions'">
    <p class="wiki-note">※ 每日任务，完成可获得宗门贡献、经验和宗门资金奖励</p>
    <div v-if="sectMissions.length" style="display:flex;flex-direction:column;gap:8px">
      <div v-for="m in sectMissions" :key="m.id||m.mission_id" class="mission-card">
        <div style="flex:1">
          <div style="font-weight:700;color:#fff">{{ m.description||m.mission_type }}</div>
          <div style="font-size:11px;color:rgba(255,255,255,.4)">
            进度: {{ m.progress||0 }}/{{ m.requirement||'?' }} ·
            ⭐+{{ m.reward_contribution||0 }} · 💰+{{ m.reward_funds||0 }}
          </div>
        </div>
        <button v-if="m.completed && !m.claimed" class="sect-btn primary" style="font-size:11px" @click="doClaimMission(m)">🎁 领取</button>
        <button v-else-if="m.claimed" class="sect-btn" disabled style="opacity:.5">✅ 已领取</button>
        <span v-else style="font-size:11px;color:rgba(255,255,255,.3)">进行中</span>
      </div>
    </div>
    <p v-else class="wiki-note">暂无宗门任务</p>
  </div>

  <!-- ====== 宗门战 ====== -->
  <div v-if="tab==='war'">
    <p class="wiki-note">※ 宗门战赛季制：7天报名 → 21天战斗 → 2天休战</p>
    <div v-if="warSeason" class="sect-info-box">
      <div class="sib-row"><span class="sib-label">赛季</span><span class="sib-val">第 {{ warSeason.season }} 赛季</span></div>
      <div class="sib-row"><span class="sib-label">状态</span><span class="sib-val">{{ warStatusText }}</span></div>
      <div class="sib-row"><span class="sib-label">已报名</span><span class="sib-val">{{ warRegisteredCount }} 个宗门</span></div>
    </div>
    <div style="display:flex;gap:8px;margin-top:12px">
      <button class="sect-btn primary" @click="doRegisterWar" :disabled="warRegistered">⚔️ 报名参战</button>
      <button class="sect-btn" @click="loadWarRankings">📊 查看排名</button>
    </div>
    <div v-if="warRankings.length" style="margin-top:12px">
      <h4 style="color:#d4a843">🏆 赛季排名</h4>
      <div v-for="(r,i) in warRankings.slice(0,10)" :key="i" class="rank-row">
        <span class="rank-num">{{ i+1 }}</span>
        <span style="flex:1;font-weight:600">{{ r.sect_name }}</span>
        <span style="color:#d4a843">{{ r.score }} 分</span>
      </div>
    </div>
  </div>
</div>

<!-- Message Toast -->
<Teleport to="body">
  <div v-if="toastMsg" class="toast-popup" :class="toastType">{{ toastMsg }}</div>
</Teleport>

</div></div></Teleport>
</template>

<script setup lang="ts">
const visible = ref(false)
const loading = ref(false)
const tab = ref('overview')
const { token, playerId } = useAuth()
const props = defineProps<{ playerRealm?: number }>()
const playerRealm = computed(() => props.playerRealm || 1)

// ====== API helper ======
async function api(url: string, opts?: { method?: string; body?: any }) {
  const res = await fetch(url, {
    method: opts?.method||'GET',
    headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token.value },
    body: opts?.body ? JSON.stringify(opts.body) : undefined,
  })
  return res.json()
}

// ====== State ======
const noSectTab = ref('create')
const mySect = ref<any>(null)
const myMember = ref<any>(null)
const contributionRank = ref<any[]>([])
const searchKeyword = ref('')
const searchResults = ref<any[]>([])
const searched = ref(false)
const creating = ref(false)
const joining = ref(false)
const createError = ref('')
const createCost = 10000
const sectSkills = ref<any[]>([])
const sectTechs = ref<any[]>([])
const sectMissions = ref<any[]>([])
const whItems = ref<any[]>([])
const whTab = ref('list')
const techniques = ref<any[]>([])
const myTechniques = ref<any[]>([])
const techTab = ref('available')
const exchanging = ref(false)
const warSeason = ref<any>(null)
const warRegisteredCount = ref(0)
const warRegistered = ref(false)
const warRankings = ref<any[]>([])
const showBuyConfirm = ref(false)
const buyWith = ref('spirit')
const buyItemTarget = ref<any>(null)
const myTechIds = computed(() => new Set(myTechniques.value.map((mt: any) => mt.technique_id)))

const toastMsg = ref('')
const toastType = ref('success')

const createForm = reactive({ name: '', description: '', notice: '' })
const donateForm = reactive({ itemType: 'weapon', itemName: '', itemIcon: '', quantity: 1, marketValue: 1000 })
const fundsAmount = ref(1000)

const realmNames: Record<number,string> = {1:'锻体',2:'练气',3:'筑基',4:'金丹',5:'元婴',6:'化神',7:'炼虚',8:'合体',9:'大乘',10:'渡劫'}

const sectTabs = [
  { key: 'overview', label: '📋 总览' },
  { key: 'members', label: '👥 成员' },
  { key: 'warehouse', label: '📦 仓库' },
  { key: 'technique', label: '📚 功法阁' },
  { key: 'skills', label: '⚡ 技能' },
  { key: 'tech', label: '🔬 科技' },
  { key: 'missions', label: '📋 任务' },
  { key: 'war', label: '⚔️ 宗门战' },
]

const rankLabel = computed(() => {
  const m = myMember.value; if (!m) return ''
  const map: Record<string,string> = { leader: '👑 宗主', elder: '🔶 长老', elite: '⭐ 精英', member: '👤 弟子' }
  return map[m.rank] || m.rank
})
const canManage = computed(() => myMember.value?.rank === 'leader' || myMember.value?.rank === 'elder')

const sectExpNeed = computed(() => Math.floor(1000 * Math.pow(mySect.value?.level||1, 1.5)))
const sectExpPct = computed(() => {
  if (!mySect.value) return 0
  return Math.min(100, Math.floor((mySect.value.experience||0) / sectExpNeed.value * 100))
})

const warStatusText = computed(() => {
  if (!warSeason.value) return '未知'
  // Simplified: check status from season data
  const now = Date.now()
  const s = warSeason.value
  return s.status || 'pending'
})

function fmtNum(n: number) { return n >= 10000 ? (n/10000).toFixed(1)+'万' : String(n||0) }
function rankName(r: string) {
  const map: Record<string,string> = { leader:'👑宗主', elder:'🔶长老', elite:'⭐精英', member:'👤弟子' }
  return map[r]||r
}

// ====== Toast ======
let toastTimer: any
function toast(msg: string, type = 'success') {
  toastMsg.value = msg; toastType.value = type
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { toastMsg.value = '' }, 2500)
}

// ====== Load sect data ======
async function loadMySect() {
  loading.value = true
  try {
    const res = await api('/api/v1/sect/my?user_id=' + playerId.value)
    if (res.sect) {
      mySect.value = res.sect
      myMember.value = res.member
      await Promise.all([
        loadContributionRank(),
        loadSectSkills(),
        loadSectTechs(),
        loadSectMissions(),
        loadWarehouse(),
        loadWarSeason(),
      ])
      tab.value = 'overview'
    } else {
      mySect.value = null
      myMember.value = null
    }
  } catch(e) { mySect.value = null }
  loading.value = false
}

async function loadContributionRank() {
  try {
    const res = await api('/api/v1/sect/contribution-rank?sect_id=' + mySect.value.id + '&limit=50')
    contributionRank.value = res.data || []
  } catch(e) {}
}

async function loadSectSkills() {
  try {
    const res = await api('/api/v1/sect/skills?sect_id=' + mySect.value.id)
    sectSkills.value = res.data || []
  } catch(e) {}
}

async function loadSectTechs() {
  try {
    const res = await api('/api/v1/sect/tech/list?sect_id=' + mySect.value.id)
    sectTechs.value = res.data || []
  } catch(e) {}
}

async function loadSectMissions() {
  try {
    const res = await api('/api/v1/sect/mission/list', {
      method: 'POST', body: { sect_id: mySect.value.id, user_id: playerId.value }
    })
    sectMissions.value = res.data || []
  } catch(e) {}
}

async function loadWarehouse() {
  try {
    const res = await api('/api/v1/sect/warehouse/list?sect_id=' + mySect.value.id)
    whItems.value = res.data || []
  } catch(e) {}
}

async function loadTechniques() {
  try {
    const res = await api('/api/v1/sect/technique/list?sect_id=' + mySect.value.id + '&realm=' + playerRealm.value)
    techniques.value = res.data || []
    await loadMyTechniques()
  } catch(e) {}
}

async function loadMyTechniques() {
  try {
    const res = await api('/api/v1/sect/technique/my?user_id=' + playerId.value)
    myTechniques.value = res.data || []
  } catch(e) {}
}

async function loadWarSeason() {
  try {
    const res = await api('/api/v1/sect/war/season')
    if (res.data) {
      warSeason.value = res.data.season
      warRegisteredCount.value = res.data.registered_count||0
    }
  } catch(e) {}
}

async function loadWarRankings() {
  try {
    const res = await api('/api/v1/sect/war/rankings')
    warRankings.value = res.data || []
  } catch(e) {}
}

// ====== Actions ======
async function doCreateSect() {
  if (!createForm.name || createForm.name.length < 2) { createError.value = '名称至少2字'; return }
  creating.value = true; createError.value = ''
  try {
    const res = await api('/api/v1/sect/create', {
      method: 'POST',
      body: { name: createForm.name, description: createForm.description, notice: createForm.notice, leader_id: playerId.value, leader_name: '修仙者' }
    })
    if (res.data) { toast('宗门创建成功！'); await loadMySect() }
    else { createError.value = res.error || '创建失败' }
  } catch(e: any) { createError.value = e.message }
  creating.value = false
}

async function searchSects() {
  searched.value = true
  try {
    const res = await api('/api/v1/sect/search?keyword=' + (searchKeyword.value||''))
    searchResults.value = res.data || []
  } catch(e) { searchResults.value = [] }
}

async function doJoinSect(sect: any) {
  joining.value = true
  try {
    const res = await api('/api/v1/sect/join', {
      method: 'POST',
      body: { sect_id: sect.id, user_id: playerId.value, user_name: '修仙者', message: '申请加入宗门' }
    })
    if (res.message) toast('申请已提交！等待审批')
    else toast(res.error||'申请失败', 'error')
  } catch(e: any) { toast(e.message, 'error') }
  joining.value = false
}

async function doKick(targetId: string) {
  if (!confirm('确定踢出该成员？')) return
  try {
    const res = await api('/api/v1/sect/kick', {
      method: 'POST',
      body: { sect_id: mySect.value.id, operator_id: playerId.value, target_id: targetId }
    })
    if (res.message) { toast('已踢出'); loadContributionRank() }
    else toast(res.error, 'error')
  } catch(e: any) { toast(e.message, 'error') }
}

async function doSetRank(targetId: string, newRank: string) {
  try {
    const res = await api('/api/v1/sect/set-rank', {
      method: 'POST',
      body: { sect_id: mySect.value.id, operator_id: playerId.value, target_id: targetId, new_rank: newRank }
    })
    if (res.message) { toast('职位已更新'); loadContributionRank() }
    else toast(res.error, 'error')
  } catch(e: any) { toast(e.message, 'error') }
}

async function doTransferLeader(targetId: string) {
  if (!confirm('确定转让宗主之位？你将降为长老。')) return
  try {
    const res = await api('/api/v1/sect/transfer-leader', {
      method: 'POST',
      body: { sect_id: mySect.value.id, current_leader_id: playerId.value, new_leader_id: targetId }
    })
    if (res.message) { toast('已转让'); loadMySect() }
    else toast(res.error, 'error')
  } catch(e: any) { toast(e.message, 'error') }
}

async function doDonateItem() {
  try {
    const res = await api('/api/v1/sect/warehouse/donate', {
      method: 'POST',
      body: {
        sect_id: mySect.value.id, user_id: playerId.value, user_name: '修仙者',
        ...donateForm
      }
    })
    if (res.data) { toast('捐献成功！'); loadWarehouse(); loadContributionRank(); loadMySect() }
    else toast(res.error||'捐献失败', 'error')
  } catch(e: any) { toast(e.message, 'error') }
}

async function doDonateFunds() {
  try {
    const res = await api('/api/v1/sect/warehouse/donate-funds', {
      method: 'POST',
      body: { sect_id: mySect.value.id, user_id: playerId.value, amount: fundsAmount.value }
    })
    if (res.message) { toast('捐献成功！'); loadMySect(); loadContributionRank() }
    else toast(res.error||'捐献失败', 'error')
  } catch(e: any) { toast(e.message, 'error') }
}

async function doBuyItem() {
  showBuyConfirm.value = false
  if (!buyItemTarget.value) return
  try {
    const res = await api('/api/v1/sect/warehouse/buy', {
      method: 'POST',
      body: {
        item_id: buyItemTarget.value.id, sect_id: mySect.value.id,
        user_id: playerId.value, currency: buyWith.value
      }
    })
    if (res.message) { toast('购买成功！'); loadWarehouse(); loadMySect(); loadContributionRank() }
    else toast(res.error||'购买失败', 'error')
  } catch(e: any) { toast(e.message, 'error') }
}

async function doExchangeTechnique(t: any) {
  exchanging.value = true
  try {
    const res = await api('/api/v1/sect/technique/exchange', {
      method: 'POST',
      body: { sect_id: mySect.value.id, user_id: playerId.value, technique_id: t.id }
    })
    if (res.data) { toast('兑换成功！'); await loadMyTechniques(); loadContributionRank() }
    else toast(res.msg||'兑换失败', 'error')
  } catch(e: any) { toast(e.message, 'error') }
  exchanging.value = false
}

async function doUpgradeTechnique(mt: any) {
  try {
    const res = await api('/api/v1/sect/technique/upgrade', {
      method: 'POST',
      body: { sect_id: mySect.value.id, user_id: playerId.value, member_tech_id: mt.id }
    })
    if (res.data) { toast('功法升级成功！'); await loadMyTechniques(); loadContributionRank() }
    else toast(res.msg||'升级失败', 'error')
  } catch(e: any) { toast(e.message, 'error') }
}

async function doUpgradeSkill(sk: any) {
  try {
    const res = await api('/api/v1/sect/skill/upgrade', {
      method: 'POST',
      body: { sect_id: mySect.value.id, user_id: playerId.value, skill_id: sk.id }
    })
    if (res.data) { toast('技能升级成功！'); loadSectSkills(); loadMySect(); loadContributionRank() }
    else toast(res.error||'升级失败', 'error')
  } catch(e: any) { toast(e.message, 'error') }
}

async function doLearnSkill(sk: any) {
  try {
    const res = await api('/api/v1/sect/skill/learn', {
      method: 'POST',
      body: { sect_id: mySect.value.id, user_id: playerId.value, skill_id: sk.id }
    })
    if (res.data) { toast('学习成功！'); loadContributionRank() }
    else toast(res.error||'学习失败', 'error')
  } catch(e: any) { toast(e.message, 'error') }
}

async function doUpgradeTech(t: any) {
  try {
    const res = await api('/api/v1/sect/tech/upgrade', {
      method: 'POST',
      body: { sect_id: mySect.value.id, user_id: playerId.value, branch: t.branch }
    })
    if (res.data) { toast('科技升级成功！'); loadSectTechs(); loadMySect(); loadContributionRank() }
    else toast(res.msg||'升级失败', 'error')
  } catch(e: any) { toast(e.message, 'error') }
}

async function doClaimMission(m: any) {
  try {
    const res = await api('/api/v1/sect/mission/claim', {
      method: 'POST',
      body: { member_mission_id: m.id, sect_id: mySect.value.id, user_id: playerId.value }
    })
    if (res.message) { toast('奖励已领取！'); loadSectMissions(); loadContributionRank() }
    else toast(res.error||'领取失败', 'error')
  } catch(e: any) { toast(e.message, 'error') }
}

async function doRegisterWar() {
  try {
    const res = await api('/api/v1/sect/war/register', {
      method: 'POST',
      body: { sect_id: mySect.value.id, user_id: playerId.value, member_ids: [playerId.value] }
    })
    if (res.message) { toast('报名成功！'); warRegistered.value = true; loadWarSeason() }
    else toast(res.error||'报名失败', 'error')
  } catch(e: any) { toast(e.message, 'error') }
}

function switchTab(key: string) {
  tab.value = key
  if (key === 'warehouse') loadWarehouse()
  if (key === 'technique') { techTab.value = 'available'; loadTechniques() }
  if (key === 'skills') loadSectSkills()
  if (key === 'tech') loadSectTechs()
  if (key === 'missions') loadSectMissions()
  if (key === 'war') loadWarSeason()
  if (key === 'members') loadContributionRank()
}

// ====== Helper getters for technique data ======
function getTechIcon(id: string): string {
  const t = techniques.value.find((x: any) => x.id === id)
  return t?.icon || '📖'
}
function getTechName(id: string): string {
  const t = techniques.value.find((x: any) => x.id === id)
  return t?.name || '未知功法'
}
function getTechDesc(id: string): string {
  const t = techniques.value.find((x: any) => x.id === id)
  return t?.description || ''
}
function getTechMaxLevel(id: string): number {
  const t = techniques.value.find((x: any) => x.id === id)
  return t?.max_level || 5
}
function getTechUpgradeCost(mt: any): number {
  const t = techniques.value.find((x: any) => x.id === mt.technique_id)
  return t ? Math.floor(t.cost_contribute / 2 * (mt.level + 1)) : 1000
}
function catLabel(c: string): string {
  const m: Record<string,string> = { attack:'⚔️攻', defense:'🛡️防', support:'💚辅', secret:'🔮秘' }
  return m[c]||c
}
function skillIcon(e: string): string {
  if (e.includes('cultivation')) return '🧘'
  if (e.includes('combat')) return '⚔️'
  if (e.includes('gathering')) return '⛏️'
  if (e.includes('economy')) return '💰'
  return '✨'
}
function skillEffectText(sk: any): string {
  const pct = Math.round(sk.effect_value * sk.level * 100)
  return '+' + pct + '%'
}
function techBranchIcon(b: string): string {
  const m: Record<string,string> = { cultivation:'🧘', combat:'⚔️', gathering:'⛏️', economy:'💰', defense:'🛡️' }
  return m[b]||'📌'
}
function techBranchName(b: string): string {
  const m: Record<string,string> = { cultivation:'修炼加成', combat:'战斗加成', gathering:'采集加成', economy:'经济效益', defense:'防御阵地' }
  return m[b]||b
}
function techBranchDesc(b: string): string {
  const m: Record<string,string> = { cultivation:'修炼速度', combat:'攻击/防御', gathering:'掉落率', economy:'交易税率', defense:'宗门战防御' }
  return m[b]||''
}

// ====== Open ======
async function open(v: boolean) {
  if (!v) { visible.value = false; return }
  visible.value = true
  await loadMySect()
}

defineExpose({ open })
</script>

<style scoped>
.sect-input {
  width: 100%; padding: 10px 14px;
  border: 1px solid rgba(212,168,67,.15); border-radius: 8px;
  background: rgba(255,255,255,.04); color: #fff;
  font-size: 14px; outline: none; font-family: inherit; box-sizing: border-box;
}
.sect-input::placeholder { color: rgba(255,255,255,.2) }
.sect-input:focus { border-color: #d4a843 }
.sect-btn {
  padding: 8px 16px; border: 1px solid rgba(212,168,67,.15); border-radius: 8px;
  background: rgba(255,255,255,.04); color: rgba(255,255,255,.6);
  font-size: 13px; cursor: pointer; font-family: inherit; transition: all .2s;
}
.sect-btn:hover { border-color: #d4a843; color: #d4a843; background: rgba(212,168,67,.08) }
.sect-btn.primary { background: linear-gradient(135deg,rgba(212,168,67,.2),rgba(184,134,11,.1)); color: #d4a843; border-color: #d4a843; font-weight: 700 }
.sect-btn.primary:hover { background: #d4a843; color: #fff }
.sect-btn:disabled { opacity: .4; pointer-events: none }
.sect-btn.active { background: rgba(212,168,67,.15); color: #d4a843; border-color: #d4a843 }
.sect-btn-sm {
  padding: 2px 8px; border: 1px solid rgba(212,168,67,.15); border-radius: 4px;
  background: transparent; color: rgba(255,255,255,.4); font-size: 12px; cursor: pointer; font-family: inherit;
}
.sect-btn-sm:hover { border-color: #d4a843; color: #d4a843 }
.sect-btn-sm.primary { border-color: #d4a843; color: #d4a843 }
.sect-select-sm {
  padding: 2px 4px; border: 1px solid rgba(212,168,67,.15); border-radius: 4px;
  background: rgba(255,255,255,.04); color: #fff; font-size: 11px; font-family: inherit;
}

.sect-header { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 12px; margin-bottom: 12px }
.sect-header-left { display: flex; align-items: center; gap: 12px }
.sect-header-right { display: flex; gap: 12px }
.sect-stat-mini { display: flex; align-items: center; gap: 4px; font-size: 14px; color: #d4a843; font-weight: 700 }
.sect-notice {
  background: rgba(212,168,67,.06); border: 1px solid rgba(212,168,67,.1);
  border-radius: 6px; padding: 6px 12px; font-size: 12px; color: rgba(255,255,255,.5); margin-bottom: 8px
}

.sect-card-row {
  display: flex; align-items: center; gap: 12px;
  padding: 12px; background: rgba(255,255,255,.03);
  border: 1px solid rgba(255,255,255,.06); border-radius: 8px;
  transition: all .2s;
}
.sect-card-row:hover { border-color: rgba(212,168,67,.2); background: rgba(212,168,67,.04) }

.sect-stats-grid { display: grid; grid-template-columns: repeat(auto-fill,minmax(140px,1fr)); gap: 8px; margin-bottom: 12px }
.sect-stat-card {
  background: rgba(255,255,255,.03); border: 1px solid rgba(255,255,255,.06);
  border-radius: 8px; padding: 12px; text-align: center
}
.ssc-val { font-size: 22px; font-weight: 900; color: #d4a843 }
.ssc-label { font-size: 11px; color: rgba(255,255,255,.4); margin-top: 4px }

.sect-info-box { background: rgba(255,255,255,.02); border: 1px solid rgba(255,255,255,.05); border-radius: 8px; padding: 8px 12px }
.sib-row { display: flex; gap: 12px; padding: 6px 0; border-bottom: 1px solid rgba(255,255,255,.03); font-size: 13px }
.sib-row:last-child { border-bottom: none }
.sib-label { color: rgba(255,255,255,.3); min-width: 60px }
.sib-val { color: rgba(255,255,255,.7) }

.rank-list { display: flex; flex-direction: column; gap: 2px; max-height: 300px; overflow-y: auto }
.rank-row { display: flex; align-items: center; gap: 8px; padding: 6px 10px; border-radius: 6px; font-size: 13px; transition: all .15s }
.rank-row:hover { background: rgba(255,255,255,.03) }
.rank-row.highlight { background: rgba(212,168,67,.1); border: 1px solid rgba(212,168,67,.2) }
.rank-num { width: 24px; text-align: center; font-weight: 700; color: rgba(255,255,255,.4) }
.rank-contrib { color: #6bcb77; font-weight: 600; font-size: 12px }
.rank-badge { font-size: 10px; padding: 1px 8px; border-radius: 8px; background: rgba(212,168,67,.1); color: #d4a843 }

.wh-item-row {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 12px; background: rgba(255,255,255,.02);
  border: 1px solid rgba(255,255,255,.05); border-radius: 8px;
  transition: all .2s;
}
.wh-item-row:hover { border-color: rgba(212,168,67,.15) }

.tech-grid { display: grid; grid-template-columns: repeat(auto-fill,minmax(220px,1fr)); gap: 10px }
.tech-card {
  background: rgba(255,255,255,.02); border: 1px solid rgba(255,255,255,.06);
  border-radius: 10px; padding: 14px; transition: all .2s;
  display: flex; flex-direction: column; gap: 6px;
}
.tech-card:hover { border-color: rgba(212,168,67,.2); background: rgba(212,168,67,.04) }
.tech-card.owned { border-color: rgba(107,203,119,.2); background: rgba(107,203,119,.04) }
.tech-card-top { display: flex; justify-content: space-between; align-items: flex-start }
.tech-cat-badge { padding: 1px 8px; border-radius: 4px; background: rgba(255,255,255,.06); font-size: 10px; color: rgba(255,255,255,.4) }
.tech-lv-badge { padding: 2px 10px; border-radius: 10px; background: rgba(212,168,67,.15); color: #d4a843; font-size: 12px; font-weight: 700 }
.tech-name { font-size: 15px; font-weight: 800; color: #fff }
.tech-desc { font-size: 11px; color: rgba(255,255,255,.35); line-height: 1.4 }
.tech-info { display: flex; gap: 12px; font-size: 10px; color: rgba(255,255,255,.25) }
.tech-cost { font-size: 14px; font-weight: 700; color: #6bcb77 }
.tech-level-bar { display: flex; gap: 4px }
.tl-dot { width: 14px; height: 14px; border-radius: 50%; border: 1.5px solid rgba(255,255,255,.15); transition: all .2s }
.tl-dot.filled { background: #d4a843; border-color: #d4a843; box-shadow: 0 0 6px rgba(212,168,67,.3) }

.skill-grid { display: grid; grid-template-columns: repeat(auto-fill,minmax(240px,1fr)); gap: 10px }
.skill-card {
  background: rgba(255,255,255,.02); border: 1px solid rgba(255,255,255,.06);
  border-radius: 10px; padding: 14px; text-align: center;
  display: flex; flex-direction: column; gap: 6px; align-items: center;
}
.skill-name { font-size: 15px; font-weight: 800; color: #d4a843 }
.skill-desc { font-size: 11px; color: rgba(255,255,255,.35) }
.skill-level-bar { display: flex; gap: 4px }
.skill-effect { font-size: 12px; color: #6bcb77; font-weight: 600 }

.tech-branch-row {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 14px; background: rgba(255,255,255,.02);
  border: 1px solid rgba(255,255,255,.05); border-radius: 8px;
}
.tech-branch-bar { display: flex; gap: 2px }

.mission-card {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 14px; background: rgba(255,255,255,.02);
  border: 1px solid rgba(255,255,255,.05); border-radius: 8px;
}

.toast-popup {
  position: fixed; bottom: 60px; left: 50%; transform: translateX(-50%); z-index: 10000;
  padding: 10px 24px; border-radius: 20px; font-size: 14px; font-weight: 700;
  animation: modal-in .25s ease;
}
.toast-popup.success { background: #4caf50; color: #fff }
.toast-popup.error { background: #e53935; color: #fff }

.light-mode .sect-input { background: rgba(0,0,0,.02); color: #000; border-color: rgba(0,0,0,.1) }
.light-mode .sect-input::placeholder { color: rgba(0,0,0,.2) }
.light-mode .sect-card-row { background: rgba(0,0,0,.01); border-color: rgba(0,0,0,.04) }
.light-mode .tech-name { color: #000 }
.light-mode .sib-val { color: rgba(0,0,0,.6) }
</style>
