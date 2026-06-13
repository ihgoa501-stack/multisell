<template>
  <n-message-provider>
    <n-loading-bar-provider>
      <n-dialog-provider>
        <router-view />
      </n-dialog-provider>
    </n-loading-bar-provider>
  </n-message-provider>
</template>

<script setup lang="ts">
import { NMessageProvider, NLoadingBarProvider, NDialogProvider } from 'naive-ui'
</script>

<style>
body {
  margin: 0;
  padding: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}
</style>
