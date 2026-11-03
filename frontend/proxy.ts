import { clerkMiddleware } from "@clerk/nextjs/server";
import { NextResponse } from "next/server";
export default process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY &&
!(process.env.WAYPOINT_E2E === "1" && process.env.NODE_ENV !== "production")
  ? clerkMiddleware()
  : () => NextResponse.next();
export const config = {
  matcher: [
    "/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico)).*)",
    "/(api|trpc)(.*)",
  ],
};
