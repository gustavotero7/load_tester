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
	Targets     []*Target
}

// Target _
type Target struct {
	URL     string
	Method  string
	Payload string
	Header  map[string]string
	Result  Result `yaml:"-"`
}

// AddResultStatus _
func (t *Target) AddResultStatus(status string) {
	if t.Result.Status == nil {
		t.Result.Status = map[string]int{status: 1}
		return
	}
	if _, ok := t.Result.Status[status]; !ok {
		t.Result.Status[status] = 1
		return
	}
	t.Result.Status[status]++
}

// Result _
type Result struct {
	Status   map[string]int
	Failures int
	Total    int
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
