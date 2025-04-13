FROM alpine:3.21

WORKDIR /app
COPY ./build/party-manager /app/party-manager
COPY ./services/party-manager/run/* /app

ENTRYPOINT ["/app/party-manager"]