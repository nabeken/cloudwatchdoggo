package doggo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// Main is an entry point for cloudwatchdoggo.
func Main(d *Doggo) error {
	alarms, err := d.ListAlarmsInAlarm()
	if err != nil {
		return fmt.Errorf("listing alarms: %w", err)
	}

	now := time.Now()

	if d.DebugOn {
		d.debugLogf("will bark after: %v", now.Add(-d.BarkInterval))
	}

	for _, alarm := range alarms {
		item, err := d.GetLatestBarkStatus(alarm)

		d.debugLogf("last barked at: %v, err: %v", item.LastBarkedAt, err)

		if item.ShouldBark(now.Add(-d.BarkInterval)) {
			d.debugLogf("BARK BARK BARK: %v", item)

			// BARK
			if err := d.Bark(alarm); err != nil {
				log.Printf("ERROR: unable to bark: %v", err)
				continue
			}

			item.barked()

			if err := d.updateLastBarkStatus(item); err != nil {
				log.Printf("ERROR: unable to save the state: %v", err)
			}
		} else {
			d.debugLogf("NO BARK PLEASE: %v", item)
		}
	}

	return nil
}

type Doggo struct {
	SNS        *sns.Client
	CloudWatch *cloudwatch.Client
	DynamoDB   *dynamodb.Client

	// DynamoDB Table to store the state
	TableName string

	// Amazon SNS Topic used with AWS Chatbot
	BarkSNSArn string

	// An interval to bark again since the last bark
	BarkInterval time.Duration

	// If true, it will print debug logs.
	DebugOn bool
}

func (d *Doggo) debugLogf(format string, v ...interface{}) {
	if d.DebugOn {
		log.Printf("DEBUG: "+format, v...)
	}
}

func (d *Doggo) ListAlarmsInAlarm() ([]types.MetricAlarm, error) {
	var nextToken *string
	var alarms []types.MetricAlarm

	for {
		resp, err := d.CloudWatch.DescribeAlarms(context.TODO(), &cloudwatch.DescribeAlarmsInput{
			StateValue: types.StateValueAlarm,
			NextToken:  nextToken,
		})

		if err != nil {
			return nil, err
		}

		for i := range resp.MetricAlarms {
			alarm := resp.MetricAlarms[i]

			// ignore alarm if the alarm action is disabled
			if !aws.ToBool(alarm.ActionsEnabled) {
				continue
			}

			// if scalingPolicy is the only alarm action, ignore it
			if len(alarm.AlarmActions) == 1 {
				alarmArn, err := arn.Parse(alarm.AlarmActions[0])
				if err != nil {
					return nil, fmt.Errorf("parsing alarm actions: %w", err)
				}

				if alarmArn.Service == "autoscaling" && strings.HasPrefix(alarmArn.Resource, "scalingPolicy:") {
					continue
				}
			}

			alarms = append(alarms, alarm)
		}

		if resp.NextToken == nil {
			break
		}

		nextToken = resp.NextToken
	}

	return alarms, nil
}

// BarkItemKey represents a primary key of the BarkItem.
type BarkItemKey struct {
	AlarmID        string `json:"alarm_id" dynamodbav:"alarm_id"`                 // hash key
	StateUpdatedAt int64  `json:"state_updated_at" dynamodbav:"state_updated_at"` // range key
}

// BarkItem represents an item stored in DynamoDB.
type BarkItem struct {
	BarkItemKey

	LastBarkedAt time.Time `json:"last_barked_at" dynamodbav:"last_barked_at"`
	TTL          int64     `json:"ttl" dynamodbav:"ttl"`
}

func (i BarkItem) ShouldBark(willBarkAfter time.Time) bool {
	return willBarkAfter.After(i.LastBarkedAt)
}

func (i *BarkItem) barked() {
	i.LastBarkedAt = time.Now()
	i.TTL = i.LastBarkedAt.Add(24 * time.Hour).Unix()
}

func (d *Doggo) GetLatestBarkStatus(alarm types.MetricAlarm) (*BarkItem, error) {
	avm, err := attributevalue.MarshalMap(BarkItemKey{
		AlarmID:        aws.ToString(alarm.AlarmArn),
		StateUpdatedAt: aws.ToTime(alarm.StateUpdatedTimestamp).Unix(),
	})

	if err != nil {
		return nil, fmt.Errorf("marshaling into DynamoDB item: %w", err)
	}

	resp, err := d.DynamoDB.GetItem(context.TODO(), &dynamodb.GetItemInput{
		Key:       avm,
		TableName: aws.String(d.TableName),
	})
	if err != nil {
		return nil, err
	}

	item := &BarkItem{}

	if err := attributevalue.UnmarshalMap(resp.Item, item); err != nil {
		return nil, fmt.Errorf("unmarshalling to the item: %w", err)
	}

	if item.AlarmID == "" {
		item.AlarmID = aws.ToString(alarm.AlarmArn)
		item.StateUpdatedAt = aws.ToTime(alarm.StateUpdatedTimestamp).Unix()
	}

	return item, err
}

// updateLastBarkStatus updates an item in DynamoDB.
// If an operation is conflicted, there is no need to bark then it will return false.
// If the operation succeeds, the bark is requested, then it will return true.
func (d *Doggo) updateLastBarkStatus(item *BarkItem) error {
	key, err := attributevalue.MarshalMap(item.BarkItemKey)
	if err != nil {
		return fmt.Errorf("marshaling into DynamoDB item: %w", err)
	}

	// build the update expression
	update := expression.
		Set(
			expression.Name("last_barked_at"),
			expression.Value(item.LastBarkedAt),
		).
		Set(
			expression.Name("ttl"),
			expression.Value(item.TTL),
		)
	expr, err := expression.NewBuilder().
		WithUpdate(update).
		Build()

	req := &dynamodb.UpdateItemInput{
		Key:       key,
		TableName: aws.String(d.TableName),

		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	}

	_, err = d.DynamoDB.UpdateItem(context.TODO(), req)
	return err
}

// Bark sends a given alarm to actions specified in the alarm.
func (d *Doggo) Bark(alarm types.MetricAlarm) error {
	alarmSNSPayload, err := ConvertMetricAlarmToAlarmSNSPayload(alarm)
	if err != nil {
		return err
	}

	msg, err := json.Marshal(alarmSNSPayload)
	if err != nil {
		return fmt.Errorf("marshalling alarm into JSON: %w", err)
	}

	fmt.Printf("sending SNS to %s\n%s", d.BarkSNSArn, string(msg))

	_, err = d.SNS.Publish(context.TODO(), &sns.PublishInput{
		TopicArn: aws.String(d.BarkSNSArn),
		Message:  aws.String(string(msg)),
	})

	if err != nil {
		return fmt.Errorf("sending SNS to '%s': %w", d.BarkSNSArn, err)
	}

	return nil
}

const alarmTimeFormat = "2006-01-02T15:04:05.999999999-0700"

func ConvertMetricAlarmToAlarmSNSPayload(alarm types.MetricAlarm) (*events.CloudWatchAlarmSNSPayload, error) {
	alarmArn, err := arn.Parse(*alarm.AlarmArn)
	if err != nil {
		return nil, fmt.Errorf("parsing AlarmArn: %w", err)
	}

	dimentions := []events.CloudWatchDimension{}
	for i := range alarm.Dimensions {
		dimentions = append(dimentions, events.CloudWatchDimension{
			Name:  *alarm.Dimensions[i].Name,
			Value: *alarm.Dimensions[i].Value,
		})
	}

	payload := &events.CloudWatchAlarmSNSPayload{
		AlarmName:        toStr(alarm.AlarmName),
		AlarmDescription: toStr(alarm.AlarmDescription),
		AWSAccountID:     alarmArn.AccountID,
		NewStateValue:    (string)(alarm.StateValue),
		NewStateReason:   toStr(alarm.StateReason) + "\n\n (CONT)",
		StateChangeTime:  aws.ToTime(alarm.StateUpdatedTimestamp).Format(alarmTimeFormat),
		Region:           alarmArn.Region,
		AlarmARN:         toStr(alarm.AlarmArn),
		OldStateValue:    "UNKOWN",

		Trigger: events.CloudWatchAlarmTrigger{
			Period:                           int64(aws.ToInt32(alarm.Period)),
			EvaluationPeriods:                int64(aws.ToInt32(alarm.EvaluationPeriods)),
			ComparisonOperator:               (string)(alarm.ComparisonOperator),
			Threshold:                        aws.ToFloat64(alarm.Threshold),
			TreatMissingData:                 toStr(alarm.TreatMissingData),
			EvaluateLowSampleCountPercentile: toStr(alarm.EvaluateLowSampleCountPercentile),

			// Metrics: omitted

			MetricName: toStr(alarm.MetricName),
			Namespace:  toStr(alarm.Namespace),
			Statistic:  (string)(alarm.Statistic),
			Unit:       (string)(alarm.Unit),

			Dimensions: dimentions,
		},
	}

	return payload, nil
}

func toStr(strptr *string) string {
	return aws.ToString(strptr)
}
