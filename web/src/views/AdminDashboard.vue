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
    <div class="card" style="padding: 0; margin-top: 8px">
      <el-table :data="files" v-loading="loading" empty-text="暂无文件">
        <el-table-column prop="id" label="ID" width="60" align="center" />
        <el-table-column prop="file_name" label="文件名" min-width="160" show-overflow-tooltip />
        <el-table-column label="大小" width="80">
          <template #default="{ row }">{{ formatSize(row.file_size) }}</template>
        </el-table-column>
        <el-table-column label="下载" width="60" prop="download_count" align="center" />
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
import { adminListFiles, adminDeleteFile, adminChangePassword } from '../api/admin'
import { adminGetSettings, adminUpdateSettings } from '../api/settings'
import { formatDateTime } from '../utils/time'
import type { ManageFile } from '../types'

const router = useRouter()
const loading = ref(true)
const files = ref<ManageFile[]>([])
const showChangePwd = ref(false)
const oldPwd = ref('')
const newPwd = ref('')
const changingPwd = ref(false)
const showShareDialog = ref(false)
const shareLink = ref('')
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
  try {
    const res = await adminListFiles()
    files.value = res.data?.files || []
  } catch (e: any) {
    if (e.message?.includes('未登录') || e.message?.includes('登录已过期')) {
      localStorage.removeItem('admin_token')
      router.push('/admin/login')
      return
    }
    ElMessage.error(e.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function showShareLink(code: string) {
  const base = baseUrl.value.trim().replace(/\/+$/, '') || window.location.origin
  shareLink.value = `${base}/#/s/${encodeURIComponent(code)}`
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
    files.value = files.value.filter((f) => f.id !== id)
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
    localStorage.removeItem('admin_token')
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

function onLogout() {
  localStorage.removeItem('admin_token')
  router.push('/admin/login')
}

onMounted(() => {
  loadFiles()
  loadSettings()
})
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
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
