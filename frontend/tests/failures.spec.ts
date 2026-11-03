import { test, expect } from "@playwright/test";
import { mapsScript } from "./fixtures/maps";

test("provider outages preserve editable activities and date changes refresh days", async ({
  page,
  request,
}) => {
  const headers = { Authorization: "Bearer e2e-user-a" };
  const base = "http://localhost:18080/api/v1";
  const created = await request.post(`${base}/trips`, {
    headers,
    data: {
      name: "Unavailable details test",
      destination: "Tokyo",
      start_date: "2026-11-01",
      end_date: "2026-11-01",
      timezone: "Asia/Tokyo",
    },
  });
  expect(created.status()).toBe(201);
  const { data: trip } = await created.json();
  const saved = await request.post(`${base}/trips/${trip.id}/places`, {
    headers,
    data: { provider_place_id: "unavailable", notes: "Keep this reminder" },
  });
  const { data: place } = await saved.json();
  const { data: days } = await (
    await request.get(`${base}/trips/${trip.id}/days`, { headers })
  ).json();
  expect(
    (
      await request.post(`${base}/days/${days[0].id}/activities`, {
        headers,
        data: {
          saved_place_id: place.id,
          duration_minutes: 60,
          notes: "My original note",
        },
      })
    ).status(),
  ).toBe(201);
  await page.route("https://maps.googleapis.com/maps/api/js?**", (route) =>
    route.fulfill({ contentType: "application/javascript", body: mapsScript }),
  );
  await page.goto(`/trips/${trip.id}/itinerary`);
  await page.getByRole("button", { name: "Sign in as traveler A" }).click();
  await expect(
    page.getByText("Place details are unavailable.", { exact: false }),
  ).toBeVisible();
  await expect(page.getByText("No mapped activities yet.")).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Sunday, November 1" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Edit", exact: true }).click();
  await page
    .getByRole("textbox", { name: "Notes", exact: true })
    .fill("Still editable offline from provider");
  await page.getByRole("button", { name: "Save activity" }).click();
  await expect(
    page.getByText("Still editable offline from provider"),
  ).toBeVisible();
  await page.getByRole("button", { name: "Edit trip", exact: true }).click();
  await page.getByLabel("End date").fill("2026-11-02");
  await page.getByRole("button", { name: "Save trip", exact: true }).click();
  await expect(page.getByRole("button", { name: /Day 2/ })).toBeVisible();
  await page.getByRole("link", { name: "Discover & save" }).click();
  await expect(page.getByText("Keep this reminder")).toBeVisible();
  await page.getByRole("button", { name: "Edit notes" }).click();
  await page.getByRole("textbox", { name: "Notes", exact: true }).fill("Updated place note");
  await page.getByRole("button", { name: "Save notes" }).click();
  await expect(page.getByText("Updated place note")).toBeVisible();
  await page.getByLabel("Search places").fill("outage");
  await expect(page.locator(".error[role=alert]")).toContainText(
    "temporarily unavailable",
  );
  await page.getByLabel("Search places").fill("nothing");
  await expect(
    page.getByRole("heading", { name: "No places found" }),
  ).toBeVisible();
  await page.getByLabel("Search places").fill("museum");
  await expect(
    page.getByRole("heading", { name: "Tokyo National Museum", exact: true }),
  ).toBeVisible();
  await page.getByRole("link", { name: "Itinerary", exact: true }).click();
  await page.getByRole("button", { name: "Remove", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "A fresh day, all yours." }),
  ).toBeVisible();
  await page.getByRole("link", { name: "Discover & save" }).click();
  await page.getByRole("button", { name: "Remove", exact: true }).click();
  await expect(
    page.getByText("Your collection is waiting.", { exact: false }),
  ).toBeVisible();
  expect(
    (await request.delete(`${base}/trips/${trip.id}`, { headers })).status(),
  ).toBe(204);
});
