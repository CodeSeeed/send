import request from '../utils/request'
import type { ApiResponse, ManageFile } from '../types'

export function adminStatus(): Promise<ApiResponse<{ registered: boolean }>> {
  return request.get('/admin/status')
}

export function adminRegister(username: string, password: string): Promise<ApiResponse<{ token: string }>> {
  return request.post('/admin/register', { username, password })
}

export function adminLogin(username: string, password: string): Promise<ApiResponse<{ token: string }>> {
  return request.post('/admin/login', { username, password })
}

export function adminCheck(): Promise<ApiResponse<{ admin_id: number }>> {
  return request.get('/admin/check')
}

export function adminLogout(): Promise<ApiResponse<void>> {
  return request.post('/admin/logout')
}

export function adminChangePassword(oldPwd: string, newPwd: string): Promise<ApiResponse<void>> {
  return request.post('/admin/password', { old_password: oldPwd, new_password: newPwd })
}

export function adminListFiles(): Promise<ApiResponse<{ files: ManageFile[] }>> {
  return request.get('/admin/files')
}

export function adminDeleteFile(id: number): Promise<ApiResponse<void>> {
  return request.delete(`/admin/files/${id}`)
}