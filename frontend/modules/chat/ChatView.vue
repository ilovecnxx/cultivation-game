<template>
  <ChatPanel :player-name="playerName" />
</template>
<script setup lang="ts">
defineProps<{ playerName: string }>()
</script>
