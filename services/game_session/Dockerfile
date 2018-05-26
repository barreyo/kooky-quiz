
# Build runtime
FROM kooky-base:latest AS build-env

RUN cd services/game_session && go build -o /build/game_session

# Post-build minimal image
FROM kooky-service:latest

COPY --from=build-env /build/game_session /service

EXPOSE 3000

ENTRYPOINT ["./game_session"]
