import { Button } from "@/components/ui/button"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Label } from "@/components/ui/label"

import { Link, useNavigate } from "react-router-dom"
import * as z from "zod"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { login } from "@/api/auth"

const Credentials = z.object({
  email: z.email({
    error: (issue) =>
      issue.input === ""
        ? "Email should not be empty"
        : "Invalid Email Address",
  }),
  password: z.string().min(1, "Password should not be empty"),
})

export default function Login() {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(Credentials),
  })

  const navigate = useNavigate()

  const onSuccess = (data: z.infer<typeof Credentials>) => {
    login(data)
      .then(() => {
        navigate(-1)
      })
      .catch((e) => console.error(e))
  }

  return (
    <div className="flex h-screen items-center justify-center">
      <Card className="w-3/4 md:w-1/2 lg:w-1/3 xl:w-2/7">
        <CardHeader>
          <CardTitle>Login</CardTitle>
          <CardDescription>
            Enter your details below to login to your account
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onSuccess)}>
            <div className="mt-3 flex flex-col gap-6">
              <Field>
                <FieldLabel htmlFor="input-email">Email</FieldLabel>
                <Input
                  id="input-email"
                  type="text"
                  placeholder="Enter your Email"
                  className={`${errors.email ? "border-destructive focus-visible:border-destructive" : ""}`}
                  {...register("email")}
                />
                {errors?.email && (
                  <Label className="mt-1 text-destructive">
                    {errors.email.message}
                  </Label>
                )}
              </Field>
              <Field>
                <FieldLabel
                  htmlFor="input-password"
                  className="flex justify-between"
                >
                  Password
                  <a className="text-end text-muted-foreground underline hover:text-foreground">
                    Forgot Password?
                  </a>
                </FieldLabel>
                <Input
                  id="input-password"
                  type="password"
                  placeholder="Enter your Password"
                  className={`${errors.password ? "border-destructive focus-visible:border-destructive" : ""}`}
                  {...register("password")}
                />
                {errors?.password && (
                  <Label className="mt-1 text-destructive">
                    {errors.password.message}
                  </Label>
                )}
              </Field>
              <Button type="submit">Login</Button>
            </div>
          </form>
          <Label className="flex flex-1 justify-center pt-5 text-muted-foreground">
            <div>
              Don't have an account?{" "}
              <Link to="/signup" className="underline hover:text-foreground">
                Sign up
              </Link>
            </div>
          </Label>
        </CardContent>
      </Card>
    </div>
  )
}
