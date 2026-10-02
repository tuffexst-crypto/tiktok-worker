FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o store-worker .

FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/store-worker /app/store-worker
COPY --from=builder /app/accounts.txt /app/accounts.txt

ENV PORT=8080
ENV WORKER=true
ENV SITE_URL=https://exststocks.org
ENV WORKER_KEY=EXST_Crypto_HMAC_Secret_991283_Uncrackable

EXPOSE 8080

CMD ["/app/store-worker"]
