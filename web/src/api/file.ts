import request from '../utils/request'
import type { ApiResponse, UploadResult, FileInfo, ManageFile } from '../types'

export function uploadFile(
  file: File,
  password?: string,
  expireHours?: number,
  onProgress?: (percent: number) => void,
  usageCode?: string
): Promise<ApiResponse<UploadResult>> {
  const form = new FormData()
  form.append('file', file)
  if (password) form.append('password', password)
  if (expireHours) form.append('expire_hours', String(expireHours))
  if (usageCode) form.append('usage_code', usageCode)

  const headers: Record<string, string> = { 'Content-Type': 'multipart/form-data' }
  const token = localStorage.getItem('admin_token')
  if (token) headers['X-Admin-Token'] = token

  return request.post('/files', form, {
    headers,
    onUploadProgress: (e) => {
      if (e.total && onProgress) {
        onProgress(Math.round((e.loaded / e.total) * 100))
      }
    },
  })
}

export function getFileInfo(code: string): Promise<ApiResponse<FileInfo>> {
  return request.get(`/files/${code}`)
}

export function verifyPassword(code: string, password: string): Promise<ApiResponse<{ download_token: string }>> {
  return request.post(`/files/${code}/verify`, { password })
}

export function getDownloadUrl(code: string, token: string): string {
  const base = import.meta.env.VITE_APP_BASE_API || '/api'
  return `${base}/files/${code}/download?token=${encodeURIComponent(token)}`
}

export function getManageFiles(token: string): Promise<ApiResponse<{ files: ManageFile[] }>> {
  return request.get('/manage/files', { headers: { 'X-Manage-Token': token } })
}

export function deleteManageFile(id: number, token: string): Promise<ApiResponse<void>> {
  return request.delete(`/manage/files/${id}`, { headers: { 'X-Manage-Token': token } })
}
