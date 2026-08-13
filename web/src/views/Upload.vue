<template>
  <div class="upload-page">
    <h1 class="page-title">上传文件</h1>

    <div class="card drop-zone" :class="{ 'is-active': isDragover }"
      @dragover.prevent="isDragover = true"
      @dragleave.prevent="isDragover = false"
      @drop.prevent="onDrop"
      @click="inputRef?.click()">
      <div class="drop-content">
        <div class="drop-icon">
          <UploadFilled style="width: 32px; height: 32px" />
        </div>
        <p class="drop-text">{{ selectedFile ? selectedFile.name : '拖拽文件到此处' }}</p>
        <p class="drop-hint">{{ selectedFile ? formatSize(selectedFile.size) : '或点击选择文件' }}</p>
      </div>
      <input ref="inputRef" type="file" style="display: none" @change="onFileSelect" />
    </div>

    <div v-if="selectedFile" class="card options-card">
      <div class="section-label">选项</div>
      <div class="options-row">
        <div class="option-item">
          <label class="option-label">密码保护</label>
          <el-input v-model="password" placeholder="设置访问密码（可选）" show-password clearable />
        </div>
        <div class="option-item">
          <label class="option-label">过期时间</label>
          <el-select v-model="expireOption" placeholder="选择过期时间" style="width: 100%">
            <el-option label="1 小时后" value="hours:1" />
            <el-option label="3 小时后" value="hours:3" />
            <el-option label="1 天后" value="hours:24" />
            <el-option label="3 天后" value="hours:72" />
            <el-option label="7 天后" value="hours:168" />
            <el-option label="31 天后" value="hours:744" />
            <el-option v-if="defaultExpireHours && ![1, 3, 24, 72, 168, 744].includes(defaultExpireHours)"
              :label="`默认（${defaultExpireHours} 小时后）`" :value="`hours:${defaultExpireHours}`" />
            <el-option label="自定义" value="custom" />
            <el-option label="永不过期" value="never" />
          </el-select>
          <el-input v-if="expireOption === 'custom'" v-model="customExpireText"
            placeholder="例如 1天2小时30分钟" style="margin-top: 8px" />
        </div>
      </div>
      <el-button type="primary" size="large" :loading="uploading" :disabled="!selectedFile" class="upload-btn"
        @click="onUpload">
        {{ uploading ? `上传中 ${progress}%` : '上传' }}
      </el-button>
      <el-progress v-if="uploading" :percentage="progress" style="margin-top: 12px" />
    </div>

    <div v-if="result" class="card result-card">
      <div class="section-label" style="color: var(--color-success)">上传成功</div>
      <div class="result-info">
        <div class="info-row"><span class="info-label">文件名</span><span>{{ result.file_name }}</span></div>
        <div class="info-row"><span class="info-label">大小</span><span>{{ formatSize(result.file_size) }}</span></div>
        <div class="info-row"><span class="info-label">密码</span><span>{{ result.has_password ? '已设置' : '无' }}</span></div>
        <div class="info-row"><span class="info-label">过期</span><span>{{ result.expire_at ? formatDateTime(result.expire_at) : '永不过期' }}</span></div>
      </div>
      <div class="share-section">
        <div class="section-label">分享链接</div>
        <div class="share-input-row">
          <el-input v-model="shareUrl" readonly />
          <el-button @click="copyText(shareUrl)" style="flex-shrink: 0; margin-left: 8px">复制</el-button>
        </div>
      </div>
      <div class="result-actions">
        <el-button @click="reset">继续上传</el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { UploadFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { uploadFile } from '../api/file'
import { adminCheck } from '../api/admin'
import { getSettings } from '../api/settings'
import { formatDateTime } from '../utils/time'
import type { UploadResult } from '../types'

const router = useRouter()
const inputRef = ref<HTMLInputElement>()
const isDragover = ref(false)
const selectedFile = ref<File>()
const password = ref('')
const defaultExpireHours = ref(168)
const expireOption = ref('hours:168')
const customExpireText = ref('')
const uploading = ref(false)
const progress = ref(0)
const result = ref<UploadResult>()
const baseUrl = ref('')

function getShareBaseUrl(baseUrl?: string) {
  return baseUrl?.trim().replace(/\/+$/, '') || window.location.origin
}

const shareUrl = computed(() => {
  if (!result.value) return ''
  return `${getShareBaseUrl(baseUrl.value)}/#/s/${encodeURIComponent(result.value.code)}`
})

function formatSize(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function onDrop(e: DragEvent) {
  isDragover.value = false
  const file = e.dataTransfer?.files[0]
  if (file) selectedFile.value = file
}

function onFileSelect(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (file) selectedFile.value = file
}

function parseExpireText(value: string) {
  const normalized = value.replace(/\s+/g, '')
  const match = normalized.match(/^(?:(\d+)天)?(?:(\d+)小时)?(?:(\d+)分钟)?$/)
  if (!match || !match[0] || !match[1] && !match[2] && !match[3]) return 0
  return Number(match[1] || 0) * 1440 + Number(match[2] || 0) * 60 + Number(match[3] || 0)
}

async function onUpload() {
  if (!selectedFile.value) return
  let expireHours: number | undefined
  let expireMinutes: number | undefined
  if (expireOption.value.startsWith('hours:')) {
    expireHours = Number(expireOption.value.slice(6))
  } else if (expireOption.value === 'custom') {
    expireMinutes = parseExpireText(customExpireText.value)
    if (!expireMinutes) {
      ElMessage.warning('请输入有效时长，例如 1天2小时30分钟')
      return
    }
  }
  uploading.value = true
  progress.value = 0
  try {
    const res = await uploadFile(
      selectedFile.value,
      password.value || undefined,
      expireHours,
      expireMinutes,
      (p) => (progress.value = p),
    )
    result.value = res.data!
    if (result.value.manage_token) {
      const tokens = JSON.parse(localStorage.getItem('manage_tokens') || '[]')
      if (!tokens.includes(result.value.manage_token)) {
        tokens.push(result.value.manage_token)
        localStorage.setItem('manage_tokens', JSON.stringify(tokens))
      }
    }
    ElMessage.success('上传成功')
  } catch (e: any) {
    ElMessage.error(e.message || '上传失败')
  } finally {
    uploading.value = false
  }
}

function copyText(text: string) {
  navigator.clipboard.writeText(text).then(() => ElMessage.success('已复制'))
}

function reset() {
  selectedFile.value = undefined
  password.value = ''
  expireOption.value = `hours:${defaultExpireHours.value}`
  customExpireText.value = ''
  result.value = undefined
  progress.value = 0
  if (inputRef.value) inputRef.value.value = ''
}

onMounted(async () => {
  // Check auth first — only logged-in admins can upload
  try {
    await adminCheck()
  } catch {
    router.push('/admin/login')
    return
  }

  try {
    const res = await getSettings()
    baseUrl.value = res.data?.base_url || ''
    const hours = res.data?.default_expire_hours
    if (hours && hours > 0) {
      defaultExpireHours.value = hours
      expireOption.value = `hours:${hours}`
    }
  } catch {
    defaultExpireHours.value = 168
    expireOption.value = 'hours:168'
  }
})
</script>

<style scoped>
.upload-page {
  max-width: 560px;
  margin: 0 auto;
}
.drop-zone {
  cursor: pointer;
  transition: all 0.2s ease;
}
.drop-zone.is-active {
  border-color: var(--color-primary);
  background: #fafafa;
}
.drop-content {
  text-align: center;
  padding: 40px 0;
}
.drop-icon {
  color: var(--text-placeholder);
  margin-bottom: 12px;
  display: flex;
  justify-content: center;
}
.drop-text {
  color: var(--text-primary);
  font-size: 15px;
  font-weight: 500;
  margin-bottom: 4px;
}
.drop-hint {
  color: var(--text-secondary);
  font-size: 13px;
}
.options-card {
  margin-top: 16px;
}
.options-row {
  display: flex;
  gap: 16px;
  margin-top: 12px;
}
.option-item {
  flex: 1;
}
.option-label {
  display: block;
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 6px;
}
.upload-btn {
  width: 100%;
  margin-top: 20px;
}
.result-card {
  margin-top: 16px;
}
.result-info {
  margin-top: 12px;
  font-size: 14px;
}
.info-row {
  display: flex;
  padding: 8px 0;
  border-bottom: 1px solid var(--border-light);
}
.info-row:last-child {
  border-bottom: none;
}
.info-label {
  color: var(--text-secondary);
  width: 80px;
  flex-shrink: 0;
}
.share-section {
  margin-top: 16px;
}
.share-input-row {
  display: flex;
  margin-top: 8px;
}
.result-actions {
  display: flex;
  gap: 12px;
  margin-top: 20px;
}
</style>
