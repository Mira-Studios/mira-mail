import type { Email, EmailFilters, ComposeEmail, MailboxSummary } from '../types/email';
import type { ApiError } from '../types/auth';
import { getServerConfig } from './auth';

class ApiClientError extends Error {
  constructor(
    message: string,
    public status?: number,
    public code?: string
  ) {
    super(message);
    this.name = 'ApiClientError';
  }
}

function getAuthHeaders(): Record<string, string> {
  const config = getServerConfig();
  if (!config) {
    throw new ApiClientError('Not configured');
  }
  const headers: Record<string, string> = {
    'Authorization': `Bearer ${config.token}`,
    'Content-Type': 'application/json',
  };
  if (config.username) {
    headers['X-Username'] = config.username;
  }
  if (config.userToken) {
    headers['X-User-Token'] = config.userToken;
  }
  return headers;
}

function getBaseUrl(): string {
  const config = getServerConfig();
  if (!config) {
    throw new ApiClientError('Not configured');
  }
  return config.url.replace(/\/$/, '');
}

async function fetchApi<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const baseUrl = getBaseUrl();
  const headers = getAuthHeaders();

  // Debug: log authentication headers for internal emails
  if (endpoint.includes('internal-emails')) {
    console.log('Debug - Endpoint:', endpoint);
    console.log('Debug - Headers:', headers);
    console.log('Debug - Full request:', {
      method: options.method,
      headers: {
        ...headers,
        ...options.headers,
      },
    });
  }

  const response = await fetch(`${baseUrl}${endpoint}`, {
    ...options,
    headers: {
      ...headers,
      ...options.headers,
    },
  });

  if (!response.ok) {
    const error: ApiError = await response.json().catch(() => ({
      message: `HTTP ${response.status}: ${response.statusText}`,
      status: response.status,
    }));
    throw new ApiClientError(error.message, error.status, error.code);
  }

  return response.json();
}

export const api = {
  login: async (username: string, password: string) => {
    const response = await fetch(`${getBaseUrl()}/api/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ username, password }),
    });
    
    if (!response.ok) {
      throw new ApiClientError('Login failed', response.status);
    }
    
    const data = await response.json();
    // Store the token for future requests
    localStorage.setItem('mira-mail-config', JSON.stringify({
      url: getBaseUrl(),
      token: data.token,
      userToken: data.user_token,
      username: data.username,
    }));
    
    return data;
  },

  // Temporary fix: Update stored token to match current server API key
  updateToken: (newToken: string) => {
    const config = getServerConfig();
    if (config) {
      localStorage.setItem('mira-mail-config', JSON.stringify({
        ...config,
        token: newToken,
      }));
    }
  },

  getMailbox: async (filters?: EmailFilters) => {
    const headers = getAuthHeaders();
    const params = new URLSearchParams();
    if (filters?.mailbox) params.append('mailbox', filters.mailbox);
    if (filters?.search) params.append('search', filters.search);
    if (filters?.label) params.append('label', filters.label);
    if (filters?.unreadOnly) params.append('unread', 'true');
    
    const response = await fetch(`${getBaseUrl()}/api/emails?${params}`, {
      headers,
    });
    
    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      if (error.error === 'no account configured') {
        // Only return empty array for non-sent mailboxes
        if (filters?.mailbox !== 'sent') {
          return [];
        }
        // For sent mailbox, let the error bubble up since server now handles internal emails
      }
      throw new ApiClientError(error.message || 'Failed to fetch emails', response.status);
    }
    
    return response.json();
  },

  getEmail: (id: string): Promise<Email> =>
    fetchApi<Email>(`/api/emails/${id}`),

  getMailboxSummary: async (): Promise<MailboxSummary> => {
    try {
      return await fetchApi<MailboxSummary>('/api/mailbox/summary');
    } catch (error) {
      if (error instanceof ApiClientError && error.message?.includes('no account configured')) {
        return {
          inbox: 0,
          starred: 0,
          sent: 0,
          drafts: 0,
          trash: 0,
          unread: 0,
        };
      }
      throw error;
    }
  },

  // Actions
  markAsRead: (id: string): Promise<void> =>
    fetchApi<void>(`/api/emails/${id}/read`, { method: 'POST' }),

  markAsUnread: (id: string): Promise<void> =>
    fetchApi<void>(`/api/emails/${id}/unread`, { method: 'POST' }),

  toggleStar: (id: string): Promise<boolean> =>
    fetchApi<boolean>(`/api/emails/${id}/star`, { method: 'POST' }),

  moveToTrash: (id: string): Promise<void> =>
    fetchApi<void>(`/api/emails/${id}/trash`, { method: 'POST' }),

  restoreFromTrash: (id: string): Promise<void> =>
    fetchApi<void>(`/api/emails/${id}/restore`, { method: 'POST' }),

  deletePermanently: (id: string): Promise<void> =>
    fetchApi<void>(`/api/emails/${id}`, { method: 'DELETE' }),

  // Compose
  sendEmail: (email: ComposeEmail): Promise<Email> =>
    fetchApi<Email>('/api/emails', {
      method: 'POST',
      body: JSON.stringify(email),
    }),

  saveDraft: (email: Partial<ComposeEmail>): Promise<Email> =>
    fetchApi<Email>('/api/drafts', {
      method: 'POST',
      body: JSON.stringify(email),
    }),

  // Account management
  getAccounts: (): Promise<any[]> =>
    fetchApi<any[]>('/api/account'),

  createUser: (user: {
    username: string;
    name: string;
    email?: string;
    password: string;
  }): Promise<{ success: boolean; user?: any; error?: string }> => {
    return fetchApi('/api/user', {
      method: 'POST',
      body: JSON.stringify(user),
    });
  },

  getCurrentUser: (): Promise<{ success: boolean; user?: any; error?: string }> => {
    return fetchApi('/api/current-user');
  },

  // Internal emails
  getInternalEmails: (): Promise<{ success: boolean; emails?: any[]; error?: string }> => {
    return fetchApi('/api/internal-emails');
  },

  sendInternalEmail: (email: {
    from: string;
    to: string[];
    subject: string;
    body: string;
  }): Promise<{ success: boolean; email?: any; error?: string }> => {
    return fetchApi('/api/internal-emails', {
      method: 'POST',
      body: JSON.stringify(email),
    });
  },

  // Health check - returns full response including has_accounts
  testConnection: (url: string, token: string): Promise<{ setup: boolean; has_accounts: boolean }> => {
    return fetch(`${url.replace(/\/$/, '')}/api/health`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    }).then(async res => {
      if (!res.ok) throw new Error('Connection failed');
      return res.json();
    });
  },

  // Account management
  addAccount: (account: {
    name: string;
    email: string;
    password: string;
    imap_server: string;
    imap_port: number;
    smtp_server: string;
    smtp_port: number;
    use_tls: boolean;
  }): Promise<{ success: boolean; account?: any; error?: string }> => {
    return fetchApi('/api/account', {
      method: 'POST',
      body: JSON.stringify(account),
    });
  },

  // Health check without auth (for testing)
  checkHealth: (url: string): Promise<{ status: string; setup: boolean; has_accounts: boolean; api_key: string }> => {
    return fetch(`${url.replace(/\/$/, '')}/api/health`).then(res => {
      if (!res.ok) throw new Error('Server not responding');
      return res.json();
    });
  },
};
