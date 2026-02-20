import { apiClient } from './client';
import type {
  ListAppsResponse,
  Application,
  SyncRequest,
} from './types';

export const appsApi = {
  list: () => apiClient.get<ListAppsResponse>('/apps'),

  get: (name: string) => apiClient.get<Application>(`/apps/${name}`),

  sync: (name: string, request: SyncRequest) =>
    apiClient.post<{ message: string }>(`/apps/${name}/sync`, request),
};
