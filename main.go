package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	tm "github.com/buger/goterm"
	"github.com/sirupsen/logrus"
)

// Result _
type Result struct {
	Key  string
	Err  error
	Code int
}

func main() {
	cPath := flag.String("c", "conf.yml", "Config file")
	flag.Parse()
	logrus.SetFormatter(&logrus.JSONFormatter{
		PrettyPrint:      true,
		DisableTimestamp: true,
		FieldMap: logrus.FieldMap{
			"msg": "Target",
		},
	})
	// Load config data
	c := &Conf{}
	if err := c.FromYAML(*cPath); err != nil {
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
						if err.(*url.Error).Timeout() {
							rs.Err = errors.New("Timeout")
						} else {
							rs.Err = errors.New(strings.Replace(err.Error(), v.URL, "", -1))
						}
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
				c.Targets[rs.Key].AddResultStatus(rs.Err.Error())

			} else {
				c.Targets[rs.Key].AddResultStatus(fmt.Sprintf("%d : %s", rs.Code, http.StatusText(rs.Code)))
			}
		}
	}

	//return
	header := map[string]struct{}{
		"Test":  struct{}{},
		"Total": struct{}{},
	}
	rows := []map[string]string{}
	for tk, t := range c.Targets {
		row := map[string]string{}
		row["Test"] = tk
		row["Total"] = fmt.Sprintf("%v", t.Results.Total)
		for k, v := range t.Results.Status {
			// Save all possible results to use them as headers
			header[k] = struct{}{}
			p := (float64(v) / float64(t.Results.Total)) * 100
			row[k] = fmt.Sprintf(`%d [%%%.2f]`, v, p)
		}
		rows = append(rows, row)
	}

	tm.Clear() // Clear current screen
	drawTable(map[string]struct{}{
		"Timeout":     struct{}{},
		"Requests":    struct{}{},
		"Concurrency": struct{}{},
		"Targets":     struct{}{},
	}, []map[string]string{
		map[string]string{
			"Timeout":     strconv.Itoa(c.Timeout),
			"Requests":    strconv.Itoa(c.Requests),
			"Concurrency": strconv.Itoa(c.Concurrency),
			"Targets":     strconv.Itoa(len(c.Targets)),
		},
	}, false)
	fmt.Println(" ################## TEST RESULTS ################## ")
	drawTable(header, rows, false)
}

func drawTable(h map[string]struct{}, r []map[string]string, clear bool) {
	if clear {
		tm.Clear() // Clear current screen
	}
	// I use this slice to keep headers order
	headers := []string{}
	header := ""
	for hk := range h {
		if header != "" {
			header += "\t"
		}
		header += "[" + hk + "]"
		headers = append(headers, hk)
	}
	header += "\n"
	results := tm.NewTable(0, 20, 5, ' ', 0)
	fmt.Fprint(results, header)

	for _, v := range r {
		row := ""
		for _, hk := range headers {
			if row != "" {
				row += "\t"
			}
			vv, ok := v[hk]
			if ok {
				row += vv
			} else {
				row += "0"
			}
		}
		row += "\n"
		fmt.Fprint(results, row)
	}

	tm.Println(results)
	tm.Flush()
}
