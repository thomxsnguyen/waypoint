"use client";
// Development-only test adapter selected by next.config.ts, never a production login.
import Link from "next/link";
import { useCallback, useSyncExternalStore, type ReactNode } from "react";
import { APIContext, request } from "@/lib/api";
const subscribe = (cb: () => void) => {
  window.addEventListener("e2e-session", cb);
  return () => window.removeEventListener("e2e-session", cb);
};
const snapshot = () => sessionStorage.getItem("waypoint-e2e-user") || "";
const empty = () => "";
function change(user: string) {
  sessionStorage.setItem("waypoint-e2e-user", user);
  window.dispatchEvent(new Event("e2e-session"));
}
export function AppSession({ children }: { children: ReactNode }) {
  const user = useSyncExternalStore(subscribe, snapshot, empty);
  const api = useCallback(
    <T,>(path: string, options?: RequestInit) =>
      request<T>(user, path, options),
    [user],
  );
  return (
    <>
      <header className="topbar">
        <Link className="brand" href="/trips">
          <span className="brand-mark">↗</span>waypoint.
        </Link>
        <span>Browser test session</span>
        {user && (
          <button className="small secondary" onClick={() => change("")}>
            Sign out
          </button>
        )}
      </header>
      {user ? (
        <APIContext.Provider key={user} value={api}>
          {children}
        </APIContext.Provider>
      ) : (
        <main className="container">
          <h1>Sign in</h1>
          <div className="actions">
            <button onClick={() => change("e2e-user-a")}>
              Sign in as traveler A
            </button>
            <button onClick={() => change("e2e-user-b")}>
              Sign in as traveler B
            </button>
          </div>
        </main>
      )}
    </>
  );
}
