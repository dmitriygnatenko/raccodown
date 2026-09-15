FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY build/app/raccodown /app/raccodown

RUN chmod +x /app/raccodown

EXPOSE 8080

ENTRYPOINT ["/app/raccodown"]