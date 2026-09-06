import Loading from "@/components/Loading"
import { router } from "@/router/router"
import { useEffect, type PropsWithChildren } from "react"
import useAuth from "@/auth/useAuth"

export default function ProtectedRoute({
  children,
}: PropsWithChildren): React.ReactNode {
  const { isLoading, isAuthenticated } = useAuth()

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.navigate("/login", { replace: true })
    }
  }, [isAuthenticated, isLoading])

  return isLoading || !isAuthenticated ? <Loading /> : children
}
