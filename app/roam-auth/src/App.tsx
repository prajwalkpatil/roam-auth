import Login from "@/pages/Login"
import Signup from "@/pages/Signup"

import { BrowserRouter, Routes, Route } from "react-router-dom"
import { ping } from "./api/requests"

export function App() {
  ping()
    .then((r) => console.log("r :>> ", r))
    .catch((e) => console.error(e))
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/signup" element={<Signup />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
