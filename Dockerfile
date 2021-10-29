FROM alpine:latest

COPY server_worker /usr/bin/
COPY server_controller /usr/bin/
