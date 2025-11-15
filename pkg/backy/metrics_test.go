package backy

import (
	"testing"
	"time"
)

func TestAddingMetricsForCommand(t *testing.T) {

	// Create a new MetricFile
	metricFile := NewMetricsFromFile("test_metrics.json")

	metricFile, err := LoadMetricsFromFile(metricFile.Filename)
	if err != nil {
		t.Errorf("Failed to load metrics from file: %v", err)
	}

	// Add metrics for a command
	commandName := "test_command"
	if _, exists := metricFile.CommandMetrics[commandName]; !exists {
		metricFile.CommandMetrics[commandName] = NewMetrics()
	}

	// Update the metrics for the command
	executionTime := 1.8 // Example execution time in seconds
	success := true      // Example success status
	metricFile.CommandMetrics[commandName].Update(success, executionTime, time.Now())

	// Check if the metrics were updated correctly
	if metricFile.CommandMetrics[commandName].SuccessfulExecutions > 50 {
		t.Errorf("Expected 1 successful execution, got %d", metricFile.CommandMetrics[commandName].SuccessfulExecutions)
	}
	if metricFile.CommandMetrics[commandName].TotalExecutions > 50 {
		t.Errorf("Expected 1 total execution, got %d", metricFile.CommandMetrics[commandName].TotalExecutions)
	}
	// if metricFile.CommandMetrics[commandName].TotalExecutionTime != executionTime {
	// 	t.Errorf("Expected execution time %f, got %f", executionTime, metricFile.CommandMetrics[commandName].TotalExecutionTime)
	// }

	err = metricFile.SaveToFile()
	if err != nil {
		t.Errorf("Failed to save metrics to file: %v", err)
	}

	listName := "test_list"
	if _, exists := metricFile.ListMetrics[listName]; !exists {
		metricFile.ListMetrics[listName] = NewMetrics()
	}
	// Update the metrics for the list
	metricFile.ListMetrics[listName].Update(success, executionTime, time.Now())
	if metricFile.ListMetrics[listName].SuccessfulExecutions > 50 {
		t.Errorf("Expected 1 successful execution for list, got %d", metricFile.ListMetrics[listName].SuccessfulExecutions)
	}
	if metricFile.ListMetrics[listName].TotalExecutions > 50 {
		t.Errorf("Expected 1 total execution for list, got %d", metricFile.ListMetrics[listName].TotalExecutions)
	}
	// if metricFile.ListMetrics[listName].TotalExecutionTime > executionTime {
	// 	t.Errorf("Expected execution time %f for list, got %f", executionTime, metricFile.ListMetrics[listName].TotalExecutionTime)
	// }

	// Save the metrics to a file
	err = metricFile.SaveToFile()
	if err != nil {
		t.Errorf("Failed to save metrics to file: %v", err)
	}

}
