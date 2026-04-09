# =============================================================================
# Arkheion Frontend — Multi-stage Dockerfile
# Stage 1: deps      — Node.js dependency installation
# Stage 2: build     — SvelteKit production build
# Stage 3: final     — Minimal Node.js runtime
# =============================================================================

# ---- Stage 1: deps ----------------------------------------------------------
FROM node:20-alpine AS deps

WORKDIR /app
COPY package.json ./
RUN npm install --frozen-lockfile 2>/dev/null || npm install

# ---- Stage 2: development (used by docker-compose.dev.yml) ------------------
FROM node:20-alpine AS development
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
EXPOSE 5173 3000
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]

# ---- Stage 3: build ---------------------------------------------------------
FROM node:20-alpine AS build

WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Build-time environment variables
ARG PUBLIC_API_BASE_URL=/api/v1
ENV PUBLIC_API_BASE_URL=${PUBLIC_API_BASE_URL}

RUN npm run build

# ---- Stage 4: Production runtime --------------------------------------------
FROM node:20-alpine AS final

RUN addgroup -g 1001 -S nodejs && adduser -S sveltekit -u 1001

WORKDIR /app

# Copy only what's needed to run the SvelteKit Node adapter output
COPY --from=build --chown=sveltekit:nodejs /app/build ./build
COPY --from=build --chown=sveltekit:nodejs /app/package.json ./

# Install only production deps for the adapter
RUN npm install --omit=dev --ignore-scripts

USER sveltekit

EXPOSE 3000

ENV PORT=3000
ENV HOST=0.0.0.0

CMD ["node", "build"]
