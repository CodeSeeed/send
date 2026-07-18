<template>
  <div class="admin-login-page">
    <div class="card login-card">
      <h1 class="login-title">管理员登录</h1>
      <el-form @submit.prevent="onLogin">
        <el-form-item>
          <el-input v-model="username" placeholder="用户名" size="large" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="password" type="password" placeholder="密码" size="large" show-password
            @keyup.enter="onLogin" />
        </el-form-item>
        <el-button type="primary" size="large" :loading="loading" style="width: 100%" @click="onLogin">
          登录
        </el-button>
      </el-form>
      <p v-if="error" class="login-error">{{ error }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { adminLogin } from '../api/admin'

const router = useRouter()
const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function onLogin() {
  if (!username.value || !password.value) return
  loading.value = true
  error.value = ''
  try {
    const res = await adminLogin(username.value, password.value)
    localStorage.setItem('admin_token', res.data!.token)
    ElMessage.success('登录成功')
    router.push('/admin')
  } catch (e: any) {
    error.value = e.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.admin-login-page {
  max-width: 400px;
  margin: 80px auto 0;
}
.login-card {
  padding: 32px;
}
.login-title {
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 24px;
  text-align: center;
}
.login-error {
  color: var(--color-danger);
  font-size: 14px;
  margin-top: 12px;
  text-align: center;
}
</style>
