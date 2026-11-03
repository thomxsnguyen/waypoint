"use client";
import { useEffect, useRef, useState } from "react";
export type MapActivity = {
  activityId: string;
  name: string;
  lat: number;
  lng: number;
  position: number;
};
let mapsPromise: Promise<void> | undefined;
function loadMaps(key: string) {
  if (
    typeof window.google !== "undefined" &&
    typeof window.google.maps !== "undefined"
  )
    return Promise.resolve();
  if (!mapsPromise)
    mapsPromise = new Promise<void>((resolve, reject) => {
      const script = document.createElement("script");
      script.src = `https://maps.googleapis.com/maps/api/js?key=${encodeURIComponent(key)}&v=weekly&loading=async&callback=waypointMapsReady`;
      script.async = true;
      const target = window as unknown as {
        waypointMapsReady: () => void;
        gm_authFailure: () => void;
      };
      target.waypointMapsReady = () => resolve();
      target.gm_authFailure = () =>
        reject(
          new Error(
            "Google Maps could not authenticate. Check the browser key restrictions.",
          ),
        );
      script.onerror = () => {
        mapsPromise = undefined;
        script.remove();
        reject(
          new Error(
            "The map could not load. Check your connection and try again.",
          ),
        );
      };
      document.head.appendChild(script);
    });
  return mapsPromise;
}
export function DayMap({
  activities,
  selected,
  onSelect,
}: {
  activities: MapActivity[];
  selected: string | null;
  onSelect: (id: string) => void;
}) {
  const element = useRef<HTMLDivElement>(null);
  const map = useRef<google.maps.Map | null>(null);
  const markers = useRef(
    new Map<
      string,
      {
        marker: google.maps.marker.AdvancedMarkerElement;
        button: HTMLButtonElement;
      }
    >(),
  );
  const callback = useRef(onSelect);
  const [ready, setReady] = useState(false);
  const [error, setError] = useState("");
  const [attempt, setAttempt] = useState(0);
  const key = process.env.NEXT_PUBLIC_MAPS_BROWSER_API_KEY;
  useEffect(() => {
    callback.current = onSelect;
  }, [onSelect]);
  useEffect(() => {
    let active = true;
    if (!key) return;
    loadMaps(key)
      .then(async () => {
        await Promise.all([
          google.maps.importLibrary("maps"),
          google.maps.importLibrary("marker"),
        ]);
        if (active && element.current) {
          map.current = new google.maps.Map(element.current, {
            mapId: process.env.NEXT_PUBLIC_MAPS_MAP_ID || "DEMO_MAP_ID",
            center: { lat: 20, lng: 0 },
            zoom: 2,
            mapTypeControl: false,
            streetViewControl: false,
            fullscreenControl: true,
          });
          setReady(true);
          setError("");
        }
      })
      .catch((e) => {
        if (active) setError(e.message);
      });
    return () => {
      active = false;
    };
  }, [key, attempt]);
  useEffect(() => {
    if (!ready || !map.current) return;
    const current = markers.current;
    for (const { marker } of current.values()) marker.map = null;
    current.clear();
    const bounds = new google.maps.LatLngBounds();
    for (const activity of activities) {
      const button = document.createElement("button");
      button.className = "map-pin";
      button.textContent = String(activity.position + 1);
      button.setAttribute(
        "aria-label",
        `Map: ${activity.name}, activity ${activity.position + 1}`,
      );
      button.onclick = () => callback.current(activity.activityId);
      const marker = new google.maps.marker.AdvancedMarkerElement({
        map: map.current,
        position: { lat: activity.lat, lng: activity.lng },
        title: activity.name,
        content: button,
      });
      current.set(activity.activityId, { marker, button });
      bounds.extend({ lat: activity.lat, lng: activity.lng });
    }
    if (activities.length === 1) {
      map.current.setCenter(bounds.getCenter());
      map.current.setZoom(14);
    } else if (activities.length > 1) map.current.fitBounds(bounds, 60);
    return () => {
      for (const { marker } of current.values()) marker.map = null;
      current.clear();
    };
  }, [ready, activities]);
  useEffect(() => {
    for (const [id, { marker, button }] of markers.current) {
      button.classList.toggle("selected", id === selected);
      button.setAttribute("aria-pressed", String(id === selected));
      if (id === selected && marker.position)
        map.current?.panTo(marker.position);
    }
  }, [selected, ready, activities]);
  return (
    <section className="map-panel panel" aria-label="Selected day map">
      <div className="map-heading">
        <h3>Your day, on the map</h3>
        <span className="badge">{activities.length} stops</span>
      </div>
      <div ref={element} className="map-canvas" />
      {!key ? (
        <div className="map-message">
          <p>Map unavailable</p>
          <p className="muted">
            Configure a Google Maps browser key to see your places here.
          </p>
        </div>
      ) : error ? (
        <div className="map-message" role="alert">
          <p>{error}</p>
          <button
            className="small secondary"
            onClick={() => setAttempt((n) => n + 1)}
          >
            Retry map
          </button>
        </div>
      ) : !ready ? (
        <div className="map-message" role="status">
          Loading map…
        </div>
      ) : activities.length === 0 ? (
        <div className="map-message">
          <p>No mapped activities yet.</p>
          <p className="muted">Add a saved place with available coordinates.</p>
        </div>
      ) : null}
      <p className="map-caption">Select a stop to find it in your itinerary.</p>
    </section>
  );
}
