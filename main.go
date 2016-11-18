package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus" // Prometheus client library
	"github.com/urfave/cli"                          // CLI helper
)

var (
	scrapeInterval       int
	nsqdUrl              string
	knownTopics          []string
	knownChannels        []string
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
	app.Version = "0.2.0"
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
				// Set and validate configuration
				nsqdUrl = c.GlobalString("nsqdUrl")
				if nsqdUrl == "" {
					log.Println("Invalid nsqd URL set, continuing with default (http://localhost:4151)")
					nsqdUrl = "http://localhost:4151"
				}
				scrapeInterval = c.GlobalInt("scrapeInterval")
				if scrapeInterval < 1 {
					log.Println("Invalid scrape interval set, continuing with default (30s)")
					scrapeInterval = 30
				}

				// Initialize Prometheus metrics
				var emptyMap map[string]string
				commonLabels := []string{"type", "topic", "paused", "channel"}
				buildInfoMetric = createGaugeVector("nsqd_prometheus_exporter_build_info", "", "",
					"nsqd-prometheus-exporter build info", emptyMap, []string{"version"})
				buildInfoMetric.WithLabelValues(app.Version).Set(1)
				// # HELP nsq_depth Queue depth
				// # TYPE nsq_depth gauge
				depthGaugeVec = createGaugeVector("nsq_depth", "", "", "Queue depth", emptyMap, commonLabels)
				// # HELP nsq_backend_depth Queue backend depth
				// # TYPE nsq_backend_depth gauge
				backendDepthGaugeVec = createGaugeVector("nsq_backend_depth", "", "", "Queue backend depth", emptyMap, commonLabels)
				// # HELP nsq_in_flight_count In flight count
				// # TYPE nsq_in_flight_count gauge
				inFlightGaugeVec = createGaugeVector("nsq_in_flight_count", "", "", "In flight count", emptyMap, commonLabels)
				// # HELP nsq_timeout_count Timeout count
				// # TYPE nsq_timeout_count gauge
				timeoutGaugeVec = createGaugeVector("nsq_timeout_count", "", "", "Timeout count", emptyMap, commonLabels)
				// # HELP nsq_requeue_count Requeue Count
				// # TYPE nsq_requeue_count gauge
				requeueGaugeVec = createGaugeVector("nsq_requeue_count", "", "", "Requeue Count", emptyMap, commonLabels)
				// # HELP nsq_deferred_count Deferred count
				// # TYPE nsq_deferred_count gauge
				deferredGaugeVec = createGaugeVector("nsq_deferred_count", "", "", "Deferred count", emptyMap, commonLabels)
				// # HELP nsq_message_count Queue message count
				// # TYPE nsq_message_count gauge
				messageCountGaugeVec = createGaugeVector("nsq_message_count", "", "", "Queue message count", emptyMap, commonLabels)
				// # HELP nsq_client_count Number of clients
				// # TYPE nsq_client_count gauge
				clientCountGaugeVec = createGaugeVector("nsq_client_count", "", "", "Number of clients", emptyMap, commonLabels)
				// # HELP nsq_channel_count Number of channels
				// # TYPE nsq_channel_count gauge
				channelCountGaugeVec = createGaugeVector("nsq_channel_count", "", "", "Number of channels", emptyMap, commonLabels[:3])

				go fetchAndSetStats()

				// Start HTTP server
				http.Handle("/metrics", prometheus.Handler())
				err := http.ListenAndServe(":"+strconv.Itoa(c.GlobalInt("listenPort")), nil)
				if err != nil {
					log.Fatal("Error starting Prometheus metrics server: " + err.Error())
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
			log.Fatal("Error scraping stats from nsqd: " + err.Error())
		}

		// Build list of detected topics and channels - the list of channels is built including the topic name that
		// each belongs to, as it is possible to have multiple channels with the same name on different topics.
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
			log.Fatal("At least one old topic no longer included in nsqd stats - exiting")
		}
		if deadTopicOrChannelExists(knownChannels, detectedChannels) {
			log.Fatal("At least one old channel no longer included in nsqd stats - exiting")
		}

		// Update list of known topics and channels
		knownTopics = detectedTopics
		knownChannels = detectedChannels

		// Loop through topics and set metrics
		for _, topic := range stats.Topics {
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

// deadTopicOrChannelExists loops through a list of known topic or channel names and compares them to a list
// of detected names. If a known name no longer exists, it is deemed dead.
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
	prometheus.MustRegister(gaugeVec)
	return gaugeVec
}
