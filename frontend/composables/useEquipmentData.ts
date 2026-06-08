// ============================================================
// 装备数据模型 — 10境界 × 5品阶 × 6部位 = 300件
// 武器攻击 = 境界基础攻击 × 0.6 × 品阶倍率 (不参与灵根乘算)
// ============================================================

export type EquipmentSlot = 'weapon'|'robe'|'headgear'|'boots'|'necklace'|'ring'

export const slotInfo: Record<EquipmentSlot, { name: string; icon: string }> = {
  weapon:   { name: '武器', icon: '⚔️' },
  robe:     { name: '法衣', icon: '👘' },
  headgear: { name: '头饰', icon: '👑' },
  boots:    { name: '鞋履', icon: '👢' },
  necklace: { name: '项链', icon: '📿' },
  ring:     { name: '戒指', icon: '💍' },
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

// ============================================================
// 300件装备名称表 (境界,品阶,部位)
// ============================================================
const equipNames: Record<number, Record<string, Record<EquipmentSlot, string>>> = {
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
  id: string
  name: string
  slot: EquipmentSlot
  realm: number
  tier: string
  tierMult: number
  attack: number
  substats: Array<{ name: string; display: string }>
  element: string
}

// 生成装备
export function generateEquip(realm: number, tierKey: string, slot: EquipmentSlot): Equipment {
  const tier = tierInfo.find(t => t.key === tierKey) || tierInfo[0]
  const baseAtk = realmBaseAttack[realm] || 10
  const attack = slot === 'weapon' ? Math.floor(baseAtk * 0.6 * tier.mult) : 0
  const name = equipNames[realm]?.[tierKey]?.[slot] || `${tier.name}·${slotInfo[slot].name}`

  const elements = ['金','木','水','火','土']
  const elem = elements[Math.floor(Math.random() * elements.length)]

  const subStatPool = ['暴击率','暴击伤害','闪避率','命中率','回蓝效率','速度']
  const substats: Equipment['substats'] = []
  const used = new Set<number>()
  for (let i = 0; i < tier.subStats; i++) {
    let idx: number
    do { idx = Math.floor(Math.random() * subStatPool.length) } while (used.has(idx))
    used.add(idx)
    const coeff = [0, 0.3, 0.45, 0.6, 0.8][i === 0 && tier.subStats > 0 ? 1 : Math.min(i + 1, 4)]
    substats.push({ name: subStatPool[idx], display: `+${(baseAtk * coeff * 0.01).toFixed(1)}%` })
  }

  return {
    id: 'eq_' + Date.now() + '_' + Math.random().toString(36).slice(2, 6),
    name, slot, realm, tier: tier.name, tierMult: tier.mult, attack, substats, element: elem,
  }
}
