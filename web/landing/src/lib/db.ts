// Postgres pool singleton + schema bootstrap.
//
// The pool is module-level and cached on `globalThis` so HMR in dev does not
// leak connections. `max: 2` keeps the per-replica connection footprint low.
// the subscribe endpoint is bursty but not high-throughput, and the marketing
// site should never hold many sockets open against the game's Postgres.

import pg from "pg";

const DSN = process.env.POSTGRES_DSN;

interface PoolHolder {
  pool: pg.Pool;
  initPromise: Promise<void>;
}

declare global {
  // eslint-disable-next-line no-var
  var __landingPg: PoolHolder | undefined;
}

const SCHEMA = `
  CREATE TABLE IF NOT EXISTS landing_subscribers (
    id          BIGSERIAL    PRIMARY KEY,
    email       TEXT         UNIQUE NOT NULL,
    source      TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
  );
`;

function build(): PoolHolder {
  if (!DSN) {
    throw new Error("POSTGRES_DSN is not set");
  }
  const pool = new pg.Pool({
    connectionString: DSN,
    max: 2,
    idleTimeoutMillis: 10_000,
  });
  // Eagerly run schema once. The promise is awaited by callers before their
  // first INSERT so we never race the DDL.
  const initPromise = pool
    .query(SCHEMA)
    .then(() => undefined)
    .catch((err) => {
      // Log and rethrow. The caller will see the rejection and return 503.
      console.error("[landing] schema bootstrap failed:", err);
      throw err;
    });
  return { pool, initPromise };
}

function holder(): PoolHolder {
  if (!globalThis.__landingPg) {
    globalThis.__landingPg = build();
  }
  return globalThis.__landingPg;
}

/** Returns true if POSTGRES_DSN is configured. Callers should 503 otherwise. */
export function dbConfigured(): boolean {
  return !!DSN;
}

/** Idempotent insert. Returns true if a new row was added. */
export async function insertSubscriber(
  email: string,
  source: string | null,
): Promise<boolean> {
  const h = holder();
  await h.initPromise;
  const res = await h.pool.query(
    `INSERT INTO landing_subscribers (email, source)
       VALUES ($1, $2)
       ON CONFLICT (email) DO NOTHING`,
    [email, source],
  );
  return (res.rowCount ?? 0) > 0;
}
