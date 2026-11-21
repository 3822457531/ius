/**
 * 将后端返回的算力明细 model 文本按当前语言替换为 i18n 文案。
 * 后端可能返回中文（如 音乐生成、用户领取每日福利、转账:收款人(用户名)），
 * 这里统一做前端多语言替换。
 * @param {(key: string, params?: Record<string, string>) => string} t - vue-i18n 的 t 函数
 * @param {string} val - 原始 model 字符串
 * @returns {string}
 */
export function formatPowerLogModel(t, val) {
  if (val === undefined || val === null || String(val).trim() === '') {
    return t('powerLog.modelOther');
  }
  let text = String(val).trim();

  // 带参数的模板：转账:收款人(用户名) / 收款:转账人(用户名)
  const transferRecipientMatch = text.match(/^转账:收款人\s*\(([^)]*)\)\s*$/);
  if (transferRecipientMatch) {
    return t('powerLog.modelTransferRecipient', { name: transferRecipientMatch[1].trim() });
  }
  const transferFeeRecipientMatch = text.match(/^转账手续费:收款人\s*\(([^)]*)\)\s*$/);
  if (transferFeeRecipientMatch) {
    return t('powerLog.modelTransferFeeRecipient', { name: transferFeeRecipientMatch[1].trim() });
  }
  const transferSenderMatch = text.match(/^收款:转账人\s*\(([^)]*)\)\s*$/);
  if (transferSenderMatch) {
    return t('powerLog.modelTransferSender', { name: transferSenderMatch[1].trim() });
  }
  const giftBeneficiaryMatch = text.match(/^赠送激活:受益人\s*\(([^)]*)\)\s*$/);
  if (giftBeneficiaryMatch) {
    return t('powerLog.modelGiftActivateBeneficiary', { name: giftBeneficiaryMatch[1].trim() });
  }
  const giftFromMatch = text.match(/^收到赠送激活:赠送人\s*\(([^)]*)\)\s*$/);
  if (giftFromMatch) {
    return t('powerLog.modelGiftActivateFrom', { name: giftFromMatch[1].trim() });
  }
  const creditOrderFromMatch = text.match(/^收到授信报单:赠送人\s*\(([^)]*)\)\s*$/);
  if (creditOrderFromMatch) {
    return t('powerLog.modelCreditOrderFrom', { name: creditOrderFromMatch[1].trim() });
  }

  // 整段完全匹配的替换（含历史 model 名 → 新展示名）
  const exactMap = {
    '矿机': 'powerLog.modelMintMachine',
    '算力机': 'powerLog.modelMintMachine',
    '音乐生成': 'powerLog.modelMusicGenerate',
    'AI音乐': 'powerLog.modelMusicGenerate',
    '用户领取每日福利': 'powerLog.modelDailyBenefit',
    '推荐静态奖励释放': 'powerLog.modelRecommendStaticRelease',
    '推荐奖静态释放': 'powerLog.modelReferralStaticRelease',
    '管理奖静态释放': 'powerLog.modelMgmtStaticRelease',
    '等级分红奖励': 'powerLog.modelLevelBonus',
    '授信额度增加': 'powerLog.modelCreditGrant',
    '授信报单': 'powerLog.modelCreditSpend',
    '授信赠送': 'powerLog.modelCreditSpend',
    '授信还款': 'powerLog.modelCreditRepay',
    '授信额度调整': 'powerLog.modelCreditRecalc',
    '最大额度增加': 'powerLog.modelCreditMaxUp',
    '最大额度减少': 'powerLog.modelCreditMaxDown',
    'power回购': 'powerLog.modelPowerBuyback',
    'power撮合池': 'powerLog.modelPowerMatchingPool',
    'ai-logo': 'powerLog.modelAiLogo',
    'AI-Logo': 'powerLog.modelAiLogo',
    'AI Logo': 'powerLog.modelAiLogo',
    'gpt-image-2-all': 'powerLog.modelAiImage',
    'gpt-image-2': 'powerLog.modelAiImage',
    'GPT-Image2': 'powerLog.modelAiImage',
    'gemini-mj': 'powerLog.modelAiImage',
    'mid-journey': 'powerLog.modelAiImage',
    'AI-图片': 'powerLog.modelAiImage',
    'AI图片': 'powerLog.modelAiImage',
    'sd2': 'powerLog.modelAiVideo',
    'AI-视频': 'powerLog.modelAiVideo',
    'AI视频': 'powerLog.modelAiVideo',
    'PPT': 'powerLog.modelAiPpt',
    'AI-PPT': 'powerLog.modelAiPpt',
    'AI搜': 'powerLog.modelAiSearch',
    // 聊天 LLM：历史流水可能存的是上游模型值（如 deepseek-flash），统一展示为 OSAI
    'deepseek-flash': 'powerLog.modelOsai',
    'deepseek-chat': 'powerLog.modelOsai',
    'deepseek-reasoner': 'powerLog.modelOsai',
    'deepseek': 'powerLog.modelOsai',
    'DeepSeek': 'powerLog.modelOsai',
    'AI对话': 'powerLog.modelOsai',
  };
  if (exactMap[text]) {
    return t(exactMap[text]);
  }

  // 片段替换（保留已有逻辑）
  const replacements = [
    { key: '矿机', value: t('powerLog.modelMintMachine') },
    { key: '购买矿机', value: t('powerLog.modelBuyMintMachine') },
    { key: '静态收益释放', value: t('powerLog.modelStaticRelease') },
    { key: '团队奖励释放', value: t('powerLog.modelTeamRelease') },
    { key: '推荐奖励释放', value: t('powerLog.modelReferralRelease') },
    { key: '推荐奖静态释放', value: t('powerLog.modelReferralStaticRelease') },
    { key: '管理奖静态释放', value: t('powerLog.modelMgmtStaticRelease') },
    { key: '推荐静态奖励释放', value: t('powerLog.modelRecommendStaticRelease') },
    { key: '管理员', value: t('powerLog.modelAdmin') },
    { key: '购买算力产品：', value: t('powerLog.modelBuyPrefix') },
    { key: '算力', value: t('powerLog.modelPowerUnit') },
  ];

  replacements.forEach(({ key, value }) => {
    if (!value || value === key) return;
    text = text.split(key).join(value);
  });

  return text;
}

/** 管理后台展示：历史文案中的「矿机」统一显示为「算力机」；聊天上游模型值统一为 OSAI */
export function normalizePowerLogText(val) {
  if (val === undefined || val === null) {
    return '';
  }
  let text = String(val).replace(/矿机/g, '算力机');
  const osaiAliases = [
    'deepseek-flash',
    'deepseek-chat',
    'deepseek-reasoner',
    'deepseek',
    'DeepSeek',
    'AI对话',
  ];
  if (osaiAliases.includes(text.trim())) {
    return 'OSAI';
  }
  return text;
}
