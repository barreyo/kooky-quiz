
# Build runtime
FROM kooky-base:latest AS build-env

RUN cd services/game_session/src && go build -o /build/game_session

# Post-build minimal image
FROM kooky-service:latest

COPY --from=build-env /build/game_session /service

EXPOSE 50051
EXPOSE 443
EXPOSE 8008

ENTRYPOINT ["./game_session", "--redis-pass", "${REDIS_PASSWORD}"]
