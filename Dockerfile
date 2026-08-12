# NOTE: Base image digests should be pinned in CI (e.g. golang:1.25-alpine@sha256:...).
FROM golang:1.25-alpine AS go-builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /build
COPY go.mod go.sum ./
ENV GOTOOLCHAIN=auto
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

RUN CGO_ENABLED=0 GOOS=linux go build -o /orca-server ./cmd/orca-server
RUN CGO_ENABLED=0 GOOS=linux go build -o /orca-cli ./cmd/orca-cli

FROM python:3.12-slim AS python-builder
WORKDIR /app
COPY pyproject.toml ./
COPY orca/ ./orca/
RUN pip install --no-cache-dir -e .

FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libc6 \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -r orca && useradd -r -g orca -d /app -s /sbin/nologin orca

WORKDIR /app

COPY --from=go-builder /orca-server .
COPY --from=go-builder /orca-cli .
COPY --from=python-builder /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages
COPY --from=python-builder /usr/local/bin /usr/local/bin
COPY --from=python-builder /app/orca /app/orca
COPY --from=python-builder /app/pyproject.toml ./
COPY configs/ ./configs/
COPY internal/db/migrations/ ./migrations/

ENV PYTHONUNBUFFERED=1

EXPOSE 8080 9091

HEALTHCHECK --interval=15s --timeout=5s --retries=3 --start-period=10s \
    CMD /app/orca-cli health 2>/dev/null || exit 1

USER orca

ENTRYPOINT ["/app/orca-server"]
