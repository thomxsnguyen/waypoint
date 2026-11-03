import { AppSession } from "@/components/session";
export default function TripsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <AppSession>{children}</AppSession>;
}
