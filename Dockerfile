# syntax=docker/dockerfile:1

ARG GO_VERSION=1.23.5
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build
WORKDIR /src

# Install swag CLI globally
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Download dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod/ \
    go mod download -x

# Copy the source code into the container
COPY . .

# Generate Swagger docs
RUN /go/bin/swag init --output ./docs

# Build the application
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod/ \
    CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /bin/server .

################################################################################

FROM alpine:latest AS final

# Install runtime dependencies
RUN --mount=type=cache,target=/var/cache/apk \
    apk --update add \
    ca-certificates \
    tzdata \
    && \
    update-ca-certificates

# Create a non-privileged user
ARG UID=10001
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    appuser
USER appuser

# Copy the executable and Swagger docs from the build stage
COPY --from=build /bin/server /bin/
COPY --from=build /src/docs /docs

# Expose the port the application listens on
EXPOSE 8080

# Start the application
ENTRYPOINT [ "/bin/server" ]
