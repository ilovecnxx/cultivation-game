<template>
  <div
    class="weapon-card"
    :class="{ selected, [`tier-${tierKey}`]: true, [`lv-${visual.level}`]: true }"
    :style="cardStyle"
    @click="$emit('select')"
  >
    <!-- 土豪金外框 -->
    <div class="wc-border"></div>
    <!-- 内框 -->
    <div class="wc-inner" :style="innerStyle">
      <!-- 品阶+名称 -->
      <div class="wc-name" v-html="colorizeName(equip.name, tierKey, equip.realm)"></div>
      <!-- 类型·境界·品阶 -->
      <div class="wc-meta">{{ slotIcon }} {{ typeName }} · {{ realmName }} · {{ tierName }}</div>
      <!-- 主属性 -->
      <div class="wc-stats">
        <span v-if="equip.attack" class="wc-stat atk">⚔️ {{ equip.attack }}</span>
        <span v-if="equip.defense" class="wc-stat def">🛡️ {{ equip.defense }}</span>
        <span v-if="equip.hp" class="wc-stat hp">❤️ {{ equip.hp }}</span>
        <span v-if="equip.mp" class="wc-stat mp">💙 {{ equip.mp }}</span>
        <span v-if="equip.speed" class="wc-stat spd">💨 {{ equip.speed }}</span>
        <span v-if="equip.element" class="wc-stat elem">{{ equip.element }}系</span>
      </div>
      <!-- 词缀 -->
      <div v-if="equip.substats?.length" class="wc-substats">
        <span v-for="s in equip.substats" :key="s.name" class="wc-sub">{{ s.name }} {{ s.display }}</span>
      </div>
      <!-- 文字介绍 -->
      <div class="wc-lore">{{ lore }}</div>
      <!-- 特效标记 -->
      <div v-if="visual.level >= 3" class="wc-shine"></div>
    </div>
    <!-- 操作按钮 -->
    <div v-if="showActions" class="wc-actions">
      <van-button size="small" type="primary" round @click.stop="$emit('equip')">装备</van-button>
      <van-button size="small" plain round @click.stop="$emit('detail')">详情</van-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { equipVisual, colorizeName, tierInfo, realmNames, slotInfo, type Equipment } from '@/composables/useEquipmentData'
import { generateLore } from '@/composables/useWeaponLore'
import { computed } from 'vue'

const props = defineProps<{ equip: Equipment; selected?: boolean; showActions?: boolean }>()
defineEmits(['select','equip','detail'])

const tierKey = computed(() => tierInfo.find(t => t.name === props.equip.tier)?.key || 'human')
const tierName = computed(() => props.equip.tier)
const realmName = computed(() => realmNames[props.equip.realm] || '')
const typeName = computed(() => slotInfo[props.equip.slot]?.name || '')
const slotIcon = computed(() => slotInfo[props.equip.slot]?.icon || '')

const visual = computed(() => equipVisual(props.equip.realm, tierKey.value))

// 卡片缩放: 人0.85/黄0.9/玄0.95/地1.0/天1.1
const scaleMap: Record<string, number> = { human:0.85, yellow:0.9, dark:0.95, earth:1.0, heaven:1.1 }
const scale = computed(() => scaleMap[tierKey.value] || 0.9)

const cardStyle = computed(() => ({
  transform: `scale(${scale.value})`,
  transition: 'transform .3s ease',
  cursor: props.showActions ? 'pointer' : 'default',
}))

// 内框: 白天黑底, 夜间白底
const innerStyle = computed(() => ({
  background: 'var(--wc-bg, #0d0d1a)',
  color: 'var(--wc-text, #e8e0d0)',
}))

const lore = computed(() => {
  const stats: Record<string,number> = {}
  if (props.equip.attack) stats.attack = props.equip.attack
  if (props.equip.defense) stats.defense = props.equip.defense
  if (props.equip.speed) stats.speed = props.equip.speed
  return generateLore(props.equip.name, props.equip.realm, tierKey.value, stats)
})
</script>

<style scoped>
.weapon-card {
  position: relative;
  display: inline-block;
  margin: 8px;
  vertical-align: top;
}
.weapon-card:hover { transform: scale(1.03) !important; z-index: 10; }
.weapon-card.selected .wc-border { border-color: #f0d878 !important; border-width: 4px !important; box-shadow: 0 0 20px rgba(212,168,67,.5) !important; }

.wc-border {
  position: absolute; inset: -3px;
  border: 3px solid #d4a843;
  border-radius: 14px;
  pointer-events: none;
  z-index: 1;
}
.wc-inner {
  position: relative; z-index: 2;
  padding: 12px 16px; border-radius: 12px;
  min-width: 200px; max-width: 280px;
}
.wc-name { font-size: 17px; margin-bottom: 4px; }
.wc-meta { font-size: 10px; color: rgba(255,255,255,.3); margin-bottom: 8px; }
.wc-stats { display: flex; gap: 6px; flex-wrap: wrap; margin-bottom: 6px; }
.wc-stat { font-size: 13px; font-weight: 700; padding: 2px 6px; border-radius: 4px; }
.wc-stat.atk { color: #ff6b6b; background: rgba(255,107,107,.08); }
.wc-stat.def { color: #4d96ff; background: rgba(77,150,255,.08); }
.wc-stat.hp { color: #6bcb77; background: rgba(107,203,119,.08); }
.wc-stat.spd { color: #64ffda; background: rgba(100,255,218,.08); }
.wc-stat.elem { color: #d4a843; background: rgba(212,168,67,.08); }
.wc-substats { display: flex; gap: 4px; flex-wrap: wrap; margin-bottom: 8px; }
.wc-sub { font-size: 10px; padding: 1px 6px; border-radius: 3px; background: rgba(212,168,67,.1); color: #d4a843; }
.wc-lore {
  font-size: 10px; color: rgba(255,255,255,.25);
  font-style: italic; line-height: 1.6;
  border-top: 1px solid rgba(255,255,255,.05);
  padding-top: 8px; margin-top: 6px;
}
.wc-shine {
  position: absolute; inset: 0; border-radius: 12px;
  background: linear-gradient(135deg, transparent 30%, rgba(212,168,67,.08) 50%, transparent 70%);
  animation: wc-shine 2s ease-in-out infinite;
  pointer-events: none; z-index: 3;
}
@keyframes wc-shine {
  0%, 100% { opacity: 0; transform: translateX(-100%); }
  50% { opacity: 1; transform: translateX(100%); }
}
.wc-actions { display: flex; gap: 8px; justify-content: center; margin-top: 8px; }

/* 白天/夜间变量 */
:root { --wc-bg: #0d0d1a; --wc-text: #e8e0d0; }
.light-mode { --wc-bg: #fff; --wc-text: #1a1a1a; }
</style>