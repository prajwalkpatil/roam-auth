import { ping } from "@/api/requests"
import { useState } from "react"

export default function Profile() {
  const [message, setMessage] = useState("")
  ping()
    .then((response) => {
      setMessage(response)
    })
    .catch((err) => console.error(err))
  return (
    <div className="flex h-screen items-center justify-center">{message}</div>
  )
}
