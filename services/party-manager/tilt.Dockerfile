FROM alpine:3.21

WORKDIR /app
COPY ./party-manager /app/party-manager

ENTRYPOINT ["/app/party-manager"]