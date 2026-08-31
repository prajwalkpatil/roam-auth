export interface LoginRequest {
  email: string
  password: string
}
export interface SignupRequest {
  name: string
  email: string
  password: string
}

export interface LoginResponse {
  id: string
  token: string
  email: string
}

export interface ProfileResponse {
  name: string
  email: string
}
