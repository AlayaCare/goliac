######################################
# Prepare npm_builder
######################################
FROM node:16 as npm_builder
WORKDIR /app
ADD . .
RUN make build_ui

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

RUN apt-get update -y && apt-get install ca-certificates -y

COPY --from=go_builder /app/goliac ./goliac
COPY --from=npm_builder /app/browser/goliac-ui/dist ./browser/goliac-ui/dist

RUN useradd --uid 1000 --gid 0 goliac && \
    chown goliac:root /app && \
    chmod g=u /app
USER 1000:0

ENV GOLIAC_SERVER_HOST=0.0.0.0
ENV GOLIAC_SERVER_PORT=18000
EXPOSE 18000

ENTRYPOINT ["./goliac"]
