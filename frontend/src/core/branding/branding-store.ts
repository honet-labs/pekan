import { apiFetch } from "../api/client";
import { useSyncExternalStore } from "react";

export type BrandingData = {
  app_name: string;
  page_title: string;
  logo: string;
  favicon: string;
  public_url: string;
};

type BrandingState = BrandingData & {
  loaded: boolean;
  setBranding: (data: Partial<BrandingData>) => void;
  fetchBranding: () => Promise<void>;
};

type Listener = () => void;
const listeners = new Set<Listener>();

function emitChange() {
  for (const listener of listeners) {
    listener();
  }
}

const brandingActions = {
  setBranding(data: Partial<BrandingData>) {
    state = {
      ...state,
      ...data,
      loaded: true,
    };
    
    // Apply page title dynamically
    if (state.page_title) {
      document.title = state.page_title;
    }
    
    // Apply favicon dynamically
    if (state.favicon) {
      let link = document.querySelector("link[rel~='icon']") as HTMLLinkElement;
      if (!link) {
        link = document.createElement('link');
        link.rel = 'icon';
        document.getElementsByTagName('head')[0].appendChild(link);
      }
      link.href = state.favicon;
      if (state.favicon.startsWith("data:")) {
        const match = state.favicon.match(/data:([^;]+);/);
        if (match && match[1]) {
          link.type = match[1];
        }
      }
    }
    
    emitChange();
  },
  async fetchBranding() {
    try {
      const data = await apiFetch<BrandingData>("/branding", { skipAuth: true } as any);
      brandingActions.setBranding(data);
    } catch (err) {
      console.error("Failed to fetch public branding settings:", err);
      // Fallback
      state = {
        ...state,
        loaded: true,
      };
      emitChange();
    }
  }
};

let state: BrandingState = {
  app_name: "PEKAN",
  page_title: "PENCATATAN KEUANGAN",
  logo: "",
  favicon: "",
  public_url: "",
  loaded: false,
  ...brandingActions
};

export const brandingStore = {
  getState() {
    return state;
  },
  subscribe(listener: Listener) {
    listeners.add(listener);
    return () => listeners.delete(listener);
  }
};

export function useBrandingStore() {
  return useSyncExternalStore(brandingStore.subscribe, brandingStore.getState);
}
