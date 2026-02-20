import { apiClient } from './client';
import type {
  ListClaimsResponse,
  GetResourceResponse,
  CreateInfraRequest,
} from './types';

export const infraApi = {
  list: () => apiClient.get<ListClaimsResponse>('/infra'),

  listStorage: () => apiClient.get<ListClaimsResponse>('/infra/storage'),

  listVaults: () => apiClient.get<ListClaimsResponse>('/infra/vaults'),

  get: (kind: string, name: string, namespace: string) =>
    apiClient.get<GetResourceResponse>(`/infra/${kind}/${name}`, { namespace }),

  create: (request: CreateInfraRequest) =>
    apiClient.post<{ message: string; commitURL: string }>('/infra', request),

  delete: (kind: string, name: string, namespace: string) =>
    apiClient.delete<{ message: string; commitURL: string }>(
      `/infra/${kind}/${name}?namespace=${namespace}`
    ),
};
