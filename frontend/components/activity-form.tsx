"use client";
import { useState, type FormEvent } from "react";
import type { Activity, ActivityInput, SavedPlace } from "@/lib/types";
import { json, useAPI } from "@/lib/api";
import { ErrorMessage } from "./ui";
export function ActivityForm({
  activity,
  dayId,
  places,
  onSaved,
}: {
  activity?: Activity;
  dayId: string;
  places: SavedPlace[];
  onSaved: () => void;
}) {
  const api = useAPI();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setBusy(true);
    setError("");
    const form = new FormData(e.currentTarget);
    const body: Partial<ActivityInput> = {
      start_time: String(form.get("start_time")) || null,
      duration_minutes: Number(form.get("duration_minutes")),
      notes: String(form.get("notes")),
    };
    if (!activity) body.saved_place_id = String(form.get("saved_place_id"));
    try {
      await api(
        activity
          ? `/api/v1/activities/${activity.id}`
          : `/api/v1/days/${dayId}/activities`,
        json(activity ? "PATCH" : "POST", body),
      );
      onSaved();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <form onSubmit={submit}>
      <ErrorMessage message={error} />
      {!activity && (
        <label>
          Saved place
          <select name="saved_place_id" required defaultValue="">
            <option disabled value="">
              Choose a place
            </option>
            {places.map((p, i) => (
              <option value={p.id} key={p.id}>
                {p.details?.name ||
                  `Saved place ${i + 1} (details unavailable)`}
              </option>
            ))}
          </select>
        </label>
      )}
      <div className="form-grid">
        <label>
          Start time <span className="optional">optional</span>
          <input
            name="start_time"
            type="time"
            defaultValue={activity?.start_time || ""}
          />
        </label>
        <label>
          Duration (minutes)
          <input
            name="duration_minutes"
            type="number"
            min={1}
            max={1440}
            required
            defaultValue={activity?.duration_minutes || 60}
          />
        </label>
      </div>
      <label>
        Notes
        <textarea
          name="notes"
          rows={3}
          maxLength={5000}
          defaultValue={activity?.notes}
          placeholder="Tickets, a reminder, or a little detail…"
        />
      </label>
      <p className="muted small">
        Times are local to your trip. Activities stay in the order you choose.
      </p>
      <button disabled={busy}>
        {busy ? "Saving…" : activity ? "Save activity" : "Add activity"}
      </button>
    </form>
  );
}
