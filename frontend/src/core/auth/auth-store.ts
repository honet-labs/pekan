import { setTokens as setClientTokens } from "../api/client";

type AuthState = {
  isAuthenticated: boolean;
  accessToken: string | null;
  refreshToken: string | null;
  setAuth: (token: string | null) => void;
  setTokens: (access: string | null, refresh: string | null) => void;
  clear: () => void;
};

type Listener = () => void;
const listeners = new Set<Listener>();

function emitChange() {
  for (const listener of listeners) {
    listener();
  }
}

const storedAccessToken = null;
const storedRefreshToken = null;
// setAccessToken(storedAccessToken); // Removed for HttpOnly migration

const authActions = {
  setAuth(token: string | null) {
    state = {
      ...state,
      accessToken: token,
      isAuthenticated: token !== null,
    };
    setClientTokens(token, state.refreshToken);
    emitChange();
  },
  setTokens(access: string | null, refresh: string | null) {
    state = {
      ...state,
      accessToken: access,
      refreshToken: refresh,
      isAuthenticated: access !== null,
    };
    setClientTokens(access, refresh);
    emitChange();
  },
  clear() {
    state = {
      ...state,
      accessToken: null,
      refreshToken: null,
      isAuthenticated: false,
    };
    emitChange();
  }
};

let state: AuthState = {
  isAuthenticated: storedAccessToken !== null,
  accessToken: storedAccessToken,
  refreshToken: storedRefreshToken,
  ...authActions
};

export const authStore = {
  getState() {
    return state;
  },
  subscribe(listener: Listener) {
    listeners.add(listener);
    return () => listeners.delete(listener);
  }
};

window.addEventListener("pekan:auth:clear", () => {
  authActions.clear();
});

import { useSyncExternalStore } from "react";

export function useAuthStore() {
  return useSyncExternalStore(authStore.subscribe, authStore.getState);
}
