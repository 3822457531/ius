<template>
  <div>
    <!-- 整个页面右上角语言切换 -->
    <div class="lang-switch">
      <div class="lang-switch-inner">
        <span class="lang-label">{{ t('invitation.language') }}</span>
        <el-select
          v-model="currentLocale"
          size="small"
          @change="switchLocale"
          class="lang-select"
        >
        <el-option :label="t('invitation.languageZh')" value="zh-CN" />
        <el-option :label="t('invitation.languageEn')" value="en-US" />
        <el-option :label="t('invitation.languageJa')" value="ja" />
        <el-option :label="t('invitation.languageKr')" value="kr-KR" />
        </el-select>
      </div>
    </div>

    <div class="bg"></div>
    <div class="main">
      <div class="contain">
        <!-- <div class="logo">
        <el-image :src="logo" fit="cover"/>
        </div> -->
        <div class="header">{{ t('auth.loginTitle') }}</div>
        <div class="content">
          <div class="block">
            <el-input
              :placeholder="t('auth.loginUsernamePlaceholder')"
              size="large"
              v-model="username"
              autocomplete="off"
              autofocus
              @keyup="handleKeyup"
            >
              <template #prefix>
                <el-icon>
                  <UserFilled />
                </el-icon>
              </template>
            </el-input>
          </div>

          <div class="block">
            <el-input
              :placeholder="t('auth.loginPasswordPlaceholder')"
              size="large"
              v-model="password"
              show-password
              autocomplete="off"
              @keyup="handleKeyup"
            >
              <template #prefix>
                <el-icon>
                  <Lock />
                </el-icon>
              </template>
            </el-input>
          </div>

          <el-row class="btn-row">
            <el-button class="login-btn" size="large" type="primary" @click="login">
              {{ t('auth.loginButton') }}
            </el-button>
          </el-row>

          <el-row class="text-line" gutter="20">
            <el-button type="primary" @click="router.push('/register')" size="small" plain>
              {{ t('auth.loginRegisterButton') }}
            </el-button>
            <el-button type="success" @click="showResetPass = true" size="small" plain>
              {{ t('auth.loginResetButton') }}
            </el-button>
          </el-row>
        </div>
      </div>

      <reset-pass @hide="showResetPass = false" :show="showResetPass" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import {Lock, UserFilled} from "@element-plus/icons-vue";
import {httpGet, httpPost} from "@/utils/http";
import {ElMessage} from "element-plus";
import {useRouter} from "vue-router";

import {isMobile} from "@/utils/libs";
import {checkSession} from "@/action/session";
import {setUserToken} from "@/store/session";
import {prevRoute} from "@/router";
import ResetPass from "@/components/ResetPass.vue";

const { t, locale } = useI18n();
const currentLocale = computed({
  get: () => locale.value,
  set: (val) => {
    if (locale.value === val) return;
    locale.value = val;
    localStorage.setItem("locale", val);
  },
});
const switchLocale = (val) => {
  currentLocale.value = val;
};

const router = useRouter();
const username = ref(process.env.VUE_APP_USER);
const password = ref(process.env.VUE_APP_PASS);
const showResetPass = ref(false)
const logo = ref("/images/logo.png")

// 获取系统配置
httpGet("/api/config/get?key=system").then(res => {
  logo.value = res.data.logo
}).catch(e => {
  ElMessage.error("获取系统配置失败：" + e.message)
})


checkSession().then(() => {
  if (isMobile()) {
    router.push('/mobile')
  } else {
    router.push('/chat')
  }
}).catch(() => {
})

const handleKeyup = (e) => {
  if (e.key === 'Enter') {
    login();
  }
};

const login = function () {
  if (username.value.trim() === '') {
    return ElMessage.error(t('auth.loginErrorUsernameRequired'))
  }
  if (password.value.trim() === '') {
    return ElMessage.error(t('auth.loginErrorPasswordRequired'));
  }

  httpPost('/api/user/login', {username: username.value.trim(), password: password.value.trim()}).then((res) => {
    setUserToken(res.data)
    // 登录成功统一跳转到邀请页
    router.push('/mobile/invitation')

  }).catch((e) => {
    ElMessage.error(t('auth.loginFailPrefix') + e.message)
  })
}


</script>

<style lang="stylus" scoped>
.bg {
  position fixed
  left 0
  right 0
  top 0
  bottom 0
  background-color #14151a
  background-image url("~@/assets/img/login-bg.jpg")
  background-size cover
  background-position center
  background-repeat repeat-y
}

.bg::before {
  content ''
  position fixed
  left 0
  right 0
  top 0
  bottom 0
  background linear-gradient(to bottom, rgba(0, 0, 0, 0.35), rgba(0, 0, 0, 0.55))
  pointer-events none
}

.lang-switch {
  position fixed
  top 0
  right 0
  z-index 1000

  .lang-switch-inner {
    display flex
    align-items center
    gap 8px
    margin 10px
    padding 6px 12px
    border-radius 999px
    background rgba(0, 0, 0, 0.35)
    border 1px solid rgba(255, 255, 255, 0.25)
    box-shadow 0 4px 12px rgba(0, 0, 0, 0.35)
    backdrop-filter blur(10px)
  }

  .lang-label {
    font-size 12px
    color #f5f5f5
    letter-spacing 1px
    white-space nowrap
  }

  .lang-select {
    min-width 110px

    :deep(.el-select__wrapper) {
      background transparent
      border-radius 999px
      border none
      box-shadow none
      padding 0 4px
      color #ffffff
    }

    :deep(.el-select__caret) {
      color #ffffff
    }
  }
}

.main {
  .contain {
    position fixed
    left 50%
    top 40%
    width 90%
    max-width 400px;
    transform translate(-50%, -50%)
    padding 20px 10px;
    color #ffffff
    border-radius 10px;

    .logo {
      text-align center

      .el-image {
        width 120px;
      }
    }

    .header {
      width 100%
      margin-bottom 24px
      font-size 24px
      color $white_v1
      letter-space 2px
      text-align center
    }

    .content {
      width 100%
      height: auto
      border-radius 3px

      .block {
        margin-bottom 16px

        .el-input__inner {
          border 1px solid $gray-v6 !important

          .el-icon-user, .el-icon-lock {
            font-size 20px
          }
        }
      }

      .btn-row {
        padding-top 10px;

        .login-btn {
          width 100%
          font-size 16px
          letter-spacing 2px
        }
      }

      .text-line {
        justify-content center
        padding-top 10px;
        font-size 14px;
      }
    }
  }

  .footer {
    color #ffffff;

    .container {
      padding 20px;
    }
  }
}
</style>