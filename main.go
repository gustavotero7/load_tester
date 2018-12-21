package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

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
		cn := make(chan int)
		for i := 0; i < c.Concurrency; i++ {
			for _, t := range c.Targets {
				go func(v *Target) {
					client := http.Client{
						Timeout: time.Second * time.Duration(c.Timeout),
					}
					req, err := http.NewRequest(v.Method, v.URL, strings.NewReader(v.Payload))
					if err != nil {
						panic(err)
					}

					// Send request
					res, err := client.Do(req)
					v.Result.Total++
					if err != nil {
						v.Result.Failures++
						v.AddResultStatus("Fail: Err")
						fmt.Println("Fail: ", err)

					} else {
						v.AddResultStatus(fmt.Sprintf("%s: %d", http.StatusText(res.StatusCode), res.StatusCode))
						fmt.Println("Success", res.StatusCode)
					}
					cn <- 0
				}(t)

			}
		}
		// Wait for all concurrent calls to finish
		for i := 0; i < c.Concurrency*len(c.Targets); i++ {
			<-cn
		}
	}

	for _, t := range c.Targets {
		logrus.WithFields(logrus.Fields{
			"Total":          t.Result.Total,
			"Failures":       t.Result.Failures,
			"Failure %":      (float64(t.Result.Failures) / float64(t.Result.Total)) * 100,
			"Status Results": t.Result.Status,
		}).Info(t.URL)
	}
}
