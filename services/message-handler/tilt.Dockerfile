FROM alpine:3.21

WORKDIR /app
COPY ./message-handler /app/message-handler

ENTRYPOINT ["/app/message-handler"]