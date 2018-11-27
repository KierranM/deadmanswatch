package cmd

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

type fakeCloudWatch struct {
	MetricsSent int
}

func (cw *fakeCloudWatch) PutMetricData(input *cloudwatch.PutMetricDataInput) (output *cloudwatch.PutMetricDataOutput, err error) {
	cw.MetricsSent = len(input.MetricData)
	return output, err
}

func TestSourceDimensionForWithSourceLabelPresent(t *testing.T) {
	expectedValue := "test"
	alert := alert{
		Labels: map[string]string{
			"clusterName": expectedValue,
		},
	}
	sut := server{
		cloudwatchClient: &fakeCloudWatch{},
		sourceLabel:      "clusterName",
	}
	actualValue := sut.sourceDimensionFor(alert).Value
	if *actualValue != expectedValue {
		t.Errorf("Expected sourceDimension value to be %s, was %s", expectedValue, *actualValue)
	}
}

func TestSourceDimensionForWithSourceLabelMissing(t *testing.T) {
	expectedValue := "prometheus"
	alert := alert{
		Labels: map[string]string{},
	}
	sut := server{
		cloudwatchClient: &fakeCloudWatch{},
		sourceLabel:      "clusterName",
	}
	actualValue := sut.sourceDimensionFor(alert).Value
	if *actualValue != expectedValue {
		t.Errorf("Expected sourceDimension value to be %s, was %s", expectedValue, *actualValue)
	}
}

func TestSourceDimensionForWithNoSourceLabel(t *testing.T) {
	expectedValue := "prometheus"
	alert := alert{
		Labels: map[string]string{},
	}
	sut := server{
		cloudwatchClient: &fakeCloudWatch{},
		sourceLabel:      "",
	}
	actualValue := sut.sourceDimensionFor(alert).Value
	if *actualValue != expectedValue {
		t.Errorf("Expected sourceDimension value to be %s, was %s", expectedValue, *actualValue)
	}
}

func TestSendMetricsForSingleAlert(t *testing.T) {
	expectedMetrics := 1
	cw := &fakeCloudWatch{}
	sut := server{
		cloudwatchClient: cw,
	}
	payload := webhookPayload{
		Alerts: []alert{
			{
				Labels: map[string]string{
					"alertname": "Test",
				},
			},
		},
	}
	sut.sendMetricsFor(payload)
	if cw.MetricsSent != expectedMetrics {
		t.Errorf("Expected CloudWatch to send PutMetricData with %d metrics, received %d", expectedMetrics, cw.MetricsSent)
	}
}

func TestSendMetricsForNAlerts(t *testing.T) {
	expectedMetrics := 3
	cw := &fakeCloudWatch{}
	sut := server{
		cloudwatchClient: cw,
	}
	payload := webhookPayload{
		Alerts: []alert{
			{
				Labels: map[string]string{
					"alertname": "Test",
				},
			},
			{
				Labels: map[string]string{
					"alertname": "Test2",
				},
			},
			{
				Labels: map[string]string{
					"alertname": "Test3",
				},
			},
		},
	}
	sut.sendMetricsFor(payload)
	if cw.MetricsSent != expectedMetrics {
		t.Errorf("Expected to send PutMetricData with %d metrics, received %d", expectedMetrics, cw.MetricsSent)
	}
}

func TestHeartbeatSendsMetric(t *testing.T) {
	expectedMetrics := 1
	cw := &fakeCloudWatch{}
	sut := server{
		cloudwatchClient: cw,
	}
	sut.heartbeat()
	if cw.MetricsSent != expectedMetrics {
		t.Errorf("Expected heartbeat to send %d metrics, received %d", expectedMetrics, cw.MetricsSent)
	}
}
