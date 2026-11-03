"use client";
import { useState, type FormEvent } from "react";
import { json, useAPI } from "@/lib/api";
import type { Trip, TripInput } from "@/lib/types";
import { ErrorMessage } from "./ui";
export function TripForm({
  trip,
  onSaved,
  onCancel,
}: {
  trip?: Trip;
  onSaved: (trip: Trip) => void;
  onCancel: () => void;
}) {
  const api = useAPI();
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setBusy(true);
    setError("");
    const form = new FormData(e.currentTarget);
    const body = Object.fromEntries(form) as TripInput;
    try {
      const saved = await api<Trip>(
        trip ? `/api/v1/trips/${trip.id}` : "/api/v1/trips",
        json(trip ? "PATCH" : "POST", body),
      );
      onSaved(saved);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <form onSubmit={submit}>
      <ErrorMessage message={error} />
      <label>
        Trip name
        <input
          name="name"
          defaultValue={trip?.name}
          placeholder="A few days in Tokyo"
          required
          maxLength={120}
        />
      </label>
      <label>
        Destination
        <input
          name="destination"
          defaultValue={trip?.destination}
          placeholder="Tokyo, Japan"
          required
          maxLength={200}
        />
      </label>
      <div className="form-grid">
        <label>
          Start date
          <input
            name="start_date"
            type="date"
            defaultValue={trip?.start_date}
            required
          />
        </label>
        <label>
          End date
          <input
            name="end_date"
            type="date"
            defaultValue={trip?.end_date}
            required
          />
        </label>
      </div>
      <label>
        Time zone
        <input
          name="timezone"
          list="timezones"
          defaultValue={
            trip?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone
          }
          placeholder="Asia/Tokyo"
          required
        />
        <datalist id="timezones">
          {Intl.supportedValuesOf("timeZone").map((zone) => (
            <option key={zone} value={zone} />
          ))}
        </datalist>
      </label>
      <label>
        Description <span className="optional">optional</span>
        <textarea
          name="description"
          defaultValue={trip?.description}
          maxLength={5000}
          rows={3}
          placeholder="What are you looking forward to?"
        />
      </label>
      <div className="actions end">
        <button
          type="button"
          className="secondary"
          onClick={onCancel}
          disabled={busy}
        >
          Cancel
        </button>
        <button disabled={busy}>
          {busy ? "Saving…" : trip ? "Save trip" : "Create trip"}
        </button>
      </div>
    </form>
  );
}
