package logs

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestLoggingToFile(t *testing.T) {
	logW, logFile, _ := runLogForwarding(t)
	defer os.Remove(logFile)
	defer logW.Close()

	logToLogWriter(t, logW, `{"level": "info","msg":"kitten"}`)

	logFileContent := waitForLogContent(t, logFile)
	if !strings.Contains(string(logFileContent), "kitten") {
		t.Fatalf("%s does not contain kitten", string(logFileContent))
	}
}

func TestLogForwardingDoesNotStopOnJsonDecodeErr(t *testing.T) {
	logW, logFile, _ := runLogForwarding(t)
	defer os.Remove(logFile)
	defer logW.Close()

	logToLogWriter(t, logW, "invalid-json-with-kitten")

	logFileContent := waitForLogContent(t, logFile)
	if !strings.Contains(string(logFileContent), "failed to decode") {
		t.Fatalf("%q does not contain decoding error", string(logFileContent))
	}

	truncateLogFile(t, logFile)

	logToLogWriter(t, logW, `{"level": "info","msg":"puppy"}`)

	logFileContent = waitForLogContent(t, logFile)
	if !strings.Contains(string(logFileContent), "puppy") {
		t.Fatalf("%s does not contain puppy", string(logFileContent))
	}
}

func TestLogForwardingDoesNotStopOnLogLevelParsingErr(t *testing.T) {
	logW, logFile, _ := runLogForwarding(t)
	defer os.Remove(logFile)
	defer logW.Close()

	logToLogWriter(t, logW, `{"level": "alert","msg":"puppy"}`)

	logFileContent := waitForLogContent(t, logFile)
	if !strings.Contains(string(logFileContent), "failed to parse log level") {
		t.Fatalf("%q does not contain log level parsing error", string(logFileContent))
	}

	truncateLogFile(t, logFile)

	logToLogWriter(t, logW, `{"level": "info","msg":"puppy"}`)

	logFileContent = waitForLogContent(t, logFile)
	if !strings.Contains(string(logFileContent), "puppy") {
		t.Fatalf("%s does not contain puppy", string(logFileContent))
	}
}

func TestLogForwardingStopsAfterClosingTheWriter(t *testing.T) {
	logW, logFile, doneForwarding := runLogForwarding(t)
	defer os.Remove(logFile)

	logToLogWriter(t, logW, `{"level": "info","msg":"sync"}`)

	logFileContent := waitForLogContent(t, logFile)
	if !strings.Contains(string(logFileContent), "sync") {
		t.Fatalf("%q does not contain sync message", string(logFileContent))
	}

	logW.Close()
	select {
	case <-doneForwarding:
	case <-time.After(10 * time.Second):
		t.Fatal("log forwarding did not stop after closing the pipe")
	}
}

func logToLogWriter(t *testing.T, logW *os.File, message string) {
	_, err := logW.Write([]byte(message + "\n"))
	if err != nil {
		t.Fatalf("failed to write %q to log writer: %v", message, err)
	}
}

func runLogForwarding(t *testing.T) (*os.File, string, chan struct{}) {
	logR, logW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	logFile := tempFile.Name()

	logConfig := Config{LogLevel: logrus.InfoLevel, LogFormat: "json", LogFilePath: logFile}
	return logW, logFile, startLogForwarding(t, logConfig, logR)
}

func startLogForwarding(t *testing.T, logConfig Config, logR *os.File) chan struct{} {
	loggingConfigured = false
	if err := ConfigureLogging(logConfig); err != nil {
		t.Fatal(err)
	}
	doneForwarding := make(chan struct{})
	go func() {
		ForwardLogs(logR)
		close(doneForwarding)
	}()
	return doneForwarding
}

func waitForLogContent(t *testing.T, logFile string) string {
	startTime := time.Now()

	for {
		if time.Now().After(startTime.Add(10 * time.Second)) {
			t.Fatal(errors.New("No content in log file after 10 seconds"))
			break
		}

		fileContent, err := ioutil.ReadFile(logFile)
		if err != nil {
			t.Fatal(err)
		}
		if len(fileContent) == 0 {
			continue
		}
		return string(fileContent)
	}

	return ""
}

func truncateLogFile(t *testing.T, logFile string) {
	file, err := os.OpenFile(logFile, os.O_RDWR, 0666)
	if err != nil {
		t.Fatalf("failed to open log file: %v", err)
		return
	}
	defer file.Close()

	err = file.Truncate(0)
	if err != nil {
		t.Fatalf("failed to truncate log file: %v", err)
	}
}
