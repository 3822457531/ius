<template>
  <div class="mobile-trade container">
    <van-nav-bar title="交易中心"/>

    <div class="content">
      <van-tabs v-model:active="activeTab">
        <van-tab title="买入">
          <div class="trade-list buy-list">
            <div class="item" v-for="item in buyList" :key="item.id">
              <van-cell-group inset class="trade-card">
                <!-- 头部信息 -->
                <div class="card-header">
                  <span class="order-no">订单号：{{ item.orderNo }}</span>
                  <van-tag type="primary" round>买入</van-tag>
                </div>

                <!-- 交易信息 -->
                <div class="trade-info">
                  <div class="amount-info">
                    <div class="amount">
                      <span class="label">数量</span>
                      <span class="value">{{ item.amount }}</span>
                    </div>
                    <div class="price">
                      <span class="label">单价</span>
                      <span class="value">${{ item.price }}</span>
                    </div>
                    <div class="total">
                      <span class="label">总价</span>
                      <span class="value highlight">${{ item.total }}</span>
                    </div>
                  </div>
                </div>

                <!-- 时间信息 -->
                <div class="time-info">
                  <div class="time-item">
                    <van-icon name="clock-o" />
                    <span class="label">创建时间：</span>
                    <span class="time">{{ item.time }}</span>
                  </div>
                </div>


                            <!-- 操作按钮 -->
              <div class="action-btn" >
                  <van-button type="danger" size="small" @click="handleCancel(item)">
                    取消订单
                  </van-button>
                </div>
              </van-cell-group>
            </div>
          </div>
        </van-tab>

        <van-tab title="卖出">
          <div class="trade-list sell-list">
            <div class="item" v-for="item in sellList" :key="item.id">
              <van-cell-group inset class="trade-card">
                <!-- 头部信息 -->
                <div class="card-header">
                  <span class="order-no">订单号：{{ item.orderNo }}</span>
                  <van-tag type="warning" round>卖出</van-tag>
                </div>

                <!-- 交易信息 -->
                <div class="trade-info">
                  <div class="amount-info">
                    <div class="amount">
                      <span class="label">数量</span>
                      <span class="value">{{ item.amount }}</span>
                    </div>
                    <div class="price">
                      <span class="label">单价</span>
                      <span class="value">${{ item.price }}</span>
                    </div>
                    <div class="total">
                      <span class="label">总价</span>
                      <span class="value highlight">${{ item.total }}</span>
                    </div>
                  </div>
                </div>

                <!-- 时间信息 -->
                <div class="time-info">
                  <div class="time-item">
                    <van-icon name="clock-o" />
                    <span class="label">创建时间：</span>
                    <span class="time">{{ item.time }}</span>
                  </div>
                </div>

                  <!-- 操作按钮 -->
              <div class="action-btn" >
                  <van-button type="danger" size="small" @click="handleCancel(item)">
                    取消订单
                  </van-button>
                </div>
              </van-cell-group>

                
            </div>
          </div>
        </van-tab>

        <van-tab title="交易中">
          <div class="trade-list processing-list">
            <div class="item" v-for="item in processingList" :key="item.id">
              <van-cell-group inset class="trade-card">
                <!-- 头部信息 -->
                <div class="card-header">
                  <span class="order-no">订单号：{{ item.orderNo }}</span>
                  <van-tag type="primary" :type="item.type.includes('买') ? 'primary' : 'warning'" round>
                    {{ item.type }}
                  </van-tag>
                </div>

                <!-- 交易信息 -->
                <div class="trade-info">
                  <div class="status-badge">
                    <van-tag type="processing" color="#ff976a">{{ item.statusText }}</van-tag>
                    <div class="countdown-wrapper" v-if="item.status === 2">
                      <van-count-down 
                        :time="getCountdownTime(item)"
                        :auto-start="true"
                        @finish="handleCountdownFinish(item)"
                      >


                        <!-- <template #default="{ time }">
                          <span class="countdown-label">剩余时间:</span>
                          <span class="countdown-time">
                            {{ Math.floor(time / 3600000).toString().padStart(2, '0') }}:
                            {{ Math.floor((time % 3600000) / 60000).toString().padStart(2, '0') }}:
                            {{ Math.floor((time % 60000) / 1000).toString().padStart(2, '0') }}
                          </span>
                        </template> -->


                      </van-count-down>
                    </div>
                  </div>
                  
                  <div class="amount-info">
                    <div class="amount">
                      <span class="label">数量</span>
                      <span class="value">{{ item.amount }}</span>
                    </div>
                    <div class="price">
                      <span class="label">单价</span>
                      <span class="value">${{ item.price }}</span>
                    </div>
                    <div class="total">
                      <span class="label">总价</span>
                      <span class="value highlight">${{ item.total }}</span>
                    </div>
                  </div>
                </div>

                <!-- ETH地址信息 -->
                <div class="eth-address-info">
                  <div class="address-container">
                    <van-icon name="link" color="#1989fa"/>
                    <span class="label">BSC地址</span>
                    <span class="address">{{ formatAddress(item.ethAddress) }}</span>
                  </div>
                  <van-button 
                    size="small" 
                    type="primary" 
                    class="copy-btn"
                    @click="copyAddress(item.ethAddress)"
                  >
                    复制
                  </van-button>
                </div>

                <!-- 时间信息 -->
                <div class="time-info">
                  <div class="time-item">
                    <van-icon name="clock-o" />
                    <span class="label">创建时间：</span>
                    <span class="time">{{ item.time }}</span>
                  </div>
                </div>

                <!-- 操作按钮 -->
                <div class="action-btn" v-if="item.userRole === 'buyer' && item.showUploadButton">
                  <van-button type="primary" size="small" @click="showUploadDialog(item)">
                    上传凭证
                  </van-button>
                  <van-button type="danger" size="small" @click="handleCancel(item)">
                    取消订单
                  </van-button>
                </div>

                <div class="action-btn" v-if="item.userRole === 'seller'">
                  <van-button 
                    v-if="item.status === 3"
                    type="success" 
                    size="small" 
                    @click="handleConfirmPayment(item)"
                  >
                    确认收款
                  </van-button>

                  <van-button 
                    v-if="item.status === 3"
                    type="danger" 
                    size="small" 
                    @click="handleAppeal(item)"
                    style="margin-left: 8px;"
                  >
                    申诉
                  </van-button>

                  
                  <!-- <van-button type="danger" size="small" @click="handleCancel(item)">
                    取消订单1
                  </van-button> -->
                </div>
              </van-cell-group>
            </div>
          </div>
        </van-tab>

        <van-tab title="已完成">
          <div class="trade-list completed-list">
            <div class="item" v-for="item in completedList" :key="item.id">
              <van-cell-group inset class="trade-card">
                <!-- 添加状态标签和时间 -->
                <div class="card-header">
                  <span class="order-no">订单号：{{ item.orderNo }}</span>
                  <van-tag type="success" round>已完成</van-tag>
                </div>

                <!-- 主要交易信息 -->
                <div class="trade-info">
                  <div class="trade-type">
                    <van-tag 
                      :type="item.type.includes('买入') ? 'primary' : 'warning'"
                      plain
                    >{{ item.type }}</van-tag>
                  </div>
                  
                  <div class="amount-info">
                    <div class="amount">
                      <span class="label">数量</span>
                      <span class="value">{{ item.amount }}</span>
                    </div>
                    <div class="price">
                      <span class="label">单价</span>
                      <span class="value">${{ item.price }}</span>
                    </div>
                    <div class="total">
                      <span class="label">总价</span>
                      <span class="value highlight">${{ item.total }}</span>
                    </div>
                  </div>
                </div>

                <!-- 时间信息 -->
                <div class="time-info">
                  <div class="time-item">
                    <van-icon name="clock-o" />
                    <span class="label">创建时间：</span>
                    <span class="time">{{ item.time }}</span>
                  </div>
                  <div class="time-item">
                    <van-icon name="checked" />
                    <span class="label">完成时间：</span>
                    <span class="time">{{ item.completedTime }}</span>
                  </div>
                </div>

                <!-- 添加交易哈希信息 -->
                <div class="hash-info" v-if="item.txHash">
                  <van-icon name="link" />
                  <span class="label">交易哈希：</span>
                  <span class="hash">{{ item.txHash }}</span>
                </div>
              </van-cell-group>
            </div>
          </div>
        </van-tab>
      </van-tabs>
    </div>

    <!-- 添加上传凭证弹窗 -->
    <van-dialog 
      v-model:show="showUpload" 
      title="上传支付凭证" 
      show-cancel-button
      @confirm="submitPaymentProof"
      @cancel="showUpload = false"
    >
      <div class="upload-content">
        <van-uploader
          v-model="uploadImages"
          :max-count="1"
          :after-read="afterRead"
          :before-read="beforeRead"
          accept="image/*"
        >
          <template #preview-cover="{ file }">
            <div class="preview-cover" v-if="file.status === 'uploading'">
              <van-loading type="spinner" size="20" />
              <span class="upload-text">{{ file.message }}</span>
            </div>
          </template>
        </van-uploader>
        <div class="upload-tip">
          <p>请上传清晰的支付凭证截图</p>
          <p>支持jpg、png格式，小不超过10M</p>
        </div>
      </div>
    </van-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from "vue";
import { showFailToast, showSuccessToast, showConfirmDialog } from "vant";
import { httpGet, httpPost } from "@/utils/http";
import Compressor from 'compressorjs';
import Clipboard from 'clipboard';

// 添加组件名称定义
defineOptions({
  name: 'TradeView'
});

const activeTab = ref(2); // 默认选中"交易中"标签

// 买入列表数据
const buyList = ref([
 // {
 //   id: 1,
 //   orderNo: 'B202409282095657',
 //   amount: '100',
 //   price: '0.75',
 //   total: '75.00',
  //  time: '2024-10-31 12:02:18'
 // }
]);

// 卖出列表数据
const sellList = ref([
  //{
  //  id: 2,
  //  orderNo: 'S202409282095658',
  //  amount: '50',
  //  price: '0.80',
  //  total: '40.00',
  //  cnyTotal: '260.00',
  //  time: '2024-10-31 13:15:20'
  //}
]);

// 交易中列表数据
const processingList = ref([

]);

//  {
 // id: 3,
 // orderNo: 'P202409282095659',
 // type: '买交易',
 // amount: '200',
 // price: '0.75',
 // total: '150.00',
 // cnyTotal: '975.00',
  //  time: '2024-10-31 14:30:45',
 //  status: 'pending',
 //  statusText: '等待打款',
 //  ethAddress: '0x2af8ae409ac94ed6fbd7d6c8751f3fbdf055ab3e',
 //  userRole: 'buyer'
  //}

// 已完成列表数据
const completedList = ref([
 // {
 //   id: 4,
 //   orderNo: 'C202409282095660',
 //   type: '卖出',
 //   amount: '150',
 //   price: '0.78',
 //   total: '117.00',
 //   cnyTotal: '760.50',
 //   time: '2024-10-31 09:20:10',
 //   completedTime: '2024-10-31 09:35:22'
 // }
]);

// 上传凭证相关
const showUpload = ref(false);
const uploadImages = ref([]);
const currentOrder = ref(null);

// 图片上传前处理
const beforeRead = (file) => {
  // 验证文件类型
  const isImage = file.type.indexOf('image') !== -1;
  if (!isImage) {
    showFailToast('请上传图片格式文件！');
    return false;
  }
  // 验证文件大小
  if (file.size > 10 * 1024 * 1024) {
    showFailToast('图片大小不能超过 10M');
    return false;
  }
  return true;
};

// 处理图片上传
const afterRead = (file) => {
  file.status = 'uploading';
  file.message = '上传中...';
  
  // 压缩图片并上传
  new Compressor(file.file, {
    quality: 0.6,
    success(result) {
      const formData = new FormData();
      formData.append('file', result, result.name);
      
      // 使用与 Invitation.vue 相同的上传接口
      httpPost('/api/upload', formData)
        .then((res) => {
          file.status = 'done';
          file.message = '上传成功';
          currentOrder.value.proofUrl = res.data.url;
          showSuccessToast('凭证上传成功');
          
          // 上传成功后立即提交凭证信息
          httpPost('/api/trade/waitComfireOrder', {
            orderId: currentOrder.value.id,
            proofUrl: res.data.url
          }).then(() => {
            showSuccessToast('凭证提交成功');
            showUpload.value = false;
            uploadImages.value = [];
            loadTradeList();
          }).catch((e) => {
            showFailToast('提交失败：' + e.message);
          });
        })
        .catch((e) => {
          file.status = 'failed';
          file.message = '上传失败';
          showFailToast('上传失败：' + e.message);
        });
    },
    error(err) {
      file.status = 'failed';
      file.message = '压缩失败';
      showFailToast('图片处理失败：' + err.message);
    },
  });
};

// 修改提交支付凭证函数
const submitPaymentProof = () => {
  if (uploadImages.value.length === 0) {
    showFailToast('请先上传支付凭证');
    return;
  }
  
  // 由于在上传后已经提交了凭证信息，这里直接关闭弹窗
  showUpload.value = false;
};

// 处理买入操作
const handleBuy = (item) => {
  httpPost('/api/trade/buy', { orderId: item.id }).then(() => {
    showSuccessToast('买入申请已提交');
    loadTradeList();
  }).catch(e => {
    showFailToast('买入失：' + e.message);
  });
};

// 处理卖出操作
const handleSell = (item) => {
  httpPost('/api/trade/sell', { orderId: item.id }).then(() => {
    showSuccessToast('卖出申请已提交');
    loadTradeList();
  }).catch(e => {
    showFailToast('卖出失败：' + e.message);
  });
};

// 删除 A 方式的 copyAddress 函数，使用以下 B 方式实现
const copyAddress = (address) => {
  const clipboard = new Clipboard('.copy-btn', {
    text: () => address
  });

  clipboard.on('success', () => {
    showSuccessToast('地址已复制');
    clipboard.destroy();
  });

  clipboard.on('error', () => {
    showFailToast('复制失败，请手动复制');
    clipboard.destroy();
  });
};

// 显示上传凭证弹窗
const showUploadDialog = (item) => {
  currentOrder.value = item;
  showUpload.value = true;
  // 清空之前的上传记录
  uploadImages.value = [];
};

// 确认收款
const handleConfirmPayment = (item) => {
  showConfirmDialog({
    title: '确认收款',
    message: '请确认您已收到买家的付款',
  }).then(() => {
    httpPost('/api/trade/comfireOrder', { orderId: item.id })
      .then(() => {
        showSuccessToast('确认收款成功');
        loadTradeList();
      })
      .catch(e => {
        showFailToast('操作失败：' + e.message);
      });
  });
};

// 取消订单
const handleCancel = (item) => {
  showConfirmDialog({
    title: '取消订单',
    message: '确定要取消该订单吗？',
  }).then(() => {
    httpPost('/api/trade/cancel', { orderId: item.id })
      .then(() => {
        showSuccessToast('订单已取消');
        loadTradeList();
      })
      .catch(e => {
        showFailToast('取消失败：' + e.message);
      });
  });
};

// 申诉处理
const handleAppeal = (item) => {
  showConfirmDialog({
    title: '申诉确认',
    message: `确定要对订单 ${item.orderNo} 进行申诉吗？\n申诉理由：买家虚假付款\n\n申诉后将提交给管理员处理，订单将被暂时冻结。`,
  }).then(() => {
    // const appealData = {
    //   orderId: item.id,
    //   orderNumber: item.orderNo,
    //   buyerName: item.userName || '未知用户',
    //   sellerName: '当前用户',
    //   amount: parseFloat(item.amount),
    //   price: parseFloat(item.price),
    //   totalAmount: item.total,
    //   reason: '买家虚假付款',
    //   status: item.status
    // };
    
    httpPost('/api/trade/appeal', { orderId: item.id })
      .then(() => {
        showSuccessToast('申诉提交成功！订单已冻结，等待管理员处理');
        loadTradeList();
      })
      .catch(e => {
        showFailToast('申诉提交失败：' + e.message);
      });
  }).catch(() => {
    // 用户取消申诉
  });
};

// 加载交易列表数据
const loadTradeList = () => {
  // 获取买入列表
  httpGet("/api/trade/buyOrderList").then((res) => {
    buyList.value = res.data.map(item => ({
      id: item.Id,
      orderNo: item.OrderNumber.toString(),
      amount: item.Number.toString(),
      price: item.Price.toFixed(4),
      total: (item.Number * item.Price).toFixed(2),
      cnyTotal: item.TotalPriceChina.toFixed(2),
      time: new Date(item.CreatedAt * 1000).toLocaleString(),
      status: item.Status,
      userName: item.UserName,
      avatar: item.Avatar
    }));
  }).catch(e => {
    showFailToast("获取买入列表失败：" + e.message);
  });

  // 获取卖出列表
  httpGet("/api/trade/sellOrderList").then((res) => {
    //sellList.value = res.data;

    sellList.value = res.data.map(item => ({
      id: item.Id,
      orderNo: item.OrderNumber.toString(),
      amount: item.Number.toString(),
      price: item.Price.toFixed(4),
      total: (item.Number * item.Price).toFixed(2),
      cnyTotal: item.TotalPriceChina.toFixed(2),
      time: new Date(item.CreatedAt * 1000).toLocaleString(),
      status: item.Status,
      userName: item.UserName,
      avatar: item.Avatar
    }));

  }).catch(e => {
    showFailToast("获取卖出列表失败：" + e.message);
  });

  // 获取交易中列
  httpGet("/api/trade/ingOrderList").then((res) => {
    processingList.value = res.data.map(item => {
      console.log('Raw item:', item);
      
      return {
        id: item.Id,
        orderNo: item.OrderNumber.toString(),
        type: item.IsComment ? '买入' : '卖出',
        amount: item.Number.toString(),
        price: item.Price.toFixed(4),
        total: (item.Number * item.Price).toFixed(2),
        cnyTotal: item.TotalPriceChina.toFixed(2),
        time: new Date(item.UpdatedAt * 1000).toLocaleString(),
        UpdatedAt: Number(item.UpdatedAt),
        status: item.Status,
        statusText: item.IsComment ? '买入' : '卖出',
        ethAddress: item.EthAddress,//'0x2af8ae409ac94ed6fbd7d6c8751f3fbdf055ab3e',
        userRole: item.IsComment ? 'buyer' : 'seller',
        showConfirmButton: !item.IsComment && item.Status === 3,
        showUploadButton: item.IsComment && item.Status === 2,
        proofUrl: item.Image || ''
      };
    });
  }).catch(e => {
    showFailToast("获取交易中列表失败：" + e.message);
  });

  // 获取已完成列表
  httpGet("/api/trade/completedOrderList").then((res) => {
    completedList.value = res.data.map(item => {
      const tradeType = item.IsComment ? '买入' : '卖出';
      return {
        id: item.Id,
        orderNo: item.OrderNumber.toString(),
        amount: item.Number.toString(),
        price: item.Price.toFixed(4),
        total: (item.Number * item.Price).toFixed(2),
        cnyTotal: item.TotalPriceChina.toFixed(2),
        time: new Date(item.CreatedAt * 1000).toLocaleString(),
        status: item.Status,
        userName: item.UserName,
        completedTime: new Date(item.UpdatedAt * 1000).toLocaleString(),
        type: tradeType,
        avatar: item.Avatar
      };
    });
  }).catch(e => {
    showFailToast("取已完成列表失败：" + e.message);
  });
};

// 格式化函数
const formatAddress = (address) => {
  if (!address) return '';
  return `${address.slice(0, 6)}...${address.slice(-4)}`;
};

// 修改计算倒计时的函数
const getCountdownTime = (item) => {
  if (!item.UpdatedAt) {
    console.log('No UpdatedAt found:', item);
    return 0;
  }
  
  const updateTime = item.UpdatedAt * 1000;
  const deadline = updateTime + 12 * 60 * 60 * 1000;
  const now = Date.now();
  const remaining = deadline - now;
  
  console.log('Countdown calculation:', {
    updateTime,
    deadline,
    now,
    remaining,
    UpdatedAt: item.UpdatedAt,
    item
  });
  
  return remaining > 0 ? remaining : 0;
};

// 倒计时结束处理函数
const handleCountdownFinish = (item) => {
  // 如果订单状态仍然是等待打款，则自动取消
  if (item.status === 2) {
    httpPost('/api/trade/cancel', { orderId: item.id })
      .then(() => {
        showSuccessToast('订单已超时自动取消');
        loadTradeList();
      })
      .catch(e => {
        showFailToast('取消失败：' + e.message);
      });
  }
};

onMounted(() => {
  loadTradeList();
});
</script>

<style lang="stylus">
.mobile-trade {
  .content {
    .trade-list {
      padding: 10px

      .item {
        margin-bottom: 15px
        
        .van-cell-group {
          border-radius: 8px
          overflow: hidden
        }

        .action-btn {
          padding: 10px
          text-align: right
        }
      }
    }
  }
}

.van-tabs {
  .van-tab {
    font-size: 14px
  }
}

.eth-address {
  display: flex
  align-items: center
  justify-content: space-between
  
  .address-text {
    flex: 1
    margin-right: 10px
    overflow: hidden
    text-overflow: ellipsis
    white-space: nowrap
  }
  
  .copy-btn {
    flex-shrink: 0
  }
}

.upload-content {
  padding: 20px
  
  .van-uploader {
    width: 100%
    display: flex
    justify-content: center
    
    .van-uploader__wrapper {
      width: 200px
      height: 200px
    }
    
    .van-uploader__upload {
      width: 100%
      height: 100%
      background-color: #f7f8fa
    }
    
    .van-uploader__preview {
      width: 200px
      height: 200px
    }
  }
  
  .upload-tip {
    margin-top: 12px
    text-align: center
    color: #969799
    font-size: 12px
    
    p {
      margin: 4px 0
    }
  }
}

.preview-cover {
  position: absolute
  top: 0
  left: 0
  right: 0
  bottom: 0
  display: flex
  flex-direction: column
  align-items: center
  justify-content: center
  color: #fff
  background: rgba(0, 0, 0, 0.5)
  
  .van-loading {
    margin-bottom: 8px
  }
  
  .upload-text {
    font-size: 12px
  }
}

.action-btn {
  padding: 10px
  text-align: right
  
  .van-button {
    margin-left: 8px
  }
}

.completed-list {
  .item {
    margin-bottom: 16px
    
    .trade-card {
      background: #fff
      border-radius: 12px
      overflow: hidden
      box-shadow: 0 2px 12px rgba(100, 101, 102, 0.08)
      
      .card-header {
        padding: 12px 16px
        display: flex
        justify-content: space-between
        align-items: center
        border-bottom: 1px solid #f5f5f5
        
        .order-no {
          color: #666
          font-size: 14px
        }
      }
      
      .trade-info {
        padding: 16px
        
        .trade-type {
          margin-bottom: 12px
        }
        
        .amount-info {
          display: flex
          justify-content: space-between
          padding: 12px 0
          
          .amount, .price, .total {
            display: flex
            flex-direction: column
            align-items: center
            
            .label {
              color: #999
              font-size: 12px
              margin-bottom: 4px
            }
            
            .value {
              font-size: 16px
              font-weight: 500
              color: #333
              
              &.highlight {
                color: #ff6b00
                font-size: 18px
              }
            }
          }
        }
      }
      
      .time-info {
        padding: 0 16px 16px
        background: #f8f9fa
        
        .time-item {
          display: flex
          align-items: center
          padding: 8px 0
          font-size: 13px
          
          .van-icon {
            margin-right: 6px
            color: #666
          }
          
          .label {
            color: #666
            margin-right: 8px
          }
          
          .time {
            color: #333
          }
        }
      }
      
      .hash-info {
        padding: 12px 16px
        border-top: 1px solid #f5f5f5
        display: flex
        align-items: center
        font-size: 13px
        
        .van-icon {
          margin-right: 6px
          color: #666
        }
        
        .label {
          color: #666
          margin-right: 8px
        }
        
        .hash {
          color: #1989fa
          overflow: hidden
          text-overflow: ellipsis
          white-space: nowrap
        }
      }
    }
  }
}

// 通用列表样式
.trade-list {
  padding: 16px
  
  .item {
    margin-bottom: 16px
  }
  
  .trade-card {
    background: #fff
    border-radius: 12px
    overflow: hidden
    box-shadow: 0 2px 12px rgba(100, 101, 102, 0.08)
    
    .card-header {
      padding: 12px 16px
      display: flex
      justify-content: space-between
      align-items: center
      border-bottom: 1px solid #f5f5f5
      
      .order-no {
        color: #666
        font-size: 14px
      }
    }
    
    .trade-info {
      padding: 16px
      
      .status-badge {
        margin-bottom: 12px
      }
      
      .amount-info {
        display: flex
        justify-content: space-between
        padding: 12px 0
        
        .amount, .price, .total {
          display: flex
          flex-direction: column
          align-items: center
          
          .label {
            color: #999
            font-size: 12px
            margin-bottom: 4px
          }
          
          .value {
            font-size: 16px
            font-weight: 500
            color: #333
            
            &.highlight {
              color: #ff6b00
              font-size: 18px
            }
          }
        }
      }
    }
    
    .eth-address-info {
      padding: 12px 16px
      background: #f8f9fa
      display: flex
      justify-content: space-between
      align-items: center
      border-radius: 4px
      margin: 0 16px
      
      .address-container {
        display: flex
        align-items: center
        flex: 1
        margin-right: 12px
        
        .van-icon {
          font-size: 16px
          margin-right: 6px
        }
        
        .label {
          color: #666
          margin-right: 8px
          font-size: 14px
        }
        
        .address {
          color: #1989fa
          font-family: monospace
          font-size: 14px
          background: #fff
          padding: 4px 8px
          border-radius: 4px
          border: 1px solid #ebedf0
        }
      }
      
      .copy-btn {
        flex-shrink: 0
        height: 28px
        padding: 0 12px
        font-size: 12px
      }
    }
    
    .time-info {
      padding: 12px 16px
      background: #f8f9fa
      border-top: 1px solid #f5f5f5
      
      .time-item {
        display: flex
        align-items: center
        font-size: 13px
        
        .van-icon {
          margin-right: 6px
          color: #666
        }
        
        .label {
          color: #666
          margin-right: 8px
        }
        
        .time {
          color: #333
        }
      }
    }
    
    .action-btn {
      padding: 12px 16px
      text-align: right
      border-top: 1px solid #f5f5f5
      
      .van-button {
        margin-left: 8px
      }
    }
  }
}

// 买入列表特定样式
.buy-list {
  .trade-card {
    border-left: 4px solid #1989fa
  }
}

// 卖出列表特定样式
.sell-list {
  .trade-card {
    border-left: 4px solid #ff976a
  }
}

// 交易中列表特定样式
.processing-list {
  .trade-card {
    border-left: 4px solid #07c160
  }
}

.trade-list {
  .trade-card {
    .trade-info {
      .status-badge {
        display: flex
        justify-content: space-between
        align-items: center
        margin-bottom: 12px
        
        .countdown-wrapper {
          display: flex
          align-items: center
          background: rgba(255, 151, 106, 0.1)
          padding: 4px 8px
          border-radius: 4px
          
          .countdown-label {
            color: #666
            font-size: 12px
            margin-right: 4px
          }
          
          .countdown-time {
            color: #ff976a
            font-size: 14px
            font-family: monospace
            font-weight: 500
          }
        }
      }
    }
  }
}
</style>