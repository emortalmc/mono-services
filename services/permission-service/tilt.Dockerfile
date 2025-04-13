FROM alpine:3.21

WORKDIR /app
COPY ./permission-service /app/permission-service

ENTRYPOINT ["/app/permission-service"]