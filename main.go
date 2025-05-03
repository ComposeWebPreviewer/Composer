package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	decodedCode, err := decodeCode(os.Args[1])
	if err != nil {
		fmt.Printf(err.Error())
		panic(err)
	}

	err = buildComposableBinaries(decodedCode)
	if err != nil {
		fmt.Printf(err.Error())
		panic(err)
	}

	err = uploadToS3()
	if err != nil {
		fmt.Printf(err.Error())
		panic(err)
	}
}

func decodeCode(encodedCode string) (string, error) {
	decodedCode, err := base64.StdEncoding.DecodeString(encodedCode)
	if err != nil {
		return "", err
	}
	fmt.Println("Decoded Code")
	fmt.Println(string(decodedCode))

	return string(decodedCode), nil
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

	_, err := cmd.Output()
	if err != nil {
		return err
	}

	fmt.Println("WASM generated")

	return nil
}

func uploadToS3() error {
	bucket := os.Getenv("S3_BUCKET")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg)

	matches, _ := filepath.Glob("/tmp/composeApp/build/kotlin-webpack/wasmJs/productionExecutable/*.wasm")
	var waitGroup sync.WaitGroup
	for _, fileName := range matches {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()

			file, _ := os.Open(fileName)
			_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
				Bucket:      aws.String(bucket),
				Key:         aws.String(fileName),
				Body:        file,
				IfNoneMatch: aws.String("*"),
			})
			if err != nil {
				fmt.Println("failed to upload")
				fmt.Println(err)
			}

			fmt.Println("Upload successful!")
		}()
	}

	waitGroup.Wait()

	return nil
}
