package pb

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestBookingContractUsesEnumsAndTimestamps(t *testing.T) {
	if BookingStatus_BOOKING_STATUS_PENDING_PAYMENT.String() == "" {
		t.Fatal("expected booking status enum values to exist")
	}

	fields := (&Booking{}).ProtoReflect().Descriptor().Fields()

	assertTimestampField(t, fields, "created_at")
	assertTimestampField(t, fields, "updated_at")
	assertTimestampField(t, fields, "reserved_until")
	assertTimestampField(t, fields, "confirmed_at")
	assertTimestampField(t, fields, "cancelled_at")

	statusField := fields.ByName("status")
	if statusField == nil {
		t.Fatal("expected booking.status field to exist")
	}
	if statusField.Kind() != protoreflect.EnumKind {
		t.Fatalf("expected booking.status to be enum, got %s", statusField.Kind())
	}
	if string(statusField.Enum().FullName()) != "booking.BookingStatus" {
		t.Fatalf("expected booking.status to use booking.BookingStatus, got %s", statusField.Enum().FullName())
	}
}

func TestBookingEventContractExists(t *testing.T) {
	if BookingEventType_BOOKING_EVENT_TYPE_CREATED.String() == "" {
		t.Fatal("expected booking event type enum values to exist")
	}

	fields := (&BookingEvent{}).ProtoReflect().Descriptor().Fields()

	eventTypeField := fields.ByName("event_type")
	if eventTypeField == nil {
		t.Fatal("expected booking_event.event_type field to exist")
	}
	if eventTypeField.Kind() != protoreflect.EnumKind {
		t.Fatalf("expected booking_event.event_type to be enum, got %s", eventTypeField.Kind())
	}

	assertTimestampField(t, fields, "created_at")
}

func assertTimestampField(t *testing.T, fields protoreflect.FieldDescriptors, name protoreflect.Name) {
	t.Helper()

	field := fields.ByName(name)
	if field == nil {
		t.Fatalf("expected %s field to exist", name)
	}
	if field.Kind() != protoreflect.MessageKind {
		t.Fatalf("expected %s to be a message, got %s", name, field.Kind())
	}
	if string(field.Message().FullName()) != "google.protobuf.Timestamp" {
		t.Fatalf("expected %s to use google.protobuf.Timestamp, got %s", name, field.Message().FullName())
	}
}
