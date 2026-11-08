FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY bin/api-linux-amd64 /app/traigalo-api
RUN chmod +x /app/traigalo-api

EXPOSE 8080

ENTRYPOINT ["/app/traigalo-api"]
