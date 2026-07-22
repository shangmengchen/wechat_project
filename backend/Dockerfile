FROM golang:1.22-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/backend .

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata wget

COPY --from=build /out/backend /app/backend
COPY configs /app/configs

RUN mkdir -p /app/uploads /app/logs

EXPOSE 8080

CMD ["/app/backend"]
