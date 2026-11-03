import { test, expect } from "@playwright/test";
import { mapsScript } from "./fixtures/maps";
test.beforeEach(async ({ page }) => {
  await page.route("https://maps.googleapis.com/maps/api/js?**", (route) =>
    route.fulfill({ contentType: "application/javascript", body: mapsScript }),
  );
});
test("persistent multi-day itinerary, marker selection, ownership, and deletion", async ({
  page,
  request,
}) => {
  await page.goto("/trips");
  await page.getByRole("button", { name: "Sign in as traveler A" }).click();
  await page.getByRole("button", { name: "New trip" }).click();
  await page.getByLabel("Trip name").fill("Tokyo autumn journey");
  await page.getByLabel("Destination", { exact: true }).fill("Tokyo, Japan");
  await page.getByLabel("Start date").fill("2026-11-12");
  await page.getByLabel("End date").fill("2026-11-16");
  await page.getByLabel("Time zone").fill("Asia/Tokyo");
  await page.getByLabel("Description").fill("Food and museums");
  await page.getByRole("button", { name: "Create trip", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Tokyo autumn journey" }),
  ).toBeVisible();
  const tripURL = page.url();
  await expect(page.getByText("Food and museums")).toBeVisible();
  await page.getByRole("link", { name: "Discover & save" }).click();
  await page.getByLabel("Search places").fill("museum");
  await expect(
    page.getByRole("heading", { name: "Tokyo National Museum", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("article")
    .filter({
      has: page.getByRole("heading", {
        name: "Tokyo National Museum",
        exact: true,
      }),
    })
    .getByRole("button", { name: "Save to trip" })
    .click();
  await page
    .getByRole("article")
    .filter({
      has: page.getByRole("heading", { name: "Sensō-ji", exact: true }),
    })
    .getByRole("button", { name: "Save to trip" })
    .click();
  await expect(
    page.getByRole("button", { name: "✓ Saved", exact: true }),
  ).toHaveCount(2);
  await page.reload();
  await expect(
    page.getByRole("heading", { name: "Tokyo National Museum", exact: true }),
  ).toBeVisible();
  await page.getByRole("link", { name: "Itinerary", exact: true }).click();
  await page.getByRole("button", { name: /Day 2/ }).click();
  const add = async (place: string, time: string, duration: string) => {
    await page.getByRole("button", { name: "Add activity" }).click();
    await page.getByLabel("Saved place").selectOption({ label: place });
    await page.getByLabel("Start time").fill(time);
    await page.getByLabel("Duration (minutes)").fill(duration);
    await page.getByLabel("Notes", { exact: true }).fill("Buy tickets onsite");
    await page
      .getByRole("dialog")
      .getByRole("button", { name: "Add activity" })
      .click();
  };
  await add("Tokyo National Museum", "09:30", "90");
  await expect(page.locator(".activity-card")).toHaveCount(1);
  await add("Sensō-ji", "12:00", "60");
  await expect(page.locator(".activity-card")).toHaveCount(2);
  await page.getByRole("button", { name: "Move activity 2 up" }).click();
  await expect(page.locator(".activity-card").first()).toContainText(
    "Sensō-ji",
  );
  await page
    .getByRole("button", { name: "Tokyo National Museum", exact: true })
    .click();
  await expect(
    page.getByRole("button", {
      name: "Map: Tokyo National Museum, activity 2",
    }),
  ).toHaveAttribute("aria-pressed", "true");
  await page.getByRole("button", { name: "Map: Sensō-ji, activity 1" }).click();
  await expect(
    page.getByRole("button", { name: "Sensō-ji", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  await page
    .locator(".activity-card")
    .first()
    .getByRole("button", { name: "Edit", exact: true })
    .click();
  await page.getByLabel("Start time").fill("23:30");
  await page.getByLabel("Duration (minutes)").fill("90");
  await page.getByRole("button", { name: "Save activity" }).click();
  await expect(
    page.getByText("This activity ends after midnight.", { exact: false }),
  ).toBeVisible();
  await page.getByRole("button", { name: /Day 3/ }).click();
  await add("Tokyo National Museum", "10:00", "45");
  await expect(page.locator(".activity-card")).toHaveCount(1);
  await page.reload();
  await page.getByRole("button", { name: /Day 2/ }).click();
  await expect(page.locator(".activity-card").first()).toContainText(
    "Sensō-ji",
  );
  await expect(page.locator(".activity-card").last()).toContainText("09:30");
  await page.getByRole("button", { name: "Edit trip", exact: true }).click();
  await page.getByLabel("End date").fill("2026-11-12");
  await page.getByRole("button", { name: "Save trip", exact: true }).click();
  await expect(page.locator(".error[role=alert]")).toContainText(
    "Remove activities",
  );
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  await page.getByRole("link", { name: "Discover & save" }).click();
  await page
    .getByRole("article")
    .filter({
      has: page.getByRole("heading", { name: "Sensō-ji", exact: true }),
    })
    .getByRole("button", { name: "Remove", exact: true })
    .click();
  await expect(page.locator(".error[role=alert]")).toContainText(
    "Remove this place's itinerary activities",
  );
  await page.getByRole("button", { name: "Sign out", exact: true }).click();
  await page.getByRole("button", { name: "Sign in as traveler B" }).click();
  await expect(page.locator(".error[role=alert]")).toContainText(
    "Resource not found",
  );
  const id = tripURL.split("/").pop();
  const res = await request.get(`http://localhost:18080/api/v1/trips/${id}`, {
    headers: { Authorization: "Bearer e2e-user-b" },
  });
  expect(res.status()).toBe(404);
  await page.getByRole("button", { name: "Sign out", exact: true }).click();
  await page.getByRole("button", { name: "Sign in as traveler A" }).click();
  await page.goto(tripURL + "/itinerary");
  await page.getByRole("button", { name: /Day 2/ }).click();
  await expect(page.locator(".activity-card")).toHaveCount(2);
  await page.screenshot({
    path: "test-results/itinerary-desktop.png",
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({
    path: "test-results/itinerary-mobile.png",
    fullPage: true,
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBeTruthy();
  await page.getByRole("button", { name: "Delete trip", exact: true }).click();
  await page.getByRole("button", { name: "Delete permanently" }).click();
  await expect(page.getByRole("heading", { name: "My trips" })).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Tokyo autumn journey" }),
  ).toHaveCount(0);
});
