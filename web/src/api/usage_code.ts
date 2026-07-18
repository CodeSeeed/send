import request from '../utils/request'
import type { ApiResponse } from '../types'

export interface UsageCode {
  id: number
  code: string
  remark: string
  max_file_size: number
  max_uses: number
  used_count: number
  expire_at: string | null
  created_at: string
}

export function createUsageCode(remark: string, maxFileSize: string, maxUses: number, expireHours: number): Promise<ApiResponse<{ code: string }>> {
  const token = localStorage.getItem('admin_token')
  return request.post('/admin/usage-codes', { remark, max_file_size: maxFileSize, max_uses: maxUses, expire_hours: expireHours }, {
    headers: { 'X-Admin-Token': token || '' },
  })
}

export function listUsageCodes(): Promise<ApiResponse<{ codes: UsageCode[] }>> {
  const token = localStorage.getItem('admin_token')
  return request.get('/admin/usage-codes', { headers: { 'X-Admin-Token': token || '' } })
}

export function deleteUsageCode(id: number): Promise<ApiResponse<void>> {
  const token = localStorage.getItem('admin_token')
  return request.delete(`/admin/usage-codes/${id}`, { headers: { 'X-Admin-Token': token || '' } })
}
