FROM alpine:3.21

WORKDIR /app
COPY ./build/permission-service /app/permission-service
COPY ./services/permission-service/run/* /app

ENTRYPOINT ["/app/permission-service"]