<template>
    <div class="market-container">
      <!-- 顶部按钮 -->
      <div class="top-buttons" :data-active="currentView">
        <div 
          class="button-item"
          :class="{ active: currentView === 'order' }"
          @click="currentView = 'order'"
        >
          自助下单
        </div>
        <div 
          class="button-item"
          :class="{ active: currentView === 'market' }"
          @click="currentView = 'market'"
        >
          交易市场
        </div>
        <div 
          class="button-item"
          :class="{ active: currentView === 'trade' }"
          @click="currentView = 'trade'; router.push('/mobile/trade')"
        >
          我的订单
        </div>
      </div>
  
      <!-- 下单界面 (A界面) -->
  
  
  
      
  
  
      <div v-if="currentView === 'order'" class="trade-section">
  
         
  
        <!-- 价格趋势图 -->
        <div class="price-chart">
          <div class="chart-title">SOHA价格线 <span class="sub-title">实时价格走势</span></div>
          <div class="chart-container" ref="chartRef"></div>
        </div>
  
         <!-- 顶部价格信息 -->
         <div class="price-info">
          <div class="current-price">
            <div class="label">当前指导价</div>
            <div class="value" style="color: #1989fa">
              {{ currentPrice }}$
              <span style="color: #07c160; margin-left: 40px">
                {{ priceChange >= 0 ? '+' : '' }}{{ priceChange }}%
              </span>
            </div>
          </div>
  
          <div class="price-stats">
            <div class="stat-item">
              <div class="label">目前已产出量</div>
              <div class="value" style="color: #1989fa; font-size: 18px; font-weight: bold;">
                {{ volume }} <span style="font-size: 14px; color: #666;">SOHA</span>
              </div>
            </div>
          </div>
        </div>
  
  
        <div class="tab-header">
          <!-- 原有的买入卖出tab -->
          <div class="tabs">
            <div 
              v-for="tab in ['买入SOHA', '卖出SOHA']" 
              :key="tab"
              :class="['tab-item', { active: currentTab === tab }]"
              @click="currentTab = tab"
            >
              我要{{ tab }}
            </div>
          </div>
        </div>
  
        <!-- 原有的交易表单 -->
        <div class="trade-form">
          <!-- 保持原有的表单内容不变 -->
          <van-field
            v-model="amount"
            type="number"
            :label="currentTab === '买入SOHA' ? '购买数量' : '卖出数量'"
            :placeholder="currentTab === '买入SOHA' ? '请输入购买数量' : '请输入卖出数量'"
            class="custom-field"
          />
          <van-field
            v-model="price"
            type="number"
            :label="currentTab === '买入SOHA' ? '购买单价' : '卖出单价'"
            :placeholder="currentTab === '买入SOHA' ? '请输入购买单价' : '请输入卖出单价'"
            class="custom-field"
          />
          
          <div class="trade-info">
            <div class="total-amount">
              <span class="label">{{ currentTab === '买入SOHA' ? '预计支出' : '预计收入' }}: </span>
              <span class="value">${{ calculateTotal }}</span>
            </div>
            <div class="balance">
              <span class="label">可用额度:</span>
              <span class="value">{{ availableBalance }}</span>
            </div>
          </div>
  
          <div class="submit-btn">
            <van-button 
              type="primary" 
              block 
              @click="handleSubmit"
              :loading="isSubmitting"
              :class="{ 'buy-button': currentTab === '买入SOHA' }"
            >
              {{ currentTab === '买入SOHA' ? '提交购买' : '提交卖出' }}
            </van-button>
          </div>
        </div>
      </div>
  
      <!-- 交易市场界面 (B面) -->
      <div v-if="currentView === 'market'" class="order-lists" :data-active="activeList">
        <div class="list-tabs">
          <div 
            class="tab-item" 
            :class="{ active: activeList === 'buy' }"
            @click="activeList = 'buy'"
          >
            求购列表
          </div>
  
          <div 
            class="tab-item" 
            :class="{ active: activeList === 'sell' }"
            @click="activeList = 'sell'"
          >
            出售列表
          </div>
        </div>
  
        <!-- 保持原有的列表内容不变 -->
        <div class="order-list">
          <div v-for="item in currentList" :key="item.id" class="order-item">
            <div class="user-info">
              <img :src="item.avatar" class="avatar" />
              <span class="username">{{ item.username }}</span>
            </div>
            <div class="order-details">
              <div class="price-info">
                <div class="main-info">
                  <span class="amount">
                    <span class="label">数量:</span>
                    <span class="q2-text">{{ item.amount }} </span>
                  </span>
                  <span class="price">
                    <span class="label">单价:</span>
                    <span class="value">${{ item.price }}</span>
                  </span>
                </div>
                <div class="total-info">
                  <span class="label">总额:</span>
                  <span class="total-value">${{ (item.amount * item.price).toFixed(4) }}</span>
                </div>
              </div>
            </div>
            <van-button 
              type="primary" 
              size="small" 
              class="trade-btn"
              @click="matchSubmit({
                id: item.id,
                types: activeList === 'buy' ? 1 : 2,
                price: item.price,
                num: item.amount
              })"
            >
              {{ activeList === 'buy' ? '卖出' : '买入' }}
            </van-button>
          </div>
        </div>
      </div>
    </div>
  </template>
  
  <script setup>
  import { ref, onMounted, computed, watch, nextTick, onUnmounted } from 'vue'
  import { showToast, showFailToast, showConfirmDialog, showSuccessToast } from 'vant'
  import * as echarts from 'echarts'
  import {httpGet, httpPost} from "@/utils/http";
  import { useRouter } from 'vue-router'
  import { enrichSohaPriceHistoryChangeRates } from '@/utils/sohaPrice'
  
  // 状态数据
  const currentPrice = computed(() => {
    if (!priceHistory.value || priceHistory.value.length === 0) return 0
    return priceHistory.value[priceHistory.value.length - 1].Price
  })
  const cnyRate = ref(6.5)
  const highPrice = ref(0.75)
  const lowPrice = ref(0.67)
  const amount = ref('')
  const price = ref('')
  const currentTab = ref('买入SOHA')
  const isSubmitting = ref(false)
  const availableBalance = ref(10200.00)
  const priceChange = computed(() => {
    if (!fullPriceHistory.value?.length) return 0
    const last = fullPriceHistory.value[fullPriceHistory.value.length - 1]
    const rate = Number(last.ChangeRate)
    return Number.isFinite(rate) ? Number(rate.toFixed(2)) : 0
  })
  const chartRef = ref(null)
  const searchKeyword = ref('')
  const activeList = ref('buy')
  const currentView = ref('order') // 控制显示哪个界面
  
    // 在顶部其他 ref 定义的地方添加 volume
  const volume = ref(0)  // 初始化为 0
  
  // 模拟数据 改为空数组初始化
  const buyList = ref([])
  const sellList = ref([])
  
  // 计算当前显示的列表
  const currentList = computed(() => {
    return activeList.value === 'buy' ? buyList.value : sellList.value
  })
  
  // 计算总额
  const calculateTotal = computed(() => {
    if (!amount.value || !price.value) return 0
    return (Number(amount.value) * Number(price.value)).toFixed(2)
  })
  
  // 添加一个新的 ref 来存储完整的历史数据和当前显示的数据
  const fullPriceHistory = ref([]) // 存储所有历史数据
  const priceHistory = ref([])     // 存储当前显示的数据
  const isLoadingMore = ref(false) // 加载状态标记
  
  // 修改初始化数据加载函数
  const initPriceData = async () => {
    try {
      const res = await httpGet('/api/trade/priceHistory')
      if (res.code === 0 && Array.isArray(res.data)) {
        fullPriceHistory.value = enrichSohaPriceHistoryChangeRates(res.data)
        priceHistory.value = fullPriceHistory.value.slice(-7)
        nextTick(() => {
          initChart()
        })
      }
    } catch (e) {
      console.error('获取价格历史数据失败:', e)
      showFailToast('获取价格历史数据失败')
    }
  }
  
  // 修改 initChart 函数中的相关部分
  const initChart = () => {
    if (!chartRef.value) return
    
    const existingChart = echarts.getInstanceByDom(chartRef.value)
    if (existingChart) {
      existingChart.dispose()
    }
    
    const chart = echarts.init(chartRef.value)
    
    // 确保数据格式正确
    const kLineData = priceHistory.value.map((item, index, arr) => {
      try {
        const prevItem = index > 0 ? arr[index - 1] : item;
        const openPrice = prevItem.Price;
        const closePrice = item.Price;
        
        if (typeof openPrice !== 'number' || typeof closePrice !== 'number') {
          console.warn('Invalid price data:', { openPrice, closePrice });
          return null;
        }
        
        return {
          value: [
            openPrice,                                    // 开盘价
            closePrice,                                   // 收盘价
            Math.min(openPrice, closePrice) * 0.99,      // 最低价（模拟）
            Math.max(openPrice, closePrice) * 1.01,      // 最高价（模拟）
          ],
          itemStyle: {
            color: closePrice >= openPrice ? '#07c160' : '#ff4d4f',
            borderColor: closePrice >= openPrice ? '#07c160' : '#ff4d4f'
          }
        };
      } catch (error) {
        console.error('Data conversion error:', error);
        return null;
      }
    }).filter(item => item !== null); // 过滤掉无效数据
  
    const option = {
      backgroundColor: '#ffffff',
      animation: false, // 关闭动画以提高性能
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'cross',
          snap: true,
          label: {
            backgroundColor: '#1989fa'
          }
        },
        backgroundColor: 'rgba(255, 255, 255, 0.9)',
        borderWidth: 1,
        borderColor: '#ccc',
        padding: [5, 10],
        textStyle: {
          color: '#333',
        },
        position: function (pos, params, el, elRect, size) {
          const obj = { top: 10 };
          obj[['left', 'right'][+(pos[0] < size.viewSize[0] / 2)]] = 30;
          return obj;
        },
        formatter: (params) => {
          // 添加安全检查
          if (!params || !params.length || !params[0].data || !params[0].data.value) {
            return '';
          }
  
          try {
            const data = params[0].data.value;
            const index = params[0].dataIndex;
            
            // 确保 index 有效且数据存在
            if (index === undefined || !priceHistory.value[index]) {
              return '';
            }
  
            const item = priceHistory.value[index];
            const color = data[1] >= data[0] ? '#07c160' : '#ff4d4f';
            
            return `
              <div style="padding: 3px;">
                <div style="color: #666;">日期：${item.Date}</div>
                <div style="color: ${color};">
                  开盘：${data[0].toFixed(6)}<br/>
                  收盘：${data[1].toFixed(6)}<br/>
                  最低：${data[2].toFixed(6)}<br/>
                  最高：${data[3].toFixed(6)}<br/>
                  涨跌幅：${item.ChangeRate}%
                </div>
              </div>
            `;
          } catch (error) {
            console.error('Tooltip formatter error:', error);
            return '';
          }
        }
      },
      grid: {
        left: '10%',
        right: '10%',
        bottom: '15%',
        top: '10%'
      },
      xAxis: {
        type: 'category',
        data: priceHistory.value.map(item => item.Date),
        scale: true,
        boundaryGap: true,
        axisLine: { lineStyle: { color: '#999' } },
        splitLine: { show: true, lineStyle: { color: '#eee' } },
        min: 'dataMin',
        max: 'dataMax',
        axisLabel: {
          show: true,
          formatter: (value) => value,
          interval: 'auto',
          rotate: 30  // 旋转标签以防重叠
        }
      },
      yAxis: {
        scale: true,
        position: 'right',
        splitLine: { show: true, lineStyle: { color: '#eee' } },
        axisLine: { lineStyle: { color: '#999' } },
        axisLabel: {
          formatter: (value) => value.toFixed(6),
          inside: true  // 将标签放在内侧
        }
      },
      dataZoom: [
        {
          type: 'inside',
          start: 0,
          end: 100,
          zoomLock: false,
          moveOnMouseWheel: true,
          zoomOnMouseWheel: true,
          preventDefaultMouseMove: true,
          rangeMode: ['value', 'value']
        },
        {
          show: true,
          type: 'slider',
          bottom: 0,
          start: 0,
          end: 100,
          rangeMode: ['value', 'value'],
          fillerColor: 'rgba(25,137,250,0.2)',
          handleIcon: 'M10.7,11.9v-1.3H9.3v1.3c-4.9,0.3-8.8,4.4-8.8,9.4c0,5,3.9,9.1,8.8,9.4v1.3h1.3v-1.3c4.9-0.3,8.8-4.4,8.8-9.4C19.5,16.3,15.6,12.2,10.7,11.9z M13.3,24.4H6.7V23h6.6V24.4z M13.3,19.6H6.7v-1.4h6.6V19.6z',
          handleSize: '80%',
          handleStyle: {
            color: '#1989fa',
            shadowBlur: 3,
            shadowColor: 'rgba(0, 0, 0, 0.2)',
            shadowOffsetX: 2,
            shadowOffsetY: 2
          },
          emphasis: {
            handleStyle: {
              color: '#1989fa'
            }
          }
        }
      ],
      series: [
        {
          type: 'candlestick',
          data: kLineData.map(item => item.value),
          itemStyle: {
            color: '#07c160',  // 上涨颜色
            color0: '#ff4d4f', // 下跌颜色
            borderColor: '#07c160',
            borderColor0: '#ff4d4f'
          },
          emphasis: {
            itemStyle: {
              shadowBlur: 5,
              shadowOffsetX: 0,
              shadowColor: 'rgba(0, 0, 0, 0.2)'
            }
          }
        }
      ]
    }
    
    // 创建新的事件处理函数
    const handleChartClick = (params) => {
      if (params.componentType === 'series' && params.data && params.data.value) {
        try {
          const data = params.data.value;
          const index = params.dataIndex;
          
          // 确保 index 有效且数据存在
          if (index === undefined || !priceHistory.value[index]) {
            return;
          }
  
          const item = priceHistory.value[index];
          showToast({
            message: `日期：${item.Date}\n开盘：${data[0].toFixed(6)}\n收盘：${data[1].toFixed(6)}\n涨跌幅：${item.ChangeRate}%`,
            duration: 3000,
            position: 'middle'
          });
        } catch (error) {
          console.error('Chart click handler error:', error);
        }
      }
    };
  
    // 修改第一个 handleDataZoom 函数的实现
    const handleDataZoom = (params) => {
      // 获取当前显示的数据范围
      const startValue = params.start
      const endValue = params.end
      
      // 计算需要显示的数据范围
      const totalDays = fullPriceHistory.value.length
      const startIndex = Math.floor(startValue * totalDays / 100)
      const endIndex = Math.ceil(endValue * totalDays / 100)
      
      // 如果拖动到了左边界，且还有更多历史数据可以加载
      if (startValue === 0 && !isLoadingMore.value) {
        loadMoreHistory()
      }
      
      // 更新显示的数据范围
      updateDisplayRange(startIndex, endIndex)
      
      // 更新缩放控件的状态
      chart.setOption({
        dataZoom: [{
          start: params.start,
          end: params.end
        }, {
          start: params.start,
          end: params.end
        }]
      })
    }
  
    // 加载更多历史数据
    const loadMoreHistory = async () => {
      if (isLoadingMore.value) return
      
      try {
        isLoadingMore.value = true
        // 计算需要加载的日期范围
        const oldestDate = fullPriceHistory.value[0].Date
        
        // 发起请求加载更早的数据
        const res = await httpGet('/api/trade/priceHistory', {
          params: {
            beforeDate: oldestDate,
            days: 7 // 每次加载7天
          }
        })
        
        if (res.code === 0 && Array.isArray(res.data) && res.data.length > 0) {
          // 将新数据添加到完整历史数据的前面
          fullPriceHistory.value = enrichSohaPriceHistoryChangeRates([...res.data, ...fullPriceHistory.value])
          
          // 更新图表数据
          chart.setOption({
            xAxis: {
              data: fullPriceHistory.value.map(item => item.Date)
            },
            series: [{
              data: transformData(fullPriceHistory.value)
            }]
          })
        }
      } catch (error) {
        console.error('加载更多历史数据失败:', error)
        showFailToast('加载更多历史数据失败')
      } finally {
        isLoadingMore.value = false
      }
    }
  
    // 更新显示范围
    const updateDisplayRange = (startIndex, endIndex) => {
      const displayData = fullPriceHistory.value.slice(startIndex, endIndex + 1)
      priceHistory.value = displayData
      
      chart.setOption({
        xAxis: {
          data: displayData.map(item => item.Date)
        },
        series: [{
          data: transformData(displayData)
        }]
      })
    }
  
    // 数据转换函数
    const transformData = (data) => {
      return data.map((item, index, arr) => {
        const prevItem = index > 0 ? arr[index - 1] : item
        const openPrice = prevItem.Price
        const closePrice = item.Price
        
        return {
          value: [
            openPrice,
            closePrice,
            Math.min(openPrice, closePrice) * 0.99,
            Math.max(openPrice, closePrice) * 1.01
          ],
          itemStyle: {
            color: closePrice >= openPrice ? '#07c160' : '#ff4d4f',
            borderColor: closePrice >= openPrice ? '#07c160' : '#ff4d4f'
          }
        }
      })
    }
  
    // 绑定缩放事件
    chart.off('datazoom')
    chart.on('datazoom', handleDataZoom)
  
    // 移除旧的事件监听器
    chart.off('click');
    chart.off('datazoom');
  
    // 添加新的事件监听器
    chart.on('click', handleChartClick);
    chart.on('datazoom', handleDataZoom);
  
    // 修改触摸事件处理
    let lastDistance = 0;
    let isDragging = false;
    
    const zr = chart.getZr();
    
    zr.off('touchstart');
    zr.off('touchmove');
    zr.off('touchend');
    
    zr.on('touchstart', (event) => {
      if (event.touches.length === 2) {
        lastDistance = Math.hypot(
          event.touches[0].clientX - event.touches[1].clientX,
          event.touches[0].clientY - event.touches[1].clientY
        );
        isDragging = true;
      }
    });
    
    zr.on('touchmove', (event) => {
      if (isDragging && event.touches.length === 2) {
        const newDistance = Math.hypot(
          event.touches[0].clientX - event.touches[1].clientX,
          event.touches[0].clientY - event.touches[1].clientY
        );
        const scale = newDistance / lastDistance;
        lastDistance = newDistance;
        
        const dataZoom = chart.getOption().dataZoom[0];
        const range = dataZoom.end - dataZoom.start;
        const center = (dataZoom.start + dataZoom.end) / 2;
        const newRange = range / scale;
        
        chart.dispatchAction({
          type: 'dataZoom',
          start: Math.max(0, center - newRange / 2),
          end: Math.min(100, center + newRange / 2)
        });
      }
    });
    
    zr.on('touchend', () => {
      isDragging = false;
    });
  
    // 设置图表配置
    chart.setOption(option);
    chartInstance.value = chart;
  
    // 添加 resize 事件监听
    const handleResize = () => {
      if (chart && !chart.isDisposed()) {
        chart.resize();
      }
    };
  
    window.addEventListener('resize', handleResize);
  
    // 在组件卸载时清理事件监听
    onUnmounted(() => {
      window.removeEventListener('resize', handleResize);
      if (chart && !chart.isDisposed()) {
        chart.dispose();
      }
    });
  }
  
  // 添加图表实例的引用
  const chartInstance = ref(null)
  
  // 添加视图切换的监听
  watch(currentView, (newView) => {
    if (newView === 'order') {
      nextTick(() => {
        // 确保在重新初始化图表前清理旧的实例
        if (chartInstance.value) {
          chartInstance.value.dispose();
        }
        initChart();
      });
    }
    if (newView === 'market') {
      activeList.value = 'buy';
    }
  });
  
  // 确保在数据更新时重新渲染图表
  watch(() => priceHistory.value, () => {
    if (currentView.value === 'order') {
      nextTick(() => {
        initChart();
      });
    }
  }, { deep: true });
  
  // 组件卸载时清理
  onUnmounted(() => {
    if (chartInstance.value) {
      chartInstance.value.dispose()
      window.removeEventListener('resize', () => {
        chartInstance.value && chartInstance.value.resize()
      })
    }
  })
  
  // 提交交易订单
  const handleSubmit = async () => {
    if (!amount.value || !price.value) {
      showToast('请输入完整的交易信息')
      return
    }
    
    // 获取当前指导价
    const guidePrice = currentPrice.value
    const inputPrice = Number(price.value)
    
    // 计算价格上下限
    const maxPrice = guidePrice * 2        // 最高价 = 当前价 * 2 (涨幅100%)
    const minPrice = guidePrice * 0.5      // 最低价 = 当前价 * 0.5 (跌幅50%)
    
    // 验证价格是否在合理区间
    if (inputPrice > maxPrice || inputPrice < minPrice) {
      showToast({
        message: `价格必须在 ${minPrice} - ${maxPrice} 之间`,
        duration: 5000
      })
      return
    }
  
    try {
      isSubmitting.value = true
      const types = currentTab.value === '买入SOHA' ? 1 : 2
  
      // 确保转换为数字类型
      const priceNum = inputPrice
      const amountNum = Number(amount.value)
      const payload = {
        price: priceNum,
        num: amountNum,
        types: types
      }
      
      console.log('Payload:', payload)
  
      // 使用 POST 方法提交交易数据
      const response = await httpPost('/api/trade/submitOrder', payload)
      
      if (response.code === 0) {
        showToast('交易提交成功')
        amount.value = ''
        refreshMarketData()
      } else {
        showToast(`交易失败: ${response.message}`)
      }
  
    } catch (error) {
      showToast(`交易失败: ${error.message}`)
    } finally {
      isSubmitting.value = false
    }
  }
  
  
  
  
  //匹配订单
  const matchSubmit = (item) => {
    showConfirmDialog({
      title: '订单匹配',
      message: '确定匹配交易吗?',
    }).then(() => {
      httpPost("/api/trade/powerPool/match", {
        id: item.id, //订单id
        types: item.types, // 如果是买单 则为1 如果是卖单则为2
        price: item.price,
        num: item.num
      }).then(res => {
        showSuccessToast("匹配成功")
        // 使用路由导航跳转到Trade页面
        router.push('/mobile/powertrade')
      }).catch(e => {
        showFailToast("匹配失败" + e.message)
      })
    }).catch(() => {
      // 用户取消操作
    });
  }
  
  
  
  
  const refreshMarketData = () => {
    initPriceData()

    httpGet('/api/user/profile').then(res => {
      availableBalance.value = res.data.coin - res.data.dj_coin
    }).catch((e) => {
      console.log(e.message)
    })

    httpGet('/api/trade/buyAndSellList').then(res => {
      if (res.code === 0 && Array.isArray(res.data)) {
        buyList.value = res.data.filter(item => item.Types === 1).map(item => ({
          id: item.Id,
          username: item.UserName,
          avatar: item.Avatar,
          amount: item.Number,
          price: item.Price,
          tradeCount: 0,
          rate: '0.00'
        }))

        sellList.value = res.data.filter(item => item.Types === 2).map(item => ({
          id: item.Id,
          username: item.UserName,
          avatar: item.Avatar,
          amount: item.Number,
          price: item.Price,
          tradeCount: 0,
          rate: '0.00'
        }))
      }
    }).catch((e) => {
      console.log(e.message)
    })

    httpGet('/api/trade/sumCoin').then(res => {
      if (res.code === 0 && res.data && res.data.total !== undefined) {
        volume.value = res.data.total
      }
    }).catch(() => {})
  }

  onMounted(() => {
    refreshMarketData()
  })
  
  // 在 setup 中获取 router 实例
  const router = useRouter()
  
  // 在 script setup 中添加一个计算属性
  const priceStyle = computed(() => {
    // 这里可以根据实际需求调整判断逻辑
    return priceChange.value >= 0 ? 'price-down' : 'price-up'
  })
  </script>
  
  <style lang="stylus" scoped>
  .market-container
    padding 16px
    background #f5f7fa
  
    .price-info
      background #fff
      padding 16px
      border-radius 8px
      margin-bottom 16px
  
      .current-price
        .label
          font-size 14px
          color #666
        .value
          font-size 24px
          font-weight bold
          transition color 0.3s ease
          
          &.price-up
            color #07c160  // 绿色
          
          &.price-down
            color #ff4d4f  // 红色
        .cny
          font-size 12px
          color #999
  
      .price-stats
        display flex
        justify-content space-between
        margin-top 16px
        
        .stat-item
          font-size 12px
          color #666
          
        .change
          .up
            color #07c160
          .down
            color #ff4d4f
  
    .price-chart
      background #fff
      padding 16px
      border-radius 8px
      margin-bottom 16px
  
      .chart-title
        font-size 20px
        font-weight bold
        margin-bottom 16px
  
        .sub-title
          font-size 12px
          color #999
          font-weight normal
  
      .chart-container
        height 200px
        background #fff
        border-radius 4px
  
    .trade-section
      margin 12px
      background #fff
      border-radius 12px
      padding 16px
      box-shadow 0 2px 8px rgba(0, 0, 0, 0.06)
  
      .tab-header
        margin-bottom 20px
  
        .tabs
          display flex
          background #f7f8fa
          border-radius 12px
          padding 4px
          
          .tab-item
            flex 1
            text-align center
            padding 12px
            font-size 15px
            color #666
            position relative
            border-radius 8px
            transition all 0.3s
            cursor pointer
            
            &.active
              color #1989fa
              font-weight 500
              background #fff
              box-shadow 0 2px 8px rgba(25, 137, 250, 0.1)
  
      .trade-form
        .custom-field
          background #f7f8fa
          border-radius 8px
          margin-bottom 12px
          
          :deep(.van-field__label)
            color #666
            font-size 14px
            
          :deep(.van-field__control)
            color #333
            font-size 15px
  
        .trade-info
          background #f7f8fa
          border-radius 8px
          padding 16px
          margin 16px 0
          
          .total-amount, .balance
            display flex
            justify-content space-between
            align-items center
            
            .label
              font-size 14px
              color #666
            
            .value
              font-size 16px
              font-weight 600
              color #333
          
          .total-amount
            margin-bottom 8px
            
            .value
              color #1989fa
  
        .submit-btn
          margin-top 20px
          margin-bottom 12px
          
          .van-button
            height 44px
            font-size 16px
            font-weight 500
            border-radius 22px
            background linear-gradient(135deg, #1989fa, #0066ff)
            border none
            
            &:active
              opacity 0.9
  
  .tabs
    .tab
      position relative
      transition all 0.3s
      
      &.active
        color #1989fa
        font-weight bold
        
        &:after
          content ''
          position absolute
          bottom -1px
          left 0
          width 100%
          height 2px
          background #1989fa
  
  .trade-form
    .van-field
      margin-bottom 12px
      
    .van-button
      height 44px
      font-size 16px
  
  .order-lists
    margin-top 16px
    background #fff
    border-radius 8px
    
    .search-bar
      padding 8px
  
    .list-tabs
      display flex
      background #fff
      border-radius 12px
      padding 3px
      border 1px solid #ebedf0
      margin-bottom 20px
      
      .tab-item
        flex 1
        text-align center
        padding 10px 0
        font-size 14px
        color #666
        position relative
        transition all 0.3s ease
        
        &.active
          font-weight 500
          background #f0f7ff
          border-radius 10px
          box-shadow 0 2px 6px rgba(25, 137, 250, 0.1)
          
          // 购买按钮激活状态使用绿色
          &:first-child
            color #07c160
            background #f0fff4
            box-shadow 0 2px 6px rgba(7, 193, 96, 0.1)
          
          // 出售按钮激活状态保持蓝色
          &:last-child
            color #1989fa
            background #f0f7ff
            box-shadow 0 2px 6px rgba(25, 137, 250, 0.1)
  
    .order-list
      padding 8px 0
      
      .order-item
        margin 8px 12px
        padding 16px
        background #ffffff
        border-radius 12px
        box-shadow 0 2px 8px rgba(0, 0, 0, 0.06)
        transition all 0.3s ease
        
        &:hover
          transform translateY(-2px)
          box-shadow 0 4px 12px rgba(0, 0, 0, 0.08)
        
        .user-info
          display flex
          align-items center
          margin-bottom 12px
          
          .avatar
            width 36px
            height 36px
            border-radius 50%
            object-fit cover
            border 2px solid #f0f2f5
          
          .username
            margin-left 12px
            font-size 15px
            font-weight 500
            color #333
        
        .order-details
          flex 1
          
          .price-info
            .main-info
              display flex
              justify-content space-between
              align-items center
              margin-bottom 10px
              
              .amount, .price
                .label
                  font-size 13px
                  color #666
                  margin-right 6px
                
                .q2-text
                  color #00b578
                  font-weight 600
                  font-size 16px
                  background rgba(0, 181, 120, 0.1)
                  padding 2px 8px
                  border-radius 4px
                
                .value
                  color #ff4d4f
                  font-weight 600
                  font-size 16px
                  background rgba(255, 77, 79, 0.1)
                  padding 2px 8px
                  border-radius 4px
            
            .total-info
              text-align right
              padding-top 8px
              border-top 1px dashed #eee
              margin-top 8px
              
              .label
                font-size 13px
                color #666
              
              .total-value
                color #333
                font-weight 600
                font-size 16px
                margin-left 6px
      
        .trade-btn
          margin-top 12px
          width 100%
          height 36px
          font-size 14px
          font-weight 500
          border-radius 18px
          background linear-gradient(135deg, #1989fa, #0066ff)
          border none
          
          &:active
            opacity 0.9
  
  // 顶部主导航按钮样式
  .top-buttons
    display flex
    background #f7f8fa
    padding 4px
    margin 0 12px 16px
    border-radius 16px
    position relative
    box-shadow 0 2px 12px rgba(100, 100, 100, 0.1)
  
    .button-item
      flex 1
      text-align center
      padding 12px 0
      font-size 15px
      color #666
      cursor pointer
      border-radius 12px
      transition all 0.3s ease
      position relative
      z-index 1
  
      &.active
        color #fff
        font-weight 500
  
    // 背景滑块
    &:before
      content ''
      position absolute
      width 33.33% // 平均分配宽度
      height 85%
      background linear-gradient(135deg, #1989fa, #0066ff)
      border-radius 12px
      transition all 0.3s ease
      top 50%
      transform translateY(-50%)
      z-index 0
      box-shadow 0 2px 8px rgba(25, 137, 250, 0.3)
  
    // 根据当前选中项移动背景滑块
    &[data-active="order"]:before
      left 2%
    
    &[data-active="market"]:before
      left 33.33%
    
    &[data-active="trade"]:before
      left 66.66%
  
  // 级买卖按钮样式
  .trade-section
    .tab-header
      .tabs
        display flex
        background #fff
        border-radius 12px
        padding 3px
        border 1px solid #ebedf0
        margin-bottom 20px
        
        .tab-item
          flex 1
          text-align center
          padding 10px 0
          font-size 14px
          color #666
          position relative
          transition all 0.3s ease
          
          &.active
            color #1989fa
            font-weight 500
            background #f0f7ff
            border-radius 10px
            box-shadow 0 2px 6px rgba(25, 137, 250, 0.1)
  
  .trade-btn
    &.van-button--primary
      // 在购买列表中的按钮使用绿色
      .order-lists[data-active="sell"] &
        background linear-gradient(135deg, #07c160, #06ad56)
        
      // 在出售表中的按钮保持蓝色
      .order-lists[data-active="buy"] &
        background linear-gradient(135deg, #1989fa, #0066ff)
  
  .trade-section
        .submit-btn
          .van-button
            &.van-button--primary
              background linear-gradient(135deg, #1989fa, #0066ff) !important
              
              &.buy-button
                background linear-gradient(135deg, #07c160, #06ad56) !important
  </style>
  