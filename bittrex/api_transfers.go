package bittrex

import (
	"encoding/json"
	"log"
	"time"

	"github.com/fiscafacile/CryptoFiscaFacile/category"
	"github.com/fiscafacile/CryptoFiscaFacile/wallet"
	scribble "github.com/nanobox-io/golang-scribble"
	"github.com/shopspring/decimal"
	"gopkg.in/resty.v1"
)

type api struct {
	client *resty.Client
}

type apiTransferTX struct {
	Time     time.Time
	Currency string
	Amount   decimal.Decimal
	Fee      decimal.Decimal
	Address  string
	Status   string
}

type transferRequestParams struct {
	Status   string `json:"status,omitempty"`
	PageSize string `json:"pageSize,omitempty"`
}

type transferResponse struct {
	Accountid          string `json:"accountId"`
	Clientwithdrawalid string `json:"clientWithdrawalId"`
	Completedat        string `json:"completedAt"`
	Confirmations      string `json:"confirmations"`
	Createdat          string `json:"createdAt"`
	Cryptoaddress      string `json:"cryptoAddress"`
	Cryptoaddresstag   string `json:"cryptoAddressTag"`
	Currencysymbol     string `json:"currencySymbol"`
	ID                 string `json:"id"`
	Quantity           string `json:"quantity"`
	Source             string `json:"source"`
	Status             string `json:"status"`
	Txcost             string `json:"txCost"`
	Txid               string `json:"txId"`
	Updatedat          string `json:"updatedAt"`
}

func (btrx *Bittrex) getDeposits(apiKey, apiSecret string) (depositTx *resty.Response, err error) {
	btrx.api.client = resty.New()
	requestParams := &transferRequestParams{
		Status:   "COMPLETED",
		PageSize: "200",
	}
	// Convert params struct to json
	jsonParams, _ := json.Marshal(requestParams)
	// Convert json to map
	map_data := make(map[string]string)
	json.Unmarshal([]byte(jsonParams), &map_data)
	// Generate signature
	request := btrx.api.client.R().SetQueryParams(map_data)
	response, err := btrx.sendRequest(apiKey, apiSecret, "deposits/closed", "GET", request)
	return response, err
}

func (btrx *Bittrex) getWithdrawals(apiKey, apiSecret string) (withdrawalTx *resty.Response, err error) {
	btrx.api.client = resty.New()
	requestParams := &transferRequestParams{
		Status:   "COMPLETED",
		PageSize: "200",
	}
	// Convert params struct to json
	jsonParams, _ := json.Marshal(requestParams)
	// Convert json to map
	map_data := make(map[string]string)
	json.Unmarshal([]byte(jsonParams), &map_data)
	// Generate signature
	request := btrx.api.client.R().SetQueryParams(map_data)
	response, err := btrx.sendRequest(apiKey, apiSecret, "withdrawals/closed", "GET", request)
	return response, err
}

func (btrx *Bittrex) GetAllTransferTXs(apiKey, apiSecret string, cat category.Category) {
	useCache := true
	var transferTx []transferResponse
	db, err := scribble.New("./Cache", nil)
	if err != nil {
		useCache = false
	}
	if useCache {
		err = db.Read("Bittrex", "transfers", &transferTx)
	}
	if !useCache || err != nil {
		// Retrieve and cache transfers
		var depositTx []transferResponse
		deposit, err := btrx.getDeposits(apiKey, apiSecret)
		if err != nil {
			time.Sleep(6 * time.Second)
			deposit, err = btrx.getDeposits(apiKey, apiSecret)
			if err != nil {
				log.Println("Bittrex API : Error while fetching deposits", err)
			}
		}
		json.Unmarshal(deposit.Body(), &depositTx)
		// Retrieve and cache withdrawals transfers
		var withdrawalTx []transferResponse
		withdrawal, err := btrx.getWithdrawals(apiKey, apiSecret)
		if err != nil {
			time.Sleep(6 * time.Second)
			withdrawal, err = btrx.getWithdrawals(apiKey, apiSecret)
			if err != nil {
				log.Println("Bittrex API : Error while fetching withdrawals", err)
			}
		}
		json.Unmarshal(withdrawal.Body(), &withdrawalTx)
		transferTx = append(depositTx, withdrawalTx...)
		if useCache {
			err = db.Write("Bittrex", "transfers", transferTx)
			if err != nil {
				log.Println("Bittrex API : Error while caching transfers", err)
			}
		}
	}
	// Process transfer transactions
	for _, trf := range transferTx {
		tx := apiTransferTX{}
		tx.Time, err = time.Parse("2006-01-02T15:04:05.99Z", trf.Completedat)
		if err != nil {
			log.Println("Error Parsing Time : ", trf.Completedat)
		}
		tx.Currency = trf.Currencysymbol
		tx.Amount, err = decimal.NewFromString(trf.Quantity)
		if err != nil {
			log.Println("Error Parsing Amount : ", trf.Quantity)
		}
		tx.Address = trf.Cryptoaddress
		tx.Status = trf.Status
		if trf.Txcost != "" {
			tx.Fee, err = decimal.NewFromString(trf.Txcost)
			if err != nil {
				log.Println("Error Parsing Amount : ", trf.Txcost)
			}
		}
		t := wallet.TX{Timestamp: tx.Time, Note: "Bittrex Transfer API : " + tx.Address}
		t.Items = make(map[string]wallet.Currencies)
		if trf.Source == "" {
			t.Items["From"] = append(t.Items["From"], wallet.Currency{Code: tx.Currency, Amount: tx.Amount})
			t.Items["Fee"] = append(t.Items["Fee"], wallet.Currency{Code: tx.Currency, Amount: tx.Fee})
			btrx.TXsByCategory["Withdrawals"] = append(btrx.TXsByCategory["Withdrawals"], t)
		} else {
			t.Items["To"] = append(t.Items["To"], wallet.Currency{Code: tx.Currency, Amount: tx.Amount})
			btrx.TXsByCategory["Deposits"] = append(btrx.TXsByCategory["Deposits"], t)
		}
	}
	btrx.transferDone <- err
}
