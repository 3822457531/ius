const MODEL_I18N_KEYS = {
  'SOHA产出': 'coinLog.modelMintOutput',
  'SOHA兑换算力': 'coinLog.modelSwapToPower',
  '自助下单买入SOHA': 'coinLog.modelBuySoha',
  '自助下单卖出SOHA': 'coinLog.modelSellSoha',
  'C2C买入SOHA': 'coinLog.modelC2cBuy',
  'C2C卖出SOHA': 'coinLog.modelC2cSell',
}

export function formatCoinLogModel(t, val) {
  if (!val) return t('coinLog.modelOther')
  const key = MODEL_I18N_KEYS[val]
  if (key) return t(key)
  return val
}
