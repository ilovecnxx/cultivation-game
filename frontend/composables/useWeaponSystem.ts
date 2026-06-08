// ============================================================
// 武器系统 — 天地玄黄人 五品阶
// 攻击力 = 境界基础攻击 × 武器系数 × 品阶倍率 (不参与灵根乘算)
// ============================================================

export const weaponTiers = [
  { name: '人阶', key: 'human', minRealm: 1, mult: 0.5, subStats: 0, color: '#888' },
  { name: '黄阶', key: 'yellow', minRealm: 2, mult: 1.0, subStats: 1, color: '#aaa' },
  { name: '玄阶', key: 'dark', minRealm: 3, mult: 1.8, subStats: 2, color: '#6bcb77' },
  { name: '地阶', key: 'earth', minRealm: 4, mult: 3.0, subStats: 3, color: '#4d96ff' },
  { name: '天阶', key: 'heaven', minRealm: 6, mult: 5.0, subStats: 4, color: '#ff6b6b' },
]

export const weaponTypes = [
  { name: '剑', icon: '🗡️', coeff: 1.0, desc: '攻守兼备' },
  { name: '刀', icon: '🔪', coeff: 1.1, desc: '侧重攻击' },
  { name: '戟', icon: '🔱', coeff: 0.9, desc: '命中较高' },
  { name: '弓', icon: '🏹', coeff: 0.8, desc: '暴击较高' },
  { name: '扇', icon: '🪭', coeff: 0.7, desc: '附加词条+1' },
]

export const elements = ['金','木','水','火','土'] as const
export type Element = typeof elements[number]

export const elementNames: Record<Element, string> = { 金: '金', 木: '木', 水: '水', 火: '火', 土: '土' }

export const realmBaseAttack: Record<number, number> = {
  1: 10, 2: 25, 3: 60, 4: 140, 5: 300, 6: 600, 7: 1200, 8: 2400, 9: 4500, 10: 8000,
}

export interface Weapon {
  id: string
  name: string
  tier: string
  type: string
  icon: string
  element: string
  attack: number      // 最终攻击力 = 基础 × 系数 × 倍率
  subStats: Array<{ name: string; value: number; display: string }>
  realmRequired: number
}

// 生成武器（模拟锻造结果）
export function generateWeapon(realmId: number, typeIdx: number, tierKey: string): Weapon {
  const tier = weaponTiers.find(t => t.key === tierKey) || weaponTiers[0]
  const wtype = weaponTypes[typeIdx] || weaponTypes[0]
  const baseAtk = realmBaseAttack[realmId] || 10
  const attack = Math.floor(baseAtk * wtype.coeff * tier.mult)
  const elem = elements[Math.floor(Math.random() * elements.length)]

  const subStatPool = [
    { name: '暴击率', key: 'critRate', gen: () => ({ value: Math.floor(baseAtk * 0.3 + Math.random() * baseAtk * 0.5), display: '+' + Math.floor(baseAtk * 0.03 + Math.random() * baseAtk * 0.05) / 10 + '%' }) },
    { name: '暴击伤害', key: 'critDmg', gen: () => ({ value: Math.floor(baseAtk * 0.3 + Math.random() * baseAtk * 0.5), display: '+' + Math.floor(baseAtk * 0.1 + Math.random() * baseAtk * 0.2) / 10 + '%' }) },
    { name: '闪避率', key: 'dodge', gen: () => ({ value: Math.floor(baseAtk * 0.2 + Math.random() * baseAtk * 0.4), display: '+' + Math.floor(baseAtk * 0.02 + Math.random() * baseAtk * 0.04) / 10 + '%' }) },
    { name: '命中率', key: 'hit', gen: () => ({ value: Math.floor(baseAtk * 0.2 + Math.random() * baseAtk * 0.4), display: '+' + Math.floor(baseAtk * 0.02 + Math.random() * baseAtk * 0.04) / 10 + '%' }) },
    { name: '回蓝效率', key: 'mpRegen', gen: () => ({ value: Math.floor(baseAtk * 0.3 + Math.random() * baseAtk * 0.5), display: '+' + Math.floor(baseAtk * 0.03 + Math.random() * baseAtk * 0.05) / 10 + '%' }) },
    { name: '速度', key: 'speed', gen: () => ({ value: Math.floor(baseAtk * 1.0 + Math.random() * baseAtk * 1.5), display: '+' + Math.floor(baseAtk * 0.2 + Math.random() * baseAtk * 0.3) }) },
  ]

  const subStats: Weapon['subStats'] = []
  const used = new Set<number>()
  for (let i = 0; i < tier.subStats; i++) {
    let idx: number
    do { idx = Math.floor(Math.random() * subStatPool.length) } while (used.has(idx) && used.size < subStatPool.length)
    used.add(idx)
    const pool = subStatPool[idx]
    const gen = pool.gen()
    subStats.push({ name: pool.name, value: gen.value, display: gen.display })
  }
  // 扇子额外 +1 词条
  if (wtype.name === '扇' && subStats.length < 5) {
    const idx = Math.floor(Math.random() * subStatPool.length)
    const gen = subStatPool[idx].gen()
    subStats.push({ name: subStatPool[idx].name, value: gen.value, display: gen.display })
  }

  // 弓额外暴击
  if (wtype.name === '弓') {
    subStats.push({ name: '暴击率', value: Math.floor(baseAtk * 0.5), display: '+' + (baseAtk * 0.05).toFixed(1) + '%' })
  }

  return {
    id: 'w_' + Date.now() + '_' + Math.random().toString(36).slice(2, 8),
    name: tier.name + '·' + wtype.name,
    tier: tier.name, type: wtype.name, icon: wtype.icon,
    element: elem, attack, subStats,
    realmRequired: tier.minRealm,
  }
}
