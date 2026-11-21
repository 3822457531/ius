<template>
  <div class="system-config" v-loading="loading">

    <el-tabs v-model="activeName" class="demo-tabs">
      <el-tab-pane label="基本设置" name="basic">
        <div class="container">
          <el-form :model="system" label-width="150px" label-position="right" ref="systemFormRef" :rules="rules">
            <el-form-item label="网站标题" prop="title">
              <el-input v-model="system['title']"/>
            </el-form-item>
            <el-form-item label="控制台标题" prop="admin_title">
              <el-input v-model="system['admin_title']"/>
            </el-form-item>
            <el-form-item label="网站 LOGO" prop="logo">
              <el-input v-model="system['logo']" placeholder="网站LOGO图片">
                <template #append>
                  <el-upload
                      :auto-upload="true"
                      :show-file-list="false"
                      @click="beforeUpload('logo')"
                      :http-request="uploadImg"
                  >
                    <el-icon class="uploader-icon">
                      <UploadFilled/>
                    </el-icon>
                  </el-upload>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item label="注册赠送算力" prop="init_power">
              <el-input v-model.number="system['init_power']" placeholder="新用户注册赠送算力"/>
            </el-form-item>
            <el-form-item label="邀请赠送算力" prop="invite_power">
              <el-input v-model.number="system['invite_power']" placeholder="邀请新用户注册赠送算力"/>
            </el-form-item>

            <el-form-item label="截止目前的销毁数" prop="limit_cancel">
              <el-input-number
                v-model="system['limit_cancel']"
                :min="0"
                :step="1"
                controls-position="right"
                placeholder="0"
                style="width: 260px"
              />
              <div class="tip" style="margin-top: 6px;">可填 0；保存后前台「目前已销毁量」同步显示为 0 SOHA</div>
            </el-form-item>


            
            <el-form-item label="转账最低算力值" prop="vip_month_power">
              <el-input v-model.number="system['vip_month_power']" placeholder="VIP用户每月赠送算力"/>
            </el-form-item>

            <el-form-item label="开启转账手续费" prop="enabled_transfer_fee">
              <el-switch v-model="system['enabled_transfer_fee']"/>
              <span class="tip" style="margin-left: 12px;">关闭后用户转账不再额外扣手续费，仅扣转出算力</span>
            </el-form-item>

            <el-form-item label="转账手续费比例(%)" prop="transfer_fee_percent">
              <div class="tip-input-line">
                <el-input-number
                  v-model="system['transfer_fee_percent']"
                  :min="0"
                  :max="100"
                  :precision="2"
                  :step="0.5"
                  :disabled="!system['enabled_transfer_fee']"
                  controls-position="right"
                  placeholder="例如 5 表示按转出金额另扣 5% 手续费"
                  style="width: 260px"
                />
                <div class="tip">
                  手续费从转出方额外扣除，不与到账金额混扣；收款方实收 = 用户填写的转出算力全额；转出方合计支出 = 转出算力 + 转出算力 × (比例/100)。需先开启「转账手续费」开关后比例才生效。
                </div>
              </div>
            </el-form-item>

            <el-form-item label="手续费回流认购池(%)" prop="transfer_fee_to_pool_percent">
              <div class="tip-input-line">
                <el-input-number
                  v-model="system['transfer_fee_to_pool_percent']"
                  :min="0"
                  :max="100"
                  :precision="2"
                  :step="1"
                  :disabled="!system['enabled_transfer_fee']"
                  controls-position="right"
                  placeholder="默认 100"
                  style="width: 260px"
                />
                <div class="tip">
                  转账/赠送手续费中按该比例静默注入认购池（算力池 total_power）；例：手续费 1、本项 100% → 池子增加 1。不额外展示用户流水。
                </div>
              </div>
            </el-form-item>

            <el-form-item label="每日赠送算力" prop="daily_power">
              <div class="tip-input-line">
                <el-input v-model.number="system['daily_power']" placeholder="默认值0"/>
                <div class="tip">
                  系统自动补算力额度（与下方「签到福利」独立）。设置为 0 表示不自动补算力。
                </div>
              </div>
            </el-form-item>

            <el-divider content-position="left">签到福利配置</el-divider>

            <el-form-item label="0-30天签到算力" prop="sign_in_reward_0_30">
              <el-input-number v-model="system['sign_in_reward_0_30']" :min="0" :precision="2" :step="1" controls-position="right" placeholder="默认 10" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="31-60天签到算力" prop="sign_in_reward_31_60">
              <el-input-number v-model="system['sign_in_reward_31_60']" :min="0" :precision="2" :step="1" controls-position="right" placeholder="默认 5" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="61-90天签到算力" prop="sign_in_reward_61_90">
              <el-input-number v-model="system['sign_in_reward_61_90']" :min="0" :precision="2" :step="1" controls-position="right" placeholder="默认 3" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="90天以后签到算力" prop="sign_in_reward_90_plus">
              <el-input-number v-model="system['sign_in_reward_90_plus']" :min="0" :precision="2" :step="1" controls-position="right" placeholder="默认 0（不可领取）" style="width: 260px"/>
            </el-form-item>

            <el-form-item>
              <div class="tip" style="color:#909399;font-size:12px;">
                用户签到（领取每日福利）按注册天数分段赠送算力；未填写的分段使用默认 10/5/3。90 天以后填 0 表示不可签到。与上方「每日赠送算力」（系统自动补算力）相互独立。
              </div>
            </el-form-item>

            <el-divider content-position="left">自定义套餐 (confirmJihuoEx)</el-divider>

            <el-form-item label="最低投入算力" prop="jihuo_ex_min_power">
              <div class="tip-input-line">
                <el-input-number
                  v-model="system['jihuo_ex_min_power']"
                  :min="0"
                  :precision="0"
                  :step="1000"
                  controls-position="right"
                  placeholder="0 表示不限制"
                  style="width: 260px"
                />
                <div class="tip">
                  confirmJihuoEx 单笔最低投入算力；填 0 表示不限制。
                </div>
              </div>
            </el-form-item>

            <el-form-item label="普通会员下限" prop="member_normal_min_power">
              <el-input-number
                v-model="system['member_normal_min_power']"
                :min="0"
                :precision="0"
                :step="1000"
                controls-position="right"
                placeholder="默认 1000"
                style="width: 260px"
              />
            </el-form-item>

            <el-form-item label="普通会员上限" prop="member_normal_max_power">
              <el-input-number
                v-model="system['member_normal_max_power']"
                :min="0"
                :precision="0"
                :step="1000"
                controls-position="right"
                placeholder="默认 9999"
                style="width: 260px"
              />
            </el-form-item>

            <el-form-item label="VIP会员下限" prop="member_vip_min_power">
              <el-input-number
                v-model="system['member_vip_min_power']"
                :min="0"
                :precision="0"
                :step="1000"
                controls-position="right"
                placeholder="默认 10000"
                style="width: 260px"
              />
            </el-form-item>

            <el-form-item label="普通会员小区奖截止" prop="team_reward_normal_cutoff">
              <el-date-picker
                v-model="system['team_reward_normal_cutoff']"
                type="datetime"
                placeholder="选择日期时间"
                format="YYYY-MM-DD HH:mm:ss"
                value-format="YYYY-MM-DD HH:mm:ss"
                style="width: 260px"
              />
              <div class="tip" style="margin-top: 6px;">
                存在单笔投入≥「历史VIP最低投入」且订单时间早于此截止的用户可收小区团队奖（历史 VIP）；之后仅 VIP 可收。
              </div>
            </el-form-item>

            <el-form-item label="历史VIP最低投入" prop="legacy_vip_min_power">
              <el-input-number
                v-model="system['legacy_vip_min_power']"
                :min="0"
                :precision="0"
                :step="1000"
                controls-position="right"
                placeholder="默认 5000"
                style="width: 260px"
              />
            </el-form-item>

            <el-form-item label="历史VIP团队奖比例" prop="legacy_vip_team_reward_pro">
              <el-input-number
                v-model="system['legacy_vip_team_reward_pro']"
                :step="0.001"
                :precision="3"
                :min="0"
                :max="1"
                controls-position="right"
                placeholder="默认 0.1"
                style="width: 260px"
              />
            </el-form-item>

            <el-divider content-position="left">授信额度（老会员分段）</el-divider>

            <el-form-item label="老会员分界时刻" prop="credit_legacy_cutoff">
              <div class="tip-input">
                <el-date-picker
                  v-model="system['credit_legacy_cutoff']"
                  type="datetime"
                  placeholder="留空则不启用老会员分段"
                  format="YYYY-MM-DD HH:mm:ss"
                  value-format="YYYY-MM-DD HH:mm:ss"
                  style="width: 260px"
                />
                <div class="tip" style="margin-top: 6px;">
                  分界前投入合计达到门槛的会员，额度上限 = 「老会员固定上限」+ 分界后新增投入 × 杠杆。
                  未达门槛的会员按「分界前投入 × 杠杆」计算。留空表示所有人按当前机制。
                </div>
              </div>
            </el-form-item>

            <el-form-item label="老会员门槛投入" prop="credit_legacy_min_power">
              <div class="tip-input">
                <el-input-number
                  v-model="system['credit_legacy_min_power']"
                  :min="0"
                  :precision="0"
                  :step="1000"
                  controls-position="right"
                  placeholder="默认 10000"
                  style="width: 260px"
                />
                <div class="tip" style="margin-top: 6px;">
                  同时作为「分界前投入」和「分界后新增投入」的门槛：任一侧不足该值都不计入。
                  分界前按合计判定，不按单笔。填 0 会自动回退为默认 1 万；想让分界前任意投入都算老会员，填 1。
                </div>
              </div>
            </el-form-item>

            <el-form-item label="老会员固定上限" prop="credit_legacy_max">
              <el-input-number
                v-model="system['credit_legacy_max']"
                :min="0"
                :precision="0"
                :step="10000"
                controls-position="right"
                placeholder="默认 50000"
                style="width: 260px"
              />
            </el-form-item>

            <el-form-item label=" ">
              <div class="tip">
                额度由订单投入实时推导，保存后立即生效，无需固化：加入时间早于分界时刻的订单按合计判定资格，
                且在投与已结束订单都计入，订单到期不会让额度回弹。
                修改上限可能下调部分用户的 Max，其可用额度会被截断，待还额度不变。
              </div>
            </el-form-item>

            <el-form-item label="最低收益时间" prop="order_pay_timeout">
              <div class="tip-input">
                <el-input v-model.number="system['order_pay_timeout']" placeholder="单位：分钟"/>
                <div class="info">
                  <el-tooltip
                      effect="dark"
                      content="系统发放收益时，最低的下单时间，要大于等于这个才会发放静态收益<br/>以分钟为单位"
                      raw-content
                      placement="right"
                  >
                    <el-icon>
                      <InfoFilled/>
                    </el-icon>
                  </el-tooltip>
                </div>
              </div>
            </el-form-item>

            <el-form-item label="团队奖励释放比例" prop="team_reward_pro">
              <div class="tip-input-line">
                <el-input-number
                  v-model="system['team_reward_pro']"
                  :step="0.001"
                  :precision="3"
                  :min="0"
                  :max="1"
                  controls-position="right"
                  placeholder="默认 0.2（20%）"
                  style="width: 260px"
                />
                <div class="tip">
                  弱区业绩 × 该比例 = 团队奖励释放算力；填 0 表示不发放（按 0 计算）。未配置时默认 0.2。
                </div>
              </div>
            </el-form-item>

            <el-form-item label="量碰测算层深" prop="team_reward_measure_layer_depth">
              <div class="tip-input-line">
                <el-input-number
                  v-model="system['team_reward_measure_layer_depth']"
                  :min="1"
                  :max="99"
                  :step="1"
                  controls-position="right"
                  placeholder="默认 4"
                  style="width: 260px"
                />
                <div class="tip">
                  以每个收奖候选用户为主体，仅测算其<strong>弱区</strong>向下第 N <strong>逻辑层</strong>入口宽度（不含大区）。单边下行（一条线）不加深，同属一层；仅开出左右双区才进入下一层。默认 N=4。
                </div>
              </div>
            </el-form-item>

            <el-form-item label="量碰宽度阈值" prop="team_reward_layer4_width_limit">
              <div class="tip-input-line">
                <el-input-number
                  v-model="system['team_reward_layer4_width_limit']"
                  :min="2"
                  :max="65536"
                  :step="1"
                  controls-position="right"
                  placeholder="默认 8"
                  style="width: 260px"
                />
                <div class="tip">
                  主体弱区第 N 逻辑层入口数<strong>大于</strong>该值时才拦截（等于不拦）。例如 4,4：弱区第 4 逻辑层有 4 个入口仍可发奖，有 5 个起拦截。单边续链不计入宽度。默认 8。
                </div>
              </div>
            </el-form-item>

            <el-form-item label="团队奖励周封顶">
              <div v-if="system.team_reward_weekly_cap" class="team-weekly-cap-grid">
                <div class="cap-row" v-for="item in teamWeeklyCapFields" :key="item.key">
                  <span class="cap-label">{{ item.label }}</span>
                  <el-input-number
                    v-model="system.team_reward_weekly_cap[item.key]"
                    :min="0"
                    :step="1000"
                    controls-position="right"
                    placeholder="算力/周"
                    style="width: 200px"
                  />
                </div>
                <div class="tip">
                  限制「团队奖励释放」算力，自然周（周一至周日）累计；推荐奖、等级分红、静态等不计入。填 0 表示该档使用系统内置默认值。
                </div>
              </div>
            </el-form-item>

            <el-form-item label="直推加速比例" prop="zj_pro">
              <el-input-number v-model="system['zj_pro']" :step="0.001" :precision="3" :min="0" controls-position="right" placeholder="直推加速比例" style="width: 260px"/>
            </el-form-item>
          
            <el-form-item label="间推加速比例" prop="jj_pro">
              <el-input-number v-model="system['jj_pro']" :step="0.001" :precision="3" :min="0" controls-position="right" placeholder="间推加速比例" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="直推静态收益比例" prop="zj_static_pro">
              <el-input-number v-model="system['zj_static_pro']" :step="0.001" :precision="3" :min="0" controls-position="right" placeholder="推荐奖静态释放比例，默认0.2" style="width: 260px"/>
              <div class="info" style="margin-left:8px;color:#909399;font-size:12px">每晚下级静态释放 × 该比例给直推上级，日志「推荐奖静态释放」</div>
            </el-form-item>
          
            <el-form-item label="间推静态收益比例" prop="jj_static_pro">
              <el-input-number v-model="system['jj_static_pro']" :step="0.001" :precision="3" :min="0" controls-position="right" placeholder="间推静态（可选，默认0关闭）" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="管理奖静态比例" prop="mgmt_static_pro">
              <el-input-number v-model="system['mgmt_static_pro']" :step="0.001" :precision="3" :min="0" controls-position="right" placeholder="管理奖静态释放比例，默认0.1" style="width: 260px"/>
              <div class="info" style="margin-left:8px;color:#909399;font-size:12px">白天获团队奖的人，当晚跟随投入人静态释放 × 该比例，日志「管理奖静态释放」</div>
            </el-form-item>

            <el-divider content-position="left">自定义投入产出倍数</el-divider>
            <el-form-item label="一档范围/倍数">
              <el-input-number v-model="system.invest_multiplier.tier1_min" :min="0" controls-position="right" style="width: 120px"/>
              <span style="margin:0 6px">～</span>
              <el-input-number v-model="system.invest_multiplier.tier1_max" :min="0" controls-position="right" style="width: 120px"/>
              <span style="margin:0 8px">×</span>
              <el-input-number v-model="system.invest_multiplier.tier1_mul" :min="0" :step="0.1" :precision="2" controls-position="right" style="width: 100px"/>
              <div class="info" style="margin-left:8px;color:#909399;font-size:12px">默认 1000～49999 → 5倍</div>
            </el-form-item>
            <el-form-item label="二档范围/倍数">
              <el-input-number v-model="system.invest_multiplier.tier2_min" :min="0" controls-position="right" style="width: 120px"/>
              <span style="margin:0 6px">～</span>
              <el-input-number v-model="system.invest_multiplier.tier2_max" :min="0" controls-position="right" style="width: 120px"/>
              <span style="margin:0 8px">×</span>
              <el-input-number v-model="system.invest_multiplier.tier2_mul" :min="0" :step="0.1" :precision="2" controls-position="right" style="width: 100px"/>
              <div class="info" style="margin-left:8px;color:#909399;font-size:12px">默认 50000～99999 → 6倍</div>
            </el-form-item>
            <el-form-item label="三档起点/倍数">
              <el-input-number v-model="system.invest_multiplier.tier3_min" :min="0" controls-position="right" style="width: 120px"/>
              <span style="margin:0 8px">及以上 ×</span>
              <el-input-number v-model="system.invest_multiplier.tier3_mul" :min="0" :step="0.1" :precision="2" controls-position="right" style="width: 100px"/>
              <div class="info" style="margin-left:8px;color:#909399;font-size:12px">默认 100000+ → 7倍</div>
            </el-form-item>


<!--   
            <el-form-item label="三级加速比例" prop="san_pro">
              <el-input v-model.number="system['san_pro']" placeholder="三级加速比例"/>
            </el-form-item>

            <el-form-item label="四级加速比例" prop="si_pro">
              <el-input v-model.number="system['si_pro']" placeholder="四级加速比例"/>
            </el-form-item>

            <el-form-item label="5-10级别加速比例" prop="other_pro">
              <el-input v-model.number="system['other_pro']" :step="0.01" placeholder="5-10级别加速比例"/>
            </el-form-item> -->

            <!-- 静态收益分段释放比例配置（10 天为粒度） -->
            <el-form-item label="0-10天释放比例" prop="zj_plus">
              <el-input-number v-model="system['zj_plus']" :step="0.0001" :precision="4" :min="0" controls-position="right" placeholder="例如 0.0015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="11-20天释放比例" prop="plus_11_20">
              <el-input-number v-model="system['plus_11_20']" :step="0.0001" :precision="4" :min="0" controls-position="right" placeholder="例如 0.0015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="21-30天释放比例" prop="plus_21_30">
              <el-input-number v-model="system['plus_21_30']" :step="0.0001" :precision="4" :min="0" controls-position="right" placeholder="例如 0.0015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="31-40天释放比例" prop="plus_31_40">
              <el-input-number v-model="system['plus_31_40']" :step="0.0001" :precision="4" :min="0" controls-position="right" placeholder="例如 0.0015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="41-50天释放比例" prop="plus_41_50">
              <el-input-number v-model="system['plus_41_50']" :step="0.0001" :precision="4" :min="0" controls-position="right" placeholder="例如 0.0015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="51-60天释放比例" prop="plus_51_60">
              <el-input-number v-model="system['plus_51_60']" :step="0.0001" :precision="4" :min="0" controls-position="right" placeholder="例如 0.0015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="61-70天释放比例" prop="plus_61_70">
              <el-input-number v-model="system['plus_61_70']" :step="0.0001" :precision="4" :min="0" controls-position="right" placeholder="例如 0.0015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="71-80天释放比例" prop="plus_71_80">
              <el-input-number v-model="system['plus_71_80']" :step="0.0001" :precision="4" :min="0" controls-position="right" placeholder="例如 0.0015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="81-90天释放比例" prop="plus_81_90">
              <el-input-number v-model="system['plus_81_90']" :step="0.0001" :precision="4" :min="0" controls-position="right" placeholder="例如 0.0015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="91-100天+释放比例" prop="plus_91_100">
              <el-input-number v-model="system['plus_91_100']" :step="0.0001" :precision="4" :min="0" controls-position="right" placeholder="例如 0.0015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="91-120天释放比例" prop="plus_91_120">
              <div class="tip-input-line">
                <el-input-number v-model="system['plus_91_120']" :step="0.0001" :precision="4" :min="0" controls-position="right" placeholder="例如 0.0015" style="width: 260px"/>
                <div class="tip">
                  不填则沿用「91-100天+释放比例」。
                </div>
              </div>
            </el-form-item>

            <el-form-item label="121-150天释放比例" prop="plus_121_150">
              <div class="tip-input-line">
                <el-input-number v-model="system['plus_121_150']" :step="0.0001" :precision="4" :min="0" controls-position="right" placeholder="例如 0.0015" style="width: 260px"/>
                <div class="tip">
                  不填则沿用「91-100天+释放比例」。
                </div>
              </div>
            </el-form-item>

            <el-form-item label="151-180天释放比例" prop="plus_151_180">
              <div class="tip-input-line">
                <el-input-number v-model="system['plus_151_180']" :step="0.0001" :precision="4" :min="0" controls-position="right" placeholder="例如 0.0015" style="width: 260px"/>
                <div class="tip">
                  181 天以后沿用本档（封顶）；不填则沿用「91-100天+释放比例」。
                </div>
              </div>
            </el-form-item>

            <el-divider content-position="left">Swap 兑换配置</el-divider>

            <el-form-item label="SOHA首日基准价" prop="soha_guide_base_price">
              <div class="tip-input-line">
                <el-input-number
                  v-model="system['soha_guide_base_price']"
                  :step="0.01"
                  :precision="4"
                  :min="0.0001"
                  controls-position="right"
                  placeholder="默认 0.1"
                  style="width: 260px"
                />
                <div class="tip">
                  K 线首日指导价；之后每日由定时任务随机涨幅 0.10%~0.50% 在前一日价格基础上复利累加。
                </div>
              </div>
            </el-form-item>

            <el-form-item label="Swap锚定算力价" prop="swap_power_price">
              <div class="tip-input-line">
                <el-input-number
                  v-model="system['swap_power_price']"
                  :step="0.001"
                  :precision="4"
                  :min="0.0001"
                  controls-position="right"
                  placeholder="默认 0.01"
                  style="width: 260px"
                />
                <div class="tip">
                  SOHA→算力动态汇率 = 当前SOHA指导价 ÷ 锚定价。锚定价默认 0.01，即 SOHA 为 1 美元时 1 SOHA 可兑换 100 算力（1:100）。
                </div>
              </div>
            </el-form-item>

            <el-divider content-position="left">算力机购买奖励配置</el-divider>

            <el-form-item label="算力机直推推荐比例" prop="kj_zj_pro">
              <el-input-number
                v-model="system['kj_zj_pro']"
                :step="0.001"
                :precision="3"
                :min="0"
                :max="1"
                controls-position="right"
                placeholder="默认 0.3"
                style="width: 260px"
              />
              <div class="tip" style="margin-top: 6px;">
                用户购买算力机时，直推上级从质押订单加速释放的比例；与上方「直推加速比例」独立。
              </div>
            </el-form-item>

            <el-form-item label="算力机团队奖比例" prop="kj_team_reward_pro">
              <el-input-number
                v-model="system['kj_team_reward_pro']"
                :step="0.001"
                :precision="3"
                :min="0"
                :max="1"
                controls-position="right"
                placeholder="默认 0.2"
                style="width: 260px"
              />
              <div class="tip" style="margin-top: 6px;">
                用户购买算力机时，弱区团队奖励释放比例；与上方「团队奖励释放比例」独立，量碰/周封顶规则仍共用。
              </div>
            </el-form-item>

            <el-divider content-position="left">算力机 SOHA 产出分段比例</el-divider>

            <el-form-item label="算力机0-10天产出" prop="kj_zj_plus">
              <el-input-number v-model="system['kj_zj_plus']" :step="0.00001" :precision="6" :min="0" controls-position="right" placeholder="例如 0.00015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="算力机11-20天产出" prop="kj_plus_11_20">
              <el-input-number v-model="system['kj_plus_11_20']" :step="0.00001" :precision="6" :min="0" controls-position="right" placeholder="例如 0.00015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="算力机21-30天产出" prop="kj_plus_21_30">
              <el-input-number v-model="system['kj_plus_21_30']" :step="0.00001" :precision="6" :min="0" controls-position="right" placeholder="例如 0.00015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="算力机31-40天产出" prop="kj_plus_31_40">
              <el-input-number v-model="system['kj_plus_31_40']" :step="0.00001" :precision="6" :min="0" controls-position="right" placeholder="例如 0.00015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="算力机41-50天产出" prop="kj_plus_41_50">
              <el-input-number v-model="system['kj_plus_41_50']" :step="0.00001" :precision="6" :min="0" controls-position="right" placeholder="例如 0.00015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="算力机51-60天产出" prop="kj_plus_51_60">
              <el-input-number v-model="system['kj_plus_51_60']" :step="0.00001" :precision="6" :min="0" controls-position="right" placeholder="例如 0.00015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="算力机61-70天产出" prop="kj_plus_61_70">
              <el-input-number v-model="system['kj_plus_61_70']" :step="0.00001" :precision="6" :min="0" controls-position="right" placeholder="例如 0.00015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="算力机71-80天产出" prop="kj_plus_71_80">
              <el-input-number v-model="system['kj_plus_71_80']" :step="0.00001" :precision="6" :min="0" controls-position="right" placeholder="例如 0.00015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="算力机81-90天产出" prop="kj_plus_81_90">
              <el-input-number v-model="system['kj_plus_81_90']" :step="0.00001" :precision="6" :min="0" controls-position="right" placeholder="例如 0.00015" style="width: 260px"/>
            </el-form-item>

            <el-form-item label="算力机91-100天+产出" prop="kj_plus_91_100">
              <el-input-number v-model="system['kj_plus_91_100']" :step="0.00001" :precision="6" :min="0" controls-position="right" placeholder="例如 0.00015" style="width: 260px"/>
            </el-form-item>
            <el-form-item>
              <div class="tip" style="color:#909399;font-size:12px;">
                每日 SOHA 产出 = 投入算力 × 对应阶段比例。与上方「静态套餐释放比例」独立配置；未配置时回退到算力机自身费率。
              </div>
            </el-form-item>

      

            <el-form-item label="MJ绘图算力" prop="mj_power">
              <div class="tip-input">
                <el-input v-model.number="system['mj_power']" placeholder=""/>
                <div class="info">
                  <el-tooltip
                      effect="dark"
                      content="使用MidJourney画一张图消耗算力"
                      raw-content
                      placement="right"
                  >
                    <el-icon>
                      <InfoFilled/>
                    </el-icon>
                  </el-tooltip>
                </div>
              </div>
            </el-form-item>
            <el-form-item label="MJ操作算力" prop="mj_action_power">
              <div class="tip-input">
                <el-input v-model.number="system['mj_action_power']" placeholder=""/>
                <div class="info">
                  <el-tooltip
                      effect="dark"
                      content="放大，变换，重绘操作一次消耗的算力"
                      raw-content
                      placement="right"
                  >
                    <el-icon>
                      <InfoFilled/>
                    </el-icon>
                  </el-tooltip>
                </div>
              </div>
            </el-form-item>

            <el-divider content-position="left">AI搜算力</el-divider>
            <el-form-item label="AI搜-检索算力" prop="ai_search_search_power">
              <div class="tip-input-line">
                <el-input-number
                  v-model="system['ai_search_search_power']"
                  :min="0"
                  :step="1"
                  controls-position="right"
                  style="width: 260px"
                />
                <div class="tip">每次关键词搜索成功扣费（含验证码通过后出结果）；默认 1，填 0 表示不扣</div>
              </div>
            </el-form-item>
            <el-form-item label="AI搜-详情算力" prop="ai_search_detail_power">
              <div class="tip-input-line">
                <el-input-number
                  v-model="system['ai_search_detail_power']"
                  :min="0"
                  :step="1"
                  controls-position="right"
                  style="width: 260px"
                />
                <div class="tip">点击搜索结果打开资源详情时扣费；默认 10，填 0 表示不扣</div>
              </div>
            </el-form-item>

            <!-- <el-form-item label="Stable-Diffusion算力" prop="sd_power">
              <el-input v-model.number="system['sd_power']" placeholder="使用Stable-Diffusion画一张图消耗算力"/>
            </el-form-item>
            <el-form-item label="DALL-E-3算力" prop="dall_power">
              <el-input v-model.number="system['dall_power']" placeholder="使用DALL-E-3画一张图消耗算力"/>
            </el-form-item> -->

            <el-form-item label="开放注册" prop="enabled_register">
              <el-switch v-model="system['enabled_register']"/>
              <el-tooltip
                  effect="dark"
                  content="关闭注册之后只能通过管理后台添加用户"
                  raw-content
                  placement="right"
              >
                <el-icon>
                  <InfoFilled/>
                </el-icon>
              </el-tooltip>
            </el-form-item>

            <el-form-item label="限制同IP注册" prop="enabled_register_ip_limit">
              <el-switch v-model="system['enabled_register_ip_limit']"/>
              <span class="tip" style="margin-left: 12px;">关闭后同一 IP 可无限注册</span>
            </el-form-item>

            <el-form-item label="同IP最多注册数" prop="register_ip_limit">
              <el-input-number
                v-model="system['register_ip_limit']"
                :min="0"
                :step="1"
                controls-position="right"
                placeholder="默认 3"
                style="width: 260px"
                :disabled="!system['enabled_register_ip_limit']"
              />
              <div class="tip" style="margin-top: 6px;">
                按用户注册 IP（login_ip）统计；填 0 表示不限制。仅在上项开启时生效。
              </div>
            </el-form-item>

            <el-form-item label="开放激活" prop="enabled_activation">
              <el-switch v-model="system['enabled_activation']"/>
              <span class="tip" style="margin-left: 12px;">关闭后用户无法质押激活（confirmJihuo / confirmJihuoEx）</span>
            </el-form-item>

            <el-form-item label="限制用户激活次数" prop="enabled_activation_user_limit">
              <el-switch v-model="system['enabled_activation_user_limit']"/>
              <span class="tip" style="margin-left: 12px;">关闭后已激活用户可无限复投</span>
            </el-form-item>

            <el-form-item label="单用户最多复投" prop="activation_user_limit">
              <el-input-number
                v-model="system['activation_user_limit']"
                :min="0"
                :step="1"
                controls-position="right"
                placeholder="默认 3"
                style="width: 260px"
                :disabled="!system['enabled_activation_user_limit']"
              />
              <div class="tip" style="margin-top: 6px;">
                已激活用户通过 confirmJihuoEx 复投的次数上限；填 0 表示不限制。仅在上项开启时生效。
              </div>
            </el-form-item>

            <el-form-item label="开放SOHA交易市场" prop="enabled_soha_market">
              <el-switch v-model="system['enabled_soha_market']"/>
              <span class="tip" style="margin-left: 12px;">关闭后所有用户不可 Swap、激活算力机；用户级「允许SOHA交易」同时失效</span>
            </el-form-item>

            <el-form-item label="开启K线展示" prop="enabled_soha_kline">
              <el-switch v-model="system['enabled_soha_kline']"/>
              <span class="tip" style="margin-left: 12px;">关闭后用户端 K 线区域为模糊状态；不影响 Swap、算力机激活等其它功能</span>
            </el-form-item>

            <el-form-item label="注册方式" prop="register_ways">
              <el-checkbox-group v-model="system['register_ways']">
                <el-checkbox label="mobile">手机注册</el-checkbox>
                <el-checkbox label="email">邮箱注册</el-checkbox>
                <el-checkbox label="username">用户名注册</el-checkbox>
              </el-checkbox-group>
            </el-form-item>

            <!-- <el-form-item label="启用众筹功能" prop="enabled_reward">
              <el-switch v-model="system['enabled_reward']"/>
              <el-tooltip
                  effect="dark"
                  content="如果关闭次功能将不在用户菜单显示众筹二维码"
                  raw-content
                  placement="right"
              >
                <el-icon>
                  <InfoFilled/>
                </el-icon>
              </el-tooltip>
            </el-form-item> -->

            <div v-if="system['enabled_reward']">
              <el-form-item label="算力单价" prop="power_price">
                <el-input v-model="system['power_price']"
                          placeholder="单位算力的价格，比如设置 0.1 表示捐赠1元钱可以得到10个单位算力"/>
              </el-form-item>
              <el-form-item label="收款二维码" prop="reward_img">
                <el-input v-model="system['reward_img']" placeholder="众筹收款二维码地址">
                  <template #append>
                    <el-upload
                        :auto-upload="true"
                        :show-file-list="false"
                        @click="beforeUpload('reward_img')"
                        :http-request="uploadImg"
                    >
                      <el-icon class="uploader-icon">
                        <UploadFilled/>
                      </el-icon>
                    </el-upload>
                  </template>
                </el-input>
              </el-form-item>
            </div>

            
            <el-form-item label="usdt收款码" prop="usdtimg">
              <el-input v-model="system['usdtimg']" placeholder="微信客服二维码">
                <template #append>
                  <el-upload
                      :auto-upload="true"
                      :show-file-list="false"
                      @click="beforeUpload('usdtimg')"
                      :http-request="uploadImg"
                  >
                    <el-icon class="uploader-icon">
                      <UploadFilled/>
                    </el-icon>
                  </el-upload>
                </template>
              </el-input>
            </el-form-item>

            <el-form-item label="收款地址" prop="address" class="highlight-form-item">
              <el-input v-model="system['address']" />
            </el-form-item>

            <el-form-item label="BSC API Key" prop="bsc_api_key">
              <div class="tip-input">
                <el-input v-model="system['bsc_api_key']" placeholder="请输入 BSC API Key，用于验证区块链交易"/>
                <div class="info">
                  <el-tooltip
                      effect="dark"
                      content="BSC API Key 用于验证区块链交易，可在 https://bscscan.com/apis 申请"
                      raw-content
                      placement="right"
                  >
                    <el-icon>
                      <InfoFilled/>
                    </el-icon>
                  </el-tooltip>
                </div>
              </div>
            </el-form-item>

            <el-form-item label="微信客服二维码" prop="wechat_card_url">
              <el-input v-model="system['wechat_card_url']" placeholder="微信客服二维码">
                <template #append>
                  <el-upload
                      :auto-upload="true"
                      :show-file-list="false"
                      @click="beforeUpload('wechat_card_url')"
                      :http-request="uploadImg"
                  >
                    <el-icon class="uploader-icon">
                      <UploadFilled/>
                    </el-icon>
                  </el-upload>
                </template>
              </el-input>
            </el-form-item>

          

            <el-form-item label="会员充值说明" prop="order_pay_timeout">
              <div class="tip-input">
                <el-input v-model="system['vip_info_text']" placeholder=""/>
                <div class="info">
                  <el-tooltip
                      effect="dark"
                      content="会员充值页面的充值说明文字"
                      raw-content
                      placement="right"
                  >
                    <el-icon>
                      <InfoFilled/>
                    </el-icon>
                  </el-tooltip>
                </div>
              </div>
            </el-form-item>

            <el-form-item label="默认AI模型" prop="default_models">
              <template #default>
                <div class="tip-input">
                  <el-select
                      v-model="system['default_models']"
                      multiple
                      :filterable="true"
                      placeholder="选择AI模型，多选"
                      style="width: 100%"
                  >
                    <el-option
                        v-for="item in models"
                        :key="item.id"
                        :label="item.name"
                        :value="item.id"
                    />
                  </el-select>
                  <div class="info">
                    <el-tooltip
                        class="box-item"
                        effect="dark"
                        content="新用户注册默认开通的 AI 模型"
                        placement="right"
                    >
                      <el-icon>
                        <InfoFilled/>
                      </el-icon>
                    </el-tooltip>
                  </div>
                </div>
              </template>
            </el-form-item>

            <el-form-item label="开启聊天上下文">
              <el-switch v-model="system['enable_context']"/>
            </el-form-item>
            <el-form-item label="会话上下文深度">
              <div class="tip-input-line">
                <el-input-number v-model="system['context_deep']" :min="0" :max="10"/>
                <div class="tip">会话上下文深度：在老会话中继续会话，默认加载多少条聊天记录作为上下文。如果设置为
                  0
                  则不加载聊天记录，仅仅使用当前角色的上下文。该配置参数最好设置需要为偶数，否则将无法兼容百度的 API。
                </div>
              </div>
            </el-form-item>
            <el-form-item label="联网搜索 API 地址">
              <el-input v-model="system['web_search_api_base']" placeholder="只填网关，如 https://api.deepseek.com（不要填 sk- 开头的 Key）"/>
            </el-form-item>
            <el-form-item label="联网搜索 API Key">
              <el-input v-model="system['web_search_api_key']" placeholder="sk-... 留空则使用 API KEY 管理中 DeepSeek 聊天 Key"/>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="save('system')">保存</el-button>
            </el-form-item>
          </el-form>
        </div>
      </el-tab-pane>
      <el-tab-pane label="公告配置" name="notice">
        <md-editor class="mgb20" v-model="notice" @on-upload-img="onUploadImg"/>
        <el-form-item>
          <el-button type="primary" @click="save('notice')">保存</el-button>
        </el-form-item>
      </el-tab-pane>

      <el-tab-pane label="PPT配置" name="ppt">
        <div class="container">
          <el-alert
            class="mgb20"
            type="info"
            show-icon
            :closable="false"
            title="PPT子系统参数配置"
          />
          <el-form :model="ppt" label-width="160px" label-position="right">
            <el-divider content-position="left">AI 提供商格式</el-divider>
            <el-form-item label="提供商格式">
              <el-select v-model="ppt['ppt_ai_provider_format']" placeholder="请选择" style="width: 100%">
                <el-option label="Gemini 格式" value="gemini"/>
                <el-option label="OpenAI 格式" value="openai"/>
              </el-select>
            </el-form-item>

            <el-divider content-position="left">文本大模型</el-divider>
            <el-form-item label="模型名称">
              <el-input v-model="ppt['ppt_text_model']" placeholder="例如 gemini-2.0-flash 或 gpt-4o-mini"/>
            </el-form-item>
            <el-form-item label="API Base URL">
              <el-input v-model="ppt['ppt_text_api_base_url']" placeholder="https://..."/>
            </el-form-item>
            <el-form-item label="API Key">
              <SecretKeyInput
                v-model="ppt['ppt_text_api_key']"
                config-key="ppt"
                field="ppt_text_api_key"
                placeholder="留空可在首次保存后仅更新其它项"
              />
            </el-form-item>

            <el-divider content-position="left">图像生成模型</el-divider>
            <el-form-item label="模型名称">
              <el-input v-model="ppt['ppt_image_model']" placeholder="例如 imagen-3.0-generate-001 或 dall-e-3"/>
            </el-form-item>
            <el-form-item label="API Base URL">
              <el-input v-model="ppt['ppt_image_api_base_url']" placeholder="可与文本不同，如使用独立生图网关"/>
            </el-form-item>
            <el-form-item label="API Key">
              <SecretKeyInput
                v-model="ppt['ppt_image_api_key']"
                config-key="ppt"
                field="ppt_image_api_key"
                placeholder="留空可在首次保存后仅更新其它项"
              />
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="save('ppt')">保存</el-button>
            </el-form-item>
          </el-form>
        </div>
      </el-tab-pane>

      <el-tab-pane label="等级分红配置" name="fenhong">
        <div class="container">
          <el-alert
            class="mgb20"
            type="info"
            show-icon
            :closable="false"
            title="等级分红（钻石等级）每日释放参数配置"
          />
          <el-form :model="fenhong" label-width="220px" label-position="right">
            <el-divider content-position="left">等级基础参数（对应等级参数 × 时间参数）</el-divider>
            <el-form-item label="1钻 基础参数">
              <el-input-number v-model="fenhong['fh_base_param_lv1']" :step="1" :precision="2" :min="0" controls-position="right" placeholder="例如 66666" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="2钻 基础参数">
              <el-input-number v-model="fenhong['fh_base_param_lv2']" :step="1" :precision="2" :min="0" controls-position="right" placeholder="例如 133332" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="3钻 基础参数">
              <el-input-number v-model="fenhong['fh_base_param_lv3']" :step="1" :precision="2" :min="0" controls-position="right" placeholder="请输入3钻基础参数" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="4钻 基础参数">
              <el-input-number v-model="fenhong['fh_base_param_lv4']" :step="1" :precision="2" :min="0" controls-position="right" placeholder="请输入4钻基础参数" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="5钻 基础参数">
              <el-input-number v-model="fenhong['fh_base_param_lv5']" :step="1" :precision="2" :min="0" controls-position="right" placeholder="请输入5钻基础参数" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="6钻 基础参数">
              <el-input-number v-model="fenhong['fh_base_param_lv6']" :step="1" :precision="2" :min="0" controls-position="right" placeholder="请输入6钻基础参数" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="7钻 基础参数">
              <el-input-number v-model="fenhong['fh_base_param_lv7']" :step="1" :precision="2" :min="0" controls-position="right" placeholder="请输入7钻基础参数" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="8钻 基础参数">
              <el-input-number v-model="fenhong['fh_base_param_lv8']" :step="1" :precision="2" :min="0" controls-position="right" placeholder="请输入8钻基础参数" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="9钻 基础参数">
              <el-input-number v-model="fenhong['fh_base_param_lv9']" :step="1" :precision="2" :min="0" controls-position="right" placeholder="请输入9钻基础参数" style="width: 260px"/>
            </el-form-item>

            <el-divider content-position="left">时间参数（分段区间同静态收益，10 天粒度，数值独立）</el-divider>

            <el-form-item label="0-10天 时间参数">
              <el-input-number v-model="fenhong['fh_zj_time']" :step="0.01" :precision="2" :min="0" controls-position="right" placeholder="例如 0.01" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="11-20天 时间参数">
              <el-input-number v-model="fenhong['fh_plus_11_20']" :step="0.01" :precision="2" :min="0" controls-position="right" placeholder="例如 0.08" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="21-30天 时间参数">
              <el-input-number v-model="fenhong['fh_plus_21_30']" :step="0.01" :precision="2" :min="0" controls-position="right" placeholder="例如 0.08" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="31-40天 时间参数">
              <el-input-number v-model="fenhong['fh_plus_31_40']" :step="0.01" :precision="2" :min="0" controls-position="right" placeholder="例如 0.05" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="41-50天 时间参数">
              <el-input-number v-model="fenhong['fh_plus_41_50']" :step="0.01" :precision="2" :min="0" controls-position="right" placeholder="例如 0.05" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="51-60天 时间参数">
              <el-input-number v-model="fenhong['fh_plus_51_60']" :step="0.01" :precision="2" :min="0" controls-position="right" placeholder="例如 0.05" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="61-70天 时间参数">
              <el-input-number v-model="fenhong['fh_plus_61_70']" :step="0.01" :precision="2" :min="0" controls-position="right" placeholder="例如 0.03" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="71-80天 时间参数">
              <el-input-number v-model="fenhong['fh_plus_71_80']" :step="0.01" :precision="2" :min="0" controls-position="right" placeholder="例如 0.03" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="81-90天 时间参数">
              <el-input-number v-model="fenhong['fh_plus_81_90']" :step="0.01" :precision="2" :min="0" controls-position="right" placeholder="例如 0.03" style="width: 260px"/>
            </el-form-item>
            <el-form-item label="91-100天+ 时间参数">
              <el-input-number v-model="fenhong['fh_plus_91_100']" :step="0.01" :precision="2" :min="0" controls-position="right" placeholder="例如 0.02" style="width: 260px"/>
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="save('fenhong')">保存</el-button>
            </el-form-item>
          </el-form>
        </div>
      </el-tab-pane>

      <el-tab-pane label="短剧 / TTS / 配图" name="teaching">
        <div class="container">
          <el-form :model="system" label-width="180px" label-position="right">
            <el-alert type="info" :closable="false" show-icon style="margin-bottom: 16px"
              title="短剧功能配置；保存后若未热加载请重启 API。" />
            <el-divider content-position="left">大模型</el-divider>
            <el-form-item label="口播 / 生图提示词模型">
              <el-select v-model.number="system['teaching_script_model_id']" filterable clearable placeholder="选择 Chat 模型" style="width: 100%">
                <el-option v-for="item in models" :key="item.id" :label="item.name + ' (id=' + item.id + ')'" :value="item.id" />
              </el-select>
            </el-form-item>
            <el-divider content-position="left">配图颗粒度</el-divider>
            <el-form-item label="一节一图（不拆条）">
              <el-switch v-model="system['teaching_visual_sparse']" :active-value="true" :inactive-value="false" />
            </el-form-item>
            <el-form-item label="每镜最大字数">
              <el-input-number v-model="system['teaching_visual_shot_max_runes']" :min="24" :max="120" :step="4" controls-position="right" />
            </el-form-item>
            <el-form-item label="全稿最多分镜条数">
              <el-input-number v-model="system['teaching_visual_max_total_shots']" :min="8" :max="48" controls-position="right" />
            </el-form-item>
            <el-form-item label="提示词 LLM 每批条数">
              <el-input-number v-model="system['teaching_image_prompts_llm_batch']" :min="4" :max="12" controls-position="right" />
            </el-form-item>
            <el-divider content-position="left">字节 OpenSpeech TTS</el-divider>
            <el-form-item label="X-Api-Key">
              <el-input v-model="system['tts_api_key']" type="password" show-password />
            </el-form-item>
            <el-form-item label="TTS 请求地址">
              <el-input v-model="system['tts_api_url']" placeholder="https://openspeech.bytedance.com/api/v3/tts/unidirectional" />
            </el-form-item>
            <el-form-item label="X-Api-Resource-Id">
              <el-input v-model="system['tts_resource_id']" placeholder="seed-tts-2.0" />
            </el-form-item>
            <el-form-item label="发音人 speaker">
              <el-input v-model="system['tts_speaker']" />
            </el-form-item>
            <el-form-item label="采样率">
              <el-input-number v-model="system['tts_sample_rate']" :min="8000" :max="48000" :step="1000" />
            </el-form-item>
            <el-divider content-position="left">TTS 资源包余量</el-divider>
            <el-alert type="info" :closable="false" show-icon style="margin-bottom: 12px"
              title="使用火山引擎 IAM 密钥查询 OpenSpeech 语音合成大模型资源包余量（与上方 X-Api-Key 合成密钥不同）。ProjectName 默认取「应用 AppID」。" />
            <el-form-item label="IAM AccessKey">
              <el-input v-model="system['tts_volc_access_key']" type="password" show-password placeholder="火山控制台 → 访问控制 → 访问密钥" />
            </el-form-item>
            <el-form-item label="IAM SecretKey">
              <el-input v-model="system['tts_volc_secret_key']" type="password" show-password />
            </el-form-item>
            <el-form-item label="应用 AppID">
              <el-input v-model="system['tts_app_id']" placeholder="豆包语音控制台应用 ID，作为 ProjectName 查询" />
            </el-form-item>
            <el-form-item label="">
              <el-button type="primary" plain :loading="ttsQuotaLoading" @click="fetchTtsQuota">查询 TTS 余量</el-button>
              <span v-if="ttsQuotaSummary" class="tts-quota-summary">{{ ttsQuotaSummary }}</span>
            </el-form-item>
            <el-table v-if="ttsQuotaPacks.length" :data="ttsQuotaPacks" border size="small" style="margin-bottom: 16px">
              <el-table-column prop="name" label="资源包" min-width="140" />
              <el-table-column prop="purchased" label="总量" width="100" />
              <el-table-column prop="used" label="已用" width="100" />
              <el-table-column prop="remaining" label="剩余" width="100" />
              <el-table-column prop="unit" label="单位" width="70" />
              <el-table-column prop="state" label="状态" width="90" />
              <el-table-column prop="expires" label="到期" min-width="120" />
            </el-table>
            <el-divider content-position="left">配图 API</el-divider>
            <el-form-item label="API 根地址">
              <el-input v-model="system['teaching_image_api_url']" />
            </el-form-item>
            <el-form-item label="Bearer API Key">
              <SecretKeyInput
                v-model="system['teaching_image_api_key']"
                config-key="system"
                field="teaching_image_api_key"
              />
            </el-form-item>
            <el-form-item label="生图模型">
              <el-input v-model="system['teaching_image_model']" />
            </el-form-item>
            <el-form-item label="尺寸 size">
              <el-input v-model="system['teaching_image_size']" placeholder="1024x1024" />
            </el-form-item>
            <el-form-item label="配图算力/张">
              <el-input-number v-model="system['prompt_power']" :min="0" controls-position="right" />
            </el-form-item>
            <el-form-item label="脚本/分镜 LLM 算力">
              <el-input-number v-model="system['teaching_script_power']" :min="0" controls-position="right" />
              <div class="form-tip">单次生成故事脚本或分镜提示词；0 表示沿用所选 Chat 模型算力</div>
            </el-form-item>
            <el-form-item label="TTS 算力/段">
              <el-input-number v-model="system['teaching_tts_power_per_segment']" :min="0" controls-position="right" />
              <div class="form-tip">每段口播配音；0 表示沿用「配图算力/张」或 MJ 算力</div>
            </el-form-item>
          </el-form>
        </div>
      </el-tab-pane>

      <el-tab-pane label="菜单配置" name="menu" lazy>
        <Menu/>
      </el-tab-pane>

      <!-- 系统用户节点示意图：独立懒加载，避免进页面就请求整棵树 + 主线程跑布局 -->
      <el-tab-pane label="节点示意图" name="placement" lazy>
        <PlacementTree/>
      </el-tab-pane>




    </el-tabs>
  </div>
</template>

<script setup>
import {defineAsyncComponent, onMounted, reactive, ref} from "vue";
import {httpGet, httpPost} from "@/utils/http";
import Compressor from "compressorjs";
import {ElMessage} from "element-plus";
import {InfoFilled, UploadFilled} from "@element-plus/icons-vue";
import MdEditor from "md-editor-v3";
import 'md-editor-v3/lib/style.css';
import Menu from "@/views/admin/Menu.vue";
import SecretKeyInput from "@/components/admin/SecretKeyInput.vue";

// 节点示意图体量大且只在切到对应 tab 时才需要：拆成独立 chunk 异步加载
const PlacementTree = defineAsyncComponent(() => import("@/components/admin/PlacementTree.vue"));

const activeName = ref('basic')

function teamWeeklyCapDefaults() {
  return {
    vip: 30000,
    d1: 100000,
    d2: 200000,
    d3: 300000,
    d4: 500000,
    d5: 1000000,
    d6: 2000000,
    d7: 3000000,
    d8: 5000000,
    d9: 10000000,
  }
}
const teamWeeklyCapFields = [
  { key: 'vip', label: 'VIP/激活会员' },
  { key: 'd1', label: '1 钻' },
  { key: 'd2', label: '2 钻' },
  { key: 'd3', label: '3 钻' },
  { key: 'd4', label: '4 钻' },
  { key: 'd5', label: '5 钻' },
  { key: 'd6', label: '6 钻' },
  { key: 'd7', label: '7 钻' },
  { key: 'd8', label: '8 钻' },
  { key: 'd9', label: '9 钻' },
]
const initSignInRewardDefaults = (s) => {
  if (typeof s !== 'object' || !s) return
  if (s.sign_in_reward_0_30 == null) s.sign_in_reward_0_30 = 10
  if (s.sign_in_reward_31_60 == null) s.sign_in_reward_31_60 = 5
  if (s.sign_in_reward_61_90 == null) s.sign_in_reward_61_90 = 3
  if (s.sign_in_reward_90_plus == null) s.sign_in_reward_90_plus = 0
}

const initTeachingVisualDefaults = (s) => {
  if (typeof s !== 'object' || !s) return
  if (s.teaching_visual_sparse === undefined || s.teaching_visual_sparse === null) s.teaching_visual_sparse = false
  if (s.teaching_visual_shot_max_runes == null) s.teaching_visual_shot_max_runes = 44
  if (s.teaching_visual_max_total_shots == null) s.teaching_visual_max_total_shots = 40
  if (s.teaching_image_prompts_llm_batch == null) s.teaching_image_prompts_llm_batch = 8
  if (s.prompt_power == null) s.prompt_power = 1
  if (s.teaching_script_power == null) s.teaching_script_power = 10
  if (s.teaching_tts_power_per_segment == null) s.teaching_tts_power_per_segment = 5
}

const fetchTtsQuota = () => {
  ttsQuotaLoading.value = true
  httpGet('/api/admin/teaching/tts-quota').then(res => {
    const d = res.data || {}
    ttsQuotaPacks.value = d.packs || []
    ttsQuotaSummary.value = d.summary || ''
    if (!ttsQuotaPacks.value.length) {
      ElMessage.warning(d.summary || '未查询到资源包')
    } else {
      ElMessage.success(d.summary || '查询成功')
    }
  }).catch(e => {
    ElMessage.error('查询失败：' + (e.message || e))
  }).finally(() => {
    ttsQuotaLoading.value = false
  })
}

const normalizeTeamWeeklyCap = (raw) => {
  const base = teamWeeklyCapDefaults()
  const out = { ...base }
  if (!raw || typeof raw !== 'object') {
    return out
  }
  teamWeeklyCapFields.forEach(({ key }) => {
    const v = parseFloat(raw[key])
    out[key] = isNaN(v) || v < 0 ? base[key] : v
  })
  return out
}

const investMultiplierDefaults = () => ({
  tier1_min: 1000,
  tier1_max: 49999,
  tier1_mul: 5,
  tier2_min: 50000,
  tier2_max: 99999,
  tier2_mul: 6,
  tier3_min: 100000,
  tier3_mul: 7,
})

const normalizeInvestMultiplier = (raw) => {
  const base = investMultiplierDefaults()
  const out = { ...base }
  if (!raw || typeof raw !== 'object') {
    return out
  }
  Object.keys(base).forEach((key) => {
    const v = parseFloat(raw[key])
    out[key] = isNaN(v) || v < 0 ? base[key] : v
  })
  return out
}

const system = ref({
  models: [],
  team_reward_weekly_cap: teamWeeklyCapDefaults(),
  invest_multiplier: investMultiplierDefaults(),
})

const pptDefaults = () => ({
  ppt_ai_provider_format: 'gemini',
  ppt_text_model: '',
  ppt_text_api_base_url: '',
  ppt_text_api_key: '',
  ppt_image_model: '',
  ppt_image_api_base_url: '',
  ppt_image_api_key: '',
})
const ppt = ref(pptDefaults())
const fenhongDefaults = () => ({
  fh_base_param_lv1: 66666,
  fh_base_param_lv2: 133332,
  fh_base_param_lv3: 0,
  fh_base_param_lv4: 0,
  fh_base_param_lv5: 0,
  fh_base_param_lv6: 0,
  fh_base_param_lv7: 0,
  fh_base_param_lv8: 0,
  fh_base_param_lv9: 0,
  fh_zj_time: 0,
  fh_plus_11_20: 0,
  fh_plus_21_30: 0,
  fh_plus_31_40: 0,
  fh_plus_41_50: 0,
  fh_plus_51_60: 0,
  fh_plus_61_70: 0,
  fh_plus_71_80: 0,
  fh_plus_81_90: 0,
  fh_plus_91_100: 0,
})
const fenhong = ref(fenhongDefaults())
const loading = ref(true)
const ttsQuotaLoading = ref(false)
const ttsQuotaPacks = ref([])
const ttsQuotaSummary = ref('')
const systemFormRef = ref(null)
const chatFormRef = ref(null)
const models = ref([])
const notice = ref("")

onMounted(() => {
  // 加载系统配置
  httpGet('/api/admin/config/get?key=system').then(res => {
    const data = res.data || {}
    system.value = {
      ...data,
      models: data.models ?? [],
      team_reward_weekly_cap: normalizeTeamWeeklyCap(data.team_reward_weekly_cap),
      invest_multiplier: normalizeInvestMultiplier(data.invest_multiplier),
    }
    if (system.value.transfer_fee_percent === undefined || system.value.transfer_fee_percent === null) {
      system.value.transfer_fee_percent = 5
    }
    if (system.value.enabled_transfer_fee === undefined || system.value.enabled_transfer_fee === null) {
      system.value.enabled_transfer_fee = true
    }
    if (system.value.transfer_fee_to_pool_percent === undefined || system.value.transfer_fee_to_pool_percent === null) {
      system.value.transfer_fee_to_pool_percent = 100
    } else if (Number(system.value.transfer_fee_to_pool_percent) === 50) {
      // 与后端一致：旧默认 50% 升级展示为 100%
      system.value.transfer_fee_to_pool_percent = 100
    }
    if (system.value.ai_search_search_power === undefined || system.value.ai_search_search_power === null) {
      system.value.ai_search_search_power = 1
    }
    if (system.value.ai_search_detail_power === undefined || system.value.ai_search_detail_power === null) {
      system.value.ai_search_detail_power = 10
    }
    if (system.value.team_reward_measure_layer_depth === undefined || system.value.team_reward_measure_layer_depth === null) {
      system.value.team_reward_measure_layer_depth = 4
    }
    if (system.value.team_reward_layer4_width_limit === undefined || system.value.team_reward_layer4_width_limit === null) {
      system.value.team_reward_layer4_width_limit = 8
    }
    if (system.value.team_reward_pro === undefined || system.value.team_reward_pro === null) {
      system.value.team_reward_pro = 0.2
    }
    if (system.value.swap_power_price === undefined || system.value.swap_power_price === null) {
      system.value.swap_power_price = 0.01
    }
    if (system.value.soha_guide_base_price === undefined || system.value.soha_guide_base_price === null) {
      system.value.soha_guide_base_price = 0.1
    }
    if (system.value.enabled_soha_market === undefined || system.value.enabled_soha_market === null) {
      system.value.enabled_soha_market = true
    }
    if (system.value.enabled_soha_kline === undefined || system.value.enabled_soha_kline === null) {
      system.value.enabled_soha_kline = true
    }
    if (system.value.enabled_register_ip_limit === undefined || system.value.enabled_register_ip_limit === null) {
      system.value.enabled_register_ip_limit = true
    }
    if (system.value.register_ip_limit === undefined || system.value.register_ip_limit === null) {
      system.value.register_ip_limit = 3
    }
    if (system.value.enabled_activation === undefined || system.value.enabled_activation === null) {
      system.value.enabled_activation = true
    }
    if (system.value.enabled_activation_user_limit === undefined || system.value.enabled_activation_user_limit === null) {
      system.value.enabled_activation_user_limit = true
    }
    if (system.value.activation_user_limit === undefined || system.value.activation_user_limit === null) {
      system.value.activation_user_limit = 3
    }
    if (system.value.jihuo_ex_min_power === undefined || system.value.jihuo_ex_min_power === null) {
      system.value.jihuo_ex_min_power = 1000
    }
    if (system.value.member_normal_min_power === undefined || system.value.member_normal_min_power === null) {
      system.value.member_normal_min_power = 1000
    }
    if (system.value.member_normal_max_power === undefined || system.value.member_normal_max_power === null) {
      system.value.member_normal_max_power = 9999
    }
    if (system.value.member_vip_min_power === undefined || system.value.member_vip_min_power === null) {
      system.value.member_vip_min_power = 10000
    }
    if (system.value.legacy_vip_min_power === undefined || system.value.legacy_vip_min_power === null) {
      system.value.legacy_vip_min_power = 5000
    }
    if (system.value.legacy_vip_team_reward_pro === undefined || system.value.legacy_vip_team_reward_pro === null) {
      system.value.legacy_vip_team_reward_pro = 0.1
    }
    // 授信老会员分界时刻刻意不给默认值：留空 = 不启用分段
    if (system.value.credit_legacy_cutoff === undefined || system.value.credit_legacy_cutoff === null) {
      system.value.credit_legacy_cutoff = ''
    }
    if (system.value.credit_legacy_min_power === undefined || system.value.credit_legacy_min_power === null) {
      system.value.credit_legacy_min_power = 10000
    }
    if (system.value.credit_legacy_max === undefined || system.value.credit_legacy_max === null) {
      system.value.credit_legacy_max = 50000
    }
    if (system.value.kj_zj_pro === undefined || system.value.kj_zj_pro === null) {
      system.value.kj_zj_pro = 0.3
    }
    if (system.value.kj_team_reward_pro === undefined || system.value.kj_team_reward_pro === null) {
      system.value.kj_team_reward_pro = 0.2
    }
    if (system.value.zj_static_pro === undefined || system.value.zj_static_pro === null) {
      system.value.zj_static_pro = 0.2
    }
    if (system.value.mgmt_static_pro === undefined || system.value.mgmt_static_pro === null) {
      system.value.mgmt_static_pro = 0.1
    }
    system.value.invest_multiplier = normalizeInvestMultiplier(system.value.invest_multiplier)
    initSignInRewardDefaults(system.value)
    initTeachingVisualDefaults(system.value)
    if (system.value.limit_cancel === undefined || system.value.limit_cancel === null) {
      system.value.limit_cancel = 0
    }
  }).catch(e => {
    ElMessage.error("加载系统配置失败: " + e.message)
  })
  // 加载聊天配置
  httpGet('/api/admin/config/get?key=notice').then(res => {
    notice.value = res.data['content']
  }).catch(e => {
    ElMessage.error("公告信息失败: " + e.message)
  })

  httpGet('/api/admin/model/list').then(res => {
    models.value = res.data
    loading.value = false
  }).catch(e => {
    ElMessage.error("获取模型失败：" + e.message)
  })

  httpGet('/api/admin/config/get?key=ppt').then(res => {
    ppt.value = { ...pptDefaults(), ...res.data }
  }).catch(() => {
    ppt.value = pptDefaults()
  })

  httpGet('/api/admin/config/get?key=fenhong').then(res => {
    fenhong.value = { ...fenhongDefaults(), ...res.data }
  }).catch(() => {
    fenhong.value = fenhongDefaults()
  })

})

const rules = reactive({
  title: [{required: true, message: '请输入网站标题', trigger: 'blur',}],
  admin_title: [{required: true, message: '请输入控制台标题', trigger: 'blur',}],
  init_chat_calls: [{required: true, message: '请输入赠送对话次数', trigger: 'blur'}],
  user_img_calls: [{required: true, message: '请输入赠送绘图次数', trigger: 'blur'}],
})
const save = function (key) {
  if (key === 'system') {
    systemFormRef.value.validate((valid) => {
      if (valid) {
        system.value['power_price'] = parseFloat(system.value['power_price']) ?? 0
        const tfp = parseFloat(system.value['transfer_fee_percent'])
        system.value['transfer_fee_percent'] = isNaN(tfp) ? 5 : Math.min(100, Math.max(0, tfp))
        const tftpp = parseFloat(system.value['transfer_fee_to_pool_percent'])
        system.value['transfer_fee_to_pool_percent'] = isNaN(tftpp) ? 100 : Math.min(100, Math.max(0, tftpp))
        const assp = parseInt(system.value['ai_search_search_power'], 10)
        system.value['ai_search_search_power'] = isNaN(assp) || assp < 0 ? 1 : assp
        const asdp = parseInt(system.value['ai_search_detail_power'], 10)
        system.value['ai_search_detail_power'] = isNaN(asdp) || asdp < 0 ? 10 : asdp
        const trmd = parseInt(system.value['team_reward_measure_layer_depth'], 10)
        system.value['team_reward_measure_layer_depth'] = isNaN(trmd) || trmd < 1 ? 4 : Math.min(99, trmd)
        const trwl = parseInt(system.value['team_reward_layer4_width_limit'], 10)
        system.value['team_reward_layer4_width_limit'] = isNaN(trwl) || trwl < 2 ? 8 : Math.min(65536, trwl)
        const trp = parseFloat(system.value['team_reward_pro'])
        system.value['team_reward_pro'] = isNaN(trp) || trp < 0 ? 0.2 : Math.min(1, trp) // 允许 0（关闭）
        const kzp = parseFloat(system.value['kj_zj_pro'])
        system.value['kj_zj_pro'] = isNaN(kzp) || kzp < 0 ? 0.3 : Math.min(1, kzp)
        const ktrp = parseFloat(system.value['kj_team_reward_pro'])
        system.value['kj_team_reward_pro'] = isNaN(ktrp) || ktrp < 0 ? 0.2 : Math.min(1, ktrp)
        const zsp = parseFloat(system.value['zj_static_pro'])
        system.value['zj_static_pro'] = isNaN(zsp) || zsp < 0 ? 0.2 : Math.min(1, zsp)
        const msp = parseFloat(system.value['mgmt_static_pro'])
        system.value['mgmt_static_pro'] = isNaN(msp) || msp < 0 ? 0.1 : Math.min(1, msp)
        const lc = parseInt(system.value['limit_cancel'], 10)
        system.value['limit_cancel'] = isNaN(lc) || lc < 0 ? 0 : lc
        system.value.team_reward_weekly_cap = normalizeTeamWeeklyCap(system.value.team_reward_weekly_cap)
        system.value.invest_multiplier = normalizeInvestMultiplier(system.value.invest_multiplier)
        initTeachingVisualDefaults(system.value)
        httpPost('/api/admin/config/update', {key: key, config: system.value}).then(() => {
          ElMessage.success("操作成功！")
        }).catch(e => {
          ElMessage.error("操作失败：" + e.message)
        })
      }
    })
  } else if (key === 'chat') {
    if (chat.value.context_deep % 2 !== 0) {
      return ElMessage.error("会话上下文深度必须为偶数！")
    }
    chatFormRef.value.validate((valid) => {
      if (valid) {
        httpPost('/api/admin/config/update', {key: key, config: chat.value}).then(() => {
          ElMessage.success("操作成功！")
        }).catch(e => {
          ElMessage.error("操作失败：" + e.message)
        })
      }
    })
  } else if (key === 'notice') {
    httpPost('/api/admin/config/update', {key: key, config: {content: notice.value, updated: true}}).then(() => {
      ElMessage.success("操作成功！")
    }).catch(e => {
      ElMessage.error("操作失败：" + e.message)
    })
  } else if (key === 'ppt') {
    httpPost('/api/admin/config/update', {key: 'ppt', config: ppt.value}).then(() => {
      ElMessage.success("操作成功！")
    }).catch(e => {
      ElMessage.error("操作失败：" + e.message)
    })
  } else if (key === 'fenhong') {
    httpPost('/api/admin/config/update', {key: 'fenhong', config: fenhong.value}).then(() => {
      ElMessage.success("操作成功！")
    }).catch(e => {
      ElMessage.error("操作失败：" + e.message)
    })
  }
}

const configKey = ref("")
const beforeUpload = (key) => {
  configKey.value = key
}

// 图片上传
const uploadImg = (file) => {
  // 压缩图片并上传
  new Compressor(file.file, {
    quality: 0.6,
    success(result) {
      const formData = new FormData();
      formData.append('file', result, result.name);
      // 执行上传操作
      httpPost('/api/admin/upload', formData).then((res) => {
        system.value[configKey.value] = res.data.url
        ElMessage.success('上传成功')
      }).catch((e) => {
        ElMessage.error('上传失败:' + e.message)
      })
    },
    error(e) {
      ElMessage.error('上传失败:' + e.message)
    },
  });
};

// 编辑期文件上传处理
const onUploadImg = (files, callback) => {
  Promise.all(
      files.map((file) => {
        return new Promise((rev, rej) => {
          const formData = new FormData();
          formData.append('file', file, file.name);
          // 执行上传操作
          httpPost('/api/admin/upload', formData).then((res) => rev(res)).catch((error) => rej(error));
        });
      })
  ).then(res => {
    ElMessage.success({message: "上传成功", duration: 500})
    callback(res.map((item) => item.data.url));
  }).catch(e => {
    ElMessage.error('图片上传失败:' + e.message)
  })


};

</script>

<style lang="stylus" scoped>
@import "@/assets/css/admin/form.styl"
.system-config {
  display flex
  justify-content center

  .el-tabs {
    width 100%
    background-color #ffffff
    padding 10px 20px 40px 20px
    border: 1px solid #ddd;
    border-radius: 5px

    .container {
      .el-form {
        .el-form-item__content {

          .tip-text {
            padding-left 10px;
          }

          .el-icon {
            font-size 16px
            margin-left 10px
            cursor pointer
          }

          .uploader-icon {
            font-size 24px
            position relative
            top 3px
          }

          .tip-input-line {
            .tip {
              margin-top 10px
              color #c1c1c1
              font-size 12px;
              line-height 1.5;
            }
          }

          .team-weekly-cap-grid {
            .cap-row {
              display flex
              align-items center
              margin-bottom 8px
              .cap-label {
                width 120px
                flex-shrink 0
              }
            }
            .tip {
              margin-top 10px
              color #c1c1c1
              font-size 12px
              line-height 1.5
            }
          }
        }
      }

      .el-alert {
        margin-bottom 15px;
      }

      .tts-quota-summary {
        margin-left 12px;
        font-size 13px;
        color #606266;
      }
    }

  }
}
</style>