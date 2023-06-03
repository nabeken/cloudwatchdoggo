# cloudwatchdoggo

`cloudwatchdoggo` is a periodic watchdoggo for CloudWatch Alarms. It continue borking through the AWS Chatbot as CloudWatch Alarm until the alarm goes back to OK state.

## Installation

**As the lambda function**:

- You can replicate [the Docker container image](https://github.com/nabeken/cloudwatchdoggo/pkgs/container/cloudwatchdoggo) to your ECR repository and deploy it as the lambda function with Container Image Support.
- You can also provision the function with [the terraform module](https://registry.terraform.io/modules/nabeken/cloudwatchdoggo/aws/latest).

**As a standalone command-line application**:

```sh
go install github.com/nabeken/cloudwatchdoggo@latest
cloudwatchdoggo -h

Usage of cloudwatchdoggo:
  -bark-interval duration
    	an interval to bark again since the last bark (default 1m0s)
  -debug
    	enable the debug log
  -sns-arn string
    	specify a ARN of SNS topic to bark
  -table string
    	specify a name of DynamoDB table to store the bark state
```

**As a part of your application**:

If you want to integrate the doggo into your existing command line application as a subcommand, you can implement an entry point with the `doggo` package.

Please read the documentation how to initialize and the doggo.

## Requirements

AWS Chatbot configuration is required because the doggo will send CloudWatch Alarm through the AWS Chatbot (SNS).

The doggo will only bark again if the last bark time passes a specified interval. To record the state, it will use DynamoDB table with the following schema.

- **Partition Key**: `alarm_id (String)`
- **Sort key**: `state_updated_at (Number)`
- **TTL**: `ttl (Number)`
  - TTL will be set for 24 hours

## Configuration

As for the command-line application and the lambda function, the configuration is done by the environment variable since the doggo is designed to work with the lambda function.

- `DOGGO_TABLE_NAME`: DynamoDB Table to store the state
- `DOGGO_BARK_SNS_ARN`: Amazon SNS Topic used with AWS Chatbot
- `DOGGO_BARK_INTERVAL`: An interval to bark again since the last bark. It must be string that [Go's `time.ParseDuration`](https://pkg.go.dev/time#ParseDuration) accepts.
- `DOGGO_DEBUG_ON`: Set to `true` if you want to see the debug log

As for using the dogggo directly, please read the document in `doggo` package.

Other than that, please feed periodical events to the doggo. EventBridge's scheduled event will be a choice.

## Example IAM policy for the lambda function

You should attach `arn:aws:iam::aws:policy/AWSLambdaExecute` and the following policy for IAM role that the lambda function will use.
```
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "VisualEditor0",
      "Effect": "Allow",
      "Action": "cloudwatch:DescribeAlarms",
      "Resource": "*"
    },
    {
      "Sid": "VisualEditor1",
      "Effect": "Allow",
      "Action": [
        "sns:Publish",
        "dynamodb:PutItem",
        "dynamodb:DeleteItem",
        "dynamodb:GetItem",
        "dynamodb:UpdateItem"
      ],
      "Resource": [
        "arn:aws:sns:*:<account-id>:<topic>",
        "arn:aws:dynamodb:*:<account-id>:table/<table name>"
      ]
    }
  ]
}
```

---

What is [your favorite doggo](https://www.youtube.com/watch?v=sowESlcktC8) BTW?
