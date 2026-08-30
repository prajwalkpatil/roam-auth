import { ping } from "@/api/requests"
import AlertWrapper from "@/components/AlertWrapper"
import { Spinner } from "@/components/ui/spinner"
import type { AxiosError } from "axios"
import { useEffect, useState } from "react"

export default function Profile() {
  const [message, setMessage] = useState("")
  const [error, setError] = useState("")
  const [fetched, setFetched] = useState(false)

  useEffect(() => {
    ping()
      .then((res) => setMessage(res))
      .catch((error: AxiosError) => {
        if (error.status != 401) setError(error.toString())
      })
      .finally(() => setFetched(true))
  }, [])

  return (
    <div className="flex h-screen items-center justify-center">
      <div>{!fetched && <Spinner />}</div>
      <div>{message}</div>
      {error && (
        <AlertWrapper
          title="Something went wrong"
          description={error}
        ></AlertWrapper>
      )}
    </div>
  )
}
