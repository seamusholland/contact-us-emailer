package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/sns"

	"context"
	"fmt"
	"os"
)

type ContactUsRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

type Response struct {
	Developer int `json:"developer"`
}

type DynamoItem struct {
	Email string `json:"Email"`
	Name  string `json:"Name"`
}

func HandleRequest(ctx context.Context, req ContactUsRequest) (string, error) {
	// Initialize a session that the SDK will use to load
	// credentials from the shared credentials file. (~/.aws/credentials).
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	dynamo_result, dynamo_err := pushToDynamo(req.Name, req.Email, sess)
	if dynamo_err != nil {
		fmt.Println("Error while pushing to Dynamo!")
	}

	sns_result, err := pushToSNS(req.Name, req.Email, req.Message, sess)
	if err != nil {
		fmt.Println("Error while pushing to SNS!")
		return "", err
	}

	if dynamo_err != nil {
		fmt.Println("N.B.: Partial Success")
	}

	return sns_result + "\n" + dynamo_result, nil
}

func pushToDynamo(name string, email string, sess *session.Session) (string, error) {
	table := os.Getenv("DYNAMODB_TABLE")

	svc := dynamodb.New(sess)

	ddbItem := DynamoItem{
		Email: email,
		Name:  name,
	}

	av, err := dynamodbattribute.MarshalMap(ddbItem)
	if err != nil {
		fmt.Println("Encountered error while marshalling new contact item!")
		fmt.Println(err.Error())
		return "", err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(table),
	}

	_, err = svc.PutItem(input)
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}
	fmt.Println("Successfully added '" + ddbItem.Email + "' (" + ddbItem.Name + ") to table " + table)

	return "Dynamo push successful.", nil
}

func pushToSNS(name string, email string, message string, sess *session.Session) (string, error) {
	msg := message + "\n\nTo reply to this message, contact " + name + " at " + email + "."
	sub := "Contact Request from " + name
	topic := os.Getenv("SNS_TOPIC")

	svc := sns.New(sess)

	result, err := svc.Publish(&sns.PublishInput{
		Message:  &msg,
		Subject:  &sub,
		TopicArn: &topic,
	})

	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}

	fmt.Printf("Successfully pushed SNS messaged with ID %s", *result.MessageId)

	return *result.MessageId, nil
}

func main() {
	lambda.Start(HandleRequest)
}
