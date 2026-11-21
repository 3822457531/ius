/** SOHA K 线涨幅：与后端 SohaChangeRateFromPrices / EnrichSohaPriceHistoryChangeRates 一致 */

export function sohaChangeRateFromPrices(prev, current) {
  const p = Number(prev)
  const c = Number(current)
  if (!p || p <= 0 || !c || c <= 0) return 0
  return Math.round(((c - p) / p) * 10000) / 100
}

/** 按相邻日价格重算 ChangeRate，首日=0 */
export function enrichSohaPriceHistoryChangeRates(list) {
  if (!list?.length) return list || []
  return list.map((item, i, arr) => ({
    ...item,
    ChangeRate: i === 0 ? 0 : sohaChangeRateFromPrices(arr[i - 1].Price, item.Price),
  }))
}
