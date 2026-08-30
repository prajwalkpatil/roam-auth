import axios, { type AxiosResponse } from "axios"
import type { LoginRequest, LoginResponse } from "./types"
import { BASE_URL } from "./constants"

const auth: { token?: string } = {}

export const setToken = (token: string) => {
  auth.token = token
}

export const getToken = () => auth?.token

const authClient = axios.create({
  baseURL: BASE_URL,
})

export async function login(payload: LoginRequest): Promise<LoginResponse> {
  const response = await authClient.post("login", payload, {
    withCredentials: true,
  })
  const data = response?.data as LoginResponse
  setToken(data.token)
  return data
}

export async function refresh(): Promise<AxiosResponse> {
  return authClient.post("refresh", null, {
    withCredentials: true,
  })
}
