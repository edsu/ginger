package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/eikeon/dynamodb"
	"github.com/eikeon/ginger"
)

func main() {
	ginger.NewMemoryGinger(true)

	ec := json.NewEncoder(os.Stdout)

	count := 0
	var last dynamodb.Key
	for {
		if qr, err := ginger.DB.Scan("fetch", &dynamodb.ScanOptions{ReturnConsumedCapacity: "TOTAL", ExclusiveStartKey: last}); err == nil {
			count += qr.Count
			for _, i := range qr.Items {
				f := ginger.DB.FromItem("fetch", i)
				if err = ec.Encode(&f); err != nil {
					panic(err)
				}
			}
			last = qr.LastEvaluatedKey
			if last == nil {
				break
			}
		} else {
			log.Println("query error:", err)
		}
	}
	log.Printf("Count: %d\n", count)

}
