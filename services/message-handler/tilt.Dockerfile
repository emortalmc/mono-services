FROM alpine:3.21

WORKDIR /app
COPY ./build/message-handler /app/message-handler
COPY ./services/message-handler/run/* /app

ENTRYPOINT ["/app/message-handler"]