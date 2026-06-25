# ==========================================
#  Deps stage — install production & dev deps
# ==========================================
FROM node:22-alpine AS deps

WORKDIR /app

# Pin npm to the lockfile version.
COPY package.json package-lock.json ./
RUN npm ci --frozen-lockfile

# ==========================================
#  Builder stage — compile Next.js
# ==========================================
FROM node:22-alpine AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

# NEXT_PUBLIC_* vars are inlined at build time; pass them through build args.
ARG NEXT_PUBLIC_API_URL=http://localhost:8080/api
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

RUN npm run build

# ==========================================
#  Runner stage — minimal production image
# ==========================================
FROM node:22-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
ENV HOSTNAME=0.0.0.0
ENV PORT=3000

# Non-root user.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Standalone output: a self-contained Next.js server.
COPY --from=builder --chown=appuser:appgroup /app/.next/standalone ./
# Public static assets (served from /public).
COPY --from=builder --chown=appuser:appgroup /app/public ./public
# Next.js build-time static chunks (images, JS, CSS).
COPY --from=builder --chown=appuser:appgroup /app/.next/static ./.next/static

USER appuser
EXPOSE 3000

CMD ["node", "server.js"]
