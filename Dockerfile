# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/rmn ./cmd/rmn

# ---- runtime stage (static file server) ----
FROM alpine:3.20
RUN adduser -D -u 10001 rmn
COPY --from=build /out/rmn /usr/local/bin/rmn
COPY docs /docs
USER rmn
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/ >/dev/null || exit 1
ENTRYPOINT ["rmn", "serve", "--addr", ":8080", "--web", "/docs"]