import Link from "next/link";
export default function Terms() {
  return (
    <main className="container legal">
      <Link href="/trips">← Waypoint</Link>
      <h1>Terms of use</h1>
      <p>
        Waypoint helps you organize personal travel plans. You control your
        itinerary and are responsible for confirming place details, opening
        times, bookings, and your travel arrangements. Waypoint does not make
        bookings or provide route optimization.
      </p>
      <p>
        Map and place information is supplied by Google. By using these
        features, you agree to the{" "}
        <a href="https://maps.google.com/help/terms_maps/">
          Google Maps / Google Earth Additional Terms of Service
        </a>{" "}
        and{" "}
        <a href="https://policies.google.com/terms">Google Terms of Service</a>.
      </p>
      <p>
        Third-party services can be unavailable and their information can
        change. Keep your own copy of essential travel arrangements.
      </p>
      <p>
        Use only your own account. Do not attempt to access another person’s
        plans or misuse the service.
      </p>
      <Link href="/privacy">Privacy policy →</Link>
    </main>
  );
}
