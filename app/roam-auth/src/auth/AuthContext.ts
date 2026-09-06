import type { ProfileResponse } from "@/api/types"
import { createContext } from "react"

interface AuthContextType {
  user: ProfileResponse | null
  isAuthenticated: boolean
  isLoading: boolean
  loginContextUser: (profile: ProfileResponse) => void
  logoutContextUser: () => void
}

export const AuthContext = createContext<AuthContextType | null>(null)
