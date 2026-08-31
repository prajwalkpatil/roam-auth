import { ping } from "@/api/requests"
import AlertWrapper from "@/components/AlertWrapper"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Spinner } from "@/components/ui/spinner"
import type { AxiosError } from "axios"
import { useEffect, useState } from "react"

export default function Profile() {
  const [, setMessage] = useState("")
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
      {fetched && !error && (
        <div className="flex w-3/4 md:w-1/2 lg:w-1/3 xl:w-2/7">
          <Card className="flex-1">
            <CardHeader>
              <CardTitle>Profile</CardTitle>
              <CardDescription>You’re now logged in.</CardDescription>
            </CardHeader>
            <CardContent className="mt-2">
              <div className="flex flex-col rounded-md bg-accent px-3 py-2 text-[0.8rem]">
                <div className="flex items-center justify-between gap-3 border-b pb-1.5">
                  <div className="font-semibold text-muted-foreground">
                    Name
                  </div>
                  <div className="overflow-y-auto text-end wrap-break-word text-secondary-foreground">
                    Prajwal Patil
                  </div>
                </div>
                <div className="flex items-center justify-between gap-3 pt-1.5">
                  <div className="font-semibold text-muted-foreground">
                    Email
                  </div>
                  <div className="overflow-y-auto text-end wrap-break-word text-secondary-foreground">
                    prajwalpatilk@gmail.com
                  </div>
                </div>
              </div>
            </CardContent>
            <CardFooter>
              <Button type="submit" className="w-full">
                Logout
              </Button>
            </CardFooter>
          </Card>
        </div>
      )}

      {error && (
        <AlertWrapper
          title="Something went wrong"
          description={error}
        ></AlertWrapper>
      )}
    </div>
  )
}
