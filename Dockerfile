# Stage 1: compile
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod .
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w" -o /out/wait-for-health-check

# Stage 2: runtime — empty base, just the binary
FROM scratch
COPY --from=build /out/wait-for-health-check /wait-for-health-check

# Container stays alive; Docker HEALTHCHECK runs `/wait-for-health-check probe` separately.
HEALTHCHECK --interval=2s --timeout=2s --retries=60 \
    CMD ["/wait-for-health-check", "probe"]

ENTRYPOINT ["/wait-for-health-check", "hold"]
