
FROM redis:alpine

COPY redis.conf /usr/local/etc/redis/redis.conf

CMD ["redis-server", "--port", "${REDIS_PORT}", "--requirepass", "${REDIS_PASSWORD}"]
