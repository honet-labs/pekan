function resolveApiBaseUrl(): string {
  const envUrl = import.meta.env.VITE_API_BASE_URL;
  if (typeof window !== "undefined") {
    // In browser production environment, always prefer relative path /api/v1
    // to avoid hardcoded domain / CORS / 502 tunnel issues
    if (!envUrl || envUrl.includes("localhost") || envUrl.startsWith("http")) {
      return "/api/v1";
    }
  }
  return envUrl ?? "/api/v1";
}

export const API_BASE_URL = resolveApiBaseUrl();
let refreshInFlight: Promise<boolean> | null = null;

type ApiFetchOptions = RequestInit & {
  skipAuth?: boolean;
  skipAuthRetry?: boolean;
};

// Kept for backward compatibility in codebase, but no longer manages tokens manually
export function setTokens(access: string | null, refresh: string | null): void {
  // Tokens are now managed via HttpOnly cookies by the browser.
  // This function is kept for backward compatibility with components 
  // that might still call it.
  if (access === null && refresh === null) {
    window.dispatchEvent(new CustomEvent("pekan:auth:clear"));
  }
}


export async function apiFetch<T>(path: string, init?: ApiFetchOptions): Promise<T> {
  const response = await doFetch(path, init);

  if (
    (response.status === 401 || response.status === 403) &&
    !init?.skipAuthRetry &&
    path !== "/auth/login" &&
    path !== "/auth/refresh" &&
    !path.startsWith("/admin")
  ) {
    const refreshedAccessToken = await refreshAccessToken();
    if (refreshedAccessToken) {
      const retryResponse = await doFetch(path, {
        ...init,
        skipAuthRetry: true
      });
      return parseResponse<T>(retryResponse);
    }
  }
  return parseResponse<T>(response);
}

async function doFetch(path: string, init?: ApiFetchOptions): Promise<Response> {
  const headers: Record<string, string> = {
    ...(init?.headers as Record<string, string> | undefined)
  };
  const isFormData = typeof FormData !== "undefined" && init?.body instanceof FormData;
  if (!isFormData && !headers["Content-Type"]) {
    headers["Content-Type"] = "application/json";
  }
  return fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers,
    credentials: "include"
  });
}


function resolveApiLocale(): "en" | "id" {
  if (typeof window === "undefined") {
    return "en";
  }
  const stored = window.localStorage.getItem("pekan_locale");
  return stored === "id" ? "id" : "en";
}

function getFallbackMessageByStatus(status: number): string {
  const locale = resolveApiLocale();
  const messages = locale === "id"
    ? {
        400: "Permintaan ke API tidak valid.",
        401: "Sesi berakhir. Silakan login kembali.",
        403: "Akses ditolak.",
        404: "Endpoint API tidak ditemukan.",
        405: "Metode API tidak diizinkan.",
        413: "Ukuran data terlalu besar.",
        429: "Terlalu banyak permintaan.",
        500: "Terjadi kesalahan pada server.",
        502: "Bad gateway.",
        503: "Layanan tidak tersedia.",
        504: "Waktu tunggu gateway habis."
      }
    : {
        400: "Bad request to API.",
        401: "Unauthorized. Please login again.",
        403: "Forbidden. You do not have access.",
        404: "API endpoint not found.",
        405: "API method not allowed.",
        413: "Payload too large.",
        429: "Too many requests.",
        500: "Internal server error.",
        502: "Bad gateway.",
        503: "Service unavailable.",
        504: "Gateway timeout."
      };
  return messages[status as keyof typeof messages] ?? (locale === "id" ? `Permintaan API gagal (HTTP ${status})` : `API request failed (HTTP ${status})`);
}

type ParsedPayload = {
  data?: unknown;
  error?: { message?: string };
  rawText?: string;
};

async function parseResponse<T>(response: Response): Promise<T> {
  const payload = await parsePayload(response);
  if (!response.ok) {
    const fallback = getFallbackMessageByStatus(response.status);
    const raw = payload.rawText ? compactText(payload.rawText) : "";
    const message = payload.error?.message ?? (raw || fallback);
    console.error(`[API Error] Request to ${response.url} failed with status ${response.status}:`, message);
    throw new Error(message);
  }
  if (typeof payload.data === "undefined") {
    return {} as T;
  }
  return payload.data as T;
}

async function parsePayload(response: Response): Promise<ParsedPayload> {
  const contentType = (response.headers.get("content-type") ?? "").toLowerCase();
  if (contentType.includes("application/json")) {
    const json = await response.json().catch(() => ({}));
    const hasEnvelope = json && typeof json === "object" && Object.prototype.hasOwnProperty.call(json, "data");
    return {
      // Support both wrapped payloads ({data,meta}) and legacy direct payloads.
      data: hasEnvelope ? json?.data : json,
      error: json?.error
    };
  }
  const rawText = await response.text().catch(() => "");
  if (rawText) {
    const trimmed = rawText.trim();
    if (trimmed.startsWith("{") || trimmed.startsWith("[")) {
      const json = safeParseJSON(trimmed);
      if (json) {
        const hasEnvelope = Object.prototype.hasOwnProperty.call(json, "data");
        return {
          data: hasEnvelope ? json.data : json,
          error: json.error,
          rawText
        };
      }
    }
  }
  return { rawText };
}

function compactText(raw: string): string {
  const withoutTags = raw.replace(/<[^>]+>/g, " ").replace(/\s+/g, " ").trim();
  if (!withoutTags) {
    return "";
  }
  if (withoutTags.length <= 180) {
    return withoutTags;
  }
  return `${withoutTags.slice(0, 177)}...`;
}

function safeParseJSON(raw: string): { data?: unknown; error?: { message?: string } } | null {
  try {
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed === "object") {
      return parsed as { data?: unknown; error?: { message?: string } };
    }
  } catch {
    return null;
  }
  return null;
}

async function refreshAccessToken(): Promise<boolean> {
  if (refreshInFlight) {
    return refreshInFlight;
  }

  refreshInFlight = (async () => {
    const response = await doFetch("/auth/refresh", {
      method: "POST",
      body: JSON.stringify({}),
      skipAuth: true,
      skipAuthRetry: true
    });
    if (!response.ok) {
      setTokens(null, null);
      return false;
    }
    // Browser automatically sets the new cookies from Set-Cookie headers
    return true;
  })();

  try {
    return await refreshInFlight;
  } finally {
    refreshInFlight = null;
  }
}

