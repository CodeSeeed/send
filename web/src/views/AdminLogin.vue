<template>
  <div class="admin-login-page">
    <div class="card login-card">
      <h1 class="login-title">{{ registered ? '管理员登录' : '注册管理员' }}</h1>
      <el-form @submit.prevent="onSubmit">
        <el-form-item>
          <el-input v-model="username" placeholder="用户名" size="large" autocomplete="username" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="password" type="password" :placeholder="registered ? '密码' : '设置密码'" size="large" show-password
            autocomplete="current-password" @keyup.enter="onSubmit" />
        </el-form-item>
        <el-button type="primary" size="large" :loading="loading" style="width: 100%" @click="onSubmit">
          {{ registered ? '登录' : '注册并登录' }}
        </el-button>
      </el-form>
      <p v-if="error" class="login-error">{{ error }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { adminLogin, adminRegister, adminStatus } from '../api/admin'

const router = useRouter()
const registered = ref(true)
const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function loadStatus() {
  try {
    const res = await adminStatus()
    registered.value = res.data?.registered ?? true
  } catch (e: any) {
    error.value = e.message || '获取管理员状态失败'
  }
}

async function onSubmit() {
  if (!username.value || !password.value) return
  loading.value = true
  error.value = ''
  try {
    const res = registered.value
      ? await adminLogin(username.value, password.value)
      : await adminRegister(username.value, password.value)
    localStorage.setItem('admin_token', res.data!.token)
    ElMessage.success(registered.value ? '登录成功' : '注册成功')
    router.push('/admin')
  } catch (e: any) {
    error.value = e.message || (registered.value ? '登录失败' : '注册失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadStatus)
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
