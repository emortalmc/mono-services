FROM alpine:3.21

WORKDIR /app
COPY ./matchmaker /app/matchmaker

ENTRYPOINT ["/app/matchmaker"]