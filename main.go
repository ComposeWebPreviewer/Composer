package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var codespaceTableName = os.Getenv("CODESPACEDB_TABLE_NAME")

func main() {
	decodedCode, err := decodeCode(os.Args[2])
	if err != nil {
		fmt.Printf(err.Error())
		panic(err)
	}

	err = buildComposableBinaries(decodedCode)
	if err != nil {
		fmt.Printf(err.Error())
		panic(err)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Printf("Unable to load SDK config, %v", err)
		panic(err)
	}

	wasmFilePath := wasmFilePath()

	var waitGroup sync.WaitGroup
	waitGroup.Add(2)
	go func() {
		defer waitGroup.Done()
		uploadToS3(cfg, wasmFilePath)
	}()
	go func() {
		defer waitGroup.Done()
		writeDataToCodespaceDB(cfg, os.Args[1], os.Args[2], wasmFilePath)
	}()
	waitGroup.Wait()
}

func decodeCode(encodedCode string) (string, error) {
	decodedCode, err := base64.StdEncoding.DecodeString(encodedCode)
	if err != nil {
		return "", err
	}
	fmt.Println("Decoded Code")

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

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	_, err := cmd.Output()
	if err != nil {
		return errors.New(stderr.String())
	}

	fmt.Println("WASM generated")

	return nil
}

func wasmFilePath() string {
	matches, _ := filepath.Glob("/tmp/composeApp/build/kotlin-webpack/wasmJs/productionExecutable/*.wasm")
	leastSizeMatch := matches[0]
	for _, filePath := range matches {
		leastSizeMatchFileInfo, _ := os.Stat(leastSizeMatch)
		currFileInfo, _ := os.Stat(filePath)

		if currFileInfo.Size() < leastSizeMatchFileInfo.Size() {
			leastSizeMatch = filePath
		}
	}

	return leastSizeMatch
}

func uploadToS3(cfg aws.Config, wasmFilePath string) {
	bucket := os.Getenv("S3_BUCKET")

	client := s3.NewFromConfig(cfg)

	file, _ := os.Open(wasmFilePath)
	filename := filepath.Base(wasmFilePath)
	_, err := client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key: aws.String(filename),
		Body: file,
		IfNoneMatch: aws.String("*"),
	})

	if err == nil {
		fmt.Println("Upload successful")
	}
}

func writeDataToCodespaceDB(cfg aws.Config, id string, code string, wasmFilePath string) {
	client := dynamodb.NewFromConfig(cfg)

	_, err := client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(codespaceTableName),
		Item: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
			"code": &types.AttributeValueMemberS{Value: code},
			"wasm": &types.AttributeValueMemberS{Value: filepath.Base(wasmFilePath)},
		},
	})

	if err == nil {
		fmt.Println("Write to codebase db done")
	}
}
