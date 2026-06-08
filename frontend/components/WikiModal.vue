<template>
  <Teleport to="body">
    <div v-if="show" class="modal-overlay" @click.self="$emit('close')">
      <div class="wiki-modal">
        <div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:18px">百科</span><div class="top-bar-spacer"/><button class="modal-close" @click="$emit('close')">✕</button></div></header><div class="gold-divider"/>
        <div class="wiki-tabs">
          <button v-for="t in wikiTabs" :key="t.key" class="modal-tab" :class="{active:tab===t.key}" @click="tab=t.key">{{ t.label }}</button>
        </div>
        <div class="wiki-body">
          <div v-if="tab==='realm'">
            <h3>境界体系</h3>
            <table class="wiki-table"><thead><tr><th>境界</th><th>系数</th><th>基础率</th><th>下品×5</th><th>中品×15</th><th>上品×40</th><th>极品×100</th><th>攻击</th><th>防御</th><th>生命</th><th>灵力</th><th>速度</th><th>暴击%</th><th>暴伤%</th><th>闪避%</th><th>回蓝%</th><th>寿元</th></tr></thead><tbody>
              <tr v-for="(r,i) in wikiRealms" :key="i" :class="{highlight:i+1===realmId}">
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
          </div>
          <div v-if="tab==='root'">
            <h3>灵根类型与属性加成</h3>
            <table class="wiki-table"><thead><tr><th>灵根</th><th>攻击</th><th>防御</th><th>生命</th><th>灵力</th><th>暴击</th><th>暴伤</th><th>闪避</th><th>回蓝</th></tr></thead><tbody>
              <tr v-for="(r,i) in wikiRootBonuses" :key="i" :class="{highlight:rootNames[i]===playerSpiritName}">
                <td class="tc"><b>{{ r.name }}</b></td>
                <td>{{ r.atk>0?'+'+r.atk+'%':'—' }}</td><td>{{ r.def>0?'+'+r.def+'%':'—' }}</td>
                <td>{{ r.hp>0?'+'+r.hp+'%':'—' }}</td><td>{{ r.mp>0?'+'+r.mp+'%':'—' }}</td>
                <td>{{ r.cr>0?'+'+r.cr+'%':'—' }}</td><td>{{ r.cd>0?'+'+r.cd+'%':'—' }}</td>
                <td>{{ r.dg>0?'+'+r.dg+'%':'—' }}</td><td>{{ r.mr>0?'+'+r.mr+'%':'—' }}</td>
              </tr></tbody></table>
          </div>
          <div v-if="tab==='equip'">
            <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap;margin-bottom:16px">
              <span style="color:#d4a843;font-weight:700;font-size:13px">境界:</span>
              <button v-for="r in 10" :key="r" class="modal-tab" :class="{active:equipRealm===r}" @click="equipRealm=r">{{ realmNames[r] }}</button>
            </div>
            <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap;margin-bottom:16px">
              <span style="color:#d4a843;font-weight:700;font-size:13px">品阶:</span>
              <button v-for="t in tierInfo" :key="t.key" class="modal-tab" :class="{active:equipTier===t.key}" @click="equipTier=t.key" :style="{color:equipTier===t.key?t.color:''}">{{ t.name }}</button>
            </div>
            <table class="wiki-table"><thead><tr><th>部位</th><th>名称</th><th>主属性</th><th>品阶倍率</th><th>词缀</th></tr></thead><tbody>
              <tr v-for="s in equipSlotNames" :key="s">
                <td class="tc">{{ slotInfo[s].icon }} {{ slotInfo[s].name }}</td>
                <td>{{ equipNames[equipRealm]?.[equipTier]?.[s] || '—' }}</td>
                <td style="font-size:12px">{{ slotInfo[s].desc }}</td>
                <td :style="{color:(tierInfo.find(t=>t.key===equipTier)?.color||'#888')}">×{{ tierInfo.find(t=>t.key===equipTier)?.mult||0.5 }}</td>
                <td>{{ tierInfo.find(t=>t.key===equipTier)?.subStats||0 }}条</td>
              </tr></tbody></table>
            <p class="wiki-note">※ 武器攻击 = 境界基础攻击({{ realmBaseAttack[equipRealm]||'?' }}) × 0.6 × 品阶倍率。其他部位按境界基础值 × 品阶倍率提供对应属性。所有装备属性直接加算到角色面板，不参与灵根乘算。佩戴要求：装备境界 ≤ 角色境界。</p>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
defineProps<{ show: boolean; realmId: number; playerSpiritName: string }>()
defineEmits(['close'])
import { fmt, wikiRealms, wikiSpiritReqs, wikiRootBonuses, rootNames, qualityNames, wikiQuality } from '@/composables/useGameData'
import { equipNames, realmNames, realmBaseAttack, tierInfo, slotInfo, type EquipmentSlot } from '@/composables/useEquipmentData'

const tab = ref('realm')
const wikiTabs = [{ key: 'realm', label: '境界体系' }, { key: 'root', label: '灵根体系' }, { key: 'equip', label: '装备体系' }]
const equipRealm = ref(1)
const equipTier = ref('human')
const equipSlotNames: EquipmentSlot[] = ['weapon','robe','headgear','boots','necklace','ring']
</script>