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

export const ErrorEnum = {
  EMAIL_DOES_NOT_EXIST: "EMAIL_DOES_NOT_EXIST",
  EMAIL_ALREADY_EXISTS: "EMAIL_ALREADY_EXISTS",
  INVALID_PASSWORD: "INVALID_PASSWORD",
}

export interface ErrorResponse {
  status: number
  error: string
}
