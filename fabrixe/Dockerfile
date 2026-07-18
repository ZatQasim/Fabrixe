# ─────────────────────────────────────────────────────────────────
# Stage 1: Build the React frontend
# ─────────────────────────────────────────────────────────────────
FROM node:20-alpine AS frontend-builder

WORKDIR /build/frontend
COPY frontend/package*.json ./
RUN npm install --legacy-peer-deps
COPY frontend/ ./
RUN npm run build

# ─────────────────────────────────────────────────────────────────
# Stage 2: Build the Go backend
# ─────────────────────────────────────────────────────────────────
FROM golang:1.21-bookworm AS backend-builder

WORKDIR /build

# Dependencies first (layer cache)
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

COPY backend/ ./backend/
COPY --from=frontend-builder /build/dist/web ./dist/web

RUN CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
      -trimpath \
      -ldflags="-s -w -extldflags=-static" \
      -tags sqlite_omit_load_extension \
      -o /build/fabrixe \
      ./backend/cmd/fabrixe

# ─────────────────────────────────────────────────────────────────
# Stage 3: Minimal production image
# ─────────────────────────────────────────────────────────────────
FROM debian:bookworm-slim AS runtime

# Install only what fabrixe needs
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates \
      libsqlite3-0 \
      systemd-libs \
      avahi-daemon \
      libnss-mdns \
    && rm -rf /var/lib/apt/lists/*

# Create fabrixe system user
RUN useradd --system --no-create-home --shell /usr/sbin/nologin fabrixe

# Directories
RUN mkdir -p \
      /opt/fabrixe/bin \
      /opt/fabrixe/web \
      /var/lib/fabrixe \
      /etc/fabrixe/certs \
    && chown -R fabrixe:fabrixe /var/lib/fabrixe /etc/fabrixe

# Copy binary and web assets
COPY --from=backend-builder /build/fabrixe       /opt/fabrixe/bin/fabrixe
COPY --from=frontend-builder /build/dist/web/    /opt/fabrixe/web/
COPY deploy/config.yaml                          /etc/fabrixe/config.yaml

RUN chmod 755 /opt/fabrixe/bin/fabrixe
RUN ln -s /opt/fabrixe/bin/fabrixe /usr/local/bin/fabrixe

# Ports
EXPOSE 8443/tcp
EXPOSE 8080/tcp
EXPOSE 5353/udp

VOLUME ["/var/lib/fabrixe", "/etc/fabrixe"]

# Initialize on first run, then start
ENTRYPOINT ["/opt/fabrixe/bin/fabrixe"]
CMD ["-config", "/etc/fabrixe/config.yaml"]
