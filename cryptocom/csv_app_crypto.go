package cryptocom

import (
	"encoding/csv"
	"io"
	"log"
	"time"

	"github.com/fiscafacile/CryptoFiscaFacile/wallet"
	"github.com/shopspring/decimal"
)

type csvAppCryptoTX struct {
	Timestamp       time.Time
	Description     string
	Currency        string
	Amount          decimal.Decimal
	ToCurrency      string
	ToAmount        decimal.Decimal
	NativeCurrency  string
	NativeAmount    decimal.Decimal
	NativeAmountUSD decimal.Decimal
	Kind            string
}

func (cdc *CryptoCom) ParseCSVAppCrypto(reader io.Reader) (err error) {
	csvReader := csv.NewReader(reader)
	records, err := csvReader.ReadAll()
	if err == nil {
		alreadyAsked := []string{}
		for _, r := range records {
			if r[0] != "Timestamp (UTC)" {
				tx := csvAppCryptoTX{}
				tx.Timestamp, err = time.Parse("2006-01-02 15:04:05", r[0])
				if err != nil {
					log.Println("Error Parsing Timestamp : ", r[0])
				}
				tx.Description = r[1]
				tx.Currency = r[2]
				tx.Amount, err = decimal.NewFromString(r[3])
				if err != nil {
					log.Println("Error Parsing Amount : ", r[3])
				}
				tx.ToCurrency = r[4]
				tx.ToAmount, _ = decimal.NewFromString(r[5])
				tx.NativeCurrency = r[6]
				tx.NativeAmount, err = decimal.NewFromString(r[7])
				if err != nil {
					log.Println("Error Parsing NativeAmount : ", r[7])
				}
				tx.NativeAmountUSD, err = decimal.NewFromString(r[8])
				if err != nil {
					log.Println("Error Parsing NativeAmountUSD : ", r[8])
				}
				tx.Kind = r[9]
				cdc.csvAppCryptoTXs = append(cdc.csvAppCryptoTXs, tx)
				// Fill TXsByCategory
				if tx.Kind == "dust_conversion_credited" ||
					tx.Kind == "dust_conversion_debited" ||
					tx.Kind == "interest_swap_credited" ||
					tx.Kind == "interest_swap_debited" ||
					tx.Kind == "lockup_swap_credited" ||
					tx.Kind == "lockup_swap_debited" ||
					tx.Kind == "crypto_wallet_swap_credited" ||
					tx.Kind == "crypto_wallet_swap_debited" {
					found := false
					for i, ex := range cdc.TXsByCategory["Exchanges"] {
						if ex.SimilarDate(2*time.Second, tx.Timestamp) &&
							ex.Note[:5] == tx.Kind[:5] {
							found = true
							if tx.Amount.IsPositive() {
								cdc.TXsByCategory["Exchanges"][i].Items["To"] = append(cdc.TXsByCategory["Exchanges"][i].Items["To"], wallet.Currency{Code: tx.Currency, Amount: tx.Amount})
							} else {
								cdc.TXsByCategory["Exchanges"][i].Items["From"] = append(cdc.TXsByCategory["Exchanges"][i].Items["From"], wallet.Currency{Code: tx.Currency, Amount: tx.Amount.Neg()})
							}
						}
					}
					if !found {
						t := wallet.TX{Timestamp: tx.Timestamp, Note: "Crypto.com App CSV : " + tx.Kind + " " + tx.Description}
						t.Items = make(map[string]wallet.Currencies)
						if tx.Amount.IsPositive() {
							t.Items["To"] = append(t.Items["To"], wallet.Currency{Code: tx.Currency, Amount: tx.Amount})
							cdc.TXsByCategory["Exchanges"] = append(cdc.TXsByCategory["Exchanges"], t)
						} else {
							t.Items["From"] = append(t.Items["From"], wallet.Currency{Code: tx.Currency, Amount: tx.Amount.Neg()})
							cdc.TXsByCategory["Exchanges"] = append(cdc.TXsByCategory["Exchanges"], t)
						}
					}
				} else if tx.Kind == "crypto_exchange" ||
					tx.Kind == "viban_purchase" {
					t := wallet.TX{Timestamp: tx.Timestamp, Note: "Crypto.com App CSV : " + tx.Kind + " " + tx.Description}
					t.Items = make(map[string]wallet.Currencies)
					t.Items["To"] = append(t.Items["To"], wallet.Currency{Code: tx.ToCurrency, Amount: tx.ToAmount})
					t.Items["From"] = append(t.Items["From"], wallet.Currency{Code: tx.Currency, Amount: tx.Amount.Neg()})
					cdc.TXsByCategory["Exchanges"] = append(cdc.TXsByCategory["Exchanges"], t)
				} else if tx.Kind == "crypto_deposit" ||
					tx.Kind == "viban_deposit" ||
					tx.Kind == "exchange_to_crypto_transfer" ||
					tx.Kind == "admin_wallet_credited" ||
					tx.Kind == "referral_card_cashback" ||
					tx.Kind == "transfer_cashback" ||
					tx.Kind == "reimbursement" ||
					tx.Kind == "crypto_earn_interest_paid" ||
					tx.Kind == "crypto_earn_extra_interest_paid" ||
					tx.Kind == "gift_card_reward" ||
					tx.Kind == "pay_checkout_reward" ||
					tx.Kind == "referral_gift" ||
					tx.Kind == "referral_bonus" ||
					tx.Kind == "mco_stake_reward" ||
					tx.Kind == "supercharger_withdrawal" ||
					tx.Kind == "crypto_purchase" ||
					tx.Kind == "staking_reward" {
					t := wallet.TX{Timestamp: tx.Timestamp, Note: "Crypto.com App CSV : " + tx.Kind + " " + tx.Description}
					t.Items = make(map[string]wallet.Currencies)
					t.Items["To"] = append(t.Items["To"], wallet.Currency{Code: tx.Currency, Amount: tx.Amount})
					if tx.Kind == "crypto_purchase" {
						t.Items["From"] = append(t.Items["From"], wallet.Currency{Code: tx.NativeCurrency, Amount: tx.NativeAmount})
						cdc.TXsByCategory["CashIn"] = append(cdc.TXsByCategory["CashIn"], t)
					} else if tx.Kind == "referral_card_cashback" ||
						tx.Kind == "transfer_cashback" ||
						tx.Kind == "reimbursement" ||
						tx.Kind == "gift_card_reward" ||
						tx.Kind == "pay_checkout_reward" {
						cdc.TXsByCategory["CommercialRebates"] = append(cdc.TXsByCategory["CommercialRebates"], t)
					} else if tx.Kind == "crypto_earn_interest_paid" ||
						tx.Kind == "crypto_earn_extra_interest_paid" ||
						tx.Kind == "mco_stake_reward" ||
						tx.Kind == "staking_reward" {
						cdc.TXsByCategory["Interests"] = append(cdc.TXsByCategory["Interests"], t)
					} else if tx.Kind == "referral_gift" ||
						tx.Kind == "referral_bonus" {
						cdc.TXsByCategory["Referrals"] = append(cdc.TXsByCategory["Referrals"], t)
					} else {
						cdc.TXsByCategory["Deposits"] = append(cdc.TXsByCategory["Deposits"], t)
					}
				} else if tx.Kind == "crypto_payment" ||
					tx.Kind == "crypto_withdrawal" ||
					tx.Kind == "crypto_transfer" ||
					tx.Kind == "card_cashback_reverted" ||
					tx.Kind == "transfer_cashback_reverted" ||
					tx.Kind == "reimbursement_reverted" ||
					tx.Kind == "crypto_to_exchange_transfer" ||
					tx.Kind == "supercharger_deposit" ||
					tx.Kind == "crypto_viban_exchange" {
					t := wallet.TX{Timestamp: tx.Timestamp, Note: "Crypto.com App CSV : " + tx.Kind + " " + tx.Description}
					t.Items = make(map[string]wallet.Currencies)
					if tx.Kind == "crypto_withdrawal" &&
						tx.Description == "Withdraw BTC" {
						fee := decimal.New(3, -4) // 0.0003, is it always the case ? I have only one occurence
						t.Items["Fee"] = append(t.Items["Fee"], wallet.Currency{Code: tx.Currency, Amount: fee})
						t.Items["From"] = append(t.Items["From"], wallet.Currency{Code: tx.Currency, Amount: tx.Amount.Neg().Sub(fee)})
					} else {
						t.Items["From"] = append(t.Items["From"], wallet.Currency{Code: tx.Currency, Amount: tx.Amount.Neg()})
					}
					if tx.Kind == "crypto_payment" ||
						tx.Kind == "crypto_viban_exchange" {
						t.Items["To"] = append(t.Items["To"], wallet.Currency{Code: tx.NativeCurrency, Amount: tx.NativeAmount.Neg()})
						cdc.TXsByCategory["CashOut"] = append(cdc.TXsByCategory["CashOut"], t)
					} else if tx.Kind == "card_cashback_reverted" ||
						tx.Kind == "transfer_cashback_reverted" ||
						tx.Kind == "reimbursement_reverted" {
						cdc.TXsByCategory["CommercialRebates"] = append(cdc.TXsByCategory["CommercialRebates"], t)
					} else {
						cdc.TXsByCategory["Withdrawals"] = append(cdc.TXsByCategory["Withdrawals"], t)
					}
				} else if tx.Kind == "crypto_earn_program_created" ||
					tx.Kind == "crypto_earn_program_withdrawn" ||
					tx.Kind == "lockup_lock" ||
					tx.Kind == "lockup_upgrade" ||
					tx.Kind == "lockup_swap_rebate" ||
					tx.Kind == "dynamic_coin_swap_bonus_exchange_deposit" ||
					tx.Kind == "dynamic_coin_swap_credited" ||
					tx.Kind == "dynamic_coin_swap_debited" {
					// Do nothing
				} else {
					found := false
					for _, k := range alreadyAsked {
						if k == tx.Kind {
							found = true
						}
					}
					if !found {
						log.Println("Unmanaged", tx.Kind, "please copy this into t.me/cryptofiscafacile so we can add support for it :", wallet.Base64String(tx))
						alreadyAsked = append(alreadyAsked, tx.Kind)
					}
				}
			}
		}
	}
	return
}
