package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func poll(){ 
	url := "https://pro-api.coinmarketcap.com/v1/cryptocurrency/listings/latest"
  
  coinClient := http.Client{
	Timeout: time.Second * 2, // Timeout after 2 seconds
}
  req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Get Enviroment Variables For API Key
	apiKey := os.Getenv("COINMARKETCAP_API_KEY")

	//Make a request to the api
	req, err = http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	//Set the api key in the header
	req.Header.Set("X-CMC_PRO_API_KEY", apiKey)

	res, getErr := coinClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	//Read the response
	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

    // Creating a strcut to hold the data from the response body	
	type CoinList struct { 
		Data []struct {
			Name string `json:"name"`
			Symbol string `json:"symbol"`
			Rank int `json:"cmc_rank"`
			Quote struct {
				USD struct {
					Price float64 `json:"price"`
					MarketCap float64 `json:"market_cap"`
					MarketCapDominance float64 `json:"market_cap_dominance"`
				} `json:"USD"`
			} `json:"quote"`
		} `json:"data"`	
	}

	// print out the coinlist and sort by rank and display price in USD with reduced precision 
	coinList := &CoinList{}
	json.Unmarshal(body, coinList)
	for i := 0; i < len(coinList.Data); i++ {
		fmt.Printf("%d. %s (%s) - %.4f USD\n", coinList.Data[i].Rank, coinList.Data[i].Name, coinList.Data[i].Symbol, coinList.Data[i].Quote.USD.Price)
	}

	//serialize the coinlist to json
	jsonCoinList, _ := json.Marshal(coinList)
	//write the json to a file
	err = ioutil.WriteFile("coinlist.json", jsonCoinList, 0644)

	// pretty print the json
	prettyJSON, _ := json.MarshalIndent(coinList, "", " ")
	fmt.Println(string(prettyJSON)) 

} 

func main(){ 
  // poll the api every 15 minutes	
  for {
		poll()
		time.Sleep(time.Second * 2)
	}
}

