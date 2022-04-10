# cloudwatchdoggo

`cloudwatchdoggo` is a periodic watchdoggo for CloudWatch Alarms. It continue borking through the AWS Chatbot as CloudWatch Alarm until the alarm goes back to OK state.

## Installation

**As the lambda function**:

You can replicate [the Docker container image](https://github.com/nabeken/cloudwatchdoggo/pkgs/container/cloudwatchdoggo) to your ECR repository and deploy it as the lambda function with Container Image Support.

**As a standalone command-line application**:

TBD

**As a part of your application**:

If you want to integrate the doggo into your existing command line application as a subcommand, you can implement an entry point with the `doggo` package.

Please read the documentation how to initialize and the doggo.

## Requirements

AWS Chatbot configuration is required because the doggo will send CloudWatch Alarm through the AWS Chatbot (SNS).

The doggo will not bark again if the last bark is within an specified interval. To record the state, it will use DynamoDB table with the following schema.

- **Partition Key**: `alarm_id (String)`
- **Sort key**: `state_updated_at (Number)`

## Configuration

As for the command-line application and the lambda function, the configuration is done by the environment variable since the doggo is designed to work with the lambda function.

- `DOGGO_TABLE_NAME`: DynamoDB Table to store the state
- `DOGGO_BARK_SNS_ARN`: Amazon SNS Topic used with AWS Chatbot
- `DOGGO_BARK_INTERVAL`: An interval to bark again since the last bark. It must be string that [Go's `time.ParseDuration`](https://pkg.go.dev/time#ParseDuration) accepts.
- `DOGGO_DEBUG_ON`: Set to `true` if you want to see the debug log

As for using the dogggo directly, please read the document in `doggo` package.

---

What is [your favorite doggo](https://www.youtube.com/watch?v=sowESlcktC8) BTW?
