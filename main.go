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

var sender string
var receiver string
var invoiceNumber float64
var ratePerHour float64
var note string

func main() {
	senderPtr := flag.String("s", "Elliot Alderson", "Name of invoice sender")
	receiverPtr := flag.String("r", "Allsafe Cybersecurity", "Name of invoice receiver")
	invoiceNumberPtr := flag.Float64("n", 1, "Invoice number")
	ratePerHourPtr := flag.Float64("ra", 0, "Rate charged per hour")
	notePtr := flag.String("no", "Thank you for your business", "Additional note provided at the end of invoice")
	filepathPtr := flag.String("f", "data.json", "Filepath to JSON file of Clockify entries")
	outputFileNamePtr := flag.String("o", "invoice.pdf", "Desired name for invoice PDF")
	flag.Parse()

	sender = *senderPtr
	receiver = *receiverPtr
	invoiceNumber = *invoiceNumberPtr
	ratePerHour = *ratePerHourPtr
	note = *notePtr

	jsonFile, err := os.Open(*filepathPtr)
	check(err)
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var clockifyEntries []clockifyEntry
	json.Unmarshal(byteValue, &clockifyEntries)

	var invoiceEntries []invoiceEntry

	for i := 0; i < len(clockifyEntries); i++ {
		invoiceEntries = append(invoiceEntries, buildInvoiceEntry(clockifyEntries[i]))
	}
	
	generateInvoice(buildRequestData(invoiceEntries), *outputFileNamePtr)
}

func check(e error) {
    if e != nil {
        panic(e)
    }
}

func buildInvoiceEntry(entryData clockifyEntry) invoiceEntry {
	invoiceEntryName := buildInvoiceEntryName(entryData.ClientName, entryData.ProjectName, entryData.Description)
	invoiceQuantity := getInvoiceEntryQuantity(entryData.Duration)
	return invoiceEntry{Name: invoiceEntryName, Quantity: invoiceQuantity, UnitCost: ratePerHour}
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
			hours, _ = strconv.ParseFloat(strings.TrimSuffix(match, "H"), 32)
		} else if strings.HasSuffix(match, "M") {
			minutes, _ = strconv.ParseFloat(strings.TrimSuffix(match, "M"), 32)
		} else if strings.HasSuffix(match, "S") {
			seconds, _ = strconv.ParseFloat(strings.TrimSuffix(match, "S"), 32)
		}
	}

	return calculateTotalHours(hours, minutes, seconds)
}

func calculateTotalHours(numHours float64, numMinutes float64, numSeconds float64) float64 {
	if numSeconds >= 30 {
		numMinutes++
	}

	return toFixed(numHours+(numMinutes/60), 2)
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

func buildRequestData(invoiceEntries []invoiceEntry) []byte {
	requestData := invoiceRequestData {From: sender, To: receiver, Number: invoiceNumber, Items: invoiceEntries, Notes: note}

	bytes, err := json.Marshal(requestData)
	check(err)
	
	return bytes
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
