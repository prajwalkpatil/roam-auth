import * as z from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useForm } from "react-hook-form"
import { Link } from "react-router-dom"

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
import { signup } from "@/api/auth"
import type { AxiosError } from "axios"
import { ErrorEnum, type ErrorResponse } from "@/api/types"

const User = z
  .object({
    name: z
      .string()
      .min(1, "Name should not be empty")
      .min(2, "Name must be atleast two characters"),
    email: z.email({
      error: (issue) =>
        issue.input === ""
          ? "Email should not be empty"
          : "Invalid Email Address",
    }),
    password: z
      .string()
      .min(1, "Password should not be empty")
      .min(6, "Password should be atleast 6 characters"),
    confirmPassword: z.string(),
  })
  .refine((data) => data.password === data.confirmPassword, {
    error: "Passwords do not match",
    path: ["confirmPassword"],
  })

export default function Signup() {
  const {
    register,
    setError,
    handleSubmit,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(User),
  })

  function onSuccess(data: z.infer<typeof User>) {
    signup(data)
      .then(() => (window.location.href = "/login"))
      .catch((e: AxiosError) => {
        //TODO: Show an error dialog
        const err = e?.response?.data as ErrorResponse
        if (err.error === ErrorEnum.EMAIL_ALREADY_EXISTS) {
          setError(
            "email",
            {
              message: "Email already registered",
            },
            { shouldFocus: true }
          )
        }
        console.error("Signup error", err)
      })
  }

  return (
    <div className="flex h-screen items-center justify-center">
      <Card className="w-3/4 md:w-1/2 lg:w-1/3 xl:w-2/7">
        <CardHeader>
          <CardTitle>Sign Up</CardTitle>
          <CardDescription>
            Enter your details to create a new account
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onSuccess)}>
            <div className="mt-3 flex flex-col gap-5">
              <Field>
                <FieldLabel htmlFor="input-name">Name</FieldLabel>
                <Input
                  id="input-name"
                  type="text"
                  placeholder="Enter your Name"
                  className={`${errors.name ? "border-destructive focus-visible:border-destructive" : ""}`}
                  {...register("name")}
                />
                {errors?.name && (
                  <Label className="mt-1 text-destructive">
                    {errors.name.message}
                  </Label>
                )}
              </Field>
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
              <Field>
                <FieldLabel
                  htmlFor="input-confirm-password"
                  className="flex justify-between"
                >
                  Confirm Password
                </FieldLabel>
                <Input
                  id="input-confirm-password"
                  type="password"
                  placeholder="Retype your Password"
                  className={`${errors.confirmPassword ? "border-destructive focus-visible:border-destructive" : ""}`}
                  {...register("confirmPassword")}
                />
                {errors?.confirmPassword && (
                  <Label className="mt-1 text-destructive">
                    {errors.confirmPassword.message}
                  </Label>
                )}
              </Field>
              <Button className="mt-2" type="submit">
                Sign Up
              </Button>
            </div>
          </form>
          <Label className="flex flex-1 justify-center pt-5 text-muted-foreground">
            <div>
              Already have an account?{" "}
              <Link to="/login" className="underline hover:text-foreground">
                Login
              </Link>
            </div>
          </Label>
        </CardContent>
      </Card>
    </div>
  )
}
