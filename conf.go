package main

import (
	"io/ioutil"

	yml "gopkg.in/yaml.v2"
)

// Conf _
type Conf struct {
	Timeout     int
	Requests    int
	Concurrency int
	Targets     map[string]*Target
}

// Target _
type Target struct {
	URL     string
	Method  string
	Payload string
	Header  map[string]string
	Results Results `yaml:"-"`
}

// AddResultStatus _
func (t *Target) AddResultStatus(status string) {
	if t.Results.Status == nil {
		t.Results.Status = map[string]int{status: 1}
		return
	}
	if _, ok := t.Results.Status[status]; !ok {
		t.Results.Status[status] = 1
		return
	}
	t.Results.Status[status]++
}

// Results _
type Results struct {
	Status    map[string]int
	Failures  int
	Total     int
	TestTime  float64
	AvgTime   float64
	MinTime   float64
	MaxTime   float64
	Responses []Response
}

// ToYAML _
func (c *Conf) ToYAML(file string) error {
	bts, err := yml.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, bts, 777)
}

// FromYAML _
func (c *Conf) FromYAML(file string) error {
	bts, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	return yml.Unmarshal(bts, c)
}
