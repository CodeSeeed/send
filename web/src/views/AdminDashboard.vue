<template>
  <div class="admin-page">
    <div class="page-header">
      <h1 class="page-title" style="margin-bottom: 0">管理后台</h1>
      <div style="display: flex; gap: 8px">
        <el-button @click="showChangePwd = true">修改密码</el-button>
        <el-button @click="onLogout">退出登录</el-button>
      </div>
    </div>

    <div class="section-label" style="margin-top: 32px">文件管理</div>
    <div class="search-bar">
      <el-input v-model="keyword" placeholder="搜索文件名、分享码或收件码…" clearable style="width: 320px"
        @keyup.enter="onSearch" @clear="onSearch" />
      <el-button @click="onSearch">搜索</el-button>
    </div>
    <div class="card" style="padding: 0; margin-top: 8px">
      <el-table :data="files" v-loading="loading" empty-text="暂无文件">
        <el-table-column prop="id" label="ID" width="60" align="center" />
        <el-table-column prop="file_name" label="文件名" min-width="160" show-overflow-tooltip />
        <el-table-column label="收件码" width="120" align="center">
          <template #default="{ row }">
            <span v-if="row.receive_code" class="receive-code-cell">{{ row.receive_code }}</span>
            <span v-else style="color: var(--text-placeholder)">—</span>
          </template>
        </el-table-column>
        <el-table-column label="大小" width="80">
          <template #default="{ row }">{{ formatSize(row.file_size) }}</template>
        </el-table-column>
        <el-table-column label="下载" width="100" align="center">
          <template #default="{ row }">
            <span v-if="row.max_downloads > 0">{{ row.download_count }} / {{ row.max_downloads }}</span>
            <span v-else>{{ row.download_count }}</span>
          </template>
        </el-table-column>
        <el-table-column label="密码" width="60" align="center">
          <template #default="{ row }">
            <el-tag :type="row.has_password ? 'warning' : 'info'" size="small" effect="plain">{{ row.has_password ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="过期" width="160">
          <template #default="{ row }">
            <span v-if="row.expire_at" style="color: var(--text-secondary); font-size: 13px">{{ formatDateTime(row.expire_at) }}</span>
            <el-tag v-else type="success" size="small" effect="plain">永不过期</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="160">
          <template #default="{ row }">
            <span style="color: var(--text-secondary); font-size: 13px">{{ formatDateTime(row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button link size="small" @click="showShareLink(row.code)">分享链接</el-button>
            <el-popconfirm title="确定删除？" @confirm="onDelete(row.id)">
              <template #reference>
                <el-button type="danger" link size="small" style="color: var(--color-danger)">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="total > 0" class="pagination-wrap">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="sizes, prev, pager, next, total"
          background
          @current-change="loadFiles"
          @size-change="onPageSizeChange" />
      </div>
    </div>

    <div class="section-label" style="margin-top: 32px">系统设置</div>
    <div class="card settings-card" style="margin-top: 8px">
      <div class="setting-row">
        <div>
          <div class="setting-label">单文件最大大小</div>
          <div class="setting-hint">例如 30MB、1GB，修改后立即生效</div>
        </div>
        <div class="setting-control">
          <el-input v-model="maxFileSizeText" placeholder="30MB" style="width: 180px" />
        </div>
      </div>
      <div class="setting-row">
        <div>
          <div class="setting-label">当前访问 URL</div>
          <div class="setting-hint">例如 https://send.example.com，用于生成分享链接</div>
        </div>
        <el-input v-model="baseUrl" placeholder="留空使用当前地址" style="width: 300px" />
      </div>
      <div class="settings-actions">
        <el-button type="primary" :loading="savingSettings" @click="onSaveSettings">保存设置</el-button>
      </div>
    </div>

    <el-dialog v-model="showShareDialog" title="文件分享链接" width="520px">
      <el-input v-model="shareLink" readonly />
      <div v-if="shareQrDataUrl" class="qr-wrap">
        <img :src="shareQrDataUrl" alt="二维码" width="200" height="200" />
      </div>
      <template #footer>
        <el-button @click="showShareDialog = false">关闭</el-button>
        <el-button type="primary" @click="copyShareLink">复制链接</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="showChangePwd" title="修改密码" width="420px" top="30vh">
      <el-form label-position="top">
        <el-form-item label="原密码">
          <el-input v-model="oldPwd" type="password" show-password />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="newPwd" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showChangePwd = false">取消</el-button>
        <el-button type="primary" :loading="changingPwd" @click="onChangePwd">确认修改</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { adminListFiles, adminDeleteFile, adminChangePassword, adminLogout, adminCheck } from '../api/admin'
import { adminGetSettings, adminUpdateSettings } from '../api/settings'
import { formatDateTime } from '../utils/time'
import { isHttpError } from '../utils/request'
import type { ManageFile } from '../types'
import QRCode from 'qrcode'

const router = useRouter()
const loading = ref(true)
const files = ref<ManageFile[]>([])
const keyword = ref('')
const currentPage = ref(1)
const pageSize = ref(10)
const total = ref(0)
const showChangePwd = ref(false)
const oldPwd = ref('')
const newPwd = ref('')
const changingPwd = ref(false)
const showShareDialog = ref(false)
const shareLink = ref('')
const shareQrDataUrl = ref('')
const maxFileSizeText = ref('30MB')
const savingSettings = ref(false)
const baseUrl = ref('')

function formatSize(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function parseSize(value: string) {
  const match = value.trim().match(/^(\d+(?:\.\d+)?)\s*(KB|MB|GB)$/i)
  if (!match) return 0
  const amount = Number(match[1])
  const unit = match[2].toUpperCase()
  const multiplier = unit === 'GB' ? 1024 ** 3 : unit === 'MB' ? 1024 ** 2 : 1024
  return Math.floor(amount * multiplier)
}

async function loadSettings() {
  try {
    const res = await adminGetSettings()
    if (res.data) {
      maxFileSizeText.value = formatSize(res.data.max_file_size)
      baseUrl.value = res.data.base_url || ''
    }
  } catch (e: any) {
    ElMessage.error(e.message || '加载设置失败')
  }
}

async function onSaveSettings() {
  const maxFileSize = parseSize(maxFileSizeText.value)
  if (!maxFileSize) {
    ElMessage.warning('请输入有效大小，例如 30MB 或 1GB')
    return
  }
  savingSettings.value = true
  try {
    const res = await adminUpdateSettings(maxFileSize, baseUrl.value)
    if (res.data) {
      maxFileSizeText.value = formatSize(res.data.max_file_size)
      baseUrl.value = res.data.base_url || ''
    }
    ElMessage.success('设置已保存')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    savingSettings.value = false
  }
}

async function loadFiles() {
  loading.value = true
  try {
    const res = await adminListFiles(keyword.value, currentPage.value, pageSize.value)
    const data = res.data
    if (data) {
      files.value = data.files || []
      total.value = data.total
      currentPage.value = data.page
      pageSize.value = data.page_size
    }
  } catch (e: any) {
    // 401 = not logged in / session expired — redirect to login
    if (isHttpError(e, 401)) {
      router.push('/admin/login')
      return
    }
    ElMessage.error(e.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function onSearch() {
  currentPage.value = 1
  loadFiles()
}

function onPageSizeChange(size: number) {
  pageSize.value = size
  currentPage.value = 1
  loadFiles()
}

function showShareLink(code: string) {
  const base = baseUrl.value.trim().replace(/\/+$/, '') || window.location.origin
  shareLink.value = `${base}/#/s/${encodeURIComponent(code)}`
  // Generate QR code asynchronously
  QRCode.toDataURL(shareLink.value, { width: 400, margin: 1 }).then((url) => {
    shareQrDataUrl.value = url
  }).catch(() => {
    shareQrDataUrl.value = ''
  })
  showShareDialog.value = true
}

async function copyShareLink() {
  try {
    await navigator.clipboard.writeText(shareLink.value)
    ElMessage.success('链接已复制')
  } catch {
    ElMessage.error('复制失败，请手动复制')
  }
}

async function onDelete(id: number) {
  try {
    await adminDeleteFile(id)
    const remainingTotal = Math.max(0, total.value - 1)
    const lastPage = Math.max(1, Math.ceil(remainingTotal / pageSize.value))
    currentPage.value = Math.min(currentPage.value, lastPage)
    await loadFiles()
    ElMessage.success('已删除')
  } catch (e: any) {
    ElMessage.error(e.message || '删除失败')
  }
}

async function onChangePwd() {
  if (!oldPwd.value || !newPwd.value) {
    ElMessage.warning('请填写完整')
    return
  }
  changingPwd.value = true
  try {
    await adminChangePassword(oldPwd.value, newPwd.value)
    ElMessage.success('密码已修改，请使用新密码重新登录')
    showChangePwd.value = false
    oldPwd.value = ''
    newPwd.value = ''
    router.push('/admin/login')
  } catch (e: any) {
    ElMessage.error(e.message || '修改失败')
  } finally {
    changingPwd.value = false
  }
}

async function onLogout() {
  try {
    await adminLogout()
  } catch {
    // Even if server call fails, still redirect to login
  }
  router.push('/admin/login')
}

async function checkAuth() {
  try {
    await adminCheck()
  } catch {
    router.push('/admin/login')
    return false
  }
  return true
}

onMounted(async () => {
  const authed = await checkAuth()
  if (authed) {
    await Promise.all([
      loadFiles(),
      loadSettings(),
    ])
  }
})
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.search-bar {
  display: flex;
  gap: 8px;
  margin-top: 8px;
  align-items: center;
}
.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  padding: 16px 12px;
  border-top: 1px solid var(--border-color);
}
.receive-code-cell {
  color: var(--color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.06em;
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
.setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.setting-label {
  font-size: 14px;
  color: var(--text-primary);
}
.setting-hint {
  margin-top: 4px;
  color: var(--text-secondary);
  font-size: 13px;
}
.setting-control {
  display: flex;
  align-items: center;
  gap: 8px;
}
.settings-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
