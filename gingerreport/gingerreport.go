package main

import (
	"log"

	"github.com/eikeon/dynamodb"
	"github.com/eikeon/ginger"
)

func main() {
	ginger.NewMemoryGinger(true)

	for _, code := range []string{"200", "301", "302", "404"} {
		count := 0
		f := dynamodb.KeyConditions{"StatusCode": {[]dynamodb.AttributeValue{{"N": code}}, "EQ"}}
		var last dynamodb.Key
		for {
			if qr, err := ginger.DB.Scan("fetch", &dynamodb.ScanOptions{ScanFilter: f, ReturnConsumedCapacity: "TOTAL", Select: "COUNT", ExclusiveStartKey: last}); err == nil {
				count += qr.Count
				last = qr.LastEvaluatedKey
				if last == nil {
					break
				}
			} else {
				log.Println("query error:", err)
			}
		}
		log.Printf("Status Code: %s Total Count: %d\n", code, count)
	}
}
