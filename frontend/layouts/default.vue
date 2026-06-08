<template>
  <div class="app-root">
    <slot />
  </div>
</template>

<script setup lang="ts">
const isDark = ref(true)

function applyTheme() {
  const d = isDark.value
  document.documentElement.classList.toggle('light-mode', !d)
  localStorage.setItem('theme-mode', d ? 'dark' : 'light')
}

function toggleTheme() {
  isDark.value = !isDark.value
  applyTheme()
}

// 暴露给子组件
provide('toggleTheme', toggleTheme)
provide('isDark', isDark)

onMounted(() => {
  const saved = localStorage.getItem('theme-mode')
  isDark.value = saved !== 'light'
  applyTheme()
})
</script>

<style lang="scss">
* { margin: 0; padding: 0; box-sizing: border-box; -webkit-tap-highlight-color: transparent; }
html, body { width: 100%; height: 100%; overflow: hidden; font-family: 'Noto Sans SC', sans-serif; }
</style>