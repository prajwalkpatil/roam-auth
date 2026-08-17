import axios from "axios"
import type { AxiosResponse } from "axios"
import type { LoginRequest, LoginResponse } from "./types"

const BASE_URL = "http://localhost:8000"

const auth: { token?: string } = {}

const MAX_RETRIES = 3

const api = axios.create({
  baseURL: BASE_URL,
  withCredentials: true,
})

const setToken = (token: string) => {
  auth.token = token
}
const getToken = () => auth?.token

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

async function refresh(): Promise<AxiosResponse> {
  const url = new URL(BASE_URL)
  url.pathname = "refresh"
  return axios.post(url.toString(), null, {
    withCredentials: true,
  })
}

export async function ping(): Promise<string> {
  const response = await api.get("/")
  return response?.data
}

export async function login(payload: LoginRequest): Promise<LoginResponse> {
  const url = new URL(BASE_URL)
  url.pathname = "login"
  const response = await axios.post(url.toString(), payload, {
    withCredentials: true,
  })
  const data = response?.data as LoginResponse
  setToken(data.token)
  return data
}
