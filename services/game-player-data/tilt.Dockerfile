FROM alpine:3.21

WORKDIR /app
COPY ./build/game-player-data /app/game-player-data
COPY ./services/game-player-data/run/* /app

ENTRYPOINT ["/app/game-player-data"]