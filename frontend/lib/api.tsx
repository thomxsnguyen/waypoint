"use client";
import {
  createContext,
  useContext,
  useCallback,
  useState,
  useEffect,
} from "react";
export type Request = <T>(path: string, options?: RequestInit) => Promise<T>;
export const APIContext = createContext<Request | null>(null);
export function useAPI() {
  const value = useContext(APIContext);
  if (!value) throw new Error("Session required");
  return value;
}
export class APIError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}
export async function request<T>(
  token: string | null,
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080"}${path}`,
    {
      ...options,
      cache: "no-store",
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...options.headers,
      },
    },
  );
  if (res.status === 204) return undefined as T;
  const body = await res.json();
  if (!res.ok)
    throw new APIError(
      res.status,
      body.error?.message || "Request failed. Please try again.",
    );
  return body.data as T;
}
export function useResource<T>(path: string, refreshKey = "") {
  const api = useAPI();
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [revision, setRevision] = useState(0);
  const reload = useCallback(() => setRevision((r) => r + 1), []);
  useEffect(() => {
    let active = true;
    const abort = new AbortController();
    api<T>(path, { signal: abort.signal })
      .then((value) => {
        if (active) {
          setData(value);
          setError("");
        }
      })
      .catch((e) => {
        if (active) setError(e.message);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
      abort.abort();
    };
  }, [api, path, revision, refreshKey]);
  return { data, error, loading, reload };
}
export function json(method: string, body?: unknown): RequestInit {
  return {
    method,
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  };
}
