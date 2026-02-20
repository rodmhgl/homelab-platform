import { config } from '../utils/config';
import type { HealthResponse } from './types';

export const healthApi = {
  check: async (): Promise<HealthResponse> => {
    const response = await fetch(`${config.apiUrl}/health`);
    if (!response.ok) {
      throw new Error('Health check failed');
    }
    return response.json();
  },
};
