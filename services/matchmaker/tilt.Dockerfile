FROM alpine:3.21

WORKDIR /app
COPY ./build/matchmaker /app/matchmaker
COPY ./services/matchmaker/run/* /app

ENTRYPOINT ["/app/matchmaker"]