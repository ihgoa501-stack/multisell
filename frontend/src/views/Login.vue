<template>
  <div style="display: flex; justify-content: center; align-items: center; min-height: 100vh; background: linear-gradient(135deg, #2c3e50 0%, #3498db 100%);">
    <n-card style="width: 400px;" :bordered="true">
      <template #header>
        <div style="text-align: center;">
          <h2 style="margin: 0; font-size: 24px;">🌐 MultiSell</h2>
          <p style="margin: 4px 0 0; color: #888; font-size: 14px;">AI跨境电商商品中台</p>
        </div>
      </template>
      <n-form ref="formRef" :model="form" :rules="rules">
        <n-form-item label="用户名" path="username">
          <n-input v-model:value="form.username" placeholder="请输入用户名" size="large" />
        </n-form-item>
        <n-form-item label="密码" path="password">
          <n-input v-model:value="form.password" type="password" show-password-on="click" placeholder="请输入密码" size="large" @keyup.enter="handleLogin" />
        </n-form-item>
      </n-form>
      <n-button type="primary" block size="large" :loading="loading" @click="handleLogin">登 录</n-button>
      <div style="text-align: center; margin-top: 12px;">
        <span style="font-size: 13px; color: #999;">测试账号: admin / admin123</span>
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import http from '@/api/http'

const router = useRouter()
const message = useMessage()
const formRef = ref<any>(null)
const loading = ref(false)
const form = ref({ username: '', password: '' })
const rules = {
  username: { required: true, message: '请输入用户名', trigger: 'blur' },
  password: { required: true, message: '请输入密码', trigger: 'blur' },
}

async function handleLogin() {
  try { await formRef.value?.validate() } catch { return }
  loading.value = true
  try {
    const res: any = await http.post('/auth/login', form.value)
    if (res.code === 200) {
      localStorage.setItem('token', res.data.access_token)
      localStorage.setItem('user', JSON.stringify(res.data.user))
      message.success('登录成功')
      router.push('/dashboard')
    } else {
      message.error(res.message || '登录失败')
    }
  } catch (e: any) {
    message.error(e.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>
