# Go Test WAF

An open-source Go project to test different WAF for detection logic and bypasses.

# How it works

It is a 3-steps requests generation process that multiply amount of payloads to encoders and placeholders. Let's say you defined 2 payloads, 3 encoders (Base64, JSON, and URLencode) and 1 placeholder (HTTP GET variable). In this case, the tool will send 2x3x1 = 6 requests in a testcase.

## Payload
The payload string you wanna send. Like ```<script>alert(111)</script>``` or something more sophisticated. There is no macroses like so far, but it's in our TODO list. Since it's a YAML string, use binary encoding if you wanna to https://yaml.org/type/binary.html

## Encoder
Data encoder the tool should apply to the payload.

## Placeholde
