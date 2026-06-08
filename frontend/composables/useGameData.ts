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
