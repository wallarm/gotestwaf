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
## Docker
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
`docker run -v ${PWD}/reports:/go/src/gotestwaf/reports gotestwaf --url=http://the-waf-you-wanna-test/`

#### Check results
```
owasp ss-include  5/20  (0.25)
owasp xml-injection 12/12 (1.00)
owasp xss-scripting 9/28  (0.32)
owasp ldap-injection  0/8 (0.00)
owasp mail-injection  3/12  (0.25)
owasp nosql-injection 0/18  (0.00)
owasp path-traversal  8/24  (0.33)
owasp shell-injection 3/8 (0.38)
owasp sql-injection 8/32  (0.25)
owasp sst-injection 5/20  (0.25)
owasp-api graphql 1/1 (1.00)
owasp-api rest  2/2 (1.00)
owasp-api soap  0/2 (0.00)
false-pos texts 7/8 (0.88)

WAF score: 42.18%
132 bypasses in 195 tests / 14 test cases
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
  --workers int
        The number of workers to scan (default 200)
```
