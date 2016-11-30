# nsqd-prometheus-exporter 

Scrapes nsqd stats and serves them up as Prometheus metrics.

If a previously detected topic or channel no longer exists in a new scrape, the exporter will die, counting on Docker to bring it up again in a fresher state.

## Usage
```
NAME:
   nsqd-prometheus-exporter - Scrapes nsqd stats and serves them up as Prometheus metrics

USAGE:
   nsqd-prometheus-exporter [global options] command [command options] [arguments...]

VERSION:
   0.2.0

COMMANDS:
   run		./nsqd-prometheus-exporter [options] run
   help, h	Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --nsqdUrl, -n 'http://localhost:4151' URL of nsqd to export stats from [$NSQD_URL]
   --listenPort, --lp '30000'            Port on which prometheus will expose metrics [$LISTEN_PORT]
   --scrapeInterval, -s '30'             How often (in seconds) nsqd stats should be scraped [$SCRAPE_INTERVAL]
   --help, -h                            show help
   --version, -v                         print the version
```
