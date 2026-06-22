<template>
  <div style="display: flex; justify-content: center; align-items: center; min-height: 100vh; background: linear-gradient(135deg, #2c3e50 0%, #3498db 100%);">
    <a-card style="width: 400px;" :bordered="true">
      <template #title>
        <div style="text-align: center;">
          <div style="display: flex; justify-content: center; align-items: center; gap: 10px;">
            <img
              src="/brand/lingmirror-icon.png"
              alt="凌镜 LingMirror"
              style="width: 36px; height: 36px; border-radius: 8px;"
            />
            <h2 style="margin: 0; font-size: 24px;">凌镜 LingMirror</h2>
          </div>
          <p style="margin: 8px 0 0; color: var(--ant-color-text-tertiary); font-size: 14px;">跨境电商 AgentOS</p>
        </div>
      </template>
      <a-form ref="formRef" :model="form" :rules="rules" layout="vertical">
        <a-form-item label="用户名" name="username">
          <a-input v-model:value="form.username" placeholder="请输入用户名" size="large" />
        </a-form-item>
        <a-form-item label="密码" name="password">
          <a-input-password v-model:value="form.password" placeholder="请输入密码" size="large" @press-enter="handleLogin" />
        </a-form-item>
      </a-form>
      <a-button type="primary" block size="large" :loading="loading" @click="handleLogin">登 录</a-button>
      <div style="text-align: center; margin-top: 12px;">
        <span style="font-size: 13px; color: var(--ant-color-text-tertiary);">测试账号: admin / admin123</span>
      </div>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import http from '@/api/http'

const router = useRouter()
const formRef = ref<any>(null)
const loading = ref(false)
const form = ref({ username: '', password: '' })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
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
