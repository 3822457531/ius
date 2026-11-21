<template>
  <div class="mobile-user-profile container">
    <van-nav-bar :title="title"/>

    <div class="content">


      <div class="product-list">
        <h3  v-if="haveOrder" >已结束算力</h3>
        <div v-else class="no-data">
          <van-empty 
                  image="https://fastly.jsdelivr.net/npm/@vant/assets/custom-empty-image.png"
                  image-size="80"
                  description="暂无记录"
          />
        </div>

        <div class="item" v-for="item in orders" :key="item.id">
          <h3 class="title">
            <span class="tpend">{{ item.subject }} </span>

               <!-- <span>
                {{ item.power }}
               </span>

               <span >
                {{ item.js_power }}
               </span> -->

          </h3>

          <van-cell-group>
            <!-- <van-cell title="商品价格">
          
            
            </van-cell> -->
            <!-- <van-cell title="有效期">
              <span v-if="item.days > 0">{{ item.days }}天</span>
              <van-tag type="primary" v-else>长期有效</van-tag>
            </van-cell> -->
            <!-- <van-cell title="当前预产算力值">
              <span v-if="item.power > 0">{{ item.power }}</span>
              <span v-else>0</span>
            </van-cell>

            <van-cell title="最近解锁算力值">
              <span v-if="item.js_power > 0">{{ item.js_power }}</span>
              <span v-else>0</span>
            </van-cell> -->

            <van-cell title="投入算力">
              <span v-if="item.sales > 0">{{ item.sales }}</span>
              <span v-else>0</span>
            </van-cell>

            <van-cell title="预产算力">
              <span v-if="item.power > 0">{{ item.power }}</span>
              <span v-else>0</span>
            </van-cell>

            <van-cell title="已解锁算力">
              <span v-if="item.js_power > 0">{{ item.js_power }}</span>
              <span v-else>0</span>
            </van-cell>

            <van-cell title="投入时间">
              <span>{{ dateFormat(item['created_at'])  }}</span>
            </van-cell>
            
            
          </van-cell-group>
        </div>
      </div>

      


    </div>

    <van-dialog v-model:show="isjihuo" title="是否确认激活" show-cancel-button
                @confirm="updateIsjihuo"
                @cancel="isjihuo = false"
                :before-close="beforeClose">
 
    </van-dialog>
  </div>
</template>

<script setup>
import {onMounted, ref} from "vue";
import {showFailToast, showNotify, showSuccessToast} from "vant";
import {httpGet, httpPost} from "@/utils/http";
import {ElMessage} from "element-plus";
import {checkSession} from "@/action/session";
import {useRouter} from "vue-router";
import {dateFormat, removeArrayItem} from "@/utils/libs";

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


const orders = ref([])
const vipMonthPower = ref(0)
const payWays = ref({})
const router = useRouter()
const loginUser = ref(null)
const haveOrder = ref(false)

onMounted(() => {
  checkSession().then(user => {
    loginUser.value = user
    httpGet('/api/user/profile').then(res => {
      form.value = res.data
      fileList.value[0].url = form.value.avatar
    }).catch((e) => {
      console.log(e.message)
      showFailToast('获取用户信息失败')
    });




      // 获取用户已经结束算力列表
    httpGet("/api/order/getEndOrderList").then((res) => {
         orders.value = res.data
          if(res.data.length!=0){
             haveOrder.value=true
          }
    }).catch(e => {
      showFailToast("获取已结束算力列表失败：" + e.message)
    })

    
  

    httpGet("/api/config/get?key=system").then(res => {
      vipMonthPower.value = res.data['vip_month_power']
    }).catch(e => {
      showFailToast("获取系统配置失败：" + e.message)
    })



  }).catch(() => {
    router.push("/login")
  })

})


const showPasswordDialog = ref(false)
const pass = ref({
  old: "",
  new: "",
  renew: ""
})

const isjihuo = ref(false)

const beforeClose = (action) => {
  new Promise((resolve) => {
    resolve(action === 'confirm');
  });
}




//  是否确认激活
const updateIsjihuo = () => {
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

        .tpend {
          font-size 18px
          color #07c160
        }
      }
    }
  }
}
</style>