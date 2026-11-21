<template>
  <div class="invitation-page custom-scroll">
    <div class="gradient-bg"></div>

    <div class="layout">
      <!-- 顶部信息卡 -->
      <div class="glass-card hero-card">

        <!-- 品牌标识（已停用）
        <div class="brand-row">
          <div class="brand-bar">
            <div class="brand-main">
              <template v-if="isEditingBrand">
                <input
                  v-model="form.div"
                  class="brand-name-input"
                  @blur="saveBrandEdit"
                  @keyup.enter="saveBrandEdit"
                  maxlength="10"
                  autofocus
                  :placeholder="t('invitation.brandPlaceholder')"
                />
              </template>
              <template v-else>
                <button
                  v-if="isProfileLoaded"
                  type="button"
                  class="brand-name-btn"
                  :class="{ 'is-empty': !form.div }"
                  @click.stop="startEditBrand"
                >
                  <span class="brand-name-text">{{ form.div || t('invitation.brandPlaceholder') }}</span>
                  <van-icon name="edit" size="14" class="brand-name-edit" />
                </button>
                <div v-else class="brand-name-skeleton"></div>
              </template>
            </div>
          </div>
        </div>
        -->

        <!-- 支付宝风格：扫一扫 / 转账 / 收取码 / 分享（语言在分享下方） -->
        <div class="alipay-nav">
          <div
            v-for="item in alipayNavActions"
            :key="item.key"
            class="alipay-nav__col"
            :class="{ 'alipay-nav__col--share': item.key === 'share' }"
          >
            <button
              type="button"
              class="alipay-nav__item"
              :class="'alipay-nav__item--' + item.tone"
              @click.stop="runQuickAction(item)"
            >
              <span class="alipay-nav__icon">
                <van-icon :name="item.icon" />
              </span>
              <span class="alipay-nav__label">{{ item.label }}</span>
            </button>

            <div
              v-if="item.key === 'share'"
              class="lang-dropdown"
              @click.stop
            >
              <button
                type="button"
                class="lang-chip"
                :title="t('invitation.language')"
                @click="toggleLangDropdown"
              >
                <img class="lang-chip__flag" :src="currentLangFlag" alt="" aria-hidden="true" />
                <span class="lang-chip__text">{{ t('invitation.language') }}</span>
                <van-icon
                  name="arrow-down"
                  size="12"
                  class="lang-chip__arrow"
                  :class="{ 'lang-chip__arrow--open': showLangDropdown }"
                />
              </button>
              <div v-show="showLangDropdown" class="lang-menu">
                <button
                  v-for="lang in LANG_LIST"
                  :key="lang.value"
                  type="button"
                  class="lang-menu__item"
                  :class="{ 'lang-menu__item--active': locale === lang.value }"
                  @click="switchLocale(lang.value)"
                >
                  <img class="lang-menu__flag" :src="lang.flag" :alt="lang.label" />
                  <span class="lang-menu__label">{{ lang.label }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <div class="hero-body">
          <div class="avatar-block" :class="{ 'avatar-square': avatarShape === 'square', 'avatar-rectangle': avatarShape === 'rectangle' }">
            <van-uploader
              v-model="fileList"
              reupload
              max-count="1"
              :deletable="false"
              :after-read="afterRead"
              :class="['avatar-uploader', { 'avatar-square': avatarShape === 'square', 'avatar-rectangle': avatarShape === 'rectangle' }]"
            />
            <!-- 头像形状切换按钮 -->
            <div class="avatar-shape-toggle" @click.stop="toggleAvatarShape" :title="getAvatarShapeTitle()">
              <van-icon :name="getAvatarShapeIcon()" size="18" />
            </div>
          </div>

          <div class="user-meta">
            <div class="meta-header">
              <!-- 第一行：用户名 -->
              <div class="meta-row meta-row-username">
                <div class="meta-label">{{ t('invitation.username') }}</div>
                <div class="meta-value user-name-wrapper">
                  <div class="user-name" :title="form.username">{{ form.username }}</div>
                </div>
              </div>
              <!-- 第二行：等级 + 类型 + 签到，表格式三列 -->
              <div class="meta-row meta-row-grid">
                <div class="meta-cell">
                  <!-- <div class="meta-cell-label">等级</div> -->
                  <div class="meta-cell-value">
                    <span
                      class="user-level-tag"
                      v-if="form.lev !== undefined && form.lev !== null"
                      @click.stop="showLevelRightsDialog"
                    >
                      {{ currentDiamondText }}
                    </span>
                    <span
                      v-else
                      class="user-level-tag level-inactive"
                    >
                      {{ t('invitation.levelInactive') }}
                    </span>
                  </div>
                </div>

                <div class="meta-cell">
                  <!-- <div class="meta-cell-label">类型</div> -->
                  <div class="meta-cell-value">
                    <div class="user-type" @click.stop="showUserTypeTipDialog" title="点击查看说明">
                      <span
                        class="tag"
                        :class="form.vip ? 'vip active' : 'normal active'"
                      >
                        {{ form.vip ? t('invitation.merchant') : t('invitation.user') }}
                      </span>
                    </div>
                  </div>
                </div>

                <div class="meta-cell meta-cell-action">
                  <!-- <div class="meta-cell-label">签到</div> -->
                  <div class="meta-cell-value">
                    <van-button
                      size="mini"
                      type="primary"
                      class="signin-btn"
                      :class="{ 'signin-btn-done': form.signed_today }"
                      @click.stop="openSignInCalendar"
                    >
                      {{ form.signed_today ? t('invitation.signedToday') : t('invitation.signin') }}
                    </van-button>
                  </div>
                </div>
              </div>
            </div>

            <!-- 签到日历 -->
            <van-popup
              v-model:show="showSignInCalendar"
              round
              position="center"
              :style="{ width: '86%', maxWidth: '360px' }"
            >
              <div class="signin-calendar">
                <div class="signin-calendar-head">
                  <span class="signin-calendar-title">{{ t('invitation.signInCalendarTitle') }}</span>
                  <van-icon name="cross" size="18" @click="showSignInCalendar = false" />
                </div>
                <div class="signin-calendar-sub">
                  <span>{{ signInCalendar.year }}-{{ String(signInCalendar.month).padStart(2, '0') }}</span>
                  <span v-if="signInCalendar.streak > 0" class="signin-streak">
                    {{ t('invitation.signInStreakText', { n: signInCalendar.streak }) }}
                  </span>
                </div>
                <div class="signin-weekdays">
                  <span v-for="(w, i) in signInWeekdays" :key="i">{{ w }}</span>
                </div>
                <div class="signin-days">
                  <span
                    v-for="(cell, i) in signInDays"
                    :key="i"
                    class="signin-day"
                    :class="{ 'is-empty': !cell.day, 'is-signed': cell.signed, 'is-today': cell.isToday }"
                  >
                    <template v-if="cell.day">
                      <span class="signin-day-num">{{ cell.day }}</span>
                      <span v-if="cell.signed" class="signin-day-check" aria-hidden="true">✓</span>
                    </template>
                  </span>
                </div>
                <van-button
                  block
                  type="primary"
                  round
                  class="signin-calendar-btn"
                  :loading="signInLoading"
                  :disabled="form.signed_today || signInLoading"
                  @click="getFuli"
                >
                  {{ form.signed_today ? t('invitation.signedToday') : t('invitation.signin') }}
                </van-button>
              </div>
            </van-popup>

            <!-- 用户等级信息 -->
            <!-- <div class="user-level-info">
              <div class="level-info-item">
                <div class="label">等级</div>
                <div class="value">{{ form.lev }}</div>
              </div>
            </div> -->

            <!-- 算力信息布局：2列grid（待产出、余额 | 已产出、复投） -->
            <div class="power-balance-container">
              <div class="power-balance-top-grid">
                <div class="power-balance-item" :class="{ 'exited-card': isPendingExited }">
                  <template v-if="isPendingExited">
                    <div class="value positive exited only-text">
                      {{ t('invitation.levelInactive') }}
                    </div>
                  </template>
                  <template v-else>
                    <div class="label">{{ t('invitation.pendingPower') }}
                      <van-icon 
                      name="search" 
                      size="16" 
                      class="search-icon"
                      @click.stop="goToOrderList('pending')"
                    />

                    </div>
                    <div class="value positive">
                      <span>{{ pendingPower }}</span>
                    </div>
                  </template>
                </div>
          

                <div class="power-balance-item">
                  <div class="label">
                    {{ t('invitation.producedPower') }}
                    <van-icon 
                      name="search" 
                      size="16" 
                      class="search-icon"
                      @click.stop="goToOrderList('exited')"
                    />
                  </div>
                  <div class="value positive">
                    <span>{{ producedPower }}</span>
                  </div>
                </div>


                <div class="power-balance-item power-balance-main">
                  <div class="label">{{ t('invitation.availablePower') }}</div>
                  <div class="value positive">
                    <span>{{ availablePower }}</span>
                  </div>
                </div>

                <!-- 复投输入框 -->
                <div class="power-balance-item power-balance-reinvest">
                  <div class="reinvest-input-wrapper">
                    <van-field
                      v-model="reinvestPower"
                      type="number"
                      :placeholder="t('invitation.reinvestPlaceholder')"
                      input-align="right"
                      :disabled="reinvestLoading"
                      class="reinvest-input-field"
                      maxlength="6"
                    />
                    <van-button
                      size="small"
                      type="primary"
                      round
                      class="reinvest-confirm-btn"
                      :loading="reinvestLoading"
                      :disabled="reinvestLoading"
                      @click.stop="handleConfirmReinvest"
                    >
                      {{
                        reinvestLoading
                          ? t('invitation.reinvestProcessing')
                          : t('invitation.reinvestConfirm')
                      }}
                    </van-button>
                  </div>
                </div>
              </div>

              <!-- 授信额度（简洁三列） -->
              <div class="credit-card">
                <div class="credit-card__head">
                  <div class="credit-card__title">
                    {{ t('invitation.creditTitle') }}
                    <van-icon
                      name="orders-o"
                      size="16"
                      class="credit-card__detail-icon"
                      @click.stop="openCreditDetail"
                    />
                  </div>
                  <div class="credit-card__actions">
                    <button type="button" class="credit-card__action credit-card__action--ghost" @click.stop="openCreditDetail">
                      {{ t('invitation.creditDetail') }}
                    </button>
                    <button type="button" class="credit-card__action" @click.stop="openCreditGiftSheet()">
                      {{ t('invitation.creditGift') }}
                    </button>
                  </div>
                </div>
                <div class="credit-card__row">
                  <div class="credit-card__cell">
                    <span>{{ t('invitation.creditMax') }}</span>
                    <strong>{{ formatCredit(creditMax) }}</strong>
                  </div>
                  <div class="credit-card__cell">
                    <span>{{ t('invitation.creditAvailable') }}</span>
                    <strong class="is-avail">{{ formatCredit(creditAvailable) }}</strong>
                  </div>
                  <div class="credit-card__cell">
                    <span>{{ t('invitation.creditDebt') }}</span>
                    <strong class="is-debt">{{ formatCredit(creditDebt) }}</strong>
                  </div>
                </div>

                <!-- 
                <div class="credit-card__hint">
                  {{
                    creditLegacy
                      ? t('invitation.creditLegacyHint', { base: formatCredit(creditLegacyBase), min: formatCredit(creditLegacyMin), max: formatCredit(creditMax) })
                      : t('invitation.creditHint')
                  }}
                </div>
                -->
              </div>
            </div>

            <!-- 账户信息 -->
            <div class="account-info">
              <div class="account-info-item">
                <div class="info-label">{{ t('invitation.email') }}</div>
                <div class="info-value">
                  <span v-if="!form.mail" class="link" @click="showMailDialog = true">{{ t('invitation.bindEmail') }}</span>
                  <span v-else>{{ hiddenMail }}</span>
                  <van-icon 
                    name="plus" 
                    size="14" 
                    color="#fff" 
                    class="add-icon-circle"
                    @click="showMailDialog = true"
                    :title="t('invitation.bindEmail')"
                  />
                </div>
              </div>
              <div class="account-info-item">
                <div class="info-label">{{ t('invitation.paymentMethod') }}</div>
                <div class="info-value">
                  <span v-if="form.default_payment_type === 'bsc'">
                    {{ t('invitation.paymentDefaultBsc') }}
                  </span>
                  <span v-else-if="form.default_payment_type === 'alipay'">
                    {{ t('invitation.paymentDefaultAlipay') }}
                  </span>
                  <span v-else class="empty-address">{{ t('invitation.notSet') }}</span>
                  <van-icon 
                    name="plus" 
                    size="14" 
                    color="#fff" 
                    class="add-icon-circle"
                    @click="openPaymentDialog"
                    :title="t('invitation.paymentMethod')"
                  />
                </div>
              </div>
              <div class="account-info-item">
                <div class="info-label">{{ t('invitation.transferCode') }}</div>
                <div class="info-value emphasis">
                  {{ userS.zzcode }}
                  <el-icon class="copy-icon" @click="copyTransferCode" title="复制转账码">
                    <DocumentCopy />
                  </el-icon>
                </div>
              </div>
            </div>


            
          </div>
        </div>
      </div>

      <!-- 快捷功能 -->
      <div class="glass-card quick-actions-card">
        <div class="team-overview-head">
          <div class="team-overview-head__main">
            <h3 class="team-overview-title">{{ t('invitation.quickActions') }}</h3>
          </div>
        </div>
        <div class="quick-actions-grid">
          <button
            v-for="item in quickActions"
            :key="item.key"
            type="button"
            class="quick-action"
            :class="'quick-action--' + item.tone"
            @click.stop="runQuickAction(item)"
          >
            <span class="quick-action__icon">
              <van-icon :name="item.icon" />
              <em
                v-if="item.key === 'contact' && csUnread > 0"
                class="quick-action__badge"
              >{{ csUnread > 99 ? '99+' : csUnread }}</em>
            </span>
            <span class="quick-action__label">{{ item.label }}</span>
          </button>
        </div>
      </div>

      <!-- 团队统计 -->
      <div class="card-grid">
        <div class="glass-card team-overview-card">
          <div class="team-overview-head">
            <div class="team-overview-head__main">
      
              <h3 class="team-overview-title">{{ t('invitation.teamOverview') }}</h3>
            </div>
            <div class="team-referrer-chip">
              <van-icon name="friends-o" />
              <span class="team-referrer-chip__label">{{ t('invitation.referrer') }}</span>
              <strong class="team-referrer-chip__value">{{ referrerName || t('invitation.none') }}</strong>
            </div>
          </div>

          <div class="team-metrics">
            <div class="team-metric team-metric--direct">
              <div class="team-metric__icon"><van-icon name="contact" /></div>
              <div class="team-metric__value">{{ regNum }}</div>
              <div class="team-metric__label">{{ t('invitation.direct') }}</div>
            </div>
            <div class="team-metric team-metric--active">
              <div class="team-metric__icon"><van-icon name="checked" /></div>
              <div class="team-metric__value">{{ jhNum }}</div>
              <div class="team-metric__label">{{ t('invitation.activated') }}</div>
            </div>
            <div class="team-metric team-metric--team">
              <div class="team-metric__icon"><van-icon name="cluster-o" /></div>
              <div class="team-metric__value">{{ hits }}</div>
              <div class="team-metric__label">{{ t('invitation.teamCount') }}</div>
            </div>
          </div>

          <div class="team-zone">
            <div class="team-zone__head">
              <span class="team-zone__title">{{ t('invitation.zonePerfTitle') }}</span>
              <span class="team-zone__total">
                {{ t('invitation.totalPerf') }}
                <em>{{ formatTeamNumber(totalZonePerf) }}</em>
              </span>
            </div>
            <div class="team-zone__track" aria-hidden="true">
              <div class="team-zone__fill team-zone__fill--left" :style="{ width: leftPerfPct + '%' }"></div>
              <div class="team-zone__fill team-zone__fill--right" :style="{ width: rightPerfPct + '%' }"></div>
            </div>
            <div class="team-zone__cols">
              <div class="team-zone__col team-zone__col--left">
                <div class="team-zone__col-top">
                  <span class="team-zone__dot"></span>
                  <span>{{ t('invitation.leftPerf') }}</span>
                </div>
                <div class="team-zone__num">{{ formatTeamNumber(leftPerf) }}</div>

              </div>
              <div class="team-zone__col team-zone__col--right">
                <div class="team-zone__col-top">
                  <span class="team-zone__dot"></span>
                  <span>{{ t('invitation.rightPerf') }}</span>
                </div>
                <div class="team-zone__num">{{ formatTeamNumber(rightPerf) }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 列表 -->
      <div class="glass-card list-card team-users-card">
        <div class="team-users-head">
          <div class="team-users-head__main">
            <h3 class="team-overview-title">{{ t('invitation.directUsers') }}</h3>
          </div>
          <span class="team-users-count" v-if="regNum > 0">{{ regNum }}</span>
        </div>
        <div class="list-body">
          <invite-list v-if="isLogin" />
        </div>
      </div>

      <div class="bottom-spacing"></div>
    </div>
  </div>

  <login-dialog :show="showLoginDialog" @hide="showLoginDialog =  false" @success="initData"/>

  <van-dialog
    v-model:show="showPowerDetail"
    :title="t('invitation.powerDetailTitle')"
    show-cancel-button
    :show-confirm-button="false"
    cancel-button-text="关闭"
    class="power-detail-dialog"
    @cancel="showPowerDetail = false"
  >
    <div class="power-detail-body" v-loading="powerDetailLoading">
      <div class="power-summary">
        <div class="summary-card">
          <div class="summary-label">{{ t('invitation.currentAvailablePower') }}</div>
          <div class="summary-value">{{ availablePower }}</div>
        </div>
        <div class="summary-card">
          <div class="summary-label">{{ t('invitation.pendingPower2') }}</div>
          <div class="summary-value">{{ pendingPower }}</div>
        </div>
      </div>

      <div class="records-header">
        <span>{{ t('invitation.recordsTitle') }}</span>
        <div class="records-actions">
          <van-button size="mini" plain type="primary" :loading="powerDetailLoading" @click="fetchPowerLogs">{{ t('invitation.refresh') }}</van-button>
          <van-button size="mini" type="primary" @click="goPowerLog">{{ t('invitation.viewAll') }}</van-button>
        </div>
      </div>

      <div v-if="powerLogs.length" class="record-list">
        <div class="record-item" v-for="item in powerLogs" :key="item.id">
          <div class="record-title">
            <span class="model">{{ formatPowerLogModel(t, item.model) }}</span>
            <span class="amount" :class="{'positive': item.mark === 1, 'negative': item.mark === 0}">
              {{ item.mark === 1 ? '+' : '-' }}{{ item.amount }}
            </span>
          </div>
          <div class="record-meta">
            <span class="balance">余额 {{ item.balance }}</span>
            <span class="time">{{ formatPowerTime(item.created_at) }}</span>
          </div>
        </div>
      </div>
      <van-empty v-else :description="t('invitation.noRecords')" />
    </div>
  </van-dialog>

  <!-- 修改密码对话框 -->
  <van-dialog 
    v-model:show="showPasswordDialog" 
    :title="false"
    :show-cancel-button="false"
    :show-confirm-button="false"
    class="change-password-dialog-custom"
    :close-on-click-overlay="true">
    <div class="change-password-dialog-wrapper">
      <!-- 自定义标题栏 -->
      <div class="change-password-dialog-header">
        <div class="change-password-icon-wrapper">
          <van-icon name="edit" size="28" color="#fff" />
        </div>
        <h3 class="change-password-title">{{ t('invitation.changePasswordDialogTitle') }}</h3>
        <van-icon name="cross" size="20" color="#95a5a6" class="close-icon" @click="showPasswordDialog = false" />
      </div>
      
      <!-- 表单内容 -->
      <div class="change-password-dialog-content">
        <van-form>
          <div class="change-password-field-wrapper">
            <div class="field-label">
              <van-icon name="lock" size="16" color="#667eea" />
              <span>{{ t('invitation.changePasswordOld') }}</span>
            </div>
            <van-field
              v-model="pass.old"
              type="password"
              :placeholder="t('invitation.changePasswordOldPlaceholder')"
              class="change-password-field"
            />
          </div>
          
          <div class="change-password-field-wrapper">
            <div class="field-label">
              <van-icon name="lock" size="16" color="#f093fb" />
              <span>{{ t('invitation.changePasswordNew') }}</span>
            </div>
            <van-field
              v-model="pass.new"
              type="password"
              :placeholder="t('invitation.changePasswordNewPlaceholder')"
              class="change-password-field"
            />
          </div>
          
          <div class="change-password-field-wrapper">
            <div class="field-label">
              <van-icon name="checked" size="16" color="#4facfe" />
              <span>{{ t('invitation.changePasswordNewConfirm') }}</span>
            </div>
            <van-field
              v-model="pass.renew"
              type="password"
              :placeholder="t('invitation.changePasswordNewConfirmPlaceholder')"
              class="change-password-field"
            />
          </div>
        </van-form>
      </div>
      
      <!-- 自定义按钮 -->
      <div class="change-password-dialog-footer">
        <van-button 
          class="change-password-btn cancel-btn" 
          @click="showPasswordDialog = false">
          {{ t('invitation.changePasswordBtnCancel') }}
        </van-button>
        <van-button 
          class="change-password-btn confirm-btn" 
          @click="updatePass"
          type="primary">
          {{ t('invitation.changePasswordBtnConfirm') }}
        </van-button>
      </div>
    </div>
  </van-dialog>


  <van-popup
    v-model:show="transferDialog"
    position="bottom"
    round
    :close-on-click-overlay="true"
    :safe-area-inset-bottom="true"
    class="wx-pay-sheet"
    teleport="body"
  >
    <div class="wx-pay">
      <div class="wx-pay__handle" />
      <div class="wx-pay__nav">
        <span class="wx-pay__nav-title">{{ t('invitation.transferDialogTitle') }}</span>
        <van-icon name="cross" size="18" class="wx-pay__close" @click="transferDialog = false" />
      </div>

      <div class="wx-pay__to">
        <span class="wx-pay__to-label">{{ t('invitation.transferUidLabel') }}</span>
        <div class="wx-pay__to-row">
          <input
            v-model="transfer.uid"
            class="wx-pay__to-input"
            :placeholder="t('invitation.transferUidPlaceholder')"
          />
          <van-icon
            name="scan"
            size="20"
            class="wx-pay__scan"
            @click.stop="startScanFromTransfer"
          />
        </div>
      </div>

      <div class="wx-pay__amount-block">
        <div class="wx-pay__amount-label">{{ t('invitation.transferPowerLabel') }}</div>
        <div class="wx-pay__amount-row">
          <input
            v-model="transfer.power"
            class="wx-pay__amount-input"
            type="number"
            inputmode="decimal"
            :placeholder="'0.00'"
          />
        </div>
      </div>

      <div v-if="transferPreview && transferFeeEnabled" class="wx-pay__fee">
        {{ t('invitation.transferFeeSummary', { pct: transferPreview.pct, fee: transferPreview.fee, recv: transferPreview.recv, total: transferPreview.total }) }}
      </div>
      <div v-if="transferPreview && transferPreview.insufficient" class="wx-pay__warn">
        <van-icon name="warning-o" />
        <span>{{ t('invitation.transferFeeInsufficient', { total: transferPreview.total, balance: transferPreview.balance }) }}</span>
      </div>

      <div class="wx-pay__pwd">
        <van-icon name="lock" size="16" class="wx-pay__pwd-icon" />
        <input
          v-model="authPassword"
          class="wx-pay__pwd-input"
          type="password"
          :placeholder="t('invitation.passwordPlaceholder')"
          autocomplete="current-password"
        />
      </div>

      <van-button
        block
        round
        type="primary"
        class="wx-pay__submit"
        :loading="transferSubmitting"
        :disabled="transferSubmitting"
        @click="submitTransferPay"
      >
        {{ t('invitation.transferBtnConfirm') }}
      </van-button>
    </div>
  </van-popup>


  <!-- 赠送激活：底部弹层 -->
  <van-popup
    v-model:show="giftDialog"
    position="bottom"
    round
    :close-on-click-overlay="true"
    :safe-area-inset-bottom="true"
    class="wx-pay-sheet"
    teleport="body"
  >
    <div class="wx-pay wx-pay--gift">
      <div class="wx-pay__handle" />
      <div class="wx-pay__nav">
        <span class="wx-pay__nav-title">{{ t('invitation.giftDialogTitle') }}</span>
        <van-icon name="cross" size="18" class="wx-pay__close" @click="giftDialog = false" />
      </div>

      <div class="wx-pay__to">
        <span class="wx-pay__to-label">{{ t('invitation.giftCodeLabel') }}</span>
        <div class="wx-pay__to-row">
          <input
            v-model="gift.code"
            class="wx-pay__to-input"
            :placeholder="t('invitation.giftCodePlaceholder')"
          />
          <van-icon
            name="scan"
            size="20"
            class="wx-pay__scan"
            @click.stop="startScanFromGift"
          />
        </div>
      </div>

      <div class="wx-pay__amount-block">
        <div class="wx-pay__amount-label">{{ t('invitation.giftPowerLabel') }}</div>
        <div class="wx-pay__amount-row">
          <input
            v-model="gift.power"
            class="wx-pay__amount-input"
            type="number"
            inputmode="decimal"
            :placeholder="'0.00'"
          />
        </div>
      </div>

      <div v-if="giftPreview" class="wx-pay__fee">
        {{ t('invitation.giftFeeSummary', { pct: giftPreview.pct, fee: giftPreview.fee, recv: giftPreview.recv, total: giftPreview.total }) }}
      </div>
      <div v-if="giftPreview?.insufficient" class="wx-pay__fee wx-pay__fee--warn">
        <span>{{ t('invitation.giftFeeInsufficient', { total: giftPreview.total, balance: giftPreview.balance }) }}</span>
      </div>

      <div class="wx-pay__pwd">
        <van-icon name="lock" size="16" class="wx-pay__pwd-icon" />
        <input
          v-model="authPassword"
          class="wx-pay__pwd-input"
          type="password"
          :placeholder="t('invitation.passwordPlaceholder')"
          autocomplete="current-password"
        />
      </div>

      <van-button
        block
        round
        type="primary"
        class="wx-pay__submit wx-pay__submit--gift"
        :loading="giftSubmitting"
        :disabled="giftSubmitting"
        @click="submitGiftPay"
      >
        {{ t('invitation.giftBtnConfirm') }}
      </van-button>
    </div>
  </van-popup>

  <!-- 授信额度明细 -->
  <van-popup
    v-model:show="creditDetailDialog"
    position="bottom"
    round
    :style="{ height: '72%' }"
    class="credit-detail-popup"
  >
    <div class="credit-detail">
      <div class="credit-detail__nav">
        <span class="credit-detail__title">{{ t('invitation.creditDetailTitle') }}</span>
        <van-icon name="cross" size="18" class="credit-detail__close" @click="creditDetailDialog = false" />
      </div>
      <div class="credit-detail__summary">
        <div class="credit-detail__cell">
          <span>{{ t('invitation.creditMax') }}</span>
          <strong>{{ formatCredit(creditMax) }}</strong>
        </div>
        <div class="credit-detail__cell">
          <span>{{ t('invitation.creditAvailable') }}</span>
          <strong class="is-avail">{{ formatCredit(creditAvailable) }}</strong>
        </div>
        <div class="credit-detail__cell">
          <span>{{ t('invitation.creditDebt') }}</span>
          <strong class="is-debt">{{ formatCredit(creditDebt) }}</strong>
        </div>
      </div>
      <div class="credit-detail__list" v-loading="creditDetailLoading">
        <div v-if="creditLogs.length" class="credit-detail__items">
          <div class="credit-detail__item" v-for="item in creditLogs" :key="item.id">
            <div class="credit-detail__item-head">
              <span class="credit-detail__model">{{ formatPowerLogModel(t, item.model) }}</span>
              <span class="credit-detail__amount" :class="{ positive: item.mark === 1, negative: item.mark === 0 }">
                {{ item.mark === 1 ? '+' : '-' }}{{ formatCredit(item.amount) }}
              </span>
            </div>
            <div class="credit-detail__remark" v-if="item.remark">{{ item.remark }}</div>
            <div class="credit-detail__meta">
              <span>{{ creditLogBalanceLabel(item) }} {{ formatCredit(item.balance) }}</span>
              <span>{{ formatPowerTime(item.created_at) }}</span>
            </div>
          </div>
          <div class="credit-detail__more" v-if="creditLogsHasMore">
            <van-button size="small" plain type="primary" :loading="creditDetailLoading" @click="fetchCreditLogs(true)">
              {{ t('invitation.creditDetailLoadMore') }}
            </van-button>
          </div>
        </div>
        <van-empty v-else :description="t('invitation.creditDetailEmpty')" />
      </div>
    </div>
  </van-popup>

  <!-- 授信报单：底部弹层（免手续费，扣授信可用额度） -->
  <van-popup
    v-model:show="creditGiftDialog"
    position="bottom"
    round
    :close-on-click-overlay="true"
    :safe-area-inset-bottom="true"
    class="wx-pay-sheet"
    teleport="body"
  >
    <div class="wx-pay wx-pay--gift">
      <div class="wx-pay__handle" />
      <div class="wx-pay__nav">
        <span class="wx-pay__nav-title">{{ t('invitation.creditGiftDialogTitle') }}</span>
        <van-icon name="cross" size="18" class="wx-pay__close" @click="creditGiftDialog = false" />
      </div>

      <div class="wx-pay__fee">
        {{ t('invitation.creditGiftBalance', { avail: formatCredit(creditAvailable), max: formatCredit(creditMax), debt: formatCredit(creditDebt) }) }}
      </div>

      <div class="wx-pay__to">
        <span class="wx-pay__to-label">{{ t('invitation.giftCodeLabel') }}</span>
        <div class="wx-pay__to-row">
          <input
            v-model="creditGift.code"
            class="wx-pay__to-input"
            :placeholder="t('invitation.giftCodePlaceholder')"
          />
          <van-icon
            name="scan"
            size="20"
            class="wx-pay__scan"
            @click.stop="startScanFromCreditGift"
          />
        </div>
      </div>

      <div class="wx-pay__amount-block">
        <div class="wx-pay__amount-label">{{ t('invitation.creditGiftPowerLabel') }}</div>
        <div class="wx-pay__amount-row">
          <input
            v-model="creditGift.power"
            class="wx-pay__amount-input"
            type="number"
            inputmode="decimal"
            :placeholder="'0.00'"
          />
        </div>
      </div>

      <div v-if="creditGiftPreview" class="wx-pay__fee">
        {{ t('invitation.creditGiftFeeSummary', { recv: creditGiftPreview.recv }) }}
      </div>
      <!-- 报单范围提示：后端仅允许报给推荐伞下的用户 -->
      <div class="wx-pay__fee wx-pay__fee--hint">
        {{ t('invitation.creditGiftScopeHint') }}
      </div>
      <div v-if="creditGiftPreview?.insufficient" class="wx-pay__fee wx-pay__fee--warn">
        <span>{{ t('invitation.creditGiftInsufficient', { need: creditGiftPreview.recv, avail: creditGiftPreview.balance }) }}</span>
      </div>

      <div class="wx-pay__pwd">
        <van-icon name="lock" size="16" class="wx-pay__pwd-icon" />
        <input
          v-model="authPassword"
          class="wx-pay__pwd-input"
          type="password"
          :placeholder="t('invitation.passwordPlaceholder')"
          autocomplete="current-password"
        />
      </div>

      <van-button
        block
        round
        type="primary"
        class="wx-pay__submit wx-pay__submit--gift"
        :loading="creditGiftSubmitting"
        :disabled="creditGiftSubmitting"
        @click="submitCreditGiftPay"
      >
        {{ t('invitation.creditGiftBtnConfirm') }}
      </van-button>
    </div>
  </van-popup>





   <!-- 弹出公告对话框（「我的」页：公告配置非空时展示） -->
        <van-dialog
          v-model:show="showAnnouncement"
          class="announcement-dialog"
          :title="t('invitation.announcementTitle')"
          show-cancel-button
          cancel-button-text="不再显示"
          confirm-button-text="我知道了"
          @confirm="handleConfirm"
          @cancel="handleDoNotShow"
          width="80%"
          style="--van-dialog-title-font-size: 22px; --van-dialog-message-padding: 20px 15px;"
          >
          <div class="announcement-content">
            <div v-html="announcements"></div>
          </div>
        </van-dialog>



        <!-- 邮箱绑定弹出框 -->
  <van-dialog
   v-model:show="showMailDialog"
   :title="t('invitation.emailDialogTitle')"
   show-cancel-button
   @confirm="updateMail"
   @cancel="showMailDialog = false "
   :confirm-button-text="t('invitation.bindEmailBtnConfirm')"
   :cancel-button-text="t('invitation.bindEmailBtnCancel')"
   :before-close="beforeClose">
    <van-form>
      <van-cell-group inset>
        <van-field
          v-model="bind.mail"
          :placeholder="t('invitation.emailPlaceholder')"
        />

     
      </van-cell-group>
    </van-form>
  </van-dialog>

  <!-- ETH地址编辑弹窗 -->
  <van-dialog 
   v-model:show="showEthAddressDialog" 
   :title="t('invitation.ethDialogTitle')" 
   show-cancel-button
   @confirm="saveEthAddress"
   @cancel="showEthAddressDialog = false"
   :confirm-button-text="t('invitation.bindEmailBtnConfirm')"
   :cancel-button-text="t('invitation.bindEmailBtnCancel')"
   width="90%"
   class="eth-address-dialog"
  >
    <div class="eth-address-content">
      <van-field
        v-model="ethAddressForm"
        :placeholder="t('invitation.ethPlaceholder')"
        :label="t('invitation.ethFieldLabel')"
        class="eth-address-field"
        maxlength="100"
      />
      <div class="eth-address-tip">
        <van-icon name="info-o" size="14" color="#4facfe" />
        <span>{{ t('invitation.ethTip') }}</span>
      </div>
    </div>
  </van-dialog>

  <!-- 收款方式管理弹窗 -->
  <van-dialog 
    v-model:show="showPaymentDialog" 
    :title="t('invitation.paymentDialogTitle')" 
    show-cancel-button
    :cancel-button-text="t('invitation.bindEmailBtnCancel')"
    :confirm-button-text="t('invitation.bindEmailBtnConfirm')"
    @confirm="savePaymentMethods"
    @cancel="showPaymentDialog = false"
    width="90%"
    class="payment-dialog"
  >
    <div class="payment-methods-content">
      <!-- BSC收款方式 -->
      <div class="payment-method-item" :class="{ 'is-default': paymentForm.default_payment_type === 'bsc' }">
        <div class="payment-method-header">
          <div class="payment-method-title">
            <van-icon name="balance-o" size="20" color="#4facfe" />
            <span>{{ t('invitation.paymentBscTitle') }}</span>
            <van-icon 
              v-if="paymentForm.default_payment_type === 'bsc'" 
              name="checked" 
              size="18" 
              color="#4facfe" 
              class="default-icon"
            />
          </div>
          <van-checkbox 
            :model-value="paymentForm.default_payment_type === 'bsc'"
            @update:model-value="(val) => handlePaymentTypeChange('bsc', val)"
            shape="square"
          >
            {{ t('invitation.paymentSetDefault') }}
          </van-checkbox>
        </div>
        <van-field
          v-model="paymentForm.eth_address"
        :placeholder="t('invitation.paymentBscPlaceholder')"
        :label="t('invitation.ethFieldLabel')"
          class="payment-field"
        />
      </div>

      <!-- 支付宝收款方式 -->
      <div class="payment-method-item" :class="{ 'is-default': paymentForm.default_payment_type === 'alipay' }">
        <div class="payment-method-header">
          <div class="payment-method-title">
            <van-icon name="alipay" size="20" color="#1677ff" />
            <span>{{ t('invitation.paymentAlipayTitle') }}</span>
            <van-icon 
              v-if="paymentForm.default_payment_type === 'alipay'" 
              name="checked" 
              size="18" 
              color="#1677ff" 
              class="default-icon"
            />
          </div>
          <van-checkbox 
            :model-value="paymentForm.default_payment_type === 'alipay'"
            @update:model-value="(val) => handlePaymentTypeChange('alipay', val)"
            shape="square"
          >
            {{ t('invitation.paymentSetDefault') }}
          </van-checkbox>
        </div>
        <van-field
          v-model="paymentForm.alipay_account"
          :placeholder="t('invitation.paymentAlipayPlaceholder')"
          :label="t('invitation.paymentAlipayLabel')"
          class="payment-field"
        />
      </div>

      <!-- 当前默认收款方式提示 -->
      <div class="default-payment-tip">
        <van-icon name="info-o" size="14" color="#4facfe" />
        <span>
          {{ t('invitation.paymentDefaultPrefix') }}
          {{ paymentForm.default_payment_type === 'bsc'
            ? t('invitation.paymentDefaultBsc')
            : t('invitation.paymentDefaultAlipay') }}
        </span>
      </div>
    </div>
  </van-dialog>

  <!-- 回购弹出框 -->
  <van-dialog v-model:show="showBuybackDialog" :title="t('invitation.buybackDialogTitle')" show-cancel-button
              @confirm="submitBuyback"
              @cancel="showBuybackDialog = false"
              :before-close="beforeClose">
    <van-form>
      <van-cell-group inset>
        <van-field
            readonly
            :model-value="currentPowerBalance"
            :label="t('invitation.buybackCurrentPower')"
            placeholder="0"
        />
        <van-field
            v-model="buyback.power"
            type="number"
            :placeholder="t('invitation.buybackPowerPlaceholder')"
            :label="t('invitation.buybackPowerLabel')"
        />
        <van-field
            readonly
            :model-value="buybackAmount"
            :label="t('invitation.buybackAmountLabel')"
            :placeholder="t('invitation.buybackAmountPlaceholder')"
        />
        <van-field
            readonly
            :model-value="currentEthAddress"
            :label="t('invitation.buybackEthLabel')"
            :placeholder="t('invitation.buybackEthPlaceholder')"
        />
        <div style="padding: 10px 16px; color: #969799; font-size: 12px;">
          <p style="margin: 5px 0;">{{ t('invitation.buybackIntroTitle') }}</p>
          <p style="margin: 5px 0;">{{ t('invitation.buybackIntro1') }}</p>
          <p style="margin: 5px 0;">{{ t('invitation.buybackIntro2') }}</p>
          <p style="margin: 5px 0;">{{ t('invitation.buybackIntro3') }}</p>
        </div>
      </van-cell-group>
    </van-form>
  </van-dialog>

  <!-- 我的收取码 -->
  <van-dialog
    v-model:show="showReceiveQrDialog"
    :title="t('invitation.receiveQrDialogTitle')"
    :show-confirm-button="false"
    :show-cancel-button="false"
    class="receive-qr-dialog"
    :close-on-click-overlay="true">
    <div class="receive-qr-wrapper">
      <van-icon name="cross" size="20" color="#95a5a6" class="receive-qr-close" @click="showReceiveQrDialog = false" />
      <div v-if="transferQrImg" class="receive-qr-img-box">
        <el-image :src="transferQrImg" class="receive-qr-img" fit="contain" />
      </div>
      <div v-else class="receive-qr-empty">{{ t('invitation.receiveQrEmpty') }}</div>
      <p class="receive-qr-code">{{ userS.zzcode }}</p>
      <van-button type="primary" block class="receive-qr-copy-btn" @click="copyTransferCode">
        {{ t('invitation.copyTransferCodeBtn') }}
      </van-button>
    </div>
  </van-dialog>

  <!-- 分享：分享码 + 推广链接 -->
  <van-dialog
    v-model:show="showShareDialog"
    :title="t('invitation.shareDialogTitle')"
    :show-confirm-button="false"
    :show-cancel-button="false"
    class="receive-qr-dialog share-qr-dialog"
    :close-on-click-overlay="true">
    <div class="receive-qr-wrapper">
      <van-icon name="cross" size="20" color="#95a5a6" class="receive-qr-close" @click="showShareDialog = false" />
      <div v-if="transferQrImg" class="receive-qr-img-box">
        <el-image :src="transferQrImg" class="receive-qr-img" fit="contain" />
      </div>
      <div v-else class="receive-qr-empty">{{ t('invitation.shareQrEmpty') }}</div>
      <p class="receive-qr-code">{{ userS.zzcode }}</p>
      <van-button type="primary" block class="receive-qr-copy-btn" @click="copyTransferCode">
        {{ t('invitation.copyShareCodeBtn') }}
      </van-button>
      <div class="receive-invite-block">
        <div class="receive-invite-label">{{ t('invitation.inviteLinkLabel') }}</div>
        <div class="receive-invite-row">
          <span class="receive-invite-link">{{ inviteURL || '-' }}</span>
          <van-button size="small" plain type="primary" class="receive-invite-copy" @click="copyInviteURL">
            {{ t('invitation.copyInviteLinkBtn') }}
          </van-button>
        </div>
      </div>
    </div>
  </van-dialog>

  <!-- 扫码弹出框 -->
  <van-dialog
    v-model:show="showScanDialog"
    :title="t('invitation.scanDialogTitle')"
    show-cancel-button
    @confirm="handleScanResult"
    @cancel="closeScan"
    @closed="stopScanCamera"
    :confirm-button-text="t('invitation.scanManual')"
    :cancel-button-text="t('invitation.transferBtnCancel')"
    :before-close="beforeClose">
    <div class="scan-container">
      <video ref="videoRef" autoplay playsinline class="scan-video"></video>
      <div class="scan-overlay">
        <div class="scan-box">
          <div class="scan-line"></div>
        </div>
        <p class="scan-tip">{{ t('invitation.scanTip') }}</p>
        <div class="scan-actions">
          <van-button size="small" type="primary" @click="switchCamera" v-if="hasMultipleCameras">
            {{ t('invitation.scanSwitchCamera') }}
          </van-button>
          <van-button size="small" @click="showManualInput = true">
            {{ t('invitation.scanManual') }}
          </van-button>
        </div>
      </div>
    </div>
  </van-dialog>

  <!-- 手动输入弹出框 -->
  <van-dialog
    v-model:show="showManualInput"
    :title="t('invitation.scanManualDialogTitle')"
    show-cancel-button
    @confirm="handleManualInput"
    @cancel="showManualInput = false">
    <van-field
      v-model="manualInputValue"
      :placeholder="t('invitation.scanManualPlaceholder')"
      clearable
    />
  </van-dialog>

  <!-- 节点信息弹出框（入口已注释，保留弹窗代码备用）
  <van-dialog
    v-model:show="showNodeDialog"
    :title="t('invitation.nodeDialogTitle')"
    show-cancel-button
    @confirm="showNodeDialog = false"
    @cancel="showNodeDialog = false">
    <div class="node-info">
      <p>{{ t('invitation.nodeStatusNormal') }}</p>
      <p>{{ t('invitation.nodeStatusClosed') }}</p>
    </div>
  </van-dialog>
  -->

  <!-- 复投弹窗 -->
  <van-dialog
    v-model:show="reinvestDialog"
    :title="t('invitation.reinvestDialogTitle')"
    :close-on-click-overlay="!reinvestLoading"
    :close-on-popstate="!reinvestLoading"
  >
    <template #footer>
      <div class="dialog-footer-custom">
        <van-button 
          v-if="!reinvestLoading"
          block 
          type="default" 
          @click="reinvestDialog = false"
        >
          {{ t('invitation.reinvestDialogBtnCancel') }}
        </van-button>
        <van-button 
          block 
          type="primary" 
          :loading="reinvestLoading"
          :disabled="reinvestLoading"
          @click="handleConfirmReinvest"
        >
          {{ reinvestLoading ? '复投中...' : '✔' }}
        </van-button>
      </div>
    </template>
    <div class="reinvest-dialog-content">
      <div class="reinvest-info">
        <div class="info-label">{{ t('invitation.reinvestInfoLabel') }}</div>
        <div class="info-value">{{ availablePower }}</div>
      </div>
      
      <van-field
        v-model="reinvestPower"
        type="number"
        :placeholder="t('invitation.reinvestPlaceholderShort')"
        input-align="right"
        clearable
        :disabled="reinvestLoading"
        :max="Math.min(availablePower, REINVEST_POWER_MAX)"
      >
        <template #button>
          <div class="power-unit">算力</div>
        </template>
      </van-field>
      
      <!-- 快捷输入按钮 -->
      <div class="quick-input-btns">
        <van-button 
          size="small" 
          type="primary" 
          plain
          :disabled="reinvestLoading || availablePower <= 0"
          @click="setQuickPower(0.25)"
        >
          {{ t('invitation.reinvestBtnQuarter') }}
        </van-button>
        <van-button 
          size="small" 
          type="primary" 
          plain
          :disabled="reinvestLoading || availablePower <= 0"
          @click="setQuickPower(0.5)"
        >
          {{ t('invitation.reinvestBtnHalf') }}
        </van-button>
        <van-button 
          size="small" 
          type="primary" 
          plain
          :disabled="reinvestLoading || availablePower <= 0"
          @click="setQuickPower(1)"
        >
          {{ t('invitation.reinvestBtnAll') }}
        </van-button>
      </div>
      
      <!-- Loading 状态 -->
      <div v-if="reinvestLoading" class="reinvest-loading">
        <van-loading type="spinner" size="20px" color="#1989fa">复投中...</van-loading>
      </div>
      
      <!-- 提示信息 -->
      <div v-if="reinvestPower && Number(reinvestPower) > 0" class="reinvest-preview">
        <div class="preview-item">
          <span class="preview-label">复投后剩余算力：</span>
          <span class="preview-value">{{ Math.max(0, availablePower - Number(reinvestPower)) }} </span>
        </div>
      </div>
    </div>
  </van-dialog>

</template>

<script setup>
import {onMounted, onActivated, onUnmounted, ref,computed, nextTick, watch} from "vue"
import { useI18n } from "vue-i18n"
import { setLocale } from "@/i18n"
import QRCode from "qrcode";
import jsQR from "jsqr";
import {httpGet, httpPost} from "@/utils/http";
import { formatPowerLogModel } from "@/utils/powerLogI18n";
import {ElMessage} from "element-plus";
import Clipboard from "clipboard";
import InviteList from "@/components/InviteList.vue";
import {checkSession} from "@/action/session";
import LoginDialog from "@/components/LoginDialog.vue";
import {showConfirmDialog,showFailToast, showNotify, showSuccessToast, showToast} from "vant";
import {removeUserToken} from "@/store/session";
import { playCsNotifySound } from "@/utils/csNotify";
import Compressor from 'compressorjs';
import {useRouter} from "vue-router";
import { DocumentCopy } from '@element-plus/icons-vue';
import {dateFormat} from "@/utils/libs";

defineOptions({ name: 'Invitation' })

const { t, locale } = useI18n()
const showLangDropdown = ref(false)
// 使用 public 静态国旗，避免 Windows emoji / webpack svg 兼容问题
const LANG_LIST = [
  { flag: '/img/flags/cn.svg', value: 'zh-CN', label: '中文' },
  { flag: '/img/flags/us.svg', value: 'en-US', label: 'English' },
  { flag: '/img/flags/jp.svg', value: 'ja', label: '日本語' },
  { flag: '/img/flags/kr.svg', value: 'kr-KR', label: '한국어' },
]
const currentLangFlag = computed(() => {
  const hit = LANG_LIST.find((item) => item.value === locale.value)
  return hit?.flag || '/img/flags/cn.svg'
})
const toggleLangDropdown = () => {
  showLangDropdown.value = !showLangDropdown.value
}
const closeLangDropdown = () => {
  if (showLangDropdown.value) showLangDropdown.value = false
}
const switchLocale = (val) => {
  closeLangDropdown()
  if (!val || locale.value === val) return
  setLocale(val)
  // 等语言切换完成后再刷新，避免同一次更新里并发改 form 触发异常
  nextTick(() => {
    try {
      fetchUserProfile()
    } catch (e) {
      console.error(e)
    }
  })
}
watch(showLangDropdown, (open) => {
  if (open) {
    nextTick(() => document.addEventListener('click', closeLangDropdown))
  } else {
    document.removeEventListener('click', closeLangDropdown)
  }
})
onUnmounted(() => {
  document.removeEventListener('click', closeLangDropdown)
})

const inviteURL = ref("")
const qrImg = ref("")
const transferQrImg = ref("")
const inviteChatCalls = ref(0)
const inviteImgCalls = ref(0)
const hits = ref(0)
const regNum = ref(0)
const jhNum = ref(0)
const rate = ref(0)
const isLogin = ref(false)

// 仅分享入口：未激活不可分享推广码/外链
const isUserActivated = computed(() => {
  const jh = form.value?.is_jh
  if (jh === 1 || jh === '1' || jh === true) return true
  const tier = form.value?.member_tier
  return !!tier && tier !== 'inactive'
})

const csUnread = ref(0)
let csUnreadTimer = null
let csUnreadPrev = 0

const fetchCsUnread = async (playSound = true) => {
  if (!isLogin.value) {
    csUnread.value = 0
    return
  }
  try {
    const url = form.value?.is_cs ? '/api/cs/agent/unread' : '/api/cs/unread'
    const res = await httpGet(url)
    const n = Number(res.data?.unread || 0)
    if (playSound && n > csUnreadPrev) {
      playCsNotifySound()
    }
    csUnreadPrev = n
    csUnread.value = n
  } catch {
    // ignore
  }
}

const startCsUnreadPoll = () => {
  stopCsUnreadPoll()
  fetchCsUnread(false)
  csUnreadTimer = setInterval(() => fetchCsUnread(true), 12000)
}

const stopCsUnreadPoll = () => {
  if (csUnreadTimer) {
    clearInterval(csUnreadTimer)
    csUnreadTimer = null
  }
}
const referrerName = ref('')
const showLoginDialog = ref(false)
const router = useRouter()
const zzcode=ref("")
const totalYeji = ref(0)
const validYeji = ref(0)
const leftPerf = ref(0)
const rightPerf = ref(0)

const totalZonePerf = computed(() => Number(leftPerf.value || 0) + Number(rightPerf.value || 0))
const leftPerfPct = computed(() => {
  const total = totalZonePerf.value
  if (total <= 0) return 50
  return Math.round((Number(leftPerf.value || 0) / total) * 100)
})
const rightPerfPct = computed(() => {
  const total = totalZonePerf.value
  if (total <= 0) return 50
  return Math.max(0, 100 - leftPerfPct.value)
})
const formatTeamNumber = (val) => {
  const n = Number(val || 0)
  if (!Number.isFinite(n)) return '0'
  try {
    return n.toLocaleString()
  } catch {
    return String(n)
  }
}



const latexPlugin = require('markdown-it-latex2img')
const mathjaxPlugin = require('markdown-it-mathjax')
const md = require('markdown-it')({
  breaks: true,
  html: true,
  linkify: true,
  typographer: true,
  highlight: function (str, lang) {
    const codeIndex = parseInt(Date.now()) + Math.floor(Math.random() * 10000000)
    // 显示复制代码按钮
    const copyBtn = `<span class="copy-code-btn" data-clipboard-action="copy" data-clipboard-target="#copy-target-${codeIndex}">复制</span>
<textarea style="position: absolute;top: -9999px;left: -9999px;z-index: -9999;" id="copy-target-${codeIndex}">${str.replace(/<\/textarea>/g, '&lt;/textarea>')}</textarea>`
    
    // 处理代码高亮
    const preCode = md.utils.escapeHtml(str)
    // 将代码包裹在 pre 中
    return `<pre class="code-container"><code class="language-${lang} hljs">${preCode}</code>${copyBtn}</pre>`
  }
});

md.use(latexPlugin)
md.use(mathjaxPlugin)

// 头像列表：不要预置一个空 url 的条目，否则 van-uploader 会先渲染“方形占位图标”
const fileList = ref([]);

// 邮箱隐藏（只隐藏 @ 前面的用户名部分，域名保持不变）
const maskEmailLocalPart = (email) => {
  const value = (email || '').trim();
  if (!value) return '';

  const atIndex = value.indexOf('@');
  if (atIndex <= 0) return value; // 非标准邮箱，直接返回

  const local = value.slice(0, atIndex);
  const domain = value.slice(atIndex);
  const len = local.length;

  // 规则：
  // - local < 6：隐藏第一位
  // - local > 10：隐藏中间四位
  // - 其它：隐藏中间两位（如 123456 -> 隐藏 34）
  if (len < 6) {
    return `*${local.slice(1)}${domain}`;
  }

  if (len > 10) {
    const start = Math.floor((len - 4) / 2);
    return `${local.slice(0, start)}****${local.slice(start + 4)}${domain}`;
  }

  // 6~10
  const start = Math.floor((len - 2) / 2);
  return `${local.slice(0, start)}**${local.slice(start + 2)}${domain}`;
};

const hiddenMail = computed(() => maskEmailLocalPart(form.value.mail || ''));





const showDialog = ref(false); // 控制弹出框
const newMail = ref(''); // 存储输入的邮箱
const showUserTypeTip = ref(false); // 控制用户类型提示显示

const submitEmail = () => {
  if (newMail.value) {
    form.mail = newMail.value; // 更新邮箱
    showDialog.value = false; // 关闭弹出框
    newMail.value = ''; // 清空输入框
  }
};

const showUserTypeTipDialog = () => {
  const tipText = form.value.vip 
    ? t('invitation.userTypeMerchantTip')
    : t('invitation.userTypeUserTip');
  showToast({
    message: tipText,
    duration: 3000,
    position: 'top',
  });
};

// 层数 ↔ 钻级 映射（按产品规则）
// - 激活会员：10层
// - 1钻：20层，2钻：30层，...，9钻：100层
const DIAMOND_LAYER_RULES = [
  { diamond: 0, layers: 10 },
  { diamond: 1, layers: 20 },
  { diamond: 2, layers: 30 },
  { diamond: 3, layers: 40 },
  { diamond: 4, layers: 50 },
  { diamond: 5, layers: 60 },
  { diamond: 6, layers: 70 },
  { diamond: 7, layers: 80 },
  { diamond: 8, layers: 90 },
  { diamond: 9, layers: 100 },
];

const getDiamondLabel = (rule) => {
  if (!rule) return t('invitation.levelInactive');
  if (rule.diamond === 0) {
    return t('invitation.levelVip');
  }
  return `${rule.diamond}${t('inviteList.diamondSuffix')}`;
};

const layersToDiamond = (layers) => {
  const v = Number(layers || 0);
  for (let i = DIAMOND_LAYER_RULES.length - 1; i >= 0; i--) {
    const rule = DIAMOND_LAYER_RULES[i];
    if (v >= rule.layers) return rule;
  }
  return null;
};

const memberTierDisplayText = () => {
  const tier = form.value?.member_tier;
  if (tier === 'normal') return t('invitation.levelNormal');
  if (tier === 'vip' || tier === 'legacy_vip') return t('invitation.levelVip');
  if (form.value?.member_tier_label) return form.value.member_tier_label;
  return t('invitation.levelInactive');
};

// 使用 floor（层数）来判断等级，lev 是等级（0-9），floor 是层数（10, 20, 30...）
const currentDiamondRule = computed(() => layersToDiamond(form.value?.floor));
const currentDiamondText = computed(() => {
  const lev = Number(form.value?.lev ?? 0);
  if (lev >= 1) {
    const rule = currentDiamondRule.value;
    if (rule) return getDiamondLabel(rule);
  }
  if (Number(form.value?.floor ?? 0) <= 0 || form.value?.member_tier === 'inactive') {
    return t('invitation.levelInactive');
  }
  return memberTierDisplayText();
});

// 等级权益提示（不再展示安置层数，与团队奖励是否限层无关）
const showLevelRightsDialog = () => {
  const rule = currentDiamondRule.value;
  const levelText = rule ? getDiamondLabel(rule) : t('invitation.levelInactive');
  const tipText = `${t('invitation.levelTipPrefix')}${levelText}`;
  showToast({
    message: tipText,
    duration: 3000,
    position: 'top',
  });
};


const form = ref({
  div: '', // 品牌名称
  username: '',
  nickname: '',
  mobile: '',
  avatar: '',
  calls: 0,
  tokens: 0,
  eth_address: '',
  alipay_account: '', // 支付宝账号
  default_payment_type: 'bsc' // 默认收款方式: 'bsc' 或 'alipay'
})

// 标记用户资料是否已加载完成，用于避免初始闪烁
const isProfileLoaded = ref(false)

// 头像形状：'circle' 圆形、'square' 方形 或 'rectangle' 长方形
const avatarShape = ref(localStorage.getItem('avatarShape') || 'circle')

// 切换头像形状：圆形 -> 方形 -> 长方形 -> 圆形
const toggleAvatarShape = () => {
  if (avatarShape.value === 'circle') {
    avatarShape.value = 'square'
  } else if (avatarShape.value === 'square') {
    avatarShape.value = 'rectangle'
  } else {
    avatarShape.value = 'circle'
  }
  localStorage.setItem('avatarShape', avatarShape.value)
}

// 获取头像形状切换按钮的标题
const getAvatarShapeTitle = () => {
  if (avatarShape.value === 'circle') {
    return '切换为方形'
  } else if (avatarShape.value === 'square') {
    return '切换为长方形'
  } else {
    return '切换为圆形'
  }
}

// 获取头像形状切换按钮的图标
const getAvatarShapeIcon = () => {
  if (avatarShape.value === 'circle') {
    return 'photo-o'
  } else if (avatarShape.value === 'square') {
    return 'photo'
  } else {
    return 'photo-o'
  }
}

const userS = ref({
  zzcode:''
})

// ETH地址编辑
const showEthAddressDialog = ref(false)
const ethAddressForm = ref('')

// 收款方式管理
const showPaymentDialog = ref(false)
const paymentForm = ref({
  eth_address: '',
  alipay_account: '',
  default_payment_type: 'bsc'
})

// 格式化ETH地址，隐藏中间字符
const formatEthAddress = (address) => {
  if (!address || address.length < 10) return address
  // 显示前6位和后4位，中间用...代替
  return `${address.slice(0, 6)}...${address.slice(-4)}`
}

// 打开ETH地址编辑对话框
const openEthAddressDialog = () => {
  ethAddressForm.value = form.value.eth_address || ''
  showEthAddressDialog.value = true
}

// 保存ETH地址
const saveEthAddress = () => {
  const address = ethAddressForm.value?.trim()
  if (!address) {
    showFailToast('请输入BSC收款地址')
    return
  }
  // 简单验证：BSC地址通常以0x开头，长度为42
  if (!address.startsWith('0x') || address.length < 20) {
    showFailToast('请输入有效的BSC钱包地址')
    return
  }
  
  // 保存到后端
  httpPost('/api/user/bindEthAddress', {
    eth_address: address
  }).then(() => {
    form.value.eth_address = address
    showSuccessToast('收款地址保存成功')
    showEthAddressDialog.value = false
  }).catch((e) => {
    showFailToast('保存失败，' + e.message)
  })
}





const invitationActivatedOnce = ref(false)
onActivated(() => {
  // keep-alive 再次进入时静默刷新余额等，首次由 onMounted 负责
  if (!invitationActivatedOnce.value) {
    invitationActivatedOnce.value = true
    return
  }
  if (isLogin.value) {
    fetchUserProfile()
    fetchCsUnread(false)
    startCsUnreadPoll()
  }
})

onUnmounted(() => {
  stopCsUnreadPoll()
})

onMounted(() => {
  initData()
  fetchAnnouncements(); // 获取公告数据
  fetchTotalYeji();


  // 复制链接
  const clipboard = new Clipboard('.copy-link');
  clipboard.on('success', () => {
    ElMessage.success('复制链接成功！');
  })


  // const clipboard2 = new Clipboard('.copy-code');
  // clipboard2.on('success', () => {
  //   ElMessage.success('复制邀请码成功！');
  // })


  clipboard.on('error', () => {
    ElMessage.error('复制失败！');
  })


  httpGet("/api/invite/code").then(res => {
      const text = `${location.protocol}//${location.host}/register?invite_code=${res.data.code}`
      hits.value = res.data["hits"]
      regNum.value = res.data["reg_num"]
      jhNum.value = res.data["jh_num"]//激活数量

      userS.value.zzcode=res.data.code;
          //赋值
      if (hits.value > 0) {
        rate.value = ((regNum.value / hits.value) * 100).toFixed(2)
      }
      // 统一二维码：推广注册链接（含 invite_code，App 扫码可解析为转账码）
      inviteURL.value = text
      QRCode.toDataURL(text, {width: 400, height: 400, margin: 2}, (error, url) => {
        if (error) {
          console.error(error)
        } else {
          qrImg.value = url
          transferQrImg.value = url
        }
      })
    }).catch(e => {
      ElMessage.error("获取邀请码失败：" + e.message)
    })



})

const initData = () => {
  checkSession().then((user) => {
    isLogin.value = true
    loginUser.value = user // 保存用户信息，用于复投时获取 user_id
    startCsUnreadPoll()


    httpGet('/api/user/profile')
      .then(res => {
        form.value = {
          ...res.data,
          alipay_account: res.data.alipay_account || '',
          default_payment_type: res.data.default_payment_type || 'bsc'
        }
        csUnreadPrev = 0
        fetchCsUnread(false)
        // 根据当前语言设置头像（支持本地默认多语言头像）
        let avatarUrl = form.value.avatar || ''
        const current = locale.value
        const isDefaultAvatar =
          !avatarUrl ||
          avatarUrl.endsWith('/user.png') ||
          avatarUrl === '/images/avatar/user.png'

        if (isDefaultAvatar) {
          if (current === 'en-US') {
            avatarUrl = '/images/avatar/user-en.png'
          } else if (current === 'ja' || current === 'ja-JP') {
            avatarUrl = '/images/avatar/user-ja.png'
          } else if (current === 'kr-KR') {
            avatarUrl = '/images/avatar/user-kr.png'
          } else {
            avatarUrl = '/images/avatar/user.png'
          }
          form.value.avatar = avatarUrl
        }
        fileList.value = [{ url: avatarUrl }]
        // 从profile中获取左区和右区业绩
        if (res.data) {
          leftPerf.value = Number(res.data.left_perf || 0);
          rightPerf.value = Number(res.data.right_perf || 0);
          if (res.data.jihuo_ex_min_power > 0) REINVEST_POWER_MIN.value = Number(res.data.jihuo_ex_min_power)
          if (res.data.member_normal_max_power > 0) REINVEST_POWER_NORMAL_MAX.value = Number(res.data.member_normal_max_power)
          if (res.data.member_vip_min_power > 0) REINVEST_POWER_VIP_MIN.value = Number(res.data.member_vip_min_power)
          creditMax.value = Number(res.data.credit_max || 0)
          creditAvailable.value = Number(res.data.credit_available || 0)
          creditDebt.value = Number(res.data.credit_debt || 0)
          creditLegacy.value = !!res.data.credit_legacy
          if (Number(res.data.credit_legacy_base) > 0) creditLegacyBase.value = Number(res.data.credit_legacy_base)
          if (Number(res.data.credit_legacy_min) > 0) creditLegacyMin.value = Number(res.data.credit_legacy_min)
          // 获取推荐人信息（尝试多个可能的字段名）
          referrerName.value = res.data.inviter_name || res.data.referrer_name || res.data.parent_name || res.data.inviter_username || res.data.referrer_username || res.data.parent_username || res.data.inviter_code || res.data.referrer_code || res.data.parent_code || '';
        } else {
          // 如果profile中没有这些字段，尝试调用专门的API
          fetchUserPlacementPerformance();
        }
      })
      .catch((e) => {
        console.log(e.message)
        showFailToast('获取用户信息失败')
        // 如果profile获取失败，尝试调用专门的API
        fetchUserPlacementPerformance();
      })
      .finally(() => {
        isProfileLoaded.value = true
      });



    httpGet("/api/invite/code").then(res => {
      const text = `${location.protocol}//${location.host}/register?invite_code=${res.data.code}`
      hits.value = res.data["hits"]
      regNum.value = res.data["reg_num"]
      form.value.userS=res.data.code;
      // 如果邀请码API返回了推荐人信息，也尝试获取
      if (res.data && !referrerName.value) {
        referrerName.value = res.data.inviter_name || res.data.referrer_name || res.data.parent_name || res.data.inviter_username || res.data.referrer_username || res.data.parent_username || res.data.inviter_code || res.data.referrer_code || res.data.parent_code || '';
      }
          //赋值
      if (hits.value > 0) {
        rate.value = ((regNum.value / hits.value) * 100).toFixed(2)
      }
      // 统一二维码：推广注册链接（含 invite_code，App 扫码可解析为转账码）
      inviteURL.value = text
      userS.value.zzcode = res.data.code
      QRCode.toDataURL(text, {width: 400, height: 400, margin: 2}, (error, url) => {
        if (error) {
          console.error(error)
        } else {
          qrImg.value = url
          transferQrImg.value = url
        }
      })
    }).catch(e => {
     // ElMessage.error("获取邀请码失败：" + e.message)
    })

    httpGet("/api/config/get?key=system").then(res => {
      inviteChatCalls.value = res.data["invite_chat_calls"]
      inviteImgCalls.value = res.data["invite_img_calls"]
      const etf = res.data["enabled_transfer_fee"]
      transferFeeEnabled.value = etf !== false && etf !== 0 && etf !== '0'
      const tfp = res.data["transfer_fee_percent"]
      if (tfp !== undefined && tfp !== null && !Number.isNaN(Number(tfp))) {
        transferFeePercent.value = Number(tfp)
      } else {
        transferFeePercent.value = 5
      }
      const jihuoMin = Number(res.data?.jihuo_ex_min_power)
      if (jihuoMin > 0) REINVEST_POWER_MIN.value = jihuoMin
      const normalMax = Number(res.data?.member_normal_max_power)
      if (normalMax > 0) REINVEST_POWER_NORMAL_MAX.value = normalMax
      const vipMin = Number(res.data?.member_vip_min_power)
      if (vipMin > 0) REINVEST_POWER_VIP_MIN.value = vipMin
    }).catch(e => {
      //ElMessage.error("获取系统配置失败：" + e.message)
    })
  }).catch(() => {
    showLoginDialog.value = true
  });
}




const afterRead = (file) => {
  file.status = 'uploading';
  file.message = '上传中...';
  // 压图片并上传
  new Compressor(file.file, {
    quality: 0.6,
    success(result) {
      const formData = new FormData();
      formData.append('file', result, result.name);
      // 执行上传操作
      httpPost('/api/upload', formData).then((res) => {
        form.value.avatar = res.data.url
        file.status = 'success'
        httpPost('/api/user/profile/update', form.value).then(() => {
          showSuccessToast('上传成功')
        }).catch(() => {
          showFailToast('上传失败')
        })
      }).catch((e) => {
        showNotify({type: 'danger', message: '上传失败：' + e.message})
      })
    },
    error(err) {
      console.log(err.message);
    },
  });
}

const showPasswordDialog = ref(false)
const pass = ref({
  old: "",
  new: "",
  renew: ""
})


const transferDialog = ref(false)
const transfer = ref({
  uid: "",
  power: "",
  password: ""
})

const giftDialog = ref(false)
const gift = ref({
  code: "",
  power: ""
})
const creditGiftDialog = ref(false)
const creditGift = ref({
  code: "",
  power: ""
})
const creditMax = ref(0)
const creditAvailable = ref(0)
const creditDebt = ref(0)
const creditLegacy = ref(false)
const creditLegacyBase = ref(50000)
const creditLegacyMin = ref(10000)
const authPassword = ref("")
const scanFillTarget = ref('transfer') // transfer | gift | creditGift
const transferSubmitting = ref(false)
const giftSubmitting = ref(false)
const creditGiftSubmitting = ref(false)

const formatCredit = (n) => {
  const v = Number(n || 0)
  if (Number.isNaN(v)) return '0'
  return v.toLocaleString('zh-CN', { maximumFractionDigits: 2 })
}

const openTransferSheet = () => {
  authPassword.value = ''
  transferDialog.value = true
}

const openGiftSheet = () => {
  authPassword.value = ''
  giftDialog.value = true
}

const openCreditGiftSheet = () => {
  authPassword.value = ''
  creditGiftDialog.value = true
}

const creditDetailDialog = ref(false)
const creditDetailLoading = ref(false)
const creditLogs = ref([])
const creditLogsPage = ref(1)
const creditLogsTotal = ref(0)
const creditLogsPageSize = 20
const creditLogsHasMore = computed(() => creditLogs.value.length < creditLogsTotal.value)

const creditLogBalanceLabel = (item) => {
  const model = String(item?.model || '')
  if (model.includes('最大额度')) return t('invitation.creditDetailBalanceMax')
  if (model.includes('还款')) return t('invitation.creditDetailBalancePower')
  return t('invitation.creditDetailBalanceAvail')
}

const fetchCreditLogs = (append = false) => {
  if (creditDetailLoading.value) return
  const nextPage = append ? creditLogsPage.value + 1 : 1
  creditDetailLoading.value = true
  httpPost('/api/powerLog/list', {
    page: nextPage,
    page_size: creditLogsPageSize,
    type: 17, // PowerCredit
  }).then((res) => {
    const items = res.data?.items || []
    creditLogsTotal.value = Number(res.data?.total || 0)
    creditLogsPage.value = nextPage
    creditLogs.value = append ? creditLogs.value.concat(items) : items
  }).catch((e) => {
    showNotify({ type: 'danger', message: t('invitation.creditDetailFailPrefix') + e.message })
  }).finally(() => {
    creditDetailLoading.value = false
  })
}

const openCreditDetail = () => {
  creditDetailDialog.value = true
  fetchCreditLogs(false)
}

/** 与后台「基本设置」一致：是否开启手续费、比例（默认开启 5%） */
const transferFeeEnabled = ref(true)
const transferFeePercent = ref(5)
const transferPreview = computed(() => {
  const pct = transferFeeEnabled.value
    ? Math.min(100, Math.max(0, Number(transferFeePercent.value)))
    : 0
  const raw = parseFloat(String(transfer.value.power ?? '').trim())
  if (transfer.value.power === '' || transfer.value.power === null || Number.isNaN(raw) || raw <= 0) {
    return null
  }
  const fee = Math.round(raw * pct / 100 * 1e6) / 1e6
  const recv = Math.round(raw * 1e6) / 1e6
  const total = Math.round((raw + fee) * 1e6) / 1e6
  const balance = Number(form.value?.power || 0)
  const insufficient = !Number.isNaN(balance) && total > balance + 1e-9
  return { pct, fee, recv, total, insufficient, balance }
})

/** 赠送激活固定另扣 1% 服务费 */
const GIFT_FEE_PCT = 1
const giftPreview = computed(() => {
  const raw = parseFloat(String(gift.value.power ?? '').trim())
  if (gift.value.power === '' || gift.value.power === null || Number.isNaN(raw) || raw <= 0) {
    return null
  }
  const pct = GIFT_FEE_PCT
  const fee = Math.round(raw * pct / 100 * 1e6) / 1e6
  const recv = Math.round(raw * 1e6) / 1e6
  const total = Math.round((raw + fee) * 1e6) / 1e6
  const balance = Number(form.value?.power || 0)
  const insufficient = !Number.isNaN(balance) && total > balance + 1e-9
  return { pct, fee, recv, total, insufficient, balance }
})

/** 授信报单免手续费，校验可用授信 */
const creditGiftPreview = computed(() => {
  const raw = parseFloat(String(creditGift.value.power ?? '').trim())
  if (creditGift.value.power === '' || creditGift.value.power === null || Number.isNaN(raw) || raw <= 0) {
    return null
  }
  const recv = Math.round(raw * 1e6) / 1e6
  const balance = Number(creditAvailable.value || 0)
  const insufficient = !Number.isNaN(balance) && recv > balance + 1e-9
  return { recv, insufficient, balance }
})


const passwordDialog = ref(false) // 保留变量兼容旧引用，转账/赠送已改为同屏输入密码


const showMailDialog = ref(false)
const bind = ref({
  mail: ""
 
})

const showBuybackDialog = ref(false)
const buyback = ref({
  power: ""
})

const showPowerDetail = ref(false)
const powerDetailLoading = ref(false)
const powerLogs = ref([])

// 复投相关
const reinvestDialog = ref(false)
const reinvestPower = ref('')
const reinvestLoading = ref(false)
const loginUser = ref(null)

const availablePower = computed(() => Number(form.value?.power || 0))
const pendingPower = computed(() => {
  // 待产出 = 所有订单的now_power之和
  return Number(form.value?.pending_power || 0)
})
const isPendingExited = computed(() => pendingPower.value === 0)
const producedPower = computed(() => {
  // 已产出 = 所有订单的js_power之和
  return Number(form.value?.produced_power || 0)
})
const totalRevenue = computed(() => {
  // 总收益 = 所有订单的power之和（应产值）- 所有订单的sales之和（投入值）
  return Number(form.value?.total_revenue || 0)
})

const formatPowerTime = (ts) => dateFormat(ts, 'MM-dd HH:mm:ss')

const fetchPowerLogs = () => {
  powerDetailLoading.value = true
  httpPost('/api/powerLog/list', {
    page: 1,
    page_size: 5
  }).then((res) => {
    powerLogs.value = res.data?.items || []
  }).catch((e) => {
    showNotify({ type: 'danger', message: '获取算力明细失败：' + e.message })
  }).finally(() => {
    powerDetailLoading.value = false
  })
}

const openPowerDetail = () => {
  showPowerDetail.value = true
  if (!powerLogs.value.length) {
    fetchPowerLogs()
  }
}

const goInvestPage = () => {
  router.push('/mobile/invest')
}

// 打开复投弹窗
const openReinvestDialog = () => {
  reinvestPower.value = ''
  reinvestDialog.value = true
}

// 快捷输入算力
const setQuickPower = (ratio) => {
  if (availablePower.value <= 0) return
  const power = Math.floor(availablePower.value * ratio)
  reinvestPower.value = power.toString()
}

// 提交复投前的验证
const beforeReinvestClose = (action) => {
  if (action === 'confirm') {
    handleConfirmReinvest()
    return false // 阻止默认关闭，由 handleConfirmReinvest 控制
  }
  return true // 取消操作允许关闭
}

// 复投算力范围：起步由后台配置，普通会员上限9999，VIP需≥10000
const REINVEST_POWER_MIN = ref(1000)
const REINVEST_POWER_NORMAL_MAX = ref(9999)
const REINVEST_POWER_VIP_MIN = ref(10000)
const REINVEST_POWER_MAX = 100000
const REINVEST_REQUEST_TIMEOUT = 60000

const applyReinvestBalanceDelta = (delta) => {
  if (!form.value) return
  const next = Math.max(0, Number(form.value.power || 0) + delta)
  form.value.power = next
}

const verifyReinvestSucceeded = async (balanceBefore, powerNum) => {
  try {
    const res = await httpGet('/api/user/profile', {}, { timeout: 15000 })
    const newBalance = Number(res.data?.power ?? res.data?.Power ?? 0)
    return balanceBefore - newBalance >= powerNum - 0.01
  } catch {
    return false
  }
}

const finishReinvestSuccess = () => {
  showSuccessToast(t('invitation.reinvestSuccess') || 'OK')
  reinvestDialog.value = false
  reinvestPower.value = ''
  fetchUserProfile()
}

// 处理确认复投
const handleConfirmReinvest = () => {
  if (reinvestLoading.value) return // 防止重复提交
  
  const powerNum = Number(reinvestPower.value)
  if (!powerNum || powerNum <= 0) {
    showNotify({type: 'danger', message: t('invitation.reinvestInvalid')})
    return
  }

  if (powerNum > REINVEST_POWER_MAX) {
    showNotify({type: 'danger', message: t('invitation.reinvestMax')})
    return
  }
  if (powerNum < REINVEST_POWER_MIN.value) {
    showNotify({type: 'danger', message: t('invitation.reinvestMin', { min: REINVEST_POWER_MIN.value }) || `最低投入${REINVEST_POWER_MIN.value}算力`})
    return
  }
  if (availablePower.value < powerNum) {
    showNotify({type: 'danger', message: t('invitation.reinvestPowerNotEnough')})
    return
  }
  if (!loginUser.value?.id) {
    showNotify({type: 'danger', message: t('invitation.loginRequired')})
    return
  }
  
  // 验证通过，执行复投
  submitReinvest()
}

// 提交复投
const submitReinvest = async () => {
  const powerNum = Number(reinvestPower.value)
  const balanceBefore = availablePower.value

  reinvestLoading.value = true
  applyReinvestBalanceDelta(-powerNum)

  let safetyTimer = null
  let settled = false
  const settleLoading = () => {
    if (settled) return
    settled = true
    reinvestLoading.value = false
    if (safetyTimer) {
      clearTimeout(safetyTimer)
      safetyTimer = null
    }
  }

  try {
    safetyTimer = setTimeout(async () => {
      if (settled) return
      settleLoading()
      const recovered = await verifyReinvestSucceeded(balanceBefore, powerNum)
      if (recovered) {
        finishReinvestSuccess()
      } else {
        fetchUserProfile()
        showNotify({
          type: 'warning',
          message: t('invitation.reinvestPendingCheck') || '请求耗时较长，请稍后刷新页面查看复投结果'
        })
      }
    }, REINVEST_REQUEST_TIMEOUT + 3000)

    await httpPost('/api/payment/confirmJihuoEx', {
      power: powerNum,
      user_id: loginUser.value.id
    }, { timeout: REINVEST_REQUEST_TIMEOUT })

    settleLoading()
    finishReinvestSuccess()
  } catch (e) {
    if (settled) return
    settleLoading()
    const recovered = await verifyReinvestSucceeded(balanceBefore, powerNum)
    if (recovered) {
      finishReinvestSuccess()
    } else {
      applyReinvestBalanceDelta(powerNum)
      const msg = e?.message || e?.msg || (t('invitation.reinvestNetworkError') || '网络异常')
      showFailToast((t('invitation.reinvestFailPrefix') || '') + msg)
    }
  }
}

// 刷新用户信息
const fetchUserProfile = () => {
  httpGet('/api/user/profile').then(res => {
    form.value = res.data

    // 根据当前语言设置头像（支持本地默认多语言头像）
    let avatarUrl = form.value.avatar || ''
    const current = locale.value
    const isDefaultAvatar =
      !avatarUrl ||
      avatarUrl.endsWith('/user.png') ||
      avatarUrl === '/images/user.png'

    if (isDefaultAvatar) {
      if (current === 'en-US') {
        avatarUrl = '/images/user-en.png'
      } else if (current === 'ja' || current === 'ja-JP') {
        avatarUrl = '/images/user-ja.png'
      } else {
        avatarUrl = '/images/user.png'
      }
      form.value.avatar = avatarUrl
    }
    fileList.value = [{ url: avatarUrl }]
    // 从profile中获取左区和右区业绩
    if (res.data) {
      leftPerf.value = Number(res.data.left_perf || 0);
      rightPerf.value = Number(res.data.right_perf || 0);
    } else {
      // 如果profile中没有这些字段，尝试调用专门的API
      fetchUserPlacementPerformance();
    }
  }).catch((e) => {
    console.log(e.message)
    showFailToast('获取用户信息失败')
    // 如果profile获取失败，尝试调用专门的API
    fetchUserPlacementPerformance();
  });
}

// tab: pending=待产出(投入的算力)，exited=已产出(已出局的算力)
const goToOrderList = (tab = 'pending') => {
  router.push(`/mobile/invest?tab=${tab}`)
}

const goPowerLog = () => {
  router.push('/mobile/powerlog')
  showPowerDetail.value = false
}

watch(showPowerDetail, (visible) => {
  if (visible && !powerLogs.value.length) {
    fetchPowerLogs()
  }
})

const beforeClose = (action) => {
  new Promise((resolve) => {
    resolve(action === 'confirm');
  });
}


// 提交修改密码
const updatePass = () => {
  if (pass.value.old === '') {
    return showNotify({type: "danger", message: t('invitation.passOldRequired')})
  }
  if (!pass.value.new || pass.value.new.length < 8) {
    return showNotify({type: "danger", message: t('invitation.passLenInvalid')})
  }
  if (pass.value.renew !== pass.value.new) {
    return showNotify({type: "danger", message: t('invitation.passNotSame')})
  }
  httpPost('/api/user/password', {
    old_pass: pass.value.old,
    password: pass.value.new,
    repass: pass.value.renew
  }).then(() => {
    showSuccessToast(t('invitation.passUpdateSuccess'))
    showPasswordDialog.value = false
  }).catch((e) => {
    showFailToast(t('invitation.passUpdateFailPrefix') + e.message)
    showPasswordDialog.value = false
  })
}


// 转账：金额+密码同一底部弹层直接提交
const submitTransferPay = () => {
  if (transfer.value.uid === '') {
    return showNotify({type: "danger", message: t('invitation.transferUidRequired')})
  }
  if (transfer.value.power === '') {
    return showNotify({type: "danger", message: t('invitation.transferPowerRequired')})
  }
  if (authPassword.value === '') {
    return showNotify({type: "danger", message: t('invitation.passwordRequired')})
  }
  if (transferPreview.value?.insufficient) {
    return showNotify({
      type: 'danger',
      message: t('invitation.transferFeeInsufficient', {
        total: transferPreview.value.total,
        balance: transferPreview.value.balance
      })
    })
  }

  transferSubmitting.value = true
  httpPost('/api/user/transfer', {
    uid: String(transfer.value.uid || '').trim(),
    power: String(transfer.value.power ?? '').trim(),
    password: authPassword.value,
  }).then(() => {
    showSuccessToast(t('invitation.transferSuccess'))
    transferDialog.value = false
    authPassword.value = ''
    transfer.value = {
      uid: "",
      power: "",
      password: ""
    }
    fetchUserProfile()
  }).catch((e) => {
    showFailToast(t('invitation.transferFailPrefix') + e.message)
  }).finally(() => {
    transferSubmitting.value = false
  })
}

// 赠送：确认后同一底部弹层提交（含密码）
const submitGiftPay = () => {
  if (!String(gift.value.code || '').trim()) {
    return showNotify({type: "danger", message: t('invitation.giftCodeRequired')})
  }
  if (!String(gift.value.power || '').trim() || Number(gift.value.power) <= 0) {
    return showNotify({type: "danger", message: t('invitation.giftPowerRequired')})
  }
  if (authPassword.value === '') {
    return showNotify({type: "danger", message: t('invitation.passwordRequired')})
  }
  if (giftPreview.value?.insufficient) {
    return showNotify({
      type: "danger",
      message: t('invitation.giftFeeInsufficient', {
        total: giftPreview.value.total,
        balance: giftPreview.value.balance,
      }),
    })
  }

  const confirmMsg = giftPreview.value
    ? t('invitation.giftConfirmMessageWithFee', {
        power: giftPreview.value.recv,
        fee: giftPreview.value.fee,
        total: giftPreview.value.total,
      })
    : t('invitation.giftConfirmMessage', { power: gift.value.power })

  showConfirmDialog({
    title: t('invitation.giftConfirmTitle'),
    message: confirmMsg,
  }).then(() => {
    giftSubmitting.value = true
    httpPost('/api/payment/confirmJihuoGift', {
      code: String(gift.value.code).trim(),
      power: Number(gift.value.power),
      password: authPassword.value,
    }).then(() => {
      showSuccessToast(t('invitation.giftSuccess'))
      giftDialog.value = false
      authPassword.value = ''
      gift.value = { code: "", power: "" }
      fetchUserProfile()
    }).catch((e) => {
      showFailToast(t('invitation.giftFailPrefix') + e.message)
    }).finally(() => {
      giftSubmitting.value = false
    })
  }).catch(() => {})
}

// 授信报单：免手续费，扣可用授信
const submitCreditGiftPay = () => {
  if (!String(creditGift.value.code || '').trim()) {
    return showNotify({ type: 'danger', message: t('invitation.giftCodeRequired') })
  }
  if (!String(creditGift.value.power || '').trim() || Number(creditGift.value.power) <= 0) {
    return showNotify({ type: 'danger', message: t('invitation.giftPowerRequired') })
  }
  if (authPassword.value === '') {
    return showNotify({ type: 'danger', message: t('invitation.passwordRequired') })
  }
  if (creditGiftPreview.value?.insufficient) {
    return showNotify({
      type: 'danger',
      message: t('invitation.creditGiftInsufficient', {
        need: creditGiftPreview.value.recv,
        avail: creditGiftPreview.value.balance,
      }),
    })
  }

  showConfirmDialog({
    title: t('invitation.creditGiftConfirmTitle'),
    message: t('invitation.creditGiftConfirmMessage', {
      power: creditGiftPreview.value?.recv ?? creditGift.value.power,
    }),
  }).then(() => {
    creditGiftSubmitting.value = true
    httpPost('/api/payment/confirmJihuoCreditGift', {
      code: String(creditGift.value.code).trim(),
      power: Number(creditGift.value.power),
      password: authPassword.value,
    }).then((res) => {
      showSuccessToast(t('invitation.creditGiftSuccess'))
      creditGiftDialog.value = false
      authPassword.value = ''
      creditGift.value = { code: '', power: '' }
      if (res?.data) {
        if (res.data.credit_max != null) creditMax.value = Number(res.data.credit_max)
        if (res.data.credit_available != null) creditAvailable.value = Number(res.data.credit_available)
        if (res.data.credit_debt != null) creditDebt.value = Number(res.data.credit_debt)
      }
      fetchUserProfile()
    }).catch((e) => {
      showFailToast(t('invitation.creditGiftFailPrefix') + e.message)
    }).finally(() => {
      creditGiftSubmitting.value = false
    })
  }).catch(() => {})
}

// 兼容旧函数名（如有外部引用）
const updateTransferBefore = submitTransferPay
const updateGiftBefore = submitGiftPay
const updateTransfer = submitTransferPay
const submitGift = submitGiftPay
const confirmPasswordAction = () => {}


// 提交绑定邮箱操作
const updateMail = () => {

if (bind.value.mail === '') {
  return showNotify({type: "danger", message: t('invitation.bindEmailRequired')})
}

    //绑定邮箱操作
    httpPost('/api/user/updateMail', {
      mail: bind.value.mail
    }).then(() => {
      showSuccessToast(t('invitation.bindSuccess'))
      if (form.value) form.value.mail = bind.value.mail
      showMailDialog.value = false
      fetchUserProfile()
    }).catch((e) => {
      showFailToast(t('invitation.bindFailPrefix') + e.message)
      showMailDialog.value = false
    })

    //  showSuccessToast("绑定邮箱模拟成功");
      // showMailDialog.value = false
    }



  // 暂未开放提示
  const handlePowerTrade = () => {

    // 商户用户可以正常跳转
    router.push('/mobile/powertrade');
    //showToast('内测中,暂未开放');
  };

  // C2C 撮合入口，跳转到撮合池页面
  const handleC2C = () => {
    router.push('/mobile/c2c')
  }

  //处理算池点击
  const handlePowerPool = () => {
    router.push('/mobile/powerpool');
  }

// 处理回购点击
const handleBuyback = () => {


  // router.push('/mobile/buyback');

 
  //  showFailToast(t('invitation.buybackNotOpen'));
  //  return;
  

  //检查是否是商户
  if (!form.value.vip) {
   showFailToast('只有商户才可以回购操作');
   return;
  }


  //检查是否设置了BSC地址
  if (!form.value.eth_address || form.value.eth_address.trim() === '') {
    showFailToast('请先设置BSC付款地址');
    return;
  }
  // 检查算力余额
  if (!form.value.power || parseFloat(form.value.power) <= 0) {
    showFailToast('算力余额不足');
    return;
  }
  // 打开回购对话框
  showBuybackDialog.value = true;
  // 重置回购表单
  buyback.value.power = '';


};

// 当前算力余额（用于弹窗显示）
const currentPowerBalance = computed(() => {
  const power = form.value.power;
  if (power === null || power === undefined || power === '') {
    return '0';
  }
  return String(power);
});

// 当前BSC地址（用于弹窗显示）
const currentEthAddress = computed(() => {
  const address = form.value.eth_address;
  if (!address || address.trim() === '') {
    return t('invitation.bscNotSetLabel');
  }
  return address;
});

// 计算回购金额（1.5折：10000算力=15美金，即1算力=0.0015美金）
const buybackAmount = computed(() => {
  const power = parseFloat(buyback.value.power);
  if (!power || isNaN(power) || power <= 0) {
    return '0.00';
  }
  // 1算力 = 0.0015美金
  const amount = power * 0.0015;
  return amount.toFixed(2);
});

// 提交回购
const submitBuyback = () => {
  // 验证算力数量
  if (!buyback.value.power || buyback.value.power.trim() === '') {
    return showNotify({type: "danger", message: t('invitation.errorPowerRequired')});
  }
  
  const power = parseFloat(buyback.value.power);
  if (isNaN(power) || power <= 0) {
    return showNotify({type: "danger", message: t('invitation.errorPowerInvalid')});
  }
  
  // 检查算力余额
  const userPower = parseFloat(form.value.power) || 0;
  if (power > userPower) {
    return showNotify({type: "danger", message: t('invitation.errorPowerNotEnough')});
  }
  
  // 检查BSC地址
  if (!form.value.eth_address || form.value.eth_address.trim() === '') {
    return showNotify({type: "danger", message: t('invitation.errorBscNotSet')});
  }
  
  // 调用回购API
  httpPost('/api/user/buyback', {
    power: power,
    eth_address: form.value.eth_address
  }).then(() => {
    showSuccessToast(t('invitation.buybackSuccess'));
    showBuybackDialog.value = false;
    // 清空回购表单
    buyback.value.power = '';
    fetchUserProfile();
  }).catch((e) => {
    showFailToast(t('invitation.buybackFailPrefix') + (e.message || ''));
  });
};

// 签到日历相关状态
const showSignInCalendar = ref(false)
const signInLoading = ref(false)
const signInCalendar = ref({ year: new Date().getFullYear(), month: new Date().getMonth() + 1, days: [], signed_today: false, streak: 0 })

const signInWeekdays = computed(() => {
  const w = t('invitation.signInWeekdays')
  return Array.isArray(w) ? w : ['日', '一', '二', '三', '四', '五', '六']
})

// 生成当月日历格子：空白占位 + 日期 + 是否已签到 + 是否今天
const signInDays = computed(() => {
  const { year, month, days, signed_today } = signInCalendar.value
  const cells = []
  if (!year || !month) return cells
  const offset = new Date(year, month - 1, 1).getDay() // 0=周日
  const total = new Date(year, month, 0).getDate()
  const signedSet = new Set((Array.isArray(days) ? days : []).map(d => Number(d)).filter(d => d > 0))
  const now = new Date()
  // 今日已签但接口 days 偶发缺当天时，前端兜底打勾
  if (signed_today && year === now.getFullYear() && month === now.getMonth() + 1) {
    signedSet.add(now.getDate())
  }
  const todayStr = `${now.getFullYear()}-${now.getMonth() + 1}-${now.getDate()}`
  for (let i = 0; i < offset; i++) cells.push({ day: 0, signed: false, isToday: false })
  for (let d = 1; d <= total; d++) {
    cells.push({
      day: d,
      signed: signedSet.has(d),
      isToday: `${year}-${month}-${d}` === todayStr,
    })
  }
  return cells
})

const fetchSignInCalendar = () => {
  const now = new Date()
  httpPost('/api/user/signInCalendar', {
    year: now.getFullYear(),
    month: now.getMonth() + 1,
  }).then(res => {
    const data = res.data || {}
    const dayList = Array.isArray(data.days) ? data.days.map(d => Number(d)).filter(d => d > 0) : []
    signInCalendar.value = {
      year: Number(data.year) || now.getFullYear(),
      month: Number(data.month) || now.getMonth() + 1,
      days: dayList,
      signed_today: !!data.signed_today,
      streak: Number(data.streak) || 0,
    }
    form.value.signed_today = !!data.signed_today
  }).catch(() => {})
}

// 打开签到日历并拉取当月数据
const openSignInCalendar = () => {
  showSignInCalendar.value = true
  fetchSignInCalendar()
}

// 领取每日免费福利（连续签到）
const getFuli = () => {
  if (signInLoading.value) return
  showConfirmDialog({
    title: t('invitation.fuliDialogTitle'),
    message: t('invitation.fuliDialogMessage'),
  }).then(() => {
    signInLoading.value = true
    httpPost("/api/user/getFuli", {}).then(() => {
      showSuccessToast(t('invitation.fuliSuccess'))
      fetchUserProfile()
      fetchSignInCalendar()
    }).catch(e => {
      showFailToast(t('invitation.fuliFailPrefix') + (e.message || ''))
    }).finally(() => {
      signInLoading.value = false
    })
  }).catch(() => {

  });
}





const logout = function () {
  httpGet('/api/user/logout').then(() => {
    removeUserToken();
    router.push('/login');
  }).catch(() => {
    ElMessage.error('注销失败！');
  })
}

const alipayNavActions = computed(() => [
  {
    key: 'scan',
    icon: 'scan',
    label: t('invitation.scanPay'),
    tone: 'blue',
    onClick: () => {
      scanFillTarget.value = 'transfer'
      startScan()
    },
  },
  {
    key: 'transfer',
    icon: 'exchange',
    label: t('invitation.transfer'),
    tone: 'orange',
    onClick: () => openTransferSheet(),
  },
  {
    key: 'receive',
    icon: 'qr',
    label: t('invitation.receiveQr'),
    tone: 'green',
    onClick: () => openReceiveQrDialog(),
  },
  {
    key: 'share',
    icon: 'share-o',
    label: t('invitation.share'),
    tone: 'sky',
    onClick: () => openShareDialog(),
  },
])

const quickActions = computed(() => {
  const isCsAgent = form.value?.is_cs === true || form.value?.is_cs === 1 || form.value?.is_cs === '1'
  const list = [
    { key: 'powerlog', icon: 'balance-list-o', label: t('invitation.detail'), tone: 'mint', onClick: () => router.push('/mobile/powerlog') },
    // { key: 'coinlog', icon: 'gold-coin-o', label: t('invitation.coinDetail'), tone: 'gold', onClick: () => router.push('/mobile/coinlog') },
    // { key: 'transfer', icon: 'exchange', label: t('invitation.transfer'), tone: 'orange', onClick: () => openTransferSheet() },
    { key: 'gift', icon: 'gift-o', label: t('invitation.gift'), tone: 'rose', onClick: () => openGiftSheet() },
    // 授信报单入口已移至「授信额度」卡片标题栏（待还额度上方）
    { key: 'pool', icon: 'shopping-cart-o', label: t('invitation.match'), tone: 'cyan', onClick: () => handlePowerPool() },
    { key: 'c2c', icon: 'friends-o', label: t('invitation.c2c'), tone: 'blue', onClick: () => handleC2C() },
    { key: 'buyback', icon: 'cash-back-record', label: t('invitation.buyback'), tone: 'coral', onClick: () => handleBuyback() },
    {
      key: 'contact',
      icon: 'chat-o',
      label: isCsAgent ? '客服接入' : t('invitation.service'),
      tone: 'sky',
      onClick: () => router.push(isCsAgent ? '/mobile/cs-desk' : '/mobile/contact'),
    },
    { key: 'settings', icon: 'edit', label: t('invitation.settings'), tone: 'slate', onClick: () => { showPasswordDialog.value = true } },
  ]
  if (form.value?.allow_placement) {
    list.push({
      key: 'placement',
      icon: 'cluster-o',
      label: t('invitation.placement'),
      tone: 'indigo',
      onClick: () => router.push('/mobile/placement'),
    })
  }
  if (form.value?.is_auditor) {
    list.push({
      key: 'audit',
      icon: 'eye-o',
      label: t('invitation.audit'),
      tone: 'violet',
      onClick: () => router.push('/mobile/audit'),
    })
  }
  list.push({
    key: 'logout',
    icon: 'close',
    label: t('invitation.logout'),
    tone: 'gray',
    onClick: () => logout(),
  })
  return list
})

const runQuickAction = (item) => {
  if (!item || typeof item.onClick !== 'function') return
  item.onClick()
}



const showAnnouncement = ref(false);
const announcements =  ref("")

// 处理 "我知道了" 按钮点击事件
const handleConfirm = () => {
  showAnnouncement.value = false;
};

// 处理 "不再显示" 按钮点击事件
const handleDoNotShow = () => {
  localStorage.setItem('doNotShowAnnouncement', 'true'); // 设置不再显示的标记
  showAnnouncement.value = false;
};


// 模拟用户重新登录时清除 localStorage 中的标记
// 你可以根据项目实际逻辑来调整这个部分，比如在用户登录时清除标记
const clearDoNotShowFlagOnLogin = () => {
  localStorage.removeItem('doNotShowAnnouncement');
};


const noticeHasText = (content) => {
  if (content === undefined || content === null) return false
  const text = String(content)
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/gi, ' ')
    .replace(/\s+/g, ' ')
    .trim()
  return text.length > 0
}

const fetchAnnouncements = function () {
  httpGet("/api/config/get?key=notice").then(res => {
    try {
      const raw = res.data && res.data.content
      if (!noticeHasText(raw)) {
        announcements.value = ''
        showAnnouncement.value = false
        return
      }
      announcements.value = md.render(raw)
      const doNotShow = localStorage.getItem('doNotShowAnnouncement')
      showAnnouncement.value = !doNotShow
    } catch (e) {
      showAnnouncement.value = false
      console.error('Error in processing announcement:', e)
    }
  }).catch(e => {
    showAnnouncement.value = false
    console.error('HTTP request error (notice):', e)
  })





};




// const copyTs = () => {

//   const clipboard2 = new Clipboard('.copy-code');
//   clipboard2.on('success', () => {
//     ElMessage.success('复制邀请码成功！');
//   })
// }

// 复制推广链接
const copyInviteURL = async () => {
  if (!inviteURL.value) {
    showFailToast(t('invitation.copyInviteFail'))
    return
  }
  try {
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(inviteURL.value)
      showSuccessToast(t('invitation.copyInviteSuccess'))
      return
    }
    const textArea = document.createElement('textarea')
    textArea.value = inviteURL.value
    textArea.style.position = 'fixed'
    textArea.style.left = '-999999px'
    document.body.appendChild(textArea)
    textArea.focus()
    textArea.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(textArea)
    if (ok) {
      showSuccessToast(t('invitation.copyInviteSuccess'))
    } else {
      showFailToast(t('invitation.copyInviteFail'))
    }
  } catch (e) {
    showFailToast(t('invitation.copyInviteFail'))
  }
}

// 复制转账码
const copyTransferCode = async () => {
  if (!userS.value?.zzcode) {
    showFailToast('转账码不存在');
    return;
  }

  try {
    // 使用现代 Clipboard API
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(userS.value.zzcode);
      showSuccessToast('转账码已复制');
    } else {
      // 降级方案：使用传统方法
      const textArea = document.createElement('textarea');
      textArea.value = userS.value.zzcode;
      textArea.style.position = 'fixed';
      textArea.style.left = '-999999px';
      textArea.style.top = '-999999px';
      document.body.appendChild(textArea);
      textArea.focus();
      textArea.select();
      try {
        const successful = document.execCommand('copy');
        if (successful) {
          showSuccessToast('转账码已复制');
        } else {
          showFailToast('复制失败，请手动复制');
        }
      } catch (err) {
        showFailToast('复制失败，请手动复制');
      }
      document.body.removeChild(textArea);
    }
  } catch (err) {
    showFailToast('复制失败，请手动复制');
  }
};

const updateEthAddress = () => {
  httpPost('/api/user/bindEthAddress', {
    eth_address: form.value.eth_address
  }).then(() => {
    showSuccessToast("BSC地址更新成功")
  }).catch((e) => {
    showFailToast('更新失败，' + e.message)
  })
}

// 打开收款方式管理弹窗
const openPaymentDialog = () => {
  // 同步当前表单数据到弹窗表单
  paymentForm.value = {
    eth_address: form.value.eth_address || '',
    alipay_account: form.value.alipay_account || '',
    default_payment_type: form.value.default_payment_type || 'bsc'
  }
  showPaymentDialog.value = true
}

// 处理收款方式切换
const handlePaymentTypeChange = (type, val) => {
  // 如果用户勾选了该类型，设置为默认
  if (val) {
    paymentForm.value.default_payment_type = type
  } else {
    // 如果用户取消勾选，且当前是该类型，则不允许取消（必须有一个默认）
    if (paymentForm.value.default_payment_type === type) {
      // 切换到另一个收款方式
      paymentForm.value.default_payment_type = type === 'bsc' ? 'alipay' : 'bsc'
    }
  }
}

// 保存收款方式
const savePaymentMethods = () => {
  // 验证必填项
  if (paymentForm.value.default_payment_type === 'bsc' && !paymentForm.value.eth_address) {
    showFailToast('请填写BSC收款地址')
    return
  }
  if (paymentForm.value.default_payment_type === 'alipay' && !paymentForm.value.alipay_account) {
    showFailToast('请填写支付宝账号')
    return
  }

  // 保存到后端
  httpPost('/api/user/updatePaymentMethods', {
    eth_address: paymentForm.value.eth_address,
    alipay_account: paymentForm.value.alipay_account,
    default_payment_type: paymentForm.value.default_payment_type
  }).then(() => {
    // 更新本地表单数据
    form.value.eth_address = paymentForm.value.eth_address
    form.value.alipay_account = paymentForm.value.alipay_account
    form.value.default_payment_type = paymentForm.value.default_payment_type
    showSuccessToast('收款方式保存成功')
    showPaymentDialog.value = false
  }).catch((e) => {
    showFailToast('保存失败，' + e.message)
  })
}

// 添加扫码相关的变量和方法
const showScanDialog = ref(false)
const showReceiveQrDialog = ref(false)
const showNodeDialog = ref(false)
const showManualInput = ref(false)
const manualInputValue = ref('')
const nodeAddress = ref('node.example.com:8080')
const videoRef = ref(null)
const scanStream = ref(null)
const scanResult = ref('')
const hasMultipleCameras = ref(false)
const currentFacingMode = ref('environment')

// 打开我的收取码（二维码内容=推广链接，转账/注册共用）
const ensureUnifiedQr = (link) => {
  const text = (link || inviteURL.value || '').trim()
  if (!text) return
  QRCode.toDataURL(text, {width: 400, height: 400, margin: 2}, (error, url) => {
    if (!error && url) {
      qrImg.value = url
      transferQrImg.value = url
    }
  })
}

const prepareShareOrReceiveQr = () => {
  if (!userS.value?.zzcode) return false
  if (!inviteURL.value) {
    inviteURL.value = `${location.protocol}//${location.host}/register?invite_code=${userS.value.zzcode}`
  }
  if (!transferQrImg.value) {
    ensureUnifiedQr(inviteURL.value)
  }
  return true
}

const openReceiveQrDialog = () => {
  if (!prepareShareOrReceiveQr()) {
    showFailToast(t('invitation.receiveQrEmpty'))
    return
  }
  showReceiveQrDialog.value = true
}

const showShareDialog = ref(false)
const openShareDialog = () => {
  // 未激活：仅拦截分享（推广码/外链），转账码与收取码保持原样
  if (!isUserActivated.value) {
    showFailToast(t('invitation.needActivateForShare'))
    return
  }
  if (!prepareShareOrReceiveQr()) {
    showFailToast(t('invitation.shareQrEmpty'))
    return
  }
  showShareDialog.value = true
}

// 从转账弹窗发起扫码
const startScanFromTransfer = async () => {
  scanFillTarget.value = 'transfer'
  transferDialog.value = false
  await startScan()
}

// 从赠送弹窗发起扫码
const startScanFromGift = async () => {
  scanFillTarget.value = 'gift'
  giftDialog.value = false
  await startScan()
}

// 从授信报单弹窗发起扫码
const startScanFromCreditGift = async () => {
  scanFillTarget.value = 'creditGift'
  creditGiftDialog.value = false
  await startScan()
}

// 从扫码结果中解析转账码/收取码
const extractTransferCode = (raw) => {
  if (!raw || typeof raw !== 'string') return ''
  const text = raw.trim()
  if (!text) return ''

  try {
    const qrData = JSON.parse(text)
    if (qrData && (qrData.uid || qrData.code)) {
      return String(qrData.uid || qrData.code).trim()
    }
  } catch (e) {
    // 非 JSON，继续按链接或纯码解析
  }

  try {
    const url = new URL(text, location.origin)
    const inviteCode = url.searchParams.get('invite_code') || url.searchParams.get('code')
    if (inviteCode) {
      return inviteCode.trim()
    }
  } catch (e) {
    // 非 URL
  }

  const inviteMatch = text.match(/[?&]invite_code=([^&#\s]+)/i)
  if (inviteMatch && inviteMatch[1]) {
    return decodeURIComponent(inviteMatch[1]).trim()
  }

  // 纯转账码 / 推荐码
  if (!/^https?:\/\//i.test(text)) {
    return text
  }
  return ''
}

// 停止摄像头
const stopScanCamera = () => {
  if (scanStream.value) {
    scanStream.value.getTracks().forEach(track => track.stop())
    scanStream.value = null
  }
  if (videoRef.value) {
    videoRef.value.srcObject = null
  }
}

// 开始扫码
const startScan = async () => {
  try {
    // 快捷入口「扫一扫」默认用于转账填充
    if (!scanFillTarget.value) {
      scanFillTarget.value = 'transfer'
    }
    // 检查是否为HTTPS环境（本地开发环境除外）
    if (location.protocol !== 'https:' && location.hostname !== 'localhost' && location.hostname !== '127.0.0.1') {
      showFailToast('摄像头功能需要在HTTPS环境下使用')
      return
    }

    // 检查浏览器是否支持媒体设备API
    if (!navigator.mediaDevices) {
      showFailToast('您的浏览器不支持摄像头功能，请使用现代浏览器')
      return
    }

    // 检查是否支持getUserMedia
    if (!navigator.mediaDevices.getUserMedia) {
      showFailToast('您的浏览器版本过低，请升级浏览器后重试')
      return
    }

    // 检查可用摄像头数量
    try {
      const devices = await navigator.mediaDevices.enumerateDevices()
      const videoDevices = devices.filter(device => device.kind === 'videoinput')
      
      if (videoDevices.length === 0) {
        showFailToast('未检测到摄像头设备')
        return
      }
      
      hasMultipleCameras.value = videoDevices.length > 1
    } catch (error) {
      console.warn('无法枚举设备:', error)
      // 继续尝试启动摄像头，可能是权限问题
    }

    showScanDialog.value = true
    
    // 等待对话框显示后再启动摄像头
    await nextTick()
    
    await startCamera()
    
  } catch (error) {
    console.error('启动扫码失败:', error)
    showFailToast('启动扫码功能失败，请检查设备权限')
    showScanDialog.value = false
  }
}

// 启动摄像头
const startCamera = async () => {
  try {
    // 请求摄像头权限
    const constraints = {
      video: { 
        facingMode: currentFacingMode.value,
        width: { ideal: 1280, max: 1920 },
        height: { ideal: 720, max: 1080 }
      }
    }
    
    const stream = await navigator.mediaDevices.getUserMedia(constraints)
    
    scanStream.value = stream
    if (videoRef.value) {
      videoRef.value.srcObject = stream
    }
    
    // 开始扫码检测
    startQRCodeDetection()
    
  } catch (error) {
    console.error('启动摄像头失败:', error)
    
    // 根据错误类型提供具体的错误信息
    let errorMessage = '无法启动摄像头'
    
    if (error.name === 'NotAllowedError') {
      errorMessage = '摄像头权限被拒绝，请在浏览器设置中允许摄像头访问'
    } else if (error.name === 'NotFoundError') {
      errorMessage = '未找到摄像头设备'
    } else if (error.name === 'NotReadableError') {
      errorMessage = '摄像头被其他应用占用，请关闭其他应用后重试'
    } else if (error.name === 'OverconstrainedError') {
      errorMessage = '摄像头不支持所需的分辨率，正在尝试降低要求'
      
      // 尝试使用更低的约束条件
      try {
        const fallbackConstraints = {
          video: { 
            facingMode: currentFacingMode.value,
            width: { ideal: 640 },
            height: { ideal: 480 }
          }
        }
        const fallbackStream = await navigator.mediaDevices.getUserMedia(fallbackConstraints)
        scanStream.value = fallbackStream
        if (videoRef.value) {
          videoRef.value.srcObject = fallbackStream
        }
        startQRCodeDetection()
        return
      } catch (fallbackError) {
        errorMessage = '摄像头启动失败，请检查设备是否正常'
      }
    } else if (error.name === 'NotSupportedError') {
      errorMessage = '您的设备不支持摄像头功能'
    } else if (error.name === 'SecurityError') {
      errorMessage = '安全限制：请确保在HTTPS环境下使用'
    }
    
    showFailToast(errorMessage)
    throw error
  }
}

// 切换摄像头
const switchCamera = async () => {
  if (scanStream.value) {
    scanStream.value.getTracks().forEach(track => track.stop())
  }
  
  currentFacingMode.value = currentFacingMode.value === 'environment' ? 'user' : 'environment'
  
  try {
    await startCamera()
  } catch (error) {
    // 如果切换失败，尝试恢复原来的摄像头
    currentFacingMode.value = currentFacingMode.value === 'environment' ? 'user' : 'environment'
    try {
      await startCamera()
    } catch (fallbackError) {
      showFailToast('切换摄像头失败')
    }
  }
}

// 处理手动输入
const handleManualInput = () => {
  if (manualInputValue.value.trim()) {
    handleScanSuccess(manualInputValue.value.trim())
    showManualInput.value = false
    manualInputValue.value = ''
  } else {
    showFailToast('请输入有效内容')
  }
}

// 二维码检测
const startQRCodeDetection = () => {
  if (!videoRef.value) return
  
  const canvas = document.createElement('canvas')
  const context = canvas.getContext('2d')
  
  const detectQRCode = () => {
    if (!showScanDialog.value || !videoRef.value) return
    
    const video = videoRef.value
    if (video.readyState === video.HAVE_ENOUGH_DATA) {
      canvas.width = video.videoWidth
      canvas.height = video.videoHeight
      context.drawImage(video, 0, 0, canvas.width, canvas.height)
      
      try {
        // 使用 jsQR 进行二维码识别
        const imageData = context.getImageData(0, 0, canvas.width, canvas.height)
        const code = jsQR(imageData.data, imageData.width, imageData.height)
        
        if (code) {
          scanResult.value = code.data
          handleScanSuccess(code.data)
          return
        }
        
      } catch (error) {
        console.error('二维码识别失败:', error)
      }
    }
    
    // 继续检测
    requestAnimationFrame(detectQRCode)
  }
  
  detectQRCode()
}

// 处理扫码成功
const handleScanSuccess = (result) => {
  scanResult.value = result
  closeScan()

  const code = extractTransferCode(result)
  if (code) {
    authPassword.value = ''
    if (scanFillTarget.value === 'gift') {
      gift.value.code = code
      giftDialog.value = true
    } else if (scanFillTarget.value === 'creditGift') {
      creditGift.value.code = code
      creditGiftDialog.value = true
    } else {
      transfer.value.uid = code
      transferDialog.value = true
    }
    scanFillTarget.value = 'transfer'
    showSuccessToast(t('invitation.scanFillSuccess'))
    return
  }
  showFailToast(t('invitation.scanInvalidQr'))
}

// 处理扫码结果
const handleScanResult = () => {
  // 点击确认按钮时显示手动输入
  showManualInput.value = true
}

// 关闭扫码
const closeScan = () => {
  showScanDialog.value = false
  stopScanCamera()
  scanResult.value = ''
}

// 显示手动输入转账码对话框
const showManualTransfer = () => {
  showManualInput.value = true
  manualInputValue.value = ''
}

const fetchTotalYeji = async () => {
  try {
    // 先获取总条数
    const res = await httpPost('/api/invite/list', { page: 1, page_size: 1 });
    const total = res.data?.total || 0;
    if (total === 0) {
      totalYeji.value = 0;
      validYeji.value = 0;
      return;
    }
    // 再获取所有数据
    const resAll = await httpPost('/api/invite/list', { page: 1, page_size: total });
    const items = resAll.data?.items || [];
    const yejiArr = items.map(item => Number(item.yeji) || 0);
    const sum = yejiArr.reduce((sum, v) => sum + v, 0);
    const max = yejiArr.length > 0 ? Math.max(...yejiArr) : 0;
    totalYeji.value = sum;
    validYeji.value = sum - max;
  } catch (e) {
    totalYeji.value = 0;
    validYeji.value = 0;
  }
};

// 获取当前用户的左区和右区业绩
const fetchUserPlacementPerformance = async () => {
  try {
    // 尝试从用户profile获取
    const res = await httpGet('/api/user/profile');
    if (res.data) {
      leftPerf.value = Number(res.data.left_perf || 0);
      rightPerf.value = Number(res.data.right_perf || 0);
    }
  } catch (e) {
    // 如果profile中没有，尝试调用其他API
    try {
      // 尝试调用用户节点信息API（如果存在）
      const nodeRes = await httpGet('/api/user/placement');
      if (nodeRes.data) {
        leftPerf.value = Number(nodeRes.data.left_perf || 0);
        rightPerf.value = Number(nodeRes.data.right_perf || 0);
      }
    } catch (err) {
      console.log('获取用户安置业绩失败:', err.message);
      leftPerf.value = 0;
      rightPerf.value = 0;
    }
  }
};

const isEditingBrand = ref(false);

function startEditBrand() {
  isEditingBrand.value = true;
  nextTick(() => {
    const input = document.querySelector('.brand-name-input');
    if (input) {
      input.focus();
    }
  });
}
function saveBrandEdit() {
  isEditingBrand.value = false;
  if (!form.value.div) {
    form.value.div = '';
  }

  // 绑定品牌，保存到后端
  httpPost('/api/user/saveDiv', {
    div: form.value.div
  }).then(() => {
    showSuccessToast(t('invitation.brandUpdateSuccess'));
  }).catch((e) => {
    showFailToast(t('invitation.brandUpdateFailPrefix') + e.message);
  });
}

</script>

<style lang="stylus" scoped>
@import "@/assets/css/custom-scroll.styl"

.invitation-page
  position relative
  min-height 100vh
  background linear-gradient(135deg, #4facfe 0%, #00f2fe 50%, #43e97b 100%)
  color #2c3e50
  overflow hidden
  &:before
    content ''
    position fixed
    inset 0
    background radial-gradient(800px 800px at 15% 25%, rgba(255, 255, 255, 0.15), transparent 55%), radial-gradient(700px 700px at 85% 10%, rgba(255, 255, 255, 0.12), transparent 55%), radial-gradient(600px 600px at 70% 80%, rgba(255, 255, 255, 0.1), transparent 65%)
    pointer-events none
    z-index 0
  &:after
    content ''
    position fixed
    inset 0
    background linear-gradient(135deg, rgba(255,255,255,0.1) 0%, transparent 30%, rgba(255,255,255,0.08) 60%, transparent 100%)
    pointer-events none
    z-index 0

.gradient-bg
  position fixed
  inset 0
  background linear-gradient(135deg, #667eea 0%, #764ba2 25%, #f093fb 50%, #4facfe 75%, #00f2fe 100%)
  filter blur(0)
  z-index 0

.layout
  position relative
  z-index 1
  max-width 1120px
  width 100%
  margin 0 auto
  padding 18px 16px 80px
  display flex
  flex-direction column
  gap 20px

.glass-card
  position relative
  background rgba(255,255,255,0.98)
  border 1px solid rgba(79,172,254,0.15)
  box-shadow 0 8px 32px rgba(0,0,0,0.1), 0 2px 8px rgba(0,0,0,0.06)
  border-radius 24px
  padding 24px
  backdrop-filter blur(12px)
  overflow hidden
  transition box-shadow 0.3s cubic-bezier(0.4, 0, 0.2, 1), border-color 0.3s ease
  &:before
    content ''
    position absolute
    inset 0
    border-radius inherit
    background linear-gradient(135deg, rgba(255,255,255,0.4), rgba(255,255,255,0))
    opacity 0
    transition opacity .3s ease
    pointer-events none
  &:after
    content ''
    position absolute
    inset -1px
    border-radius inherit
    background linear-gradient(120deg, rgba(79,172,254,0.15), rgba(67,233,123,0.12), rgba(255,159,67,0.1))
    filter blur(20px)
    opacity 0.3
    z-index -1
    pointer-events none

@media (hover: hover) and (pointer: fine)
  .glass-card:hover
    transform translateY(-3px)
    box-shadow 0 16px 48px rgba(79,172,254,0.12), 0 6px 16px rgba(0,0,0,0.1)
    border-color rgba(79,172,254,0.25)
    &:before
      opacity 1

.hero-card
  padding 24px
  border 1px solid rgba(255,255,255,0.9)
  background rgba(255,255,255,0.98)
  box-shadow 0 12px 40px rgba(0,0,0,0.15)
  overflow visible

.brand-row
  position relative
  margin-bottom 12px

.lang-dropdown
  position relative
  z-index 30
  display flex
  justify-content center
  margin-top 12px

.lang-chip
  display inline-flex
  align-items center
  gap 4px
  margin 0
  padding 0 8px 0 6px
  height 26px
  border none
  border-radius 999px
  background rgba(255, 255, 255, 0.92)
  border 1px solid rgba(79, 172, 254, 0.18)
  color #4a5d73
  cursor pointer
  font-family inherit
  white-space nowrap
  box-shadow 0 2px 8px rgba(0, 0, 0, 0.06)
  -webkit-tap-highlight-color transparent
  transition background 0.15s ease, transform 0.15s ease, border-color 0.15s ease
  &:active
    transform scale(0.96)
    background #eef5ff

.lang-chip__flag
  width 16px
  height 11px
  object-fit cover
  border-radius 2px
  box-shadow 0 0 0 1px rgba(0, 0, 0, 0.08)
  flex-shrink 0

.lang-chip__text
  font-size 11px
  font-weight 600
  line-height 1
  letter-spacing 0.02em

.lang-chip__arrow
  color #8fa0b3
  margin-left 0
  transition transform 0.2s ease
  &--open
    transform rotate(180deg)

.lang-menu
  position absolute
  top calc(100% + 6px)
  right 0
  min-width 148px
  padding 6px
  border-radius 12px
  background #fff
  border 1px solid #ebedf0
  box-shadow 0 8px 24px rgba(0, 0, 0, 0.12)
  z-index 50

.lang-menu__item
  display flex
  align-items center
  gap 10px
  width 100%
  margin 0
  padding 10px 12px
  border none
  border-radius 8px
  background transparent
  color #323233
  cursor pointer
  font-family inherit
  text-align left
  -webkit-tap-highlight-color transparent
  transition background 0.15s ease
  &:active
    background #f2f3f5
  &--active
    background #e8f8ef
    color #07c160
    font-weight 600

.lang-menu__flag
  width 22px
  height 15px
  object-fit cover
  border-radius 2px
  box-shadow 0 0 0 1px rgba(0, 0, 0, 0.08)
  flex-shrink 0

.lang-menu__label
  font-size 13px
  line-height 1
  white-space nowrap

.alipay-nav
  display flex
  align-items flex-start
  justify-content space-around
  gap 4px
  margin 0 -4px 16px
  padding 14px 4px 4px
  position relative
  z-index 25
  overflow visible

.alipay-nav__col
  flex 1
  min-width 0
  display flex
  flex-direction column
  align-items center

.alipay-nav__col--share
  position relative
  z-index 30

.alipay-nav__item
  width 100%
  margin 0
  padding 0
  border none
  background transparent
  display flex
  flex-direction column
  align-items center
  justify-content flex-start
  gap 8px
  cursor pointer
  font-family inherit
  -webkit-tap-highlight-color transparent
  touch-action manipulation
  &:active
    .alipay-nav__icon
      transform scale(0.92)
      opacity 0.88

.alipay-nav__icon
  width 48px
  height 48px
  border-radius 14px
  display flex
  align-items center
  justify-content center
  color #fff
  box-shadow 0 6px 16px rgba(0, 0, 0, 0.14)
  transition transform 0.15s ease, opacity 0.15s ease
  :deep(.van-icon)
    font-size 24px

.alipay-nav__label
  font-size 13px
  font-weight 600
  color #1f2937
  line-height 1.2
  text-align center
  white-space nowrap
  overflow hidden
  text-overflow ellipsis
  max-width 100%
  letter-spacing 0.02em

.alipay-nav__item--blue .alipay-nav__icon
  background linear-gradient(135deg, #4facfe 0%, #1677ff 100%)
.alipay-nav__item--orange .alipay-nav__icon
  background linear-gradient(135deg, #ffb347 0%, #ff8c00 100%)
.alipay-nav__item--green .alipay-nav__icon
  background linear-gradient(135deg, #43e97b 0%, #07c160 100%)
.alipay-nav__item--sky .alipay-nav__icon
  background linear-gradient(135deg, #70a1ff 0%, #3b82f6 100%)

.brand-bar
  display flex
  align-items center
  justify-content space-between
  gap 12px
  min-height 44px

.brand-main
  flex 1
  min-width 0

.brand-name-btn
  display inline-flex
  align-items center
  gap 8px
  max-width 100%
  padding 8px 12px 8px 14px
  border none
  border-radius 12px
  background linear-gradient(135deg, rgba(7, 193, 96, 0.1) 0%, rgba(7, 193, 96, 0.04) 100%)
  border 1px solid rgba(7, 193, 96, 0.18)
  cursor pointer
  text-align left
  -webkit-tap-highlight-color transparent
  transition background 0.2s ease, transform 0.15s ease, border-color 0.2s ease
  &:active
    transform scale(0.98)
    background rgba(7, 193, 96, 0.14)
  &.is-empty
    background #f7f8fa
    border-color #ebedf0
    .brand-name-text
      color #969799
      font-weight 500
    .brand-name-edit
      color #c8c9cc

.brand-name-text
  flex 1
  min-width 0
  overflow hidden
  text-overflow ellipsis
  white-space nowrap
  font-size 20px
  font-weight 700
  color #111
  letter-spacing 0.3px
  line-height 1.2

.brand-name-edit
  flex-shrink 0
  color #07c160
  opacity 0.85

.brand-name-input
  width 100%
  box-sizing border-box
  padding 10px 14px
  border 1.5px solid #07c160
  border-radius 12px
  outline none
  font-size 18px
  font-weight 600
  color #111
  background #fff
  box-shadow 0 0 0 3px rgba(7, 193, 96, 0.12)
  &::placeholder
    color #c8c9cc
    font-weight 500

.brand-name-skeleton
  height 40px
  width 56%
  max-width 220px
  border-radius 12px
  background linear-gradient(90deg, #f2f3f5, #ebedf0, #f2f3f5)
  background-size 200% 100%
  animation brandTitleShimmer 1.1s infinite linear

@keyframes brandTitleShimmer
  0%
    background-position -150% 0
  100%
    background-position 150% 0

.brand-bar-actions
  display flex
  align-items center
  gap 8px
  flex-shrink 0

.hero-body
  display grid
  grid-template-columns 110px 1fr
  gap 16px
  align-items start

// 头像区域：仿示例图的圆形头像 + 外发光环 + 底部钻石勋章
.avatar-block
  position relative
  width 110px
  height 110px
  display flex
  align-items center
  justify-content center

  // 外层发光圆环（圆形时）
  &::before
    content ''
    position absolute
    inset -4px
    border-radius 50%
    background radial-gradient(circle at 30% 0, rgba(255,255,255,0.9), transparent 55%), linear-gradient(135deg, #66a6ff, #89f7fe)
    opacity 0.95
    box-shadow 0 0 18px rgba(79,172,254,0.8)
    transition all 0.3s ease

  // 方形头像时，外层发光环变为方形
  &.avatar-square::before
    border-radius 12px
  
  // 长方形头像时，调整容器尺寸和外层发光环
  &.avatar-rectangle
    width 134px
    height 94px
    &::before
      border-radius 8px

.avatar-uploader
  position relative
  z-index 1
  transition all 0.3s ease

.avatar-uploader :deep(.van-uploader__upload)
  width 96px
  height 96px
  border-radius 50%
  overflow hidden
  padding 0
  margin 0
  border 3px solid rgba(255,255,255,0.9)
  background #000
  box-shadow 0 6px 14px rgba(0,0,0,0.45)
  transition all 0.3s ease

// 方形头像样式
.avatar-uploader.avatar-square :deep(.van-uploader__upload)
  border-radius 12px

// 长方形头像样式
.avatar-uploader.avatar-rectangle :deep(.van-uploader__upload)
  width 120px
  height 80px
  border-radius 8px

// 已上传头像预览区域
.avatar-uploader :deep(.van-uploader__preview)
  width 96px
  height 96px

// 长方形头像预览区域
.avatar-uploader.avatar-rectangle :deep(.van-uploader__preview)
  width 120px
  height 80px

.avatar-uploader :deep(.van-uploader__preview-image)
  width 100%
  height 100%
  border-radius 50%
  overflow hidden
  border 3px solid rgba(255,255,255,0.9)
  box-shadow 0 6px 14px rgba(0,0,0,0.45)
  transition all 0.3s ease

// 方形头像预览样式
.avatar-uploader.avatar-square :deep(.van-uploader__preview-image)
  border-radius 12px

// 长方形头像预览样式
.avatar-uploader.avatar-rectangle :deep(.van-uploader__preview-image)
  border-radius 8px

// 确保上传后的图片铺满
.avatar-uploader :deep(img),
.avatar-uploader :deep(.van-image__img)
  width 100%
  height 100%
  object-fit cover
  display block
  border-radius 50%
  transition all 0.3s ease

// 方形头像时，图片也变为方形
.avatar-uploader.avatar-square :deep(img),
.avatar-uploader.avatar-square :deep(.van-image__img)
  border-radius 12px

// 长方形头像时，图片也变为长方形
.avatar-uploader.avatar-rectangle :deep(img),
.avatar-uploader.avatar-rectangle :deep(.van-image__img)
  border-radius 8px

// 头像形状切换按钮
.avatar-shape-toggle
  position absolute
  bottom -8px
  right -8px
  width 32px
  height 32px
  border-radius 50%
  background linear-gradient(135deg, #4facfe, #00f2fe)
  display flex
  align-items center
  justify-content center
  cursor pointer
  box-shadow 0 4px 12px rgba(79,172,254,0.5)
  z-index 10
  transition all 0.3s ease
  color #fff
  &:active
    transform scale(0.9)
  &:hover
    box-shadow 0 6px 16px rgba(79,172,254,0.7)

// 等级勋章：底部圆角徽章，与圆形重叠
.level-chip
  position absolute
  left 50%
  bottom -2px
  transform translateX(-50%)
  padding 0 14px
  min-width 54px
  height 26px
  line-height 26px
  font-size 13px
  border-radius 999px
  background linear-gradient(135deg, #ffb347, #ffcc33)
  color #2a1c00
  font-weight 700
  text-align center
  box-shadow 0 6px 14px rgba(255,198,64,0.6)
  z-index 3

.user-meta
  display flex
  flex-direction column
  gap 14px
  padding 0 8px
  min-width 0
  overflow visible

.meta-header
  display flex
  flex-direction column
  gap 8px
  width 100%
  margin-top 4px
  overflow visible
  position relative
  z-index 20

.meta-row
  display flex
  align-items center
  gap 10px
  width 100%

.meta-row-username
  padding 8px 14px
  border-radius 14px
  background rgba(255,255,255,0.95)
  box-shadow 0 3px 10px rgba(0,0,0,0.08)
  box-sizing border-box
  gap 8px
  min-width 0
  max-width 100%
  .meta-label
    flex 0 0 auto
    padding 2px 8px
    margin-right 0
    font-size 13px
    font-weight 600
    color #6b7a8c
    background rgba(79,172,254,0.08)
    border 1px solid rgba(79,172,254,0.12)
    border-radius 6px
    line-height 1.3
  .meta-value
    flex 1 1 0
    display flex
    justify-content flex-start
    align-items center
    min-width 0
    overflow hidden
  .user-name-wrapper
    justify-content flex-start
    width 100%
    flex 1 1 0
    min-width 0
    max-width 100%
    overflow hidden
  .user-name
    display block
    width 100%
    max-width 100%
    min-width 0
    text-align left
    font-weight 500
    overflow hidden
    text-overflow ellipsis
    white-space nowrap

.meta-row-grid
  display grid
  grid-template-columns repeat(3, 1fr)
  gap 10px
  width 100%
  margin-top 2px

  @media (max-width: 200px)
    grid-template-columns repeat(2, 1fr)
    .meta-cell-action
      grid-column 1 / -1

.meta-label
  flex 0 0 auto
  padding 4px 10px
  font-size 14px
  font-weight 700
  color #6b7a8c
  background rgba(79,172,254,0.10)
  border 1px solid rgba(79,172,254,0.18)
  border-radius 999px
  line-height 1.1

.meta-value
  flex 1 1 auto
  min-width 0

.meta-cell
  display flex
  flex-direction column
  align-items center
  justify-content center
  padding 8px 10px
  border-radius 14px
  background rgba(255,255,255,0.96)
  box-shadow 0 3px 10px rgba(0,0,0,0.06)
  min-width 0
  min-height 44px

.meta-cell-label
  font-size 11px
  color #8c9aa9
  margin-bottom 2px

.meta-cell-value
  display flex
  align-items center
  justify-content center
  gap 6px
  width 100%
  min-width 0

.type-row-value
  display flex
  align-items center
  justify-content center
  gap 12px
  flex-wrap wrap
  margin-left -85px
  padding-left 85px
  width 100%
  position relative

.user-name-wrapper
  display flex
  align-items center
  gap 8px
  min-width 0
  overflow hidden

.user-meta-actions
  display flex
  align-items center
  gap 6px
  margin-left 8px

.user-level-tag
  padding 4px 10px
  font-size 12px
  font-weight 600
  border-radius 999px
  background linear-gradient(135deg, #ffb347, #ffcc33)
  color #2a1c00
  box-shadow 0 4px 10px rgba(255,198,64,0.5)
  cursor pointer

.user-level-tag.level-inactive
  background linear-gradient(135deg, #e0e0e0, #f5f5f5)
  color #888
  box-shadow none

.signin-btn
  padding 0 10px
  height 26px
  border-radius 999px
  font-size 12px
  font-weight 600
  margin-left auto
  margin-right 0

// 等级/类型/签到区域里的签到按钮，居中显示
.meta-row-grid .signin-btn
  margin-left 0
  margin-right 0

.signin-btn-done
  background #b7c4d6 !important
  border-color #b7c4d6 !important

.signin-calendar
  padding 20px 18px 22px

  .signin-calendar-head
    display flex
    align-items center
    justify-content space-between
    font-size 16px
    font-weight 700
    color #2c3e50

  .signin-calendar-sub
    display flex
    align-items center
    justify-content space-between
    margin-top 8px
    font-size 12px
    color #909399

  .signin-streak
    color #ff9f43

  .signin-weekdays
    display grid
    grid-template-columns repeat(7, 1fr)
    margin-top 14px
    font-size 12px
    color #909399
    text-align center

  .signin-days
    display grid
    grid-template-columns repeat(7, 1fr)
    gap 6px 0
    margin-top 8px

  .signin-day
    position relative
    height 42px
    display flex
    flex-direction column
    align-items center
    justify-content center
    gap 1px
    font-size 13px
    color #333

    &.is-empty
      visibility hidden

    .signin-day-num
      line-height 1.2

    &.is-today .signin-day-num
      color #1989fa
      font-weight 700

    &.is-signed .signin-day-num
      color #07c160
      font-weight 600

    .signin-day-check
      font-size 12px
      line-height 1
      color #07c160
      font-weight 700

  .signin-calendar-btn
    margin-top 18px

.user-name
  font-size 22px
  font-weight 700
  color #2c3e50
  line-height 1.2
  letter-spacing 0.5px
  white-space nowrap
  overflow hidden
  text-overflow ellipsis
  min-width 0
  max-width 100%
  box-sizing border-box

.user-power
  font-size 14px
  font-weight 600
  color #4facfe
  padding 4px 10px
  background rgba(79,172,254,0.1)
  border-radius 8px
  display inline-block
  width fit-content

.user-type
  display inline-flex
  align-items center
  justify-content center
  padding 0
  border-radius 999px
  background transparent
  border none
  cursor pointer  
  box-shadow none
  flex-shrink 0
  transition all 0.2s ease
  &:active
    transform scale(0.96)

.tag
  display inline-flex
  align-items center
  justify-content center
  min-width 52px
  padding 4px 12px
  border-radius 999px
  font-size 12px
  font-weight 600
  letter-spacing 0.5px
  background rgba(255,255,255,0.85)
  color #7b8895
  box-shadow 0 4px 12px rgba(79,172,254,0.18)
  border 1px solid rgba(255,255,255,0.8)
  transition all 0.3s ease

.tag.normal.active
  background linear-gradient(135deg, #e3f2ff 0%, #c2e0ff 100%)
  color #2470ff
  box-shadow 0 4px 14px rgba(36,112,255,0.45)
  border-color rgba(255,255,255,0.9)

.tag.vip.active
  background linear-gradient(135deg, #ffe6c3 0%, #ffc073 100%)
  color #b05700
  box-shadow 0 4px 14px rgba(255,176,82,0.55)
  border-color rgba(255,255,255,0.9)

.power-balance-container
  display flex
  flex-direction column
  gap 12px
  margin-top 0

.power-balance-top-grid
  display grid
  grid-template-columns repeat(2, 1fr)
  gap 12px
  width 100%
  box-sizing border-box

.power-balance-item
  padding 14px 16px
  border none
  border-radius 16px
  background rgba(255,255,255,0.9)
  display flex
  flex-direction column
  gap 8px
  box-shadow 0 4px 16px rgba(0,0,0,0.1)
  transition all 0.3s ease
  position relative
  width 100%
  box-sizing border-box
  min-width 0
  &:hover
    transform translateY(-2px)
    box-shadow 0 6px 20px rgba(0,0,0,0.15)
  // 复投输入的样式
  &.power-balance-reinvest
    background linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)
    color #fff
    box-shadow 0 6px 20px rgba(79,172,254,0.3)
    padding 12px
    display flex
    flex-direction column
    justify-content center
    .reinvest-input-wrapper
      display flex
      align-items center
      gap 8px
      width 100%
      .reinvest-input-field
        flex 1
        min-width 0
        // 覆盖全局 van-field 白底（否则在渐变背景上会变成一个白块，且文字/光标不明显）
        :deep(.van-field)
          background transparent !important
          width 100%
        :deep(.van-field__wrapper)
          background transparent !important
          width 100%
        :deep(.van-field__body)
          background transparent !important
          border none !important
          border-radius 0
          padding 0
          width 100%
          flex 1
          min-width 0
        // 使用最具体的选择器确保优先级，覆盖全局样式
        :deep(.van-field__control)
          color #fff !important
          caret-color #fff !important
          font-weight 600 !important
          font-size 16px !important
          -webkit-text-fill-color #fff !important
          width 100% !important
          min-width 0 !important
          overflow visible !important
          text-overflow clip !important
          white-space nowrap !important
          &::placeholder
            color rgba(255,255,255,0.6) !important
            -webkit-text-fill-color rgba(255,255,255,0.6) !important
            opacity 1 !important
        // 直接针对 input 元素，使用属性选择器提高优先级
        :deep(input[type="number"])
          color #fff !important
          caret-color #fff !important
          -webkit-text-fill-color #fff !important
          width 100% !important
          min-width 0 !important
          overflow visible !important
          text-overflow clip !important
          white-space nowrap !important
          &::placeholder
            color rgba(255,255,255,0.6) !important
            -webkit-text-fill-color rgba(255,255,255,0.6) !important
            opacity 1 !important
        :deep(input)
          color #fff !important
          caret-color #fff !important
          -webkit-text-fill-color #fff !important
          width 100% !important
          min-width 0 !important
          overflow visible !important
          text-overflow clip !important
          white-space nowrap !important
          &::placeholder
            color rgba(255,255,255,0.6) !important
            -webkit-text-fill-color rgba(255,255,255,0.6) !important
            opacity 1 !important
        // 针对所有可能的输入元素
        :deep(textarea)
          color #fff !important
          caret-color #fff !important
          -webkit-text-fill-color #fff !important
        // 确保清除按钮可见
        :deep(.van-field__clear-icon)
          color rgba(255,255,255,0.8) !important
          &:hover
            color #fff !important
      .reinvest-confirm-btn
        flex-shrink 0
        padding 0 12px
        height 36px
        min-width 60px
        background rgba(255,255,255,0.9)
        color #4facfe
        border none
        font-weight 600
        white-space nowrap
        display flex
        align-items center
        justify-content center
        line-height 1
        font-size 16px
        &:active
          transform scale(0.98)
          background rgba(255,255,255,0.8)
        &:disabled
          opacity 0.6

// 上方两个框的样式
.power-balance-top-grid .power-balance-item:nth-child(1)
  background linear-gradient(135deg, #ff9a56 0%, #ff6a88 100%)
  color #fff
  box-shadow 0 6px 20px rgba(255,154,86,0.3)

.power-balance-item.exited-card
  background linear-gradient(135deg, #9ca3af 0%, #d1d5db 100%)
  color #fff
  box-shadow 0 6px 20px rgba(156,163,175,0.3)

.power-balance-top-grid .power-balance-item:nth-child(2)
  background linear-gradient(135deg, #667eea 0%, #764ba2 100%)
  color #fff
  box-shadow 0 6px 20px rgba(102,126,234,0.3)

.power-balance-top-grid .power-balance-item:nth-child(3)
  background linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)
  color #fff
  box-shadow 0 6px 20px rgba(79,172,254,0.3)

// 算力余额特殊样式 - 更突出
.power-balance-item.power-balance-main
  padding 20px 20px
  box-shadow 0 8px 24px rgba(79,172,254,0.4)
  border 2px solid rgba(255,255,255,0.3)
  .label
    font-size 16px
    font-weight 600
  .value
    font-size 22px
    font-weight 900
    margin-top 4px

.credit-card
  margin-top 0
  padding 14px 16px
  border-radius 16px
  background linear-gradient(135deg, #434e9e 0%, #5b6fd8 48%, #6a8cff 100%)
  border 1px solid rgba(255, 255, 255, 0.22)
  box-shadow 0 6px 20px rgba(67, 78, 158, 0.35)
  color #fff

.credit-card__head
  display flex
  align-items center
  justify-content space-between
  margin-bottom 10px

.credit-card__title
  display flex
  align-items center
  gap 6px
  font-size 13px
  font-weight 700
  color rgba(255, 255, 255, 0.95)

.credit-card__detail-icon
  opacity 0.9
  cursor pointer
  &:active
    opacity 0.7

.credit-card__actions
  display flex
  align-items center
  gap 6px

.credit-card__action
  padding 4px 12px
  border none
  border-radius 8px
  font-size 12px
  font-weight 800
  color #fff
  background linear-gradient(135deg, #ff6b4a 0%, #e8210f 100%)
  box-shadow 0 4px 12px rgba(232, 33, 15, 0.45)
  cursor pointer
  &:active
    transform scale(0.96)
  &--ghost
    background rgba(255, 255, 255, 0.16)
    box-shadow none
    border 1px solid rgba(255, 255, 255, 0.28)

.credit-card__row
  display grid
  grid-template-columns repeat(3, 1fr)
  gap 8px

.credit-card__cell
  display flex
  flex-direction column
  gap 4px
  span
    font-size 11px
    color rgba(255, 255, 255, 0.78)
  strong
    font-size 14px
    font-weight 800
    letter-spacing 0.02em
    color #fff
  .is-avail
    color #9ef0c4
  .is-debt
    color #ffb4b4

.credit-card__hint
  margin-top 10px
  padding-top 8px
  border-top 1px solid rgba(255, 255, 255, 0.18)
  font-size 11px
  line-height 1.5
  color rgba(255, 255, 255, 0.82)

.credit-detail
  display flex
  flex-direction column
  height 100%
  background #f5f6fa
  color #1f2330

.credit-detail__nav
  display flex
  align-items center
  justify-content space-between
  padding 14px 16px 8px
  background #fff

.credit-detail__title
  font-size 16px
  font-weight 700

.credit-detail__close
  color #888
  cursor pointer

.credit-detail__summary
  display grid
  grid-template-columns repeat(3, 1fr)
  gap 8px
  margin 10px 12px 0
  padding 12px
  border-radius 12px
  background linear-gradient(135deg, #434e9e 0%, #5b6fd8 48%, #6a8cff 100%)
  color #fff

.credit-detail__cell
  display flex
  flex-direction column
  gap 4px
  span
    font-size 11px
    opacity 0.8
  strong
    font-size 14px
    font-weight 800
  .is-avail
    color #9ef0c4
  .is-debt
    color #ffb4b4

.credit-detail__list
  flex 1
  overflow auto
  padding 10px 12px 20px

.credit-detail__items
  display flex
  flex-direction column
  gap 8px

.credit-detail__item
  padding 12px
  border-radius 12px
  background #fff
  box-shadow 0 1px 4px rgba(31, 35, 48, 0.06)

.credit-detail__item-head
  display flex
  align-items center
  justify-content space-between
  gap 8px

.credit-detail__model
  font-size 13px
  font-weight 700
  color #1f2330

.credit-detail__amount
  font-size 14px
  font-weight 800
  &.positive
    color #16a34a
  &.negative
    color #e11d48

.credit-detail__remark
  margin-top 6px
  font-size 12px
  line-height 1.45
  color #667085

.credit-detail__meta
  margin-top 8px
  display flex
  justify-content space-between
  gap 8px
  font-size 11px
  color #98a2b3

.credit-detail__more
  display flex
  justify-content center
  padding 8px 0 4px

.power-balance-top-grid .search-icon
  cursor pointer
  opacity 0.8
  transition all 0.2s ease
  padding 4px
  border-radius 50%
  background rgba(255,255,255,0.2)
  flex-shrink 0
  display inline-flex
  align-items center
  justify-content center
  flex-shrink 0
  &:hover
    opacity 1
    background rgba(255,255,255,0.3)
    transform scale(1.1)
  &:active
    transform scale(0.95)

.label
  font-size 13px
  color rgba(255,255,255,0.9)
  letter-spacing 0.3px
  font-weight 600
  display flex
  align-items center
  justify-content center
  gap 6px

.value
  font-size 20px
  font-weight 800
  color #fff
  letter-spacing 0.2px
  text-align center

.value .inline-btn
  margin-left 8px
  padding 4px 12px
  height 28px
  line-height 20px
  border-color rgba(255,255,255,0.5)
  color #fff
  background rgba(255,255,255,0.2)
  &:active
    transform translateY(1px)

.value.emphasis
  letter-spacing 1px

.value.positive
  color inherit

.value.exited
  color #fff

.value.only-text
  display flex
  align-items center
  justify-content center
  height 100%

.power-detail-dialog
  :deep(.van-dialog__header)
    font-weight 800
    color #2c3e50
  :deep(.van-dialog)
    background rgba(255,255,255,0.98)
    border-radius 20px
  .power-detail-body
    display flex
    flex-direction column
    gap 14px
    padding 8px 6px
  .power-summary
    display grid
    grid-template-columns repeat(2, 1fr)
    gap 12px
  .summary-card
    padding 16px
    border-radius 16px
    background rgba(255,255,255,0.9)
    border 1px solid rgba(79,172,254,0.2)
    box-shadow 0 4px 16px rgba(0,0,0,0.1)
  .summary-label
    font-size 13px
    color #7f8c8d
    font-weight 600
  .summary-value
    margin-top 8px
    font-size 24px
    font-weight 800
    color #4facfe
    letter-spacing 0.3px
  .records-header
    display flex
    align-items center
    justify-content space-between
    gap 10px
    color #2c3e50
    font-weight 600
  .records-actions
    display flex
    gap 8px
  .record-list
    display flex
    flex-direction column
    gap 10px
    max-height 320px
    overflow-y auto
    padding-right 4px
  .record-item
    padding 12px
    border-radius 12px
    background rgba(255,255,255,0.8)
    border 1px solid rgba(79,172,254,0.2)
    box-shadow 0 2px 8px rgba(0,0,0,0.08)
  .record-title
    display flex
    justify-content space-between
    align-items center
    color #2c3e50
    font-weight 700
  .record-title .model
    font-size 14px
  .record-meta
    margin-top 6px
    display flex
    justify-content space-between
    color #7f8c8d
    font-size 12px
  .amount
    font-size 18px
    font-weight 800
  .amount.positive
    color #43e97b
  .amount.negative
    color #ff6b81

.link
  color #4facfe
  cursor pointer
  font-weight 600
  &:hover
    color #00f2fe

// 统一所有标签的基准样式
.payment-row .label,
.power-balance-bottom .power-balance-row .label,
.account-info-item .info-label
  min-width 70px
  font-size 13px
  font-weight 600
  line-height 1.5
  padding 0
  margin 0
  box-sizing border-box
  display flex
  align-items center
  height auto

.payment-row
  display flex
  align-items center
  gap 12px
  padding 14px 16px
  background rgba(255,255,255,0.8)
  border-radius 14px
  border 1px solid rgba(79,172,254,0.2)
  box-shadow 0 2px 8px rgba(0,0,0,0.04)
  transition all 0.25s ease
  &:hover
    background rgba(255,255,255,0.95)
    border-color rgba(79,172,254,0.3)
    box-shadow 0 4px 12px rgba(79,172,254,0.08)
  .label
    color #2c3e50

.payment-button
  flex 1
  display flex
  align-items center
  justify-content space-between
  padding 12px 16px
  background rgba(255,255,255,0.95)
  border-radius 12px
  cursor pointer
  transition all 0.25s cubic-bezier(0.4, 0, 0.2, 1)
  border 1px solid rgba(79,172,254,0.1)
  &:hover
    background rgba(255,255,255,1)
    border-color rgba(79,172,254,0.2)
    box-shadow 0 2px 8px rgba(79,172,254,0.08)
  &:active
    transform scale(0.98)
    background rgba(255,255,255,0.9)
  .payment-type
    display flex
    align-items center
    color #2c3e50
    font-weight 600
    font-size 14px
  .empty-address
    color #95a5a6
    font-weight 400
  .edit-icon
    cursor pointer
    transition all 0.3s
    padding 6px
    border-radius 50%
    border 1.5px solid #4facfe
    display inline-flex
    align-items center
    justify-content center
    &:hover
      background rgba(79,172,254,0.1)
      transform scale(1.1)
      border-color #3a9de8
    &:active
      transform scale(0.95)

.bsc-row
  display flex
  align-items center
  gap 12px
  padding 12px
  background rgba(255,255,255,0.6)
  border-radius 12px
  border 1px solid rgba(79,172,254,0.3)
  .label
    color #2c3e50
    font-weight 600

.bsc-input
  flex 1
  background rgba(255,255,255,0.9)
  border-radius 10px
  :deep(.van-field__control)
    color #2c3e50
  :deep(.van-field__control::placeholder)
    color #95a5a6

.eth-address-dialog
  :deep(.van-dialog__header)
    font-weight 800
    color #2c3e50
    padding 20px 20px 16px
  :deep(.van-dialog__content)
    padding 0 20px 20px

.eth-address-content
  display flex
  flex-direction column
  gap 16px
  padding 8px 0

.eth-address-field
  background rgba(255,255,255,0.8)
  border-radius 10px
  :deep(.van-field__control)
    color #2c3e50
    font-size 14px
  :deep(.van-field__control::placeholder)
    color #95a5a6

.eth-address-tip
  display flex
  align-items center
  gap 6px
  color #7f8c8d
  font-size 12px
  padding 8px 12px
  background rgba(79,172,254,0.1)
  border-radius 8px

.payment-dialog
  :deep(.van-dialog__header)
    font-weight 800
    color #2c3e50
    padding 20px 20px 16px
  :deep(.van-dialog__content)
    padding 0 20px 20px

.payment-methods-content
  display flex
  flex-direction column
  gap 20px

.payment-method-item
  padding 16px
  background rgba(255,255,255,0.8)
  border-radius 12px
  border 1px solid rgba(79,172,254,0.2)
  box-shadow 0 2px 8px rgba(0,0,0,0.05)
  transition all 0.3s ease
  &.is-default
    border-color #4facfe
    border-width 2px
    background rgba(79,172,254,0.05)
    box-shadow 0 4px 12px rgba(79,172,254,0.2)

.payment-method-header
  display flex
  align-items center
  justify-content space-between
  margin-bottom 12px
  padding-bottom 12px
  border-bottom 1px solid rgba(79,172,254,0.15)

.payment-method-title
  display flex
  align-items center
  gap 8px
  font-weight 700
  font-size 16px
  color #2c3e50
  .default-icon
    margin-left 4px

.payment-field
  :deep(.van-field__label)
    min-width 80px
    color #2c3e50
    font-weight 600
  :deep(.van-field__control)
    color #2c3e50
  :deep(.van-field__control::placeholder)
    color #95a5a6

.default-payment-tip
  display flex
  align-items center
  gap 6px
  padding 10px 14px
  background rgba(79,172,254,0.1)
  border-radius 8px
  font-size 13px
  color #4facfe
  font-weight 600

.account-info
  display flex
  flex-direction column
  gap 0
  margin-top 12px
  padding 16px
  background rgba(255,255,255,0.8)
  border-radius 14px
  border 1px solid rgba(79,172,254,0.15)
  box-shadow 0 2px 8px rgba(0,0,0,0.04)

.account-info-item
  display flex
  align-items center
  justify-content space-between
  padding 0
  transition all 0.2s ease
  &:not(:last-child)
    border-bottom 1px solid rgba(79,172,254,0.12)
    padding-bottom 14px
    margin-bottom 14px
  &:hover
    .info-value
      color #4facfe

.info-label
  color #7f8c8d
  letter-spacing 0.3px

.info-value
  font-size 14px
  color #2c3e50
  font-weight 600
  letter-spacing 0.2px
  display flex
  align-items center
  gap 8px
  .link
    color #667eea
    cursor pointer
    transition all 0.2s ease
    &:hover
      color #764ba2
      text-decoration underline
  .emphasis
    color #667eea
    font-weight 700
    font-size 15px

.lang-active
  background linear-gradient(135deg, #667eea 0%, #764ba2 100%) !important
  color #fff !important
  border none !important

.section-card .section-header,
.card-header
  display flex
  align-items center
  justify-content space-between
  gap 10px
  margin-bottom 0
  padding-bottom 16px
  border-bottom none

.title
  font-size 20px
  font-weight 700
  color #1f2937
  letter-spacing 0.3px
  position relative
  &::after
    content ''
    position absolute
    bottom -12px
    left 0
    width 32px
    height 2px
    background #667eea
    border-radius 1px

.list-card
  overflow-x visible !important
  overflow-y visible
  
  .list-body
    overflow-x visible !important
    overflow-y visible !important

.list-card .title::after
  width 40px

.hint
  font-size 13px
  color #7f8c8d
  letter-spacing 0.2px

.modern-grid
  padding 12px 0

.quick-actions-card
  padding 18px 14px 16px !important

.quick-actions-grid
  display grid
  grid-template-columns repeat(4, minmax(0, 1fr))
  gap 10px

.quick-action
  position relative
  z-index 1
  margin 0
  padding 14px 6px 12px
  border none
  border-radius 16px
  background linear-gradient(180deg, #ffffff 0%, #f7fbff 100%)
  border 1px solid rgba(79, 172, 254, 0.12)
  box-shadow 0 4px 14px rgba(79, 172, 254, 0.06)
  display flex
  flex-direction column
  align-items center
  justify-content center
  gap 10px
  cursor pointer
  font-family inherit
  -webkit-tap-highlight-color transparent
  touch-action manipulation
  transition transform 0.15s ease, box-shadow 0.2s ease, border-color 0.2s ease
  &:active
    transform scale(0.96)

  &__icon
    position relative
    width 56px
    height 56px
    border-radius 16px
    display flex
    align-items center
    justify-content center
    color #fff
    box-shadow 0 6px 18px rgba(0, 0, 0, 0.16)
    :deep(.van-icon)
      font-size 26px

  &__badge
    position absolute
    top -4px
    right -6px
    min-width 18px
    height 18px
    padding 0 5px
    border-radius 999px
    background #f04438
    color #fff
    font-size 11px
    font-weight 700
    font-style normal
    line-height 18px
    text-align center
    box-shadow 0 2px 6px rgba(240, 68, 56, 0.45)
    z-index 2

  &__label
    font-size 14px
    font-weight 700
    color #374151
    line-height 1.25
    text-align center
    white-space nowrap
    overflow hidden
    text-overflow ellipsis
    max-width 100%
    letter-spacing 0.02em

  &--mint &__icon
    background linear-gradient(135deg, #43e97b, #38f9d7)
  &--gold &__icon
    background linear-gradient(135deg, #ffd666, #faad14)
  &--orange &__icon
    background linear-gradient(135deg, #ffa502, #ff8c00)
  &--rose &__icon
    background linear-gradient(135deg, #ff6b81, #ff4757)
  &--cyan &__icon
    background linear-gradient(135deg, #4facfe, #00f2fe)
  &--blue &__icon
    background linear-gradient(135deg, #1e90ff, #0066cc)
  &--coral &__icon
    background linear-gradient(135deg, #ff6b35, #e55a2b)
  &--sky &__icon
    background linear-gradient(135deg, #70a1ff, #5a8fff)
  &--slate &__icon
    background linear-gradient(135deg, #a8b5c6, #8896a6)
  &--indigo &__icon
    background linear-gradient(135deg, #5b8def, #3b6fd9)
  &--violet &__icon
    background linear-gradient(135deg, #667eea, #764ba2)
  &--gray &__icon
    background linear-gradient(135deg, #747d8c, #5a6268)

.modern-grid :deep(.van-grid-item)
  padding 6px
  
.modern-grid :deep(.van-grid-item__content)
  background transparent
  border none
  border-radius 0
  box-shadow none
  transition all 0.2s ease
  position relative
  overflow visible
  padding 12px 8px
  display flex
  flex-direction column
  align-items center
  justify-content center
  gap 12px
  min-height auto
  &:active
    transform scale(0.95)
    opacity 0.8

.grid-icon-wrapper
  width 55px
  height 55px
  border-radius 16px
  display flex
  align-items center
  justify-content center
  box-shadow 0 6px 20px rgba(0,0,0,0.25), 0 3px 10px rgba(0,0,0,0.18)
  transition all 0.2s cubic-bezier(0.4, 0, 0.2, 1)
  position relative
  overflow hidden
  &:active
    transform scale(0.95)
    box-shadow 0 3px 10px rgba(0,0,0,0.2)

.modern-grid :deep(.van-grid-item__icon)
  margin-bottom 0
  display flex
  align-items center
  justify-content center

.modern-grid :deep(.van-grid-item__text)
  margin-top 0
  padding-top 0

.grid-text
  color #2c3e50
  font-weight 500
  font-size 13px
  letter-spacing 0.3px
  transition color 0.2s ease
  text-align center
  white-space nowrap

.card-grid
  display grid
  grid-template-columns repeat(auto-fit, minmax(260px, 1fr))
  gap 14px

.qr-wrapper
  display flex
  justify-content center
  padding 10px 0 6px

.qr-img
  max-width 240px
  border-radius 16px
  overflow hidden
  box-shadow 0 8px 24px rgba(0,0,0,0.15)
  border 2px solid rgba(255,255,255,0.8)

.link-row
  margin-top 12px
  display flex
  align-items center
  gap 10px
  padding 12px 14px
  background rgba(255,255,255,0.9)
  border 1px dashed #4facfe
  border-radius 12px
  font-size 13px
  color #2c3e50
  word-break break-all
  box-shadow 0 2px 8px rgba(0,0,0,0.08)

.copy-icon
  cursor pointer
  color #4facfe
  margin-left 4px
  font-size 18px
  transition all 0.2s ease
  &:hover
    color #00f2fe
    transform scale(1.1)

.add-icon-circle
  cursor pointer
  margin-left 8px
  width 24px
  height 24px
  border-radius 50%
  background #4facfe
  display inline-flex
  align-items center
  justify-content center
  transition all 0.2s ease
  flex-shrink 0
  &:hover
    background #00f2fe
    transform scale(1.1)
  &:active
    transform scale(0.95)

.stats-content
  display flex
  flex-direction column
  gap 0

.stat-wrap
  display grid
  grid-template-columns repeat(3, 1fr)
  gap 12px
  margin-top 16px

.stat-pill
  padding 20px 16px
  border-radius 16px
  background #ffffff
  border 1px solid rgba(0,0,0,0.08)
  text-align center
  box-shadow 0 2px 8px rgba(0,0,0,0.06)
  transition all 0.3s ease
  display flex
  flex-direction column
  align-items center
  justify-content center
  gap 8px

.team-overview-card
  padding 18px 16px 16px !important
  overflow hidden

.team-overview-head
  display flex
  align-items flex-start
  justify-content space-between
  gap 12px
  margin-bottom 16px

.team-overview-head__main
  min-width 0

.team-overview-kicker
  display inline-block
  margin-bottom 4px
  font-size 11px
  font-weight 600
  letter-spacing 0.14em
  color #4facfe

.team-overview-title
  margin 0
  font-size 20px
  font-weight 700
  color #1f2937
  letter-spacing 0.02em
  line-height 1.25

.team-referrer-chip
  display inline-flex
  align-items center
  gap 4px
  max-width 58%
  padding 7px 10px
  border-radius 999px
  background rgba(79, 172, 254, 0.1)
  border 1px solid rgba(79, 172, 254, 0.18)
  color #5b6b7c
  font-size 12px
  line-height 1.2
  :deep(.van-icon)
    color #4facfe
    font-size 14px
    flex-shrink 0

.team-referrer-chip__label
  flex-shrink 0
  color #7b8794

.team-referrer-chip__value
  min-width 0
  overflow hidden
  text-overflow ellipsis
  white-space nowrap
  color #2c7be5
  font-weight 600

.team-metrics
  display grid
  grid-template-columns repeat(3, 1fr)
  gap 10px
  margin-bottom 14px

.team-metric
  position relative
  display flex
  flex-direction column
  align-items center
  gap 6px
  padding 14px 8px 12px
  border-radius 16px
  background linear-gradient(180deg, #ffffff 0%, #f7fbff 100%)
  border 1px solid rgba(79, 172, 254, 0.12)
  box-shadow 0 4px 14px rgba(79, 172, 254, 0.06)

  &__icon
    width 30px
    height 30px
    border-radius 10px
    display flex
    align-items center
    justify-content center
    color #fff
    :deep(.van-icon)
      font-size 16px

  &--direct &__icon
    background linear-gradient(135deg, #4facfe, #00f2fe)
  &--active &__icon
    background linear-gradient(135deg, #43e97b, #38f9d7)
  &--team &__icon
    background linear-gradient(135deg, #667eea, #5b8cff)

  &__value
    font-size 24px
    font-weight 600
    color #1f2937
    line-height 1.1
    letter-spacing 0.01em
    font-variant-numeric tabular-nums
    font-synthesis none
    -webkit-font-smoothing antialiased

  &__label
    font-size 15px
    font-weight 500
    color #5b6b7c

.team-zone
  padding 14px
  border-radius 16px
  background linear-gradient(145deg, rgba(79, 172, 254, 0.08), rgba(67, 233, 123, 0.08))
  border 1px solid rgba(79, 172, 254, 0.12)

  &__head
    display flex
    align-items center
    justify-content space-between
    gap 8px
    margin-bottom 10px

  &__title
    font-size 16px
    font-weight 600
    color #374151

  &__total
    font-size 15px
    font-weight 500
    color #4b5563
    em
      margin-left 4px
      font-style normal
      font-weight 600
      color #2c7be5
      font-variant-numeric tabular-nums
      font-size 16px
      letter-spacing 0.01em
      font-synthesis none
      -webkit-font-smoothing antialiased

  &__track
    display flex
    height 8px
    border-radius 999px
    overflow hidden
    background rgba(255, 255, 255, 0.7)
    margin-bottom 12px

  &__fill
    height 100%
    transition width 0.35s ease
    &--left
      background linear-gradient(90deg, #4facfe, #00c6fb)
    &--right
      background linear-gradient(90deg, #43e97b, #38f9d7)

  &__cols
    display grid
    grid-template-columns 1fr 1fr
    gap 10px

  &__col
    padding 10px 12px
    border-radius 12px
    background rgba(255, 255, 255, 0.88)
    border 1px solid rgba(255, 255, 255, 0.9)

  &__col-top
    display flex
    align-items center
    gap 6px
    font-size 15px
    font-weight 500
    color #4b5563
    margin-bottom 6px

  &__dot
    width 8px
    height 8px
    border-radius 50%
    flex-shrink 0

  &__col--left &__dot
    background #4facfe
  &__col--right &__dot
    background #43e97b

  &__num
    font-size 20px
    font-weight 600
    color #1f2937
    line-height 1.2
    letter-spacing 0.01em
    font-variant-numeric tabular-nums
    font-synthesis none
    -webkit-font-smoothing antialiased

  &__pct
    margin-top 2px
    font-size 11px
    font-weight 500
    color #8a97a5

.team-users-card
  .team-users-head
    display flex
    align-items flex-start
    justify-content space-between
    gap 10px
    margin-bottom 12px

  .team-users-count
    min-width 28px
    height 28px
    padding 0 8px
    border-radius 999px
    display inline-flex
    align-items center
    justify-content center
    font-size 13px
    font-weight 800
    color #2c7be5
    background rgba(79, 172, 254, 0.12)

.referrer-info
  display flex
  align-items center
  gap 4px
  font-size 16px
  white-space nowrap

.referrer-label
  font-size 16px
  font-weight 500
  color #6b7280

.referrer-value
  font-size 18px
  font-weight 700
  color #4facfe
  line-height 1
  display inline-block
  transform translateY(1px)
  overflow hidden
  text-overflow ellipsis
  white-space nowrap
  max-width 120px

.performance-row
  display flex
  align-items center
  gap 0
  margin-top 20px
  padding 20px 24px
  background #ffffff
  border 1px solid rgba(0,0,0,0.08)
  border-radius 16px
  box-shadow 0 2px 8px rgba(0,0,0,0.06)

.performance-item
  flex 1
  display flex
  flex-direction column
  gap 8px
  align-items center
  justify-content center
  padding 0

.perf-label
  font-size 14px
  color #6b7280
  letter-spacing 0.2px
  font-weight 500

.perf-value
  font-size 24px
  font-weight 700
  color #4facfe
  letter-spacing 0.3px
  line-height 1.2

.performance-divider
  width 1px
  height 50px
  background rgba(0,0,0,0.1)
  flex-shrink 0
  margin 0 20px

.list-card .list-body
  margin-top 0
  background transparent
  border-radius 12px
  padding 0
  box-shadow none
  transition all 0.3s ease
  overflow-x visible
  overflow-y visible

.bottom-spacing
  height 70px

.announcement-content
  max-height 320px
  overflow-y auto
  padding-right 10px
  color #2c3e50
  line-height 1.6

.scan-container
  position relative
  width 100%
  height 300px
  background #2c3e50
  border-radius 16px
  overflow hidden
  box-shadow 0 4px 16px rgba(0,0,0,0.2)

.transfer-scan-icon
  cursor pointer
  padding 4px

.receive-qr-wrapper
  position relative
  padding 12px 16px 20px
  text-align center

.receive-qr-close
  position absolute
  top 0
  right 8px
  cursor pointer
  z-index 2

.receive-qr-img-box
  display flex
  justify-content center
  align-items center
  padding 8px
  background #fff
  border-radius 12px

.receive-qr-img
  width 220px
  height 220px

.receive-qr-empty
  padding 48px 16px
  color #95a5a6
  font-size 14px

.receive-qr-code
  margin 12px 0 4px
  font-size 18px
  font-weight 700
  color #2c3e50
  letter-spacing 1px
  word-break break-all

.receive-qr-tip
  margin 0 0 16px
  font-size 13px
  color #7f8c8d

.receive-qr-copy-btn
  border-radius 10px
  background linear-gradient(135deg, #5352ed 0%, #3742fa 100%)
  border none

.receive-invite-block
  margin-top 18px
  padding-top 14px
  border-top 1px dashed #ebedf0
  text-align left

.receive-invite-label
  font-size 13px
  color #969799
  margin-bottom 8px

.receive-invite-row
  display flex
  align-items center
  gap 8px

.receive-invite-link
  flex 1
  min-width 0
  font-size 12px
  color #323233
  word-break break-all
  line-height 1.4
  background #f7f8fa
  border-radius 8px
  padding 8px 10px

.receive-invite-copy
  flex-shrink 0
  border-radius 8px

.scan-video
  width 100%
  height 100%
  object-fit cover

.scan-overlay
  position absolute
  inset 0
  display flex
  flex-direction column
  align-items center
  justify-content center
  pointer-events none

.scan-box
  width 200px
  height 200px
  border 2px solid #fff
  border-radius 12px
  position relative
  &::before,
  &::after
    content ''
    position absolute
    width 20px
    height 20px
    border 3px solid #4facfe
  &::before
    top -3px
    left -3px
    border-right none
    border-bottom none
  &::after
    bottom -3px
    right -3px
    border-left none
    border-top none

.scan-line
  position absolute
  top 0
  left 0
  width 100%
  height 2px
  background linear-gradient(90deg, transparent, #4facfe, transparent)
  animation scanAnimation 2s linear infinite

@keyframes scanAnimation
  0%
    top 0
    opacity 1
  50%
    opacity 1
  100%
    top 100%
    opacity 0

.scan-tip
  color #fff
  margin-top 16px
  font-size 13px
  text-align center
  pointer-events none

.scan-actions
  margin-top 12px
  display flex
  gap 8px
  pointer-events auto
  .van-button
    background rgba(255,255,255,0.2)
    border 1px solid rgba(255,255,255,0.3)
    color #fff

.node-info
  padding 16px
  p
    margin 10px 0
    color #2c3e50
    font-size 14px
    line-height 1.6

// 对话框样式覆盖
:deep(.van-dialog)
  background rgba(255,255,255,0.98)
  border-radius 20px
  box-shadow 0 12px 40px rgba(0,0,0,0.15)

:deep(.van-dialog__header)
  color #2c3e50
  font-weight 700

:deep(.van-dialog__message)
  color #2c3e50

:deep(.van-field)
  background rgba(255,255,255,0.9)
  border-radius 8px

:deep(.van-field__control)
  color #2c3e50

:deep(.van-field__control::placeholder)
  color #95a5a6

.dialog-tip
  margin-top 8px
  font-size 12px
  color #9aa3c2
  text-align right

// 复投弹窗样式优化
.reinvest-dialog-content
  padding 4px 0
  
.reinvest-info
  display flex
  justify-content space-between
  align-items center
  padding 18px 20px
  background linear-gradient(135deg, #667eea 0%, #764ba2 100%)
  border-radius 16px
  margin-bottom 24px
  box-shadow 0 8px 24px rgba(102, 126, 234, 0.35), 0 2px 8px rgba(102, 126, 234, 0.2)
  .info-label
    font-size 15px
    color rgba(255, 255, 255, 0.95)
    font-weight 500
    letter-spacing 0.5px
  .info-value
    font-size 28px
    color #fff
    font-weight 700
    font-family 'DIN Alternate', 'Helvetica Neue', Arial, sans-serif
    text-shadow 0 2px 8px rgba(0, 0, 0, 0.15)
    letter-spacing -0.5px

.quick-input-btns
  display flex
  gap 10px
  margin-top 20px
  justify-content flex-end
  .van-button
    flex 0 0 auto
    min-width 76px
    height 38px
    font-size 13px
    border-radius 19px
    transition all 0.25s cubic-bezier(0.4, 0, 0.2, 1)
    border-width 1.5px
    font-weight 500
    &:active:not(:disabled)
      transform scale(0.96)
      box-shadow 0 2px 8px rgba(102, 126, 234, 0.2)
    &:disabled
      opacity 0.4
      cursor not-allowed
    &:not(:disabled):hover
      transform translateY(-1px)
      box-shadow 0 4px 12px rgba(102, 126, 234, 0.25)

.reinvest-loading
  display flex
  justify-content center
  align-items center
  padding 20px 0
  margin-top 12px
  :deep(.van-loading__text)
    margin-left 10px
    font-size 14px
    color #667eea
    font-weight 500

.reinvest-preview
  margin-top 20px
  padding 16px 18px
  background linear-gradient(135deg, rgba(102, 126, 234, 0.08) 0%, rgba(118, 75, 162, 0.08) 100%)
  border-radius 12px
  border 1px solid rgba(102, 126, 234, 0.15)
  backdrop-filter blur(10px)
  .preview-item
    display flex
    justify-content space-between
    align-items center
    .preview-label
      font-size 14px
      color #606266
      font-weight 500
    .preview-value
      font-size 16px
      color #323233
      font-weight 600
      font-family 'DIN Alternate', 'Helvetica Neue', Arial, sans-serif
      background linear-gradient(135deg, #667eea 0%, #764ba2 100%)
      -webkit-background-clip text
      -webkit-text-fill-color transparent
      background-clip text

:deep(.van-dialog)
  border-radius 20px
  overflow hidden
  box-shadow 0 20px 60px rgba(0, 0, 0, 0.15), 0 8px 24px rgba(0, 0, 0, 0.1)

// 微信支付风格底部转账/赠送弹层
.wx-pay-sheet
  overflow hidden

.wx-pay
  padding 8px 20px calc(20px + env(safe-area-inset-bottom))
  background #fff
  color #1a1a1a

.wx-pay__handle
  width 36px
  height 4px
  margin 4px auto 12px
  border-radius 999px
  background #e5e5e5

.wx-pay__nav
  position relative
  display flex
  align-items center
  justify-content center
  margin-bottom 18px
  min-height 28px

.wx-pay__nav-title
  font-size 16px
  font-weight 600
  color #111

.wx-pay__close
  position absolute
  right 0
  top 50%
  transform translateY(-50%)
  color #999
  padding 6px
  border-radius 50%
  &:active
    background #f5f5f5

.wx-pay__to
  margin-bottom 8px

.wx-pay__to-label
  display block
  font-size 13px
  color #888
  margin-bottom 8px

.wx-pay__to-row
  display flex
  align-items center
  gap 10px
  padding 12px 14px
  background #f7f7f7
  border-radius 12px

.wx-pay__to-input
  flex 1
  border none
  outline none
  background transparent
  font-size 15px
  color #111
  min-width 0
  &::placeholder
    color #bbb

.wx-pay__scan
  color #07c160
  flex-shrink 0
  padding 4px
  &:active
    opacity 0.7

.wx-pay__amount-block
  margin 18px 0 8px
  padding-bottom 12px
  border-bottom 1px solid #f0f0f0

.wx-pay__amount-label
  font-size 13px
  color #888
  margin-bottom 10px

.wx-pay__amount-row
  display flex
  align-items baseline
  gap 6px

.wx-pay__amount-input
  flex 1
  width 100%
  border none
  outline none
  background transparent
  font-size 42px
  font-weight 600
  line-height 1.1
  color #111
  min-width 0
  font-family -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif
  &::placeholder
    color #ddd
  &::-webkit-outer-spin-button,
  &::-webkit-inner-spin-button
    -webkit-appearance none
    margin 0
  appearance textfield

.wx-pay__fee
  font-size 12px
  color #888
  line-height 1.5
  margin 4px 0 8px

.wx-pay__fee--hint
  padding 8px 10px
  border-radius 8px
  background #fff7e6
  color #b26a00

.wx-pay__warn
  display flex
  align-items flex-start
  gap 6px
  font-size 12px
  color #ee0a24
  margin 4px 0 8px
  line-height 1.5

.wx-pay__pwd
  display flex
  align-items center
  gap 10px
  margin 16px 0 20px
  padding 14px 14px
  background #f7f7f7
  border-radius 12px

.wx-pay__pwd-icon
  color #999
  flex-shrink 0

.wx-pay__pwd-input
  flex 1
  border none
  outline none
  background transparent
  font-size 15px
  color #111
  letter-spacing 2px
  &::placeholder
    color #bbb
    letter-spacing 0

.wx-pay__submit
  height 48px
  font-size 16px
  font-weight 600
  background #07c160 !important
  border none !important
  box-shadow none
  &:active
    opacity 0.88
  &--gift
    background #ff4757 !important

.wx-pay--gift
  .wx-pay__scan
    color #ff4757

// 修改密码对话框样式
.change-password-dialog-custom
  :deep(.van-dialog)
    background linear-gradient(135deg, #ffffff 0%, #f8f9fa 100%)
    border-radius 24px
    overflow visible
    box-shadow 0 24px 64px rgba(102, 126, 234, 0.25), 0 8px 24px rgba(0, 0, 0, 0.1)
  
  :deep(.van-dialog__content)
    padding 0
    overflow visible

.change-password-dialog-wrapper
  display flex
  flex-direction column
  background #fff
  border-radius 24px
  overflow hidden

.change-password-dialog-header
  position relative
  display flex
  flex-direction column
  align-items center
  padding 20px 20px 16px
  background linear-gradient(135deg, #667eea 0%, #764ba2 100%)
  .change-password-icon-wrapper
    width 56px
    height 56px
    display flex
    align-items center
    justify-content center
    background rgba(255, 255, 255, 0.2)
    border-radius 50%
    margin-bottom 8px
    backdrop-filter blur(10px)
    box-shadow 0 8px 24px rgba(0, 0, 0, 0.15)
  .change-password-title
    font-size 20px
    font-weight 700
    color #fff
    margin 0
    letter-spacing 0.5px
    text-shadow 0 2px 8px rgba(0, 0, 0, 0.1)
  .close-icon
    position absolute
    top 16px
    right 16px
    cursor pointer
    padding 4px
    border-radius 50%
    transition all 0.3s ease
    &:active
      background rgba(255, 255, 255, 0.2)
      transform scale(0.9)

.change-password-dialog-content
  padding 20px 20px 16px
  background #fff

.change-password-field-wrapper
  margin-bottom 16px
  &:last-child
    margin-bottom 0
  .field-label
    display flex
    align-items center
    gap 8px
    margin-bottom 8px
    font-size 15px
    font-weight 600
    color #2c3e50
    letter-spacing 0.3px
  .change-password-field
    :deep(.van-cell)
      padding 0
      background transparent
    :deep(.van-field__control)
      padding 14px 16px
      background linear-gradient(135deg, #f8f9fa 0%, #ffffff 100%)
      border 2px solid #e9ecef
      border-radius 14px
      font-size 15px
      color #2c3e50
      transition all 0.3s ease
      &:focus
        border-color #667eea
        background #fff
        box-shadow 0 4px 16px rgba(102, 126, 234, 0.15)
    :deep(.van-field__control::placeholder)
      color #95a5a6
      font-size 14px

.change-password-dialog-footer
  display flex
  gap 12px
  padding 16px
  background linear-gradient(to bottom, #fafbfc 0%, #ffffff 100%)
  border-top 1px solid rgba(0, 0, 0, 0.06)
  .change-password-btn
    flex 1
    height 46px
    border-radius 14px
    font-size 16px
    font-weight 600
    letter-spacing 0.5px
    transition all 0.3s ease
    border none
    &:active
      transform scale(0.97)
  .cancel-btn
    background linear-gradient(135deg, #f5f7fa 0%, #e9ecef 100%)
    color #606266
    box-shadow 0 2px 8px rgba(0, 0, 0, 0.08)
    &:active
      box-shadow 0 1px 4px rgba(0, 0, 0, 0.1)
  .confirm-btn
    background linear-gradient(135deg, #667eea 0%, #764ba2 100%)
    color #fff
    box-shadow 0 4px 16px rgba(102, 126, 234, 0.35)
    &:active
      box-shadow 0 2px 8px rgba(102, 126, 234, 0.3)

:deep(.van-dialog__content)
  padding 24px 20px 20px
  max-height 70vh
  overflow-y auto

:deep(.van-dialog__header)
  padding 24px 20px 20px
  font-size 19px
  font-weight 600
  color #2c3e50
  letter-spacing 0.3px
  border-bottom 1px solid rgba(0, 0, 0, 0.06)

.dialog-footer-custom
  display flex
  gap 12px
  padding 16px 20px
  border-top 1px solid rgba(0, 0, 0, 0.06)
  background linear-gradient(to bottom, #fafbfc 0%, #ffffff 100%)
  :deep(.van-button--default)
    background #fff
    border 1.5px solid #dcdfe6
    color #606266
    font-weight 500
    &:active
      background #f5f7fa
      border-color #c0c4cc
  :deep(.van-button--primary)
    background linear-gradient(135deg, #667eea 0%, #764ba2 100%)
    border none
    box-shadow 0 4px 12px rgba(102, 126, 234, 0.35)
    font-weight 500
    &:active
      box-shadow 0 2px 8px rgba(102, 126, 234, 0.3)
      transform scale(0.98)
  .van-button
    flex 1
    height 46px
    border-radius 23px
    font-size 16px
    transition all 0.25s cubic-bezier(0.4, 0, 0.2, 1)

:deep(.van-field)
  padding 14px 18px
  background #fff
  border-radius 14px
  margin-bottom 0
  border 1.5px solid #e4e7ed
  transition all 0.3s cubic-bezier(0.4, 0, 0.2, 1)
  box-shadow 0 2px 8px rgba(0, 0, 0, 0.04)
  &:focus-within
    border-color #667eea
    background #fff
    box-shadow 0 0 0 3px rgba(102, 126, 234, 0.12), 0 4px 12px rgba(102, 126, 234, 0.15)
    transform translateY(-1px)
  :deep(.van-field__clear-icon)
    color #909399
    font-size 16px
    transition all 0.2s ease
    &:active
      color #667eea
      transform scale(1.1)
  
// 全局 van-field 样式
:deep(.van-field__control)
  font-size 19px
  font-weight 600
  color #2c3e50
  text-align right
  letter-spacing 0.3px

:deep(.van-field__control::placeholder)
  color #c8c9cc
  font-weight normal
  font-size 15px

// 复投输入框样式覆盖（必须在全局样式之后，确保优先级）
// 使用最具体的选择器确保覆盖全局样式
.power-balance-reinvest .reinvest-input-wrapper .reinvest-input-field
  :deep(.van-field__control)
    color #fff !important
    caret-color #fff !important
    font-weight 600 !important
    font-size 16px !important
    -webkit-text-fill-color #fff !important
    &::placeholder
      color rgba(255,255,255,0.6) !important
      -webkit-text-fill-color rgba(255,255,255,0.6) !important
      opacity 1 !important
  :deep(input)
    color #fff !important
    caret-color #fff !important
    -webkit-text-fill-color #fff !important
    &::placeholder
      color rgba(255,255,255,0.6) !important
      -webkit-text-fill-color rgba(255,255,255,0.6) !important
      opacity 1 !important
  // 针对所有可能的输入元素类型
  :deep(input[type="number"])
    color #fff !important
    caret-color #fff !important
    -webkit-text-fill-color #fff !important
    &::placeholder
      color rgba(255,255,255,0.6) !important
      -webkit-text-fill-color rgba(255,255,255,0.6) !important
      opacity 1 !important
  :deep(textarea)
    color #fff !important
    caret-color #fff !important
    -webkit-text-fill-color #fff !important

.power-unit
  padding 4px 10px
  font-size 13px
  color #667eea
  font-weight 600
  background linear-gradient(135deg, rgba(102, 126, 234, 0.1) 0%, rgba(118, 75, 162, 0.1) 100%)
  border-radius 6px
  margin-left 6px
  border 1px solid rgba(102, 126, 234, 0.2)
  transition all 0.2s ease
  &:hover
    background linear-gradient(135deg, rgba(102, 126, 234, 0.15) 0%, rgba(118, 75, 162, 0.15) 100%)
    border-color rgba(102, 126, 234, 0.3)

@media (max-width: 720px)
  .invitation-page
    width 100%
    margin 0
    padding 0
    box-sizing border-box
  .hero-body
    grid-template-columns 1fr
  .avatar-block
    margin 0 auto
  .meta-header
    flex-direction column
    gap 8px
    margin-top 4px
  .meta-row
    gap 8px
  .meta-label
    padding 3px 9px
    font-size 13px
  .type-row-value
    justify-content space-between
    gap 8px
    margin-left -60px
    padding-left 60px
    width 100%
  .user-name
    font-size 20px
  .user-type
    padding 6px 14px
    gap 6px
  .user-power
    font-size 12px
    padding 3px 8px
  .layout
    padding 14px 12px 70px
    max-width 100% !important
    width 100% !important
    margin 0 !important
    box-sizing border-box
  .alipay-nav__col--share
    .lang-chip__text
      display none
  .lang-chip
    padding 0 6px
    height 24px
  // 品牌栏移动端适配
  .brand-name-text
    font-size 18px
  .brand-name-input
    font-size 16px
    padding 9px 12px
  .brand-name-btn
    padding 7px 10px 7px 12px
  .power-balance-top-grid
    grid-template-columns repeat(2, 1fr)
    gap 10px
  .power-balance-container
    gap 10px
  .power-balance-item
    padding 12px 14px
    .value
      font-size 18px
    &.power-balance-reinvest
      .reinvest-input-wrapper
        flex-direction column
        gap 8px
        .reinvest-input-field
          width 100%
        .reinvest-confirm-btn
          width 100%
          height 32px
          font-size 14px
          min-width 100%
  .card-grid
    grid-template-columns 1fr
  .team-overview-card
    padding 16px 14px 14px !important
  .team-overview-title
    font-size 18px
  .team-referrer-chip
    max-width 54%
    padding 6px 8px
    font-size 11px
  .team-metric
    padding 12px 6px 10px
    &__value
      font-size 20px
    &__label
      font-size 14px
  .team-zone
    padding 12px
    &__title
      font-size 15px
    &__total
      font-size 14px
      em
        font-size 15px
    &__col-top
      font-size 14px
    &__num
      font-size 18px
  .team-users-card
    .team-overview-title
      font-size 18px
  .alipay-nav
    margin 0 -2px 16px
    padding 10px 2px 2px
  .alipay-nav__icon
    width 44px
    height 44px
    border-radius 12px
    :deep(.van-icon)
      font-size 22px
  .alipay-nav__label
    font-size 12px
  .quick-actions-card
    padding 16px 12px 14px !important
  .quick-actions-grid
    gap 8px
  .quick-action
    padding 12px 4px 10px
    &__icon
      width 52px
      height 52px
      border-radius 15px
      :deep(.van-icon)
        font-size 24px
    &__label
      font-size 13px
  // 快捷菜单移动端适配
  .modern-grid
    :deep(.van-grid-item)
      padding 6px 4px
      flex 0 0 25% !important
      max-width 25% !important
      width 25% !important
      box-sizing border-box
    :deep(.van-grid-item__content)
      padding 12px 8px
      min-height auto
      gap 12px
      display flex
      flex-direction column
      align-items center
      justify-content center
      background transparent
      border none
      box-shadow none
    .grid-icon-wrapper
      width 55px
      height 55px
      border-radius 16px
      display flex
      align-items center
      justify-content center
      box-shadow 0 6px 20px rgba(0,0,0,0.25), 0 3px 10px rgba(0,0,0,0.18)
    .grid-text
      font-size 13px
      font-weight 500
      text-align center
      white-space nowrap
      color #2c3e50

</style>

<!-- 非scoped全局样式，强制覆盖复投输入框的文字颜色 -->
<style lang="stylus">
// 使用最高优先级的选择器强制覆盖复投输入框样式
.power-balance-reinvest .reinvest-input-wrapper .reinvest-input-field.van-cell,
.power-balance-reinvest .reinvest-input-wrapper .reinvest-input-field.van-field,
.power-balance-reinvest .reinvest-input-wrapper .reinvest-input-field
  background transparent !important
  display block !important
  visibility visible !important
  opacity 1 !important

.power-balance-reinvest .reinvest-input-wrapper .reinvest-input-field .van-cell__value,
.power-balance-reinvest .
 .reinvest-input-field .van-field__value,
.power-balance-reinvest .reinvest-input-wrapper .reinvest-input-field .van-field__wrapper
  background transparent !important

.power-balance-reinvest .reinvest-input-wrapper .reinvest-input-field .van-field__body
  background transparent !important
  border none !important
  border-radius 0 !important
  padding 0 !important
  display flex !important
  align-items center !important
  box-sizing border-box !important

.power-balance-reinvest .reinvest-input-wrapper .reinvest-input-field .van-field__control,
.power-balance-reinvest .reinvest-input-wrapper .reinvest-input-field input[type="number"],
.power-balance-reinvest .reinvest-input-wrapper .reinvest-input-field input
  color #fff !important
  caret-color #fff !important
  -webkit-text-fill-color #fff !important
  font-weight 600 !important
  font-size 14px !important
  width 100% !important
  min-width 0 !important
  overflow visible !important
  text-overflow clip !important
  white-space nowrap !important

.power-balance-reinvest .reinvest-input-wrapper .reinvest-input-field .van-field__control::placeholder,
.power-balance-reinvest .reinvest-input-wrapper .reinvest-input-field input[type="number"]::placeholder,
.power-balance-reinvest .reinvest-input-wrapper .reinvest-input-field input::placeholder
  color rgba(255,255,255,0.6) !important
  -webkit-text-fill-color rgba(255,255,255,0.6) !important
  font-size 12px !important
  opacity 1 !important

.power-balance-reinvest .reinvest-input-wrapper .reinvest-input-field .van-field__clear-icon
  color rgba(255,255,255,0.8) !important
</style>

