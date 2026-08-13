<template>
  <div class="share-page">
    <div v-if="loading" class="card">
      <el-skeleton :rows="5" animated />
    </div>

    <div v-else-if="notFound" class="card" style="text-align: center; padding: 64px 24px">
      <div style="font-size: 48px; color: var(--text-placeholder); margin-bottom: 16px">—</div>
      <h2 style="font-size: 18px; font-weight: 600; margin-bottom: 8px">文件不存在</h2>
      <p style="color: var(--text-secondary); font-size: 14px; margin-bottom: 24px">该文件可能已过期或已被删除</p>
      <el-button type="primary" @click="$router.push('/')">去上传</el-button>
    </div>

    <div v-else-if="fileInfo" class="file-detail">
      <h1 class="page-title">{{ fileInfo.file_name }}</h1>

      <div class="card info-card">
        <div class="info-row">
          <span class="info-label">大小</span>
          <span>{{ formatSize(fileInfo.file_size) }}</span>
        </div>
        <div class="info-row">
          <span class="info-label">密码保护</span>
          <span>
            <el-tag :type="fileInfo.has_password ? 'warning' : 'info'" size="small" effect="plain">
              {{ fileInfo.has_password ? '需要密码' : '无需密码' }}
            </el-tag>
          </span>
        </div>
        <div class="info-row">
          <span class="info-label">下载次数</span>
          <span>{{ fileInfo.download_count }}</span>
        </div>
        <div class="info-row">
          <span class="info-label">过期时间</span>
          <span>{{ fileInfo.expire_at ? formatDateTime(fileInfo.expire_at) : '永不过期' }}</span>
        </div>
        <div class="info-row" style="border-bottom: none">
          <span class="info-label">创建时间</span>
          <span>{{ formatDateTime(fileInfo.created_at) }}</span>
        </div>
      </div>

      <div v-if="fileInfo.has_password" class="card password-card">
        <div class="section-label">请输入访问密码</div>
        <div class="password-row">
          <el-input v-model="password" placeholder="输入密码" show-password
            @keyup.enter="onDownload" />
          <el-button type="primary" style="margin-left: 8px; flex-shrink: 0"
            :loading="downloading" @click="onDownload">确认</el-button>
        </div>
      </div>

      <el-button v-if="!fileInfo.has_password" type="primary" size="large"
        class="download-btn" :loading="downloading" @click="onDownload">
        下载文件
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getFileInfo, verifyPassword, downloadFile } from '../api/file'
import { formatDateTime } from '../utils/time'
import type { FileInfo } from '../types'

const route = useRoute()
const loading = ref(true)
const notFound = ref(false)
const fileInfo = ref<FileInfo>()
const password = ref('')
const downloading = ref(false)

function formatSize(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

async function loadInfo() {
  const code = route.params.code as string
  try {
    const res = await getFileInfo(code)
    fileInfo.value = res.data
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
}

async function onDownload() {
  if (!fileInfo.value) return
  const code = fileInfo.value.code

  if (fileInfo.value.has_password && !password.value) {
    ElMessage.warning('请输入密码')
    return
  }

  downloading.value = true
  try {
    // Verify password (or skip if no password) via POST to get one-time download token
    const payload = fileInfo.value.has_password ? { password: password.value } : { password: '' }
    const res = await verifyPassword(code, payload.password)
    const token = res.data!.download_token
    // Download via fetch + blob with the token in the Authorization header
    await downloadFile(code, token)
  } catch (e: any) {
    ElMessage.error(e.message || '下载失败')
  } finally {
    downloading.value = false
  }
}

onMounted(loadInfo)
</script>

<style scoped>
.share-page {
  max-width: 560px;
  margin: 0 auto;
}
.info-card {
  font-size: 14px;
}
.info-row {
  display: flex;
  padding: 10px 0;
  border-bottom: 1px solid var(--border-light);
}
.info-label {
  color: var(--text-secondary);
  width: 80px;
  flex-shrink: 0;
}
.password-card {
  margin-top: 16px;
}
.password-row {
  display: flex;
  margin-top: 8px;
}
.download-btn {
  width: 100%;
  margin-top: 16px;
  height: 44px;
  font-size: 15px;
}
</style>
