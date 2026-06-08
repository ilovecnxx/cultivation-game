<template>
  <van-popup v-model:show="visible" position="bottom" round :style="{ height:'60%', background:'#1a1a2e' }">
    <div style="padding:20px;text-align:center;color:#8a8578">
      <h3 style="color:#d4a843">Ranking</h3>
      <p style="margin-top:16px">🏗️ 模块开发中...</p>
      <van-button type="primary" round style="margin-top:20px" @click="visible=false">关闭</van-button>
    </div>
  </van-popup>
</template>
<script setup lang="ts">
const visible = ref(false)
defineExpose({ open:(v:boolean)=>visible.value=v })
</script>
