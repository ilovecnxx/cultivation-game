<template>
  <!-- PVE 弹窗 -->
  <Teleport to="body">
    <div v-if="pveReport" class="modal-overlay" @click.self="emit('update:pveReport',null);emit('update:pveRounds',[])">
      <div class="death-modal" :style="{borderColor:pveReport.won?'#6bcb77':'#e53935'}">
        <div class="dm-header" :style="{color:pveReport.won?'#6bcb77':'#e53935'}">{{ pveReport.won?'⚔️ 胜利！':'💀 战败' }}</div>
        <div v-if="pveReport.monster" style="font-size:16px;color:#ffd700;margin:8px 0">👹 {{ pveReport.monster }}期怪物</div>
        <div v-if="pveReport.won" style="font-size:14px;color:#6bcb77;margin-bottom:8px">🎁 +{{ pveReport.cult }}修为 +{{ pveReport.gold }}灵石</div>
        <div class="dm-battle" style="max-height:200px;text-align:left">
          <div v-for="r in pveRounds" :key="r.round" class="cr-round"><span>第{{ r.round }}回合:</span><span class="cr-dmg">造成 {{ r.player_dmg }} 伤害</span><span v-if="r.desc" class="cr-special">{{ r.desc }}</span><span class="cr-dmg">受到 {{ r.monster_dmg }} 伤害</span><span class="cr-hp">❤️{{ r.player_hp }} | 👹{{ r.monster_hp>0?r.monster_hp:'击败' }}</span></div>
        </div>
        <button class="map-enter-btn" style="margin-top:12px;font-size:14px;padding:8px 24px" @click="emit('update:pveReport',null);emit('update:pveRounds',[])">关闭</button>
      </div>
    </div>
  </Teleport>
  <!-- 奇遇弹窗 -->
  <Teleport to="body">
    <div v-if="encounterResult" class="modal-overlay">
      <div class="encounter-modal">
        <div class="em-icon">{{ encounterResult.icon }}</div>
        <div class="em-title">{{ encounterResult.title }}</div>
        <div class="em-desc">历练中偶然发现一处隐秘洞府，获得了洗炼灵根的机会！</div>
        <div class="em-change"><span class="em-old">{{ encounterResult.old }}</span> → <span class="em-new">{{ encounterResult.new }}</span></div>
        <div class="em-note">属性已自动更新，修炼速度和战斗属性全面提升</div>
        <button class="map-enter-btn" style="margin-top:12px;font-size:14px;padding:8px 24px" @click="emit('update:encounterResult',null)">收下</button>
      </div>
    </div>
  </Teleport>
  <!-- 死亡弹窗 -->
  <Teleport to="body">
    <div v-if="isDead" class="modal-overlay">
      <div class="death-modal">
        <div class="dm-header">{{ gender==='female'?'💀 香消玉殒':'💀 道心破碎' }}</div>
        <div class="dm-timer">复活倒计时: <b>{{ reviveCountdown }}</b> 秒</div>
        <div class="dm-info">复活后恢复 50% HP · 50% MP · 神识+10</div>
        <div class="dm-status"><span>❤️ {{ hp }}%</span><span>💙 {{ mp }}%</span><span>👁️ {{ spiritSense }}</span></div>
        <div v-if="deathLog" class="dm-battle"><div class="dm-battle-title">⚔️ 最后一战</div>
          <div v-for="r in deathLog.rounds" :key="r.round" class="cr-round"><span>第{{ r.round }}回合:</span><span class="cr-dmg">造成 {{ r.player_dmg }} 伤害</span><span v-if="r.desc" class="cr-special">{{ r.desc }}</span><span class="cr-dmg">受到 {{ r.monster_dmg }} 伤害</span><span class="cr-hp">❤️{{ r.player_hp }} | 👹{{ r.monster_hp>0?r.monster_hp:'击败' }}</span></div>
        </div>
        <p class="dm-note">死亡期间无法进行任何操作，但可以发送聊天消息</p>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
defineProps<{ pveReport: any; pveRounds: any[]; encounterResult: any; isDead: boolean; reviveCountdown: number; deathLog: any; gender: string; hp: number; mp: number; spiritSense: number }>()
const emit = defineEmits(['update:pveReport','update:pveRounds','update:encounterResult'])
</script>