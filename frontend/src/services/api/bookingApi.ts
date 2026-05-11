export type BookingStatus =
  | "UNSPECIFIED"
  | "PENDING_PAYMENT"
  | "CONFIRMED"
  | "CANCELLED"
  | "EXPIRED";

export type Booking = {
  booking_id: string;
  patient_id?: string;
  doctor_id?: string;
  slot_id?: string;
  status: BookingStatus;
  payment_id?: string;
  confirmation_code?: string;
  notes?: string;
  created_at?: string;
  updated_at?: string;
  reserved_until?: string;
  confirmed_at?: string;
  cancelled_at?: string;
};

export type BookingEvent = {
  event_id: string;
  booking_id?: string;
  event_type: string;
  payload?: Record<string, unknown>;
  created_at?: string;
};

export type CreateBookingPayload = {
  patient_id: string;
  doctor_id: string;
  slot_id: string;
  notes?: string;
};

export type CreateBookingResult = {
  booking_id: string;
  status: BookingStatus;
  reserved_until?: string;
};

export type BookingActionResult = {
  booking_id: string;
  status: BookingStatus;
  updated_at?: string;
};

export type BookingDetail = {
  booking: Booking;
  events: BookingEvent[];
};

export type BookingApi = ReturnType<typeof createBookingApi>;

const defaultBaseUrl = import.meta.env.VITE_API_BASE_URL || "/api";

export function createBookingApi(baseUrl = defaultBaseUrl) {
  const normalizedBaseUrl = normalizeBaseUrl(baseUrl);

  async function request<T>(path: string, init: RequestInit): Promise<T> {
    const headers = requestHeaders(init);
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
    createBooking(payload: CreateBookingPayload) {
      return request<CreateBookingResult>("/bookings", {
        method: "POST",
        body: JSON.stringify(payload),
      });
    },

    listBookings(patientId: string, status?: BookingStatus | "") {
      const params = new URLSearchParams({ patient_id: patientId });
      if (status) {
        params.set("status", status);
      }

      return request<{ bookings: Booking[] }>(`/bookings?${params.toString()}`, {
        method: "GET",
      });
    },

    getBooking(bookingId: string) {
      return request<BookingDetail>(`/bookings/${encodeURIComponent(bookingId)}`, {
        method: "GET",
      });
    },

    confirmBooking(bookingId: string, paymentId: string) {
      return request<BookingActionResult>(
        `/bookings/${encodeURIComponent(bookingId)}/confirm`,
        {
          method: "POST",
          body: JSON.stringify({ payment_id: paymentId }),
        },
      );
    },

    cancelBooking(bookingId: string, reason: string) {
      return request<BookingActionResult>(
        `/bookings/${encodeURIComponent(bookingId)}/cancel`,
        {
          method: "PATCH",
          body: JSON.stringify({ reason }),
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

function requestHeaders(init: RequestInit) {
  const headers = objectHeaders(init.headers);
  const hasContentType = Object.keys(headers).some(
    (headerName) => headerName.toLowerCase() === "content-type",
  );

  if (init.body && !hasContentType) {
    headers["Content-Type"] = "application/json";
  }

  return Object.keys(headers).length > 0 ? headers : undefined;
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
