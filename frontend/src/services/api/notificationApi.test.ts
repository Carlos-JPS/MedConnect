import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createNotificationApi } from "./notificationApi";

function jsonResponse(body: unknown, status = 200) {
  return Promise.resolve(
    new Response(JSON.stringify(body), {
      status,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

describe("notificationApi", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("lists notifications and sends bearer token through the gateway", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(
      await jsonResponse({
        notifications: [
          {
            id: 7,
            event_id: "event-1",
            booking_id: "booking-1",
            event_type: "booking.created",
            message: "Reserva creada.",
          },
        ],
      }),
    );

    const api = createNotificationApi("http://gateway.test", "access-token-1");
    const result = await api.listNotifications(5);

    expect(result.notifications[0].id).toBe(7);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://gateway.test/notifications?limit=5",
      {
        method: "GET",
        headers: { Authorization: "Bearer access-token-1" },
      },
    );
  });

  it("reads unread count, marks as read and loads dev status", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(await jsonResponse({ unread_count: 2 }))
      .mockResolvedValueOnce(await jsonResponse({ id: 7, read_at: "2026-05-04T13:05:00Z" }))
      .mockResolvedValueOnce(
        await jsonResponse({
          service: "notification-service",
          store: "postgres",
          kafka_brokers: ["kafka:9092"],
          booking_topic: "medconnect.booking.events.v1",
          dlq_topic: "medconnect.booking.events.dlq.v1",
          consumer_group: "medconnect-notification-service-v1",
          total_notifications: 2,
          recipient_id: "patient-1",
          recipient_total: 2,
          recipient_unread: 1,
          latest_notifications: [],
        }),
      );

    const api = createNotificationApi("http://gateway.test/", () => "access-token-1");
    await api.getUnreadCount();
    await api.markRead(7);
    await api.getDevStatus(4);

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "http://gateway.test/notifications/unread-count",
      { method: "GET", headers: { Authorization: "Bearer access-token-1" } },
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "http://gateway.test/notifications/7/read",
      { method: "PATCH", headers: { Authorization: "Bearer access-token-1" } },
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      3,
      "http://gateway.test/notifications/dev/status?limit=4",
      { method: "GET", headers: { Authorization: "Bearer access-token-1" } },
    );
  });
});
