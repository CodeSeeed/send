import axios, { AxiosError } from 'axios'

const request = axios.create({
  baseURL: import.meta.env.VITE_APP_BASE_API || '/api',
  timeout: 60000,
  withCredentials: true,
})

// Attach the HTTP status code to rejected errors so callers can branch on
// status instead of fragile message-text matching.
export function isHttpError(e: unknown, status: number): boolean {
  const error = e as AxiosError & { status?: number }
  return error?.status === status || error?.response?.status === status
}

request.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code !== 0) {
      const err = new Error(res.message || '请求失败') as Error & { status?: number }
      err.status = response.status
      return Promise.reject(err)
    }
    return res
  },
  (error) => {
    const status = error.response?.status as number | undefined
    const msg = error.response?.data?.message || error.message || '网络错误'
    const err = new Error(msg) as Error & { status?: number }
    err.status = status
    return Promise.reject(err)
  }
)

export default request
