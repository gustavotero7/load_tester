package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Result _
type Result struct {
	Key  string
	Err  error
	Code int
}

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{
		PrettyPrint:      true,
		DisableTimestamp: true,
		FieldMap: logrus.FieldMap{
			"msg": "Target",
		},
	})
	// Load config data
	c := &Conf{}
	if err := c.FromYAML("conf.yml"); err != nil {
		panic(err)
	}

	// Run while all requests are done
	for r := c.Requests; r > 0; r -= c.Concurrency {
		// Run concurrent tests
		cn := make(chan Result)
		for i := 0; i < c.Concurrency; i++ {
			for k, t := range c.Targets {
				go func(key string, v Target) {
					took := time.Now()
					client := http.Client{
						Timeout: time.Second * time.Duration(c.Timeout),
					}
					req, err := http.NewRequest(v.Method, v.URL, strings.NewReader(v.Payload))
					if err != nil {
						panic(err)
					}

					for hk, hv := range v.Header {
						req.Header.Set(hk, hv)
					}

					rs := Result{
						Key: key,
					}

					// Send request
					res, err := client.Do(req)
					if err != nil {
						rs.Err = err
						fmt.Println("Fail: ", err)

					} else {
						rs.Code = res.StatusCode
						fmt.Println("Call to", v.URL)
						fmt.Println("Success", res.StatusCode, "/ Took:", time.Since(took).Seconds(), "Seconds")
					}
					cn <- rs
				}(k, *t)
			}
		}
		// Wait for all concurrent calls to finish
		for i := 0; i < c.Concurrency*len(c.Targets); i++ {
			// Workarround to avoid data race
			// We fill results for each target after the test is done to avoid multiple writes to the same results object
			rs := <-cn
			c.Targets[rs.Key].Results.Total++
			if rs.Err != nil {
				c.Targets[rs.Key].Results.Failures++
				c.Targets[rs.Key].AddResultStatus("Fail: Err")

			} else {
				c.Targets[rs.Key].AddResultStatus(fmt.Sprintf("%s: %d", http.StatusText(rs.Code), rs.Code))
			}
		}
	}

	for _, t := range c.Targets {
		logrus.WithFields(logrus.Fields{
			"Total":          t.Results.Total,
			"Failures":       t.Results.Failures,
			"Failure %":      (float64(t.Results.Failures) / float64(t.Results.Total)) * 100,
			"Status Results": t.Results.Status,
		}).Info(t.URL)
	}
}
