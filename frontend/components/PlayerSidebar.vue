<template>
  <aside class="gh-sidebar">
    <!-- 位置 -->
    <div class="side-loc" v-if="currentLocInfo">
      <span class="sl-icon">{{ currentLocInfo.icon }}</span>
      <span class="sl-name">{{ currentLocInfo.name }}</span>
    </div>

    <!-- 玩家信息 -->
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
        <div class="profile-power-badge"><span class="pp-label">战力</span><span class="pp-val">{{ fmt(player.power) }}</span></div>
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

    <!-- 修炼面板 -->
    <div class="side-card side-cultivation">
      <div ref="yyWrapRef" class="cult-yy-wrap" :class="{meditating:player.isMeditating}" @mouseenter="$emit('showTooltip')" @mouseleave="$emit('hideTooltip')">
        <div class="cult-yy" :class="[player.gender==='female'?'female':'male']" :style="{animationDuration:yySpeed+'s'}" />
        <canvas ref="energyCanvas" v-if="player.isMeditating" class="energy-canvas" />
      </div>
      <div class="cult-realm-info">
        <span class="realm-tag">{{ player.realmName }}·{{ player.realmStage }}期</span>
        <span v-if="player.spiritName!=='无灵根'" class="root-tag">{{ player.spiritName }}·{{ player.qualityName }}</span>
      </div>
      <div class="cult-info">
        <div class="cult-bar-wrap"><span class="cult-bar-label">EXP</span><div class="cult-bar"><div class="cult-fill" :style="{width:cultExpPct+'%'}" /></div></div>
        <div class="cult-stats"><span>{{ fmt(player.spirit||0) }} / {{ fmt(player.maxSpirit||100) }}</span><span>+{{ player.cultRate||0 }}/s</span></div>
      </div>
      <div class="cult-btns">
        <button class="cult-btn primary" @click="$emit('toggleMeditation')">{{ player.isMeditating?'出关':'闭关' }}</button>
        <button class="cult-btn" :class="{danger:player.breakRate<60}" @click="$emit('doBreakthrough')">{{ player.realmStage<10?'突破':'突破 '+player.breakRate+'%' }}</button>
        <button class="cult-btn train-btn" @click="$emit('toggleTraining')">{{ training?'历练中':'历练' }}</button>
      </div>
      <div v-if="training" class="train-panel">
        <div class="train-info"><span>倍率</span><span>{{ trainMult }}×</span></div>
        <span>倍率: <button v-for="m in [1,2,5,10]" :key="m" class="mult-btn" :class="{on:trainMult===m}" @click="$emit('update:trainMult',m)" :disabled="training||isDead">×{{ m }}</button></span>
        <button class="manual-train-btn" @click="$emit('startManual')" :disabled="training||isDead||player.spiritSense<trainMult">⚡ 历练一次</button>
      </div>
      <div v-if="isDead" class="dead-warn">💀 角色已死亡，等待复活...</div>
    </div>

    <!-- 装备槽 -->
    <div class="side-card side-equip">
      <div class="equip-slots">
        <div v-for="s in equipSlots" :key="s.key" class="eq-slot" :title="getEquipTooltip(s.key)">{{ s.icon }}<span>{{ getEquipName(s.key) }}</span></div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { qualityColors, fmt } from '@/composables/useGameData'

const props = defineProps<{
  player: any; isDead?: boolean; training?: boolean; trainMult?: number;
  currentLocInfo?: any; yySpeed?: number; hpPct?: number; mpPct?: number;
  cultExpPct?: number;
}>()
defineEmits(['toggleMeditation','doBreakthrough','toggleTraining','startManual','showTooltip','hideTooltip','update:trainMult'])

const {token} = useAuth()
const equips = ref<any[]>([])
async function loadEquips() {
  try {
    const pid = localStorage.getItem('player_id')||'0'
    const r = await fetch('/api/v1/player/'+pid+'/equipment',{headers:{Authorization:'Bearer '+token.value}})
    const d = await r.json(); equips.value = d.data||[]
  } catch {}
}
onMounted(loadEquips)

const yyWrapRef = ref<HTMLElement|null>(null)
const energyCanvas = ref<HTMLCanvasElement|null>(null)

const equipSlots = [
  {key:'weapon',icon:'🗡️',name:'武器'},{key:'headgear',icon:'👑',name:'发冠'},
  {key:'robe',icon:'👘',name:'法袍'},{key:'boots',icon:'👢',name:'云靴'},
  {key:'necklace',icon:'📿',name:'项链'},{key:'ring',icon:'💍',name:'戒指'},
]

function getEquipName(slot: string): string { const eq = equips.value.find((e:any)=>e.slot===slot); return eq ? eq.name : equipSlots.find(s=>s.key===slot)?.name || slot }
function getEquipTooltip(slot: string): string { const eq = equips.value.find((e:any)=>e.slot===slot); if(!eq) return slotInfo[slot]?.name||slot; const lines=[eq.name]; if(eq.atk)lines.push('攻击:'+eq.atk); if(eq.def)lines.push('防御:'+eq.def); if(eq.hp)lines.push('生命:'+eq.hp); return lines.join('\n') }
</script>