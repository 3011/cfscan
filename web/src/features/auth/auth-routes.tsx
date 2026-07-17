import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { useAuth } from '@/features/auth/auth-context'

function AuthLoading() {
  return <div className="grid min-h-svh place-items-center bg-muted/30"><Loader2 className="size-6 animate-spin text-muted-foreground" aria-label="正在验证登录状态" /></div>
}

export function ProtectedRoute() {
  const auth = useAuth()
  const location = useLocation()
  if (auth.isLoading) return <AuthLoading />
  if (!auth.user) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  return <Outlet />
}

export function PublicOnlyRoute() {
  const auth = useAuth()
  if (auth.isLoading) return <AuthLoading />
  if (auth.user) return <Navigate to="/" replace />
  return <Outlet />
}

export function AdminRoute() {
  const auth = useAuth()
  if (!auth.canManage) return <Navigate to="/" replace />
  return <Outlet />
}
