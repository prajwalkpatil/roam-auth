import { Alert, AlertDescription, AlertTitle } from "./ui/alert"
import { AlertCircleIcon } from "lucide-react"

interface AlertDialogProps {
  title: string
  description?: string
  variant?: "default" | "destructive"
}

export default function AlertWrapper({
  variant = "destructive",
  title,
  description,
}: AlertDialogProps) {
  return (
    <Alert
      variant={variant}
      className="absolute top-0 right-0 mx-4 my-5 w-80 max-w-[80%]"
    >
      <AlertCircleIcon />
      <AlertTitle>{title}</AlertTitle>
      {description && <AlertDescription>{description}</AlertDescription>}
    </Alert>
  )
}
