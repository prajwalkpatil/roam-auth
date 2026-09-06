import ProtectedRoute from "@/auth/ProtectedRoute"
import PublicRoute from "@/auth/PublicRoute"
import Login from "@/pages/Login"
import Profile from "@/pages/Profile"
import Signup from "@/pages/Signup"

import { createBrowserRouter, type RouteObject } from "react-router-dom"

const publicRoutes: RouteObject[] = [
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
]

const protectedRoutes: RouteObject[] = [
  {
    path: "/",
    element: (
      <ProtectedRoute>
        <Profile />
      </ProtectedRoute>
    ),
  },
]

export const router = createBrowserRouter([...protectedRoutes, ...publicRoutes])

export const publicRoutePaths = new Set(publicRoutes.map((route) => route.path))
