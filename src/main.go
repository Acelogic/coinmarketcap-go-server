package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/jamespearly/loggly"
)

type CoinList struct {
	Data []struct {
		Name   string `json:"name"`
		Symbol string `json:"symbol"`
		Rank   int    `json:"cmc_rank"`
		Quote  struct {
			USD struct {
				Price              float64 `json:"price"`
				MarketCap          float64 `json:"market_cap"`
				MarketCapDominance float64 `json:"market_cap_dominance"`
			} `json:"USD"`
		} `json:"quote"`
	} `json:"data"`
}
type DBitem struct {
	CoinRank   int     `json:"coinRank"`
	CoinName   string  `json:"coinName"`
	CoinSymbol string  `json:"coinSymbol"`
	CoinPrice  float64 `json:"coinPrice"`
}

func checkEnv() {
	if len(strings.TrimSpace(os.Getenv("AWS_ACCESS_KEY_ID"))) == 0 {
		fmt.Println("\nNO AWS KEY ID LOADED")
	} else {
		fmt.Println("AWS_ACCESS_KEY_ID: " + os.Getenv("AWS_ACCESS_KEY_ID"))

	}

	if len(strings.TrimSpace(os.Getenv("AWS_SECRET_ACCESS_KEY"))) == 0 {
		fmt.Println("\nNO AWS SECRET LOADED")
	} else {
		fmt.Println("AWS_SECRET_ACCESS_KEY: " + os.Getenv("AWS_SECRET_ACCESS_KEY"))

	}

	if len(strings.TrimSpace(os.Getenv("LOGGLY_TOKEN"))) == 0 {
		fmt.Println("\nNO LOGGY TOKEN LOADED")
	} else {
		fmt.Println("LOGGLY TOKEN: " + os.Getenv("LOGGLY_TOKEN"))
	}

	if len(strings.TrimSpace(os.Getenv("COINMARKETCAP_API_KEY"))) == 0 {
		fmt.Println("\nNO COINMARKETCAP_API_KEY LOADED")
	} else {
		fmt.Println("COINMARKETCAP_API_KEY: " + os.Getenv("COINMARKETCAP_API_KEY"))
	}
}

// Return a populated struct of the top 100 coins from CoinMarketCap
// while providing logging, printing, and caching of the results
func fetchCoinAPI() CoinList {
	// Loading in Enviroment Variables
	apiKey := os.Getenv("COINMARKETCAP_API_KEY")
	url := "https://pro-api.coinmarketcap.com/v1/cryptocurrency/listings/latest"

	//Create a Request to the API
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Set API Key
	req.Header.Set("X-CMC_PRO_API_KEY", apiKey)

	// Create a HTTP Client to Execute the Request and generate a response
	coinClient := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}
	// Execute the Request and generate a response
	res, getErr := coinClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	//Read the response
	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	// Unmarshall the response into a coinList structure
	coinList := &CoinList{}
	json.Unmarshal(body, coinList)

	// Establish Connection to Loggly API
	client := loggly.New("CoinAPI")

	// print out the coinlist and sort by rank and display price in USD with reduced precision
	for i := 0; i < len(coinList.Data); i++ {
		fmt.Printf("%d. %s (%s) - %.4f USD\n", coinList.Data[i].Rank, coinList.Data[i].Name, coinList.Data[i].Symbol, coinList.Data[i].Quote.USD.Price)
		//Store in a string
		coinString := fmt.Sprintf("%d. %s (%s) - %.4f USD", coinList.Data[i].Rank, coinList.Data[i].Name, coinList.Data[i].Symbol, coinList.Data[i].Quote.USD.Price)
		//Send to loggly
		err := client.EchoSend("info", coinString)
		fmt.Println("err", err)
	}

	// Serialize Json and Write the json to a file in PrettyPrint form
	prettyJSON, _ := json.MarshalIndent(coinList, "", " ")
	err = ioutil.WriteFile("coinlist.json", prettyJSON, 0644)
	fmt.Print(err)

	// Return Fetched CoinList
	return *coinList

}

func addItemtoDB(item DBitem) {
	// Start AWS Session (Env variables must be provided before hand)
	//sess := session.Must(session.NewSessionWithOptions(session.Options{SharedConfigState: session.SharedConfigEnable}))
	
	// Start AWS Session in us east 
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String("us-east-1")}))


	// Create DynamoDB client
	svc := dynamodb.New(sess)
	_ = svc

	// Process Item Input
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		fmt.Println("Got error marshalling item:")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Add Item to Table
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String("mcruz-CoinMarketCap"),
	}

	_, err = svc.PutItem(input)

	if err != nil {
		fmt.Println("Got error calling addItemtoDB: ")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	
	// Map struct contents to string
	itemString, _ := json.Marshal(item)
	
	// Loggly
	client := loggly.New("CoinAPI")
	
	// Send to Loggly
	fmt.Println("Added Item to DB")
	err2 := client.EchoSend("info", "Added item to DB:" + string(itemString))
	fmt.Println("err", err2)

}

func pollAndStore(PollingRateSecs time.Duration) {
	time.Sleep(PollingRateSecs * time.Second)
	fmt.Println("Debug: POLLING NOW")

	// Fetch the CoinList
	coinList := fetchCoinAPI()

	// Loop through the CoinList and add each item to the DB
	for i := 0; i < len(coinList.Data); i++ {
		// Create a DBitem
		item := DBitem{
			CoinRank:        coinList.Data[i].Rank,
			CoinName:        coinList.Data[i].Name,
			CoinSymbol:      coinList.Data[i].Symbol,
			CoinPrice:       coinList.Data[i].Quote.USD.Price,
		}
		// Add to DB
		addItemtoDB(item)
	}
	fmt.Println("Debug: POLLING COMPLETE")
}

func main() {
	checkEnv()
	for {
		// 15 mins = 900 seconds
		// 6 hours = 21600 seconds
		pollAndStore(21600)
	}
}
