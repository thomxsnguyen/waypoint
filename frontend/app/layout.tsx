import type { Metadata } from "next";
import "./globals.css";
export const metadata: Metadata = {
  title: "Waypoint — Your personal trip planner",
  description:
    "Save places and put your next trip together, one day at a time.",
};
export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
