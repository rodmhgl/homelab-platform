# Build stage
FROM node:22-alpine AS builder

WORKDIR /build

# Install dependencies (cached layer)
COPY package.json package-lock.json ./
RUN npm ci

# Copy source and build
COPY . .

# Build-time environment variables
ARG VITE_API_URL=
ARG BUILD_VERSION=dev
ARG BUILD_COMMIT=unknown
ARG BUILD_DATE=unknown

ENV VITE_API_URL=${VITE_API_URL}

# Add version info as a file for runtime access
RUN echo "{\"version\":\"${BUILD_VERSION}\",\"commit\":\"${BUILD_COMMIT}\",\"buildDate\":\"${BUILD_DATE}\"}" > version.json

RUN npm run build

# Copy version info to dist
RUN cp version.json dist/

# Runtime stage
FROM nginx:1.27-alpine

# Create non-root user
RUN addgroup -g 1000 portal && \
    adduser -D -u 1000 -G portal portal && \
    chown -R portal:portal /usr/share/nginx/html /var/cache/nginx /var/run && \
    chmod -R 755 /usr/share/nginx/html

# Copy built assets
COPY --from=builder --chown=portal:portal /build/dist /usr/share/nginx/html

# Copy Nginx config
COPY --chown=portal:portal nginx/default.conf /etc/nginx/conf.d/default.conf

# Nginx PID file → /tmp (read-only rootfs compatible)
# RUN sed -i 's|pid /var/run/nginx.pid;|pid /tmp/nginx.pid;|' /etc/nginx/nginx.conf
RUN sed -i 's|pid .*/nginx.pid;|pid /tmp/nginx.pid;|' /etc/nginx/nginx.conf

USER portal
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

CMD ["nginx", "-g", "daemon off;"]
