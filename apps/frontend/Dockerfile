# syntax=docker/dockerfile:1
#
# Imagen del frontend Next.js de LangLint (contexto = raíz del monorepo).
# Usa `turbo prune` para reducir el workspace a lo que `frontend` necesita y
# `output: standalone` (next.config.js) para un runtime mínimo.
# NEXT_PUBLIC_API_URL se inyecta en build time (se inlinea en el bundle).

FROM node:22-alpine AS base
RUN corepack enable
WORKDIR /app

# --- prune: subconjunto del monorepo necesario para `frontend` ---
FROM base AS pruner
RUN apk add --no-cache libc6-compat
COPY . .
RUN pnpm dlx turbo@^2 prune frontend --docker

# --- install + build ---
FROM base AS builder
RUN apk add --no-cache libc6-compat
COPY --from=pruner /app/out/json/ ./
RUN pnpm install --frozen-lockfile
COPY --from=pruner /app/out/full/ ./
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
RUN pnpm --filter frontend build

# --- runtime (standalone) ---
FROM base AS runner
ENV NODE_ENV=production
RUN addgroup --system --gid 1001 nodejs \
 && adduser --system --uid 1001 nextjs
COPY --from=builder /app/apps/frontend/.next/standalone ./
COPY --from=builder /app/apps/frontend/.next/static ./apps/frontend/.next/static
COPY --from=builder /app/apps/frontend/public ./apps/frontend/public
USER nextjs
EXPOSE 3000
ENV PORT=3000 HOSTNAME=0.0.0.0
CMD ["node", "apps/frontend/server.js"]
