import type { AccessTokenProvider } from "./bookingApi";

export type Notification = {
  id: number;
  event_id: string;
  booking_id: string;
  event_type: string;
  recipient_id: string;
  channel: string;
  message: string;
  status: string;
  payload?: Record<string, unknown>;
  created_at?: string;
  read_at?: string;
};

export type NotificationDevStatus = {
  service: string;
  store: string;
  kafka_brokers: string[];
  booking_topic: string;
  dlq_topic: string;
  consumer_group: string;
  total_notifications: number;
  recipient_id: string;
  recipient_total: number;
  recipient_unread: number;
  latest_notifications: Notification[];
};

const defaultBaseUrl = import.meta.env.VITE_API_BASE_URL || "/api";

export function createNotificationApi(
  baseUrl = defaultBaseUrl,
  accessToken?: AccessTokenProvider,
) {
  const normalizedBaseUrl = normalizeBaseUrl(baseUrl);

  async function request<T>(path: string, init: RequestInit): Promise<T> {
    const headers = requestHeaders(init, accessToken);
    const response = await fetch(`${normalizedBaseUrl}${path}`, {
      ...init,
      ...(headers ? { headers } : {}),
    });

    if (!response.ok) {
      throw new Error(await errorMessage(response));
    }

    return (await response.json()) as T;
  }

  return {
    listNotifications(limit = 10) {
      const params = new URLSearchParams({ limit: String(limit) });
      return request<{ notifications: Notification[] }>(`/notifications?${params.toString()}`, {
        method: "GET",
      });
    },

    getUnreadCount() {
      return request<{ unread_count: number }>("/notifications/unread-count", {
        method: "GET",
      });
    },

    markRead(notificationId: number) {
      return request<{ id: number; read_at: string }>(
        `/notifications/${encodeURIComponent(String(notificationId))}/read`,
        {
          method: "PATCH",
        },
      );
    },

    getDevStatus(limit = 5) {
      const params = new URLSearchParams({ limit: String(limit) });
      return request<NotificationDevStatus>(
        `/notifications/dev/status?${params.toString()}`,
        {
          method: "GET",
        },
      );
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

function requestHeaders(init: RequestInit, accessToken?: AccessTokenProvider) {
  const headers = objectHeaders(init.headers);
  const token = resolveAccessToken(accessToken);

  if (init.body && !Object.keys(headers).some((name) => name.toLowerCase() === "content-type")) {
    headers["Content-Type"] = "application/json";
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  return Object.keys(headers).length > 0 ? headers : undefined;
}

function resolveAccessToken(accessToken?: AccessTokenProvider) {
  if (!accessToken) {
    return "";
  }
  return typeof accessToken === "function" ? accessToken() ?? "" : accessToken;
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
