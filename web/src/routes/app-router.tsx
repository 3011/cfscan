import { lazy, Suspense } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from '@/components/layout/app-shell'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { AdminRoute, ProtectedRoute, PublicOnlyRoute } from '@/features/auth/auth-routes'

const DashboardPage = lazy(() => import('@/features/dashboard/dashboard-page').then((module) => ({ default: module.DashboardPage })))
const SourcesPage = lazy(() => import('@/features/sources/sources-page').then((module) => ({ default: module.SourcesPage })))
const ScansPage = lazy(() => import('@/features/scans/scans-page').then((module) => ({ default: module.ScansPage })))
const ResultsPage = lazy(() => import('@/features/results/results-page').then((module) => ({ default: module.ResultsPage })))
const LeaguePage = lazy(() => import('@/features/league/league-page').then((module) => ({ default: module.LeaguePage })))
const IPTrendPage = lazy(() => import('@/features/league/ip-trend-page').then((module) => ({ default: module.IPTrendPage })))
const BlacklistPage = lazy(() => import('@/features/blacklist/blacklist-page').then((module) => ({ default: module.BlacklistPage })))
const AgentsPage = lazy(() => import('@/features/agents/agents-page').then((module) => ({ default: module.AgentsPage })))
const EnrollmentApprovalPage = lazy(() => import('@/features/agents/enrollment-approval-page').then((module) => ({ default: module.EnrollmentApprovalPage })))
const SettingsPage = lazy(() => import('@/features/settings/settings-page').then((module) => ({ default: module.SettingsPage })))
const LoginPage = lazy(() => import('@/features/auth/login-page').then((module) => ({ default: module.LoginPage })))
const UsersPage = lazy(() => import('@/features/users/users-page').then((module) => ({ default: module.UsersPage })))

export function AppRouter() {
  return (
    <Suspense fallback={<div className="p-4 md:p-8"><PageSkeleton /></div>}>
      <Routes>
        <Route element={<PublicOnlyRoute />}>
          <Route path="login" element={<LoginPage />} />
        </Route>
        <Route element={<ProtectedRoute />}>
          <Route path="agents/pair/:pairingToken" element={<EnrollmentApprovalPage />} />
          <Route path="agents/enrollments/:enrollmentID" element={<EnrollmentApprovalPage />} />
          <Route element={<AppShell />}>
            <Route index element={<DashboardPage />} />
            <Route path="sources" element={<SourcesPage />} />
            <Route path="jobs" element={<ScansPage />} />
            <Route path="results" element={<ResultsPage />} />
            <Route path="league" element={<LeaguePage />} />
            <Route path="league/ip/:agentID/:targetIP" element={<IPTrendPage />} />
            <Route path="blacklist" element={<BlacklistPage />} />
            <Route path="agents" element={<AgentsPage />} />
            <Route path="settings" element={<SettingsPage />} />
            <Route element={<AdminRoute />}>
              <Route path="users" element={<UsersPage />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Route>
      </Routes>
    </Suspense>
  )
}
