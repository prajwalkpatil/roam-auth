import Loading from "@/components/Loading"
import { router } from "@/router/router"
import { useEffect, type PropsWithChildren } from "react"

export default function PublicRoute({
  children,
}: PropsWithChildren): React.ReactNode {
  const isAuthenticated = false
  const isLoading = false

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      router.navigate("/", { replace: true })
    }
  }, [isAuthenticated, isLoading])

  return isLoading || isAuthenticated ? <Loading /> : children
}
