import request from '../utils/request'
import type { ApiResponse } from '../types'

export interface SystemSettings {
  id: number
  max_file_size: number
  default_expire_hours: number
  base_url: string
  updated_at: string
}

export function getSettings(): Promise<ApiResponse<SystemSettings>> {
  return request.get('/settings')
}

export function adminGetSettings(): Promise<ApiResponse<SystemSettings>> {
  const token = localStorage.getItem('admin_token')
  return request.get('/admin/settings', { headers: { 'X-Admin-Token': token || '' } })
}

export function adminUpdateSettings(maxFileSize: number, baseUrl: string): Promise<ApiResponse<SystemSettings>> {
  const token = localStorage.getItem('admin_token')
  return request.put('/admin/settings', { max_file_size: maxFileSize, base_url: baseUrl }, {
    headers: { 'X-Admin-Token': token || '' },
  })
}
