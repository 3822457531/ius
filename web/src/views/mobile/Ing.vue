<template>
  <div class="mobile-user-profile container">
    <van-nav-bar :title="title"/>

    <div class="content">


      <div class="product-list">
        <h3 v-if="haveOrder">我的算力列表</h3>
        <div v-else class="no-data">
          <van-empty 
                  image="https://fastly.jsdelivr.net/npm/@vant/assets/custom-empty-image.png"
                  image-size="80"
                  description="暂无记录"
          />
        </div>
        <div class="item" v-for="item in orders" :key="item.id">
          <h3 class="title"> 
            <!-- <span class="tping">{{ item.subject }}  </span> -->


            <span class="tping">
  {{ item.subject.includes('自定义套餐：') ? 
     '自定义套餐：' + parseFloat(item.subject.split('自定义套餐：')[1]) : 
     item.subject }}
</span>


            <div class="buy-btn">
             <van-button type="success" :to="`/mobile/syitems?productId=${item.order_no}`" size="small">
              <span><i class="iconfont"></i> 收益详情</span>
             </van-button>
        </div>




          </h3>

          <van-cell-group>
      
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

      <div class="product-list">
        <h3 v-if="haveMint">我的矿机列表</h3>
        <div class="item" v-for="item in mints" :key="item.id">
          <h3 class="title">
            <span class="tping">{{ item.subject }}</span>


            <div class="buy-btn" v-if="item.js_coin>0" >
              <!-- <van-button type="success" @click="withdraw(item)" size="small" >
               <span><i class="iconfont  "></i> 提币</span> 
                <span><i class="iconfont"></i> 收益详情</span>
              </van-button> -->


            <van-button type="success" :to="`/mobile/coin?productId=${item.order_no}&title=${encodeURIComponent(item.subject || '算力矿机')}`" size="small">
              <span><i class="iconfont"></i> 矿机明细</span>
             </van-button>

            </div> 


          </h3>


          <van-cell-group>

            <van-cell title="投入算力值">
              <span >{{ item.power }}</span>
            </van-cell>

            
            <van-cell title="已产出SOHA币">
              <span v-if="item.js_coin > 0">{{ item.js_coin }}</span>
              <span v-else>0</span>
            </van-cell>


            <van-cell title="投入时间">
              <span >{{ dateFormat(item['created_at'])  }}</span>
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
import {  showConfirmDialog,showFailToast, showNotify, showSuccessToast,showToast} from "vant";
import {httpGet, httpPost} from "@/utils/http";
import Compressor from 'compressorjs';
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
const mints = ref([])
const vipMonthPower = ref(0)
const router = useRouter()
const loginUser = ref(null)


const haveMint = ref(false)
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


      // 获取用户算力套餐列表
      httpGet("/api/order/getOrderList").then((res) => {
      orders.value = res.data
      console.log("orders.value为"+orders.value)
         //没有值
         if(res.data.length!=0){
           haveOrder.value=true
          }
    }).catch(e => {
      showFailToast("获取订单套餐失败：" + e.message)
    })


      // 获取用户矿机列表
    httpGet("/api/mintOrder/getMintOrderList").then((res) => {
      mints.value = res.data
      //没有值
      if(res.data.length!=0){
        haveMint.value=true
      
      }
      
    
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

const isjihuo = ref(false)

const beforeClose = (action) => {
  new Promise((resolve) => {
    resolve(action === 'confirm');
  });
}




//  是否确认激活
const withdraw = (item) => {
  showConfirmDialog({
    title: '是否提现',
    message:
        '确定提现吗?',
  }).then(() => {

    //提现通道暂未开启

    showToast("提现通道暂未开启,敬请期待...")

  }).catch(() => {

  //  showToast("您取消了激活")

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
          color #409eff
        }

        .tping {
          font-size 18px
          color #ed6a0c
        }
      }
    }
  }
}
</style>