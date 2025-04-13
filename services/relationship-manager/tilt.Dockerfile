FROM alpine:3.21

WORKDIR /app
COPY ./relationship-manager /app/relationship-manager

ENTRYPOINT ["/app/relationship-manager"]