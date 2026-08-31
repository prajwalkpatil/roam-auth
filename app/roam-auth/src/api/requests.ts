import axios from "axios"
import type { LoginResponse } from "./types"
import { BASE_URL, MAX_RETRIES } from "./constants"
import { getToken, refresh, setToken } from "./auth"

const api = axios.create({
  baseURL: BASE_URL,
  withCredentials: true,
})

api.interceptors.request.use((config) => {
  const token = getToken()
  if (token) config.headers.set("Authorization", `Bearer ${token}`)
  return config
})

api.interceptors.response.use(
  (config) => config,
  async (error) => {
    const config = error.config
    const shouldRetry = error.response?.status === 401
    if (!shouldRetry) {
      return Promise.reject(error)
    }
    if (config._retryCount >= MAX_RETRIES) {
      return Promise.reject(error)
    }
    config._retryCount = config._retryCount ? config._retryCount + 1 : 1
    try {
      const response = await refresh()
      const data = response?.data as LoginResponse
      config.headers.set("Authorization", `Bearer ${data.token}`)
      setToken(data.token)
      return api(config)
    } catch (error) {
      if (window.location.href != "login") window.location.href = "/login"
      return Promise.reject(error)
    }
  }
)

export async function ping(): Promise<string> {
  const response = await api.get("/")
  return response?.data
}

export async function getProfile(): Promise<string> {
  const response = await api.get("/profile")
  return response?.data
}
