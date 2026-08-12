import request from '../utils/request'
import type { ApiResponse, UploadResult, FileInfo } from '../types'

export function uploadFile(
  file: File,
  password?: string,
  expireHours?: number,
  expireMinutes?: number,
  onProgress?: (percent: number) => void,
): Promise<ApiResponse<UploadResult>> {
  const form = new FormData()
  form.append('file', file)
  if (password) form.append('password', password)
  if (expireHours !== undefined) form.append('expire_hours', String(expireHours))
  if (expireMinutes !== undefined) form.append('expire_minutes', String(expireMinutes))

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
