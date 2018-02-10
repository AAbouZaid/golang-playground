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

// QUESTION:
// Is there a better/standard way to handle http connections?
// So no need to write all stuff here from scratch?
func (metrics *Metrics) getJson(flumeUrl string) {

	// QUESTION:
	// If there are 2 urls and both return non 200, 1 only appear as error!
	// You can reproduce it by run the code without start the http server.
	// I don't get why, but it should be something around here
	defer recover()

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

	// QUESTION:
	// Is there a way to call value by name from struct?
	// So no need to create this map?
	filtersMap := map[string][]string{
		"SOURCE":  filters.Source,
		"CHANNEL": filters.Channel,
		"SINK":    filters.Sink,
	}

	for key, value := range metrics[keyName] {
		if field, err := createField(key, value); err == nil {
			fieldsArr = addField(filtersMap, typeName, key, field, fieldsArr)
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
) string {

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

// QUESTION:
// Couldn't find anyway to have "if x in list" like python but writing this.
// If it's not in the language by default, I'd like to use module or so.
func inArray(arr []string, str string) bool {
	for _, elem := range arr {
		if elem == str {
			return true
		}
	}
	return false
}

// QUESTION:
// Flume itself returns all JSON as strings even numerical values,
// and only numerical values are desired, that's why this part is need.
// Any better ideas than what done here?
func createField(key string, value string) (string, error) {
	var fieldName string
	var err error

	// QUESTION:
	// I'm not sure if this will work fine with big numbers.
	if intValue, err := strconv.ParseInt(value, 10, 0); err == nil {
		fieldName = fmt.Sprintf("%s=%v", key, intValue)
	} else if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
		fieldName = fmt.Sprintf("%s=%v", key, floatValue)
	} else {
		return fieldName, err
	}

	return fieldName, err
}

func addField(
	filters map[string][]string,
	typeName string,
	key string,
	field string,
	fieldsArr []string,
) []string {
	typeFiltersLen := len(filters[typeName])
	isTypeFiltered := inArray(filters[typeName], key)
	if (typeFiltersLen > 0 && isTypeFiltered) || typeFiltersLen == 0 {
		fieldsArr = append(fieldsArr, field)
	}
	return fieldsArr
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
