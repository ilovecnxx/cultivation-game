<template>
  <div class="app-root" :class="{ 'light-mode': !isDark }">
    <slot />
  </div>
</template>

<script setup lang="ts">
const isDark = ref(true)
onMounted(() => {
  const m = localStorage.getItem('theme-mode')
  if (m) isDark.value = m === 'dark'
})
</script>

<style lang="scss">
// 全局重置
* { margin: 0; padding: 0; box-sizing: border-box; -webkit-tap-highlight-color: transparent; }
html, body { width: 100%; height: 100%; overflow: hidden; font-family: 'Noto Sans SC', sans-serif; }
</style>