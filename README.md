# GoTestWAF

GoTestWAF is a tool for API and OWASP attack simulation that supports a wide range of API protocols including
REST, GraphQL, gRPC, WebSockets, SOAP, XMLRPC, and others.

It was designed to evaluate web application security solutions, such as API security proxies, Web Application Firewalls,
IPS, API gateways, and others.

## How it works

GoTestWAF generates malicious requests using encoded payloads placed in different parts of HTTP requests: its body, headers,
URL parameters, etc. Generated requests are sent to the application security solution URL specified during GoTestWAF launch.
The results of the security solution evaluation are recorded in the report file created on your machine.

Default conditions for request generation are defined in the `testcases` folder in the YAML files of the following format:

```
payload:
  - '"union select -7431.1, name, @aaa from u_base--w-'
  - "'or 123.22=123.22"
  - "' waitfor delay '00:00:10'--"
  - "')) or pg_sleep(5)--"
encoder:
  - Base64Flat
  - URL
placeholder:
  - UrlPath
  - UrlParam
  - JSUnicode
  - Header
```

* `payload` is a malicious attack sample (e.g XSS string like ```<script>alert(111)</script>``` or something more sophisticated).
Since the format of the YAML string is required for payloads, they must be [encoded as binary data](https://yaml.org/type/binary.html).
* `encoder` is an encoder to be applied to the payload before placing it to the HTTP request. Possible encoders are:

    * Base64
    * Base64Flat
    * JSUnicode
    * URL
    * Plain (to keep the payload string as-is)
    * XML Entity
* `placeholder` is a place inside HTTP request where encoded payload should be. Possible placeholders are:

    * Header
    * RequestBody
    * SOAPBody
    * JSONBody
    * URLParam
    * URLPath

Request generation is a three-step process involving the multiplication of payload amount by encoder and placeholder amounts.
Let's say you defined 2 **payloads**, 3 **encoders** (Base64, JSUnicode, and URL) and 1 **placeholder** (URLParameter - HTTP GET parameter).
In this case, GoTestWAF will send 2x3x1 = 6 requests in a test case.

During GoTestWAF launch, you can also choose test cases between two embedded: OWASP Top-10, OWASP-API,
or your own (by using the [configuration option](https://github.com/wallarm/gotestwaf#configuration-options) `testCasePath`).

## Requirements

* GoTestwaf supports all the popular operating systems (Linux, Windows, macOS), and can be built natively
if [Go](https://golang.org/doc/install) is installed in the system.
* If running GoTestWAF as the Docker container, please ensure you have [installed and configured Docker](https://docs.docker.com/get-docker/),
and GoTestWAF and evaluated application security solution are connected to the same [Docker network](https://docs.docker.com/network/).
* For GoTestWAF to be successfully started, please ensure the IP address of the machine running GoTestWAF is whitelisted
on the machine running the application security solution.

## Quick start with Docker

The steps below walk through downloading and starting GoTestWAF with minimal configuration on Docker.

1. Pull the [GoTestWAF image](https://hub.docker.com/r/wallarm/gotestwaf) from Docker Hub:

    ```
    docker pull wallarm/gotestwaf
    ```
2. Start the GoTestWAF image:

    ```
    docker run -v ${PWD}/reports:/go/src/gotestwaf/reports --network=<NETWORK_FOR_GOTESTWAF_AND_SECSOLUTION> \
        wallarm/gotestwaf --url=<EVALUATED_SECURITY_SOLUTION_URL>
    ```

    If required, you can replace `${PWD}/reports` with the path to another folder used to place the evaluation report.
3. Find the report file `waf-evaluation-report-<date>.pdf` in the `reports` folder that you mapped to `/go/src/gotestwaf/reports`
inside the container.

You have successfully evaluated your application security solution by using GoTestWAF with minimal configuration.
To learn advanced configuration options, please use this [link](https://github.com/wallarm/gotestwaf#configuration-options).

## Demos

You can try GoTestWAF by running the demo environment that deploys NGINX‑based [ModSecurity using OWASP Core Rule Set](https://owasp.org/www-project-modsecurity-core-rule-set/)
and GoTestWAF evaluating ModSecurity. There are two options to run the demo environment:

* By using Docker
* By using the `make` utility

### Running the demo using Docker

1. Create the Docker network to link GoTestWAF and ModSecurity to. For example, to create the Docker network named `gotestwaf-modsecurity`:

    ```bash
    docker network create gotestwaf-modsecurity
    ```
2. Start containerized [ModSecurity](https://hub.docker.com/r/owasp/modsecurity-crs/) with minimal configuration:

    ```bash
    docker run -p <PORT_FOR_MODSECURITY>:80 -d -e PARANOIA=1 --rm --network=gotestwaf-modsecurity \
        owasp/modsecurity-crs:nginx
    ```

    You will find more ModSecurity configuration options and other image tags on [Docker Hub](https://hub.docker.com/r/owasp/modsecurity-crs/).

    Other options for the ModSecurity launch are described on the [ModSecurity GitHub](https://github.com/SpiderLabs/ModSecurity).
3. Start containerized GoTestWAF with minimal configuration:

    ```bash
    docker run -v ${PWD}/reports:/go/src/gotestwaf/reports --network=gotestwaf-modsecurity \
        wallarm/gotestwaf --url=<MODSECURITY_URL>
    ```

    If required, you can replace `${PWD}/reports` with the path to another folder used to place the evaluation report.

### Running the demo using the `make` utility

You can also run NGINX‑based ModSecurity using OWASP Core Rule Set and GoTestWAF evaluating ModSecurity by using
the `make` utility as follows (executed commands are defined in the Makefile located in the repository root):

1. Clone this repository and go to the local folder:

    ```
    git clone https://github.com/wallarm/gotestwaf.git
    cd gotestwaf
    ```
2. Start ModSecurity from the [Docker image](https://hub.docker.com/r/owasp/modsecurity-crs/) with minimal configuration:
    
    ```bash
    make modsec
    ```
3. Start GoTestWAF with minimal configuration by using one of the following commands:

    ```
    make scan_local # to run GoTestWAF natively with go
    make scan_local_from_docker # to run GoTestWAF from the Docker image
    ```

### Checking the evaluation results

Check the evaluation results logged using the `STDOUT` and `STDERR` services. For example:

```
GOTESTWAF : 2021/03/03 15:15:48.072331 main.go:61: Test cases loading started
GOTESTWAF : 2021/03/03 15:15:48.077093 main.go:68: Test cases loading finished
GOTESTWAF : 2021/03/03 15:15:48.077123 main.go:74: Scanned URL: http://127.0.0.1:8080/
GOTESTWAF : 2021/03/03 15:15:48.083134 main.go:85: WAF pre-check: OK. Blocking status code: 403
GOTESTWAF : 2021/03/03 15:15:48.083179 main.go:97: WebSocket pre-check. URL to check: ws://127.0.0.1:8080/
GOTESTWAF : 2021/03/03 15:15:48.251824 main.go:101: WebSocket pre-check: connection is not available, reason: websocket: bad handshake
GOTESTWAF : 2021/03/03 15:15:48.252047 main.go:129: Scanning http://127.0.0.1:8080/
GOTESTWAF : 2021/03/03 15:15:48.252076 scanner.go:124: Scanning started
GOTESTWAF : 2021/03/03 15:15:51.210216 scanner.go:129: Scanning Time:  2.958076338s
GOTESTWAF : 2021/03/03 15:15:51.210235 scanner.go:160: Scanning finished

Negative Tests:
+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+
|       TEST SET        |       TEST CASE       |     PERCENTAGE, %     |        BLOCKED        |       BYPASSED        |      UNRESOLVED       |
+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+
| community             | community-lfi         |                 66.67 |                     4 |                     2 |                     0 |
| community             | community-rce         |                 14.29 |                     6 |                    36 |                     0 |
| community             | community-sqli        |                 70.83 |                    34 |                    14 |                     0 |
| community             | community-xss         |                 91.78 |                   279 |                    25 |                     0 |
| community             | community-xxe         |                100.00 |                     4 |                     0 |                     0 |
| owasp                 | ldap-injection        |                  0.00 |                     0 |                     8 |                     0 |
| owasp                 | mail-injection        |                  0.00 |                     0 |                     6 |                     6 |
| owasp                 | nosql-injection       |                  0.00 |                     0 |                    12 |                     6 |
| owasp                 | path-traversal        |                 38.89 |                     7 |                    11 |                     6 |
| owasp                 | shell-injection       |                 37.50 |                     3 |                     5 |                     0 |
| owasp                 | sql-injection         |                 33.33 |                     8 |                    16 |                     8 |
| owasp                 | ss-include            |                 50.00 |                     5 |                     5 |                    10 |
| owasp                 | sst-injection         |                 45.45 |                     5 |                     6 |                     9 |
| owasp                 | xml-injection         |                100.00 |                    12 |                     0 |                     0 |
| owasp                 | xss-scripting         |                 56.25 |                     9 |                     7 |                    12 |
| owasp-api             | graphql               |                100.00 |                     1 |                     0 |                     0 |
| owasp-api             | rest                  |                100.00 |                     2 |                     0 |                     0 |
| owasp-api             | soap                  |                100.00 |                     2 |                     0 |                     0 |
+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+
|         DATE:         |       WAF NAME:       |  WAF AVERAGE SCORE:   |  BLOCKED (RESOLVED):  | BYPASSED (RESOLVED):  |      UNRESOLVED:      |
|      2021-03-03       |        GENERIC        |        55.83%         |   381/534 (71.35%)    |   153/534 (28.65%)    |    57/591 (9.64%)     |
+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+

Positive Tests:
+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+
|       TEST SET        |       TEST CASE       |     PERCENTAGE, %     |        BLOCKED        |       BYPASSED        |      UNRESOLVED       |
+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+
| false-pos             | texts                 |                 50.00 |                     1 |                     1 |                     6 |
+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+
|         DATE:         |       WAF NAME:       |  WAF POSITIVE SCORE:  | FALSE POSITIVE (RES): | TRUE POSITIVE (RES):  |      UNRESOLVED:      |
|      2021-03-03       |        GENERIC        |        50.00%         |     1/2 (50.00%)      |     1/2 (50.00%)      |     6/8 (75.00%)      |
+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+-----------------------+

PDF report is ready: reports/waf-evaluation-report-generic-2021-March-03-15-15-51.pdf
```

Find the report file `waf-evaluation-report-<date>.pdf` in the `reports` folder of the user directory.

## Other options to run GoTestWAF

In addition to running the GoTestWAF Docker image downloaded from Docker Hub, you can run GoTestWAF by using the following options:

* Clone this repository and build the GoTestWAF Docker image from the [Dockerfile](https://github.com/wallarm/gotestwaf/blob/master/Dockerfile), 
for example:

    ```
    git clone https://github.com/wallarm/gotestwaf.git
    cd gotestwaf
    docker build . --force-rm -t gotestwaf
    docker run -v ${PWD}/reports:/go/src/gotestwaf/reports gotestwaf --url=<EVALUATED_SECURITY_SOLUTION_URL>
    ```
* Clone this repository and run GoTestWAF with [`go`](https://golang.org/doc/), for example:

    ```
    git clone https://github.com/wallarm/gotestwaf.git
    cd gotestwaf
    go run ./cmd --url=<EVALUATED_SECURITY_SOLUTION_URL> --verbose
    ```
* Build GoTestWAF as the Go module:

    ```
    go build -mod vendor
    ```

Supported GoTestWAF configuration options are described below.

## Configuration options

```
Usage of /go/src/gotestwaf/gotestwaf:
      --addHeader string       An HTTP header to add to requests
      --blockConnReset         If true, connection resets will be considered as block
      --blockRegex string      Regex to detect a blocking page with the same HTTP response status code as a not blocked request
      --blockStatusCode int    HTTP status code that WAF uses while blocking requests (default 403)
      --configPath string      Path to the config file (default "config.yaml")
      --followCookies          If true, use cookies sent by the server. May work only with --maxIdleConns=1
      --idleConnTimeout int    The maximum amount of time a keep-alive connection will live (default 2)
      --ignoreUnresolved       If true, unresolved test cases will be considered as bypassed (affect score and results)
      --maxIdleConns int       The maximum number of keep-alive connections (default 2)
      --maxRedirects int       The maximum number of handling redirects (default 50)
      --nonBlockedAsPassed     If true, count requests that weren't blocked as passed. If false, requests that don't satisfy to PassStatuscode/PassRegExp as blocked
      --passRegex string       Regex to a detect normal (not blocked) web page with the same HTTP status code as a blocked request
      --passStatusCode int     HTTP response status code that WAF uses while passing requests (default 200)
      --proxy string           Proxy URL to use
      --randomDelay int        Random delay in ms in addition to the delay between requests (default 400)
      --reportPath string      A directory to store reports (default "reports")
      --sendDelay int          Delay in ms between requests (default 400)
      --skipWAFBlockCheck      If true, WAF detection tests will be skipped
      --testCase string        If set then only this test case will be run
      --testCasesPath string   Path to a folder with test cases (default "testcases")
      --testSet string         If set then only this test set's cases will be run
      --tlsVerify              If true, the received TLS certificate will be verified
      --url string             URL to check (default "http://localhost/")
      --verbose                If true, enable verbose logging (default true)
      --wafName string         Name of the WAF product (default "generic")
      --workers int            The number of workers to scan (default 200)
      --wsURL string           WebSocket URL to check
```

The listed options can be passed to GoTestWAF as follows:

* If running the GoTestWAF Docker container, pass the configuration options in the `docker run` command after the Docker image name.

    For example, to run GoTestWAF with WebSocket check, you can specify the WebSocket URL via the `wsURL` option
    and `verbose` flag to include more information about the checking process:

    ```
    docker run -v ${PWD}/reports:/go/src/gotestwaf/reports wallarm/gotestwaf --url=http://172.17.0.1:8080/ \
        --wsURL=ws://172.17.0.1:8080/api/ws --verbose
    ```

* If running GoTestWAF with `go run`, pass the configuration options and its values as the parameters for the main script.

    For example, to run GoTestWAF with WebSocket check, you can specify the WebSocket URL via the `wsURL` option and `verbose` flag to include more information about the checking process:

    ```
    go run ./cmd --url=http://127.0.0.1:8080/ --wsURL=ws://172.17.0.1:8080/api/ws --verbose
    ```
