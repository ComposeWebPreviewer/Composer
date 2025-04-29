package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	uploadToS3()
}

func decodeCode(encodedCode string) string {
	decodedCode, _ := base64.StdEncoding.DecodeString(encodedCode)
	fmt.Println("Decoded Code")
	fmt.Println(string(decodedCode))

	return string(decodedCode)
}

func buildComposableBinaries(snippet string) error {
    code := []byte("package compose.builder\n" + snippet)
    os.WriteFile(
        "/tmp/composeApp/src/wasmJsMain/kotlin/compose/builder/Composable.kt",
        code,
        0644,
    )

    cmd := exec.Command("./gradlew", "wasmJsBrowserDistribution")
    cmd.Dir = "/tmp"

    stdout, err := cmd.Output()
    if err != nil {
        return err
    }

    fmt.Println(string(stdout))

	return nil
}

func uploadToS3() {
	bucket := os.Getenv("S3_BUCKET")
	key := "example.txt"
	content := "Hello from ECS task!"

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Println("Unable to load SDK config")
	}

	client := s3.NewFromConfig(cfg)

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader([]byte(content)),
	})
	if err != nil {
		fmt.Println("failed to upload")
		fmt.Println(err)
	}

	fmt.Println("Upload successful!")
}
