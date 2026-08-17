import axios from "axios"
import type { AxiosResponse } from "axios"
import type { LoginRequest, LoginResponse } from "./types"

const BASE_URL = "http://localhost:8000"

let authToken: string | null = null

const MAX_RETRIES = 3

const api = axios.create({
  baseURL: BASE_URL,
  withCredentials: true,
})

api.interceptors.request.use(
  (config) => {
    if (authToken) config.headers.set("Authorization", `Bearer ${authToken}`)
    return config
  },
  (error) => {
    console.log("error :>> ", error)
  }
)

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
    const data = await refresh()
    config.headers.set("Authorization", `Bearer ${data.token}`)
    return api(config)
  }
)

async function refresh(): Promise<LoginResponse> {
  const url = new URL(BASE_URL)
  url.pathname = "refresh"
  const response = await axios.post(url.toString(), null, {
    withCredentials: true,
  })
  const data = response?.data as LoginResponse
  authToken = data.id
  return data
}

export async function ping(): Promise<AxiosResponse> {
  const response = await api.get("/")
  return response?.data
}

export async function login(payload: LoginRequest): Promise<LoginResponse> {
  const response = await api.post("/login", payload)
  const data = response?.data as LoginResponse
  authToken = data.token
  return data
}
