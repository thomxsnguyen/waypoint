"use client";
import Link from "next/link";
import { useTrip } from "@/components/trip-context";
import { useResource } from "@/lib/api";
import { dateLabel } from "@/lib/dates";
import type { Day, SavedPlace } from "@/lib/types";
import { ErrorMessage } from "@/components/ui";
export default function Dashboard() {
  const { trip } = useTrip();
  const days = useResource<Day[]>(
    `/api/v1/trips/${trip.id}/days`,
    `${trip.start_date}:${trip.end_date}`,
  );
  const places = useResource<SavedPlace[]>(
    `/api/v1/trips/${trip.id}/places?details=false`,
  );
  return (
    <>
      <section className="overview-hero panel">
        <div>
          <p className="eyebrow">The plan is taking shape</p>
          <h2>Your {trip.destination} chapter.</h2>
          <p>
            {trip.description ||
              "Good trips start with a few places you can’t wait to see."}
          </p>
          <Link className="button" href={`/trips/${trip.id}/places`}>
            Find your next stop ↗
          </Link>
        </div>
        <div className="overview-compass" aria-hidden="true">
          <span>N</span>✳<span>S</span>
        </div>
      </section>
      <ErrorMessage
        message={days.error || places.error}
        retry={() => {
          days.reload();
          places.reload();
        }}
      />
      <div className="stats">
        <div>
          <strong>{days.data?.length ?? "—"}</strong>
          <span>days to explore</span>
        </div>
        <div>
          <strong>{places.data?.length ?? "—"}</strong>
          <span>saved places</span>
        </div>
        <div>
          <strong>
            {days.data?.reduce((n, d) => n + d.activities.length, 0) ?? "—"}
          </strong>
          <span>planned activities</span>
        </div>
      </div>
      <div className="section-heading">
        <div>
          <p className="eyebrow">One day at a time</p>
          <h2>Your itinerary</h2>
        </div>
        <Link className="text-link" href={`/trips/${trip.id}/itinerary`}>
          Open itinerary →
        </Link>
      </div>
      <div className="day-overview">
        {days.data?.map((day, i) => (
          <Link
            className="panel day-summary"
            key={day.id}
            href={`/trips/${trip.id}/itinerary?day=${day.id}`}
          >
            <span className="day-number">{String(i + 1).padStart(2, "0")}</span>
            <div>
              <h3>
                {dateLabel(day.date, {
                  weekday: "long",
                  month: "short",
                  day: "numeric",
                })}
              </h3>
              <p className="muted">
                {day.activities.length
                  ? `${day.activities.length} activities planned`
                  : "A day full of possibility"}
              </p>
            </div>
            <span aria-hidden="true">↗</span>
          </Link>
        ))}
      </div>
    </>
  );
}
