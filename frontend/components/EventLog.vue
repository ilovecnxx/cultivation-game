<template>
  <aside class="gh-panel gh-log">
    <div class="panel-header">
      <h4>📜 修仙日志</h4>
      <div class="panel-actions panel-actions-right">
        <van-button size="small" plain type="default" @click="logLocked=!logLocked">{{ logLocked?'🔓':'🔒' }}</van-button>
      </div>
    </div>
    <div class="panel-body log-body" ref="logBody">
      <div v-for="(l,i) in logs" :key="i" class="log-msg" :class="l.type">
        <span class="log-time">{{ l.time }}</span>
        <span class="log-text">{{ l.text }}</span>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
const logs = defineModel<any[]>('logs', { default: () => [] })
const logLocked = ref(false)
const logBody = ref<HTMLElement | null>(null)

function addLog(type: string, text: string) {
  logs.value.push({ time: new Date().toLocaleTimeString('zh-CN', { hour12: false }).slice(0, 5), type, text })
  if (logs.value.length > 500) logs.value.shift()
  if (!logLocked.value) nextTick(() => { if (logBody.value) logBody.value.scrollTop = logBody.value.scrollHeight })
}

function clearLogs() { logs.value.splice(0) }

defineExpose({ addLog, clearLogs })
</script>