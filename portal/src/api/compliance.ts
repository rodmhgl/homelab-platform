import { apiClient } from './client';
import type {
  SummaryResponse,
  ListViolationsResponse,
  ListVulnerabilitiesResponse,
  ListSecurityEventsResponse,
} from './types';

export const complianceApi = {
  summary: () => apiClient.get<SummaryResponse>('/compliance/summary'),

  violations: (params?: { namespace?: string; constraint?: string }) =>
    apiClient.get<ListViolationsResponse>('/compliance/violations', params),

  vulnerabilities: (params?: { severity?: string; namespace?: string }) =>
    apiClient.get<ListVulnerabilitiesResponse>('/compliance/vulnerabilities', params),

  events: (params?: {
    namespace?: string;
    severity?: string;
    rule?: string;
    since?: string;
    limit?: string;
  }) => apiClient.get<ListSecurityEventsResponse>('/compliance/events', params),
};
