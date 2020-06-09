package main

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

type configObj struct {
	Sender string `json:"sender"`
	Receiver string `json:"receiver"`
	InvoiceNumber float64 `json:"invoiceNumber"`
	RatePerHour float64 `json:"ratePerHour"`
	Name string `json:"name"`
	Email string `json:"email"`
	Address string `json:"address"`
	InvoicePeriod string `json:"invoicePeriod"`
	Notes string `json:"notes"`
	InvoiceDataFilePath string `json:"invoiceDataFilePath"`
	OutPutFilePath string `json:"outputFilePath"`
}

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

type invoiceRequestData struct {
	From string `json:"from"`
	To string `json:"to"`
	Number float64 `json:"number"`
	Items []invoiceEntry `json:"items"`
	Notes string `json:"notes"`
}

const INVOICE_GENERATOR_URL = "https://invoice-generator.com"

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

func check(e error) {
    if e != nil {
        panic(e)
    }
}

func parseConfigFile(filePath string) {
	jsonFile, err := os.Open(filePath)
	check(err)
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &config)
}

func parseDataFile(filePath string) {
	jsonFile, err := os.Open(filePath)
	check(err)
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &clockifyEntries)
}

func buildInvoiceEntry(entryData clockifyEntry) invoiceEntry {
	invoiceEntryName := buildInvoiceEntryName(entryData.ClientName, entryData.ProjectName, entryData.Description)
	invoiceQuantity := getInvoiceEntryQuantity(entryData.Duration)
	return invoiceEntry{Name: invoiceEntryName, Quantity: invoiceQuantity, UnitCost: config.RatePerHour}
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

func calculateHours(numHours float64, numMinutes float64, numSeconds float64) float64 {
	if numSeconds > 0 {
		numMinutes++
	}
	return numHours + toFixed(numMinutes/60, 1)
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

func buildRequestData(invoiceEntries []invoiceEntry) []byte {
	requestData := invoiceRequestData {From: config.Sender, To: config.Receiver, Number: config.InvoiceNumber, Items: invoiceEntries, Notes: buildNotes()}

	bytes, err := json.Marshal(requestData)
	check(err)
	return bytes
}

func buildNotes() string {
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n", config.Name, config.Email, config.Address, config.InvoicePeriod, config.Notes)
}

func generateInvoice(requestData []byte, outputFileName string) {
	req, err := http.NewRequest("POST", INVOICE_GENERATOR_URL, bytes.NewBuffer(requestData))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	check(err)
	defer resp.Body.Close()
	
	if (resp.Status == "200 OK") {
		body, _ := ioutil.ReadAll(resp.Body)
		createFile(body, outputFileName)
	}
}

func createFile(data []byte, filePath string) {
	f, err := os.Create(filePath)
	defer f.Close()
	check(err)
	f.Write(data)
	fmt.Println("Invoice generated successfully")
}
