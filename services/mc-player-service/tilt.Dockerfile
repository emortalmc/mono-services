FROM alpine:3.21

WORKDIR /app
COPY ./mc-player-service /app/mc-player-service

ENTRYPOINT ["/app/mc-player-service"]