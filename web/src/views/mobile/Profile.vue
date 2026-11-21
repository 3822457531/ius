<template>
  <div class="mobile-user-profile container">
    <van-nav-bar :title="title"/>

    <div class="content">


      <!-- <div class="product-list">
        <h3> 云算力套餐</h3>
        <div class="item" v-for="item in products" :key="item.id"   >
          <h3 class="title">
            <span class="tp">{{ item.name }}</span>


            <div class="buy-btn">
              
              <van-button type="success" @click="confirmJihuo(item)" size="small" >
                <span><i class="iconfont  "></i> 算力激活</span>
              </van-button>
           
            </div>
          </h3>

       
          <van-cell-group>
            
            <van-cell>
              <template #title>
                <span>投入<span style="color: red; font-weight: bold;">{{ item.price ? item.price + '$' : '10$' }}</span>算力</span>
              </template>
              <span v-if="item.sales > 0">{{ item.sales }}</span>
              <span v-else>{{ sales }}</span>
            </van-cell>

            <van-cell>
              <template #title>
                <span>产出<span style="color: red; font-weight: bold;">{{ item.price ? (item.price * 7) + '$' : '70$' }}</span>算力</span>
              </template>
              <span v-if="item.power > 0">{{ item.power }}</span>
              <span v-else>{{ power }}</span>
            </van-cell>

          </van-cell-group>
        </div>
      </div> -->

   <!--  这里添加一个自定义的算力套餐列表 ，用户输入整数类型的算力（最低1000,如果输入低于1000,则提示最少输入1000） 然后另外一行input款显示产出输入的 7倍算力，如用户输入2900,则另外一行显示产出 
     
   然后点击激活按钮调用confirmJihuoEx(item)方法      
   
        js传入的参数为
        power: 用户输入的算力
        user_id: loginUser.value.id
   -->

      <div class="product-list custom-power-list">
        <h3>自定义算力</h3>
        <div class="item">
          <h3 class="title">
            <span class="tp">自定义投入</span>
            <div class="buy-btn">
              <van-button type="success" @click="confirmJihuoEx" size="small" :disabled="!isCustomPowerValid" class="custom-power-btn">
                <span><i class="iconfont"></i> 算力激活</span>
              </van-button>
            </div>
          </h3>

          <van-cell-group>
            <van-cell class="custom-power-cell">
              <template #title>
                <span class="custom-label">投入</span>
                <span class="custom-value">{{ customPower ? (customPower / 100) + '$' : '0$' }}</span>
                <span class="custom-label">算力</span>
              </template>
              <van-field
                v-model="customPower"
                type="number"
                placeholder="1万~100万"
                :error="customPowerError"
                :error-message="customPowerErrorMsg"
                @input="validateCustomPower"
                class="custom-power-input"
              />
            </van-cell>

            <van-cell class="custom-power-cell">
              <template #title>
                <span class="custom-label">产出</span>
                <span class="custom-value output">{{ customPower && isCustomPowerValid ? ((customPower / 100) * investOutputMultiplier(customPower)) + '$' : '0$' }}</span>
                <span class="custom-label">算力</span>
              </template>
              <span class="custom-output">{{ customPower && isCustomPowerValid ? customPower * investOutputMultiplier(customPower) : 0 }}</span>
            </van-cell>
          </van-cell-group>
        </div>
      </div>

      <div class="product-list">
        <h3> 矿机</h3>
        <div class="item" v-for="item in mints" :key="item.id">
          <h3 class="title">
            <span class="tp" >{{ item.name }}</span>

           

            <div class="buy-btn">
               
                <van-button type="primary" @click="buyKj(item)" size="small" >
                  <span><i class="iconfont  "></i> 激活矿机</span>
                </van-button>
           

            </div>
 

          </h3>

          

          <van-cell-group>
  
            <van-cell>
              <template #title>
                <span>投入<span style="color: red; font-weight: bold;">{{ item.power }}</span>算力</span>
              </template>
            </van-cell>

            <van-cell>
              <template #title>
                <span>预估日产出<span style="color: #1989fa; font-weight: bold;">{{ estimateDailyOutput(item) }}</span> SOHA</span>
              </template>
              <template #label>
                <span class="mint-period-tip">当前阶段 {{ mintReleaseLabel }} 释放</span>
              </template>
            </van-cell>
          </van-cell-group>
        </div>
      </div>




     


    </div>

  

 
  </div>
</template>

<script setup>
import {onMounted, ref} from "vue";
import {  showConfirmDialog,showFailToast, showNotify, showSuccessToast,showToast, showDialog} from "vant";
import {httpGet, httpPost} from "@/utils/http";
import Compressor from 'compressorjs';
import {dateFormat} from "@/utils/libs";
import {removeUserToken} from "@/store/session";
import {ElMessage} from "element-plus";
import {checkSession} from "@/action/session";
import {useRouter} from "vue-router";

defineOptions({ name: 'Profile' })

const title = ref('产品中心')
const form = ref({
  username: '',
  nickname: '',
  mobile: '',
  avatar: '',
  calls: 0,
  tokens: 0
})
const fileList = ref([
  {
    url: '',
    message: '上传中...',
  }
]);

const products = ref([])
const vipMonthPower = ref(0)
//const payWays = ref({})
const mints = ref([])
const mintReleaseRates = ref({
  kj_zj_plus: 0,
  kj_jj_plus: 0,
  kj_san_plus: 0,
  kj_si_plus: 0,
  kj_other_plus: 0,
})
const mintReleaseLabel = ref('0-10天')
const router = useRouter()
const loginUser = ref(null)

// 添加自定义算力所需的变量
const customPower = ref('')
const customPowerError = ref(false)
const customPowerErrorMsg = ref('')
const isCustomPowerValid = ref(false)
const jihuoExMinPower = ref(1000)
const memberNormalMaxPower = ref(9999)
const memberVipMinPower = ref(10000)
const allowMint = ref(false)
const investMultiplier = ref({
  tier1_min: 1000,
  tier1_max: 49999,
  tier1_mul: 5,
  tier2_min: 50000,
  tier2_max: 99999,
  tier2_mul: 6,
  tier3_min: 100000,
  tier3_mul: 7,
})

const investOutputMultiplier = (sales) => {
  const n = Number(sales)
  if (!n || n <= 0) return investMultiplier.value.tier1_mul || 5
  const m = investMultiplier.value
  if (n >= (m.tier3_min || 100000)) return m.tier3_mul || 7
  if (n >= (m.tier2_min || 50000) && n <= (m.tier2_max || 99999)) return m.tier2_mul || 6
  if (n >= (m.tier1_min || 1000) && n <= (m.tier1_max || 49999)) return m.tier1_mul || 5
  if (n > (m.tier1_max || 49999) && n < (m.tier2_min || 50000)) return m.tier2_mul || 6
  if (n > (m.tier2_max || 99999) && n < (m.tier3_min || 100000)) return m.tier3_mul || 7
  return m.tier1_mul || 5
}

const refreshProfile = () => {
  return httpGet('/api/user/profile').then(res => {
    form.value = res.data
    if (fileList.value[0]) fileList.value[0].url = form.value.avatar
    if (res.data?.jihuo_ex_min_power > 0) jihuoExMinPower.value = Number(res.data.jihuo_ex_min_power)
    if (res.data?.member_normal_max_power > 0) memberNormalMaxPower.value = Number(res.data.member_normal_max_power)
    if (res.data?.member_vip_min_power > 0) memberVipMinPower.value = Number(res.data.member_vip_min_power)
    allowMint.value = !!res.data?.allow_mint
  }).catch((e) => {
    console.log(e.message)
    showFailToast('获取用户信息失败')
  })
}

const refreshProductLists = () => {
  httpGet("/api/product/list").then((res) => {
    products.value = res.data
  }).catch(e => {
    showFailToast("获取产品套餐失败：" + e.message)
  })
  httpGet("/api/mint/list").then((res) => {
    mints.value = res.data
  }).catch(e => {
    showFailToast("获取矿机列表失败：" + e.message)
  })
}

onMounted(() => {
  checkSession().then(user => {
    loginUser.value = user
    refreshProfile()
    refreshProductLists()

    httpGet("/api/mint/releaseConfig").then((res) => {
      if (res.data) {
        mintReleaseRates.value = res.data
      }
    }).catch(() => {})

    httpGet("/api/config/get?key=system").then(res => {
      const min = Number(res.data?.jihuo_ex_min_power)
      if (min > 0) jihuoExMinPower.value = min
      const normalMax = Number(res.data?.member_normal_max_power)
      if (normalMax > 0) memberNormalMaxPower.value = normalMax
      const vipMin = Number(res.data?.member_vip_min_power)
      if (vipMin > 0) memberVipMinPower.value = vipMin
      const im = res.data?.invest_multiplier
      if (im && typeof im === 'object') {
        investMultiplier.value = {
          ...investMultiplier.value,
          ...im,
        }
      }
    }).catch(() => {})
  }).catch(() => {
    router.push("/login")
  })
})


//  是否确认激活
const usdtJh = (item) => {
  showConfirmDialog({
    title: 'usdt激活',
    message:
        '确定用usdt激活算力?',
  }).then(() => {

    //提现通道暂未开启

    showToast("usdt激活目前已关闭,敬请期待...")

  }).catch(() => {

  //  showToast("您取消了激活")

  });
}


const afterRead = (file) => {
  file.status = 'uploading';
  file.message = '上传中...';
  // 压缩图片并上传
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


// const transferDialog = ref(false)
// const transfer = ref({
//   uid: "",
//   power: "",
// })


const beforeClose = (action) => {
  new Promise((resolve) => {
    resolve(action === 'confirm');
  });
}


// 提交修改密码
const updatePass = () => {
  if (pass.value.old === '') {
    return showNotify({type: "danger", message: "请输入旧密码"})
  }
  if (!pass.value.new || pass.value.new.length < 8) {
    return showNotify({type: "danger", message: "密码的长度为8-16个字符"})
  }
  if (pass.value.renew !== pass.value.new) {
    return showNotify({type: "danger", message: "两次输入密码不一致"})
  }
  httpPost('/api/user/password', {
    old_pass: pass.value.old,
    password: pass.value.new,
    repass: pass.value.renew
  }).then(() => {
    showSuccessToast("更新成功！")
    showPasswordDialog.value = false
  }).catch((e) => {
    showFailToast('更新失败，' + e.message)
    showPasswordDialog.value = false
  })
}




const confirmJihuo = (item) => {
  showConfirmDialog({
    title: '是否激活',
    message:
        '确定激活吗?',
  }).then(() => {

        httpPost("/api/payment/confirmJihuo", {
        product_id: item.id,
        user_id: loginUser.value.id
      }).then(() => {
        showSuccessToast("激活成功....")
        refreshProfile()
        refreshProductLists()
      }).catch(e => {
        showFailToast("激活失败：" + e.message)
      })
  }).catch(() => {

    //showToast("您取消了激活")

  });
}

const estimateDailyOutput = (item) => {
  const power = Number(item?.power) || 0
  const rates = mintReleaseRates.value || {}
  let rate = Number(rates.kj_zj_plus) || 0
  if (rate <= 0) {
    rate = Number(item?.fee) || 0
  }
  if (rate <= 0 || power <= 0) return '0'
  return (power * rate).toFixed(4)
}

const buyKj = (item) => {
  if (!allowMint.value) {
    showDialog({ title: '提示', message: '已抢购完毕' })
    return
  }
  showConfirmDialog({
    title: '是否购买',
    message:
        '确定购买吗?',
  }).then(() => {


    //showToast("矿机套餐暂未对外开放,敬请期待...")
    //激活框架暂未开启

      httpPost("/api/mintOrder/buyKj", {
        mint_id: item.id,
        user_id: loginUser.value.id
      }).then(() => {
        showSuccessToast("购买矿机成功....")
        refreshProfile()
        refreshProductLists()
      }).catch(e => {
        showFailToast("购买失败：" + e.message)
      })




  }).catch(() => {

   // showToast("您取消了购买")

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

// 验证自定义算力输入
const validateCustomPower = () => {
  // 转换为数字类型进行比较
  const powerValue = Number(customPower.value)
  
  if (!customPower.value) {
    customPowerError.value = true
    customPowerErrorMsg.value = '请输入算力值'
    isCustomPowerValid.value = false
    return
  }
  
  if (isNaN(powerValue) || !Number.isInteger(powerValue)) {
    customPowerError.value = true
    customPowerErrorMsg.value = '请输入整数'
    isCustomPowerValid.value = false
    return
  }
  
  if (powerValue < jihuoExMinPower.value) {
    customPowerError.value = true
    customPowerErrorMsg.value = `最小算力为${jihuoExMinPower.value}`
    isCustomPowerValid.value = false
    return
  }

  if (powerValue > memberNormalMaxPower.value && powerValue < memberVipMinPower.value) {
    customPowerError.value = true
    customPowerErrorMsg.value = `普通会员单笔${jihuoExMinPower.value}~${memberNormalMaxPower.value}，VIP需≥${memberVipMinPower.value}`
    isCustomPowerValid.value = false
    return
  }
  
  if (powerValue > 1000000) {
    customPowerError.value = true
    customPowerErrorMsg.value = '最大算力为100万'
    isCustomPowerValid.value = false
    return
  }
  
  // 输入有效
  customPowerError.value = false
  customPowerErrorMsg.value = ''
  isCustomPowerValid.value = true
}

// 确认激活自定义算力
const confirmJihuoEx = () => {
  if (!isCustomPowerValid.value) {
    showToast(`请输入有效算力（${jihuoExMinPower.value}~${memberNormalMaxPower.value} 或 ≥${memberVipMinPower.value}）`)
    return
  }
  
  showConfirmDialog({
    title: '自定义算力激活',
    message: `确定激活${customPower.value}算力吗?`,
  }).then(() => {
    httpPost("/api/payment/confirmJihuoEx", {
      power: Number(customPower.value),
      user_id: loginUser.value.id
    }).then(() => {
      showSuccessToast("激活成功....")
      customPower.value = ''
      isCustomPowerValid.value = false
      refreshProfile()
    }).catch(e => {
      showFailToast("激活失败：" + e.message)
    })
  }).catch(() => {
    // 用户取消操作
  });
}

</script>

<style lang="stylus">
.mobile-user-profile {
  .content {
    .van-field__label {
      width 100px
      text-align right
    }

    .modify-pass {
      padding 10px 15px
    }

    .product-list {
      padding 0 15px

      .item {
        border 1px solid #e5e5e5
        border-radius 10px
        margin-bottom 15px
        overflow hidden

        .title {
          padding 0 12px
          position relative

          .buy-btn {
            position absolute
            top -5px
            right 10px

            .van-button {
              font-size 14px
              margin-left 10px
            }
          }
        }

        .price {
          font-size 18px
          color #f56c6c
          padding 38px
        }

        .tp {
          font-size 18px
          color #f56c6c
        }

        .mint-period-tip {
          font-size 12px
          color #999
        }
      }
    }
  }
}

.custom-power-list {
  .item {
    background: #fafbfc;
    box-shadow: 0 2px 8px rgba(0,0,0,0.04);
    border-radius: 14px;
    padding: 18px 14px 10px 14px;
    .title {
      .tp {
        font-size: 20px;
        font-weight: 600;
        color: #2d8cf0;
      }
      .buy-btn {
        .custom-power-btn {
          font-size: 16px;
          padding: 0 18px;
          border-radius: 8px;
        }
      }
    }
    .custom-power-cell {
      margin-bottom: 10px;
      .custom-label {
        font-size: 16px;
        color: #888;
        margin-right: 4px;
      }
      .custom-value {
        font-size: 22px;
        font-weight: bold;
        color: #f56c6c;
        margin: 0 4px;
      }
      .custom-value.output {
        color: #19be6b;
      }
      .custom-output {
        font-size: 20px;
        font-weight: 600;
        color: #19be6b;
        margin-left: 8px;
      }
      .custom-power-input {
        margin-top: 8px;
        font-size: 20px;
        border-radius: 8px;
        box-shadow: 0 1px 4px rgba(45,140,240,0.08);
        padding: 8px 12px;
        background: #fff;
      }
    }
  }
}
</style>