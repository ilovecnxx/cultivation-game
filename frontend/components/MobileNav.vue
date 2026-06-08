<template>
  <van-tabbar v-model:active="active" :fixed="true" :border="true" :safe-area-inset-bottom="true" active-color="#d4a843" inactive-color="#8a8578">
    <van-tabbar-item icon="user-o" @click="$emit('openSidebar')">角色</van-tabbar-item>
    <van-tabbar-item icon="chat-o" @click="$emit('openChat')">聊天</van-tabbar-item>
    <van-tabbar-item icon="apps-o" @click="$emit('openMenu')">功能</van-tabbar-item>
    <van-tabbar-item icon="map-marked" @click="$emit('openMap')">地图</van-tabbar-item>
    <van-tabbar-item icon="bookmark-o" @click="$emit('openWiki')">百科</van-tabbar-item>
  </van-tabbar>
</template>

<script setup lang="ts">
const active = ref(0)
defineEmits(['openSidebar','openChat','openMenu','openMap','openWiki'])
</script>