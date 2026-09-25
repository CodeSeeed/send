<template>
  <div class="share-page">
    <div v-if="loading" class="card">
      <el-skeleton :rows="5" animated />
    </div>

    <div v-else-if="notFound" class="card" style="text-align: center; padding: 64px 24px">
      <div style="font-size: 48px; color: var(--text-placeholder); margin-bottom: 16px">—</div>
      <h2 style="font-size: 18px; font-weight: 600; margin-bottom: 8px">文件不存在</h2>
      <p style="color: var(--text-secondary); font-size: 14px; margin-bottom: 24px">该文件可能已过期或已被删除</p>
      <el-button type="primary" @click="$router.push('/')">返回首页</el-button>
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
          <span class="info-label">过期时间</span>
          <span>{{ fileInfo.expire_at ? formatDateTime(fileInfo.expire_at) : '永不过期' }}</span>
        </div>
        <div class="info-row" style="border-bottom: none">
          <span class="info-label">创建时间</span>
          <span>{{ formatDateTime(fileInfo.created_at) }}</span>
        </div>
      </div>

      <!-- Password input (only for password-protected files before verification) -->
      <div v-if="fileInfo.has_password && !verified" class="card password-card">
        <div class="section-label">请输入访问密码</div>
        <div class="password-row">
          <el-input v-model="password" placeholder="输入密码" show-password
            :disabled="downloadLimitReached" @keyup.enter="onVerify" />
          <el-button type="primary" style="margin-left: 8px; flex-shrink: 0"
            :loading="verifying" :disabled="downloadLimitReached" @click="onVerify">
            {{ downloadLimitReached ? '已达上限' : '确认' }}
          </el-button>
        </div>
        <p v-if="downloadLimitReached" class="limit-hint">该文件的下载次数已达上限</p>
      </div>

      <p v-if="downloadLimitReached && !fileInfo.has_password" class="limit-hint">
        该文件的下载次数已达上限
      </p>

      <!-- Actions after token is obtained -->
      <div v-if="verified" class="action-section">
        <!-- Preview area -->
        <div v-if="hasPreview" class="preview-area">
          <iframe v-if="previewType === 'pdf'" :src="previewDataUrl" class="preview-frame" />
          <img v-else-if="previewType === 'image'" :src="previewDataUrl" class="preview-image" />
          <pre v-else-if="previewType === 'text'" class="preview-text">{{ previewText }}</pre>
          <div v-else class="preview-unsupported">该文件类型不支持预览</div>
        </div>

        <div class="btn-row">
          <el-button :loading="previewLoading" :disabled="!verified" @click="onPreview">
            {{ hasPreview ? '刷新预览' : '预览文件' }}
          </el-button>
          <el-button type="primary" size="large"
            :loading="downloading" :disabled="downloadLimitReached" @click="onDownload">
            {{ downloadLimitReached ? '下载次数已达上限' : '下载文件' }}
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getFileInfo, verifyPassword, downloadFile, previewFile } from '../api/file'
import { formatDateTime } from '../utils/time'
import { isHttpError } from '../utils/request'
import type { PublicFileInfo } from '../types'

const route = useRoute()
const loading = ref(true)
const notFound = ref(false)
const fileInfo = ref<PublicFileInfo>()
const password = ref('')
const downloading = ref(false)
const verified = ref(false)
const verifying = ref(false)
const token = ref('')
const tokenExpiresAt = ref(0)
const previewLoading = ref(false)
const previewDataUrl = ref('')
const previewType = ref<'pdf' | 'image' | 'text' | 'unsupported'>('unsupported')
const previewText = ref('')
// The server signals an exhausted download quota with HTTP 409, so the limit
// state is driven by status codes rather than message-text matching. The
// public file-info response no longer carries download_count/max_downloads.
const limitReached = ref(false)
// Reuse a download token only while it is comfortably inside the server-side
// TTL (2 minutes, server/service/file.go tokenTTL). This mirrors the server
// constant: if the server TTL changes, keep this below it so the client never
// presents a token that already expired server-side.
const DOWNLOAD_TOKEN_REUSE_MS = 110 * 1000

const downloadLimitReached = computed(() => limitReached.value)

function handleActionError(e: any, fallback: string) {
  if (isHttpError(e, 409)) {
    limitReached.value = true
  } else {
    ElMessage.error(e.message || fallback)
  }
}

const hasPreview = computed(() => previewType.value === 'text' || previewDataUrl.value !== '')

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
    // Non-password files: obtain the token right away so preview/download are one click.
    // If the file has hit its download limit, the server returns an error here and we
    // surface the limit state instead of entering the verified flow.
    if (res.data && !res.data.has_password) {
      await onVerify()
    }
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
}

async function onVerify() {
  if (!fileInfo.value) return
  if (fileInfo.value.has_password && !password.value) {
    ElMessage.warning('请输入密码')
    return
  }
  verifying.value = true
  try {
    await getDownloadToken(true)
  } catch (e: any) {
    handleActionError(e, '密码验证失败')
  } finally {
    verifying.value = false
  }
}

async function getDownloadToken(forceRefresh = false): Promise<string> {
  if (!fileInfo.value) throw new Error('文件信息尚未加载')
  if (!forceRefresh && token.value && Date.now() < tokenExpiresAt.value) {
    return token.value
  }

  const downloadPassword = fileInfo.value.has_password ? password.value : ''
  const res = await verifyPassword(fileInfo.value.code, downloadPassword)
  const nextToken = res.data?.download_token
  if (!nextToken) throw new Error('获取下载凭证失败')

  token.value = nextToken
  // The server token lives for two minutes. Refresh a little early so a token
  // does not expire between the check and the following preview/download.
  tokenExpiresAt.value = Date.now() + DOWNLOAD_TOKEN_REUSE_MS
  verified.value = true
  return nextToken
}

function clearDownloadToken() {
  token.value = ''
  tokenExpiresAt.value = 0
}

async function onPreview() {
  if (!fileInfo.value) return
  previewLoading.value = true
  try {
    const activeToken = await getDownloadToken()
    const { blob, mimeType } = await previewFile(fileInfo.value.code, activeToken)
    // Map MIME type to a preview renderer
    const base = mimeType.split(';')[0].trim()
    revokePreviewUrl()
    previewText.value = ''
    if (base === 'application/pdf') {
      previewType.value = 'pdf'
      previewDataUrl.value = URL.createObjectURL(blob)
    } else if (base.startsWith('image/')) {
      previewType.value = 'image'
      previewDataUrl.value = URL.createObjectURL(blob)
    } else if (base.startsWith('text/') || base === 'application/json') {
      previewType.value = 'text'
      previewText.value = await blob.text()
    } else {
      previewType.value = 'unsupported'
      ElMessage.warning('该文件类型不支持在线预览')
    }
  } catch (e: any) {
    handleActionError(e, '预览失败')
  } finally {
    previewLoading.value = false
  }
}

function revokePreviewUrl() {
  if (previewDataUrl.value) {
    URL.revokeObjectURL(previewDataUrl.value)
  }
}

async function onDownload() {
  if (!fileInfo.value) return

  downloading.value = true
  try {
    const activeToken = await getDownloadToken()
    await downloadFile(fileInfo.value.code, activeToken)
  } catch (e: any) {
    handleActionError(e, '下载失败')
  } finally {
    // Download tokens are one-time credentials. Clear after every attempt so
    // the next click obtains a fresh token, including after server-side errors.
    clearDownloadToken()
    downloading.value = false
  }
}

onMounted(loadInfo)
onBeforeUnmount(revokePreviewUrl)
</script>

<style scoped>
.share-page {
  max-width: 700px;
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
.action-section {
  margin-top: 16px;
}
.btn-row {
  display: flex;
  gap: 12px;
  margin-top: 16px;
}
.btn-row .el-button {
  flex: 1;
  height: 44px;
  font-size: 15px;
}
.preview-area {
  border: 1px solid var(--border-color);
  border-radius: 4px;
  overflow: hidden;
  background: var(--bg-card);
}
.preview-frame {
  width: 100%;
  height: 600px;
  border: none;
  display: block;
}
.preview-image {
  display: block;
  max-width: 100%;
  max-height: 600px;
  margin: 0 auto;
  object-fit: contain;
}
.preview-text {
  padding: 16px;
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  max-height: 500px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
  background: var(--preview-text-bg);
  color: var(--text-primary);
}
.preview-unsupported {
  padding: 40px 16px;
  text-align: center;
  color: var(--text-secondary);
}
.limit-hint {
  margin-top: 8px;
  color: var(--color-danger);
  font-size: 13px;
  text-align: center;
}
</style>
