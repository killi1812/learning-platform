# Stage 1: Build Vue client application
FROM node:22-alpine AS client-builder
WORKDIR /client

COPY client/package.json client/package-lock.json ./
RUN npm ci

COPY client/ ./
RUN npm run build

# Stage 2: Build Go server binary with embedded client assets
FROM golang:1.25-alpine AS server-builder
WORKDIR /app

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
# Copy compiled client assets into server/web/dist for go:embed
COPY --from=client-builder /client/dist ./web/dist

ARG BUILD=prod
ARG VERSION=0.0.0
ARG COMMIT_HASH=n/a
ARG BUILD_TIMESTAMP=n/a
ARG PACKAGE="git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform"

RUN CGO_ENABLED=0 GOOS=linux go build \
  -ldflags="-X '${PACKAGE}/app.Build=${BUILD}' -X '${PACKAGE}/app.Version=${VERSION}' -X '${PACKAGE}/app.CommitHash=${COMMIT_HASH}' -X '${PACKAGE}/app.BuildTimestamp=${BUILD_TIMESTAMP}'" \
  -o learning-platform main.go

# Stage 3: Lightweight runtime image
FROM alpine:latest
WORKDIR /app

RUN apk add --no-cache ca-certificates
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser:appgroup

COPY --from=server-builder --chown=appuser:appgroup /app/learning-platform .

ENV PORT=8080
ENV MONGO_CONN=mongodb://database:27017

EXPOSE 8080

ENTRYPOINT ["./learning-platform"]

