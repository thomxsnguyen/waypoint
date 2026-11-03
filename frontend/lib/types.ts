export type Trip = {
  id: string;
  name: string;
  destination: string;
  start_date: string;
  end_date: string;
  timezone: string;
  description: string;
};
export type TripInput = Omit<Trip, "id">;
export type Place = {
  provider: string;
  provider_place_id: string;
  name: string;
  address: string;
  location: { lat: number; lng: number } | null;
  external_url: string;
  attributions: { name: string; url: string }[];
};
export type SavedPlace = {
  id: string;
  trip_id: string;
  provider_place_id: string;
  notes: string;
  details: Place | null;
  unavailable: boolean;
};
export type Activity = {
  id: string;
  day_id: string;
  saved_place_id: string;
  start_time: string | null;
  duration_minutes: number;
  notes: string;
  position: number;
};
export type Day = { id: string; date: string; activities: Activity[] };
export type ActivityInput = Pick<
  Activity,
  "saved_place_id" | "start_time" | "duration_minutes" | "notes"
>;
