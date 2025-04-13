FROM alpine:3.21

WORKDIR /app
COPY ./build/mc-player-service /app/mc-player-service
COPY ./services/mc-player-service/run /app/

ENTRYPOINT ["/app/mc-player-service"]