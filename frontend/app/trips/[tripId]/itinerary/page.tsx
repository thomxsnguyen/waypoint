"use client";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { useTrip } from "@/components/trip-context";
import { ActivityForm } from "@/components/activity-form";
import { DayMap } from "@/components/day-map";
import { ErrorMessage, Modal } from "@/components/ui";
import { GoogleAttribution, PlaceInfo } from "@/components/place-info";
import { json, useAPI, useResource } from "@/lib/api";
import { crossesMidnight, dateLabel } from "@/lib/dates";
import type { Activity, Day, SavedPlace } from "@/lib/types";
export default function Itinerary() {
  const { trip } = useTrip();
  const api = useAPI();
  const days = useResource<Day[]>(
    `/api/v1/trips/${trip.id}/days`,
    `${trip.start_date}:${trip.end_date}`,
  );
  const places = useResource<SavedPlace[]>(`/api/v1/trips/${trip.id}/places`);
  const params = useSearchParams();
  const [dayId, setDayId] = useState(params.get("day") || "");
  const [selected, setSelected] = useState<string | null>(null);
  const [editing, setEditing] = useState<Activity | "new" | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");
  const day = days.data?.find((d) => d.id === dayId) || days.data?.[0];
  const placeById = useMemo(
    () => new Map(places.data?.map((p) => [p.id, p])),
    [places.data],
  );
  const markers = useMemo(
    () =>
      day?.activities.flatMap((a) => {
        const p = placeById.get(a.saved_place_id)?.details;
        return p?.location
          ? [
              {
                activityId: a.id,
                name: p.name,
                position: a.position,
                ...p.location,
              },
            ]
          : [];
      }) || [],
    [day, placeById],
  );
  useEffect(() => {
    if (selected)
      document
        .getElementById(`activity-${selected}`)
        ?.scrollIntoView({ behavior: "smooth", block: "nearest" });
  }, [selected]);
  async function mutate(path: string, method: string, body?: unknown) {
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await api(path, json(method, body));
      days.reload();
      setNotice("Changes saved.");
    } catch (e) {
      setError((e as Error).message);
      days.reload();
    } finally {
      setBusy(false);
    }
  }
  function move(index: number, direction: number) {
    if (!day) return;
    const ids = day.activities.map((a) => a.id);
    [ids[index], ids[index + direction]] = [ids[index + direction], ids[index]];
    void mutate(`/api/v1/days/${day.id}/order`, "PUT", { activity_ids: ids });
  }
  return (
    <>
      <div className="section-heading">
        <div>
          <p className="eyebrow">Make each day your own</p>
          <h2>The day-by-day</h2>
        </div>
        <p className="muted small">All times in {trip.timezone}</p>
      </div>
      <ErrorMessage
        message={error || days.error || places.error}
        retry={() => {
          days.reload();
          places.reload();
        }}
      />
      <div className="day-tabs" aria-label="Choose itinerary day">
        {days.data?.map((d, i) => (
          <button
            className={day?.id === d.id ? "day-tab active" : "day-tab"}
            aria-pressed={day?.id === d.id}
            key={d.id}
            onClick={() => {
              setDayId(d.id);
              setSelected(null);
              setNotice("");
            }}
          >
            <span>Day {i + 1}</span>
            <strong>
              {dateLabel(d.date, { month: "short", day: "numeric" })}
            </strong>
            <small>{d.activities.length} activities</small>
          </button>
        ))}
      </div>
      {days.loading ? (
        <p role="status">Loading itinerary…</p>
      ) : (
        day && (
          <div className="itinerary-grid">
            <section>
              <div className="row day-heading">
                <h3>
                  {dateLabel(day.date, {
                    weekday: "long",
                    month: "long",
                    day: "numeric",
                  })}
                </h3>
                <button
                  className="small"
                  disabled={!places.data?.length || busy}
                  onClick={() => setEditing("new")}
                >
                  ＋ Add activity
                </button>
              </div>
              <p className="success save-status" role="status">
                {busy ? "Saving…" : notice}
              </p>
              {day.activities.length === 0 ? (
                <div className="empty panel">
                  <span className="empty-icon" aria-hidden="true">
                    ↗
                  </span>
                  <h3>A fresh day, all yours.</h3>
                  <p>
                    Add a place from your saved collection to start this day.
                  </p>
                  {!places.data?.length && (
                    <Link
                      className="text-link"
                      href={`/trips/${trip.id}/places`}
                    >
                      Discover places →
                    </Link>
                  )}
                </div>
              ) : (
                <ol className="activity-list">
                  {day.activities.map((activity, i) => {
                    const p = placeById.get(activity.saved_place_id);
                    return (
                      <li
                        key={activity.id}
                        id={`activity-${activity.id}`}
                        className={`activity-card panel ${selected === activity.id ? "selected" : ""}`}
                      >
                        <div className="activity-top">
                          <span className="stop-number">{i + 1}</span>
                          <button
                            className="activity-title"
                            aria-pressed={selected === activity.id}
                            onClick={() => setSelected(activity.id)}
                          >
                            {p?.details?.name ||
                              "Saved place (details unavailable)"}
                          </button>
                          <span className="activity-time">
                            {activity.start_time || "Any time"}
                          </span>
                        </div>
                        <div className="activity-body">
                          {p?.details ? (
                            <PlaceInfo place={p.details} />
                          ) : (
                            <p className="muted">
                              Place details are unavailable. You can still edit
                              this activity.
                            </p>
                          )}
                          <span className="duration">
                            ◷ {activity.duration_minutes} min
                          </span>
                          {activity.notes && (
                            <p className="note">{activity.notes}</p>
                          )}
                          {crossesMidnight(
                            activity.start_time,
                            activity.duration_minutes,
                          ) && (
                            <p className="warning">
                              This activity ends after midnight. It stays on
                              this day.
                            </p>
                          )}
                          <div className="activity-actions">
                            <div className="actions">
                              <button
                                className="small quiet"
                                onClick={() => setEditing(activity)}
                              >
                                Edit
                              </button>
                              <button
                                className="small quiet danger"
                                disabled={busy}
                                onClick={() =>
                                  mutate(
                                    `/api/v1/activities/${activity.id}`,
                                    "DELETE",
                                  )
                                }
                              >
                                Remove
                              </button>
                            </div>
                            <div className="actions">
                              <button
                                className="icon-button"
                                aria-label={`Move activity ${i + 1} up`}
                                disabled={i === 0 || busy}
                                onClick={() => move(i, -1)}
                              >
                                ↑
                              </button>
                              <button
                                className="icon-button"
                                aria-label={`Move activity ${i + 1} down`}
                                disabled={
                                  i === day.activities.length - 1 || busy
                                }
                                onClick={() => move(i, 1)}
                              >
                                ↓
                              </button>
                            </div>
                          </div>
                        </div>
                      </li>
                    );
                  })}
                </ol>
              )}
              <GoogleAttribution />
            </section>
            <DayMap
              activities={markers}
              selected={selected}
              onSelect={setSelected}
            />
          </div>
        )
      )}
      {editing && day && (
        <Modal
          title={editing === "new" ? "Add to your day" : "Edit activity"}
          onClose={() => setEditing(null)}
        >
          <ActivityForm
            activity={editing === "new" ? undefined : editing}
            dayId={day.id}
            places={places.data || []}
            onSaved={() => {
              setEditing(null);
              days.reload();
              setNotice("Changes saved.");
            }}
          />
        </Modal>
      )}
    </>
  );
}
