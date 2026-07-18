<template>
  <div class="admin-page">
    <div class="page-header">
      <h1 class="page-title" style="margin-bottom: 0">管理后台</h1>
      <div style="display: flex; gap: 8px">
        <el-button @click="showChangePwd = true">修改密码</el-button>
        <el-button @click="onLogout">退出登录</el-button>
      </div>
    </div>

    <!-- Files -->
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
            <span v-if="row.expire_at" style="color: var(--text-secondary); font-size: 13px">{{ row.expire_at.slice(0, 16) }}</span>
            <el-tag v-else type="success" size="small" effect="plain">永不过期</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="160">
          <template #default="{ row }">
            <span style="color: var(--text-secondary); font-size: 13px">{{ row.created_at.slice(0, 16) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80" fixed="right">
          <template #default="{ row }">
            <el-popconfirm title="确定删除？" @confirm="onDelete(row.id)">
              <template #reference>
                <el-button type="danger" link size="small" style="color: var(--color-danger)">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Usage Codes -->
    <div style="display: flex; align-items: center; gap: 12px; margin-top: 32px">
      <div class="section-label" style="margin-bottom: 0">使用码管理</div>
      <el-button size="small" @click="showCreateCode = true">创建使用码</el-button>
    </div>
    <div class="card" style="padding: 0; margin-top: 8px">
      <el-table :data="codes" v-loading="codesLoading" empty-text="暂无使用码">
        <el-table-column label="备注" prop="remark" min-width="120" show-overflow-tooltip />
        <el-table-column label="使用码" prop="code" width="200" />
        <el-table-column label="使用次数" width="100">
          <template #default="{ row }">
            {{ row.used_count }} / {{ row.max_uses || '∞' }}
          </template>
        </el-table-column>
        <el-table-column label="大小限制" width="100">
          <template #default="{ row }">
            {{ row.max_file_size ? formatSize(row.max_file_size) : '不限' }}
          </template>
        </el-table-column>
        <el-table-column label="过期时间" width="160">
          <template #default="{ row }">
            <span v-if="row.expire_at" style="color: var(--text-secondary); font-size: 13px">{{ row.expire_at.slice(0, 16) }}</span>
            <el-tag v-else type="success" size="small" effect="plain">永不过期</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="160">
          <template #default="{ row }">
            <span style="color: var(--text-secondary); font-size: 13px">{{ row.created_at.slice(0, 16) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80" fixed="right">
          <template #default="{ row }">
            <el-popconfirm title="确定删除？" @confirm="onDeleteCode(row.id)">
              <template #reference>
                <el-button type="danger" link size="small" style="color: var(--color-danger)">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Change password dialog -->
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

    <!-- Create usage code dialog -->
    <el-dialog v-model="showCreateCode" title="创建使用码" width="420px" top="25vh">
      <el-form label-position="top">
        <el-form-item label="备注">
          <el-input v-model="newRemark" placeholder="用于标识此使用码的用途" />
        </el-form-item>
        <el-form-item label="限制文件大小（如 1GB200MB500kb，留空不限制）">
          <el-input v-model="newMaxSizeStr" placeholder="例: 100MB、1GB、500MB200kb" />
        </el-form-item>
        <el-form-item label="最大使用次数（0 表示不限）">
          <el-input-number v-model="newMaxUses" :min="0" style="width: 100%" />
        </el-form-item>
        <el-form-item label="过期时间">
          <el-select v-model="newExpireHours" style="width: 100%">
            <el-option label="永不过期" :value="0" />
            <el-option label="1 小时后" :value="1" />
            <el-option label="3 小时后" :value="3" />
            <el-option label="1 天后" :value="24" />
            <el-option label="3 天后" :value="72" />
            <el-option label="7 天后" :value="168" />
            <el-option label="30 天后" :value="720" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateCode = false">取消</el-button>
        <el-button type="primary" :loading="creatingCode" @click="onCreateCode">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { adminListFiles, adminDeleteFile, adminChangePassword } from '../api/admin'
import { listUsageCodes, createUsageCode, deleteUsageCode, type UsageCode } from '../api/usage_code'
import type { ManageFile } from '../types'

const router = useRouter()

// Files
const loading = ref(true)
const files = ref<ManageFile[]>([])

// Password
const showChangePwd = ref(false)
const oldPwd = ref('')
const newPwd = ref('')
const changingPwd = ref(false)

// Usage codes
const codes = ref<UsageCode[]>([])
const codesLoading = ref(true)
const showCreateCode = ref(false)
const newRemark = ref('')
const newMaxSizeStr = ref('')
const newMaxUses = ref(0)
const newExpireHours = ref(0)
const creatingCode = ref(false)

function formatSize(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
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

async function loadCodes() {
  try {
    const res = await listUsageCodes()
    codes.value = res.data?.codes || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载使用码失败')
  } finally {
    codesLoading.value = false
  }
}

async function onCreateCode() {
  if (!newRemark.value) {
    ElMessage.warning('请填写备注')
    return
  }
  if (newMaxUses.value <= 0 && newExpireHours.value <= 0) {
    ElMessage.warning('必须设置使用次数或过期时间')
    return
  }
  creatingCode.value = true
  try {
    await createUsageCode(newRemark.value, newMaxSizeStr.value, newMaxUses.value, newExpireHours.value)
    ElMessage.success('使用码已创建')
    showCreateCode.value = false
    newRemark.value = ''
    newMaxSizeStr.value = ''
    newMaxUses.value = 0
    newExpireHours.value = 0
    loadCodes()
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    creatingCode.value = false
  }
}

async function onDeleteCode(id: number) {
  try {
    await deleteUsageCode(id)
    codes.value = codes.value.filter((c) => c.id !== id)
    ElMessage.success('已删除')
  } catch (e: any) {
    ElMessage.error(e.message || '删除失败')
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
    ElMessage.success('密码已修改')
    showChangePwd.value = false
    oldPwd.value = ''
    newPwd.value = ''
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
  loadCodes()
})
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
