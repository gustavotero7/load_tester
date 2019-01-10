package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	tm "github.com/buger/goterm"
	"github.com/sirupsen/logrus"
)

// Response _
type Response struct {
	Status string
	Body   json.RawMessage
	Header http.Header
}

// Result _
type Result struct {
	Key      string
	Err      error
	Code     int
	Took     float64
	Response Response
}

func main() {
	cPath := flag.String("c", "conf.yml", "Config file")
	rPath := flag.String("o", "", "Path/File name to store results in JSON format EX: load_tester -o myResults.json")
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

					var reader io.Reader
					if len(v.Payload) > 0 {
						reader = strings.NewReader(v.Payload)
					}
					req, err := http.NewRequest(v.Method, v.URL, reader)
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

					if res != nil {
						if res.Body != nil {
							json.NewDecoder(res.Body).Decode(&rs.Response.Body)
						}
						rs.Response.Header = res.Header
						rs.Response.Status = res.Status
					}
					rs.Took = time.Since(took).Seconds()
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

			// Check request time
			c.Targets[rs.Key].Results.TestTime += rs.Took
			if c.Targets[rs.Key].Results.MinTime > 0 && c.Targets[rs.Key].Results.MaxTime > 0 {
				if rs.Took < c.Targets[rs.Key].Results.MinTime {
					c.Targets[rs.Key].Results.MinTime = rs.Took
				}

				if rs.Took > c.Targets[rs.Key].Results.MaxTime {
					c.Targets[rs.Key].Results.MaxTime = rs.Took
				}
			} else {
				c.Targets[rs.Key].Results.MinTime = rs.Took
				c.Targets[rs.Key].Results.MaxTime = rs.Took
			}

			// Store response
			c.Targets[rs.Key].Results.Responses = append(c.Targets[rs.Key].Results.Responses, rs.Response)

		}
	}

	// Store results only if a results file path is provided
	if *rPath != "" {
		// Store/Write responses
		bts, _ := json.Marshal(c.Targets)
		ioutil.WriteFile(*rPath, bts, 777)
	}

	// Render results
	header := map[string]struct{}{
		"Test":    struct{}{},
		"Total":   struct{}{},
		"MinTime": struct{}{},
		"MaxTime": struct{}{},
		"AvgTime": struct{}{},
	}
	rows := []map[string]string{}
	for tk, t := range c.Targets {
		log.Println("Test Time", t.Results.TestTime)
		row := map[string]string{}
		row["Test"] = tk
		row["Total"] = fmt.Sprintf("%v", t.Results.Total)
		row["MinTime"] = fmt.Sprintf("%.2f s", t.Results.MinTime)
		row["MaxTime"] = fmt.Sprintf("%.2f s", t.Results.MaxTime)
		row["AvgTime"] = fmt.Sprintf("%.2f s", t.Results.TestTime/float64(t.Results.Total))
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
