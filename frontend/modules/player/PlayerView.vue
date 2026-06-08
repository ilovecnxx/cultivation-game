<template>
<Teleport to="body">
<div v-if="visible" class="modal-overlay" @click.self="visible=false">
<div class="wiki-modal" style="width:500px;max-width:95vw">
<div class="gold-divider"/><header class="top-bar" style="border-radius:8px 8px 0 0"><div class="top-bar-inner"><div class="top-bar-spacer"/><span class="brand-name" style="font-size:16px">👤 角色</span><div class="top-bar-spacer"/><button class="modal-close" @click="visible=false">✕</button></div></header><div class="gold-divider"/>
<div class="wiki-body" style="text-align:center;padding:40px;color:rgba(255,255,255,.3)">🏗️ 角色详情开发中...</div>
</div>
</div>
</Teleport>
</template>
<script setup lang="ts">
const visible=ref(false)
defineExpose({open:(v:boolean)=>{visible.value=v}})
</script>