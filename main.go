package main

// BUG(spacesailor24) calculateHours method does not match total time calculation done by Clockify.

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// configObj used to populate PDF with non-standard values.
type configObj struct {
	Sender              string  `json:"sender"`
	Receiver            string  `json:"receiver"`
	InvoiceNumber       float64 `json:"invoiceNumber"`
	RatePerHour         float64 `json:"ratePerHour"`
	Name                string  `json:"name"`
	Email               string  `json:"email"`
	Address             string  `json:"address"`
	InvoicePeriod       string  `json:"invoicePeriod"`
	Notes               string  `json:"notes"`
	InvoiceDataFilePath string  `json:"invoiceDataFilePath"`
	OutPutFilePath      string  `json:"outputFilePath"`
}

// clockifyEntry needed values from Clockify's response data to construct invoiceEntry.
type clockifyEntry struct {
	Description string `json:"description"`
	ProjectName string `json:"projectName"`
	ClientName  string `json:"clientName"`
	Duration    string `json:"duration"`
}

// invoiceEntry needed values to construct Clockify time entries.
type invoiceEntry struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	UnitCost float64 `json:"unit_cost"`
}

// invoiceRequestData data being sent to InvoiceGeneratorURL to generate PDF.
type invoiceRequestData struct {
	From   string         `json:"from"`
	To     string         `json:"to"`
	Number float64        `json:"number"`
	Items  []invoiceEntry `json:"items"`
	Notes  string         `json:"notes"`
}

// invoiceGeneratorURL URL invoiceRequestData is sent to, to generate PDF.
const invoiceGeneratorURL = "https://invoice-generator.com"

var config configObj
var clockifyEntries []clockifyEntry

func main() {
	configPathPtr := flag.String("c", "config.json", "Config file for personal invoice details")
	flag.Parse()

	parseConfigFile(*configPathPtr)
	parseDataFile(config.InvoiceDataFilePath)

	var invoiceEntries []invoiceEntry

	for i := 0; i < len(clockifyEntries); i++ {
		invoiceEntries = append(invoiceEntries, buildInvoiceEntry(clockifyEntries[i]))
	}

	generateInvoice(buildRequestData(invoiceEntries), config.OutPutFilePath)
}

// check if error is found, panics.
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// parseConfigFile reads filePath and unmarshals into config type object.
func parseConfigFile(filePath string) {
	var bytesFile = getFileAsBytes(filePath)
	json.Unmarshal(bytesFile, &config)
}

// parseDataFile reads filePath and unmarshals into clockifyEntries type object.
func parseDataFile(filePath string) {
	var bytesFile = getFileAsBytes(filePath)
	json.Unmarshal(bytesFile, &clockifyEntries)
}

// getFileAsBytes reads filePath and returns bytes value.
func getFileAsBytes(filePath string) []byte {
	jsonFile, err := os.Open(filePath)
	check(err)
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	check(err)
	return byteValue
}

// buildInvoiceEntry uses provided data to generate a single invoiceEntry that will be displayed in generated invoice.
func buildInvoiceEntry(entryData clockifyEntry) invoiceEntry {
	invoiceEntryName := buildInvoiceEntryName(entryData.ClientName, entryData.ProjectName, entryData.Description)
	invoiceQuantity := getInvoiceEntryQuantity(entryData.Duration)
	return invoiceEntry{Name: invoiceEntryName, Quantity: invoiceQuantity, UnitCost: config.RatePerHour}
}

// buildInvoiceEntryName returns formatted string used to represent what was worked on.
// The format of the string is generic and can be changed to anything.
func buildInvoiceEntryName(clientName string, projectName string, description string) string {
	return fmt.Sprintf("Client Name: %s | Project Name: %s | Description: %s", clientName, projectName, description)
}

// getInvoiceEntryQuantity parses a Clockify formatted duration string (in the format of PT1H1M1S) into
// each individual time segment.
// Returns total amount of hours worked.
func getInvoiceEntryQuantity(duration string) float64 {
	var hours float64
	var minutes float64
	var seconds float64

	re := regexp.MustCompile(`PT(\d+H)?(\d+M)?(\d+S)?`)
	for _, match := range re.FindStringSubmatch(duration) {
		if strings.HasPrefix(match, "P") {
			continue
		} else if strings.HasSuffix(match, "H") {
			hours, _ = strconv.ParseFloat(strings.TrimSuffix(match, "H"), 64)
		} else if strings.HasSuffix(match, "M") {
			minutes, _ = strconv.ParseFloat(strings.TrimSuffix(match, "M"), 64)
		} else if strings.HasSuffix(match, "S") {
			seconds, _ = strconv.ParseFloat(strings.TrimSuffix(match, "S"), 64)
		}
	}

	return calculateHours(hours, minutes, seconds)
}

// calculateHours uses provided data to calculate the amount of hours worked.
// Rounded to nearest 1/10th of an hour.
func calculateHours(numHours float64, numMinutes float64, numSeconds float64) float64 {
	if numSeconds > 0 {
		numMinutes++
	}
	return numHours + toFixed(numMinutes/60, 1)
}

// round Helper function to round float64s.
func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

// toFixed Rounds a number to given number of decimals.
func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

// buildRequestData Builds need JSON object to submit to invoice generater.
func buildRequestData(invoiceEntries []invoiceEntry) []byte {
	requestData := invoiceRequestData{From: config.Sender, To: config.Receiver, Number: config.InvoiceNumber, Items: invoiceEntries, Notes: buildNotes()}

	bytes, err := json.Marshal(requestData)
	check(err)
	return bytes
}

// buildNotes Uses configObj to build notes section of invoice.
func buildNotes() string {
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n", config.Name, config.Email, config.Address, config.InvoicePeriod, config.Notes)
}

// generateInvoice Takes requestData and submits data to invoiceGeneratorURL and save PDF to outputFileName.
func generateInvoice(requestData []byte, outputFileName string) {
	req, err := http.NewRequest("POST", invoiceGeneratorURL, bytes.NewBuffer(requestData))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	check(err)
	defer resp.Body.Close()

	if resp.Status == "200 OK" {
		body, _ := ioutil.ReadAll(resp.Body)
		createFile(body, outputFileName)
	}
}

// createFile Write data to filePath.
func createFile(data []byte, filePath string) {
	f, err := os.Create(filePath)
	defer f.Close()
	check(err)
	f.Write(data)
	fmt.Println("Invoice generated successfully")
}
