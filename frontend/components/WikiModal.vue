<template>
  <Teleport to="body">
    <div v-if="show" class="modal-overlay" @click.self="$emit('close')">
      <div class="wiki-modal">
        <div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:18px">百科</span><div class="top-bar-spacer"/><button class="modal-close" @click="$emit('close')">✕</button></div></header><div class="gold-divider"/>
        <div class="wiki-tabs">
          <button v-for="t in wikiTabs" :key="t.key" class="modal-tab" :class="{active:tab===t.key}" @click="tab=t.key">{{ t.label }}</button>
        </div>
        <div class="wiki-body">

          <!-- ====== 境界体系 ====== -->
          <div v-if="tab==='realm'">
            <h3>境界体系</h3>
            <table class="wiki-table"><thead><tr><th>境界</th><th>系数</th><th>突破率</th><th>攻击</th><th>防御</th><th>生命</th><th>灵力</th><th>速度</th><th>暴击</th><th>暴伤</th><th>闪避</th><th>回蓝</th><th>寿元</th></tr></thead><tbody>
              <tr v-for="(r,i) in wikiRealms" :key="i" :class="{highlight:i+1===realmId}">
                <td class="tc"><b>{{ r.name }}</b></td><td>{{ r.coef }}</td><td>{{ (r.brk*100).toFixed(2) }}%</td>
                <td>{{ r.atk }}</td><td>{{ r.def }}</td><td>{{ r.hp }}</td><td>{{ r.mp }}</td><td>{{ r.spd }}</td>
                <td>{{ (r.cr/100).toFixed(1) }}%</td><td>{{ (r.cd/100).toFixed(1) }}%</td>
                <td>{{ (r.dg/100).toFixed(1) }}%</td><td>{{ (r.mr/100).toFixed(1) }}%</td><td>{{ r.life }}年</td>
              </tr></tbody></table>
            <p class="wiki-note">※ 小期每级 +8% 固定属性 / +5% 百分比属性。速度只随境界变化。</p>
            <h4>修炼速度公式</h4>
            <p class="wiki-formula">每秒修为 = (10 + 境界系数 × 小期) × 灵根修炼倍率 × (1 + 丹药加成)</p>
            <h4>突破概率公式</h4>
            <p class="wiki-formula">基础成功率 = 50% + (当前修为/需求修为 - 1) × 100% · 最终成功率 = 基础 + 气运×0.5% + 丹药加成 - 心魔×2%（上限95%）</p>
            <h4>修为需求表</h4>
            <table class="wiki-table"><thead><tr><th>境界</th><th v-for="s in 9">{{ s }}→{{ s+1 }}期</th></tr></thead><tbody>
              <tr v-for="(r,i) in wikiSpiritReqs" :key="i"><td class="tc"><b>{{ wikiRealms[i]?.name }}</b></td><td v-for="v in r">{{ fmt(v) }}</td></tr>
            </tbody></table>
          </div>

          <!-- ====== 灵根体系 ====== -->
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
            <h4>灵根品质</h4>
            <table class="wiki-table"><thead><tr><th>品质</th><th>修炼倍率</th><th>属性倍率</th><th>概率</th></tr></thead><tbody>
              <tr v-for="q in wikiQuality" :key="q.name" :class="{highlight:qualityNames[1]===q.name}">
                <td class="tc"><b>{{ q.name }}</b></td><td>×{{ q.speed }}</td><td>×{{ q.attr }}</td><td>{{ q.chance }}%</td>
              </tr></tbody></table>
          </div>

          <!-- ====== 装备体系 ====== -->
          <div v-if="tab==='equip'">
            <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap;margin-bottom:12px">
              <span style="color:#d4a843;font-weight:700;font-size:13px">境界:</span>
              <button v-for="r in 10" :key="r" class="modal-tab" :class="{active:equipRealm===r}" @click="equipRealm=r">{{ realmNames[r] }}</button>
            </div>
            <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap;margin-bottom:12px">
              <span style="color:#d4a843;font-weight:700;font-size:13px">品阶:</span>
              <button v-for="t in tierInfo" :key="t.key" class="modal-tab" :class="{active:equipTier===t.key}" @click="equipTier=t.key" :style="{color:equipTier===t.key?t.color:''}">{{ t.name }}</button>
            </div>
            <table class="wiki-table" style="font-size:12px"><thead><tr><th>部位</th><th>名称</th><th>属性值</th><th>倍率</th><th>词缀</th></tr></thead><tbody>
              <tr v-for="s in equipSlotNames" :key="s">
                <td class="tc" style="padding:3px 6px">{{ slotInfo[s].icon }} {{ slotInfo[s].name }}</td>
                <td><span :style="{border:'2px solid '+getSlotVisual(s,equipRealm,equipTier).borderColor,padding:'2px 6px',borderRadius:'3px',fontWeight:'700',fontSize:'15px',display:'inline-block'}" v-html="colorizeName(equipNames[equipRealm]?.[equipTier]?.[s]||'—', equipTier, equipRealm)"></span></td>
                <td style="font-size:11px;color:#6bcb77">{{ getSlotStats(s, equipRealm, tierInfo.find(t=>t.key===equipTier)?.mult||0.5) }}</td>
                <td :style="{color:(tierInfo.find(t=>t.key===equipTier)?.color||'#888')}">×{{ tierInfo.find(t=>t.key===equipTier)?.mult||0.5 }}</td>
                <td>{{ tierInfo.find(t=>t.key===equipTier)?.subStats||0 }}条</td>
              </tr></tbody></table>
            <p class="wiki-note">※ 武器攻击 = 境界基础攻击 × 0.6 × 品阶倍率。其他部位按境界基础值提供对应属性。所有装备属性直接加算到角色面板，不参与灵根乘算。佩戴要求：装备境界 ≤ 角色境界。</p>
          </div>

          <!-- ====== 战斗属性 ====== -->
          <div v-if="tab==='combat'">
            <h3>战斗属性说明</h3>
            <table class="wiki-table"><thead><tr><th>属性</th><th>计算方式</th><th>作用</th></tr></thead><tbody>
              <tr><td>⚔️ 攻击</td><td>境界基础 × (1+小期×8%) × (1+灵根加成%) × 品质倍率 + 装备</td><td>决定物理伤害</td></tr>
              <tr><td>🛡️ 防御</td><td>同上</td><td>减免受到的伤害(减免=防御/2)</td></tr>
              <tr><td>❤️ 生命</td><td>同上</td><td>归零则死亡</td></tr>
              <tr><td>💙 灵力</td><td>同上</td><td>释放技能消耗</td></tr>
              <tr><td>💨 速度</td><td>境界基础(仅随大境界变化)</td><td>决定战斗先手</td></tr>
              <tr><td>💥 暴击</td><td>境界基础% × (1+灵根加成%) × 品质倍率 × (1+小期×5%) + 装备</td><td>概率造成暴伤</td></tr>
              <tr><td>💢 暴伤</td><td>同上</td><td>暴击时伤害倍率(基础2.0×)</td></tr>
              <tr><td>🎯 命中</td><td>同上（基础95%）</td><td>对抗闪避</td></tr>
              <tr><td>💨 闪避</td><td>同上（基础5%）</td><td>完全回避攻击</td></tr>
              <tr><td>🌀 修炼</td><td>基础0，装备/功法提供</td><td>加到修炼速度中</td></tr>
              <tr><td>💧 回蓝</td><td>境界基础% × 灵根加成 × 品质</td><td>打坐时每秒回蓝%</td></tr>
              <tr><td>⏳ 寿元</td><td>境界基础 × (1+小期×8%)</td><td>角色寿命上限</td></tr>
            </tbody></table>
            <h4>伤害公式</h4>
            <p class="wiki-formula">基础伤害 = 攻击 - 防御/2（最低1）</p>
            <p class="wiki-formula">最终伤害 = 基础伤害 × 技能倍率 × 五行克制(0.7~1.3) × 暴击(×2.0) × 境界压制(0.75~1.25)</p>
            <h4>五行克制</h4>
            <table class="wiki-table"><thead><tr><th></th><th>金</th><th>木</th><th>水</th><th>火</th><th>土</th></tr></thead><tbody>
              <tr><td><b>金</b></td><td>×1.0</td><td style="color:#6bcb77">×1.3</td><td style="color:#e53935">×0.7</td><td>×1.0</td><td>×1.0</td></tr>
              <tr><td><b>木</b></td><td style="color:#e53935">×0.7</td><td>×1.0</td><td>×1.0</td><td>×1.0</td><td style="color:#6bcb77">×1.3</td></tr>
              <tr><td><b>水</b></td><td>×1.0</td><td>×1.0</td><td>×1.0</td><td style="color:#6bcb77">×1.3</td><td style="color:#e53935">×0.7</td></tr>
              <tr><td><b>火</b></td><td style="color:#6bcb77">×1.3</td><td>×1.0</td><td style="color:#e53935">×0.7</td><td>×1.0</td><td>×1.0</td></tr>
              <tr><td><b>土</b></td><td>×1.0</td><td style="color:#e53935">×0.7</td><td style="color:#6bcb77">×1.3</td><td>×1.0</td><td>×1.0</td></tr>
            </tbody></table>
            <h4>战力公式</h4>
            <p class="wiki-formula">战力 = 攻击×1.5 + 防御×1.2 + 生命×0.15 + 灵力×0.1 + 速度×0.5 + 暴击%×3 + 暴伤%×0.15 + 闪避%×3 + 命中%×0.5 + 回蓝%×2</p>
          </div>

          <!-- ====== 先天属性 ====== -->
          <div v-if="tab==='innate'">
            <h3>📖 悟性</h3>
            <p class="wiki-formula">悟性 = 灵根品质基础值(随机) + 大境界加成</p>
            <table class="wiki-table"><thead><tr><th>品质</th><th>基础范围</th></tr></thead><tbody>
              <tr><td>无品</td><td>5~15</td></tr><tr><td>下品</td><td>15~35</td></tr><tr><td>中品</td><td>35~55</td></tr><tr><td>上品</td><td>55~75</td></tr><tr><td>极品</td><td>75~100</td></tr>
            </tbody></table>
            <p class="wiki-note">大境界加成：练气+5→筑基+10→金丹+20→元婴+35→化神+55→炼虚+80→合体+110→大乘+150→渡劫+200。作用：功法参悟速度。</p>
            <h3>🍀 气运</h3>
            <p class="wiki-formula">每日气运 = 随机(0 ~ 品质上限) + 灵根类型固定加成</p>
            <table class="wiki-table"><thead><tr><th>品质</th><th>随机范围</th></tr></thead><tbody>
              <tr><td>无品</td><td>0~30</td></tr><tr><td>下品</td><td>0~50</td></tr><tr><td>中品</td><td>0~70</td></tr><tr><td>上品</td><td>0~85</td></tr><tr><td>极品</td><td>0~100</td></tr>
            </tbody></table>
            <p class="wiki-note">灵根类型加成：天+10, 地+5, 木+3, 土+3, 水+2（上限100）。作用：掉落率 = 基础 × (1+气运/200)，奇遇加成 = 气运/100。</p>
            <h3>👁️ 神识</h3>
            <p class="wiki-formula">神识 = 境界基础 × 灵根品质倍率</p>
            <table class="wiki-table"><thead><tr><th>境界</th><th v-for="r in wikiRealms.slice(0,5)">{{ r.name }}</th></tr></thead><tbody>
              <tr><td>基础值</td><td v-for="r in wikiRealms.slice(0,5)">{{ r.ss }}</td></tr>
              <tr><td>× 无品(0.7)</td><td v-for="r in wikiRealms.slice(0,5)">{{ Math.floor(r.ss*0.7) }}</td></tr>
              <tr><td>× 中品(1.3)</td><td v-for="r in wikiRealms.slice(0,5)">{{ Math.floor(r.ss*1.3) }}</td></tr>
              <tr><td>× 极品(2.0)</td><td v-for="r in wikiRealms.slice(0,5)">{{ Math.floor(r.ss*2.0) }}</td></tr>
            </tbody></table>
            <p class="wiki-note">品质倍率：无品×0.7, 下品×1.0, 中品×1.3, 上品×1.6, 极品×2.0。作用：①副职成功率=神识/50% ②优良品质概率=神识/200。</p>
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
import { equipNames, realmNames, realmBaseAttack, realmBaseStats, tierInfo, slotInfo, equipVisual, colorizeName, type EquipmentSlot } from '@/composables/useEquipmentData'

const tab = ref('realm')
const wikiTabs = [
  { key: 'realm', label: '境界体系' },
  { key: 'root', label: '灵根体系' },
  { key: 'combat', label: '战斗属性' },
  { key: 'innate', label: '先天属性' },
  { key: 'equip', label: '装备体系' },
]

const equipRealm = ref(1)
const equipTier = ref('human')
const equipSlotNames: EquipmentSlot[] = ['weapon','robe','headgear','boots','necklace','ring']
function getSlotVisual(slot: EquipmentSlot, realm: number, tierKey: string) { return equipVisual(realm, tierKey) }
function getSlotStats(slot: EquipmentSlot, realm: number, mult: number): string {
  const b = realmBaseStats[realm]||realmBaseStats[1]; const m = mult
  switch(slot){
    case 'weapon': return '攻+'+Math.floor(b.atk*0.6*m)
    case 'robe': return '防+'+Math.floor(b.def*1.2*m)+' 血+'+Math.floor(b.hp*0.3*m)
    case 'headgear': return '灵+'+Math.floor(b.mp*0.8*m)+' 回蓝+'+Math.floor(5*m)
    case 'boots': return '速+'+Math.floor(b.spd*0.5*m)+' 闪+'+Math.floor(5*m)
    case 'necklace': return '血+'+Math.floor(b.hp*0.4*m)+' 回蓝+'+Math.floor(5*m)
    case 'ring': return '攻+'+Math.floor(b.atk*0.4*m)+' 暴伤+'+Math.floor(10*m)
    default: return '—'
  }
}
</script>