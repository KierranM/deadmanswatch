# DeadMansWatch

DeadMansWatch is a tiny tool for forwarding Prometheus DeadMansWatch alerts from AlertManager
to CloudWatch as metrics, these metrics can be used to create CloudWatch alarms to notify you
when Prometheus is down.

It also sends it's own dead mans switch to CloudWatch so that you can alarm when DeadMansWatch is down.

## Usage
To run deadmanswatch, use the `watch` command
```
All software has versions. This is DeadMansWatch's

Usage:
  deadmanswatch watch [flags]

Flags:
      --alert-source-label string           The alert label to use for the 'source' dimension. If unset the 'source' will always be 'prometheus'
      --graceful-timeout duration           Time to wait for the server to gracefully shutdown (default 15s)
      --heartbeat-interval duration         Time between sending metrics for DeadMansWatchs own DeadMansSwitch (default 1m0s)
  -h, --help                                help for watch
  -a, --listen-address ip                   Address to bind to (default 0.0.0.0)
      --log-level string                    The level at which to log. Valid values are debug, info, warn, error (default "info")
      --metric-dimensions stringToString    Dimensions for the metrics in CloudWatch (default [])
      --metric-name string                  metric name for DeadManWatch's own DeadManSwitch metric (default "DeadMansSwitch")
      --metric-namespace string             Metric namespace in CloudWatch (default "DeadMansWatch")
  -p, --port int                            Port to listen on (default 8080)
  -r, --region string                       AWS Region for CloudWatch
```

This will start the deadmanswatch server and listen for connections that match the alertmanager [webhook payload](https://prometheus.io/docs/alerting/configuration/#%3Cwebhook_config%3E)

### AWS Credentials
DeadMansWatch uses the aws sdk for go, which supports the following authentication methods:
- IAM Instance Profile
- Environment variables
- Shared credentials file (`~/.aws/credentials`)

## Deploying
### Service
#### Kubes
The `deploy/kubes` folder contains kubernetes manifests to get DeadMansWatch up and running in kubernetes.

#### Helm
The `deploy/helm` directory contains a helm chart so that you can deploy without having to modify the manifests manually.

### CloudWatch Alarm
The main idea behind this tool is to have CloudWatch alarm when the dead mans switch metric is no longer being received,
you could create such an alarm with terraform like this:
```hcl
resource "aws_cloudwatch_metric_alarm" "deadmanswatch" {
  alarm_name = "deadmansswitch-missing"
  comparison_operator = "LessThanThreshold"
  metric_name = "DeadMansSwitch"
  namespace = "DeadMansWatch"
  evaluation_periods = 3
  treat_missing_data = "breaching"
  threshold = 1
  dimensions {
      source = "prometheus"
  }
  alarm_description = "This alarm fires when prometheus is down in a kubernetes"
  alarm_actions = [] # SNS Arn or something
  ok_actions = [] # SNS ARN or something
}
```

## Contributing

1. Fork it
2. Download your fork to your PC (`git clone https://github.com/your_username/deadmanswatch && cd deadmanswatch`)
3. Create your feature branch (`git checkout -b my-new-feature`)
4. Make changes and add them (`git add .`)
5. Commit your changes (`git commit -m 'Add some feature'`)
6. Push to the branch (`git push origin my-new-feature`)
7. Create new pull request