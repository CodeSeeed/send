// PublicFileInfo mirrors the redacted server response from the file-info and
// receive-code endpoints. The server omits id, receive_code, download_count
// and max_downloads on these public routes, so this type reflects that.
export interface PublicFileInfo {
  code: string
  file_name: string
  file_size: number
  has_password: boolean
  expire_at: string | null
  created_at: string
}

export interface UploadResult {
  code: string
  receive_code?: string | null
  file_name: string
  file_size: number
  has_password: boolean
  max_downloads: number
  expire_at: string | null
  created_at: string
}

export interface ManageFile {
  id: number
  code: string
  receive_code?: string | null
  file_name: string
  file_size: number
  download_count: number
  max_downloads: number
  has_password: boolean
  expire_at: string | null
  created_at: string
}

export interface FileListResult {
  files: ManageFile[]
  total: number
  page: number
  page_size: number
}

export interface ApiResponse<T = any> {
  code: number
  message: string
  data?: T
}
