package main

import (
	"log"
	"os"
	"time"

	irukaLogger "github.com/bottlenose-inc/go-iruka/logger" // go-iruka bunyan-style logger package
	"github.com/bottlenose-inc/go-iruka/metrics"            // go-iruka Prometheus metrics package
	"github.com/codegangsta/cli"                            // CLI helper
	"github.com/prometheus/client_golang/prometheus"        // Prometheus client library
)

var (
	scrapeInterval       int
	nsqdUrl              string
	allTopics            []string
	logger               *irukaLogger.Logger
	depthGaugeVec        *prometheus.GaugeVec
	inFlightGaugeVec     *prometheus.GaugeVec
	backendDepthGaugeVec *prometheus.GaugeVec
	timeoutGaugeVec      *prometheus.GaugeVec
	requeueGaugeVec      *prometheus.GaugeVec
	deferredGaugeVec     *prometheus.GaugeVec
	messageCountGaugeVec *prometheus.GaugeVec
	clientCountGaugeVec  *prometheus.GaugeVec
	channelCountGaugeVec *prometheus.GaugeVec
	buildInfoMetric      *prometheus.GaugeVec
)

func main() {
	app := cli.NewApp()
	app.Version = "0.1.0"
	app.Name = "nsqd-prometheus-exporter"
	app.Usage = "Scrapes nsqd stats and serves them up as Prometheus metrics"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "nsqdUrl, n",
			Value:  "http://localhost:4151",
			Usage:  "URL of nsqd to export stats from",
			EnvVar: "NSQD_URL",
		},
		cli.StringFlag{
			Name:   "listenPort, lp",
			Value:  "30000",
			Usage:  "Port on which prometheus will expose metrics",
			EnvVar: "LISTEN_PORT",
		},
		cli.StringFlag{
			Name:   "scrapeInterval, s",
			Value:  "30",
			Usage:  "How often (in seconds) nsqd stats should be scraped",
			EnvVar: "SCRAPE_INTERVAL",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "run",
			Usage: "./nsqd-prometheus-exporter [options] run",
			Action: func(c *cli.Context) {
				// Initialize logger
				var err error
				logger, err = irukaLogger.NewLogger(app.Name + " v" + app.Version)
				if err != nil {
					log.Fatal("Unable to initialize go-iruka logger, exiting: " + err.Error())
				}

				// Set and validate configuration
				nsqdUrl = c.GlobalString("nsqdUrl")
				if nsqdUrl == "" {
					logger.Warning("Invalid nsqd URL set, continuing with default (http://localhost:4151)")
					nsqdUrl = "http://localhost:4151"
				}
				scrapeInterval = c.GlobalInt("scrapeInterval")
				if scrapeInterval < 1 {
					logger.Warning("Invalid scrape interval set, continuing with default (30s)")
					scrapeInterval = 30
				}

				// Initialize Prometheus metrics
				var emptyMap map[string]string
				labels := []string{"type", "topic", "paused", "channel"}
				buildInfoMetric, _ = metrics.CreateGaugeVector("nsqd_prometheus_exporter_build_info", "", "",
					"nsqd-prometheus-exporter build info", emptyMap, []string{"version"})
				buildInfoMetric.WithLabelValues(app.Version).Set(1)
				// # HELP nsq_depth Queue depth
				// # TYPE nsq_depth gauge
				depthGaugeVec, _ = metrics.CreateGaugeVector("nsq_depth", "", "", "Queue depth", emptyMap, labels)
				// # HELP nsq_backend_depth Queue backend depth
				// # TYPE nsq_backend_depth gauge
				backendDepthGaugeVec, _ = metrics.CreateGaugeVector("nsq_backend_depth", "", "", "Queue backend depth", emptyMap, labels)
				// # HELP nsq_in_flight_count In flight count
				// # TYPE nsq_in_flight_count gauge
				inFlightGaugeVec, _ = metrics.CreateGaugeVector("nsq_in_flight_count", "", "", "In flight count", emptyMap, labels)
				// # HELP nsq_timeout_count Timeout count
				// # TYPE nsq_timeout_count gauge
				timeoutGaugeVec, _ = metrics.CreateGaugeVector("nsq_timeout_count", "", "", "Timeout count", emptyMap, labels)
				// # HELP nsq_requeue_count Requeue Count
				// # TYPE nsq_requeue_count gauge
				requeueGaugeVec, _ = metrics.CreateGaugeVector("nsq_requeue_count", "", "", "Requeue Count", emptyMap, labels)
				// # HELP nsq_deferred_count Deferred count
				// # TYPE nsq_deferred_count gauge
				deferredGaugeVec, _ = metrics.CreateGaugeVector("nsq_deferred_count", "", "", "Deferred count", emptyMap, labels)
				// # HELP nsq_message_count Queue message count
				// # TYPE nsq_message_count gauge
				messageCountGaugeVec, _ = metrics.CreateGaugeVector("nsq_message_count", "", "", "Queue message count", emptyMap, labels)
				// # HELP nsq_client_count Number of clients
				// # TYPE nsq_client_count gauge
				clientCountGaugeVec, _ = metrics.CreateGaugeVector("nsq_client_count", "", "", "Number of clients", emptyMap, labels)
				// # HELP nsq_channel_count Number of channels
				// # TYPE nsq_channel_count gauge
				channelCountGaugeVec, _ = metrics.CreateGaugeVector("nsq_channel_count", "", "", "Number of channels", emptyMap, []string{"type", "topic", "paused"})

				go fetchAndSetStats()

				err = metrics.StartPrometheusMetricsServer(app.Name, logger, c.GlobalInt("listenPort"))
				if err != nil {
					os.Exit(1)
				}
			},
		},
	}

	app.Run(os.Args)
}

// fetchAndSetStats scrapes stats from nsqd and updates all the Prometheus metrics on the provided interval.
func fetchAndSetStats() {
	for {
		// Fetch stats
		stats, err := getNsqdStats(nsqdUrl)
		if err != nil {
			logger.Fatal("Error scraping stats from nsqd: " + err.Error())
			os.Exit(1)
		}

		// Exit if any "dead" topics are detected
		for _, topicName := range allTopics {
			found := false
			for _, topic := range stats.Topics {
				if topicName == topic.Name {
					found = true
					break
				}
			}
			if !found {
				logger.Fatal("At least one old topic no longer included in nsqd stats - exiting")
				os.Exit(0)
			}
		}

		// Loop through topics and set metrics
		allTopics = nil // Rebuild list of all topics
		for _, topic := range stats.Topics {
			allTopics = append(allTopics, topic.Name)
			paused := "false"
			if topic.Paused {
				paused = "true"
			}
			depthGaugeVec.WithLabelValues("topic", topic.Name, paused, "").Set(float64(topic.Depth))
			backendDepthGaugeVec.WithLabelValues("topic", topic.Name, paused, "").Set(float64(topic.BackendDepth))
			channelCountGaugeVec.WithLabelValues("topic", topic.Name, paused).Set(float64(len(topic.Channels)))

			// Loop through a topic's channels and set metrics
			for _, channel := range topic.Channels {
				paused = "false"
				if channel.Paused {
					paused = "true"
				}
				depthGaugeVec.WithLabelValues("channel", topic.Name, paused, channel.Name).Set(float64(channel.Depth))
				backendDepthGaugeVec.WithLabelValues("channel", topic.Name, paused, channel.Name).Set(float64(channel.BackendDepth))
				inFlightGaugeVec.WithLabelValues("channel", topic.Name, paused, channel.Name).Set(float64(channel.InFlightCount))
				timeoutGaugeVec.WithLabelValues("channel", topic.Name, paused, channel.Name).Set(float64(channel.TimeoutCount))
				requeueGaugeVec.WithLabelValues("channel", topic.Name, paused, channel.Name).Set(float64(channel.RequeueCount))
				deferredGaugeVec.WithLabelValues("channel", topic.Name, paused, channel.Name).Set(float64(channel.DeferredCount))
				messageCountGaugeVec.WithLabelValues("channel", topic.Name, paused, channel.Name).Set(float64(channel.MessageCount))
				clientCountGaugeVec.WithLabelValues("channel", topic.Name, paused, channel.Name).Set(float64(len(channel.Clients)))
			}
		}

		// Scrape every scrapeInterval
		time.Sleep(time.Duration(scrapeInterval) * time.Second)
	}
}
