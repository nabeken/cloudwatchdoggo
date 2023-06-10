package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/nabeken/cloudwatchdoggo/doggo"
)

func mustFetchEnv(n string) string {
	v := os.Getenv(n)
	if v == "" {
		panic(fmt.Sprintf("environment variable %s must be set", n))
	}

	return v
}

func mustParseDuration(dur string) time.Duration {
	d, err := time.ParseDuration(dur)
	if err != nil {
		panic(err)
	}

	return d
}

func parseBool(v string) bool {
	b, _ := strconv.ParseBool(v)
	return b
}

type lambdaHandler struct {
	doggo *doggo.Doggo
}

func (h *lambdaHandler) handleLambda() error {
	return doggo.Main(h.doggo)
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

		TableName:    mustFetchEnv("DOGGO_TABLE_NAME"),
		BarkSNSArn:   mustFetchEnv("DOGGO_BARK_SNS_ARN"),
		BarkInterval: mustParseDuration(mustFetchEnv("DOGGO_BARK_INTERVAL")),

		DebugOn: parseBool(os.Getenv("DOGGO_DEBUG_ON")),
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
	f.BoolVar(&d.DebugOn, "debug", false, "enable the debug log")
	f.DurationVar(&d.BarkInterval, "bark-interval", time.Minute, "an interval to bark again since the last bark")

	if err := f.Parse(args[1:]); err != nil {
		return err
	}

	if d.TableName == "" {
		return fmt.Errorf("no table name specified")
	}
	if d.BarkSNSArn == "" {
		return fmt.Errorf("no ARN of SNS topic")
	}

	return doggo.Main(d)
}

func main() {
	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		if err := LambdaMain(); err != nil {
			log.Fatalf("LambdaMain: FATAL: %v", err)
		}
	} else {
		if err := Main(os.Args); err != nil {
			log.Fatalf("Main: FATAL: %v", err)
		}
	}
}
