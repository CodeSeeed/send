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

  return request.post('/files', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
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

// Download via fetch + blob to support Authorization header instead of query param
export async function downloadFile(code: string, token: string): Promise<void> {
  const base = import.meta.env.VITE_APP_BASE_API || '/api'
  const url = `${base}/files/${code}/download`
  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!response.ok) {
    const err = await response.json().catch(() => ({ message: '下载失败' }))
    throw new Error(err.message || '下载失败')
  }
  const blob = await response.blob()
  // Extract filename from Content-Disposition header
  const disposition = response.headers.get('Content-Disposition') || ''
  let filename = 'download'
  const match = disposition.match(/filename="?([^";]+)"?/)
  if (match) filename = decodeURIComponent(match[1])
  // Trigger download
  const blobUrl = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = blobUrl
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  window.URL.revokeObjectURL(blobUrl)
}