# Build Vue frontend
FROM node:24-alpine@sha256:156b55f92e98ccd5ef49578a8cea0df4679826564bad1c9d4ef04462b9f0ded6 AS web-build
ARG APP_VERSION=dev
ARG BUILD_DATE=""
WORKDIR /web
COPY src/web/package.json src/web/package-lock.json* ./
RUN npm ci
COPY VERSION /VERSION
COPY src/web/ .
ENV VITE_API_BASE_URL=""
ENV VITE_APP_VERSION=${APP_VERSION}
ENV VITE_BUILD_DATE=${BUILD_DATE}
RUN npm run build

# Build Go API
FROM golang:1.26.4-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS api-build
WORKDIR /src
COPY src/api/go.mod src/api/go.sum ./
RUN go mod download
COPY src/api/ .
RUN CGO_ENABLED=0 go build -o /app/ancient-coins-api .

# Final image
FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d
RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app
WORKDIR /app
COPY --from=api-build --chown=app:app /app/ancient-coins-api .
COPY --from=web-build --chown=app:app /web/dist ./wwwroot
RUN mkdir -p /app/uploads /app/data \
    && chown -R app:app /app
VOLUME ["/app/uploads", "/app/data"]
ENV PORT=8080
ENV DB_PATH=/app/data/ancientcoins.db
EXPOSE 8080
USER 10001:10001
ENTRYPOINT ["./ancient-coins-api"]
