<template>
  <div class="mobile-user-profile container">
    <van-nav-bar :title="title"/>

    <div class="content">
      <van-form>
        <van-cell-group inset v-model="form">
          <van-field
              v-model="form.username"
              name="账号"
              label="账号"
              readonly
              disabled
          />
          <van-field label="头像">
            <template #input>
              <van-uploader v-model="fileList"
                            reupload max-count="1"
                            :deletable="false"
                            :after-read="afterRead"/>
            </template>
          </van-field>

          <van-field label="剩余算力">
            <template #input>
              <van-tag type="primary">{{ form.power }}</van-tag>
            </template>
          </van-field>

          <van-field label="转账">
             <template #input>
              <van-button  @click="transferDialog = true"> <span><i class="iconfont  "></i> 转账</span></van-button>
             </template>
           </van-field>

          <van-field label="推广">
            <template #input>
              <van-button to="/mobile/invitation" name="invitation" icon="share-o"></van-button> 
            </template>

          </van-field>

          <!-- <van-field label="VIP到期时间" v-if="form.expired_time > 0">
            <template #input>
              <van-tag type="warning">{{ dateFormat(form.expired_time) }}</van-tag>
            </template>
          </van-field> -->

        </van-cell-group>
      </van-form>
<!-- 

      <div class="modify-pass">
         <van-button to="/mobile/invitation" name="invitation" icon="share-o">分享</van-button> 
        <van-tabbar-item to="/mobile/invitation" name="invitation" icon="share-o">分享</van-tabbar-item>
      </div> -->



      <div class="modify-pass">
        <van-button round block type="primary" @click="showPasswordDialog = true" >修改密码</van-button>
      </div>

      
      <div class="modify-pass">
        <van-button round block type="danger" @click="logout">退出登录</van-button>
      </div>
      

    </div>

    <van-dialog v-model:show="showPasswordDialog" title="修改密码" show-cancel-button
                @confirm="updatePass"
                @cancel="showPasswordDialog = false"
                :before-close="beforeClose">
      <van-form>
        <van-cell-group inset>
          <van-field
              v-model="pass.old"
              placeholder="旧密码"
          />
          <van-field
              v-model="pass.new"
              type="password"
              placeholder="新密码"
          />
          <van-field
              v-model="pass.renew"
              type="password"
              placeholder="确认密码"
          />
        </van-cell-group>
      </van-form>
    </van-dialog>


    <van-dialog v-model:show="transferDialog" title="转账" show-cancel-button
                @confirm="updateTransfer"
                @cancel="transferDialog = false "
                :before-close="beforeClose">
      <van-form>
        <van-cell-group inset>
          <van-field
              v-model="transfer.uid"
              placeholder="收款人推荐码"
          />
          <van-field
              v-model="transfer.power"
              placeholder="算力值"
          />
       
        </van-cell-group>
      </van-form>
    </van-dialog>
  </div>
</template>

<script setup>
import {onMounted, ref} from "vue";
import {showFailToast, showNotify, showSuccessToast, showDialog} from "vant";
import {httpGet, httpPost} from "@/utils/http";
import Compressor from 'compressorjs';
import {dateFormat} from "@/utils/libs";
import {removeUserToken} from "@/store/session";
import {ElMessage} from "element-plus";
import {checkSession} from "@/action/session";
import {useRouter} from "vue-router";

const title = ref('用户设置')
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
const router = useRouter()
const loginUser = ref(null)
const allowMint = ref(false)

onMounted(() => {
  checkSession().then(user => {
    loginUser.value = user
    httpGet('/api/user/profile').then(res => {
      form.value = res.data
      fileList.value[0].url = form.value.avatar
      allowMint.value = !!res.data?.allow_mint
    }).catch((e) => {
      console.log(e.message)
      showFailToast('获取用户信息失败')
    });

    //获取产品列表
    // httpGet("/api/product/list").then((res) => {
    //   products.value = res.data
    // }).catch(e => {
    //   showFailToast("获取产品套餐失败：" + e.message)
    // })

  


    // httpGet("/api/config/get?key=system").then(res => {
    //   vipMonthPower.value = res.data['vip_month_power']
    // }).catch(e => {
    //   showFailToast("获取系统配置失败：" + e.message)
    // })


        // 获取产品列表
    httpGet("/api/product/list").then((res) => {
      products.value = res.data
    }).catch(e => {
      showFailToast("获取产品套餐失败：" + e.message)
    })
    
  // 获取矿机列表
    httpGet("/api/mint/list").then((res) => {
      mints.value = res.data
    }).catch(e => {
      showFailToast("获取矿机列表失败：" + e.message)
    })


  }).catch(() => {
    router.push("/login")
  })

})

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


const transferDialog = ref(false)
const transfer = ref({
  uid: "",
  power: "",
})


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

// 提交转账
const updateTransfer = () => {

  if (transfer.value.uid === '') {
    return showNotify({type: "danger", message: "请输入转账人id"})
  }
  if (transfer.value.power === '') {
    return showNotify({type: "danger", message: "请输入算力值"})
  }
  httpPost('/api/user/transfer', {
    uid: transfer.value.uid,
    power: transfer.value.power,
  }).then(() => {
    showSuccessToast("转账成功")
    transferDialog.value = false
  }).catch((e) => {
    showFailToast('转账失败，' + e.message)
    transferDialog.value = false
  })

 // showSuccessToast("转账模拟成功")
}




const confirmJihuo = (item) => {
  httpPost("/api/payment/confirmJihuo", {
    product_id: item.id,
    user_id: loginUser.value.id
  }).then(res => {
    // console.log(res.data)
   // location.href = res.data
  //  alert("模拟支付测试")
    showSuccessToast("激活套餐成功....")
  }).catch(e => {
    showFailToast("激活套餐失败：" + e.message)
  })
}

//购买矿机
const buyKj = (item) => {
  if (!allowMint.value) {
    showDialog({ title: '提示', message: '已抢购完毕' })
    return
  }
  httpPost("/api/mintOrder/buyKj", {
    mint_id: item.id,
    user_id: loginUser.value.id
  }).then(res => {

    showSuccessToast("购买矿机成功....")
  }).catch(e => {
    showFailToast("激活套餐失败：" + e.message)
  })
}




const logout = function () {
  httpGet('/api/user/logout').then(() => {
    removeUserToken();
    router.push('/login');
  }).catch(() => {
    ElMessage.error('注销失败！');
  })
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
        }
      }
    }
  }
}
</style>