<template>
  <div v-if="error" class="error-boundary">
    <div class="error-boundary-card">
      <h3>⚡ 组件异常</h3>
      <p>{{ error.message || '未知错误' }}</p>
      <van-button type="primary" round @click="retry">重试</van-button>
    </div>
  </div>
  <slot v-else />
</template>

<script setup lang="ts">
const error = ref<Error | null>(null)
function retry() { error.value = null }
onErrorCaptured((err) => { error.value = err as Error; return false })
</script>

<style scoped>
.error-boundary { display:flex; align-items:center; justify-content:center; min-height:100px; }
.error-boundary-card { text-align:center; padding:24px; color:#8a8578; }
.error-boundary-card h3 { color:#ff6b6b; margin-bottom:8px; }
</style>