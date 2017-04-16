# nsqd-prometheus-exporter

Scrapes nsqd stats and serves them up as Prometheus metrics.

If a previously detected topic or channel no longer exists in a new scrape, the exporter will rebuild all metrics to remove any label values associated with the old topic or channel.

TODO: add tests

## Usage
```
NAME:
   nsqd-prometheus-exporter - Scrapes nsqd stats and serves them up as Prometheus metrics

USAGE:
   nsqd-prometheus-exporter [global options] command [command options] [arguments...]

VERSION:
   0.4.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --nsqdURL value, -n value         URL of nsqd to export stats from (default: "http://localhost:4151") [$NSQD_URL]
   --listenPort value, --lp value    Port on which prometheus will expose metrics (default: "30000") [$LISTEN_PORT]
   --scrapeInterval value, -s value  How often (in seconds) nsqd stats should be scraped (default: "30") [$SCRAPE_INTERVAL]
   --help, -h                        show help
   --version, -v                     print the version
```
