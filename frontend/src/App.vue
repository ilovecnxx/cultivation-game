<template>
  <router-view />
</template>

<style lang="scss">
* { margin: 0; padding: 0; box-sizing: border-box; }

html, body {
  width: 100%; height: 100%; overflow: hidden;
  font-family: 'Noto Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif;
  font-size: 14px; background: #0d0d1a; color: #e8e0d0;
  -webkit-font-smoothing: antialiased;
}

::-webkit-scrollbar { width: 4px; height: 4px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: rgba(212,168,67,0.25); border-radius: 3px; }
::selection { background: rgba(212,168,67,0.15); color: #d4a843; }
</style>
