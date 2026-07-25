FROM node:20-alpine AS admin-build

WORKDIR /src/admin-ui

COPY admin-ui/package.json admin-ui/package-lock.json ./
RUN npm ci

COPY admin-ui ./
RUN npm run build

FROM golang:1.22-alpine AS build

WORKDIR /src

ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/backend .

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata wget

COPY --from=build /out/backend /app/backend
COPY configs /app/configs
COPY --from=admin-build /src/web /app/web

RUN mkdir -p /app/uploads /app/logs /app/web/admin

EXPOSE 8080

CMD ["/app/backend"]
