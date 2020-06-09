package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type clockifyEntry struct {
	Description string `json:"description"`
	ProjectName string `json:"projectName"`
	ClientName  string `json:"clientName"`
	Duration    string `json:"duration"`
}

type invoiceEntry struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	UnitCost float64 `json:"unit_cost"`
}

var ratePerHour float64

func main() {
	ratePerHourPtr := flag.Float64("r", 0, "Rate charged per hour")
	filepathPtr := flag.String("f", "data.json", "Filepath to JSON file of Clockify entries")
	// outputFileNamePtr := flag.String("o", "invoice.pdf", "Desired name for invoice PDF")
	flag.Parse()

	ratePerHour = *ratePerHourPtr

	jsonFile, err := os.Open(*filepathPtr)
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var clockifyEntries []clockifyEntry
	json.Unmarshal(byteValue, &clockifyEntries)

	for i := 0; i < len(clockifyEntries); i++ {
		fmt.Println(buildInvoiceEntry(clockifyEntries[i]))
	}
}

func buildInvoiceEntry(entryData clockifyEntry) invoiceEntry {
	invoiceEntryName := buildInvoiceEntryName(entryData.ClientName, entryData.ProjectName, entryData.Description)
	invoiceQuantity := getInvoiceEntryQuantity(entryData.Duration)
	return invoiceEntry{ Name: invoiceEntryName, Quantity: invoiceQuantity, UnitCost: ratePerHour }
}

func buildInvoiceEntryName(clientName string, projectName string, description string) string {
	return fmt.Sprintf("Client Name: %s | Project Name: %s | Description: %s", clientName, projectName, description)
}

func getInvoiceEntryQuantity(duration string) float64 {
	var hours float64
	var minutes float64
	var seconds float64

	re := regexp.MustCompile(`PT(\d+H)?(\d+M)?(\d+S)?`)
	for _, match := range re.FindStringSubmatch(duration) {
		if (strings.HasPrefix(match, "P")) {
			continue
		} else if (strings.HasSuffix(match, "H")) {
			hours, _ = strconv.ParseFloat(strings.TrimSuffix(match, "H"), 32)
		} else if (strings.HasSuffix(match, "M")) {
			minutes, _ = strconv.ParseFloat(strings.TrimSuffix(match, "M"), 32)
		} else if (strings.HasSuffix(match, "S")) {
			seconds, _ = strconv.ParseFloat(strings.TrimSuffix(match, "S"), 32)
		}
	}
	
	return calculateTotalHours(hours, minutes, seconds)
}

func calculateTotalHours(numHours float64, numMinutes float64, numSeconds float64) float64 {
	if (numSeconds >= 30) {
		numMinutes++
	}
	
	return toFixed(numHours + (numMinutes / 60), 2)
}

func round(num float64) int {
    return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
    output := math.Pow(10, float64(precision))
    return float64(round(num * output)) / output
}
