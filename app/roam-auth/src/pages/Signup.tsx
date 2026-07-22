import { Button } from "@/components/ui/button"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Label } from "@/components/ui/label"

import { Link } from "react-router-dom"


export default function Signup() {
  return (
    <div className="flex justify-center border-2 h-screen items-center">
      <Card className="xl:w-2/7 lg:w-1/3 md:w-1/2 w-3/4">
        <CardHeader>
          <CardTitle>Sign Up</CardTitle>
          <CardDescription>Enter your details to create a new account</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col gap-5 pt-3">
            <Field>
              <FieldLabel htmlFor="input-name">Name</FieldLabel>
              <Input id="input-name" type="text" placeholder="Enter your Name"/>
            </Field>
            <Field>
              <FieldLabel htmlFor="input-email">Email</FieldLabel>
              <Input id="input-email" type="text" placeholder="Enter your Email" />
            </Field>
            <Field>
              <FieldLabel htmlFor="input-password" className="flex justify-between">
                Password
              </FieldLabel>
              <Input id="input-password" type="password" placeholder="Enter your Password" />
            </Field>
            <Button className="mt-2">Sign Up</Button>
          </div>
          <Label className="text-muted-foreground pt-5 justify-center flex flex-1">
            <div>
              Already have an account? <Link to="/login"className="underline hover:text-foreground">Login</Link>
            </div>
          </Label>
        </CardContent>
      </Card>
    </div>

  );
}