# Build Vue frontend
FROM node:24-alpine AS web-build
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
FROM golang:1.26-alpine AS api-build
WORKDIR /src
COPY src/api/go.mod src/api/go.sum ./
RUN go mod download
COPY src/api/ .
RUN CGO_ENABLED=0 go build -o /app/ancient-coins-api .

# Final image
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=api-build /app/ancient-coins-api .
COPY --from=web-build /web/dist ./wwwroot
RUN mkdir -p /app/uploads /app/data
VOLUME ["/app/uploads", "/app/data"]
ENV PORT=8080
ENV DB_PATH=/app/data/ancientcoins.db
EXPOSE 8080
ENTRYPOINT ["./ancient-coins-api"]
