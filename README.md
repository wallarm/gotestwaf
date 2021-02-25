# Go Test WAF

An open-source Go project to test different web application firewalls (WAF) for detection logic and bypasses.

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
```
git clone https://github.com/SpiderLabs/owasp-modsecurity-crs
cd owasp-modsecurity-crs/util/docker
docker build -t modsec_crs --file Dockerfile-3.0-nginx .
docker run --rm -p 8080:80 -e PARANOIA=1 modsec_crs
```

You may choose the PARANOIA level to increase the level of security.  
Learn more https://coreruleset.org/faq/

#### Run gotestwaf
`docker run -v ${PWD}/reports:/go/src/gotestwaf/reports gotestwaf --url=http://172.17.0.1:8080/`

#### Run gotestwaf with WebSocket check
You can additionally set the WebSocket URL to check via the `wsURL` flag and `verbose` flag to include more information about the checking process:  
`docker run -v ${PWD}/reports:/go/src/gotestwaf/reports gotestwaf --url=http://172.17.0.1:8080/ --wsURL=ws://172.17.0.1:8080/api/ws --verbose`


#### Check results
```
GOTESTWAF : 2021/02/25 02:54:04.782526 cmd.go:66: Test cases loading started
GOTESTWAF : 2021/02/25 02:54:04.784032 cmd.go:72: Test cases loading finished
GOTESTWAF : 2021/02/25 02:54:04.784050 cmd.go:78: Scanned URL: http://172.17.0.1:8080/
GOTESTWAF : 2021/02/25 02:54:04.788380 cmd.go:91: WAF pre-check: OK. Blocking status code: 403
GOTESTWAF : 2021/02/25 02:54:04.788397 cmd.go:102: WebSocket pre-check. URL to check: ws://172.17.0.1:8080/api/ws
GOTESTWAF : 2021/02/25 02:54:04.791253 cmd.go:106: WebSocket pre-check: connection is not available, reason: websocket: bad handshake
GOTESTWAF : 2021/02/25 02:54:04.791354 cmd.go:135: Scanning http://172.17.0.1:8080/
GOTESTWAF : 2021/02/25 02:54:04.791373 scanner.go:124: Scanning started
GOTESTWAF : 2021/02/25 02:54:07.268681 scanner.go:129: Scanning Time:  2.477299327s
GOTESTWAF : 2021/02/25 02:54:07.268693 scanner.go:160: Scanning finished
+------------------+-----------------+---------------+----------------+-----------------+
|     TEST SET     |    TEST CASE    | PERCENTAGE, % | PASSED/BLOCKED | FAILED/BYPASSED |
+------------------+-----------------+---------------+----------------+-----------------+
| community        | community-lfi   |         66.67 |              4 |               2 |
| community        | community-rce   |         14.29 |              6 |              36 |
| community        | community-sqli  |         70.83 |             34 |              14 |
| community        | community-xss   |         91.78 |            279 |              25 |
| community        | community-xxe   |        100.00 |              4 |               0 |
| false-pos        | texts           |         87.50 |              1 |               7 |
| owasp            | ldap-injection  |         12.50 |              1 |               7 |
| owasp            | mail-injection  |         25.00 |              3 |               9 |
| owasp            | nosql-injection |          0.00 |              0 |              18 |
| owasp            | path-traversal  |         33.33 |              8 |              16 |
| owasp            | shell-injection |         37.50 |              3 |               5 |
| owasp            | sql-injection   |         25.00 |              8 |              24 |
| owasp            | ss-include      |         25.00 |              5 |              15 |
| owasp            | sst-injection   |         25.00 |              5 |              15 |
| owasp            | xml-injection   |        100.00 |             12 |               0 |
| owasp            | xss-scripting   |         32.14 |              9 |              19 |
| owasp-api        | graphql         |        100.00 |              1 |               0 |
| owasp-api        | rest            |        100.00 |              2 |               0 |
| owasp-api        | soap            |        100.00 |              2 |               0 |
+------------------+-----------------+---------------+----------------+-----------------+
| DATE: 2021-02-25 |    WAF NAME:    |    GENERIC    |   WAF SCORE:   |     55.08%      |
+------------------+-----------------+---------------+----------------+-----------------+

PDF report is ready: reports/waf-evaluation-report-generic-2021-February-25-02-54-07.pdf
```
---

### Configuration options
```
Usage of /go/src/gotestwaf/gotestwaf:
  --blockRegExp string    
        Regexp to detect a blocking page with the same HTTP response status code as a not blocked request
  --blockStatusCode int   
        HTTP status code that WAF uses while blocking requests (default 403)
  --config string         
        Path to a config file (default "config.yaml")
  --followCookies         
        If true, use cookies sent by the server. May work only with --maxIdleConns=1
  --idleConnTimeout int   
        The maximum number of keep-alive connections (default 2)
  --maxIdleConns int      
        The maximum amount of time a keep-alive connection will live (default 2)
  --maxRedirects int      
        The maximum number of handling redirects (default 50)
  --nonBlockedAsPassed    
        If true, count requests that weren't blocked as passed. If false, requests that don't satisfy to PassStatuscode/PassRegExp as blocked
  --passRegExp string     
        Regexp to a detect normal (not blocked) web page with the same HTTP status code as a blocked request
  --passStatusCode int    
        HTTP response status code that WAF uses while passing requests (default 200)
  --proxy string          
        Proxy URL to use
  --randomDelay int       
        Random delay in ms in addition to --sendDelay (default 400)
  --reportDir string      
        A directory to store reports (default "/tmp/gotestwaf/")
  --sendDelay int         
        Delay in ms between requests (default 400)
  --testCase string
          If set, then only this test case will be run
  --testCasesPath string      
        Path to a folder with test cases (default "./testcases/")
  --testSet string
          If set, then only this test set's cases will be run
  --tlsVerify             
        If true, the received TLS certificate will be verified
  --url string            
        URL to check (default "http://localhost/")
  --verbose                
        If true, enable verbose logging (default true)
  --wafName string
        Name of the WAF product (default "generic")
  --workers int
        The number of workers to scan (default 200)
  --wsURL string
        WebSocket URL to check
```
