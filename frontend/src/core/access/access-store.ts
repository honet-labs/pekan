export type AccessState = {
  modules: Set<string>;
  features: Set<string>;
  permissions: Set<string>;
  setAccess: (input: { modules: string[]; features: string[]; permissions: string[] }) => void;
  clearAccess: () => void;
};

type Listener = () => void;
const listeners = new Set<Listener>();

function emitChange() {
  for (const listener of listeners) {
    listener();
  }
}

const STORAGE_KEY = "pekan_access_state";

type AccessSnapshot = {
  modules: string[];
  features: string[];
  permissions: string[];
};

function loadSnapshot(): AccessSnapshot {
  const raw = localStorage.getItem(STORAGE_KEY);
  if (!raw) {
    return { modules: [], features: [], permissions: [] };
  }
  try {
    const parsed = JSON.parse(raw) as Partial<AccessSnapshot>;
    return {
      modules: Array.isArray(parsed.modules) ? parsed.modules : [],
      features: Array.isArray(parsed.features) ? parsed.features : [],
      permissions: Array.isArray(parsed.permissions) ? parsed.permissions : []
    };
  } catch {
    return { modules: [], features: [], permissions: [] };
  }
}

function saveSnapshot(snapshot: AccessSnapshot): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(snapshot));
}

const initial = loadSnapshot();

const accessActions = {
  setAccess(input: { modules: string[]; features: string[]; permissions: string[] }) {
    const safeInput = {
      modules: Array.isArray(input?.modules) ? input.modules : [],
      features: Array.isArray(input?.features) ? input.features : [],
      permissions: Array.isArray(input?.permissions) ? input.permissions : []
    };
    const snapshot = {
      modules: Array.from(new Set(safeInput.modules)),
      features: Array.from(new Set(safeInput.features)),
      permissions: Array.from(new Set(safeInput.permissions))
    };
    state = {
      ...state,
      modules: new Set(snapshot.modules),
      features: new Set(snapshot.features),
      permissions: new Set(snapshot.permissions),
    };
    saveSnapshot(snapshot);
    emitChange();
  },
  clearAccess() {
    state = {
      ...state,
      modules: new Set<string>(),
      features: new Set<string>(),
      permissions: new Set<string>(),
    };
    localStorage.removeItem(STORAGE_KEY);
    emitChange();
  }
};

let state: AccessState = {
  modules: new Set<string>(initial.modules),
  features: new Set<string>(initial.features),
  permissions: new Set<string>(initial.permissions),
  ...accessActions
};

export const accessStore = {
  getState() {
    return state;
  },
  subscribe(listener: Listener) {
    listeners.add(listener);
    return () => listeners.delete(listener);
  }
};

import { useSyncExternalStore } from "react";

export function useAccessStore() {
  return useSyncExternalStore(accessStore.subscribe, accessStore.getState);
}
