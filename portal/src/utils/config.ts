export const config = {
  // Empty string = same-origin requests (proxied by nginx to Platform API)
  apiUrl: import.meta.env.VITE_API_URL || '',
  apiVersion: 'v1',
  // TODO: Replace with proper token management (ExternalSecret + runtime injection)
  // For now, static token for demo (API only checks presence, not validity)
  apiToken: import.meta.env.VITE_API_TOKEN || 'homelab-portal-token',
};
