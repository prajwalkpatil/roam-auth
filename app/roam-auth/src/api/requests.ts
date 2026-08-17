import axios from "axios"
import type { AxiosResponse } from "axios"
import type { LoginRequest, LoginResponse } from "./types"

const BASE_URL = "http://localhost:8000"

export async function ping(): Promise<AxiosResponse> {
  const url = new URL(BASE_URL)
  const response = await axios.get(url.toString())
  return response?.data
}

export async function login(payload: LoginRequest): Promise<LoginResponse> {
  const url = new URL(BASE_URL)
  url.pathname = "login"
  const response = await axios.post(url.toString(), payload, {
    withCredentials: true,
  })
  return response?.data
}
