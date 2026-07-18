export interface FileInfo {
  code: string
  file_name: string
  file_size: number
  download_count: number
  has_password: boolean
  expire_at: string | null
  created_at: string
}

export interface UploadResult {
  code: string
  file_name: string
  file_size: number
  manage_token: string
  has_password: boolean
  expire_at: string | null
  created_at: string
}

export interface ManageFile {
  id: number
  code: string
  file_name: string
  file_size: number
  download_count: number
  has_password: boolean
  expire_at: string | null
  created_at: string
}

export interface ApiResponse<T = any> {
  code: number
  message: string
  data?: T
}
