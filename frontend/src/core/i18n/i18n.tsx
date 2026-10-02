import { createContext, ReactNode, useCallback, useContext, useMemo, useState } from "react";
import { defaultLocale, Locale, translations } from "./translations";

type I18nContextValue = {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: string) => string;
};

const I18nContext = createContext<I18nContextValue | undefined>(undefined);

function resolveInitialLocale(): Locale {
  const stored = window.localStorage.getItem("pekan_locale");
  if (stored === "en" || stored === "id") {
    return stored;
  }
  const browserLang = (navigator.language || "").toLowerCase();
  if (browserLang.startsWith("en")) {
    return "en";
  }
  if (browserLang.startsWith("id")) {
    return "id";
  }
  return defaultLocale;
}

export function I18nProvider({ children }: { children: ReactNode }): JSX.Element {
  const [locale, setLocaleState] = useState<Locale>(resolveInitialLocale);

  const setLocale = useCallback((next: Locale) => {
    setLocaleState(next);
    window.localStorage.setItem("pekan_locale", next);
    window.dispatchEvent(new CustomEvent("pekan:locale:changed", { detail: next }));
  }, []);

  useEffect(() => {
    const handleStorage = (e: StorageEvent) => {
      if (e.key === "pekan_locale" && (e.newValue === "en" || e.newValue === "id")) {
        setLocaleState(e.newValue as Locale);
      }
    };
    const handleCustom = (e: Event) => {
      const custom = e as CustomEvent<Locale>;
      if (custom.detail === "en" || custom.detail === "id") {
        setLocaleState(custom.detail);
      }
    };
    window.addEventListener("storage", handleStorage);
    window.addEventListener("pekan:locale:changed", handleCustom);
    return () => {
      window.removeEventListener("storage", handleStorage);
      window.removeEventListener("pekan:locale:changed", handleCustom);
    };
  }, []);

  const t = useCallback(
    (key: string) => {
      return translations[locale][key] ?? translations.en[key] ?? key;
    },
    [locale]
  );

  const value = useMemo<I18nContextValue>(() => ({ locale, setLocale, t }), [locale, setLocale, t]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext);
  if (!ctx) {
    throw new Error("useI18n must be used inside I18nProvider");
  }
  return ctx;
}
