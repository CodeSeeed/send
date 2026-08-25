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
        <p v-if="maxFileSize" class="size-limit">最大文件大小：{{ formatSize(maxFileSize) }}</p>
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
      <div class="options-row">
        <div class="option-item">
          <label class="option-label">下载次数</label>
          <el-select v-model="downloadOption" placeholder="选择下载次数限制" style="width: 100%">
            <el-option label="不限制" value="unlimited" />
            <el-option label="1 次" value="1" />
            <el-option label="3 次" value="3" />
            <el-option label="5 次" value="5" />
            <el-option label="10 次" value="10" />
            <el-option label="自定义" value="custom" />
          </el-select>
          <el-input v-if="downloadOption === 'custom'" v-model="customDownloadsText"
            placeholder="输入下载次数" style="margin-top: 8px" />
        </div>
        <div class="option-item">
          <label class="option-label">文件收件码</label>
          <div class="receive-code-option">
            <el-switch v-model="generateReceiveCode" active-text="生成" inactive-text="不生成" />
            <span>生成后可在首页输入收件码提取</span>
          </div>
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
        <div class="info-row"><span class="info-label">下载次数</span><span>{{ result.max_downloads > 0 ? result.max_downloads + ' 次' : '不限制' }}</span></div>
        <div v-if="result.receive_code" class="info-row">
          <span class="info-label">收件码</span>
          <strong class="receive-code-value">{{ result.receive_code }}</strong>
          <el-button link type="primary" @click="copyText(result.receive_code)">复制</el-button>
        </div>
      </div>
      <div class="share-section">
        <div class="section-label">分享链接</div>
        <div class="share-input-row">
          <el-input v-model="shareUrl" readonly />
          <el-button @click="copyText(shareUrl)" style="flex-shrink: 0; margin-left: 8px">复制</el-button>
        </div>
        <div v-if="qrDataUrl" class="qr-wrap">
          <img :src="qrDataUrl" alt="二维码" width="160" height="160" />
        </div>
      </div>
      <div class="result-actions">
        <el-button @click="reset">继续上传</el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { UploadFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { uploadFile } from '../api/file'
import { adminGetSettings } from '../api/settings'
import { formatDateTime } from '../utils/time'
import type { UploadResult } from '../types'
import QRCode from 'qrcode'

const inputRef = ref<HTMLInputElement>()
const isDragover = ref(false)
const selectedFile = ref<File>()
const password = ref('')
const defaultExpireHours = ref(168)
const expireOption = ref('hours:168')
const customExpireText = ref('')
const downloadOption = ref('unlimited')
const customDownloadsText = ref('')
const generateReceiveCode = ref(false)
const uploading = ref(false)
const progress = ref(0)
const result = ref<UploadResult>()
const baseUrl = ref('')
const maxFileSize = ref(0)
const qrDataUrl = ref('')

function getShareBaseUrl(baseUrl?: string) {
  return baseUrl?.trim().replace(/\/+$/, '') || window.location.origin
}

const shareUrl = computed(() => {
  if (!result.value) return ''
  return `${getShareBaseUrl(baseUrl.value)}/#/s/${encodeURIComponent(result.value.code)}`
})

watch(shareUrl, async (url) => {
  if (!url) {
    qrDataUrl.value = ''
    return
  }
  try {
    qrDataUrl.value = await QRCode.toDataURL(url, { width: 320, margin: 1 })
  } catch {
    // QR generation can fail in exotic environments (missing canvas support);
    // the share link is still displayed, so the QR code is purely optional.
    qrDataUrl.value = ''
  }
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
  if (!normalized) return 0

  // Accept units in any order, but require every input character to belong to
  // exactly one valid segment. This rejects partial matches such as "1天abc"
  // and ambiguous duplicates such as "1天2天".
  const segment = /(\d+)(天|小时|分钟)/g
  const seenUnits = new Set<string>()
  const multipliers: Record<string, number> = { 天: 1440, 小时: 60, 分钟: 1 }
  let cursor = 0
  let total = 0
  let match: RegExpExecArray | null

  while ((match = segment.exec(normalized)) !== null) {
    const unit = match[2]
    const amount = Number(match[1])
    if (match.index !== cursor || seenUnits.has(unit) || !Number.isSafeInteger(amount)) return 0
    seenUnits.add(unit)
    total += amount * multipliers[unit]
    if (!Number.isSafeInteger(total)) return 0
    cursor = segment.lastIndex
  }

  return cursor === normalized.length && total > 0 ? total : 0
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
  let maxDownloads: number | undefined
  if (downloadOption.value === 'custom') {
    const n = Number(customDownloadsText.value)
    if (!Number.isInteger(n) || n <= 0) {
      ElMessage.warning('请输入有效的下载次数')
      return
    }
    maxDownloads = n
  } else if (downloadOption.value !== 'unlimited') {
    maxDownloads = Number(downloadOption.value)
  }
  uploading.value = true
  progress.value = 0
  try {
    const res = await uploadFile(
      selectedFile.value,
      password.value || undefined,
      expireHours,
      expireMinutes,
      maxDownloads,
      generateReceiveCode.value,
      (p) => (progress.value = p),
    )
    result.value = res.data!
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
  downloadOption.value = 'unlimited'
  customDownloadsText.value = ''
  generateReceiveCode.value = false
  result.value = undefined
  progress.value = 0
  if (inputRef.value) inputRef.value.value = ''
}

onMounted(async () => {
  // Auth is enforced by the router guard (beforeEach) — reaching this hook
  // means the admin session is valid, so only load the settings here.
  try {
    const res = await adminGetSettings()
    baseUrl.value = res.data?.base_url || ''
    maxFileSize.value = res.data?.max_file_size || 0
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
  background: var(--bg-hover);
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
.size-limit {
  color: var(--text-placeholder);
  font-size: 12px;
  margin-top: 8px;
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
.receive-code-option {
  min-height: 32px;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  color: var(--text-placeholder);
  font-size: 12px;
}
.receive-code-value {
  flex: 1;
  color: var(--color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 16px;
  letter-spacing: 0.12em;
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
.qr-wrap {
  display: flex;
  justify-content: center;
  margin-top: 16px;
  padding: 12px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 4px;
}
.qr-wrap img {
  display: block;
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
