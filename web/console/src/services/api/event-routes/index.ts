import { get, post, put, deleteRequest } from '../client'
import { API_ENDPOINTS } from '../config'
import { unwrap, assertSuccess } from '../_lib/unwrap'
import type { ApiResponse } from '../types'
import type { EventRoute, CreateEventRouteRequest, UpdateEventRouteRequest } from './types'

const BASE = API_ENDPOINTS.EVENT_ROUTE

export async function fetchEventRoutes(): Promise<EventRoute[]> {
  const r = await get<ApiResponse<EventRoute[]>>(BASE)
  return unwrap(r, 'fetch event routes')
}

export async function createEventRoute(data: CreateEventRouteRequest): Promise<EventRoute> {
  const r = await post<ApiResponse<EventRoute>>(BASE, data)
  return unwrap(r, 'create event route')
}

export async function updateEventRoute(uuid: string, data: UpdateEventRouteRequest): Promise<EventRoute> {
  const r = await put<ApiResponse<EventRoute>>(`${BASE}/${uuid}`, data)
  return unwrap(r, 'update event route')
}

export async function deleteEventRoute(uuid: string): Promise<void> {
  const r = await deleteRequest<ApiResponse<void>>(`${BASE}/${uuid}`)
  assertSuccess(r, 'delete event route')
}

export async function fetchEventRouteById(routeId: string): Promise<EventRoute> {
  const r = await get<ApiResponse<EventRoute>>(`${BASE}/${routeId}`)
  return unwrap(r, 'fetch event route')
}
