package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type I3Entry struct {
	Name     string `json:"name"`
	Instance string `json:"instance"`
	Markup   string `json:"markup"`
	FullText string `json:"full_text"`
}

const (
	rxPath = "/sys/class/net/enp4s0/statistics/rx_bytes"
	txPath = "/sys/class/net/enp4s0/statistics/tx_bytes"
)

var reader *bufio.Reader
var lastTime time.Time
var lastRxBytes int
var lastTxBytes int

// Initial output of program before main loop
//
// {"version":1}
// [
func processHeader() {
	// Preprocessing
	preProcessLines := 2
	for i := 0; i < preProcessLines; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(line)
	}
}

func getEvent() string {
	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return line
}

func parseEvent(event string) []*I3Entry {
	var events []*I3Entry
	bytes := []byte(event)
	err := json.Unmarshal(bytes, &events)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return events
}

func getNetworkBytes(path string) int {
	contents, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	bytesStr := strings.TrimRight(string(contents), "\n")

	bytes, err := strconv.Atoi(bytesStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return bytes
}

func convertBytesToBitsPerSecond(bps float64) float64 {
	mbps := bps * 8.0 / 1000000.0
	return mbps
}

func updateNetworkStats(rxBytes int, txBytes int, time time.Time) {
	lastTime = time
	lastRxBytes = rxBytes
	lastTxBytes = txBytes
}

func getTimeDuration(now time.Time) time.Duration {
	duration := now.Sub(lastTime)
	return duration
}

func calculateNewBytesTransferred(rxBytes int, txBytes int) (int, int) {
	rx := rxBytes - lastRxBytes
	tx := txBytes - lastTxBytes

	return rx, tx
}

func calculateBytesPerSecond(bytes int, seconds float64) float64 {
	return float64(bytes) / seconds
}

func getMbps() (float64, float64) {
	now := time.Now()
	duration := getTimeDuration(now)

	rxBytes := getNetworkBytes(rxPath)
	txBytes := getNetworkBytes(txPath)

	rxMbps := 0.0
	txMbps := 0.0
	if duration.Seconds() > 0 {
		rx, tx := calculateNewBytesTransferred(rxBytes, txBytes)
		rxBytesPerSecond := calculateBytesPerSecond(rx, duration.Seconds())
		txBytesPerSecond := calculateBytesPerSecond(tx, duration.Seconds())
		rxMbps = convertBytesToBitsPerSecond(rxBytesPerSecond)
		txMbps = convertBytesToBitsPerSecond(txBytesPerSecond)
	}

	updateNetworkStats(rxBytes, txBytes, now)

	return rxMbps, txMbps
}

func generateOutput(prefix string, suffix string, i3Entries *[]I3Entry) string {
	output := prefix
	for _, entry := range *i3Entries {
		e, err := json.Marshal(entry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		output = fmt.Sprintf("%s%s,", output, string(e))
	}
	// Remove the last comma
	output = strings.TrimRight(output, ",")
	output = fmt.Sprintf("%s%s", output, suffix)
	return output
}

func generateI3Entries(entries *[]*I3Entry) []I3Entry {
	i3Entries := []I3Entry{}

	for _, entry := range *entries {
		if entry.Name == "ethernet" {
			rxMbps, txMbps := getMbps()
			fullText := fmt.Sprintf("R: %0.2f T: %0.2f (Mbit/s)", rxMbps, txMbps)
			netSpeed := I3Entry{
				FullText: fullText,
			}
			i3Entries = append(i3Entries, netSpeed)
		}
		i3Entries = append(i3Entries, *entry)
	}

	return i3Entries
}

func processFirstEvent() {
	// First entry
	// [{"name":"memory","markup":"none","full_text":"Mem: 3.1 GiB / 31.1 GiB"},{"name":"load","markup":"none","full_text":"CPU: 0.32"},{"name":"cpu_temperature","instance":"/sys/devices/platform/coretemp.0/hwmon/hwmon2/temp1_input","markup":"none","full_text":"T: 26 °C"},{"name":"ethernet","instance":"enp4s0","color":"#00FF00","markup":"none","full_text":"E: 192.168.1.150 (1000 Mbit/s)"},{"name":"tztime","instance":"local","markup":"none","full_text":"2022-10-21 20:15:46"}]
	event := getEvent()
	events := parseEvent(event)
	i3Entries := generateI3Entries(&events)
	output := generateOutput("[", "]", &i3Entries)
	fmt.Println(output)
}

func init() {
	reader = bufio.NewReader(os.Stdin)
	lastTime = time.Now()
	lastRxBytes = getNetworkBytes(rxPath)
	lastTxBytes = getNetworkBytes(txPath)
}

func main() {
	processHeader()
	processFirstEvent()

	// Main loop
	for {
		// Get i3Status input
		event := getEvent()
		// ,[{"name":"memory","markup":"none","full_text":"Mem: 3.6 GiB / 31.1 GiB"},{"name":"load","markup":"none","full_text":"CPU: 0.45"},{"name":"cpu_temperature","instance":"/sys/devices/platform/coretemp.0/hwmon/hwmon2/temp1_input","markup":"none","full_text":"T: 26 °C"},{"name":"ethernet","instance":"enp4s0","color":"#00FF00","markup":"none","full_text":"E: 192.168.1.150 (1000 Mbit/s)"},{"name":"tztime","instance":"local","markup":"none","full_text":"2022-10-21 21:41:20"}]
		event = strings.TrimLeft(event, ",")
		events := parseEvent(event)
		i3Entries := generateI3Entries(&events)
		output := generateOutput(",[", "]", &i3Entries)
		fmt.Println(output)
	}
}
