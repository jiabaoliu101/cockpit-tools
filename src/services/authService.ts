import { useAuthStore, type AuthUser, type SubscriptionInfo } from '../stores/useAuthStore';

// Backend API base URL — change this to your production server address
// Format: https://your-domain.com/api/v1
const API_BASE = (typeof window !== 'undefined' && (window as any).__AISWITCH_API_BASE__)
  || import.meta.env.VITE_AISWITCH_API_BASE
  || 'http://localhost:8080/api/v1';

// For Tauri apps, we go through Rust backend for security
// But for now, direct HTTP calls work for development

async function apiCall(
  endpoint: string,
  options: RequestInit = {},
  authRequired = true,
): Promise<any> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };

  if (authRequired) {
    const token = useAuthStore.getState().accessToken;
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
  }

  const res = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  if (res.status === 401 && authRequired) {
    // Try refresh
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      // Retry with new token
      headers['Authorization'] = `Bearer ${useAuthStore.getState().accessToken}`;
      const retryRes = await fetch(`${API_BASE}${endpoint}`, { ...options, headers });
      if (!retryRes.ok) {
        const err = await retryRes.json().catch(() => ({ error: 'Request failed' }));
        throw new Error(err.error || 'Request failed');
      }
      return retryRes.json();
    }
    // Refresh failed — logout
    useAuthStore.getState().logout();
    throw new Error('Session expired');
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Request failed' }));
    throw new Error(err.error || 'Request failed');
  }

  return res.json();
}

async function refreshAccessToken(): Promise<boolean> {
  const refreshToken = useAuthStore.getState().refreshToken;
  if (!refreshToken) return false;

  try {
    const res = await fetch(`${API_BASE}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!res.ok) return false;

    const data = await res.json();
    const { user } = useAuthStore.getState();
    useAuthStore.getState().setAuth(
      user!,
      data.access_token,
      data.refresh_token,
    );
    return true;
  } catch {
    return false;
  }
}

export async function login(email: string, password: string): Promise<void> {
  const data = await apiCall('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  }, false);

  const user: AuthUser = {
    id: data.user.id,
    email: data.user.email,
    nickname: data.user.nickname,
    is_admin: data.user.is_admin,
  };

  useAuthStore.getState().setAuth(user, data.access_token, data.refresh_token);
}

export async function register(
  email: string,
  password: string,
  nickname: string,
): Promise<void> {
  const data = await apiCall('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ email, password, nickname }),
  }, false);

  const user: AuthUser = {
    id: data.user.id,
    email: data.user.email,
    nickname: data.user.nickname,
    is_admin: data.user.is_admin,
  };

  useAuthStore.getState().setAuth(user, data.access_token, data.refresh_token);
}

export async function logout(): Promise<void> {
  try {
    await apiCall('/auth/logout', { method: 'POST' });
  } catch {
    // Ignore errors on logout
  }
  useAuthStore.getState().logout();
}

export async function fetchCheckin(): Promise<any> {
  return apiCall('/client/checkin', { method: 'POST' });
}

export async function fetchMyAccounts(): Promise<any[]> {
  return apiCall('/my-accounts');
}

export async function fetchCurrentSubscription(): Promise<SubscriptionInfo> {
  const data = await apiCall('/subscriptions/current');
  useAuthStore.getState().setSubscription(data);
  return data;
}

export async function fetchSubscriptionPlans(): Promise<any[]> {
  return apiCall('/subscriptions/plans');
}

export async function createOrder(planId: string): Promise<any> {
  return apiCall('/subscriptions/order', {
    method: 'POST',
    body: JSON.stringify({ plan_id: planId }),
  });
}

export async function checkOrderStatus(orderId: string): Promise<any> {
  return apiCall(`/subscriptions/status/${orderId}`);
}
