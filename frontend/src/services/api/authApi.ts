export type RegisterPayload = {
  email: string;
  password: string;
  full_name: string;
  role: string;
};

export type RegisterResult = {
  user_id: string;
  role: string;
  is_active: boolean;
  created_at?: string;
};

export type LoginResult = {
  access_token: string;
  user_id: string;
  role: string;
  expires_at?: string;
};

const defaultBaseUrl = import.meta.env.VITE_API_BASE_URL || "/api";

export function createAuthApi(baseUrl = defaultBaseUrl) {
  const normalizedBaseUrl = normalizeBaseUrl(baseUrl);

  async function request<T>(path: string, init: RequestInit): Promise<T> {
    const response = await fetch(`${normalizedBaseUrl}${path}`, {
      ...init,
      headers: {
        "Content-Type": "application/json",
        ...objectHeaders(init.headers),
      },
    });

    if (!response.ok) {
      throw new Error(await errorMessage(response));
    }

    return (await response.json()) as T;
  }

  return {
    register(payload: RegisterPayload) {
      return request<RegisterResult>("/auth/register", {
        method: "POST",
        body: JSON.stringify(payload),
      });
    },

    login(email: string, password: string) {
      return request<LoginResult>("/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
    },
  };
}

function normalizeBaseUrl(baseUrl: string) {
  const trimmedBaseUrl = baseUrl.trim();
  if (trimmedBaseUrl === "/") {
    return "";
  }

  return trimmedBaseUrl.replace(/\/+$/, "");
}

function objectHeaders(headers: RequestInit["headers"]) {
  if (!headers) {
    return {};
  }

  return Object.fromEntries(new Headers(headers).entries());
}

async function errorMessage(response: Response) {
  try {
    const payload = (await response.json()) as { error?: string };
    return payload.error ?? `HTTP ${response.status}`;
  } catch {
    return `HTTP ${response.status}`;
  }
}
