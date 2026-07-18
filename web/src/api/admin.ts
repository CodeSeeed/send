import request from '../utils/request'
import type { ApiResponse, ManageFile } from '../types'

export function adminLogin(username: string, password: string): Promise<ApiResponse<{ token: string }>> {
  return request.post('/admin/login', { username, password })
}

export function adminChangePassword(oldPwd: string, newPwd: string): Promise<ApiResponse<void>> {
  const token = localStorage.getItem('admin_token')
  return request.post('/admin/password', { old_password: oldPwd, new_password: newPwd }, {
    headers: { 'X-Admin-Token': token || '' },
  })
}

export function adminListFiles(): Promise<ApiResponse<{ files: ManageFile[] }>> {
  const token = localStorage.getItem('admin_token')
  return request.get('/admin/files', { headers: { 'X-Admin-Token': token || '' } })
}

export function adminDeleteFile(id: number): Promise<ApiResponse<void>> {
  const token = localStorage.getItem('admin_token')
  return request.delete(`/admin/files/${id}`, { headers: { 'X-Admin-Token': token || '' } })
}
