/**
 * Generic CRUD resource API factory.
 *
 * Removes the per-resource boilerplate (build query → call verb → unwrap → throw).
 * A new resource's service becomes a few lines instead of ~90.
 *
 *   const api = createResourceApi<Row, Create, Update>('/things', 'thing')
 *
 * The `TList` type parameter lets a resource declare its list shape:
 *   - paginated resources:  createResourceApi<Row, ..., PaginatedResponse<Row>>(...)
 *   - flat-array resources: createResourceApi<Row, ..., Row[]>(...)  (the default)
 *
 * Resources with extra/bespoke endpoints can spread this and add their own calls.
 */

import { get, post, put, deleteRequest } from '../client'
import type { ApiResponse } from '../types'
import { unwrap, assertSuccess } from './unwrap'
import { buildQuery } from './query'

export interface ResourceApi<TRow, TCreate, TUpdate, TList> {
  list: (params?: Record<string, unknown>) => Promise<TList>
  getById: (id: string) => Promise<TRow>
  create: (data: TCreate) => Promise<TRow>
  update: (id: string, data: TUpdate) => Promise<TRow>
  remove: (id: string) => Promise<void>
}

export function createResourceApi<TRow, TCreate, TUpdate, TList = TRow[]>(
  base: string,
  label: string,
): ResourceApi<TRow, TCreate, TUpdate, TList> {
  return {
    list: (params) =>
      get<ApiResponse<TList>>(`${base}${buildQuery(params)}`).then((r) => unwrap(r, `fetch ${label}s`)),
    getById: (id) =>
      get<ApiResponse<TRow>>(`${base}/${id}`).then((r) => unwrap(r, `fetch ${label}`)),
    create: (data) =>
      post<ApiResponse<TRow>>(base, data).then((r) => unwrap(r, `create ${label}`)),
    update: (id, data) =>
      put<ApiResponse<TRow>>(`${base}/${id}`, data).then((r) => unwrap(r, `update ${label}`)),
    remove: (id) =>
      deleteRequest<ApiResponse<void>>(`${base}/${id}`).then((r) => assertSuccess(r, `delete ${label}`)),
  }
}
