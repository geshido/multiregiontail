package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type LogItem struct {
	Region  string
	Message string
	Time    time.Time
}

func main() {
	var regs, profile, logGroup, filter string
	var rlimit int
	flag.StringVar(&regs, "regs", "", "regs, comma separated")
	flag.StringVar(&profile, "profile", "default", "AWS profile to use for credentials")
	flag.StringVar(&logGroup, "group", "", "log group name")
	flag.StringVar(&filter, "filter", "", "filter events as described at https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html")
	flag.IntVar(&rlimit, "rlimit", 0, "if provided, log items will be printed each <rlimit> ms")
	flag.Parse()

	if logGroup == "" {
		flag.Usage()
		os.Exit(1)
	}

	var regions []string
	if regs == "" {
		for region := range endpoints.AwsPartition().Regions() {
			regions = append(regions, region)
		}
	} else {
		regions = strings.Split(regs, ",")
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Profile: profile,
	})
	if err != nil {
		log.Panicf("can not create session: %v", err)
	}

	logs := make(chan LogItem, 100)
	wg := sync.WaitGroup{}
	wg.Add(len(regions))

	for _, region := range regions {
		go func(reg string) {
			defer wg.Done()
			cw := cloudwatchlogs.New(sess, aws.NewConfig().WithRegion(reg))

			startTime := time.Now()
			var seenIds []string
			for {
				input := cloudwatchlogs.FilterLogEventsInput{
					LogGroupName: aws.String(logGroup),
					StartTime:    aws.Int64(timeToMillis(startTime)),
				}
				if filter != "" {
					input.FilterPattern = aws.String(filter)
				}

				var events []*cloudwatchlogs.FilteredLogEvent
				err := cw.FilterLogEventsPages(&input, func(out *cloudwatchlogs.FilterLogEventsOutput, lastPage bool) bool {
					for _, e := range out.Events {
						events = append(events, e)
					}
					return true
				})
				if err != nil {
					log.Printf("%s: can not get events: %v", reg, err)
					return
				}

				for _, event := range events {
					seen := false
					for _, id := range seenIds {
						if id == *event.EventId {
							seen = true
							break
						}
					}
					if seen {
						continue
					}
					seenIds = append(seenIds, *event.EventId)

					eventTime := millisToTime(*event.Timestamp)

					logs <- LogItem{
						Region:  reg,
						Message: strings.TrimSpace(*event.Message),
						Time:    eventTime,
					}

					if *event.Timestamp > timeToMillis(startTime) {
						startTime = eventTime
					}
				}

				time.Sleep(2 * time.Second)
			}
		}(region)
	}

	done := make(chan struct{})
	go func() {
		if rlimit > 0 {
			ticker := time.Tick(time.Duration(rlimit) * time.Millisecond)

			for {
				select {
				case <-ticker:
					select {
					case item := <-logs:
						logItem(item)
					}
				case <-done:
					return
				}
			}
		} else {
			for item := range logs {
				logItem(item)
			}
		}

	}()

	wg.Wait()
	close(logs)
	close(done)
	time.Sleep(time.Second)
}

func logItem(item LogItem) {
	fmt.Printf("[%20s] %s: %s\n", item.Region, item.Time.Format(time.RFC3339), item.Message)
}

func timeToMillis(t time.Time) int64 {
	return t.UnixNano() / 1000000
}
func millisToTime(ms int64) time.Time {
	return time.Unix(0, ms*int64(time.Millisecond))
}
