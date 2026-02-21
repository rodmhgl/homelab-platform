export const config = {
  // Empty string = same-origin requests (proxied by nginx to Platform API)
  apiUrl: import.meta.env.VITE_API_URL || '',
  apiVersion: 'v1',
};
