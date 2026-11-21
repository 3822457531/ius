<template>
  <div class="custom-scroll">
    <div class="page-invitation">
      <div class="inner">
        <!-- <h2>支付确认</h2> -->



        <div class="share-box">

          <div class="invite-url">
            <h2>产品名称 </h2>  
            <h2>{{ $route.params.name }} </h2>  
          </div>


          <div class="invite-url">
            <h2>支付金额  </h2>
            <h2>{{ $route.params.price }} $</h2>  
          </div>
<!-- 
          <div class="invite-url">
            <h4>BSC-USDT转账地址  </h4>
            <h4>  </h4>
          </div> -->
          <div class="invite-qrcode">
            <h2>二维码转账  </h2> <el-image :src="usdtimg"/>
          </div>

<!-- 
          <div class="logo">
          <el-image :src="logo" fit="cover"/>
          </div> -->

          <!-- <div class="invite-qrcode">
            
            <h2>BSC-USDT转账地址  </h2> 
          </div> -->

            <span>{{address}} </span>


          <div class="invite-url">
            <el-button type="primary" plain class="copy-code" :data-clipboard-text="address">复制地址</el-button>
          </div>


        </div>

        <div class="invite-stats">
        

        </div>


        <div class="info">
           
          </div>
          <div class="info">
           
          </div>
          <div class="info">
           
          </div>


        <div class="modify-pass">
         <van-button round block type="success" @click="paySuccess($route.params.id)" >我已付款</van-button>
        </div>



        <div class="content">  
         </div>
          <h2></h2>

          <h2></h2>
          <div class="info">
           
          </div>

        
        
      </div>
    </div>
   

  </div>
</template>

<script setup>
import {onMounted, ref} from "vue"
import {  showConfirmDialog,showFailToast, showNotify, showSuccessToast,showToast} from "vant";
import QRCode from "qrcode";
import {httpGet, httpPost} from "@/utils/http";
import {ElMessage} from "element-plus";
import Clipboard from "clipboard";
import {checkSession} from "@/action/session";
import {useRouter} from "vue-router";

const inviteURL = ref("")
const qrImg = ref("")
const inviteChatCalls = ref(0)
const inviteImgCalls = ref(0)
const hits = ref(0)
const regNum = ref(0)
const rate = ref(0)
const isLogin = ref(false)
const showLoginDialog = ref(false)
const router = useRouter()
const loginUser = ref(null)
const usdtimg = ref("/images/usdtimg.png")
const address = ref("")


const fileList = ref([
  {
    url: '',
    message: '上传中...',
  }
]);



const form = ref({
  username: '',
  nickname: '',
  mobile: '',
  avatar: '',
  calls: 0,
  tokens: 0
})

const userS = ref({
  zzcode:''
})





onMounted(() => {
  initData()
  

  // 复制链接



  const clipboard2 = new Clipboard('.copy-code');
  clipboard2.on('success', () => {
    ElMessage.success('复制转账地址成功！');
  })


  clipboard2.on('error', () => {
    ElMessage.error('复制失败！');
  })


  httpGet("/api/invite/code").then(res => {
      const text = `${location.protocol}//${location.host}/register?invite_code=${res.data.code}`
      hits.value = res.data["hits"]
      regNum.value = res.data["reg_num"]
      userS.value.zzcode=res.data.code;
          //赋值
      if (hits.value > 0) {
        rate.value = ((regNum.value / hits.value) * 100).toFixed(2)
      }
      QRCode.toDataURL(text, {width: 400, height: 400, margin: 2}, (error, url) => {
        if (error) {
          console.error(error)
        } else {
          qrImg.value = url;
        
        }
      });
      inviteURL.value = text
    }).catch(e => {
      ElMessage.error("获取邀请码失败：" + e.message)
    })



})

const initData = () => {
  checkSession().then(() => {
    isLogin.value = true
    


    httpGet('/api/user/profile').then(res => {
      form.value = res.data
      fileList.value[0].url = form.value.avatar
    }).catch((e) => {
      console.log(e.message)
      showFailToast('获取用户信息失败')
    });



    httpGet("/api/invite/code").then(res => {
      const text = `${location.protocol}//${location.host}/register?invite_code=${res.data.code}`
      hits.value = res.data["hits"]
      regNum.value = res.data["reg_num"]
      form.value.userS=res.data.code;
          //赋值
      if (hits.value > 0) {
        rate.value = ((regNum.value / hits.value) * 100).toFixed(2)
      }
      QRCode.toDataURL(text, {width: 400, height: 400, margin: 2}, (error, url) => {
        if (error) {
          console.error(error)
        } else {
          qrImg.value = url;
        
        }
      });
      inviteURL.value = text
    }).catch(e => {
      ElMessage.error("获取邀请码失败：" + e.message)
    })

 
  }).catch(() => {
    showLoginDialog.value = true
  });
}

// 获取系统配置
httpGet("/api/config/get?key=system").then(res => {
  usdtimg.value = res.data.usdtimg
  address.value = res.data.address
}).catch(e => {
 // ElMessage.error("获取系统配置失败：" + e.message)
})







//确认支付
const paySuccess = (item) => {
  showConfirmDialog({
    title: '已付款确认',
    message:
        '你确认已经付款了吗?如果未付款请不要点击此按钮?',
  }).then(() => {

    console.log(item);

        httpPost("/api/payment/paySuccess", {
          product_id: item
      }).then(res => {
        showSuccessToast("支付状态提交成功等待管理员审核....")
         setTimeout(() => history.go(-1), 3000)
      }).catch(e => {
        showFailToast("激活失败：" + e.message)
       // location.reload()
        // setTimeout(() => location.reload(), 3000)
      })
  }).catch(() => {

    //showToast("您取消了激活")

  });
}




// const copyTs = () => {

//   const clipboard2 = new Clipboard('.copy-code');
//   clipboard2.on('success', () => {
//     ElMessage.success('复制邀请码成功！');
//   })
// }

</script>

<style lang="stylus" scoped>
@import "@/assets/css/custom-scroll.styl"
.page-invitation {
  display: flex;
  justify-content: center;
  background-color: #282c34;
  overflow-x hidden
  overflow-y visible


  

  .inner {
    display flex
    flex-flow column
    max-width 1000px
    width 100%
    color #e1e1e1

    h2 {
      color #ffffff;
    }

    .share-box {
      .info {
        line-height 1.5
        border 1px solid #444444
        border-radius 10px
        padding 10px

        strong {
          color #f56c6c
        }
      }

      .invite-qrcode {
        padding 50px
        text-align center
      }

      .invite-url {
        padding 15px
        display flex
        justify-content space-between
        border 1px solid #444444
        border-radius 10px

        span {
          position relative
          font-family 'Microsoft YaHei', '微软雅黑', Arial, sans-serif
          top 5px
        }
      }
    }

    .invite-stats {
      padding 30px 10px

      .item-box {
        border-radius 10px
        padding 0 10px

        .el-col {
          height 140px
          display flex
          align-items center
          justify-content center

          .iconfont {
            font-size 60px
          }

          .item-info {
            font-size 18px

            .text, .num {
              padding 3px 0
              text-align center
            }

            .num {
              font-size 40px
            }
          }
        }
      }

      .yellow {
        background-color #ffeecc
        color #D68F00
      }

      .blue {
        background-color #D6E4FF
        color #1062FE
      }

      .green {
        background-color #E7F8EB
        color #2D9F46
      }
    }


    .invite-logs {
      padding-bottom 20px
    }
  }

}
</style>
