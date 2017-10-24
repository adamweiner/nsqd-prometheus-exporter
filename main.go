package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus" // Prometheus client library
	logger "github.com/sirupsen/logrus"              // Logging library
	"gopkg.in/urfave/cli.v1"                         // CLI helper
)

var (
	// Version is defined at build time - see VERSION file
	Version string

	scrapeInterval  int
	nsqdURL         string
	knownTopics     []string
	knownChannels   []string
	buildInfoMetric *prometheus.GaugeVec
	nsqMetrics      = make(map[string]*prometheus.GaugeVec)
)

const (
	PrometheusNamespace = "nsqd"
	DepthMetric         = "depth"
	BackendDepthMetric  = "backend_depth"
	InFlightMetric      = "in_flight_count"
	TimeoutCountMetric  = "timeout_count_total"
	RequeueCountMetric  = "requeue_count_total"
	DeferredCountMetric = "deferred_count_total"
	MessageCountMetric  = "message_count_total"
	ClientCountMetric   = "client_count"
	ChannelCountMetric  = "channel_count"
	InfoMetric          = "info"
)

func main() {
	app := cli.NewApp()
	app.Version = Version
	app.Name = "nsqd-prometheus-exporter"
	app.Usage = "Scrapes nsqd stats and serves them up as Prometheus metrics"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "nsqdURL, n",
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
	app.Action = func(c *cli.Context) {
		// Set and validate configuration
		nsqdURL = c.GlobalString("nsqdURL")
		if nsqdURL == "" {
			logger.Warn("Invalid nsqd URL set, continuing with default (http://localhost:4151)")
			nsqdURL = "http://localhost:4151"
		}
		scrapeInterval = c.GlobalInt("scrapeInterval")
		if scrapeInterval < 1 {
			logger.Warn("Invalid scrape interval set, continuing with default (30s)")
			scrapeInterval = 30
		}

		// Initialize Prometheus metrics
		var emptyMap map[string]string
		commonLabels := []string{"type", "topic", "paused", "channel"}
		buildInfoMetric = createGaugeVector("nsqd_prometheus_exporter_build_info", "", "",
			"nsqd-prometheus-exporter build info", emptyMap, []string{"version"})
		buildInfoMetric.WithLabelValues(app.Version).Set(1)
		// # HELP nsqd_info nsqd info
		// # TYPE nsqd_info gauge
		nsqMetrics[InfoMetric] = createGaugeVector(InfoMetric, PrometheusNamespace,
			"", "nsqd info", emptyMap, []string{"health", "start_time", "version"})
		// # HELP nsqd_depth Queue depth
		// # TYPE nsqd_depth gauge
		nsqMetrics[DepthMetric] = createGaugeVector(DepthMetric, PrometheusNamespace,
			"", "Queue depth", emptyMap, commonLabels)
		// # HELP nsqd_backend_depth Queue backend depth
		// # TYPE nsqd_backend_depth gauge
		nsqMetrics[BackendDepthMetric] = createGaugeVector(BackendDepthMetric, PrometheusNamespace,
			"", "Queue backend depth", emptyMap, commonLabels)
		// # HELP nsqd_in_flight_count In flight count
		// # TYPE nsqd_in_flight_count gauge
		nsqMetrics[InFlightMetric] = createGaugeVector(InFlightMetric, PrometheusNamespace,
			"", "In flight count", emptyMap, commonLabels)
		// # HELP nsqd_timeout_count_total Timeout count
		// # TYPE nsqd_timeout_count_total gauge
		nsqMetrics[TimeoutCountMetric] = createGaugeVector(TimeoutCountMetric, PrometheusNamespace,
			"", "Timeout count", emptyMap, commonLabels)
		// # HELP nsqd_requeue_count_total Requeue count
		// # TYPE nsqd_requeue_count_total gauge
		nsqMetrics[RequeueCountMetric] = createGaugeVector(RequeueCountMetric, PrometheusNamespace,
			"", "Requeue count", emptyMap, commonLabels)
		// # HELP nsqd_deferred_count_total Deferred count
		// # TYPE nsqd_deferred_count_total gauge
		nsqMetrics[DeferredCountMetric] = createGaugeVector(DeferredCountMetric, PrometheusNamespace,
			"", "Deferred count", emptyMap, commonLabels)
		// # HELP nsqd_message_count_total Total message count
		// # TYPE nsqd_message_count_total gauge
		nsqMetrics[MessageCountMetric] = createGaugeVector(MessageCountMetric, PrometheusNamespace,
			"", "Total message count", emptyMap, commonLabels)
		// # HELP nsqd_client_count Number of clients
		// # TYPE nsqd_client_count gauge
		nsqMetrics[ClientCountMetric] = createGaugeVector(ClientCountMetric, PrometheusNamespace,
			"", "Number of clients", emptyMap, commonLabels)
		// # HELP nsqd_channel_count Number of channels
		// # TYPE nsqd_channel_count gauge
		nsqMetrics[ChannelCountMetric] = createGaugeVector(ChannelCountMetric, PrometheusNamespace,
			"", "Number of channels", emptyMap, commonLabels[:3])

		go fetchAndSetStats()

		// Start HTTP server
		http.Handle("/metrics", prometheus.Handler())
		err := http.ListenAndServe(":"+strconv.Itoa(c.GlobalInt("listenPort")), nil)
		if err != nil {
			logger.Fatal("Error starting Prometheus metrics server: " + err.Error())
		}
	}

	app.Run(os.Args)
}

// fetchAndSetStats scrapes stats from nsqd and updates all the Prometheus metrics
// above on the provided interval. If a dead topic or channel is detected, the
// application exits.
func fetchAndSetStats() {
	for {
		// Fetch stats
		stats, err := getNsqdStats(nsqdURL)
		if err != nil {
			logger.Fatal("Error scraping stats from nsqd: " + err.Error())
		}

		// Build list of detected topics and channels - the list of channels is built
		// including the topic name that each belongs to, as it is possible to have
		// multiple channels with the same name on different topics.
		var detectedChannels []string
		var detectedTopics []string
		for _, topic := range stats.Topics {
			detectedTopics = append(detectedTopics, topic.Name)
			for _, channel := range topic.Channels {
				detectedChannels = append(detectedChannels, topic.Name+channel.Name)
			}
		}

		// Exit if a dead topic or channel is detected
		if deadTopicOrChannelExists(knownTopics, detectedTopics) {
			logger.Warning("At least one old topic no longer included in nsqd stats - rebuilding metrics")
			for _, metric := range nsqMetrics {
				metric.Reset()
			}
		}
		if deadTopicOrChannelExists(knownChannels, detectedChannels) {
			logger.Warning("At least one old channel no longer included in nsqd stats - rebuilding metrics")
			for _, metric := range nsqMetrics {
				metric.Reset()
			}
		}

		// Update list of known topics and channels
		knownTopics = detectedTopics
		knownChannels = detectedChannels

		// Update info metric with health, start time, and nsqd version
		nsqMetrics[InfoMetric].
			WithLabelValues(stats.Health, fmt.Sprintf("%d", stats.StartTime), stats.Version).Set(1)

		// Loop through topics and set metrics
		for _, topic := range stats.Topics {
			paused := "false"
			if topic.Paused {
				paused = "true"
			}
			nsqMetrics[DepthMetric].WithLabelValues("topic", topic.Name, paused, "").
				Set(float64(topic.Depth))
			nsqMetrics[BackendDepthMetric].WithLabelValues("topic", topic.Name, paused, "").
				Set(float64(topic.BackendDepth))
			nsqMetrics[ChannelCountMetric].WithLabelValues("topic", topic.Name, paused).
				Set(float64(len(topic.Channels)))

			// Loop through a topic's channels and set metrics
			for _, channel := range topic.Channels {
				paused = "false"
				if channel.Paused {
					paused = "true"
				}
				nsqMetrics[DepthMetric].WithLabelValues("channel", topic.Name, paused, channel.Name).
					Set(float64(channel.Depth))
				nsqMetrics[BackendDepthMetric].WithLabelValues("channel", topic.Name, paused, channel.Name).
					Set(float64(channel.BackendDepth))
				nsqMetrics[InFlightMetric].WithLabelValues("channel", topic.Name, paused, channel.Name).
					Set(float64(channel.InFlightCount))
				nsqMetrics[TimeoutCountMetric].WithLabelValues("channel", topic.Name, paused, channel.Name).
					Set(float64(channel.TimeoutCount))
				nsqMetrics[RequeueCountMetric].WithLabelValues("channel", topic.Name, paused, channel.Name).
					Set(float64(channel.RequeueCount))
				nsqMetrics[DeferredCountMetric].WithLabelValues("channel", topic.Name, paused, channel.Name).
					Set(float64(channel.DeferredCount))
				nsqMetrics[MessageCountMetric].WithLabelValues("channel", topic.Name, paused, channel.Name).
					Set(float64(channel.MessageCount))
				nsqMetrics[ClientCountMetric].WithLabelValues("channel", topic.Name, paused, channel.Name).
					Set(float64(len(channel.Clients)))
			}
		}

		// Scrape every scrapeInterval
		time.Sleep(time.Duration(scrapeInterval) * time.Second)
	}
}

// deadTopicOrChannelExists loops through a list of known topic or channel names
// and compares them to a list of detected names. If a known name no longer exists,
// it is deemed dead, and the function returns true.
func deadTopicOrChannelExists(known []string, detected []string) bool {
	// Loop through all known names and check against detected names
	for _, knownName := range known {
		found := false
		for _, detectedName := range detected {
			if knownName == detectedName {
				found = true
				break
			}
		}
		// If a topic/channel isn't found, it's dead
		if !found {
			return true
		}
	}
	return false
}

// createGaugeVector creates a GaugeVec and registers it with Prometheus.
func createGaugeVector(name string, namespace string, subsystem string, help string,
	labels map[string]string, labelNames []string) *prometheus.GaugeVec {
	gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:        name,
		Help:        help,
		Namespace:   namespace,
		Subsystem:   subsystem,
		ConstLabels: prometheus.Labels(labels),
	}, labelNames)
	if err := prometheus.Register(gaugeVec); err != nil {
		logger.Fatal("Failed to register prometheus metric: " + err.Error())
	}
	return gaugeVec
}
