package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var ddbClient *dynamodb.Client

type Response events.APIGatewayProxyResponse

type ValidateCodeResponse struct {
	Message string `json:"message"`
}

type ValidateCodeRequest struct {
	Id   string `json:"id"`
	Code string `json:"code"`
}

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (Response, error) {
	var input ValidateCodeRequest
	err := json.Unmarshal([]byte(event.Body), &input)
	if err != nil {
		log.Printf("unmarshal request: %v\n", err)
		return GenerateResponse(err.Error(), 500)
	}

	getItemResponse, err := ddbClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("TABLE")),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: input.Id},
		},
	})

	if err != nil {
		log.Printf("put item: %v\n", err)
		return GenerateResponse(err.Error(), 500)
	}

	if getItemResponse.Item == nil {
		return GenerateResponse("NOT FOUND", 404)
	}

	actualCode := getItemResponse.Item["code"].(*types.AttributeValueMemberS).Value
	ttl, err := strconv.Atoi(getItemResponse.Item["TTL"].(*types.AttributeValueMemberN).Value)

	if actualCode != input.Code {
		return GenerateResponse("INCORRECT", 400)
	} else if int64(ttl) < time.Now().Unix() {
		return GenerateResponse("NOT FOUND", 404)
	}

	return GenerateResponse("OK", 200)

}

func GenerateResponse(message string, status int) (Response, error) {

	response := ValidateCodeResponse{
		Message: message,
	}

	body, err := json.Marshal(response)
	if err != nil {
		return Response{StatusCode: 500}, err
	}
	resp := Response{
		StatusCode:      status,
		IsBase64Encoded: false,
		Body:            string(body),
	}

	return resp, nil
}

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	ddbClient = dynamodb.NewFromConfig(cfg)

	rand.Seed(time.Now().UnixNano())
}

func main() {
	lambda.Start(Handler)
}
