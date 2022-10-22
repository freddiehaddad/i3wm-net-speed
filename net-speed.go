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
func preProcess() {
	// Preprocessing
	preProcessLines := 2
	for i := 0; i < preProcessLines; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf(line)
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

func convertBytesToBitsPerSecond(bytesPerSecond float64, duration float64) float64 {
	mbps := bytesPerSecond * 8 / float64(1000000) / duration
	return mbps
}

func updateValues(rxBytes int, txBytes int, time time.Time) {
	lastTime = time
	lastRxBytes = rxBytes
	lastTxBytes = txBytes
}

func getMbps() (float64, float64) {
	now := time.Now()

	rxBytes := getNetworkBytes(rxPath)
	txBytes := getNetworkBytes(txPath)

	elapsed := now.Sub(lastTime)

	rxMbps := 0.0
	txMbps := 0.0
	if elapsed.Seconds() > 0 {
		rxMbps = convertBytesToBitsPerSecond(float64(rxBytes-lastRxBytes), elapsed.Seconds())
		txMbps = convertBytesToBitsPerSecond(float64(txBytes-lastTxBytes), elapsed.Seconds())
	}

	updateValues(rxBytes, txBytes, now)

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

func init() {
	reader = bufio.NewReader(os.Stdin)
	lastTime = time.Now()
	lastRxBytes = getNetworkBytes(rxPath)
	lastTxBytes = getNetworkBytes(txPath)
}

func main() {
	preProcess()

	// First entry
	// [{"name":"memory","markup":"none","full_text":"Mem: 3.1 GiB / 31.1 GiB"},{"name":"load","markup":"none","full_text":"CPU: 0.32"},{"name":"cpu_temperature","instance":"/sys/devices/platform/coretemp.0/hwmon/hwmon2/temp1_input","markup":"none","full_text":"T: 26 °C"},{"name":"ethernet","instance":"enp4s0","color":"#00FF00","markup":"none","full_text":"E: 192.168.1.150 (1000 Mbit/s)"},{"name":"tztime","instance":"local","markup":"none","full_text":"2022-10-21 20:15:46"}]
	event := getEvent()
	events := parseEvent(event)

	i3Entries := []I3Entry{}
	for _, entry := range events {
		if entry.Name == "ethernet" {
			netSpeed := I3Entry{
				FullText: "R: 0.0 T: 0.0 (Mbit/s)",
			}
			i3Entries = append(i3Entries, netSpeed)
		}
		i3Entries = append(i3Entries, *entry)
	}

	output := generateOutput("[", "]", &i3Entries)
	fmt.Println(output)

	// Main loop
	for {
		// Get i3Status input
		event = getEvent()
		// ,[{"name":"memory","markup":"none","full_text":"Mem: 3.6 GiB / 31.1 GiB"},{"name":"load","markup":"none","full_text":"CPU: 0.45"},{"name":"cpu_temperature","instance":"/sys/devices/platform/coretemp.0/hwmon/hwmon2/temp1_input","markup":"none","full_text":"T: 26 °C"},{"name":"ethernet","instance":"enp4s0","color":"#00FF00","markup":"none","full_text":"E: 192.168.1.150 (1000 Mbit/s)"},{"name":"tztime","instance":"local","markup":"none","full_text":"2022-10-21 21:41:20"}]
		event = strings.TrimLeft(event, ",")
		events = parseEvent(event)

		i3Entries := []I3Entry{}
		for _, entry := range events {
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

		output := generateOutput(",[", "]", &i3Entries)
		fmt.Println(output)
	}
}
