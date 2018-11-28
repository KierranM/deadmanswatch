package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	logLevel          string
	region            string
	port              int
	address           net.IP
	timeout           time.Duration
	metricDimensions  map[string]string
	sourceLabel       string
	metricNamespace   string
	metricName        string
	heartbeatInterval time.Duration
)

const (
	pingMessage string = "PONG\n"
)

type webhookPayload struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []alert           `json:"alerts"`
}

type alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

type cloudwatchClient interface {
	PutMetricData(*cloudwatch.PutMetricDataInput) (*cloudwatch.PutMetricDataOutput, error)
}

type server struct {
	cloudwatchClient    cloudwatchClient
	port                int
	address             net.IP
	timeout             time.Duration
	sourceLabel         string
	metricNamespace     string
	metricName          string
	heartbeatInterval   time.Duration
	dmwDimensions       []*cloudwatch.Dimension
	forwardedDimensions []*cloudwatch.Dimension
}

func init() {
	watchCmd.Flags().StringVar(&logLevel, "log-level", "info", "The level at which to log. Valid values are debug, info, warn, error")
	watchCmd.Flags().StringVarP(&region, "region", "r", os.Getenv("AWS_REGION"), "AWS Region for CloudWatch")
	watchCmd.Flags().IPVarP(&address, "listen-address", "a", net.IPv4zero, "Address to bind to")
	watchCmd.Flags().StringVar(&metricNamespace, "metric-namespace", "DeadMansWatch", "Metric namespace in CloudWatch")
	watchCmd.Flags().StringVar(&metricName, "metric-name", "DeadMansSwitch", "metric name for DeadManWatch's own DeadManSwitch metric")
	watchCmd.Flags().StringToStringVar(&metricDimensions, "metric-dimensions", make(map[string]string), "Dimensions for the metrics in CloudWatch")
	watchCmd.Flags().StringVar(&sourceLabel, "alert-source-label", "", "The alert label to use for the 'source' dimension. If unset the 'source' will always be 'prometheus'")
	watchCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	watchCmd.Flags().DurationVar(&heartbeatInterval, "heartbeat-interval", time.Second*60, "Time between sending metrics for DeadMansWatchs own DeadMansSwitch")
	watchCmd.Flags().DurationVar(&timeout, "graceful-timeout", time.Second*15, "Time to wait for the server to gracefully shutdown")
	rootCmd.AddCommand(watchCmd)
	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
		DisableColors: true,
	}
	logrus.SetFormatter(formatter)
}

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Print the version of deadmanswatch",
	Long:  `All software has versions. This is DeadMansWatch's`,
	Run: func(cmd *cobra.Command, args []string) {
		parsedLevel, err := logrus.ParseLevel(logLevel)
		if err != nil {
			logrus.Errorf("Unable to parse log level (will use default): %v", err)
		} else {
			logrus.SetLevel(parsedLevel)
		}
		if region == "" {
			logrus.Fatal("Please specify an aws region by setting --region or with the AWS_REGION environment variable")
		} else {
			logrus.Infof("Will send metrics to CloudWatch in %s", region)
			sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))
			cloudWatch := cloudwatch.New(sess)
			srv := newServer(cloudWatch, port, address, timeout, metricDimensions, sourceLabel, metricNamespace, metricName, heartbeatInterval)
			srv.startServer()
		}
	},
}

func newServer(cloudwatchClient cloudwatchClient, port int, address net.IP, timeout time.Duration, metricDimensions map[string]string, sourceLabel string, metricNamespace string, metricName string, heartbeatInterval time.Duration) *server {
	// Build the lists of dimensions once
	baseDimensions := make([]*cloudwatch.Dimension, 0, len(metricDimensions))
	for k, v := range metricDimensions {
		baseDimensions = append(baseDimensions, &cloudwatch.Dimension{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}
	dmwDimensions := append(baseDimensions, &cloudwatch.Dimension{
		Name:  aws.String("source"),
		Value: aws.String("DeadMansWatch"),
	})
	return &server{
		cloudwatchClient:    cloudwatchClient,
		port:                port,
		address:             address,
		timeout:             timeout,
		sourceLabel:         sourceLabel,
		metricNamespace:     metricNamespace,
		metricName:          metricName,
		heartbeatInterval:   heartbeatInterval,
		dmwDimensions:       dmwDimensions,
		forwardedDimensions: baseDimensions,
	}
}

func (s *server) heartbeat() {
	logrus.Debug("Sending Heartbeat")
	_, err := s.cloudwatchClient.PutMetricData(&cloudwatch.PutMetricDataInput{
		MetricData: []*cloudwatch.MetricDatum{
			{
				Value:      aws.Float64(1),
				Dimensions: s.dmwDimensions,
				MetricName: aws.String(metricName),
			},
		},
		Namespace: aws.String(metricNamespace),
	})
	if err != nil {
		logrus.Errorf("Failed to send heartbeat %v", err)
	} else {
		logrus.Debug("Heartbeat sent")
	}
}

func (s *server) sourceDimensionFor(alert alert) *cloudwatch.Dimension {
	var sourceDimension *cloudwatch.Dimension
	if s.sourceLabel == "" || alert.Labels[s.sourceLabel] == "" {
		sourceDimension = &cloudwatch.Dimension{
			Name:  aws.String("source"),
			Value: aws.String("prometheus"),
		}
	} else {
		sourceDimension = &cloudwatch.Dimension{
			Name:  aws.String("source"),
			Value: aws.String(alert.Labels[s.sourceLabel]),
		}
	}
	return sourceDimension
}

func (s *server) sendMetricsFor(payload webhookPayload) {
	metricDatum := make([]*cloudwatch.MetricDatum, 0, len(payload.Alerts))
	for _, alert := range payload.Alerts {

		metricDatum = append(metricDatum, &cloudwatch.MetricDatum{
			Value:      aws.Float64(1),
			Dimensions: append(s.forwardedDimensions, s.sourceDimensionFor(alert)),
			MetricName: aws.String(alert.Labels["alertname"]),
		})
	}
	_, err := s.cloudwatchClient.PutMetricData(&cloudwatch.PutMetricDataInput{
		MetricData: metricDatum,
		Namespace:  aws.String(metricNamespace),
	})
	if err != nil {
		logrus.Errorf("Failed to put metric to CloudWatch %v", err)
	} else {
		logrus.Debugf("%d metrics published to CloudWatch", len(metricDatum))
	}
}

func (s *server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		logrus.Printf("%s %s %s %s", r.Method, r.RequestURI, r.RemoteAddr, r.UserAgent())
	})
}

func (s *server) pingHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, pingMessage)
}

func (s *server) switchHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var payload webhookPayload
	err := decoder.Decode(&payload)
	if err != nil {
		logrus.Errorf("Failed to decode a payload: %v", err)
	} else {
		s.sendMetricsFor(payload)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *server) startServer() {
	logrus.Infof("Starting server on %s:%d", address, port)
	rtr := mux.NewRouter()
	rtr.HandleFunc("/ping", s.pingHandler).Methods("GET", "HEAD")
	rtr.HandleFunc("/alert", s.switchHandler).Methods("POST")
	rtr.Use(s.loggingMiddleware)
	srv := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", address, port),
		Handler:        rtr,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logrus.Fatal(err)
		}
	}()

	// Start sending the DeadMansSwitch for this application
	ticker := time.NewTicker(heartbeatInterval)
	go func() {
		for range ticker.C {
			s.heartbeat()
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c
	logrus.Info("Server shutting down")
	ticker.Stop()

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	err := srv.Shutdown(ctx)
	if err != nil {
		logrus.Warnf("Failed to shutdown server gracefully: %v", err)
	}
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	logrus.Info("Goodbye")
	os.Exit(0)
}
