# Go Test WAF

An open-source Go project to test different WAF for detection logic and bypasses.

# How it works

It is a 3-steps requests generation process that multiply amount of payloads to encoders and placeholders. Let's say you defined 2 payloads, 3 encoders (Base64, JSON, and URLencode) and 1 placeholder (HTTP GET variable). In this case, the tool will send 2x3x1 = 6 requests in a testcase.

## Payload

The payload string you wanna send. Like ```<script>alert(111)</script>``` or something more sophisticated. There is no macroses like so far, but it's in our TODO list. Since it's a YAML string, use binary encoding if you wanna to https://yaml.org/type/binary.html

## Encoder

Data encoder the tool should apply to the payload. Base64, JSON unicode (\u0027 instead of '), etc.

## Placeholder

A place inside HTTP request where encoded payload should be. Like URL parameter, URI, POST form parameter, or JSON POST body.

# Quick start
```
docker build -t gotestwaf .
docker run -v /tmp:/tmp/report gotestwaf --url=https://the-waf-you-wanna-test/
```

Find the report file waf-test-report-<date>.pdf in a /tmp folder you mapped to /tmp/report inside the container.
