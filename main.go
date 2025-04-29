package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	encodedCode := os.Args[1]

	decodedCode, _ := base64.StdEncoding.DecodeString(encodedCode)
	fmt.Println("Decoded Code")
	fmt.Println(string(decodedCode))

	buildComposableBinaries(string(decodedCode))
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
