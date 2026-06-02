# syntax=docker/dockerfile:1.7
# Multi-stage build for the Pyrrhic Stars landing page (Astro SSR).
# Stage 1: install deps + run astro build
# Stage 2: runtime with prod deps only, non-root user, healthcheck

ARG NODE_VERSION=22-alpine
ARG PNPM_VERSION=10.11.0

FROM node:${NODE_VERSION} AS build
WORKDIR /app

# pnpm via corepack, pinned to the version the lockfile expects
RUN corepack enable \
 && corepack prepare pnpm@${PNPM_VERSION} --activate

# Copy the lockfile + manifest first so dependency layer is cacheable
COPY package.json pnpm-lock.yaml .npmrc ./
RUN pnpm install --frozen-lockfile

# Now the rest of the source
COPY . .

# Build = `astro build` (produces dist/server + dist/client for Node adapter)
RUN pnpm run build \
 && pnpm prune --prod

# -----------------------------------------------------------------------------

FROM node:${NODE_VERSION} AS runtime
WORKDIR /app

# Same pnpm version at runtime so we can run `pnpm install --prod` if needed
# (we already pruned the build stage, but having pnpm available is useful for
# `pnpm dlx` debugging in the running container).
RUN corepack enable \
 && corepack prepare pnpm@${PNPM_VERSION} --activate

# Non-root
RUN chown -R node:node /app
USER node

# Bring in the built artefacts and the pruned prod node_modules
COPY --from=build --chown=node:node /app/dist ./dist
COPY --from=build --chown=node:node /app/package.json /app/pnpm-lock.yaml /app/.npmrc ./
COPY --from=build --chown=node:node /app/node_modules ./node_modules

ENV HOST=0.0.0.0 \
    PORT=4321 \
    NODE_ENV=production

EXPOSE 4321

# HEALTHCHECK uses Node's built-in fetch. No curl/wget needed in the image.
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD node -e "fetch('http://127.0.0.1:'+process.env.PORT+'/').then(r=>process.exit(r.ok?0:1)).catch(()=>process.exit(1))"

CMD ["node", "./dist/server/entry.mjs"]
