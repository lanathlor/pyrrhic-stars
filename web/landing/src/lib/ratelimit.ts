// In-memory IP token bucket. Process-local; not shared across replicas, which
// is acceptable for a marketing form. A single replica + sensible HPA settings
// will keep this useful enough.
//
// 5 requests per 15 minutes per IP. The map is swept every minute.

const CAPACITY = 5;
const WINDOW_MS = 15 * 60 * 1000;
const SWEEP_MS = 60 * 1000;

interface Bucket {
  count: number;
  resetAt: number;
}

const buckets = new Map<string, Bucket>();

if (typeof setInterval !== "undefined") {
  // Only run on the server.
  setInterval(() => {
    const now = Date.now();
    for (const [ip, b] of buckets) {
      if (b.resetAt <= now) buckets.delete(ip);
    }
  }, SWEEP_MS).unref?.();
}

export interface RateLimitResult {
  allowed: boolean;
  /** Seconds until the bucket resets. */
  retryAfter: number;
}

export function check(ip: string): RateLimitResult {
  const now = Date.now();
  let b = buckets.get(ip);
  if (!b || b.resetAt <= now) {
    b = { count: 0, resetAt: now + WINDOW_MS };
    buckets.set(ip, b);
  }
  if (b.count >= CAPACITY) {
    return {
      allowed: false,
      retryAfter: Math.max(1, Math.ceil((b.resetAt - now) / 1000)),
    };
  }
  b.count += 1;
  return { allowed: true, retryAfter: 0 };
}

/** Extract the client IP from a request, honoring X-Forwarded-For. */
export function clientIp(request: Request): string {
  const xff = request.headers.get("x-forwarded-for");
  if (xff) return xff.split(",")[0]!.trim();
  const xr = request.headers.get("x-real-ip");
  if (xr) return xr.trim();
  return "unknown";
}
