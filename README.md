# Speed Test Exporter

This service will run ping and speed tests agains the speedtest.net servers at a set interval, and export the results in prometheus metric format.

The binary can be built cross-platform as a standalone binary, or this repo also publishes docker containers as part of its release pipeline.

See: https://github.com/RichieSams/speedtest-exporter/releases

```bash
docker pull ghcr.io/richiesams/speedtest-exporter:<version>
```

Example:

```bash
docker run --rm \
    -p 8080:8080
    ghcr.io/richiesams/speedtest-exporter:0.1.0 \
    /usr/local/bin/speedtest-exporter \
    --port 8080 \
    --test-interval 30s
```

Then browse to `http://localhost:8080/metrics` and you will see the exported metrics. (It may take a few seconds for the first speed test to run).

## Config

```bash
--port <int>
```

The port to expose the metrics and health endpoints on

```bash
--test-interval <duration>
```

How often the ping and speed tests should be run. The tests run continuously in the service, the `/metrics` endpoint merely exposes the last values of the test

```bash
--log-level <debug|info|warn|error>
```

The log level to use

## Metrics

```text
speedtest_exporter_test_count
```

This is a monotonic counter that is incremented for each iteration. If this stops incrementing, it indicates the service is hung in some way. If this increments slower than the interval, the ping/speed tests are taking longer than the interval.

```text
speedtest_exporter_test_status
```

This will be set to `1` if an iteration ran without any errors. It will be set to `0` if any errors occurred.

```text
speedtest_exporter_latency
```

The ping latency (in seconds) from the client to the server

```text
speedtest_exporter_upload_speed
```

The upload speed (in bytes/second) to the server

```text
speedtest_exporter_download_speed
```

The download speed (in bytes/second) from the server
