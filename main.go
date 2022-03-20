package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/nabeken/cloudwatchdoggo/doggo"
)

var barkInterval = 1 * time.Minute

type lambdaHandler struct {
	doggo *doggo.Doggo
}

func (h *lambdaHandler) handleLambda() error {
	return realMain(h.doggo)
}

func realMain(d *doggo.Doggo) error {
	alarms, err := d.ListAlarmsInAlarm()
	if err != nil {
		return fmt.Errorf("listing alarms: %w", err)
	}

	now := time.Now()

	log.Printf("INFO: will bark after: %v\n", now.Add(-barkInterval))

	for _, alarm := range alarms {
		item, err := d.GetLatestBarkStatus(alarm)
		log.Printf("INFO: last barked at: %v, err: %v\n", item.LastBarkedAt, err)

		if item.ShouldBark(now.Add(-barkInterval)) {
			log.Println("INFO: BARK BARK BARK:", item)

			// BARK
			if err := d.Bark(alarm); err != nil {
				log.Printf("ERROR: unable to bark: %v", err)
				continue
			}

			item.Barked()

			if err := d.UpdateLastBarkStatus(item); err != nil {
				log.Printf("ERROR: unable to save the state: %v", err)
			}
		} else {
			log.Printf("INFO: NO BARK PLEASE: %v", item)
		}
	}

	return nil
}

func LambdaMain() error {
	log.Print("Launching AWS Lambda handler...")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return fmt.Errorf("loading SDK config, %w", err)
	}

	d := &doggo.Doggo{
		CloudWatch: cloudwatch.NewFromConfig(cfg),
		DynamoDB:   dynamodb.NewFromConfig(cfg),
		SNS:        sns.NewFromConfig(cfg),
		TableName:  os.Getenv("DOGGO_TABLE_NAME"),
		BarkSNSArn: os.Getenv("BARK_SNS_ARN"),
	}

	h := &lambdaHandler{doggo: d}

	lambda.Start(h.handleLambda)

	return nil
}

func Main(args []string) error {
	f := flag.NewFlagSet(args[0], flag.ExitOnError)

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return fmt.Errorf("loading SDK config, %w", err)
	}

	d := &doggo.Doggo{
		CloudWatch: cloudwatch.NewFromConfig(cfg),
		DynamoDB:   dynamodb.NewFromConfig(cfg),
		SNS:        sns.NewFromConfig(cfg),
	}

	f.StringVar(&d.TableName, "table", "", "specify a name of DynamoDB table to store the bark state")
	f.StringVar(&d.BarkSNSArn, "sns-arn", "", "specify a ARN of SNS topic to bark")

	if err := f.Parse(args[1:]); err != nil {
		return err
	}

	if d.TableName == "" {
		return fmt.Errorf("no table name specified")
	}
	if d.BarkSNSArn == "" {
		return fmt.Errorf("no ARN of SNS topic")
	}

	return realMain(d)
}

func main() {
	if strings.HasPrefix(os.Getenv("AWS_EXECUTION_ENV"), "AWS_Lambda") {
		if err := LambdaMain(); err != nil {
			log.Fatalf("FATAL: %v", err)
		}
	} else {
		if err := Main(os.Args); err != nil {
			log.Fatalf("FATAL: %v", err)
		}
	}
}
