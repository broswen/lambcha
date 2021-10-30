package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/goki/freetype"
	"github.com/goki/freetype/truetype"
	"github.com/segmentio/ksuid"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

var ddbClient *dynamodb.Client
var s3Client *s3.Client

type Response events.APIGatewayProxyResponse

type GenerateCodeResponse struct {
	Id       string `json:"id"`
	ImageUrl string `json:"imageUrl"`
}

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context) (Response, error) {

	id, err := ksuid.NewRandom()
	if err != nil {
		log.Fatalf("generate ksuid: %v\n", err)
	}

	code := GenerateCode(6)
	// because the font only has uppercase with different styles
	upperCode := strings.ToUpper(code)

	img, err := GenerateImage(code, 200, 100)
	if err != nil {
		log.Fatalf("generate image: %v\n", err)
	}

	buf := new(bytes.Buffer)

	err = png.Encode(buf, img)
	if err != nil {
		log.Fatalf("png encode: %v\n", err)
	}

	_, err = s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(os.Getenv("BUCKET")),
		Key:         aws.String(id.String()),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("image/png"),
	})
	if err != nil {
		log.Fatalf("put object: %v\n", err)
	}

	imageUrl := fmt.Sprintf("%s/%s", os.Getenv("BUCKET_DOMAIN"), id.String())

	_, err = ddbClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("TABLE")),
		Item: map[string]types.AttributeValue{
			"PK":       &types.AttributeValueMemberS{Value: id.String()},
			"code":     &types.AttributeValueMemberS{Value: upperCode},
			"imageUrl": &types.AttributeValueMemberS{Value: imageUrl},
			"TTL":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", time.Now().Unix()+int64(60))},
		},
	})

	if err != nil {
		log.Fatalf("put item: %v\n", err)
	}

	response := GenerateCodeResponse{
		Id:       id.String(),
		ImageUrl: imageUrl,
	}

	body, err := json.Marshal(response)
	if err != nil {
		return Response{StatusCode: 500}, err
	}
	resp := Response{
		StatusCode:      200,
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

	s3Client = s3.NewFromConfig(cfg)
	ddbClient = dynamodb.NewFromConfig(cfg)

	rand.Seed(time.Now().UnixNano())
}

func main() {
	lambda.Start(Handler)
}
