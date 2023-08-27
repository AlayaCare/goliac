######################################
# Prepare go_builder
######################################
FROM golang:1.20-bullseye as go_builder
WORKDIR /app
ADD . .
RUN make build

######################################
# Copy from builder to debian image
######################################
FROM debian:bullseye-slim
RUN mkdir /app
WORKDIR /app

COPY --from=go_builder /app/goliac ./goliac

RUN useradd --uid 1000 --gid 0 goliac && \
    chown goliac:root /app && \
    chmod g=u /app
USER 1000:0

EXPOSE 18000

ENTRYPOINT ["./goliac"]
