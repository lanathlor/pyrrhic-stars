import type { APIRoute } from "astro";
import { insertSubscriber, dbConfigured } from "../../lib/db.ts";
import { check, clientIp } from "../../lib/ratelimit.ts";

export const prerender = false;

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const MAX_EMAIL_LEN = 254; // RFC 5321
const MAX_SOURCE_LEN = 64;

function jsonError(status: number, error: string, extra?: Record<string, unknown>) {
  return new Response(JSON.stringify({ error, ...extra }), {
    status,
    headers: {
      "content-type": "application/json; charset=utf-8",
      "cache-control": "no-store",
    },
  });
}

export const POST: APIRoute = async ({ request }) => {
  // 1. Rate limit by IP. Done before parsing the body so abusers can't burn CPU.
  const ip = clientIp(request);
  const rl = check(ip);
  if (!rl.allowed) {
    return jsonError(429, "Slow down, please try again in a few minutes.", {
      retry_after_seconds: rl.retryAfter,
    });
  }

  // 2. Parse form-encoded or JSON body. The HTML form uses FormData, but
  //    an API client might POST JSON.
  let email = "";
  let consent = "";
  let company = "";
  let source = "home";

  const contentType = request.headers.get("content-type") || "";
  try {
    if (contentType.includes("application/json")) {
      const body = (await request.json()) as Record<string, unknown>;
      email = typeof body.email === "string" ? body.email : "";
      consent = typeof body.consent === "string" ? body.consent : "";
      company = typeof body.company === "string" ? body.company : "";
      if (typeof body.source === "string") source = body.source;
    } else {
      const fd = await request.formData();
      email = String(fd.get("email") || "");
      consent = String(fd.get("consent") || "");
      company = String(fd.get("company") || "");
      const srcField = fd.get("source");
      if (typeof srcField === "string") source = srcField;
    }
  } catch {
    return jsonError(400, "Invalid request body.");
  }

  // 3. Honeypot. Real users never fill this. Silent 202 so attackers don't
  //    learn anything.
  if (company.trim().length > 0) {
    return new Response(JSON.stringify({ ok: true }), {
      status: 202,
      headers: {
        "content-type": "application/json; charset=utf-8",
        "cache-control": "no-store",
      },
    });
  }

  // 4. Validate email + consent.
  email = email.trim().toLowerCase();
  if (!email || email.length > MAX_EMAIL_LEN || !EMAIL_RE.test(email)) {
    return jsonError(400, "Please enter a valid email address.");
  }
  if (consent !== "true") {
    return jsonError(400, "Please accept the privacy notice to continue.");
  }
  if (source.length > MAX_SOURCE_LEN) {
    source = source.slice(0, MAX_SOURCE_LEN);
  }

  // 5. DB not configured (e.g. local dev without postgres). Return 503.
  if (!dbConfigured()) {
    return jsonError(503, "Signups are temporarily unavailable. Try again later.");
  }

  // 6. Insert. ON CONFLICT DO NOTHING means duplicate emails are idempotent.
  try {
    await insertSubscriber(email, source);
  } catch (err) {
    console.error("[subscribe] insert failed", err);
    return jsonError(503, "Signups are temporarily unavailable. Try again later.");
  }

  return new Response(JSON.stringify({ ok: true }), {
    status: 202,
    headers: {
      "content-type": "application/json; charset=utf-8",
      "cache-control": "no-store",
    },
  });
};

// Other verbs: 405.
export const ALL: APIRoute = ({ request }) => {
  if (request.method === "POST") return new Response(null, { status: 404 });
  return new Response(JSON.stringify({ error: "Method not allowed" }), {
    status: 405,
    headers: {
      "content-type": "application/json; charset=utf-8",
      allow: "POST",
      "cache-control": "no-store",
    },
  });
};
