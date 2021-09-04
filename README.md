# Go Test WAF

GoTestWAF is a tool for API and OWASP attack simulation, that supports a wide range of API protocols including
REST, GraphQL, gRPC, WebSockets, SOAP, XMLRPC, and others.

It was designed to evaluate web application security solutions, such as API security proxies, Web Application Firewalls, IPS, API gateways, and others.

# How it works

It is a 3-steps requests generation process that multiply amount of payloads to encoders and placeholders.
Let's say you defined 2 payloads, 3 encoders (Base64, JSON, and URLencode) and 1 placeholder (HTTP GET variable).
In this case, the tool will send 2x3x1 = 6 requests in a testcase.

## Payload

The payload string you wanna send. Like ```<script>alert(111)</script>``` or something more sophisticated.
There is no macroses like so far, but it's in our TODO list. 
Since it's a YAML string, use binary encoding if you wanna to https://yaml.org/type/binary.html

## Encoder

Data encoder the tool should apply to the payload. Base64, JSON unicode (\u0027 instead of '), etc.

## Placeholder

A place inside HTTP request where encoded payload should be.
Like URL parameter, URI, POST form parameter, or JSON POST body.

## Caveats
We recommend adding the scanner's IP address to the whitelists before executing the test.

# Quick start
## Dockerhub
The latest gotestwaf always available via the dockerhub repository: https://hub.docker.com/r/wallarm/gotestwaf  
It can be easily pulled via the following command:  
```
docker pull wallarm/gotestwaf
```
## Local Docker build
```
docker build . --force-rm -t gotestwaf
docker run -v ${PWD}/reports:/go/src/gotestwaf/reports gotestwaf --url=https://the-waf-you-wanna-test/
```
Find the report file `waf-test-report-<date>.pdf` in the `reports` folder that you mapped to `/go/src/gotestwaf/reports` inside the container.

## Build
Gotestwaf supports all the popular platforms (Linux, Windows, macOS), and can be built natively if Go is installed in the system.
```
go build -mod vendor
```

# Examples

## Testing on OWASP ModSecurity Core Rule Set

#### Build & run ModSecurity CRS docker image
You can pull, build and run ModSecurity CRS docker image automatically:
```
make modsec
```
Or manually with your configuration flags to test:
```
docker pull owasp/modsecurity-crs
docker run -p 8080:80 -d -e PARANOIA=1 --rm owasp/modsecurity-crs
```
You may choose the PARANOIA level to increase the level of security.  
Learn more https://coreruleset.org/faq/

#### Run gotestwaf
If you want to test the functionality on the running ModSecurity CRS docker container, you can use the following commands:
```
make scan_local               (to run natively)
make scan_local_from_docker   (to run from docker)
```
Or manually from docker:
```
docker run -v ${PWD}/reports:/go/src/gotestwaf/reports --network="host" gotestwaf --url=http://127.0.0.1:8080/ --verbose
```
And manually with `go run` (natively):
```
go run ./cmd --url=http://127.0.0.1:8080/ --verbose
```

#### Run gotestwaf with WebSocket check
You can additionally set the WebSocket URL to check via the `wsURL` flag and `verbose` flag to include more information about the checking process:  
```
docker run -v ${PWD}/reports:/go/src/gotestwaf/reports gotestwaf --url=http://172.17.0.1:8080/ --wsURL=ws://172.17.0.1:8080/api/ws --verbose
```


#### Check results
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
---

### Configuration options
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
