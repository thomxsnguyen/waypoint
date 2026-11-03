"use client";
import { useEffect, useState, type FormEvent } from "react";
import { useTrip } from "@/components/trip-context";
import { ErrorMessage, Modal } from "@/components/ui";
import { GoogleAttribution, PlaceInfo } from "@/components/place-info";
import { json, useAPI, useResource } from "@/lib/api";
import type { Place, SavedPlace } from "@/lib/types";
export default function Places() {
  const { trip } = useTrip();
  const api = useAPI();
  const saved = useResource<SavedPlace[]>(`/api/v1/trips/${trip.id}/places`);
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<Place[] | null>(null);
  const [searching, setSearching] = useState(false);
  const [searchError, setSearchError] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const [editing, setEditing] = useState<SavedPlace | null>(null);
  const [notice, setNotice] = useState("");
  useEffect(() => {
    const abort = new AbortController();
    const q = query.trim();
    const timer = setTimeout(() => {
      if (q.length < 2) {
        setResults(null);
        setSearchError("");
        setSearching(false);
        return;
      }
      setSearching(true);
      setSearchError("");
      api<Place[]>(
        `/api/v1/places/search?trip_id=${trip.id}&q=${encodeURIComponent(q)}`,
        { signal: abort.signal },
      )
        .then(setResults)
        .catch((e) => {
          if (!abort.signal.aborted) setSearchError(e.message);
        })
        .finally(() => {
          if (!abort.signal.aborted) setSearching(false);
        });
    }, 450);
    return () => {
      clearTimeout(timer);
      abort.abort();
    };
  }, [query, trip.id, api]);
  async function mutate(
    id: string,
    path: string,
    method: string,
    body?: unknown,
  ) {
    setBusy(id);
    setError("");
    setNotice("");
    try {
      await api(path, json(method, body));
      saved.reload();
      setNotice(method === "DELETE" ? "Place removed." : "Saved to your trip.");
      return true;
    } catch (e) {
      setError((e as Error).message);
      return false;
    } finally {
      setBusy("");
    }
  }
  async function saveNotes(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!editing) return;
    const notes = new FormData(e.currentTarget).get("notes");
    if (
      await mutate(
        editing.id,
        `/api/v1/trips/${trip.id}/places/${editing.id}`,
        "PATCH",
        { notes },
      )
    )
      setEditing(null);
  }
  return (
    <>
      <div className="section-heading">
        <div>
          <p className="eyebrow">Collect a little inspiration</p>
          <h2>Find your favorite places</h2>
        </div>
        <span className="muted">Searching near {trip.destination}</span>
      </div>
      <div className="discovery-grid">
        <section>
          <label className="search-label">
            Search places
            <input
              type="search"
              value={query}
              maxLength={200}
              onChange={(e) => {
                setQuery(e.target.value);
                setResults(null);
              }}
              placeholder="Try museums, ramen, or a place name…"
            />
          </label>
          <GoogleAttribution />
          <ErrorMessage message={searchError} />
          {searching ? (
            <p role="status">Finding places…</p>
          ) : results?.length === 0 ? (
            <div className="empty panel">
              <h3>No places found</h3>
              <p>Try a more specific name or a different search.</p>
            </div>
          ) : results === null ? (
            <div className="empty panel search-empty">
              <span className="empty-icon" aria-hidden="true">
                ⌖
              </span>
              <h3>A great day starts with a place.</h3>
              <p>
                Search for attractions, restaurants, landmarks, or somewhere to
                stay.
              </p>
            </div>
          ) : (
            <div className="place-list">
              {results.map((place) => {
                const isSaved = saved.data?.some(
                  (p) => p.provider_place_id === place.provider_place_id,
                );
                return (
                  <article
                    className="panel place-card"
                    key={place.provider_place_id}
                  >
                    <div className="place-symbol" aria-hidden="true">
                      ⌖
                    </div>
                    <div className="place-content">
                      <h3>{place.name}</h3>
                      <PlaceInfo place={place} />
                      <button
                        className="small secondary"
                        disabled={Boolean(busy) || isSaved}
                        onClick={() =>
                          mutate(
                            place.provider_place_id,
                            `/api/v1/trips/${trip.id}/places`,
                            "POST",
                            { provider_place_id: place.provider_place_id },
                          )
                        }
                      >
                        {isSaved
                          ? "✓ Saved"
                          : busy === place.provider_place_id
                            ? "Saving…"
                            : "＋ Save to trip"}
                      </button>
                    </div>
                  </article>
                );
              })}
            </div>
          )}
        </section>
        <aside className="saved-panel">
          <div className="row">
            <h2>Saved places</h2>
            <span className="count">{saved.data?.length ?? 0}</span>
          </div>
          <p className="muted">Your collection for this trip.</p>
          <ErrorMessage message={error || saved.error} retry={saved.reload} />
          <p role="status" className="success">
            {notice}
          </p>
          {saved.loading ? (
            <p role="status">Loading saved places…</p>
          ) : saved.data?.length === 0 ? (
            <div className="empty">
              <p>
                Your collection is waiting.
                <br />
                Save a place to get started.
              </p>
            </div>
          ) : (
            saved.data?.map((place) => (
              <article className="saved-card" key={place.id}>
                <h3>{place.details?.name || "Saved place"}</h3>
                {place.details ? (
                  <PlaceInfo place={place.details} />
                ) : (
                  <p className="muted">
                    Place details are temporarily unavailable. Your saved place
                    and notes are safe.
                  </p>
                )}
                {place.notes && <p className="note">{place.notes}</p>}
                <div className="actions">
                  <button
                    className="small quiet"
                    onClick={() => setEditing(place)}
                  >
                    Edit notes
                  </button>
                  <button
                    className="small quiet danger"
                    disabled={Boolean(busy)}
                    onClick={() =>
                      mutate(
                        place.id,
                        `/api/v1/trips/${trip.id}/places/${place.id}`,
                        "DELETE",
                      )
                    }
                  >
                    Remove
                  </button>
                </div>
              </article>
            ))
          )}
        </aside>
      </div>
      {editing && (
        <Modal title="Place notes" onClose={() => setEditing(null)}>
          <form onSubmit={saveNotes}>
            <ErrorMessage message={error} />
            <label>
              Notes
              <textarea
                name="notes"
                rows={4}
                maxLength={5000}
                defaultValue={editing.notes}
              />
            </label>
            <button disabled={Boolean(busy)}>
              {busy ? "Saving…" : "Save notes"}
            </button>
          </form>
        </Modal>
      )}
    </>
  );
}
