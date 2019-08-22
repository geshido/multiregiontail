package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
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

var allRegions = strings.Join([]string{
	"eu-north-1",
	"ap-south-1",
	"eu-west-3",
	"eu-west-2",
	"eu-west-1",
	"ap-northeast-2",
	"ap-northeast-1",
	"sa-east-1",
	"ca-central-1",
	"ap-southeast-1",
	"ap-southeast-2",
	"eu-central-1",
	"us-east-1",
	"us-east-2",
	"us-west-1",
	"us-west-2",
}, ",")

func main() {
	var regs, profile, logGroup string
	var rlimit int
	flag.StringVar(&regs, "regs", "", "regs, comma separated")
	flag.StringVar(&profile, "profile", "default", "AWS profile to use for credentials")
	flag.StringVar(&logGroup, "group", "", "log group name")
	flag.IntVar(&rlimit, "rlimit", 0, "if provided, log items will be printed each <rlimit> ms")
	flag.Parse()

	if logGroup == "" {
		flag.Usage()
		os.Exit(1)
	}
	if regs == "" {
		regs = allRegions
	}

	regions := strings.Split(regs, ",")

	logs := make(chan LogItem, 100)
	wg := sync.WaitGroup{}
	wg.Add(len(regions))

	for _, region := range regions {
		go func(reg string, logs chan<- LogItem) {
			defer wg.Done()
			sess, err := session.NewSessionWithOptions(session.Options{
				Profile: profile,
			})
			if err != nil {
				log.Printf("%s: can not create session: %v", reg, err)
				return
			}
			sess.Config.Region = aws.String(reg)
			cw := cloudwatchlogs.New(sess)

			startTime := time.Now()
			var seenIds []string
			for {
				input := cloudwatchlogs.FilterLogEventsInput{
					LogGroupName: aws.String(logGroup),
					StartTime:    aws.Int64(timeToMillis(startTime)),
				}
				out, err := cw.FilterLogEvents(&input)
				if err != nil {
					log.Printf("%s: can not get events: %v", reg, err)
					return
				}

				for _, event := range out.Events {
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
		}(region, logs)
	}

	done := make(chan struct{})
	go func(logs <-chan LogItem) {
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

	}(logs)

	wg.Wait()
	close(logs)
	close(done)
	time.Sleep(time.Second)
}

func logItem(item LogItem) {
	fmt.Printf("[%15s] %s: %s\n", item.Region, item.Time.Format(time.RFC3339), item.Message)
}

func timeToMillis(t time.Time) int64 {
	return t.UnixNano() / 1000000
}
func millisToTime(ms int64) time.Time {
	return time.Unix(0, ms*int64(time.Millisecond))
}
