// ============================================================
// 武器介绍文案 — 根据品阶+境界+属性批量生成
// ============================================================

const tierLore: Record<string, string[]> = {
  human:  ['凡铁初锻，质朴无华。', '人间工匠倾力所铸，虽无灵气，胜在坚实。', '凡人亦可挥舞，修仙之路由此始。', '粗砺无锋，然承千斤之力。'],
  yellow: ['百炼千锤，锋芒初显。', '淬火百次方成，已非凡器。', '灵光隐现，渐入佳境。', '历经锤炼，锋芒渐露。'],
  dark:   ['暗藏玄机，灵光内敛。', '玄铁寒铸，触之冰凉彻骨。', '寒气逼人，锋芒不可直视。', '玄之又玄，众妙之门。'],
  earth:  ['大地孕育千载，地脉淬炼而成。', '山川之灵所化，厚重如岳。', '地心深处淬炼万年，坚不可摧。', '大地为炉，星火为锤，锻此神兵。'],
  heaven: ['天道所钟，日月精华铸就。', '九天之上坠落的神兵，凡人不可直视。', '渡天劫而生，碎星辰之力化万千锋芒。', '仙品天成，非人间之物。'],
}

const realmLore: Record<number, string[]> = {
  1:  ['踏上修仙之路的第一件兵器。', '古朴厚重，承载着修士最初的梦想。'],
  2:  ['灵气初灌，焕发新生。', '练气之士可感其中灵气流转。'],
  3:  ['筑基之后，方显真正威力。', '灵力灌注，锋芒倍增。'],
  4:  ['金丹之力加持，锋芒无可匹敌。', '内蕴金丹之力，出鞘必有斩获。'],
  5:  ['元婴所炼，灵性自成。', '剑中有灵，与主共鸣。'],
  6:  ['化神之境，神兵有魂。', '历经百战而不毁，锋芒愈利。'],
  7:  ['炼虚之力灌注，可破虚空。', '虚实之间，锋芒无界。'],
  8:  ['合体之威，天地共鸣。', '人兵合一，所向披靡。'],
  9:  ['大乘之器，近于仙品。', '普照众生，慈悲为刃。'],
  10: ['历经九劫而不毁，见证飞升之道。', '渡劫而生，已非凡器，乃仙器也。'],
}

const statLore: Record<string, string[]> = {
  attack:  ['锋芒毕露，一击致命。', '锐不可当，破甲如纸。'],
  defense: ['坚如磐石，固若金汤。', '护体至宝，万法不侵。'],
  hp:      ['生命之力流转其中。', '佩戴者气血充盈。'],
  speed:   ['轻若鸿毛，快如闪电。', '风驰电掣，瞬息千里。'],
  critRate:['锋芒所指，必中要害。', '凌厉无匹，一击必杀。'],
  critDmg: ['暴击时威力倍增。', '致命一击，毁天灭地。'],
  dodge:   ['灵动飘逸，难以捉摸。', '身法如电，来去无踪。'],
  mpRegen: ['灵气如泉涌，绵绵不绝。', '回蓝如饮水，法力无穷。'],
}

function randomPick(arr: string[]): string {
  return arr[Math.floor(Math.random() * arr.length)]
}

export function generateLore(name: string, realm: number, tierKey: string, stats: Record<string, number>): string {
  const parts: string[] = []

  // 品阶描述 (pick 1)
  if (tierLore[tierKey]) parts.push(randomPick(tierLore[tierKey]))

  // 境界描述 (pick 1)
  if (realmLore[realm]) parts.push(randomPick(realmLore[realm]))

  // 属性描述 (pick from the highest stat)
  const entries = Object.entries(stats).filter(([_,v]) => v > 0).sort((a,b) => b[1] - a[1])
  if (entries.length > 0) {
    const topStat = entries[0][0]
    if (statLore[topStat]) parts.push(randomPick(statLore[topStat]))
  }

  // 随机打乱顺序并拼接
  const shuffled = parts.sort(() => Math.random() - 0.5)
  return shuffled.join(' ')
}
