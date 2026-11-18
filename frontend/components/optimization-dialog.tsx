"use client";
import { useState, type FormEvent } from "react";
import { Modal, ErrorMessage } from "@/components/ui";
import { json, useAPI } from "@/lib/api";
import type { Day, SavedPlace } from "@/lib/types";

type Schedule = {
  visits: { activity_id: string; start_minute: number; end_minute: number }[];
  conflicts: { activity_id: string; reason: string }[];
  travel_minutes: number;
};
type Preview = {
  original: Schedule;
  proposed: Schedule;
  savings_minutes: number | null;
  explanation: string;
  assumptions: string[];
  approval_id: string | null;
};
function clock(minute: number) {
  if (minute < 0) return "Unknown";
  return `${String(Math.floor(minute / 60)).padStart(2, "0")}:${String(minute % 60).padStart(2, "0")}`;
}
function minutes(value: string) {
  const [hour, minute] = value.split(":").map(Number);
  return hour * 60 + minute;
}
export function OptimizationDialog({
  tripId,
  timezone,
  day,
  places,
  onClose,
  onSaved,
}: {
  tripId: string;
  timezone: string;
  day: Day;
  places: SavedPlace[];
  onClose: () => void;
  onSaved: () => void;
}) {
  const api = useAPI();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [preview, setPreview] = useState<Preview | null>(null);
  const base = `/api/v1/trips/${tripId}/days/${day.id}`;
  function name(id: string) {
    const activity = day.activities.find((a) => a.id === id);
    return (
      places.find((p) => p.id === activity?.saved_place_id)?.details?.name ||
      `Activity ${day.activities.findIndex((a) => a.id === id) + 1} (place details unavailable)`
    );
  }
  async function generate(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const data = new FormData(e.currentTarget);
    const start = minutes(String(data.get("start")));
    const end = minutes(String(data.get("end")));
    setPreview(null);
    setError("");
    if (!Number.isFinite(start) || !Number.isFinite(end) || start >= end) {
      setError("Day end must be after day start.");
      return;
    }
    const windows: {
      activity_id: string;
      start_minute: number;
      end_minute: number;
    }[] = [];
    for (const activity of day.activities) {
      const opening = String(data.get(`opening-${activity.id}`) || "");
      const closing = String(data.get(`closing-${activity.id}`) || "");
      if (!opening && !closing) continue;
      if (!opening || !closing || minutes(opening) >= minutes(closing)) {
        setError(
          `Enter both visiting-window bounds, with finish after start, for ${name(activity.id)}.`,
        );
        return;
      }
      windows.push({
        activity_id: activity.id,
        start_minute: minutes(opening),
        end_minute: minutes(closing),
      });
    }
    setBusy(true);
    try {
      setPreview(
        await api<Preview>(
          `${base}/optimization-preview`,
          json("POST", {
            travel_mode: data.get("mode"),
            origin_saved_place_id: data.get("origin"),
            start_minute: start,
            end_minute: end,
            fixed_activity_ids: data.getAll("fixed"),
            visiting_windows: windows,
          }),
        ),
      );
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function apply() {
    if (!preview?.approval_id) return;
    setBusy(true);
    setError("");
    try {
      await api(
        `${base}/optimization-apply`,
        json("POST", { approval_id: preview.approval_id }),
      );
      onSaved();
    } catch (e) {
      setError(
        `${(e as Error).message} Generate a new preview before applying.`,
      );
      setPreview(null);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal
      title="Optimize this day"
      onClose={() => {
        if (!busy) onClose();
      }}
    >
      <p>
        {day.date} · All times in {timezone}
      </p>
      <ErrorMessage message={error} />
      <form
        onSubmit={generate}
        onChange={() => {
          setPreview(null);
          setError("");
        }}
      >
        <fieldset disabled={busy} className="optimization-inputs">
          <label>
            Travel mode
            <select name="mode" defaultValue="WALK">
              <option value="WALK">Walking</option>
              <option value="DRIVE">Driving (without traffic)</option>
              <option value="TRANSIT">Public transit</option>
            </select>
          </label>
          <p className="muted small">
            Transit supports up to 10 locations including your starting place,
            within 7 days past or 100 days ahead. It is unavailable on
            daylight-saving transition days.
          </p>
          <label>
            Starting place
            <select name="origin" defaultValue="">
              <option value="">Start at the first activity</option>
              {places.map((p, i) => (
                <option key={p.id} value={p.id}>
                  {p.details?.name ||
                    `Saved place ${i + 1} (details unavailable)`}
                </option>
              ))}
            </select>
          </label>
          <p className="muted small">
            With a starting place, day start is your departure time. Return
            travel is not included.
          </p>
          <div className="optimization-columns">
            <label>
              Day start
              <input name="start" type="time" required defaultValue="09:00" />
            </label>
            <label>
              Day end
              <input name="end" type="time" required defaultValue="18:00" />
            </label>
          </div>
          <fieldset>
            <legend>Keep these start times fixed</legend>
            <p className="muted small">
              Unchecked activities can move to a different time.
            </p>
            {day.activities
              .filter((a) => a.start_time)
              .map((a) => (
                <label className="optimization-check" key={a.id}>
                  <input type="checkbox" name="fixed" value={a.id} />
                  {name(a.id)} · {a.start_time}
                </label>
              ))}
            {!day.activities.some((a) => a.start_time) && (
              <p>No saved start times to fix.</p>
            )}
          </fieldset>
          <details>
            <summary>Visiting windows (optional)</summary>
            <p className="muted small">
              Enter times you have checked. Each visit must fit fully within its
              window. These limits apply to this preview only and are not saved
              as opening hours.
            </p>
            {day.activities.map((activity) => (
              <fieldset key={activity.id}>
                <legend>{name(activity.id)}</legend>
                <div className="optimization-columns">
                  <label>
                    Earliest start for {name(activity.id)}
                    <input type="time" name={`opening-${activity.id}`} />
                  </label>
                  <label>
                    Latest finish for {name(activity.id)}
                    <input type="time" name={`closing-${activity.id}`} />
                  </label>
                </div>
              </fieldset>
            ))}
          </details>
          <button type="submit">
            {busy ? "Working…" : "Generate preview"}
          </button>
        </fieldset>
      </form>
      {preview && (
        <section aria-label="Schedule comparison">
          <p>{preview.explanation}</p>
          {preview.savings_minutes !== null && (
            <p>
              <strong>
                {preview.savings_minutes} min estimated travel saved
              </strong>
            </p>
          )}
          <div className="optimization-columns">
            {(
              [
                ["Original order", preview.original],
                ["Proposed schedule", preview.proposed],
              ] as const
            ).map(([title, schedule]) => (
              <section key={title} aria-label={title}>
                <h3>{title}</h3>
                <p className="muted small">
                  Estimated times, including travel and waiting
                </p>
                <p>
                  Travel:{" "}
                  {schedule.travel_minutes < 0
                    ? "Unavailable"
                    : `${schedule.travel_minutes} min`}
                </p>
                <ol>
                  {schedule.visits.map((v) => (
                    <li key={v.activity_id}>
                      <strong>{name(v.activity_id)}</strong>
                      <br />
                      {clock(v.start_minute)} – {clock(v.end_minute)}
                    </li>
                  ))}
                </ol>
                {schedule.conflicts.length > 0 && (
                  <ul className="warning">
                    {schedule.conflicts.map((c, i) => (
                      <li key={i}>
                        {name(c.activity_id)}: {c.reason}
                      </li>
                    ))}
                  </ul>
                )}
              </section>
            ))}
          </div>
          <ul className="muted small">
            {preview.assumptions.map((text) => (
              <li key={text}>{text}</li>
            ))}
          </ul>
          <p className="small">Travel estimates from Google Maps.</p>
          {preview.approval_id ? (
            <p>
              Apply saves this order and these start times. Approval expires
              after 10 minutes.
            </p>
          ) : (
            <p>
              Resolve the conflicts and generate a new preview before applying.
            </p>
          )}
          <button disabled={busy || !preview.approval_id} onClick={apply}>
            Apply proposed schedule
          </button>
        </section>
      )}
      <div className="actions">
        <button className="secondary" disabled={busy} onClick={onClose}>
          Cancel
        </button>
      </div>
      {busy && <p role="status">Working on your schedule…</p>}
    </Modal>
  );
}
