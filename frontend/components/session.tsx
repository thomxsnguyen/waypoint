"use client";
import {
  ClerkProvider,
  SignInButton,
  SignUpButton,
  UserButton,
  useAuth,
  useClerk,
} from "@clerk/nextjs";
import Link from "next/link";
import { useCallback, useEffect, useState, type ReactNode } from "react";
import { APIContext, APIError, request } from "@/lib/api";

function Header({ account }: { account?: ReactNode }) {
  return (
    <header className="topbar">
      <Link className="brand" href="/trips">
        <span className="brand-mark" aria-hidden="true">
          ↗
        </span>
        waypoint<span className="brand-dot">.</span>
      </Link>
      <div className="topbar-right">
        <span className="tagline">A little planning. A world to explore.</span>
        {account}
      </div>
    </header>
  );
}
export function AppSession({ children }: { children: ReactNode }) {
  const key = process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY;
  if (!key)
    return (
      <>
        <Header />
        <main className="container">
          <section className="empty panel">
            <p className="eyebrow">Welcome to Waypoint</p>
            <h1>Your next trip starts here.</h1>
            <p>Connect authentication to start planning.</p>
            <p className="muted">
              Set the Clerk keys in your environment, then restart the app.
              Setup instructions are in the project README.
            </p>
            <Health />
          </section>
        </main>
      </>
    );
  return (
    <ClerkProvider publishableKey={key}>
      <Authenticated>{children}</Authenticated>
    </ClerkProvider>
  );
}
function Authenticated({ children }: { children: ReactNode }) {
  const { isLoaded, isSignedIn, getToken, userId } = useAuth();
  const { signOut } = useClerk();
  const [expired, setExpired] = useState(false);
  const api = useCallback(
    async <T,>(path: string, options?: RequestInit) => {
      try {
        return await request<T>(await getToken(), path, options);
      } catch (error) {
        if (error instanceof APIError && error.status === 401) setExpired(true);
        throw error;
      }
    },
    [getToken],
  );
  return (
    <>
      <Header account={isSignedIn ? <UserButton /> : undefined} />
      {!isLoaded ? (
        <main className="container">
          <p role="status">Loading your session…</p>
        </main>
      ) : !isSignedIn ? (
        <main className="container">
          <section className="welcome">
            <p className="eyebrow">Your plans, in one place</p>
            <h1>
              Make room
              <br />
              for the journey.
            </h1>
            <p>
              Save the places you want to go. Put your days together.
              <br />
              Come back to a plan that feels like you.
            </p>
            <div className="actions">
              <SignInButton mode="modal">
                <button>Sign in</button>
              </SignInButton>
              <SignUpButton mode="modal">
                <button className="secondary">Create account</button>
              </SignUpButton>
            </div>
            <Health />
          </section>
        </main>
      ) : expired ? (
        <main className="container">
          <h1>Please sign in again</h1>
          <p>Your session has expired. Your saved plans are still here.</p>
          <button onClick={() => signOut({ redirectUrl: "/trips" })}>
            Sign in again
          </button>
        </main>
      ) : (
        <APIContext.Provider key={userId} value={api}>
          {children}
        </APIContext.Provider>
      )}
      <footer>
        <span>Waypoint · Personal trip planner</span>
        <span>
          <Link href="/terms">Terms</Link>
          <Link href="/privacy">Privacy</Link>
        </span>
      </footer>
    </>
  );
}
function Health() {
  const [healthy, setHealthy] = useState<boolean | null>(null);
  useEffect(() => {
    request<{ status: string }>(null, "/health")
      .then(() => setHealthy(true))
      .catch(() => setHealthy(false));
  }, []);
  return (
    <p className="health" role="status">
      {healthy === null
        ? "Connecting…"
        : healthy
          ? "● Connected to the planning API"
          : "○ Planning API unavailable"}
    </p>
  );
}
