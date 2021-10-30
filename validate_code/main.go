package main

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/goki/freetype"
	"github.com/goki/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
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
		log.Fatalf("unmarshal request: %v\n", err)
	}

	getItemResponse, err := ddbClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("TABLE")),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: input.Id},
		},
	})

	if err != nil {
		log.Fatalf("put item: %v\n", err)
	}

	if getItemResponse.Item == nil {
		return Response{
			StatusCode:      404,
			IsBase64Encoded: false,
			Body:            "",
		}, nil
	}

	actualCode := getItemResponse.Item["code"].(*types.AttributeValueMemberS).Value

	var response ValidateCodeResponse
	var status int
	if actualCode != input.Code {
		response = ValidateCodeResponse{
			Message: "INCORRECT",
		}
		status = 400
	} else {
		response = ValidateCodeResponse{
			Message: "OK",
		}
		status = 200
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

func GenerateCode(length int) string {
	charset := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	chars := make([]byte, length)
	for i := range chars {
		chars[i] = charset[rand.Intn(len(charset))]
	}

	return string(chars)
}

func GenerateImage(code string, width, height int) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Pt(0, 0), draw.Src)

	Colorify(img)

	f, err := LoadFont(os.Getenv("FONT"))
	if err != nil {
		return &image.RGBA{}, err
	}

	d := &font.Drawer{
		Dst: img,
		Src: image.Black,
		Face: truetype.NewFace(f, &truetype.Options{
			Size:    40,
			DPI:     100,
			Hinting: font.HintingNone,
		}),
		Dot: fixed.P(img.Bounds().Dx()/10, img.Bounds().Dy()/4*3),
	}

	d.DrawString(code)
	return img, nil
}

func Colorify(img draw.Image) {
	for x := 0; x < img.Bounds().Dx(); x += 10 {
		for y := 0; y < img.Bounds().Dy(); y += 10 {
			randomColor := color.RGBA{byte(rand.Intn(255)), byte(rand.Intn(255)), byte(rand.Intn(255)), 255}
			randomSize := rand.Intn(25)
			draw.Draw(img, image.Rect(x, y, x+randomSize, y+randomSize), image.NewUniform(randomColor), image.Pt(x, y), draw.Src)
		}
	}
}

func LoadFont(name string) (*truetype.Font, error) {
	fontBytes, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}

	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func ListFiles() error {
	files, err := ioutil.ReadDir(".")
	if err != nil {
		return err
	}

	for _, file := range files {
		log.Println(file.Name(), file.IsDir())
	}
	return nil
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
