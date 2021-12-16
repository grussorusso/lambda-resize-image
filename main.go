package main

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"golang.org/x/image/draw"
)

var sess = connectAWS()

func connectAWS() *session.Session {
	sess, err := session.NewSession(
		&aws.Config{Region: aws.String("eu-central-1")},
	)

	if err != nil {
		panic(err)
	}
	return sess
}

type MyEvent struct {
	Image  string `json:image`
	Bucket string `json:bucket`
}

func DownloadImage(url, dest string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func Resize(srcFile, dest string) error {
	input, _ := os.Open(srcFile)
	defer input.Close()

	output, _ := os.Create(dest)
	defer output.Close()

	// Decode the image
	src, _, err := image.Decode(input)
	if err != nil {
		return err
	}

	// Set the expected size that you want:
	dst := image.NewRGBA(image.Rect(0, 0, 400, 400))

	// Resize:
	draw.NearestNeighbor.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)

	// Encode to `output`:
	png.Encode(output, dst)

	return nil
}

func Upload(src, dest, bucket string) error {
	input, _ := os.Open(src)
	defer input.Close()

	uploader := s3manager.NewUploader(sess)

	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket), // Bucket to be used
		Key:    aws.String(dest),   // Name of the file to be saved
		Body:   input,              // File
	})

	return err
}

func HandleRequest(ctx context.Context, e MyEvent) (string, error) {
	fmt.Printf("Got a request: %v\n", e)
	basename := path.Base(e.Image)
	destination := fmt.Sprintf("/tmp/%s", basename)
	err := DownloadImage(e.Image, destination)
	if err != nil {
		fmt.Printf("Could not download image: %v\n", err)
		return "", err
	}

	resizedPng := "/tmp/resized.png"
	err = Resize(destination, resizedPng)
	if err != nil {
		fmt.Printf("Resize failed: %v\n", err)
		return "", err
	}

	err = Upload(resizedPng, basename, e.Bucket)
	if err != nil {
		fmt.Printf("Upload failed: %v\n", err)
		return "", err
	}

	return fmt.Sprintf("Done: %s", basename), nil
}

func main() {
	lambda.Start(HandleRequest)
}
