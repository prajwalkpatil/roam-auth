import { logout } from "@/api/requests"
import { getProfile } from "@/api/requests"
import type { ProfileResponse } from "@/api/types"
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
import { useNavigate } from "react-router-dom"

export default function Profile() {
  const navigate = useNavigate()
  const [profile, setProfile] = useState<ProfileResponse | null>(null)
  const [error, setError] = useState("")
  const [fetched, setFetched] = useState(false)

  useEffect(() => {
    getProfile()
      .then((res) => setProfile(res))
      .catch((error: AxiosError) => {
        if (error.status != 401) setError(error.toString())
      })
      .finally(() => setFetched(true))
  }, [])

  const logoutUser = () => {
    logout()
      .then((ok) => {
        if (ok) {
          navigate("/login")
        } else {
          setError("Couldn't logout user")
        }
      })
      .catch((err) => {
        setError(err.toString())
      })
  }

  return (
    <div className="flex h-screen items-center justify-center">
      <div>{!fetched && <Spinner />}</div>
      {fetched && !error && profile && (
        <div className="flex w-3/4 selection:bg-sidebar-primary selection:text-background md:w-1/2 lg:w-1/3 xl:w-2/7">
          <Card className="flex-1">
            <CardHeader>
              <CardTitle>Profile</CardTitle>
              <CardDescription>You're now logged in.</CardDescription>
            </CardHeader>
            <CardContent className="mt-2">
              <div className="flex flex-col rounded-md bg-accent px-3 py-2 text-[0.8rem]">
                <div className="flex items-center justify-between gap-3 border-b pb-1.5">
                  <div className="font-semibold text-muted-foreground">
                    Name
                  </div>
                  <div className="overflow-y-auto text-end wrap-break-word text-secondary-foreground">
                    {profile?.name}
                  </div>
                </div>
                <div className="flex items-center justify-between gap-3 pt-1.5">
                  <div className="font-semibold text-muted-foreground">
                    Email
                  </div>
                  <div className="overflow-y-auto text-end wrap-break-word text-secondary-foreground">
                    {profile?.email}
                  </div>
                </div>
              </div>
            </CardContent>
            <CardFooter>
              <Button
                variant="destructive"
                className="w-full"
                onClick={logoutUser}
              >
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
