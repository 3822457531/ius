<template>
  <div 
    class="tree-node-wrapper" 
    :class="{ 'highlight': isHighlighted, 'dragging': isDragging }"
    :data-node-id="node.user_id"
    :style="{ transform: `translate(${position.x}px, ${position.y}px)` }"
    draggable="true"
    @dragstart="handleDragStart"
    @dragend="handleDragEnd"
    @dragover.prevent="handleDragOver"
    @drop.prevent="handleDrop"
  >
    <div class="node-container">
      <!-- 节点卡片 -->
      <div class="node-card" :class="{ 'root-node': level === 0, 'matched': isMatched, 'dragging': isDragging }">
        <!-- 节点头部 -->
        <div class="node-header">
          <div class="user-badge">
            <span class="node-id">ID: {{ node.user_id }}</span>
          </div>
          <div v-if="node.user_info && node.user_info.username" class="node-username">
            {{ node.user_info.username || node.user_info.nickname || '未命名' }}
          </div>
          <!-- 折叠/展开按钮 -->
          <div v-if="hasChildren" class="collapse-btn" @click.stop="toggleCollapse">
            <el-icon :class="{ 'rotated': isCollapsed }">
              <ArrowDown />
            </el-icon>
          </div>
        </div>
        
        <!-- 业绩信息 -->
        <div class="node-info">
          <div class="perf-info">
            <div class="perf-item left">
              <div class="perf-icon left-icon">
                <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M9 18L3 12L9 6M15 6L21 12L15 18" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
              </div>
              <div class="perf-content">
                <span class="label">左区</span>
                <span class="value">{{ formatNumber(node.left_perf) }}</span>
              </div>
            </div>
            <div class="perf-item right">
              <div class="perf-icon right-icon">
                <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M15 18L21 12L15 6M9 6L3 12L9 18" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
              </div>
              <div class="perf-content">
                <span class="label">右区</span>
                <span class="value">{{ formatNumber(node.right_perf) }}</span>
              </div>
            </div>
          </div>

          <!-- 自己的充值 --> 
          <div class="recharge-amount">
            <span class="recharge-label">充值量</span>
            <span class="recharge-value">{{ formatNumber(node.recharge_amount || 0) }}</span>
          </div>

          <div class="total-perf">
            <span class="total-label">总业绩</span>
            <span class="total-value">{{ formatNumber(node.total_perf) }}</span>
          </div>
       
        </div>
      </div>
      
      <!-- 连接线容器 -->
      <div class="children-container" v-if="hasChildren && !isCollapsed">
        <!-- 垂直箭头线 -->
        <div class="vertical-arrow">
          <svg class="arrow-svg" viewBox="0 0 100 30" preserveAspectRatio="none">
            <line x1="50" y1="0" x2="50" y2="30" stroke="#dc2626" stroke-width="2.5" />
            <polygon points="45,25 50,30 55,25" fill="#dc2626" />
          </svg>
        </div>
        <div class="children-wrapper">
          <!-- 左子节点 -->
          <div class="child-node left-child" v-if="leftChild">
            <div class="connector-arrow left-arrow">
              <svg class="arrow-svg" viewBox="0 0 200 30" preserveAspectRatio="none">
                <defs>
                  <marker :id="`arrowhead-left-${node.user_id}`" markerWidth="10" markerHeight="10" refX="5" refY="5" orient="auto">
                    <polygon points="10,0 0,5 10,10" fill="#dc2626" />
                  </marker>
                </defs>
                <path d="M 0 15 L 100 15 L 100 0" stroke="#dc2626" stroke-width="2.5" fill="none" :marker-end="`url(#arrowhead-left-${node.user_id})`" />
                <line x1="100" y1="0" x2="100" y2="30" stroke="#dc2626" stroke-width="2.5" />
              </svg>
              <!-- 左区文字标签（不受翻转影响） -->
              <div class="arrow-label left-label">左区</div>
            </div>
            <tree-node
              :node="leftChild"
              :all-nodes="allNodes"
              :search-keyword="searchKeyword"
              :level="level + 1"
              :collapsed-nodes="collapsedNodes"
              @toggle-collapse="emit('toggle-collapse', $event)"
            />
          </div>
          
          <!-- 右子节点 -->
          <div class="child-node right-child" v-if="rightChild">
            <div class="connector-arrow right-arrow">
              <svg class="arrow-svg" viewBox="0 0 200 30" preserveAspectRatio="none">
                <defs>
                  <marker :id="`arrowhead-right-${node.user_id}`" markerWidth="10" markerHeight="10" refX="5" refY="5" orient="auto">
                    <polygon points="0,0 10,5 0,10" fill="#dc2626" />
                  </marker>
                </defs>
                <path d="M 200 15 L 100 15 L 100 0" stroke="#dc2626" stroke-width="2.5" fill="none" :marker-end="`url(#arrowhead-right-${node.user_id})`" />
                <line x1="100" y1="0" x2="100" y2="30" stroke="#dc2626" stroke-width="2.5" />
              </svg>
              <!-- 右区文字标签 -->
              <div class="arrow-label right-label">右区</div>
            </div>
            <tree-node
              :node="rightChild"
              :all-nodes="allNodes"
              :search-keyword="searchKeyword"
              :level="level + 1"
              :collapsed-nodes="collapsedNodes"
              @toggle-collapse="emit('toggle-collapse', $event)"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, reactive, onMounted } from 'vue'
import { ArrowDown } from '@element-plus/icons-vue'

const props = defineProps({
  node: {
    type: Object,
    required: true
  },
  allNodes: {
    type: Object,
    required: true
  },
  searchKeyword: {
    type: String,
    default: ''
  },
  level: {
    type: Number,
    default: 0
  },
  collapsedNodes: {
    type: Set,
    default: () => new Set()
  }
})

const emit = defineEmits(['toggle-collapse'])

// 节点位置（用于拖拽）
const position = reactive({ x: 0, y: 0 })
const isDragging = ref(false)
const dragOffset = reactive({ x: 0, y: 0 })

// 折叠状态
const isCollapsed = computed(() => {
  return props.collapsedNodes.has(props.node.user_id)
})

// 格式化数字
const formatNumber = (num) => {
  if (!num && num !== 0) return '0'
  return Number(num).toLocaleString('zh-CN', { 
    minimumFractionDigits: 0, 
    maximumFractionDigits: 2 
  })
}

// 获取左子节点
const leftChild = computed(() => {
  if (!props.node.left_child_id) return null
  return props.allNodes[props.node.left_child_id] || null
})

// 获取右子节点
const rightChild = computed(() => {
  if (!props.node.right_child_id) return null
  return props.allNodes[props.node.right_child_id] || null
})

// 是否有子节点
const hasChildren = computed(() => {
  return leftChild.value || rightChild.value
})

// 是否匹配搜索关键词
const isMatched = computed(() => {
  if (!props.searchKeyword) return false
  const keyword = props.searchKeyword.toLowerCase()
  const userId = String(props.node.user_id)
  const username = props.node.user_info?.username?.toLowerCase() || ''
  const nickname = props.node.user_info?.nickname?.toLowerCase() || ''
  return userId.includes(keyword) || username.includes(keyword) || nickname.includes(keyword)
})

// 是否高亮（匹配搜索或子节点匹配）
const isHighlighted = computed(() => {
  return isMatched.value
})

// 切换折叠状态
const toggleCollapse = () => {
  emit('toggle-collapse', props.node.user_id)
}

// 拖拽开始
const handleDragStart = (e) => {
  isDragging.value = true
  const rect = e.currentTarget.getBoundingClientRect()
  dragOffset.x = e.clientX - rect.left
  dragOffset.y = e.clientY - rect.top
  e.dataTransfer.effectAllowed = 'move'
  e.dataTransfer.setData('text/plain', props.node.user_id.toString())
  e.dataTransfer.setData('application/json', JSON.stringify({
    nodeId: props.node.user_id,
    offsetX: dragOffset.x,
    offsetY: dragOffset.y
  }))
  
  // 添加拖拽样式
  e.currentTarget.style.opacity = '0.6'
  e.currentTarget.style.cursor = 'grabbing'
}

// 拖拽结束
const handleDragEnd = (e) => {
  isDragging.value = false
  e.currentTarget.style.opacity = '1'
  e.currentTarget.style.cursor = 'grab'
  
  // 重置位置（如果需要的话，可以保存位置）
  // position.x = 0
  // position.y = 0
}

// 拖拽悬停
const handleDragOver = (e) => {
  e.preventDefault()
  e.dataTransfer.dropEffect = 'move'
}

// 拖拽放置
const handleDrop = (e) => {
  e.preventDefault()
  e.stopPropagation()
  
  try {
    const data = JSON.parse(e.dataTransfer.getData('application/json'))
    const draggedNodeId = data.nodeId
    
    if (draggedNodeId !== props.node.user_id) {
      // 这里可以添加节点移动的逻辑
      console.log(`节点 ${draggedNodeId} 被拖拽到节点 ${props.node.user_id}`)
      // 可以触发事件通知父组件进行节点关系更新
    }
  } catch (err) {
    console.error('拖拽数据解析失败:', err)
  }
}
</script>

<style lang="stylus" scoped>
.tree-node-wrapper {
  display: inline-block;
  vertical-align: top;
  margin: 15px;
  position: relative;
  cursor: grab;
  transition: transform 0.2s ease-out, opacity 0.2s;
  user-select: none;
  
  &:active {
    cursor: grabbing;
  }
  
  &.dragging {
    z-index: 1000;
    transition: none;
  }
  
  &.highlight {
    .node-card {
      box-shadow: 0 0 20px rgba(64, 158, 255, 0.6);
      border-color: #409eff;
      transform: scale(1.05);
    }
  }
  
  .node-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    position: relative;
    
    .node-card {
      background: #ffffff;
      border: 2px solid #e4e7ed;
      border-radius: 12px;
      padding: 16px;
      min-width: 200px;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
      transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
      position: relative;
      overflow: hidden;
      
      &::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        height: 3px;
        background: linear-gradient(90deg, #409eff, #67c23a);
        opacity: 0;
        transition: opacity 0.3s;
      }
      
      &:hover {
        transform: translateY(-2px);
        box-shadow: 0 8px 20px rgba(0, 0, 0, 0.12);
        border-color: #c0c4cc;
        
        &::before {
          opacity: 1;
        }
      }
      
      &.root-node {
        border: none;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: #fff;
        box-shadow: 0 8px 24px rgba(102, 126, 234, 0.4);
        
        &::before {
          display: none;
        }
        
        .node-header {
          .user-badge {
            background: rgba(255, 255, 255, 0.2);
            border-color: rgba(255, 255, 255, 0.3);
            
            .node-id {
              color: #fff;
            }
          }
          
          .node-username {
            color: rgba(255, 255, 255, 0.95);
          }
        }
        
        .node-info {
          .perf-item {
            .perf-icon {
              background: rgba(255, 255, 255, 0.15);
              color: #fff;
            }
            
            .perf-content {
              .label {
                color: rgba(255, 255, 255, 0.8);
              }
              
              .value {
                color: #fff;
              }
            }
            
            &.left .perf-content .value {
              color: #a8d8ff;
            }
            
            &.right .perf-content .value {
              color: #b8f5b8;
            }
          }
          
          .total-perf {
            border-top-color: rgba(255, 255, 255, 0.2);
            
            .total-label {
              color: rgba(255, 255, 255, 0.8);
            }
            
            .total-value {
              color: #fff;
            }
          }
          
          .recharge-amount {
            border-top-color: rgba(255, 255, 255, 0.2);
            
            .recharge-label {
              color: rgba(255, 255, 255, 0.8);
            }
            
            .recharge-value {
              color: #ffd700;
            }
          }
        }
      }
      
      &.matched {
        border-color: #67c23a;
        background: linear-gradient(135deg, #f0f9ff 0%, #e8f5e9 100%);
        animation: pulse 2s infinite;
        
        &::before {
          background: linear-gradient(90deg, #67c23a, #95d475);
          opacity: 1;
        }
      }
      
      &.dragging {
        opacity: 0.6;
        cursor: grabbing;
        z-index: 1000;
      }
      
      .node-header {
        display: flex;
        flex-direction: column;
        align-items: center;
        margin-bottom: 12px;
        padding-bottom: 12px;
        border-bottom: 1px solid #f0f0f0;
        
        .user-badge {
          background: linear-gradient(135deg, #f5f7fa 0%, #e9ecef 100%);
          border: 1px solid #e4e7ed;
          border-radius: 20px;
          padding: 6px 14px;
          margin-bottom: 8px;
          transition: all 0.3s;
          
          .node-id {
            font-weight: 600;
            font-size: 13px;
            color: #303133;
            letter-spacing: 0.5px;
          }
        }
        
        .node-username {
          font-size: 13px;
          color: #606266;
          font-weight: 500;
          max-width: 100%;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
        
        .collapse-btn {
          position: absolute;
          top: 8px;
          right: 8px;
          width: 24px;
          height: 24px;
          display: flex;
          align-items: center;
          justify-content: center;
          background: rgba(64, 158, 255, 0.1);
          border-radius: 50%;
          cursor: pointer;
          transition: all 0.3s;
          z-index: 10;
          
          &:hover {
            background: rgba(64, 158, 255, 0.2);
            transform: scale(1.1);
          }
          
          .el-icon {
            font-size: 14px;
            color: #409eff;
            transition: transform 0.3s;
            
            &.rotated {
              transform: rotate(-90deg);
            }
          }
        }
      }
      
      .node-info {
        .perf-info {
          display: flex;
          justify-content: space-between;
          gap: 12px;
          margin-bottom: 12px;
          
          .perf-item {
            display: flex;
            align-items: center;
            gap: 8px;
            flex: 1;
            padding: 8px;
            background: #f8f9fa;
            border-radius: 8px;
            transition: all 0.3s;
            
            &:hover {
              background: #f0f2f5;
              transform: scale(1.02);
            }
            
            .perf-icon {
              width: 28px;
              height: 28px;
              border-radius: 6px;
              display: flex;
              align-items: center;
              justify-content: center;
              flex-shrink: 0;
              
              svg {
                width: 16px;
                height: 16px;
              }
              
              &.left-icon {
                background: linear-gradient(135deg, #e3f2fd 0%, #bbdefb 100%);
                color: #1976d2;
              }
              
              &.right-icon {
                background: linear-gradient(135deg, #e8f5e9 0%, #c8e6c9 100%);
                color: #388e3c;
              }
            }
            
            .perf-content {
              display: flex;
              flex-direction: column;
              flex: 1;
              min-width: 0;
              
              .label {
                font-size: 11px;
                color: #909399;
                margin-bottom: 2px;
                font-weight: 500;
              }
              
              .value {
                font-size: 15px;
                font-weight: 700;
                color: #303133;
                line-height: 1.2;
              }
            }
            
            &.left .perf-content .value {
              color: #1976d2;
            }
            
            &.right .perf-content .value {
              color: #388e3c;
            }
          }
        }
        
        .total-perf {
          text-align: center;
          padding: 10px;
          background: linear-gradient(135deg, #fafafa 0%, #f5f5f5 100%);
          border-radius: 8px;
          border-top: 1px solid #e4e7ed;
          margin-top: 8px;
          display: flex;
          justify-content: space-between;
          align-items: center;
          
          .total-label {
            font-size: 12px;
            color: #909399;
            font-weight: 500;
          }
          
          .total-value {
            font-size: 16px;
            font-weight: 700;
            color: #303133;
          }
        }
        
        .recharge-amount {
          text-align: center;
          padding: 10px;
          background: linear-gradient(135deg, #fff5e6 0%, #ffe8cc 100%);
          border-radius: 8px;
          border-top: 1px solid #e4e7ed;
          margin-top: 8px;
          display: flex;
          justify-content: space-between;
          align-items: center;
          transition: all 0.3s;
          
          &:hover {
            background: linear-gradient(135deg, #ffe8cc 0%, #ffd9b3 100%);
          }
          
          .recharge-label {
            font-size: 12px;
            color: #909399;
            font-weight: 500;
          }
          
          .recharge-value {
            font-size: 16px;
            font-weight: 700;
            color: #e6a23c;
          }
        }
      }
    }
    
    .children-container {
      margin-top: 25px;
      position: relative;
      width: 100%;
      display: flex;
      flex-direction: column;
      align-items: center;
      
      .vertical-arrow {
        width: 2px;
        height: 30px;
        margin-bottom: 0;
        position: relative;
        
        .arrow-svg {
          width: 100%;
          height: 100%;
          display: block;
        }
      }
      
      .children-wrapper {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        position: relative;
        width: 100%;
        min-width: 450px;
        gap: 20px;
        
        .child-node {
          position: relative;
          display: flex;
          flex-direction: column;
          align-items: center;
          flex: 1;
          min-width: 0;
          
          .connector-arrow {
            width: 100%;
            height: 30px;
            margin-bottom: 0;
            position: relative;
            overflow: visible;
            
            .arrow-svg {
              width: 100%;
              height: 100%;
              display: block;
            }
            
            .arrow-label {
              position: absolute;
              top: 50%;
              transform: translateY(-50%);
              background: white;
              color: #dc2626;
              font-size: 11px;
              font-weight: bold;
              padding: 2px 6px;
              border-radius: 3px;
              border: 1px solid #dc2626;
              white-space: nowrap;
              z-index: 10;
              pointer-events: none;
            }
            
            &.left-arrow {
              padding-right: 50%;
              
              .arrow-svg {
                transform: scaleX(-1);
              }
              
              .left-label {
                right: 25%;
              }
            }
            
            &.right-arrow {
              padding-left: 50%;
              
              .right-label {
                left: 25%;
              }
            }
          }
          
          &.left-child {
            align-items: flex-end;
          }
          
          &.right-child {
            align-items: flex-start;
          }
        }
      }
    }
  }
}

@keyframes pulse {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(103, 194, 58, 0.4);
  }
  50% {
    box-shadow: 0 0 0 10px rgba(103, 194, 58, 0);
  }
}

// 响应式设计
@media (max-width: 768px) {
  .tree-node-wrapper {
    margin: 10px 5px;
    
    .node-container {
      .node-card {
        min-width: 160px;
        padding: 12px;
      }
      
      .children-container {
        .children-wrapper {
          min-width: 320px;
          flex-direction: column;
          gap: 15px;
          
          .child-node {
            width: 100%;
            
            .connector-arrow {
              display: none;
            }
          }
        }
      }
    }
  }
}
</style>

