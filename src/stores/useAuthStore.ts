import { create } from 'zustand';

export interface AuthUser {
  id: string;
  email: string;
  nickname: string;
  is_admin: boolean;
}

export interface SubscriptionInfo {
  has_subscription: boolean;
  subscription?: {
    id: string;
    plan_name: string;
    max_accounts: number;
    status: string;
    expires_at: string;
    assigned_count: number;
  };
}

interface AuthState {
  user: AuthUser | null;
  accessToken: string | null;
  refreshToken: string | null;
  isAuthenticated: boolean;
  isAuthLoading: boolean;
  subscription: SubscriptionInfo | null;

  setAuth: (user: AuthUser, accessToken: string, refreshToken: string) => void;
  logout: () => void;
  setSubscription: (sub: SubscriptionInfo) => void;
  setAuthLoading: (loading: boolean) => void;
}

const TOKEN_KEY = 'aiswitch.auth.access_token';
const REFRESH_KEY = 'aiswitch.auth.refresh_token';
const USER_KEY = 'aiswitch.auth.user';

function loadFromStorage() {
  try {
    const token = localStorage.getItem(TOKEN_KEY);
    const refresh = localStorage.getItem(REFRESH_KEY);
    const userStr = localStorage.getItem(USER_KEY);
    const user = userStr ? JSON.parse(userStr) : null;
    return { token, refresh, user };
  } catch {
    return { token: null, refresh: null, user: null };
  }
}

const stored = loadFromStorage();

export const useAuthStore = create<AuthState>((set) => ({
  user: stored.user,
  accessToken: stored.token,
  refreshToken: stored.refresh,
  isAuthenticated: !!(stored.token && stored.user),
  isAuthLoading: true,
  subscription: null,

  setAuth: (user, accessToken, refreshToken) => {
    localStorage.setItem(TOKEN_KEY, accessToken);
    localStorage.setItem(REFRESH_KEY, refreshToken);
    localStorage.setItem(USER_KEY, JSON.stringify(user));
    set({
      user,
      accessToken,
      refreshToken,
      isAuthenticated: true,
      isAuthLoading: false,
    });
  },

  logout: () => {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(REFRESH_KEY);
    localStorage.removeItem(USER_KEY);
    set({
      user: null,
      accessToken: null,
      refreshToken: null,
      isAuthenticated: false,
      isAuthLoading: false,
      subscription: null,
    });
  },

  setSubscription: (subscription) => set({ subscription }),

  setAuthLoading: (loading) => set({ isAuthLoading: loading }),
}));
