package backy

import (
	"encoding/json"
	"os"
)

type Metrics struct {
	SuccessfulExecutions uint64  `json:"successful_executions"`
	FailedExecutions     uint64  `json:"failed_executions"`
	TotalExecutions      uint64  `json:"total_executions"`
	ExecutionTime        float64 `json:"execution_time"`         // in seconds
	AverageExecutionTime float64 `json:"average_execution_time"` // in seconds
	SuccessRate          float64 `json:"success_rate"`           // percentage of successful executions
	FailureRate          float64 `json:"failure_rate"`           // percentage of failed executions
}

func NewMetrics() *Metrics {
	return &Metrics{
		SuccessfulExecutions: 0,
		FailedExecutions:     0,
		TotalExecutions:      0,
		ExecutionTime:        0.0,
		AverageExecutionTime: 0.0,
		SuccessRate:          0.0,
		FailureRate:          0.0,
	}
}

func (m *Metrics) Update(success bool, executionTime float64) {
	m.TotalExecutions++
	if success {
		m.SuccessfulExecutions++
	} else {
		m.FailedExecutions++
	}

	m.ExecutionTime += executionTime
	m.AverageExecutionTime = m.ExecutionTime / float64(m.TotalExecutions)

	if m.TotalExecutions > 0 {
		m.SuccessRate = float64(m.SuccessfulExecutions) / float64(m.TotalExecutions) * 100
		m.FailureRate = float64(m.FailedExecutions) / float64(m.TotalExecutions) * 100
	}
}

func SaveToFile(metrics *Metrics, filename string) error {
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func LoadFromFile(filename string) (*Metrics, error) {
	return nil, nil
}
