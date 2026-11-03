import Link from "next/link";
export default function Privacy() {
  return (
    <main className="container legal">
      <Link href="/trips">← Waypoint</Link>
      <h1>Privacy</h1>
      <p>
        Waypoint stores an authentication account identifier, your trips, saved
        place identifiers, activity schedules, and notes so you can reopen your
        plans. Trips are accessible only through the owning account.
      </p>
      <p>
        Clerk manages account registration and sign-in. Its processing is
        described in the{" "}
        <a href="https://clerk.com/legal/privacy">Clerk Privacy Policy</a>.
        Waypoint does not store your password.
      </p>
      <p>
        Place search sends your search text and destination to Google. Resolving
        saved places sends their Google place identifiers. Loading the map
        connects your browser to Google. See the{" "}
        <a href="https://policies.google.com/privacy">Google Privacy Policy</a>.
      </p>
      <p>
        Deleting a trip removes its saved places, days, and activities from the
        active database. Database backups may retain older records until they
        expire under the deployment operator’s retention policy. Operational
        logs record request status and timing without authorization tokens or
        note content.
      </p>
      <p>
        Contact the operator of this deployment for account data requests and
        its backup retention period.
      </p>
    </main>
  );
}
