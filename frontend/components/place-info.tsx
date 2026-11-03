import type { Place } from "@/lib/types";
export function GoogleAttribution() {
  return (
    <p className="attribution">
      Place information from{" "}
      <span aria-label="Google Maps" className="google-maps">
        Google Maps
      </span>
    </p>
  );
}
function safeURL(url: string) {
  try {
    const parsed = new URL(url);
    return parsed.protocol === "https:" || parsed.protocol === "http:"
      ? url
      : undefined;
  } catch {
    return undefined;
  }
}
export function PlaceInfo({ place }: { place: Place }) {
  return (
    <>
      <p className="muted">{place.address}</p>
      {safeURL(place.external_url) && (
        <a
          className="text-link small"
          href={safeURL(place.external_url)}
          target="_blank"
          rel="noopener noreferrer"
        >
          View on Google Maps ↗
        </a>
      )}
      {place.attributions?.map((a, i) => (
        <p className="attribution" key={i}>
          {safeURL(a.url) ? (
            <a href={safeURL(a.url)} target="_blank" rel="noopener noreferrer">
              {a.name}
            </a>
          ) : (
            a.name
          )}
        </p>
      ))}
    </>
  );
}
