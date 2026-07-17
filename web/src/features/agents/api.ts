import { request, type ItemsResponse } from '@/lib/http'
import type {
  Agent,
  AgentEnrollment,
  AgentEnrollmentConfig,
  ApproveAgentEnrollmentInput,
  CreatePreauthorizedEnrollmentInput,
  CreatePreauthorizedEnrollmentResponse,
  EnrollmentLocator,
} from '@/features/agents/types'

export function getAgents() {
  return request<ItemsResponse<Agent>>('/api/v1/agents')
}

export function getAgentEnrollments() {
  return request<ItemsResponse<AgentEnrollment>>('/api/v1/agent-enrollments')
}

export function getAgentEnrollmentConfig() {
  return request<AgentEnrollmentConfig>('/api/v1/agent-enrollments/config')
}

function enrollmentPath(locator: EnrollmentLocator) {
  return locator.kind === 'id'
    ? `/api/v1/agent-enrollments/id/${encodeURIComponent(locator.value)}`
    : `/api/v1/agent-enrollments/${encodeURIComponent(locator.value)}`
}

export function getAgentEnrollment(locator: EnrollmentLocator) {
  return request<AgentEnrollment>(enrollmentPath(locator))
}

export function approveAgentEnrollment(locator: EnrollmentLocator, input: ApproveAgentEnrollmentInput) {
  return request<AgentEnrollment>(`${enrollmentPath(locator)}/approve`, { method: 'POST', body: JSON.stringify(input) })
}

export function rejectAgentEnrollment(locator: EnrollmentLocator) {
  return request<AgentEnrollment>(`${enrollmentPath(locator)}/reject`, { method: 'POST', body: '{}' })
}

export function createPreauthorizedEnrollment(input: CreatePreauthorizedEnrollmentInput) {
  return request<CreatePreauthorizedEnrollmentResponse>('/api/v1/agent-enrollments/preauthorized', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}
