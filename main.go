package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	Total        string `json:"total"`
}

type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

type ReceiptPoints struct {
	ID     string `json:"id"`
	Points int    `json:"points"`
}

var (
	receipts = make(map[string]int)
	mutex    = &sync.Mutex{}
)

func main() {
	r := gin.Default()
	r.POST("/receipts/process", processReceipt)
	r.GET("/receipts/:id/points", getPoints)

	r.Run(":8080")
}

func processReceipt(c *gin.Context) {
	var receipt Receipt
	if err := c.ShouldBindJSON(&receipt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	id := uuid.New().String()
	points := calculatePoints(receipt)

	mutex.Lock()
	receipts[id] = points
	mutex.Unlock()

	c.JSON(http.StatusOK, gin.H{"id": id})
}

func getPoints(c *gin.Context) {
	id := c.Param("id")
	mutex.Lock()
	points, exists := receipts[id]
	mutex.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Receipt ID not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"points": points})
}

func calculatePoints(receipt Receipt) int {
	points := 0

	// 1. One point per alphanumeric character in retailer name
	alphanumeric := regexp.MustCompile("[a-zA-Z0-9]")
	points += len(alphanumeric.FindAllString(receipt.Retailer, -1))

	// 2. 50 points if total is a round dollar
	if total, err := strconv.ParseFloat(receipt.Total, 64); err == nil {
		if total == math.Floor(total) {
			points += 50
		}
		if math.Mod(total, 0.25) == 0 {
			points += 25
		}
		if total > 10.00 {
			points += 5
		}
	}

	// 3. 5 points for every two items
	points += (len(receipt.Items) / 2) * 5

	// 4. Item description rule
	for _, item := range receipt.Items {
		desc := strings.TrimSpace(item.ShortDescription)
		if len(desc)%3 == 0 {
			if price, err := strconv.ParseFloat(item.Price, 64); err == nil {
				points += int(math.Ceil(price * 0.2))
			}
		}
	}

	// 5. 6 points if purchase date is odd
	if date, err := time.Parse("2006-01-02", receipt.PurchaseDate); err == nil {
		if date.Day()%2 != 0 {
			points += 6
		}
	}

	// 6. 10 points if purchase time is between 2:00pm and 4:00pm
	if t, err := time.Parse("15:04", receipt.PurchaseTime); err == nil {
		if t.Hour() == 14 || (t.Hour() == 15 && t.Minute() < 60) {
			points += 10
		}
	}

	return points
}

// Dockerfile
/*
FROM golang:1.19-alpine
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o receipt-processor
CMD ["./receipt-processor"]
EXPOSE 8080
*/
