FROM alpine:3.21

WORKDIR /app
COPY ./build/game-tracker /app/game-tracker
COPY ./services/game-tracker/run/* /app

ENTRYPOINT ["/app/game-tracker"]