FROM ghcr.io/linuxcontainers/alpine:3.20

RUN apk add --no-cache \
    tini

# Copy the binary we built from the builder image
COPY speedtest-exporter /usr/local/bin/speedtest-exporter

# Default to running the `serve` command of our binary
ENTRYPOINT ["/sbin/tini", "-g", "--"]
CMD ["/usr/local/bin/speedtest-exporter"]
