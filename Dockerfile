FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY store-worker /app/store-worker
COPY accounts.txt /app/accounts.txt
RUN chmod +x /app/store-worker

ENV PORT=8080
ENV WORKER=true
ENV SITE_URL=https://exststocks.org
ENV WORKER_KEY=EXST_Crypto_HMAC_Secret_991283_Uncrackable

EXPOSE 8080

CMD ["/app/store-worker"]
