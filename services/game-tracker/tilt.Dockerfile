FROM alpine:3.21

WORKDIR /app
COPY ./game-tracker /app/game-tracker

ENTRYPOINT ["/app/game-tracker"]