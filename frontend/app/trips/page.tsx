"use client";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useResource } from "@/lib/api";
import { dateLabel, tripStatus } from "@/lib/dates";
import type { Trip } from "@/lib/types";
import { ErrorMessage, Modal } from "@/components/ui";
import { TripForm } from "@/components/trip-form";
export default function Trips() {
  const [offset, setOffset] = useState(0);
  const { data, error, loading, reload } = useResource<Trip[]>(
    `/api/v1/trips?offset=${offset}`,
  );
  const [create, setCreate] = useState(false);
  const router = useRouter();
  return (
    <main className="container">
      <div className="page-heading">
        <div>
          <p className="eyebrow">A world of possibilities</p>
          <h1>My trips</h1>
          <p className="muted">From the first idea to your last day away.</p>
        </div>
        <button onClick={() => setCreate(true)}>＋ New trip</button>
      </div>
      <ErrorMessage message={error} retry={reload} />
      {loading ? (
        <p role="status">Loading your trips…</p>
      ) : data?.length === 0 ? (
        <section className="empty panel">
          <span className="empty-icon" aria-hidden="true">
            ↗
          </span>
          <h2>Where will you go next?</h2>
          <p>Create a trip, collect your favorite places, and make a plan.</p>
          <button onClick={() => setCreate(true)}>Plan your first trip</button>
        </section>
      ) : (
        <div className="trip-grid">
          {data?.map((trip, i) => (
            <Link
              key={trip.id}
              href={`/trips/${trip.id}`}
              className="trip-card"
            >
              <div className={`trip-art art-${i % 3}`} aria-hidden="true">
                <span>W / {String(offset + i + 1).padStart(2, "0")}</span>
                <div className="sun" />
                <div className="terrain terrain-back" />
                <div className="terrain" />
                <span className="art-caption">{trip.destination}</span>
              </div>
              <div className="trip-card-body">
                <div className="row">
                  <span className="eyebrow">{trip.destination}</span>
                  <span className="badge">
                    {tripStatus(trip.start_date, trip.end_date, trip.timezone)}
                  </span>
                </div>
                <h2>{trip.name}</h2>
                <p className="muted">
                  {dateLabel(trip.start_date)} – {dateLabel(trip.end_date)}
                </p>
                <span className="text-link">
                  Open trip <span aria-hidden="true">↗</span>
                </span>
              </div>
            </Link>
          ))}
        </div>
      )}
      {(offset > 0 || data?.length === 50) && (
        <div className="actions">
          <button
            className="secondary"
            disabled={offset === 0}
            onClick={() => setOffset((n) => Math.max(0, n - 50))}
          >
            Previous
          </button>
          <button
            className="secondary"
            disabled={data?.length !== 50}
            onClick={() => setOffset((n) => n + 50)}
          >
            Next
          </button>
        </div>
      )}
      {create && (
        <Modal title="A new adventure" onClose={() => setCreate(false)}>
          <TripForm
            onCancel={() => setCreate(false)}
            onSaved={(trip) => router.push(`/trips/${trip.id}`)}
          />
        </Modal>
      )}
    </main>
  );
}
