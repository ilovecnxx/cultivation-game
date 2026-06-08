// 修仙世界 — 游戏静态数据（地图、境界、灵根等）
// 从 game.vue 提取，减少 God Component 体积

export const realmNames: Record<number,string> = {1:'锻体',2:'练气',3:'筑基',4:'金丹',5:'元婴',6:'化神',7:'炼虚',8:'合体',9:'大乘',10:'渡劫'}
export const realmCoefs: Record<number,number> = {1:1,2:2,3:3,4:5,5:8,6:12,7:18,8:25,9:35,10:50}
export const rootMults: Record<number,number> = {0:0.7,1:1.0,2:1.3,3:1.6,4:2.0}
export const qualityNames: Record<number,string> = {0:'无品',1:'下品',2:'中品',3:'上品',4:'极品'}
export const qualityColors: Record<number,string> = {0:'#888',1:'#aaa',2:'#6bcb77',3:'#4d96ff',4:'#ff6b6b'}
export const pillQualityColors: Record<number,string> = {0:'#888',1:'#aaa',2:'#6bcb77',3:'#4d96ff',4:'#ff6b9e'}
export const rootNames: Record<number,string> = {0:'无灵根',1:'金灵根',2:'木灵根',3:'水灵根',4:'火灵根',5:'土灵根',6:'地灵根',7:'天灵根'}

export interface MapLoc { key:string; icon:string; name:string; desc:string; minRealm:number; minStage:number; monsters:string }
export interface MapRegion { name:string; icon:string; minRealm:number; maxRealm:number; locations:MapLoc[] }

export const mapRegions: MapRegion[] = [
  {name:'新手村',icon:'🏘️',minRealm:1,maxRealm:9,locations:[
    {key:'qys',icon:'⛰️',name:'青云山',desc:'山势险峻，灵气充沛，适合初入修仙的弟子修炼',minRealm:1,minStage:0,monsters:'野兔·山鸡·毒蛇·山贼'},
    {key:'hyc',icon:'🌊',name:'黑渊池',desc:'深不见底的黑暗水域，传说有上古妖兽潜伏',minRealm:2,minStage:0,monsters:'水蛇·暗影鱼·水鬼'},
    {key:'lhg',icon:'🌋',name:'烈火山',desc:'活火山区域，炽热的岩浆中蕴含火灵精华',minRealm:3,minStage:0,monsters:'火蜥蜴·熔岩怪·火凤雏'},
    {key:'mls',icon:'🌲',name:'迷踪林',desc:'千年古木遮天蔽日，迷阵重重',minRealm:4,minStage:0,monsters:'树妖·毒蜘蛛·幻影狐'},
    {key:'sgy',icon:'🏔️',name:'霜谷崖',desc:'终年积雪的极寒之地，冰属性修士的修炼圣地',minRealm:5,minStage:0,monsters:'雪狼·冰晶怪·寒冰蛟'},
    {key:'jgc',icon:'🌪️',name:'金光草原',desc:'一望无际的金色草原，暗藏杀机',minRealm:6,minStage:0,monsters:'金甲虫·沙暴蝎·草原狼王'},
    {key:'tyg',icon:'⛩️',name:'天渊谷',desc:'传说中连接天界的深渊裂谷',minRealm:7,minStage:0,monsters:'天兵残魂·虚空兽·堕天使'},
    {key:'ldh',icon:'🌀',name:'雷动海',desc:'永不停止的雷霆之海，雷属性修士朝圣之地',minRealm:8,minStage:0,monsters:'电鳗·雷鹰·雷霆巨人'},
    {key:'xjd',icon:'🧊',name:'玄冰洞',desc:'万古不化的极寒冰洞，藏有上古冰龙的秘密',minRealm:9,minStage:0,monsters:'冰龙幼崽·寒冰卫士·玄冰精灵'},
  ]},
  {name:'秘境',icon:'🏰',minRealm:3,maxRealm:9,locations:[
    {key:'yjg',icon:'🕯️',name:'远古遗迹',desc:'上古大能留下的秘宝之地',minRealm:3,minStage:5,monsters:'守卫傀儡·机关兽·守护兽'},
    {key:'hmg',icon:'🌑',name:'黑魔谷',desc:'魔气四溢的禁忌之地',minRealm:5,minStage:5,monsters:'魔化妖兽·邪修·魔将'},
    {key:'xht',icon:'🕳️',name:'星海通道',desc:'通往星辰大海的神秘通道',minRealm:7,minStage:5,monsters:'星际妖兽·星灵'},
    {key:'tjj',icon:'🚪',name:'天劫祭坛',desc:'渡劫飞升的终极试炼之地',minRealm:9,minStage:8,monsters:'天劫雷兽·天道化身'},
  ]},
]

export const menus: Array<{key:string;label:string;children?:Array<{key:string;label:string}>}> = [
  {key:'profession',label:'职业',children:[{key:'alchemy',label:'丹师'},
  {key:'craft',label:'器师'},{key:'array',label:'阵师'},{key:'tamer',label:'驯兽师'}]},
  {key:'cultivation',label:'修炼',children:[{key:'meditate',label:'打坐修炼'},{key:'breakthrough',label:'突破'},{key:'technique',label:'功法'},{key:'pill',label:'丹药'},{key:'tribulation',label:'渡劫'}]},
  {key:'combat',label:'战斗',children:[{key:'pve',label:'野外战斗'},{key:'dungeon',label:'副本'},{key:'arena',label:'竞技场'},{key:'tower',label:'爬塔'},{key:'world-boss',label:'世界Boss'}]},
  {key:'sect',label:'宗门',children:[{key:'my-sect',label:'我的宗门'},{key:'sect-list',label:'宗门列表'},{key:'sect-war',label:'宗门大战'}]},
  {key:'world',label:'世界',children:[{key:'world-map',label:'世界地图'},{key:'encounter',label:'奇遇'},{key:'fishing',label:'钓鱼'},{key:'ascend',label:'飞升'}]},
  {key:'trade',label:'交易',children:[{key:'buy',label:'购买'},{key:'sell',label:'出售'},{key:'auction',label:'拍卖'},{key:'auction-list',label:'拍卖列表'}]},
  {key:'social',label:'社交',children:[{key:'friend-list',label:'好友列表'},{key:'pigeon',label:'飞鸽传书'},{key:'master',label:'师徒'},{key:'daolv',label:'道侣'}]},
  {key:'inventory',label:'背包',children:[{key:'items',label:'全部物品'},{key:'equipment',label:'装备'},{key:'skills',label:'技能'},{key:'pets',label:'灵兽'},{key:'artifact',label:'法宝'},{key:'formation',label:'阵法'}]},
  {key:'ranking',label:'排行',children:[{key:'power-rank',label:'战力排行'},{key:'realm-rank',label:'境界排行'},{key:'wealth-rank',label:'财富排行'}]},
  {key:'chat',label:'传音',children:[{key:'world-chat',label:'世界频道'},{key:'sect-chat',label:'宗门频道'},{key:'private-chat',label:'私聊'}]},
  {key:'settings',label:'设置',children:[{key:'account',label:'账号设置'},{key:'display',label:'显示偏好'},{key:'about',label:'关于游戏'}]},
]

export const descs: Record<string,string> = {
  'my-sect':'创建或加入宗门','world-map':'探索广阔修仙世界','my-dongfu':'洞府是修士的修炼根基',
  'buy':'浏览坊市商品','auction-list':'参与竞拍稀有物品','meditate':'打坐修炼积累灵力',
  'breakthrough':'灵力充盈即可突破','items':'查看背包物品','my-pets':'灵兽是修士的伙伴',
  'my-artifact':'本命法宝与修士心血相连','power-rank':'全服战力排行榜','world-chat':'世界频道全服可见',
  'account':'修改账号信息','friend-list':'添加道友为好友','master':'拜师学艺或收徒传道',
  'daolv':'寻找道侣双修','tribulation':'突破需渡天劫','technique':'学习功法提升效率',
  'equipment':'穿戴装备提升属性','pills':'使用丹药恢复','pet-hatch':'孵化灵兽蛋','pet-release':'放生灵兽',
  'artifact-enhance':'淬炼法宝提升品阶','realm-rank':'全服境界排行榜','wealth-rank':'全服财富排行榜',
  'sect-chat':'宗门内部频道','private-chat':'与道友私聊','display':'调整主题偏好','about':'修仙世界v1.0'
}

export function fmt(n: number): string { return n >= 10000 ? (n/10000).toFixed(1)+'万' : n.toLocaleString() }

// ====== 百科数据 ======
export const wikiRealms = [
  {name:'锻体',coef:1,brk:70,atk:10,def:5,hp:100,mp:50,spd:100,cr:300,cd:15000,dg:200,mr:300,life:100,ss:100},
  {name:'练气',coef:2,brk:25,atk:25,def:12,hp:250,mp:125,spd:110,cr:500,cd:15500,dg:300,mr:400,life:150,ss:200},
  {name:'筑基',coef:3,brk:4,atk:60,def:30,hp:600,mp:300,spd:125,cr:800,cd:16000,dg:500,mr:500,life:200,ss:300},
  {name:'金丹',coef:4,brk:0.8,atk:140,def:70,hp:1400,mp:700,spd:140,cr:1200,cd:17000,dg:700,mr:600,life:300,ss:500},
  {name:'元婴',coef:5,brk:0.12,atk:300,def:150,hp:3000,mp:1500,spd:160,cr:1800,cd:18000,dg:1000,mr:800,life:500,ss:800},
  {name:'化神',coef:6,brk:0.015,atk:600,def:300,hp:6000,mp:3000,spd:185,cr:2500,cd:19000,dg:1300,mr:1000,life:800,ss:1200},
  {name:'炼虚',coef:7,brk:0.002,atk:1200,def:600,hp:12000,mp:6000,spd:210,cr:3200,cd:20000,dg:1600,mr:1200,life:1300,ss:1800},
  {name:'合体',coef:8,brk:0.0002,atk:2400,def:1200,hp:24000,mp:12000,spd:240,cr:4000,cd:21500,dg:2000,mr:1500,life:2000,ss:2500},
  {name:'大乘',coef:9,brk:0.00002,atk:4500,def:2250,hp:45000,mp:22500,spd:275,cr:5000,cd:23000,dg:2500,mr:1800,life:3500,ss:3500},
  {name:'渡劫',coef:10,brk:0.000002,atk:8000,def:4000,hp:80000,mp:40000,spd:310,cr:6000,cd:25000,dg:3000,mr:2200,life:5000,ss:5000},
]

export const wikiSpiritReqs = [
  [100,120,150,180,220,270,330,400,500],
  [600,700,850,1000,1200,1450,1750,2100,2600],
  [3000,3600,4300,5200,6300,7600,9200,11000,13500],
  [16000,19000,23000,28000,34000,41000,50000,60000,73000],
  [88000,105000,125000,150000,180000,215000,260000,310000,375000],
  [450000,540000,650000,780000,935000,1120000,1350000,1620000,1950000],
  [2340000,2810000,3370000,4050000,4860000,5830000,7000000,8400000,10080000],
  [12100000,14500000,17400000,20900000,25100000,30100000,36100000,43300000,52000000],
  [62400000,74900000,89900000,107900000,129500000,155400000,186500000,223800000,268600000],
  [322300000,386800000,464200000,557000000,668400000,802100000,962500000,1155000000,1386000000],
]

export const wikiRootBonuses = [
  {name:'金灵根',atk:12,def:0,hp:0,mp:0,cr:5,cd:0,dg:0,mr:0},
  {name:'木灵根',atk:0,def:0,hp:15,mp:0,cr:0,cd:0,dg:0,mr:10},
  {name:'水灵根',atk:0,def:0,hp:0,mp:12,cr:0,cd:0,dg:3,mr:15},
  {name:'火灵根',atk:8,def:0,hp:0,mp:0,cr:10,cd:10,dg:0,mr:0},
  {name:'土灵根',atk:0,def:15,hp:8,mp:0,cr:0,cd:0,dg:0,mr:0},
  {name:'地灵根',atk:8,def:8,hp:8,mp:8,cr:0,cd:0,dg:0,mr:0},
  {name:'天灵根',atk:12,def:10,hp:12,mp:10,cr:8,cd:0,dg:0,mr:8},
]

export const wikiQuality = [
  {name:'无品',speed:0.7,brk:1,attr:0.5,chance:8},
  {name:'下品',speed:1.0,brk:5,attr:1.0,chance:20},
  {name:'中品',speed:1.3,brk:15,attr:1.5,chance:40},
  {name:'上品',speed:1.6,brk:40,attr:2.0,chance:22},
  {name:'极品',speed:2.0,brk:100,attr:3.0,chance:8},
]
