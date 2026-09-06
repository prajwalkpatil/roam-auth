import ProtectedRoute from "@/auth/ProtectedRoute"
import PublicRoute from "@/auth/PublicRoute"
import Login from "@/pages/Login"
import Profile from "@/pages/Profile"
import Signup from "@/pages/Signup"

import { createBrowserRouter } from "react-router-dom"

export const router = createBrowserRouter([
  {
    path: "/",
    element: (
      <ProtectedRoute>
        <Profile />
      </ProtectedRoute>
    ),
  },
  {
    path: "/login",
    element: (
      <PublicRoute>
        <Login />
      </PublicRoute>
    ),
  },
  {
    path: "/signup",
    element: (
      <PublicRoute>
        <Signup />
      </PublicRoute>
    ),
  },
])
