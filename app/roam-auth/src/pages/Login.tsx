import { Button } from "@/components/ui/button"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Label } from "@/components/ui/label"

export default function Login() {
  return (
    <div className="flex justify-center border-2 h-screen items-center">
      <Card className="w-1/4">
        <CardHeader>
          <CardTitle>Login</CardTitle>
          <CardDescription>Enter your details below to login to your account</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col gap-6 pt-3">
            <Field>
              <FieldLabel htmlFor="input-email">Email</FieldLabel>
              <Input id="input-email" type="text" placeholder="Enter Email Address" />
            </Field>
            <Field>
              <FieldLabel htmlFor="input-password" className="flex justify-between">
                Password
                <a className="text-end text-muted-foreground underline hover:text-foreground">Forgot Password?</a>
              </FieldLabel>
              <Input id="input-password" type="password" placeholder="Enter Password" />
            </Field>
            <Button>Login</Button>
          </div>
          <Label className="text-muted-foreground pt-5 justify-center flex flex-1">
            <div>
              Don't have an account? <a href="#" className="underline hover:text-foreground">Sign up</a>
            </div>
          </Label>
        </CardContent>
      </Card>
    </div>
  );
}