import Login from "@/pages/Login"
import Signup from "@/pages/Signup"

import { BrowserRouter, Routes, Route } from "react-router-dom"
import Profile from "./pages/Profile"

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/signup" element={<Signup />} />
        <Route path="/" element={<Profile />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
