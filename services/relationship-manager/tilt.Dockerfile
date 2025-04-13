FROM alpine:3.21

WORKDIR /app
COPY ./build/relationship-manager /app/relationship-manager
COPY ./services/relationship-manager/run/* /app

ENTRYPOINT ["/app/relationship-manager"]