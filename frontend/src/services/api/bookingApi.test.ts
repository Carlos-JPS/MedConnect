import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createBookingApi } from "./bookingApi";

function jsonResponse(body: unknown, status = 200) {
  return Promise.resolve(
    new Response(JSON.stringify(body), {
      status,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

describe("bookingApi", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("creates a booking through the API Gateway", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(
      await jsonResponse({
        booking_id: "booking-1",
        status: "PENDING_PAYMENT",
        reserved_until: "2026-05-04T14:15:00Z",
      }, 201),
    );

    const api = createBookingApi("http://gateway.test");
    const result = await api.createBooking({
      patient_id: "patient-1",
      doctor_id: "doctor-1",
      slot_id: "slot-1",
      notes: "Control inicial",
    });

    expect(result.booking_id).toBe("booking-1");
    expect(fetchMock).toHaveBeenCalledWith(
      "http://gateway.test/bookings",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          patient_id: "patient-1",
          doctor_id: "doctor-1",
          slot_id: "slot-1",
          notes: "Control inicial",
        }),
      }),
    );
  });

  it("lists, retrieves, confirms and cancels patient bookings", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        await jsonResponse({
          bookings: [
            {
              booking_id: "booking-1",
              patient_id: "patient-1",
              doctor_id: "doctor-1",
              slot_id: "slot-1",
              status: "PENDING_PAYMENT",
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        await jsonResponse({
          booking: { booking_id: "booking-1", status: "PENDING_PAYMENT" },
          events: [{ event_id: "event-1", event_type: "CREATED" }],
        }),
      )
      .mockResolvedValueOnce(
        await jsonResponse({
          booking_id: "booking-1",
          status: "CONFIRMED",
          updated_at: "2026-05-04T14:00:00Z",
        }),
      )
      .mockResolvedValueOnce(
        await jsonResponse({
          booking_id: "booking-1",
          status: "CANCELLED",
          updated_at: "2026-05-04T14:05:00Z",
        }),
      );

    const api = createBookingApi("http://gateway.test/");
    await api.listBookings("patient-1", "PENDING_PAYMENT");
    await api.getBooking("booking-1");
    await api.confirmBooking("booking-1", "payment-1");
    await api.cancelBooking("booking-1", "Cambio de agenda");

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "http://gateway.test/bookings?patient_id=patient-1&status=PENDING_PAYMENT",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "http://gateway.test/bookings/booking-1",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      3,
      "http://gateway.test/bookings/booking-1/confirm",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ payment_id: "payment-1" }),
      }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      4,
      "http://gateway.test/bookings/booking-1/cancel",
      expect.objectContaining({
        method: "PATCH",
        body: JSON.stringify({ reason: "Cambio de agenda" }),
      }),
    );
  });
});
