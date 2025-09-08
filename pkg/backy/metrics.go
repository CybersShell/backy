package backy

import (
	"encoding/json"
	"os"
	"time"
)

type MetricFile struct {
	Filename       string              `json:"filename"`
	CommandMetrics map[string]*Metrics `json:"commandMetrics"`
	ListMetrics    map[string]*Metrics `json:"listMetrics"`
}

type Metrics struct {
	DateStartedLast              string  `json:"dateStartedLast"`
	DateLastFinished             string  `json:"dateLastFinished"`
	DateLastFinishedSuccessfully string  `json:"dateLastFinishedSuccessfully"`
	SuccessfulExecutions         uint64  `json:"successfulExecutions"`
	FailedExecutions             uint64  `json:"failedExecutions"`
	TotalExecutions              uint64  `json:"totalExecutions"`
	TotalExecutionTime           float64 `json:"lastExecutionTime"`  // in seconds
	AverageExecutionTime         float64 `json:"totalExecutionTime"` // in seconds
	SuccessRate                  float64 `json:"successRate"`        // percentage of successful executions
	FailureRate                  float64 `json:"failureRate"`        // percentage of failed executions
}

func NewMetrics() *Metrics {
	return &Metrics{
		DateStartedLast:      time.Now().Format(time.RFC3339),
		SuccessfulExecutions: 0,
		FailedExecutions:     0,
		TotalExecutions:      0,
		TotalExecutionTime:   0.0,
		AverageExecutionTime: 0.0,
		SuccessRate:          0.0,
		FailureRate:          0.0,
	}
}

func NewMetricsFromFile(filename string) *MetricFile {
	return &MetricFile{
		Filename:       filename,
		CommandMetrics: make(map[string]*Metrics),
		ListMetrics:    make(map[string]*Metrics),
	}

}

func (m *Metrics) Update(success bool, executionTime float64, dateLastFinished time.Time) {
	m.TotalExecutions++
	if success {
		m.SuccessfulExecutions++
	} else {
		m.FailedExecutions++
	}

	m.DateLastFinished = dateLastFinished.Format(time.RFC3339)

	m.TotalExecutionTime += executionTime
	m.AverageExecutionTime = m.TotalExecutionTime / float64(m.TotalExecutions)

	if m.TotalExecutions > 0 {
		m.SuccessRate = float64(m.SuccessfulExecutions) / float64(m.TotalExecutions) * 100
		m.FailureRate = float64(m.FailedExecutions) / float64(m.TotalExecutions) * 100
	}
}

func (metricFile *MetricFile) SaveToFile() error {
	data, err := json.MarshalIndent(metricFile, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metricFile.Filename, data, 0644)
}

func LoadMetricsFromFile(filename string) (*MetricFile, error) {
	jsonData, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var metrics MetricFile
	err = json.Unmarshal(jsonData, &metrics)
	if err != nil {
		return nil, err
	}
	return &metrics, nil
}
