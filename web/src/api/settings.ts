import request from '../utils/request'
import type { ApiResponse } from '../types'

export interface SystemSettings {
  id: number
  max_file_size: number
  default_expire_hours: number
  base_url: string
  updated_at: string
}

export function adminGetSettings(): Promise<ApiResponse<SystemSettings>> {
  return request.get('/admin/settings')
}

export function adminUpdateSettings(maxFileSize: number, baseUrl: string): Promise<ApiResponse<SystemSettings>> {
  return request.put('/admin/settings', { max_file_size: maxFileSize, base_url: baseUrl })
}