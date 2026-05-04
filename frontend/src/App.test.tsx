import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { act } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";

const booking = {
  booking_id: "booking-1",
  patient_id: "patient-demo",
  doctor_id: "doctor-cardiologia",
  slot_id: "slot-cardio-0900",
  status: "PENDING_PAYMENT",
  reserved_until: "2026-05-04T13:15:00Z",
};

function jsonResponse(body: unknown, status = 200) {
  return Promise.resolve(
    new Response(JSON.stringify(body), {
      status,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

describe("App", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        const method = init?.method ?? "GET";

        if (url.endsWith("/bookings") && method === "POST") {
          return jsonResponse({
            booking_id: booking.booking_id,
            status: booking.status,
            reserved_until: booking.reserved_until,
          }, 201);
        }

        if (url.includes("/bookings?") && method === "GET") {
          return jsonResponse({ bookings: [booking] });
        }

        if (url.endsWith(`/bookings/${booking.booking_id}`) && method === "GET") {
          return jsonResponse({
            booking,
            events: [{ event_id: "event-1", event_type: "CREATED" }],
          });
        }

        if (url.endsWith(`/bookings/${booking.booking_id}/confirm`) && method === "POST") {
          return jsonResponse({
            booking_id: booking.booking_id,
            status: "CONFIRMED",
            updated_at: "2026-05-04T13:04:00Z",
          });
        }

        if (url.endsWith(`/bookings/${booking.booking_id}/cancel`) && method === "PATCH") {
          return jsonResponse({
            booking_id: booking.booking_id,
            status: "CANCELLED",
            updated_at: "2026-05-04T13:08:00Z",
          });
        }

        return jsonResponse({ error: `unexpected request: ${method} ${url}` }, 500);
      }),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("allows a patient to select a slot and manage a booking through the gateway", async () => {
    const user = userEvent.setup();

    render(<App />);

    await act(async () => {
      await user.click(screen.getByRole("button", { name: /seleccionar bloque cardiología/i }));
    });
    expect(screen.getByLabelText(/slot seleccionado/i)).toHaveValue("slot-cardio-0900");

    await act(async () => {
      await user.click(screen.getByRole("button", { name: /crear reserva/i }));
    });
    expect(await screen.findByText(/reserva creada/i)).toBeInTheDocument();

    await act(async () => {
      await user.click(screen.getByRole("button", { name: /cargar reservas/i }));
    });
    expect(await screen.findByText("booking-1")).toBeInTheDocument();

    await act(async () => {
      await user.click(screen.getByRole("button", { name: /ver detalle booking-1/i }));
    });
    expect(await screen.findByText("CREATED")).toBeInTheDocument();

    await act(async () => {
      await user.click(screen.getByRole("button", { name: /confirmar booking-1/i }));
    });
    expect(await screen.findByText(/reserva confirmada/i)).toBeInTheDocument();

    await act(async () => {
      await user.click(screen.getByRole("button", { name: /cancelar booking-1/i }));
    });
    expect(await screen.findByText(/reserva cancelada/i)).toBeInTheDocument();

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/bookings",
        expect.objectContaining({ method: "POST" }),
      );
    });
  });
});
