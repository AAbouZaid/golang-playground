package main

import (
	"encoding/json"
	"fmt"
	"github.com/influxdata/toml"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Flume conf.
type Flume struct {
	Name    string   `toml:"name"`
	Servers []string `toml:"servers"`
	Filters Filters
}

type Filters struct {
	Source  []string `toml:"source"`
	Channel []string `toml:"channel"`
	Sink    []string `toml:"sink"`
}

// Metrics.
type Metrics map[string]map[string]string

func (metrics *Metrics) getJson(flumeUrl string) {

	// TODO more connection checks.
	resp, err := http.Get(flumeUrl)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("%s %s", flumeUrl, resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		log.Fatalf("%s", err)
	}
}

func (metrics Metrics) createTags(tagsMap map[string]string) string {
	tagsArr := []string{}

	for key, value := range tagsMap {
		tag := fmt.Sprintf("%s=%s", key, value)
		tagsArr = append(tagsArr, tag)
	}

	tags := strings.Join(tagsArr, ",")
	return tags
}

func (metrics Metrics) createFields(
	filters Filters,
	keyName string,
) string {
	fieldsArr := []string{}
	typeName := strings.SplitN(keyName, ".", 2)[0]

	filtersMap := map[string][]string{
		"SOURCE":  filters.Source,
		"CHANNEL": filters.Channel,
		"SINK":    filters.Sink,
	}

	for key, value := range metrics[keyName] {
		if field, err := createField(key, value); err == nil {
			if len(filtersMap[typeName]) > 0 && inArray(filtersMap[typeName], key) {
				fieldsArr = append(fieldsArr, field)
			} else if len(filtersMap[typeName]) == 0 {
				fieldsArr = append(fieldsArr, field)
			}
		}
	}

	fields := strings.Join(fieldsArr, ",")
	return fields
}

func (metrics Metrics) createMeasurement(
	filters Filters,
	measurementName string,
	keyName string,
	tagsName map[string]string,
) interface{} {

	measurement := "flume_" + measurementName
	tags := metrics.createTags(tagsName)
	fields := metrics.createFields(filters, keyName)
	output := fmt.Sprintf("%s,%s %s", measurement, tags, fields)
	return output
}

func (metrics Metrics) gatherServer(
	serverURL string,
	measurementName string,
	filters Filters,
) {
	metrics.getJson(serverURL)

	for keyName, _ := range metrics {
		keyArr := strings.SplitN(keyName, ".", 2)
		fixedTags := map[string]string{
			"type": keyArr[0],
			"name": keyArr[1],
		}
		output := metrics.createMeasurement(
			filters,
			measurementName,
			keyName,
			fixedTags)
		fmt.Println(output)
	}
}

func inArray(arr []string, str string) bool {
	for _, elem := range arr {
		if elem == str {
			return true
		}
	}
	return false
}

func createField(key string, value string) (string, error) {
	var fieldName string
	var err error

	if intValue, err := strconv.ParseInt(value, 10, 0); err == nil {
		fieldName = fmt.Sprintf("%s=%v", key, intValue)
	} else if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
		fieldName = fmt.Sprintf("%s=%v", key, floatValue)
	} else {
		return fieldName, err
	}

	return fieldName, err
}

var sampleConfig = `
  ## NOTE This plugin only reads numerical measurements, strings and booleans
  ## will be ignored.
  ##
  name = "agents_metrics"
  ## URL of each server in the service's cluster
  servers = [
    "http://localhost:8000/flume01.json",
    "http://localhost:8000/flume02.json",
  ]
  ## Specific metrics could be selected for each type,
  ## instead collecting all metrics as they come from flume.
  [filters]
    channel = [
      "EventPutSuccessCount",
      "EventPutAttemptCount"
    ]
`

// Main.
func main() {

	var flume Flume
	var metrics Metrics
	var wg sync.WaitGroup
	toml.Unmarshal([]byte(sampleConfig), &flume)
	for _, server := range flume.Servers {
		wg.Add(1)
		go func(server string) {
			defer wg.Done()
			metrics.gatherServer(server, flume.Name, flume.Filters)
		}(server)
	}
	wg.Wait()
}