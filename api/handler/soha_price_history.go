package handler

import (
	"chatplus/store/vo"
)

// EnrichSohaPriceHistoryChangeRates 按相邻日价格重算涨幅（首日=0），保证展示与价格一致
func EnrichSohaPriceHistoryChangeRates(list []vo.PriceHistory) []vo.PriceHistory {
	if len(list) == 0 {
		return list
	}
	out := make([]vo.PriceHistory, len(list))
	copy(out, list)
	out[0].ChangeRate = 0
	for i := 1; i < len(out); i++ {
		out[i].ChangeRate = SohaChangeRateFromPrices(out[i-1].Price, out[i].Price)
	}
	return out
}
