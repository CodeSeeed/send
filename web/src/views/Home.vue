<template>
  <div class="receive-page">
    <section class="receive-hero">
      <div class="eyebrow">文件收件</div>
      <h1>输入收件码，提取文件</h1>
      <p class="hero-description">收件码不区分大小写。若文件设置了访问密码，进入后仍需验证密码。</p>

      <div class="card receive-card">
        <label class="receive-label" for="receive-code">收件码</label>
        <div class="receive-row">
          <el-input
            id="receive-code"
            v-model="receiveCode"
            size="large"
            maxlength="32"
            clearable
            autocomplete="one-time-code"
            placeholder="例如 8K7M2XPA"
            @input="normalizeCode"
            @keyup.enter="onReceive"
          />
          <el-button type="primary" size="large" :loading="loading" @click="onReceive">
            提取文件
          </el-button>
        </div>
        <p class="receive-hint">收件码由上传者在上传文件时选择生成。</p>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { receiveFile } from '../api/file'

const router = useRouter()
const receiveCode = ref('')
const loading = ref(false)

function normalizeCode(value: string) {
  receiveCode.value = value.toUpperCase().replace(/[^A-Z0-9]/g, '')
}

async function onReceive() {
  const code = receiveCode.value.trim()
  if (!code) {
    ElMessage.warning('请输入收件码')
    return
  }

  loading.value = true
  try {
    const res = await receiveFile(code)
    if (!res.data?.code) throw new Error('未找到文件')
    await router.push({ name: 'share', params: { code: res.data.code } })
  } catch (e: any) {
    ElMessage.error(e.message || '收件码无效或文件已过期')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.receive-page {
  min-height: calc(100vh - 160px);
  display: grid;
  place-items: center;
}

.receive-hero {
  width: min(100%, 620px);
  text-align: center;
}

.eyebrow {
  color: var(--color-primary);
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.18em;
  margin-bottom: 12px;
}

h1 {
  font-size: clamp(28px, 6vw, 42px);
  line-height: 1.2;
  letter-spacing: -1px;
}

.hero-description {
  color: var(--text-secondary);
  line-height: 1.7;
  margin: 14px auto 28px;
}

.receive-card {
  padding: 24px;
  text-align: left;
}

.receive-label {
  display: block;
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 8px;
}

.receive-row {
  display: flex;
  gap: 10px;
}

.receive-row :deep(.el-input__inner) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 17px;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.receive-row .el-button {
  min-width: 112px;
}

.receive-hint {
  color: var(--text-placeholder);
  font-size: 12px;
  margin-top: 10px;
}

@media (max-width: 560px) {
  .receive-row {
    flex-direction: column;
  }

  .receive-row .el-button {
    width: 100%;
  }
}
</style>
