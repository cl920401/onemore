ARG GOPROXY=https://proxy.golang.org,direct

FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.27-alpine AS build
ARG GOPROXY
ENV GOPROXY=${GOPROXY}
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -o /out/onemore ./cmd/server

FROM alpine:3.20
RUN adduser -D -u 10001 onemore
WORKDIR /app
COPY --from=build /out/onemore /usr/local/bin/onemore
COPY --from=web /app/web/dist ./web/dist
USER onemore
ENV ONEMORE_ADDR=:8080 \
    ONEMORE_WEB_DIR=/app/web/dist
EXPOSE 8080
ENTRYPOINT ["onemore"]
