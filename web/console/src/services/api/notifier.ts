import { get, put } from "./client"
import { unwrap } from "./_lib/unwrap"
import type { ApiResponse } from "./types"

// ── Types ────────────────────────────────────────────────────────────────────

export interface EmailConfig {
  email_config_id: string
  provider: string
  host: string
  port: number
  username: string
  from_address: string
  from_name: string
  reply_to: string
  encryption: string
  logo_url: string
  test_mode: boolean
  status: string
  created_at: string
  updated_at: string
}

export interface EmailConfigUpdate {
  provider: string
  host?: string
  port?: number
  username?: string
  password?: string
  from_address: string
  from_name?: string
  reply_to?: string
  encryption?: string
  logo_url?: string
  test_mode?: boolean
}

export async function fetchEmailConfig(): Promise<EmailConfig> {
  const r = await get<ApiResponse<EmailConfig>>("/email-config")
  return unwrap(r, "fetch email config")
}

export async function updateEmailConfig(data: EmailConfigUpdate): Promise<EmailConfig> {
  const r = await put<ApiResponse<EmailConfig>>("/email-config", data)
  return unwrap(r, "update email config")
}

// ── Delivery status (is the channel configured enough to send?) ──────────────

export interface ConfigStatus {
  configured: boolean
  provider?: string
  status?: string
}

export async function fetchEmailConfigStatus(): Promise<ConfigStatus> {
  const r = await get<ApiResponse<ConfigStatus>>("/email-config/status")
  return unwrap(r, "fetch email config status")
}

// ── SMS Config ───────────────────────────────────────────────────────────────

export interface SMSConfig {
  sms_config_id: string
  provider: string
  account_sid: string
  from_number: string
  sender_id: string
  daily_send_limit: number
  test_mode: boolean
  status: string
  created_at: string
  updated_at: string
}

export interface SMSConfigUpdate {
  provider: string
  account_sid?: string
  auth_token?: string
  from_number?: string
  sender_id?: string
  daily_send_limit?: number
  test_mode?: boolean
}

export async function fetchSMSConfig(): Promise<SMSConfig> {
  const r = await get<ApiResponse<SMSConfig>>("/sms-config")
  return unwrap(r, "fetch SMS config")
}

export async function updateSMSConfig(data: SMSConfigUpdate): Promise<SMSConfig> {
  const r = await put<ApiResponse<SMSConfig>>("/sms-config", data)
  return unwrap(r, "update SMS config")
}
