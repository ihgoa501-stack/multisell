<template>
  <n-config-provider :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-loading-bar-provider>
        <n-dialog-provider>
          <a-config-provider :theme="antdTheme">
            <router-view />
          </a-config-provider>
        </n-dialog-provider>
      </n-loading-bar-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { NConfigProvider, NMessageProvider, NLoadingBarProvider, NDialogProvider } from 'naive-ui'
import { themeOverrides } from '@/config/theme'
import { antdTheme } from '@/config/antd-theme'
</script>
