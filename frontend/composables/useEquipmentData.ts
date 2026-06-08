// ============================================================
// 装备数据模型 — 10境界 × 5品阶 × 6部位 = 300件
// 武器攻击 = 境界基础攻击 × 0.6 × 品阶倍率 (不参与灵根乘算)
// ============================================================

export type EquipmentSlot = 'weapon'|'robe'|'headgear'|'boots'|'necklace'|'ring'

export interface SlotStats { attack?: number; defense?: number; hp?: number; mp?: number; speed?: number; critRate?: number; critDmg?: number; dodge?: number; mpRegen?: number }

export const slotInfo: Record<EquipmentSlot, { name: string; icon: string; stats: SlotStats; desc: string }> = {
  weapon:   { name: '武器', icon: '⚔️', stats: {attack:1},                     desc: '攻击' },
  robe:     { name: '法衣', icon: '👘', stats: {defense:1,hp:1},              desc: '防御·生命' },
  headgear: { name: '头饰', icon: '👑', stats: {mp:1,mpRegen:1},              desc: '灵力·回蓝' },
  boots:    { name: '鞋履', icon: '👢', stats: {speed:1,dodge:1},             desc: '速度·闪避' },
  necklace: { name: '项链', icon: '📿', stats: {hp:1,mpRegen:1},              desc: '生命·回蓝' },
  ring:     { name: '戒指', icon: '💍', stats: {attack:1,critDmg:1},           desc: '攻击·暴伤' },
}

export const tierInfo = [
  { name: '人阶', key: 'human', mult: 0.5, subStats: 0, color: '#888' },
  { name: '黄阶', key: 'yellow', mult: 1.0, subStats: 1, color: '#aaa' },
  { name: '玄阶', key: 'dark', mult: 1.8, subStats: 2, color: '#6bcb77' },
  { name: '地阶', key: 'earth', mult: 3.0, subStats: 3, color: '#4d96ff' },
  { name: '天阶', key: 'heaven', mult: 5.0, subStats: 4, color: '#ff6b6b' },
]

export const realmNames = ['','锻体','练气','筑基','金丹','元婴','化神','炼虚','合体','大乘','渡劫']

export const realmBaseAttack: Record<number, number> = {
  1:10, 2:25, 3:60, 4:140, 5:300, 6:600, 7:1200, 8:2400, 9:4500, 10:8000,
}

// 全属性基础值（来自百科境界表 wikiRealms）
export const realmBaseStats: Record<number, { atk:number; def:number; hp:number; mp:number; spd:number; cr:number; cd:number; dg:number; mr:number }> = {
  1:  {atk:10, def:5,   hp:100,  mp:50,   spd:100, cr:300, cd:15000, dg:200, mr:300},
  2:  {atk:25, def:12,  hp:250,  mp:125,  spd:110, cr:500, cd:15500, dg:300, mr:400},
  3:  {atk:60, def:30,  hp:600,  mp:300,  spd:125, cr:800, cd:16000, dg:500, mr:500},
  4:  {atk:140, def:70, hp:1400, mp:700,  spd:140, cr:1200,cd:17000, dg:700, mr:600},
  5:  {atk:300, def:150,hp:3000, mp:1500, spd:160, cr:1800,cd:18000, dg:1000,mr:800},
  6:  {atk:600, def:300,hp:6000, mp:3000, spd:185, cr:2500,cd:19000, dg:1300,mr:1000},
  7:  {atk:1200,def:600,hp:12000,mp:6000, spd:210, cr:3200,cd:20000, dg:1600,mr:1200},
  8:  {atk:2400,def:1200,hp:24000,mp:12000,spd:240, cr:4000,cd:21500, dg:2000,mr:1500},
  9:  {atk:4500,def:2250,hp:45000,mp:22500,spd:275, cr:5000,cd:23000, dg:2500,mr:1800},
  10: {atk:8000,def:4000,hp:80000,mp:40000,spd:310, cr:6000,cd:25000, dg:3000,mr:2200},
}

// ============================================================
// 300件装备名称表 (境界,品阶,部位)
// ============================================================
export const equipNames: Record<number, Record<string, Record<EquipmentSlot, string>>> = {
  1: { // 锻体
    human:   {weapon:'人·凡铁剑',robe:'人·凡铁铠',headgear:'人·凡铁冠',boots:'人·凡铁靴',necklace:'人·凡铁坠',ring:'人·凡铁戒'},
    yellow:  {weapon:'黄·百炼刀',robe:'黄·百炼袍',headgear:'黄·百炼盔',boots:'黄·百炼履',necklace:'黄·百炼佩',ring:'黄·百炼环'},
    dark:    {weapon:'玄·寒铁斧',robe:'玄·寒铁甲',headgear:'玄·寒铁冠',boots:'玄·寒铁靴',necklace:'玄·寒铁符',ring:'玄·寒铁戒'},
    earth:   {weapon:'地·斩岩锤',robe:'地·斩岩铠',headgear:'地·斩岩盔',boots:'地·斩岩履',necklace:'地·斩岩坠',ring:'地·斩岩环'},
    heaven:  {weapon:'天·碎星戟',robe:'天·碎星甲',headgear:'天·碎星冠',boots:'天·碎星靴',necklace:'天·碎星佩',ring:'天·碎星戒'},
  },
  2: { // 练气
    human:   {weapon:'人·青竹剑',robe:'人·青竹衫',headgear:'人·青竹簪',boots:'人·青竹履',necklace:'人·青竹珠',ring:'人·青竹戒'},
    yellow:  {weapon:'黄·青锋刀',robe:'黄·青锋袍',headgear:'黄·青锋巾',boots:'黄·青锋靴',necklace:'黄·青锋坠',ring:'黄·青锋环'},
    dark:    {weapon:'玄·引气刺',robe:'玄·引气裳',headgear:'玄·引气冠',boots:'玄·引气履',necklace:'玄·引气佩',ring:'玄·引气戒'},
    earth:   {weapon:'地·风云刃',robe:'地·风云衣',headgear:'地·风云钗',boots:'地·风云靴',necklace:'地·风云符',ring:'地·风云环'},
    heaven:  {weapon:'天·灵泉剑',robe:'天·灵泉袍',headgear:'天·灵泉冠',boots:'天·灵泉履',necklace:'天·灵泉玉',ring:'天·灵泉戒'},
  },
  3: { // 筑基
    human:   {weapon:'人·磐石锤',robe:'人·磐石衣',headgear:'人·磐石冠',boots:'人·磐石靴',necklace:'人·磐石坠',ring:'人·磐石戒'},
    yellow:  {weapon:'黄·玄铁刀',robe:'黄·玄铁铠',headgear:'黄·玄铁盔',boots:'黄·玄铁履',necklace:'黄·玄铁符',ring:'黄·玄铁环'},
    dark:    {weapon:'玄·玉髓剑',robe:'玄·玉髓袍',headgear:'玄·玉髓簪',boots:'玄·玉髓靴',necklace:'玄·玉髓佩',ring:'玄·玉髓戒'},
    earth:   {weapon:'地·寒渊刃',robe:'地·寒渊甲',headgear:'地·寒渊冠',boots:'地·寒渊履',necklace:'地·寒渊珠',ring:'地·寒渊环'},
    heaven:  {weapon:'天·登云剑',robe:'天·登云衣',headgear:'天·登云冠',boots:'天·登云靴',necklace:'天·登云玉',ring:'天·登云戒'},
  },
  4: { // 金丹
    human:   {weapon:'人·乌金斧',robe:'人·乌金甲',headgear:'人·乌金冠',boots:'人·乌金靴',necklace:'人·乌金链',ring:'人·乌金环'},
    yellow:  {weapon:'黄·霜银钩',robe:'黄·霜银袍',headgear:'黄·霜银簪',boots:'黄·霜银履',necklace:'黄·霜银锁',ring:'黄·霜银戒'},
    dark:    {weapon:'玄·流金剑',robe:'玄·流金裳',headgear:'玄·流金冠',boots:'玄·流金靴',necklace:'玄·流金佩',ring:'玄·流金环'},
    earth:   {weapon:'地·丹心刺',robe:'地·丹心衣',headgear:'地·丹心冠',boots:'地·丹心履',necklace:'地·丹心符',ring:'地·丹心戒'},
    heaven:  {weapon:'天·九转剑',robe:'天·九转袍',headgear:'天·九转冠',boots:'天·九转靴',necklace:'天·九转坠',ring:'天·九转戒'},
  },
  5: { // 元婴
    human:   {weapon:'人·青木杖',robe:'人·青木衫',headgear:'人·青木簪',boots:'人·青木屐',necklace:'人·青木珠',ring:'人·青木戒'},
    yellow:  {weapon:'黄·玉罄剑',robe:'黄·玉罄衣',headgear:'黄·玉罄冠',boots:'黄·玉罄履',necklace:'黄·玉罄佩',ring:'黄·玉罄环'},
    dark:    {weapon:'玄·紫虚镜',robe:'玄·紫虚袍',headgear:'玄·紫虚钗',boots:'玄·紫虚靴',necklace:'玄·紫虚玉',ring:'玄·紫虚戒'},
    earth:   {weapon:'地·赤婴刺',robe:'地·赤婴裳',headgear:'地·赤婴冠',boots:'地·赤婴履',necklace:'地·赤婴符',ring:'地·赤婴环'},
    heaven:  {weapon:'天·羽化剑',robe:'天·羽化袍',headgear:'天·羽化冠',boots:'天·羽化靴',necklace:'天·羽化坠',ring:'天·羽化戒'},
  },
  6: { // 化神
    human:   {weapon:'人·化凡剑',robe:'人·化凡衫',headgear:'人·化凡巾',boots:'人·化凡履',necklace:'人·化凡符',ring:'人·化凡戒'},
    yellow:  {weapon:'黄·聚神刀',robe:'黄·聚神袍',headgear:'黄·聚神缨',boots:'黄·聚神靴',necklace:'黄·聚神佩',ring:'黄·聚神环'},
    dark:    {weapon:'玄·游神剑',robe:'玄·游神衣',headgear:'玄·游神冠',boots:'玄·游神履',necklace:'玄·游神玉',ring:'玄·游神戒'},
    earth:   {weapon:'地·元幡戟',robe:'地·元幡袍',headgear:'地·元幡盔',boots:'地·元幡靴',necklace:'地·元幡珠',ring:'地·元幡环'},
    heaven:  {weapon:'天·神照剑',robe:'天·神照衣',headgear:'天·神照冠',boots:'天·神照履',necklace:'天·神照坠',ring:'天·神照戒'},
  },
  7: { // 炼虚
    human:   {weapon:'人·空刃刀',robe:'人·空刃衫',headgear:'人·空刃冠',boots:'人·空刃靴',necklace:'人·空刃符',ring:'人·空刃戒'},
    yellow:  {weapon:'黄·虚灵剑',robe:'黄·虚灵袍',headgear:'黄·虚灵簪',boots:'黄·虚灵履',necklace:'黄·虚灵佩',ring:'黄·虚灵环'},
    dark:    {weapon:'玄·破空刺',robe:'玄·破空衣',headgear:'玄·破空冠',boots:'玄·破空靴',necklace:'玄·破空玉',ring:'玄·破空戒'},
    earth:   {weapon:'地·碎界斧',robe:'地·碎界甲',headgear:'地·碎界盔',boots:'地·碎界履',necklace:'地·碎界坠',ring:'地·碎界环'},
    heaven:  {weapon:'天·虚空剑',robe:'天·虚空裳',headgear:'天·虚空冠',boots:'天·虚空靴',necklace:'天·虚空珠',ring:'天·虚空戒'},
  },
  8: { // 合体
    human:   {weapon:'人·合气剑',robe:'人·合气衣',headgear:'人·合气冠',boots:'人·合气靴',necklace:'人·合气链',ring:'人·合气环'},
    yellow:  {weapon:'黄·融灵刀',robe:'黄·融灵袍',headgear:'黄·融灵巾',boots:'黄·融灵履',necklace:'黄·融灵符',ring:'黄·融灵戒'},
    dark:    {weapon:'玄·归元刃',robe:'玄·归元裳',headgear:'玄·归元冠',boots:'玄·归元靴',necklace:'玄·归元佩',ring:'玄·归元戒'},
    earth:   {weapon:'地·太虚戟',robe:'地·太虚袍',headgear:'地·太虚盔',boots:'地·太虚履',necklace:'地·太虚玉',ring:'地·太虚环'},
    heaven:  {weapon:'天·混元剑',robe:'天·混元衣',headgear:'天·混元冠',boots:'天·混元靴',necklace:'天·混元坠',ring:'天·混元戒'},
  },
  9: { // 大乘
    human:   {weapon:'人·凡渡杖',robe:'人·凡渡衫',headgear:'人·凡渡冠',boots:'人·凡渡履',necklace:'人·凡渡珠',ring:'人·凡渡戒'},
    yellow:  {weapon:'黄·慈航剑',robe:'黄·慈航衣',headgear:'黄·慈航簪',boots:'黄·慈航靴',necklace:'黄·慈航佩',ring:'黄·慈航环'},
    dark:    {weapon:'玄·普照剑',robe:'玄·普照袍',headgear:'玄·普照冠',boots:'玄·普照履',necklace:'玄·普照符',ring:'玄·普照戒'},
    earth:   {weapon:'地·大觉杵',robe:'地·大觉袍',headgear:'地·大觉冠',boots:'地·大觉靴',necklace:'地·大觉玉',ring:'地·大觉戒'},
    heaven:  {weapon:'天·彼岸剑',robe:'天·彼岸衣',headgear:'天·彼岸冠',boots:'天·彼岸履',necklace:'天·彼岸坠',ring:'天·彼岸戒'},
  },
  10: { // 渡劫
    human:   {weapon:'人·混元剑',robe:'人·混元衣',headgear:'人·混元冠',boots:'人·混元靴',necklace:'人·混元符',ring:'人·混元戒'},
    yellow:  {weapon:'黄·引雷刀',robe:'黄·引雷袍',headgear:'黄·引雷盔',boots:'黄·引雷履',necklace:'黄·引雷佩',ring:'黄·引雷环'},
    dark:    {weapon:'玄·化雷戟',robe:'玄·化雷甲',headgear:'玄·化雷冠',boots:'玄·化雷靴',necklace:'玄·化雷玉',ring:'玄·化雷戒'},
    earth:   {weapon:'地·御雷剑',robe:'地·御雷铠',headgear:'地·御雷盔',boots:'地·御雷履',necklace:'地·御雷坠',ring:'地·御雷环'},
    heaven:  {weapon:'天·天神剑',robe:'天·天神衣',headgear:'天·天神冠',boots:'天·天神靴',necklace:'天·天神链',ring:'天·天神戒'},
  },
}

export interface Equipment {
  id: string; name: string; slot: EquipmentSlot; realm: number
  tier: string; tierMult: number; element: string
  attack: number; defense: number; hp: number; mp: number
  speed: number; critRate: number; critDmg: number; dodge: number; hit: number; mpRegen: number
  substats: Array<{ name: string; display: string }>
}

// 各部位属性公式系数
const slotFormulas: Record<EquipmentSlot, (b: ReturnType<typeof getBase>, mult:number) => Partial<Equipment>> = {
  weapon:   (b, m) => ({ attack:  Math.floor(b.atk * 0.6 * m) }),
  robe:     (b, m) => ({ defense: Math.floor(b.def * 1.2 * m), hp: Math.floor(b.hp * 0.3 * m) }),
  headgear: (b, m) => ({ mp:      Math.floor(b.mp  * 0.8 * m), mpRegen: Math.floor(b.mr/100 * m) }),
  boots:    (b, m) => ({ speed:   Math.floor(b.spd * 0.5 * m), dodge: Math.floor(b.dg/100 * m) }),
  necklace: (b, m) => ({ hp:      Math.floor(b.hp  * 0.4 * m), mpRegen: Math.floor(b.mr/100 * m) }),
  ring:     (b, m) => ({ attack:  Math.floor(b.atk * 0.4 * m), critDmg: Math.floor(b.cd/100 * m) }),
}
function getBase(realm: number) { return realmBaseStats[realm] || realmBaseStats[1] }

// 生成装备
export function generateEquip(realm: number, tierKey: string, slot: EquipmentSlot): Equipment {
  const tier = tierInfo.find(t => t.key === tierKey) || tierInfo[0]
  const base = getBase(realm)
  const name = equipNames[realm]?.[tierKey]?.[slot] || `${tier.name}·${slotInfo[slot].name}`
  const stats = slotFormulas[slot](base, tier.mult)
  const elem = ['金','木','水','火','土'][Math.floor(Math.random()*5)]

  const subStatPool = ['暴击率','暴击伤害','闪避率','命中率','回蓝效率','速度']
  const substats: Equipment['substats'] = []
  const used = new Set<number>()
  for (let i = 0; i < tier.subStats; i++) {
    let idx: number
    do { idx = Math.floor(Math.random() * subStatPool.length) } while (used.has(idx))
    used.add(idx)
    const coeff = tier.subStats === 1 ? 0.3 : tier.subStats === 2 ? 0.45 : tier.subStats === 3 ? 0.6 : 0.8
    substats.push({ name: subStatPool[idx], display: `+${(base.atk * coeff * 0.01).toFixed(1)}%` })
  }

  return {
    id: 'eq_'+Date.now()+'_'+Math.random().toString(36).slice(2,6),
    name, slot, realm, tier: tier.name, tierMult: tier.mult, element: elem,
    attack: stats.attack||0, defense: stats.defense||0, hp: stats.hp||0, mp: stats.mp||0,
    speed: stats.speed||0, critRate: stats.critRate||0, critDmg: stats.critDmg||0,
    dodge: stats.dodge||0, hit: stats.hit||0, mpRegen: stats.mpRegen||0, substats,
  }
}

// ============================================================
// 装备视觉效果 — 境界+品阶双维度
// ============================================================
export interface EquipVisual {
  color: string
  borderColor: string
  glow: string
  box: string
  bg: string
  anim: string
  level: number
  style: string
}

const tierHue: Record<string, number> = { human:30, yellow:45, dark:130, earth:210, heaven:350 }
const tierIdx: Record<string, number> = { human:0, yellow:1, dark:2, earth:3, heaven:4 }

// 品阶字独立颜色（天/地/玄/黄/人）
export const tierColors: Record<string, string> = {
  human:  '#9e9e9e',  // 人·灰
  yellow: '#daa520',  // 黄·金
  dark:   '#4caf50',  // 玄·翠
  earth:  '#448aff',  // 地·蓝
  heaven: '#ff5252',  // 天·赤
}

// 给装备名加颜色: 前缀(天/地/玄/黄/人)=品阶色, 身体文字=境界色
export function colorizeName(name: string, tier: string, realm: number): string {
  const tierColor = tierColors[tier] || '#fff'
  const hue = realmHue[realm] || 0
  const ti = tierIdx[tier] || 0
  const sat = realm === 1 ? 0 : Math.min(100, 25 + ti * 18)
  const light = Math.min(90, 30 + ti * 12 + realm * 2)
  const bodyColor = `hsl(${hue}, ${sat}%, ${light}%)`
  const dot = name.indexOf('·')
  if (dot > 0) {
    return `<span style="color:${tierColor};font-weight:900">${name.slice(0,dot)}</span>·<span style="color:${bodyColor};font-weight:700">${name.slice(dot+1)}</span>`
  }
  return `<span style="color:${tierColor}">${name}</span>`
}

// 境界色相映射（字体/边框基础色）
const realmHue: Record<number, number> = {
  1:0,     // 锻体·黑
  2:120,   // 练气·绿
  3:270,   // 筑基·紫
  4:195,   // 金丹·天蓝
  5:25,    // 元婴·橙
  6:0,     // 化神·老红(深)
  7:330,   // 炼虚·粉
  8:0,     // 合体·红
  9:50,    // 大乘·黄
  10:40,   // 渡劫·金
}

export function equipVisual(realm: number, tier: string): EquipVisual {
  const hue = realmHue[realm] || 0
  const ti = tierIdx[tier] || 0
  const sat = realm === 1 ? 0 : Math.min(100, 25 + ti * 18)
  const light = Math.min(90, 30 + ti * 12 + realm * 2)
  // 文字色：境界色系
  const color = `hsl(${hue}, ${sat}%, ${light}%)`
  // 边框色：跟品阶字颜色一致（天=赤/地=蓝/玄=翠/黄=金/人=灰）
  const borderColor = tierColors[tier] || '#888'
  const lv = realm <= 2 ? 0 : realm <= 4 ? 1 : realm <= 6 ? 2 : realm <= 8 ? 3 : 4
  const glow = lv >= 1 ? `text-shadow:0 0 ${2+realm+ti}px ${color}` : ''
  const box  = lv >= 2 ? `box-shadow:0 0 ${4+realm+ti}px ${color}44` : ''
  const bg   = lv >= 3 ? `background:linear-gradient(135deg,${color}22,transparent)` : ''
  const anim = lv >= 4 ? `animation:equip-shimmer ${3-(realm-8)*0.5}s ease-in-out infinite` : ''
  const style = [glow, box, bg, anim].filter(Boolean).join(';')
  return { color, borderColor, glow, box, bg, anim, level: lv, style }
}
