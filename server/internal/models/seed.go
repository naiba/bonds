package models

import "gorm.io/gorm"

var currencyCodes = []string{
	"AFA", "ALL", "DZD", "AOA", "ARS", "AMD", "AWG", "AUD", "AZN",
	"BSD", "BHD", "BDT", "BBD", "BYR", "BEF", "BZD", "BMD", "BTN",
	"BTC", "BOB", "BAM", "BWP", "BRL", "GBP", "BND", "BGN", "BIF",
	"KHR", "CAD", "CVE", "KYD", "XOF", "XAF", "XPF", "CLP", "CNY",
	"COP", "KMF", "CDF", "CRC", "HRK", "CUC", "CZK", "DKK", "DJF",
	"DOP", "XCD", "EGP", "ERN", "EEK", "ETB", "EUR", "FKP", "FJD",
	"GMD", "GEL", "DEM", "GHS", "GIP", "GRD", "GTQ", "GNF", "GYD",
	"HTG", "HNL", "HKD", "HUF", "ISK", "INR", "IDR", "IRR", "IQD",
	"ILS", "ITL", "JMD", "JPY", "JOD", "KZT", "KES", "KWD", "KGS",
	"LAK", "LVL", "LBP", "LSL", "LRD", "LYD", "LTL", "MOP", "MKD",
	"MGA", "MWK", "MYR", "MVR", "MRO", "MUR", "MXN", "MDL", "MNT",
	"MAD", "MZM", "MMK", "NAD", "NPR", "ANG", "TWD", "NZD", "NIO",
	"NGN", "KPW", "NOK", "OMR", "PKR", "PAB", "PGK", "PYG", "PEN",
	"PHP", "PLN", "QAR", "RON", "RUB", "RWF", "SVC", "WST", "SAR",
	"RSD", "SCR", "SLL", "SGD", "SKK", "SBD", "SOS", "ZAR", "KRW",
	"XDR", "LKR", "SHP", "SDG", "SRD", "SZL", "SEK", "CHF", "SYP",
	"STD", "TJS", "TZS", "THB", "TOP", "TTD", "TND", "TRY", "TMT",
	"UGX", "UAH", "AED", "UYU", "USD", "UZS", "VUV", "VEF", "VND",
	"YER", "ZM",
}

func SeedCurrencies(db *gorm.DB) error {
	var count int64
	if err := db.Model(&Currency{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	currencies := make([]Currency, len(currencyCodes))
	for i, code := range currencyCodes {
		currencies[i] = Currency{Code: code}
	}
	return db.CreateInBatches(currencies, 50).Error
}
