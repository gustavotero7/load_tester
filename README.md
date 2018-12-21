# Load Tester

HTTP load testing tool with concurrent calls support

## Setup

Load tester uses a file called conf.yml in the same location as the binary

* timeout: Default timeout for http request
* requests: Total requests to be made for each target
* concurrency: Concurrent calls count for each target, for instance, if you set concurrency to 10 the load tester will have 10 concurrent calls for each target, so if you have 3 targets the load tester will have 30 concurrent calls running
* targets: Target hosts to test
  * url: Host address
  * method: http method
  * payload: request body
  * header: request header, here you can specify multiple key/value 

```yaml
timeout: 2
requests: 100
concurrency: 10
targets:
- url: https://google.com
  method: GET
- url: http://api.example.com
  method: POST
  payload: '{"ping":"pong"}'
  header:
    Authorization: Basic 2321321321312312
    Ping: pong

```

## Run

Open your bash/terminal and just run the downloaded binary, don't forget to place the config.yml file in the same location as the binary before running the load tester

`./load_tester`