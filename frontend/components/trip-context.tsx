"use client";
import { createContext, useContext } from "react";
import { useParams, usePathname, useRouter } from "next/navigation";
import Link from "next/link";
import { useState, type ReactNode } from "react";
import { json, useAPI, useResource } from "@/lib/api";
import { dateLabel } from "@/lib/dates";
import type { Trip } from "@/lib/types";
import { ErrorMessage, Modal } from "./ui";
import { TripForm } from "./trip-form";
const Context = createContext<{ trip: Trip; reload: () => void } | null>(null);
export function useTrip() {
  const value = useContext(Context);
  if (!value) throw new Error("Trip required");
  return value;
}
export function TripLayout({ children }: { children: ReactNode }) {
  const { tripId } = useParams<{ tripId: string }>();
  const {
    data: trip,
    error,
    loading,
    reload,
  } = useResource<Trip>(`/api/v1/trips/${tripId}`);
  const [edit, setEdit] = useState(false);
  const [remove, setRemove] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState("");
  const api = useAPI();
  const router = useRouter();
  const pathname = usePathname();
  async function deleteTrip() {
    setDeleting(true);
    try {
      await api(`/api/v1/trips/${tripId}`, json("DELETE"));
      router.push("/trips");
    } catch (e) {
      setDeleteError((e as Error).message);
      setDeleting(false);
    }
  }
  return (
    <main className="container">
      <Link className="back-link" href="/trips">
        ← All trips
      </Link>
      <ErrorMessage message={error} retry={reload} />
      {loading ? (
        <p role="status">Loading trip…</p>
      ) : (
        trip && (
          <Context.Provider value={{ trip, reload }}>
            <div className="page-heading trip-heading">
              <div>
                <p className="eyebrow">{trip.destination}</p>
                <h1>{trip.name}</h1>
                <p className="muted">
                  {dateLabel(trip.start_date)} – {dateLabel(trip.end_date)}{" "}
                  <span className="separator">/</span>{" "}
                  {trip.timezone.replaceAll("_", " ")}
                </p>
              </div>
              <div className="actions">
                <button
                  className="secondary small"
                  onClick={() => setEdit(true)}
                >
                  Edit trip
                </button>
                <button
                  className="quiet danger small"
                  onClick={() => setRemove(true)}
                >
                  Delete trip
                </button>
              </div>
            </div>
            <nav className="tabs" aria-label="Trip sections">
              {[
                ["", "Overview"],
                ["/places", "Discover & save"],
                ["/itinerary", "Itinerary"],
              ].map(([suffix, label]) => (
                <Link
                  key={suffix}
                  href={`/trips/${tripId}${suffix}`}
                  aria-current={
                    pathname === `/trips/${tripId}${suffix}`
                      ? "page"
                      : undefined
                  }
                >
                  {label}
                </Link>
              ))}
            </nav>
            {children}
            {edit && (
              <Modal title="Edit trip" onClose={() => setEdit(false)}>
                <TripForm
                  trip={trip}
                  onCancel={() => setEdit(false)}
                  onSaved={() => {
                    setEdit(false);
                    reload();
                  }}
                />
              </Modal>
            )}
            {remove && (
              <Modal title="Delete this trip?" onClose={() => setRemove(false)}>
                <p>
                  This permanently removes “{trip.name}”, its saved places, and
                  its itinerary.
                </p>
                <ErrorMessage message={deleteError} />
                <div className="actions end">
                  <button
                    className="secondary"
                    onClick={() => setRemove(false)}
                    disabled={deleting}
                  >
                    Keep trip
                  </button>
                  <button
                    className="danger-fill"
                    onClick={deleteTrip}
                    disabled={deleting}
                  >
                    {deleting ? "Deleting…" : "Delete permanently"}
                  </button>
                </div>
              </Modal>
            )}
          </Context.Provider>
        )
      )}
    </main>
  );
}
