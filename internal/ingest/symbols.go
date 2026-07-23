package ingest

import "strings"

var CanonicalSymbolMap = map[string]string{
	"eurusd": "EURUSD", "gbpusd": "GBPUSD", "usdjpy": "USDJPY",
	"usdchf": "USDCHF", "audusd": "AUDUSD", "usdcad": "USDCAD", "nzdusd": "NZDUSD",
	"eurgbp": "EURGBP", "eurjpy": "EURJPY", "eurchf": "EURCHF",
	"gbpjpy": "GBPJPY", "gbpchf": "GBPCHF", "chfjpy": "CHFJPY",
	"audjpy": "AUDJPY", "audnzd": "AUDNZD", "audchf": "AUDCHF",
	"cadjpy": "CADJPY", "nzdjpy": "NZDJPY", "nzdchf": "NZDCHF",
	"euraud": "EURAUD", "eurcad": "EURCAD", "eurnzd": "EURNZD",
	"gbpaud": "GBPAUD", "gbpcad": "GBPCAD", "gbpnzd": "GBPNZD",
	"^dji": "US30", "^spx": "SPX500", "^ndq": "NAS100",
	"^dax": "GER40", "^nkx": "JPN225", "^ukx": "UK100",
	"btc.v": "BTCUSD", "eth.v": "ETHUSD",
	"xauusd": "XAUUSD", "xagusd": "XAGUSD",
}

func ResolveTicker(raw string) string {
	upper := strings.ToUpper(raw)
	if ticker, ok := CanonicalSymbolMap[strings.ToLower(raw)]; ok {
		return ticker
	}
	return upper
}
