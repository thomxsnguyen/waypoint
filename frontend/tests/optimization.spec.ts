import { test, expect } from "@playwright/test";
import { mapsScript } from "./fixtures/maps";

test("review, cancel, stale approval, apply and reload", async ({
  page,
  request,
}) => {
  const headers = { Authorization: "Bearer e2e-user-a" };
  const base = "http://localhost:18080/api/v1";
  const tomorrow = new Date(Date.now() + 86400000).toISOString().slice(0, 10);
  const response = await request.post(`${base}/trips`, {
    headers,
    data: {
      name: "Optimization review",
      destination: "Tokyo",
      start_date: tomorrow,
      end_date: tomorrow,
      timezone: "Asia/Tokyo",
    },
  });
  expect(response.status()).toBe(201);
  const { data: trip } = await response.json();
  const dayResponse = await request.get(`${base}/trips/${trip.id}/days`, {
    headers,
  });
  const { data: days } = await dayResponse.json();
  for (const placeId of ["tokyo-museum", "sensoji"]) {
    const saved = await request.post(`${base}/trips/${trip.id}/places`, {
      headers,
      data: { provider_place_id: placeId },
    });
    const { data: place } = await saved.json();
    expect(
      (
        await request.post(`${base}/days/${days[0].id}/activities`, {
          headers,
          data: {
            saved_place_id: place.id,
            duration_minutes: 30,
            notes: "Keep this note",
          },
        })
      ).status(),
    ).toBe(201);
  }
  const readDays = async () =>
    (
      await (
        await request.get(`${base}/trips/${trip.id}/days`, { headers })
      ).json()
    ).data;
  const before = await readDays();
  await page.route("https://maps.googleapis.com/maps/api/js?**", (route) =>
    route.fulfill({ contentType: "application/javascript", body: mapsScript }),
  );
  await page.goto(`/trips/${trip.id}/itinerary`);
  await page.getByRole("button", { name: "Sign in as traveler A" }).click();
  await page.getByRole("button", { name: "Optimize day", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Optimize this day" });
  await dialog.getByRole("button", { name: "Generate preview" }).click();
  await expect(dialog.getByText("7 min estimated travel saved")).toBeVisible();
  expect(await readDays()).toEqual(before);
  await dialog.getByLabel("Travel mode").selectOption("DRIVE");
  await expect(
    dialog.getByRole("region", { name: "Schedule comparison" }),
  ).toHaveCount(0);
  await dialog.getByRole("button", { name: "Generate preview" }).click();
  await expect(dialog.getByText("7 min estimated travel saved")).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({
    path: "test-results/optimization-mobile.png",
    fullPage: true,
  });
  await dialog
    .getByRole("button", { name: "Apply proposed schedule" })
    .scrollIntoViewIfNeeded();
  await page.screenshot({
    path: "test-results/optimization-mobile-comparison.png",
  });
  expect(
    await dialog.evaluate((el) => el.scrollWidth <= el.clientWidth),
  ).toBeTruthy();
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  expect(await readDays()).toEqual(before);
  await page.getByRole("button", { name: "Optimize day", exact: true }).click();
  await dialog.getByRole("button", { name: "Generate preview" }).click();
  await expect(dialog.getByText("7 min estimated travel saved")).toBeVisible();
  await request.patch(`${base}/activities/${before[0].activities[0].id}`, {
    headers,
    data: { notes: "Edited in another tab" },
  });
  await dialog.getByRole("button", { name: "Apply proposed schedule" }).click();
  await expect(dialog.getByRole("alert")).toContainText("itinerary changed");
  await expect(
    dialog.getByRole("button", { name: "Apply proposed schedule" }),
  ).toHaveCount(0);
  expect((await readDays())[0].activities[0].notes).toBe(
    "Edited in another tab",
  );
  await dialog
    .getByText("Visiting windows (optional)", { exact: true })
    .click();
  await dialog
    .getByLabel("Earliest start for Tokyo National Museum")
    .fill("10:00");
  await dialog.getByRole("button", { name: "Generate preview" }).click();
  await expect(dialog.getByRole("alert")).toContainText(
    "Enter both visiting-window bounds",
  );
  await dialog
    .getByLabel("Latest finish for Tokyo National Museum")
    .fill("10:20");
  await dialog.getByRole("button", { name: "Generate preview" }).click();
  await expect(
    dialog.getByRole("button", { name: "Apply proposed schedule" }),
  ).toBeDisabled();
  await expect(
    dialog.getByRole("region", { name: "Proposed schedule", exact: true }),
  ).toContainText("activity exceeds visiting window");
  await dialog
    .getByLabel("Latest finish for Tokyo National Museum")
    .fill("10:30");
  await dialog.getByLabel("Starting place").selectOption({ label: "Sensō-ji" });
  await dialog.getByLabel("Travel mode").selectOption("TRANSIT");
  await dialog
    .getByLabel("Earliest start for Tokyo National Museum")
    .scrollIntoViewIfNeeded();
  await page.screenshot({
    path: "test-results/optimization-windows-mobile.png",
  });
  expect(
    await dialog.evaluate((el) => el.scrollWidth <= el.clientWidth),
  ).toBeTruthy();
  await dialog.getByRole("button", { name: "Generate preview" }).click();
  await expect(dialog.getByText("10 min estimated travel saved")).toBeVisible();
  await expect(
    dialog.getByRole("region", { name: "Proposed schedule", exact: true }),
  ).toContainText("10:00 – 10:30");
  await dialog.getByRole("button", { name: "Apply proposed schedule" }).click();
  await expect(dialog).toHaveCount(0);
  await expect(page.getByText("Approved schedule saved.")).toBeVisible();
  await page.reload();
  await expect(page.locator(".activity-card").first()).toContainText(
    "Sensō-ji",
  );
  await expect(page.locator(".activity-card").last()).toContainText("10:00");
  await expect(page.locator(".activity-card").last()).toContainText(
    "Edited in another tab",
  );
  expect(
    (await request.delete(`${base}/trips/${trip.id}`, { headers })).status(),
  ).toBe(204);
});
