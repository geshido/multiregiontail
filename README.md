Multi-region tail
===

A tool for tailing CloudWatch logs of different regions at once

Install
---

`go get github.com/geshido/multiregiontail`

Run
---

```
Usage of multiregiontail:
  -debug
    	if provided, profiler will be launched on localhost:6060
  -filter string
    	filter events as described at https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html
  -group string
    	log group name
  -profile string
    	AWS profile to use for credentials (default "default")
  -regs string
    	regs, comma separated
  -rlimit int
    	if provided, log items will be printed each <rlimit> ms
  -since string
    	YYYY-MM-DDTHH:MM:SS version of initial point from which log events will be retrieved
```
