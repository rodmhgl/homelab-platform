import { apiClient } from './client';
import type { ScaffoldRequest, ScaffoldResponse } from './types';

export const scaffoldApi = {
  create: (request: ScaffoldRequest) =>
    apiClient.post<ScaffoldResponse>('/scaffold', request),
};
