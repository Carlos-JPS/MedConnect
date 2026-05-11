import {
  Activity,
  CalendarCheck2,
  CalendarDays,
  CalendarPlus,
  CheckCircle2,
  ClipboardList,
  Clock3,
  FileSearch,
  Loader2,
  MapPin,
  RefreshCw,
  ShieldCheck,
  Stethoscope,
  XCircle,
} from "lucide-react";
import { FormEvent, ReactNode, useMemo, useState } from "react";
import {
  Booking,
  BookingActionResult,
  BookingDetail,
  BookingStatus,
  createBookingApi,
} from "./services/api/bookingApi";
import "./styles.css";

type DemoSlot = {
  id: string;
  specialty: string;
  doctorId: string;
  doctorName: string;
  startsAt: string;
  duration: string;
  room: string;
  triage: string;
};

const demoPatientId = "46bd4a6f-6a4d-4e81-ae7c-c9d7ac05b235";

const demoSlots: DemoSlot[] = [
  {
    id: "0f5c2b6a-1a87-4b7e-ae2c-37ef2f9f1c21",
    specialty: "Cardiología",
    doctorId: "7e0d2ab1-164e-4a28-8b95-f24293dd0e91",
    doctorName: "Dra. Valentina Rojas",
    startsAt: "2026-05-04T09:00:00-04:00",
    duration: "30 min",
    room: "Box 204",
    triage: "prioridad normal",
  },
  {
    id: "8d3d26e8-4d55-4d0d-99e8-0ed2133b8c31",
    specialty: "Traumatología",
    doctorId: "d2f50707-24ab-4df8-8a6b-cc12a8c47a91",
    doctorName: "Dr. Matías Fuentes",
    startsAt: "2026-05-04T10:30:00-04:00",
    duration: "45 min",
    room: "Box 118",
    triage: "control postoperatorio",
  },
  {
    id: "5cb4ad32-545f-4810-bd7f-0a979e4f25e5",
    specialty: "Medicina interna",
    doctorId: "4f1cb247-7810-4ae4-9267-a8df9c7a0835",
    doctorName: "Dra. Camila Soto",
    startsAt: "2026-05-04T15:45:00-04:00",
    duration: "30 min",
    room: "Teleconsulta",
    triage: "seguimiento remoto",
  },
];

const statusOptions: Array<{ value: BookingStatus | ""; label: string }> = [
  { value: "", label: "Todos" },
  { value: "PENDING_PAYMENT", label: "Pendiente de pago" },
  { value: "CONFIRMED", label: "Confirmadas" },
  { value: "CANCELLED", label: "Canceladas" },
  { value: "EXPIRED", label: "Expiradas" },
];

const api = createBookingApi();

function App() {
  const [patientId, setPatientId] = useState(demoPatientId);
  const [statusFilter, setStatusFilter] = useState<BookingStatus | "">("");
  const [doctorId, setDoctorId] = useState(demoSlots[0].doctorId);
  const [slotId, setSlotId] = useState("");
  const [notes, setNotes] = useState("Paciente solicita confirmar hora desde portal web.");
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [detail, setDetail] = useState<BookingDetail | null>(null);
  const [lookupBookingId, setLookupBookingId] = useState("");
  const [paymentId, setPaymentId] = useState("");
  const [cancelReason, setCancelReason] = useState("Paciente solicita reagendar.");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState("");

  const selectedSlot = useMemo(
    () => demoSlots.find((slot) => slot.id === slotId),
    [slotId],
  );

  function selectSlot(slot: DemoSlot) {
    setSlotId(slot.id);
    setDoctorId(slot.doctorId);
    setMessage(`Bloque seleccionado: ${slot.specialty} con ${slot.doctorName}`);
    setError("");
  }

  async function createBooking(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await run("create", async () => {
      const result = await api.createBooking({
        patient_id: patientId,
        doctor_id: doctorId,
        slot_id: slotId,
        notes,
      });
      setLookupBookingId(result.booking_id);
      setMessage(`Reserva creada: ${result.booking_id} quedó ${statusLabel(result.status)}.`);
    });
  }

  async function loadBookings() {
    await run("list", async () => {
      const result = await api.listBookings(patientId, statusFilter);
      setBookings(result.bookings);
      setMessage(`Reservas cargadas para ${patientId}.`);
    });
  }

  async function loadDetail(bookingId = lookupBookingId) {
    if (!bookingId) {
      setError("Ingresa un booking_id para consultar el detalle.");
      return;
    }

    await run("detail", async () => {
      const result = await api.getBooking(bookingId);
      setDetail(result);
      setLookupBookingId(result.booking.booking_id);
      setMessage(`Detalle cargado para ${result.booking.booking_id}.`);
    });
  }

  async function confirmBooking(bookingId: string) {
    if (!paymentId.trim()) {
      setError("Ingresa un payment_id asociado a esta reserva antes de confirmar.");
      return;
    }

    await run(`confirm-${bookingId}`, async () => {
      const result = await api.confirmBooking(bookingId, paymentId.trim());
      applyActionResult(result);
      setMessage(`Reserva confirmada: ${bookingId}.`);
    });
  }

  async function cancelBooking(bookingId: string) {
    await run(`cancel-${bookingId}`, async () => {
      const result = await api.cancelBooking(bookingId, cancelReason);
      applyActionResult(result);
      setMessage(`Reserva cancelada: ${bookingId}.`);
    });
  }

  async function run(operation: string, action: () => Promise<void>) {
    setLoading(operation);
    setError("");
    try {
      await action();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Error desconocido";
      setError(message);
    } finally {
      setLoading("");
    }
  }

  function applyActionResult(result: BookingActionResult) {
    setBookings((current) =>
      current.map((booking) =>
        booking.booking_id === result.booking_id
          ? { ...booking, status: result.status, updated_at: result.updated_at }
          : booking,
      ),
    );
    setDetail((current) =>
      current?.booking.booking_id === result.booking_id
        ? {
            ...current,
            booking: {
              ...current.booking,
              status: result.status,
              updated_at: result.updated_at,
            },
          }
        : current,
    );
  }

  const pendingOperation = loading !== "";

  return (
    <main className="app-shell">
      <section className="command-bar" aria-labelledby="app-title">
        <div>
          <p className="eyebrow">Portal paciente · reservas</p>
          <h1 id="app-title">MedConnect Booking Console</h1>
          <p className="intro">
            Flujo mínimo de agenda clínica vía API Gateway: seleccionar bloque, crear reserva,
            consultar estado y ejecutar confirmación o cancelación.
          </p>
        </div>
        <div className="gateway-badge" aria-label="Conexión por API Gateway">
          <ShieldCheck size={22} />
          <span>HTTP REST · Gateway</span>
        </div>
      </section>

      {(message || error) && (
        <section className={`feedback ${error ? "feedback-error" : "feedback-ok"}`}>
          {error ? <XCircle size={18} /> : <CheckCircle2 size={18} />}
          <span>{error || message}</span>
        </section>
      )}

      <section className="workspace-grid">
        <section className="panel schedule-panel" aria-labelledby="slots-title">
          <PanelHeader
            eyebrow="Disponibilidad demo"
            id="slots-title"
            icon={<CalendarDays size={22} />}
            title="Bloques de agenda"
            subtitle="Estos slots simulan la selección de disponibilidad hasta que availability-service exponga su endpoint."
          />
          <div className="slot-list">
            {demoSlots.map((slot) => (
              <article className="slot-card" key={slot.id}>
                <div>
                  <p className="slot-time">{formatTime(slot.startsAt)}</p>
                  <h3>{slot.specialty}</h3>
                  <p>{slot.doctorName}</p>
                </div>
                <div className="slot-meta">
                  <span>
                    <Clock3 size={15} />
                    {slot.duration}
                  </span>
                  <span>
                    <MapPin size={15} />
                    {slot.room}
                  </span>
                  <span>
                    <Activity size={15} />
                    {slot.triage}
                  </span>
                </div>
                <button
                  className="secondary-action"
                  type="button"
                  onClick={() => selectSlot(slot)}
                  aria-label={`Seleccionar bloque ${slot.specialty} ${formatTime(slot.startsAt)}`}
                >
                  Seleccionar bloque
                </button>
              </article>
            ))}
          </div>
        </section>

        <section className="panel create-panel" aria-labelledby="create-title">
          <PanelHeader
            eyebrow="Nueva reserva"
            id="create-title"
            icon={<CalendarPlus size={22} />}
            title="Crear reserva"
            subtitle="El navegador llama solo a POST /bookings del gateway."
          />
          <form className="form-stack" onSubmit={createBooking}>
            <label>
              Paciente
              <input
                value={patientId}
                onChange={(event) => setPatientId(event.target.value)}
                placeholder="patient-id"
                required
              />
            </label>
            <label>
              Doctor seleccionado
              <input
                value={doctorId}
                onChange={(event) => setDoctorId(event.target.value)}
                placeholder="doctor-id"
                required
              />
            </label>
            <label htmlFor="slot-id">
              Slot seleccionado
              <input
                id="slot-id"
                value={slotId}
                onChange={(event) => setSlotId(event.target.value)}
                placeholder="slot-id"
                required
              />
            </label>
            {selectedSlot && (
              <div className="selected-slot">
                <Stethoscope size={18} />
                <span>
                  {selectedSlot.specialty}, {selectedSlot.doctorName}, {formatDate(selectedSlot.startsAt)}
                </span>
              </div>
            )}
            <label>
              Notas clínicas
              <textarea value={notes} onChange={(event) => setNotes(event.target.value)} />
            </label>
            <button className="primary-action" disabled={pendingOperation} type="submit">
              {loading === "create" ? <Loader2 className="spin" size={18} /> : <CalendarCheck2 size={18} />}
              Crear reserva
            </button>
          </form>
        </section>

        <section className="panel bookings-panel" aria-labelledby="bookings-title">
          <PanelHeader
            eyebrow="Paciente"
            id="bookings-title"
            icon={<ClipboardList size={22} />}
            title="Reservas del paciente"
            subtitle="Listado y acciones sobre GET /bookings, POST /confirm y PATCH /cancel."
          />
          <div className="filters">
            <label>
              Paciente
              <input
                value={patientId}
                onChange={(event) => setPatientId(event.target.value)}
                placeholder="patient-id"
              />
            </label>
            <label>
              Estado
              <select
                value={statusFilter}
                onChange={(event) => setStatusFilter(event.target.value as BookingStatus | "")}
              >
                {statusOptions.map((option) => (
                  <option key={option.value || "all"} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>
            <button
              className="secondary-action"
              type="button"
              onClick={loadBookings}
              disabled={pendingOperation}
            >
              {loading === "list" ? <Loader2 className="spin" size={18} /> : <RefreshCw size={18} />}
              Cargar reservas
            </button>
          </div>

          <div className="action-inputs">
            <label>
              Pago para confirmar
              <input value={paymentId} onChange={(event) => setPaymentId(event.target.value)} />
            </label>
            <label>
              Motivo de cancelación
              <input value={cancelReason} onChange={(event) => setCancelReason(event.target.value)} />
            </label>
          </div>

          <div className="booking-list" aria-live="polite">
            {bookings.length === 0 ? (
              <div className="empty-state">
                <FileSearch size={28} />
                <p>Carga las reservas para ver estado, detalle y acciones disponibles.</p>
              </div>
            ) : (
              bookings.map((booking) => (
                <BookingCard
                  booking={booking}
                  key={booking.booking_id}
                  loading={loading}
                  onCancel={cancelBooking}
                  onConfirm={confirmBooking}
                  onDetail={loadDetail}
                />
              ))
            )}
          </div>
        </section>

        <section className="panel detail-panel" aria-labelledby="detail-title">
          <PanelHeader
            eyebrow="Auditoría"
            id="detail-title"
            icon={<FileSearch size={22} />}
            title="Detalle de reserva"
            subtitle="Consulta directa de GET /bookings/{id}, incluyendo eventos persistidos."
          />
          <div className="detail-search">
            <label>
              Booking ID
              <input
                value={lookupBookingId}
                onChange={(event) => setLookupBookingId(event.target.value)}
                placeholder="booking-id"
              />
            </label>
            <button
              className="secondary-action"
              type="button"
              onClick={() => loadDetail()}
              disabled={pendingOperation}
            >
              {loading === "detail" ? <Loader2 className="spin" size={18} /> : <FileSearch size={18} />}
              Buscar detalle
            </button>
          </div>

          {detail ? (
            <article className="detail-card">
              <div className="detail-heading">
                <div>
                  <span className="muted-label">Booking</span>
                  <h3>{detail.booking.booking_id}</h3>
                </div>
                <StatusBadge status={detail.booking.status} />
              </div>
              <dl className="detail-grid">
                <div>
                  <dt>Paciente</dt>
                  <dd>{detail.booking.patient_id || patientId}</dd>
                </div>
                <div>
                  <dt>Doctor</dt>
                  <dd>{detail.booking.doctor_id || "Sin dato"}</dd>
                </div>
                <div>
                  <dt>Slot</dt>
                  <dd>{detail.booking.slot_id || "Sin dato"}</dd>
                </div>
                <div>
                  <dt>Reserva hasta</dt>
                  <dd>{formatDate(detail.booking.reserved_until)}</dd>
                </div>
              </dl>
              <div className="event-timeline">
                <h4>Eventos</h4>
                {detail.events.length === 0 ? (
                  <p className="muted-text">Sin eventos registrados.</p>
                ) : (
                  detail.events.map((event) => (
                    <div className="timeline-row" key={event.event_id}>
                      <span>{event.event_type}</span>
                      <small>{formatDate(event.created_at)}</small>
                    </div>
                  ))
                )}
              </div>
            </article>
          ) : (
            <div className="empty-state compact">
              <FileSearch size={28} />
              <p>Selecciona una reserva o ingresa un ID para revisar el detalle.</p>
            </div>
          )}
        </section>
      </section>
    </main>
  );
}

function PanelHeader({
  eyebrow,
  id,
  icon,
  subtitle,
  title,
}: {
  eyebrow: string;
  id: string;
  icon: ReactNode;
  subtitle: string;
  title: string;
}) {
  return (
    <div className="panel-header">
      <div className="panel-icon">{icon}</div>
      <div>
        <p className="eyebrow">{eyebrow}</p>
        <h2 id={id}>{title}</h2>
        <p>{subtitle}</p>
      </div>
    </div>
  );
}

function BookingCard({
  booking,
  loading,
  onCancel,
  onConfirm,
  onDetail,
}: {
  booking: Booking;
  loading: string;
  onCancel: (bookingId: string) => void;
  onConfirm: (bookingId: string) => void;
  onDetail: (bookingId: string) => void;
}) {
  const isConfirming = loading === `confirm-${booking.booking_id}`;
  const isCancelling = loading === `cancel-${booking.booking_id}`;
  const canConfirm = booking.status === "PENDING_PAYMENT";
  const canCancel = booking.status === "PENDING_PAYMENT";

  return (
    <article className="booking-card">
      <div className="booking-main">
        <div>
          <span className="muted-label">Booking ID</span>
          <strong>{booking.booking_id}</strong>
        </div>
        <StatusBadge status={booking.status} />
      </div>
      <div className="booking-facts">
        <span>{booking.doctor_id || "doctor sin dato"}</span>
        <span>{booking.slot_id || "slot sin dato"}</span>
        <span>{formatDate(booking.reserved_until || booking.updated_at || booking.created_at)}</span>
      </div>
      <div className="booking-actions">
        <button
          className="ghost-action"
          type="button"
          onClick={() => onDetail(booking.booking_id)}
          aria-label={`Ver detalle ${booking.booking_id}`}
        >
          Ver detalle
        </button>
        <button
          className="confirm-action"
          type="button"
          onClick={() => onConfirm(booking.booking_id)}
          aria-label={`Confirmar ${booking.booking_id}`}
          disabled={!canConfirm || isConfirming || isCancelling}
        >
          {isConfirming ? <Loader2 className="spin" size={16} /> : <CheckCircle2 size={16} />}
          Confirmar
        </button>
        <button
          className="cancel-action"
          type="button"
          onClick={() => onCancel(booking.booking_id)}
          aria-label={`Cancelar ${booking.booking_id}`}
          disabled={!canCancel || isConfirming || isCancelling}
        >
          {isCancelling ? <Loader2 className="spin" size={16} /> : <XCircle size={16} />}
          Cancelar
        </button>
      </div>
    </article>
  );
}

function StatusBadge({ status }: { status: BookingStatus }) {
  return <span className={`status-badge status-${status.toLowerCase()}`}>{statusLabel(status)}</span>;
}

function statusLabel(status: BookingStatus) {
  const labels: Record<BookingStatus, string> = {
    UNSPECIFIED: "Sin estado",
    PENDING_PAYMENT: "Pendiente de pago",
    CONFIRMED: "Confirmada",
    CANCELLED: "Cancelada",
    EXPIRED: "Expirada",
  };
  return labels[status] ?? status;
}

function formatDate(value?: string) {
  if (!value) {
    return "Sin fecha";
  }

  return new Intl.DateTimeFormat("es-CL", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function formatTime(value: string) {
  return new Intl.DateTimeFormat("es-CL", {
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

export default App;
