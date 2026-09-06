import { useCallback, useEffect, useState, type PropsWithChildren } from "react"
import { AuthContext } from "./AuthContext"
import type { ProfileResponse } from "@/api/types"
import { getProfile } from "@/api/requests"

export default function AuthProvider({
  children,
}: PropsWithChildren): React.ReactNode {
  const [user, setUser] = useState<ProfileResponse | null>(null)
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [isLoading, setIsLoading] = useState(true)

  const loginContextUser = useCallback((profile: ProfileResponse) => {
    setUser(profile)
    setIsAuthenticated(true)
    setIsLoading(false)
  }, [])

  const logoutContextUser = useCallback(() => {
    setUser(null)
    setIsAuthenticated(false)
    setIsLoading(false)
  }, [])

  useEffect(() => {
    ;(async () => {
      setIsLoading(true)
      try {
        const profile = await getProfile()
        setUser(profile)
        setIsAuthenticated(true)
      } catch {
        setIsAuthenticated(false)
      } finally {
        setIsLoading(false)
      }
    })()
  }, [])

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated,
        isLoading,
        loginContextUser,
        logoutContextUser,
        setIsLoading,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}
