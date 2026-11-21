<template>
  <div>
    <div class="bg"></div>
    <div class="register-page">
      <div class="page-inner">
        <div class="contain" v-if="enableRegister">
          <!-- <div class="logo">
            <el-image :src="logo" fit="cover"/>
          </div> -->

          <div class="header">{{ title }}</div>
          <div class="content">
            <el-form :model="formData" label-width="120px" ref="formRef">
              <!-- 用户名输入框 -->
              <div class="block">
                <el-input
                  :placeholder="t('auth.registerUsernamePlaceholder')"
                  size="large"
                  maxlength="10"
                  show-word-limit
                  v-model="formData.username"
                  autocomplete="off"
                >
                  <template #prefix>
                    <el-icon>
                      <Iphone/>
                    </el-icon>
                  </template>
                </el-input>
              </div>

              <!-- 密码输入框 -->
              <div class="block">
                <el-input
                  :placeholder="t('auth.registerPasswordPlaceholder')"
                  maxlength="16"
                  size="large"
                  v-model="formData.password"
                  show-password
                  autocomplete="off"
                >
                  <template #prefix>
                    <el-icon>
                      <Lock/>
                    </el-icon>
                  </template>
                </el-input>
              </div>

              <!-- 重复密码输入框 -->
              <div class="block">
                <el-input
                  :placeholder="t('auth.registerRePasswordPlaceholder')"
                  size="large"
                  maxlength="16"
                  v-model="formData.repass"
                  show-password
                  autocomplete="off"
                >
                  <template #prefix>
                    <el-icon>
                      <Lock/>
                    </el-icon>
                  </template>
                </el-input>
              </div>


               <!-- 邮箱输入框 -->
              <div class="block">
                <el-input
                  :placeholder="t('auth.registerEmailPlaceholder')"
                  size="large"
                  v-model="formData.mail"
                  autocomplete="off"
                >
                  <template #prefix>
                    <el-icon>
                      <Check/>
                    </el-icon>
                  </template>
                </el-input>
              </div>

          

              <!-- 邀请码输入框 -->
              <div class="block">
                <el-input
                  :placeholder="t('auth.registerInvitePlaceholder')"
                  size="large"
                  v-model="formData.invite_code"
                  autocomplete="off"
                >
                  <template #prefix>
                    <el-icon>
                      <Message/>
                    </el-icon>
                  </template>
                </el-input>
              </div>

              <!-- 注册按钮 -->
              <el-row class="btn-row">
                <el-button class="login-btn" size="large" type="primary" @click="loadCaptcha">
                  {{ t('auth.registerButton') }}
                </el-button>
              </el-row>

              <!-- 人机验证弹窗 -->
              <el-dialog
                  v-model="showCaptcha"
                  :close-on-click-modal="true"
                  :show-close="false"
                  style="width: 360px;"
               >
                <slide-captcha
                    :bg-img="bgImg"
                    :bk-img="bkImg"
                    :result="result"
                    @refresh="getSlideCaptcha"
                    @confirm="handleSlideConfirm"
                    @hide="showCaptcha = false"/>
            
            </el-dialog>

              <el-row class="text-line">
                {{ t('auth.registerHasAccount') }}
                <el-link type="primary" @click="router.push('/login')">{{ t('auth.registerLoginLink') }}</el-link>
              </el-row>
            </el-form>
          </div>
        </div>

        <!-- 注册功能关闭提示 -->
        <div class="tip-result" v-else>
          <el-result icon="error" :title="t('auth.registerClosedTitle')">
            <template #sub-title>
              <p>{{ t('auth.registerClosedDesc') }}</p>
              <div class="wechat-card">
                <el-image :src="wxImg"/>
              </div>
            </template>
          </el-result>
        </div>

        <footer class="footer">
          <footer-bar/>
        </footer>
      </div>
    </div>
  </div>
</template>

<script setup>

import { ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import {Check, Checked, Iphone, Lock, Message} from "@element-plus/icons-vue";
import {validateEmail} from "@/utils/validate";
import {httpGet, httpPost} from "@/utils/http";
import {ElMessage} from "element-plus";
import {useRouter} from "vue-router";
import FooterBar from "@/components/FooterBar.vue";
import {arrayContains} from "@/utils/libs";
import {setUserToken} from "@/store/session";
import CaptchaPlus from "@/components/CaptchaPlus.vue";
import SlideCaptcha from "@/components/SlideCaptcha.vue";

const { t } = useI18n();

const router = useRouter();
const title = computed(() => t('auth.registerTitle'));
const formData = ref({
  username: '',
  password: '',
  code: '',
  repass: '',
  invite_code: router.currentRoute.value.query['invite_code'],
})
const formRef = ref(null)
const enableMobile = ref(false)
const enableEmail = ref(false)
const enableRegister = ref(true)
const wxImg = ref("/images/wx.png")
const logo = ref("/images/logo.png")


const canSend = ref(true)
const showCaptcha = ref(false)
const maxDot = ref(5)
const imageBase64 = ref('')
const thumbBase64 = ref('')
const captKey = ref('')
const dots = ref(null)

httpGet("/api/config/get?key=system").then(res => {
  if (res.data) {
    const registerWays = res.data['register_ways']
    if (arrayContains(registerWays, "mobile")) {
      enableMobile.value = true
    }
    if (arrayContains(registerWays, "email")) {
      enableEmail.value = true
    }
    // 是否启用注册
    enableRegister.value = res.data['enabled_register']
    // 覆盖微信二维码
    if (res.data['wechat_card_url'] !== '') {
      wxImg.value = res.data['wechat_card_url']
    }
    logo.value = res.data.logo
  }
}).catch(e => {
  ElMessage.error("获取系统配置失败：" + e.message)
})

httpGet("/api/invite/hits", {code: formData.value.invite_code}).then(() => {
}).catch(() => {
})


const register = function () {
  if (formData.value.username === '') {
    return ElMessage.error(t('auth.registerErrorUsernameRequired'));
  }

  if (Array.from((formData.value.username ?? '').trim()).length > 10) {
    return ElMessage.error(t('auth.registerErrorUsernameMax'));
  }

  if (formData.value.password.length < 6) {
    return ElMessage.error(t('auth.registerErrorPasswordLen'));
  }
  if (formData.value.repass !== formData.value.password) {
    return ElMessage.error(t('auth.registerErrorPasswordNotSame'));
  }

  if ((enableEmail.value || enableMobile.value) && formData.value.code === '') {
    return ElMessage.error(t('auth.registerErrorCodeRequired'));
  }

  console.log(formData.value.invite_code);

  console.log("test")

  //校验邮箱地址
  // if (!validateMobile(props.receiver) && !validateEmail(props.receiver)) {
  //   return showMessageError("请输入合法的手机号/邮箱地址")
  // }

  if (!validateEmail(formData.value.mail)) {
    return ElMessage.error(t('auth.registerErrorEmailInvalid'));
  }


  if (!String(formData.value.invite_code || '').trim()) {
    return ElMessage.error(t('auth.registerErrorInviteRequired'));
  }


  httpPost('/api/user/register', formData.value).then((res) => {
    setUserToken(res.data)
    ElMessage.success({
      "message": t('auth.registerSuccessMessage'),
      onClose: () => router.push("/mobile/invitation"),
      duration: 1000
    })
  }).catch((e) => {
    ElMessage.error(t('auth.registerFailPrefix') + e.message)
  })
}


//人机校验
const handleConfirm = (dots) => {
  if (lodash.size(dots) <= 0) {
    return ElMessage.error(t('auth.captchaErrorRequired'));
  }

  let dotArr = [];
  lodash.forEach(dots, (dot) => {
    dotArr.push(dot.x, dot.y);
  });

  dots.value = dotArr.join(',');

  httpPost('/api/captcha/check', {
    dots: dots.value,
    key: captKey.value
  }).then(() => {
    showCaptcha.value = false;
    // 调用注册方法
    register();
  }).catch(() => {
    ElMessage.error(t('auth.captchaVerifyFail'));
    handleRequestCaptCode();
  });
}


const loadCaptcha = () => {

register();
 
 /* 

  showCaptcha.value = true
 
    getSlideCaptcha() */
 
}



// 滑动验证码
const bgImg = ref('')
const bkImg = ref('')
const result = ref(0)

const getSlideCaptcha = () => {
  result.value = 0
  httpGet("/api/captcha/slide/get").then(res => {
    bkImg.value = res.data.bkImg
    bgImg.value = res.data.bgImg
    captKey.value = res.data.key
  }).catch(e => {
    showMessageError(t('auth.captchaDataFailPrefix') + e.message)
  })
}

const handleSlideConfirm = (x) => {
  httpPost("/api/captcha/slide/check", {
    key: captKey.value,
    x: x
  }).then(() => {
    result.value = 1
    showCaptcha.value = false
    console.log("注册滑块验证成功");
    
    register();
    //sendMsg()

  }).catch(() => {
    result.value = 2
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
  background-color #091519
  background-image url("~@/assets/img/reg-bg.jpg")
  background-size cover
  background-position center
  background-repeat no-repeat
}

.bg::before {
  content ''
  position fixed
  left 0
  right 0
  top 0
  bottom 0
  background linear-gradient(to bottom, rgba(0, 0, 0, 0.5), rgba(0, 0, 0, 0.7))
  pointer-events none
}

.register-page {
  display flex
  justify-content center

  .page-inner {
    max-width 450px
    min-width 360px
    height 100vh
    display flex
    justify-content center
    align-items center

    .contain {
      padding 0 40px 20px 40px;
      width 100%
      color #ffffff
      border-radius 10px;
      z-index 10
      background-color rgba(255, 255, 255, 0.3)

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


    .tip-result {
      z-index 10

      .wechat-card {
        padding 20px
      }
    }

    .footer {
      color #ffffff;

      .container {
        padding 20px;
      }
    }

  }

}
</style>

<style lang="stylus">
.register-page {
  .el-result {

    border-radius 10px;
    background-color rgba(14, 25, 30, 0.6)
    border 1px solid #666

    .el-result__title p {
      color #ffffff
    }

    .el-result__subtitle p {
      color #c1c1c1
    }
  }
}
</style>