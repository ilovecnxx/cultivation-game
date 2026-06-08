<template>
  <aside class="gh-sidebar">
    <div class="side-card side-profile">
      <div class="profile-top">
        <div class="profile-top-left">
          <div class="profile-name-row">
            <span class="side-avatar">{{ player.name?.[0] || '修' }}</span>
            <span class="side-name">{{ player.name || '修仙者' }}
              <span v-if="isDead" class="dead-skull">💀</span>
              <span class="gender-tag" :style="{color:player.gender==='female'?'#ff6b6b':'#4d96ff'}">{{ player.gender==='female'?'♀':'♂' }}</span>
            </span>
          </div>
          <div class="profile-tags">
            <span class="side-realm">{{ player.realmName }} {{ player.realmStage }}期</span>
            <span v-if="player.spiritName!=='无灵根'" class="side-spirit" :style="{color:qualityColors[player.rootQuality]||'#fff'}">{{ player.spiritName }} · {{ player.qualityName }}</span>
            <span v-else class="side-spirit" style="color:#666">灵根未觉醒</span>
          </div>
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
        <div class="pa-row"><span>💰 灵石</span><span>{{ player.gold }}</span></div>
        <div class="pa-row"><span>💎 仙玉</span><span>{{ player.jade }}</span></div>
        <div class="pa-row"><span>🍀 气运</span><span>{{ player.luck }}</span></div>
        <div class="pa-row"><span>⏳ 寿元</span><span>{{ player.lifespan }}年</span></div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
defineProps<{ player: any; isDead?: boolean }>()

const qualityColors: Record<number,string> = {0:'#888',1:'#aaa',2:'#6bcb77',3:'#4d96ff',4:'#ff6b6b'}

function fmt(n: number): string { return n >= 10000 ? (n/10000).toFixed(1)+'万' : n.toLocaleString() }
</script>