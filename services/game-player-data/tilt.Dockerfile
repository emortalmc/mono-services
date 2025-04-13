FROM alpine:3.21

WORKDIR /app
COPY ./game-player-data /app/game-player-data

ENTRYPOINT ["/app/game-player-data"]