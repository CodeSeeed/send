import request from '../utils/request'
import type { ApiResponse, UploadResult, PublicFileInfo } from '../types'

export function uploadFile(
  file: File,
  password?: string,
  expireHours?: number,
  expireMinutes?: number,
  maxDownloads?: number,
  generateReceiveCode?: boolean,
  onProgress?: (percent: number) => void,
): Promise<ApiResponse<UploadResult>> {
  const form = new FormData()
  form.append('file', file)
  if (password) form.append('password', password)
  if (expireHours !== undefined) form.append('expire_hours', String(expireHours))
  if (expireMinutes !== undefined) form.append('expire_minutes', String(expireMinutes))
  if (maxDownloads !== undefined && maxDownloads > 0) form.append('max_downloads', String(maxDownloads))
  if (generateReceiveCode) form.append('generate_receive_code', 'true')

  return request.post('/files', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 0, // no timeout — uploads can be large and slow
    onUploadProgress: (e) => {
      if (e.total && onProgress) {
        onProgress(Math.round((e.loaded / e.total) * 100))
      }
    },
  })
}

export function getFileInfo(code: string): Promise<ApiResponse<PublicFileInfo>> {
  return request.get(`/files/${code}`)
}

export function receiveFile(receiveCode: string): Promise<ApiResponse<PublicFileInfo>> {
  return request.post('/files/receive', { receive_code: receiveCode })
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
    const error = new Error(err.message || '下载失败') as Error & { status?: number }
    error.status = response.status
    throw error
  }
  const blob = await response.blob()
  const filename = extractFilename(response.headers.get('Content-Disposition') || '')
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

// Preview fetches the file inline for browser rendering (PDF, images, text).
// Unlike download, it does NOT consume the download token or increment the
// download counter, so the user can still download afterwards.
export async function previewFile(code: string, token: string): Promise<{ blob: Blob; mimeType: string }> {
  const base = import.meta.env.VITE_APP_BASE_API || '/api'
  const url = `${base}/files/${code}/preview`
  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!response.ok) {
    const err = await response.json().catch(() => ({ message: '预览失败' }))
    const error = new Error(err.message || '预览失败') as Error & { status?: number }
    error.status = response.status
    throw error
  }
  const blob = await response.blob()
  return { blob, mimeType: response.headers.get('Content-Type') || '' }
}

// Parse Content-Disposition, supporting both:
//   filename="example.pdf"          (plain ASCII)
//   filename*=UTF-8''%E6%B5%8B.pdf  (RFC 5987 / RFC 2231, used for non-ASCII names)
function extractFilename(disposition: string): string {
  // Prefer the RFC 5987 filename*= variant — it carries the original (non-ASCII) name
  const rfc5987 = disposition.match(/filename\*=([^']*)''([^;]+)/)
  if (rfc5987) {
    try {
      return decodeURIComponent(rfc5987[2])
    } catch {
      // fall through to the plain variant
    }
  }
  const plain = disposition.match(/filename="?([^";]+)"?/)
  if (plain) {
    try {
      return decodeURIComponent(plain[1])
    } catch {
      return plain[1]
    }
  }
  return 'download'
}
