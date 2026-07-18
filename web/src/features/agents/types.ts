export type AgentStatus = 'online' | 'offline'
export interface Agent {
  id: string
  name: string
  region: string
  continent: string
  concurrency: number
  status: AgentStatus
  os: string
  architecture: string
  version: string
  last_seen_at: string
  created_at: string
}

export type AgentEnrollmentMode = 'device' | 'preauthorized'
export type AgentEnrollmentStatus = 'pending' | 'approved' | 'claimed' | 'rejected' | 'revoked' | 'expired'

export interface AgentEnrollment {
  id: string
  mode: AgentEnrollmentMode
  status: AgentEnrollmentStatus
  requested_name: string
  os: string
  architecture: string
  version: string
  requested_concurrency: number
  name?: string
  region?: string
  continent?: string
  concurrency?: number
  agent_id?: string
  expires_at: string
  approved_at?: string
  claimed_at?: string
  created_at: string
  updated_at: string
}

export interface AgentEnrollmentConfig {
  public_url: string
  agent_image: string
  agent_version: string
  ttl_seconds: number
  poll_interval: number
}

export interface ApproveAgentEnrollmentInput {
  name: string
  region: string
  continent: string
  concurrency: number
}

export interface CreatePreauthorizedEnrollmentInput extends ApproveAgentEnrollmentInput {
  ttl_minutes: number
}

export interface CreatePreauthorizedEnrollmentResponse {
  enrollment: AgentEnrollment
  pairing_token: string
  expires_in: number
}

export type EnrollmentLocator =
  | { kind: 'token'; value: string }
  | { kind: 'id'; value: string }
