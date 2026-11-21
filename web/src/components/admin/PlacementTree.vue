<template>
  <div class="placement-tree-container" :class="{ 'fullscreen-mode': isFullscreen }">
    <div class="tree-header">
      <div class="header-left">
        <h3 class="page-title">
          <el-icon class="title-icon"><Connection /></el-icon>
          安置树结构图
        </h3>
        <el-tag type="info" size="small" v-if="treeData.length > 0">
          共 {{ treeData.length }} 个节点
        </el-tag>
        <el-tag type="success" size="small" v-if="visibleNodeCount > 0">
          显示 {{ visibleNodeCount }} 个节点
        </el-tag>
      </div>
      <div class="header-actions">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索用户ID、用户名或昵称"
          style="width: 280px;"
          clearable
          @input="handleSearch"
          size="default"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button-group>
          <el-button 
            type="primary" 
            @click="toggleFullscreen"
            :icon="FullScreen"
            :title="isFullscreen ? '退出全屏' : '全屏显示'"
          >
            {{ isFullscreen ? '退出全屏' : '全屏' }}
          </el-button>
          <el-button 
            type="primary" 
            @click="resetView"
            title="重置视图"
          >
            重置
          </el-button>
        <el-button 
          type="primary" 
          @click="loadTreeData" 
          :loading="loading"
          :icon="Refresh"
        >
          刷新数据
        </el-button>
        </el-button-group>
      </div>
    </div>
    
    <div class="tree-content" v-loading="loading">
      <div v-if="treeData.length === 0 && !loading" class="empty-state">
        <el-empty description="暂无节点数据">
          <el-button type="primary" @click="loadTreeData">立即加载</el-button>
        </el-empty>
      </div>
      
      <div v-else class="tree-visualization-wrapper">
        <!-- 工具栏 -->
        <div class="toolbar">
          <div class="zoom-controls">
            <el-button size="small" @click="zoomOut" :icon="ZoomOut" circle />
            <span class="zoom-level">{{ Math.round(zoom * 100) }}%</span>
            <el-button size="small" @click="zoomIn" :icon="ZoomIn" circle />
            <el-button
              size="small"
              type="primary"
              @click="organizeView"
            >
              整理
            </el-button>
          </div>
        </div>
        
        <!-- 树形可视化区域 -->
        <div 
          class="tree-visualization" 
          ref="treeContainer"
          @wheel.prevent="handleWheel"
          @mousedown="handleMouseDown"
          @mousemove="handleMouseMove"
          @mouseup="handleMouseUp"
          @mouseleave="handleMouseUp"
        >
          <!-- SVG 连接线层 -->
          <svg 
            class="connections-layer" 
            ref="svgLayer"
            :width="canvasWidth" 
            :height="canvasHeight"
            :style="{
              transform: `translate(${panX}px, ${panY}px) scale(${zoom})`,
              transformOrigin: '0 0'
            }"
          >
            <defs>
              <marker 
                id="arrowhead-left" 
                :markerWidth="arrowSize" 
                :markerHeight="arrowSize" 
                :refX="arrowSize * 0.9" 
                :refY="arrowSize * 0.5" 
                orient="auto"
              >
                <polygon :points="`0,0 ${arrowSize},${arrowSize * 0.5} 0,${arrowSize}`" fill="#dc2626" />
              </marker>
              <marker 
                id="arrowhead-right" 
                :markerWidth="arrowSize" 
                :markerHeight="arrowSize" 
                :refX="arrowSize * 0.1" 
                :refY="arrowSize * 0.5" 
                orient="auto"
              >
                <polygon :points="`${arrowSize},0 0,${arrowSize * 0.5} ${arrowSize},${arrowSize}`" fill="#dc2626" />
              </marker>
            </defs>
            <!-- 连接线 -->
            <g v-for="connection in connections" :key="connection.id">
              <path
                :d="connection.path"
                stroke="#dc2626"
                :stroke-width="Math.max(0.5, 2 * zoom)"
                fill="none"
                :marker-end="connection.marker"
                class="connection-line"
              />
              <!-- 连接线标签 -->
              <text
                :x="connection.labelX"
                :y="connection.labelY"
                class="connection-label"
                fill="#dc2626"
                :font-size="Math.max(8, 11 * zoom)"
                font-weight="bold"
                text-anchor="middle"
              >
                {{ connection.label }}
              </text>
            </g>
          </svg>
          
          <!-- 节点层 -->
          <div 
            class="nodes-layer"
            :style="{
              transform: `translate(${panX}px, ${panY}px) scale(${zoom})`,
              transformOrigin: '0 0'
            }"
          >
            <div
              v-for="node in visibleNodes"
              :key="node.user_id"
              :ref="el => setNodeRef(el, node.user_id)"
              class="tree-node"
              :class="{
                'root-node': node.isRoot,
                'matched': isNodeMatched(node),
                'collapsed': collapsedNodes.has(node.user_id)
              }"
              :style="{
                left: node.x + 'px',
                top: node.y + 'px'
              }"
              @mousedown.stop="startNodeDrag(node, $event)"
              @click.stop="handleNodeClick(node)"
            >
              <div class="node-card">
                <div class="node-header">
                  <div class="user-badge">
                    <span class="node-id">ID: {{ node.user_id }}</span>
                  </div>
                  <div class="node-username">
                    {{ node.user_info?.username || node.user_info?.nickname || '未命名' }}
                  </div>
                  <div 
                    v-if="hasChildren(node)" 
                    class="collapse-btn" 
                    @click.stop="toggleCollapse(node.user_id)"
                  >
                    <el-icon :class="{ 'rotated': collapsedNodes.has(node.user_id) }">
                      <ArrowDown />
                    </el-icon>
                  </div>
                </div>
                
                <div class="node-info">
                  <div class="perf-info">
                    <div class="perf-item left">
                      <span class="perf-label">左区</span>
                      <span class="perf-value">{{ formatNumber(node.left_perf) }}</span>
                    </div>
                    <div class="perf-item right">
                      <span class="perf-label">右区</span>
                      <span class="perf-value">{{ formatNumber(node.right_perf) }}</span>
                    </div>
                  </div>
                  
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
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, nextTick } from 'vue'
import { httpGet } from '@/utils/http'
import { ElMessage } from 'element-plus'
import { Search, Refresh, Connection, ZoomIn, ZoomOut, FullScreen, ArrowDown } from '@element-plus/icons-vue'

const loading = ref(false)
const treeData = ref([])
const searchKeyword = ref('')
const nodeMap = ref({})
const collapsedNodes = ref(new Set())
const treeContainer = ref(null)
const svgLayer = ref(null)

// 画布状态
const panX = ref(0)
const panY = ref(0)
const zoom = ref(1)
const MIN_ZOOM = 0.02
const MAX_ZOOM = 3
const isDragging = ref(false)
const isNodeDragging = ref(false)
const dragStart = ref({ x: 0, y: 0 })
const draggedNode = ref(null)
const nodeRefs = ref({})
const isFullscreen = ref(false)

// 节点布局
// 单个节点卡片的基础尺寸
// 可以根据需要调整整体缩放感（比如 160/150 会更小）
const nodeWidth = 170
const nodeHeight = 160
const horizontalSpacing = 250
const verticalSpacing = 220

// 计算属性
const rootNodes = computed(() => {
  return treeData.value.filter(node => node.parent_id === 0 || !node.parent_id)
})

const visibleNodeCount = computed(() => {
  return visibleNodes.value.length
})

// 布局后的节点
const layoutedNodes = ref([])

// 箭头标记大小（根据缩放调整）
const arrowSize = computed(() => {
  return Math.max(5, 10 * zoom.value)
})

// 可见节点（根据折叠状态和搜索过滤）
const visibleNodes = computed(() => {
  return layoutedNodes.value.filter(node => {
    // 检查节点是否应该显示（考虑折叠状态）
    if (!isNodeVisible(node)) {
      return false
    }
    // 搜索过滤
    if (searchKeyword.value) {
      return isNodeMatched(node)
    }
    return true
  })
})

// 连接线
const connections = computed(() => {
  const conns = []
  visibleNodes.value.forEach(node => {
    if (collapsedNodes.value.has(node.user_id)) return
    
    // 左子节点连接
    if (node.left_child_id) {
      const child = nodeIndex[node.left_child_id]
      if (child && isNodeVisible(child)) {
        const parentCenter = { x: node.x + nodeWidth / 2, y: node.y + nodeHeight }
        const childCenter = { x: child.x + nodeWidth / 2, y: child.y }
        const midY = parentCenter.y + (childCenter.y - parentCenter.y) / 2
        
        const path = `M ${parentCenter.x} ${parentCenter.y} 
                     L ${parentCenter.x} ${midY} 
                     L ${childCenter.x} ${midY} 
                     L ${childCenter.x} ${childCenter.y}`
        
        conns.push({
          id: `conn-${node.user_id}-left`,
          path,
          marker: 'url(#arrowhead-right)',
          label: '左区',
          labelX: parentCenter.x - 30,
          labelY: midY - 5
        })
      }
    }
    
    // 右子节点连接
    if (node.right_child_id) {
      const child = nodeIndex[node.right_child_id]
      if (child && isNodeVisible(child)) {
        const parentCenter = { x: node.x + nodeWidth / 2, y: node.y + nodeHeight }
        const childCenter = { x: child.x + nodeWidth / 2, y: child.y }
        const midY = parentCenter.y + (childCenter.y - parentCenter.y) / 2
        
        const path = `M ${parentCenter.x} ${parentCenter.y} 
                     L ${parentCenter.x} ${midY} 
                     L ${childCenter.x} ${midY} 
                     L ${childCenter.x} ${childCenter.y}`
        
        conns.push({
          id: `conn-${node.user_id}-right`,
          path,
          marker: 'url(#arrowhead-left)',
          label: '右区',
          labelX: parentCenter.x + 30,
          labelY: midY - 5
        })
      }
    }
  })
  return conns
})

// 画布尺寸（用循环求极值，避免大数组 spread 造成调用栈溢出）
const canvasWidth = computed(() => {
  const nodes = layoutedNodes.value
  if (nodes.length === 0) return 2000
  let maxX = 0
  for (let i = 0; i < nodes.length; i++) {
    const v = nodes[i].x + nodeWidth
    if (v > maxX) maxX = v
  }
  return Math.max(2000, maxX + 200)
})

const canvasHeight = computed(() => {
  const nodes = layoutedNodes.value
  if (nodes.length === 0) return 2000
  let maxY = 0
  for (let i = 0; i < nodes.length; i++) {
    const v = nodes[i].y + nodeHeight
    if (v > maxY) maxY = v
  }
  return Math.max(2000, maxY + 200)
})

// 检查节点是否可见（沿 parentIndex 上溯，O(深度)）
const isNodeVisible = (node) => {
  let parentId = parentIndex[node.user_id]
  while (parentId && parentId !== node.user_id) {
    if (collapsedNodes.value.has(parentId)) return false
    parentId = parentIndex[parentId]
  }
  return true
}

// 检查节点是否匹配搜索
const isNodeMatched = (node) => {
  if (!searchKeyword.value) return true
  const keyword = searchKeyword.value.toLowerCase()
  const userId = String(node.user_id)
  const username = node.user_info?.username?.toLowerCase() || ''
  const nickname = node.user_info?.nickname?.toLowerCase() || ''
  return userId.includes(keyword) || username.includes(keyword) || nickname.includes(keyword)
}

// 检查节点是否有子节点
const hasChildren = (node) => {
  return !!(node.left_child_id || node.right_child_id)
}

// 格式化数字
const formatNumber = (num) => {
  if (!num && num !== 0) return '0'
  return Number(num).toLocaleString('zh-CN', { 
    minimumFractionDigits: 0, 
    maximumFractionDigits: 2 
  })
}

// 设置节点引用
const setNodeRef = (el, nodeId) => {
  if (el) {
    nodeRefs.value[nodeId] = el
  }
}

// 布局索引：每次 calculateLayout 后重建，替代数组 find 的 O(n²) 查找
// nodeIndex: user_id → 节点对象；parentIndex: 子节点 user_id → 父节点 user_id
let nodeIndex = {}
let parentIndex = {}

const rebuildIndexes = () => {
  nodeIndex = {}
  parentIndex = {}
  const nodes = layoutedNodes.value
  for (let i = 0; i < nodes.length; i++) {
    const n = nodes[i]
    nodeIndex[n.user_id] = n
  }
  for (let i = 0; i < nodes.length; i++) {
    const n = nodes[i]
    if (n.left_child_id) parentIndex[n.left_child_id] = n.user_id
    if (n.right_child_id) parentIndex[n.right_child_id] = n.user_id
  }
}

// 树形布局算法
const calculateLayout = () => {
  if (treeData.value.length === 0) {
    layoutedNodes.value = []
    rebuildIndexes()
    return
  }
  
  const nodes = treeData.value.map(node => ({
    ...node,
    x: 0,
    y: 0,
    isRoot: !node.parent_id
  }))
  
  // 局部索引：一次 O(n) 建好，供 buildTree 使用，避免递归里反复 find
  const localIndex = {}
  for (let i = 0; i < nodes.length; i++) {
    localIndex[nodes[i].user_id] = nodes[i]
  }

  // 构建树结构
  const buildTree = (nodeId, level = 0) => {
    const node = localIndex[nodeId]
    if (!node) return null
    
    const leftChild = node.left_child_id ? buildTree(node.left_child_id, level + 1) : null
    const rightChild = node.right_child_id ? buildTree(node.right_child_id, level + 1) : null
    
    return {
      node,
      leftChild,
      rightChild,
      level
    }
  }
  
  // 计算节点位置
  const layoutNode = (tree, x, y) => {
    if (!tree) return { width: 0, x }
    
    const node = tree.node
    const leftTree = tree.leftChild
    const rightTree = tree.rightChild
    
    let currentX = x
    let leftWidth = 0
    let rightWidth = 0
    
    // 布局左子树
    if (leftTree) {
      const leftLayout = layoutNode(leftTree, currentX, y + verticalSpacing)
      leftWidth = leftLayout.width
      currentX = leftLayout.x + horizontalSpacing
    }
    
    // 设置当前节点位置
    node.x = currentX - nodeWidth / 2
    node.y = y
    
    const nodeCenterX = currentX
    currentX += nodeWidth / 2 + horizontalSpacing / 2
    
    // 布局右子树
    if (rightTree) {
      const rightLayout = layoutNode(rightTree, currentX, y + verticalSpacing)
      rightWidth = rightLayout.width
      currentX = rightLayout.x
    }
    
    const totalWidth = Math.max(leftWidth + rightWidth + horizontalSpacing, nodeWidth)
    const startX = nodeCenterX - totalWidth / 2
    
    return { width: totalWidth, x: startX + totalWidth }
  }
  
  // 为每个根节点布局
  let startX = 100
  rootNodes.value.forEach(root => {
    const tree = buildTree(root.user_id)
    if (tree) {
      const layout = layoutNode(tree, startX, 100)
      startX = layout.x + horizontalSpacing * 2
    }
  })
  
  layoutedNodes.value = nodes
  // 发布索引给 visibleNodes / connections / isNodeVisible 使用
  rebuildIndexes()
}

// 加载树数据
const loadTreeData = async () => {
  loading.value = true
  try {
    const res = await httpGet('/api/admin/placement/tree')
    treeData.value = res.data || []
    
    // 构建节点映射
    const map = {}
    treeData.value.forEach(node => {
      map[node.user_id] = node
    })
    nodeMap.value = map
    
    // 计算布局
    calculateLayout()
    
    // 重置视图
    resetView()
    
    if (treeData.value.length === 0) {
      ElMessage.info('暂无安置树数据')
    } else {
      ElMessage.success(`成功加载 ${treeData.value.length} 个节点`)
    }
  } catch (e) {
    ElMessage.error('加载安置树数据失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

// 搜索处理
const handleSearch = () => {
  // 搜索时自动展开匹配节点的父节点
  if (searchKeyword.value) {
    let changed = false
    visibleNodes.value.forEach(node => {
      if (isNodeMatched(node)) {
        // 展开所有祖先节点（走 parentIndex，避免数组 find）
        let parentId = parentIndex[node.user_id]
        while (parentId) {
          if (collapsedNodes.value.has(parentId)) {
            collapsedNodes.value.delete(parentId)
            changed = true
          }
          parentId = parentIndex[parentId]
        }
      }
    })
    if (changed) {
      collapsedNodes.value = new Set(collapsedNodes.value)
    }
  }
}

// 切换节点折叠状态
const toggleCollapse = (nodeId) => {
  if (collapsedNodes.value.has(nodeId)) {
    collapsedNodes.value.delete(nodeId)
  } else {
    collapsedNodes.value.add(nodeId)
  }
  collapsedNodes.value = new Set(collapsedNodes.value)
}

// 节点点击
const handleNodeClick = (node) => {
  // 可以添加节点详情查看等功能
}

// 画布拖拽
const handleMouseDown = (e) => {
  if (e.target.closest('.tree-node')) return
  isDragging.value = true
  dragStart.value = { x: e.clientX - panX.value, y: e.clientY - panY.value }
}

const handleMouseMove = (e) => {
  if (isDragging.value && !isNodeDragging.value) {
    panX.value = e.clientX - dragStart.value.x
    panY.value = e.clientY - dragStart.value.y
  } else if (isNodeDragging.value && draggedNode.value) {
    // 节点拖拽
    const deltaX = (e.clientX - dragStart.value.x) / zoom.value
    const deltaY = (e.clientY - dragStart.value.y) / zoom.value
    draggedNode.value.x += deltaX
    draggedNode.value.y += deltaY
    dragStart.value = { x: e.clientX, y: e.clientY }
  }
}

const handleMouseUp = () => {
  isDragging.value = false
  isNodeDragging.value = false
  draggedNode.value = null
}

// 节点拖拽
const startNodeDrag = (node, e) => {
  isNodeDragging.value = true
  draggedNode.value = node
  dragStart.value = { x: e.clientX, y: e.clientY }
  e.preventDefault()
}

// 缩放
const handleWheel = (e) => {
  const delta = e.deltaY > 0 ? -0.1 : 0.1
  const newZoom = Math.max(MIN_ZOOM, Math.min(MAX_ZOOM, zoom.value + delta))
  zoom.value = newZoom
}

const zoomIn = () => {
  zoom.value = Math.min(MAX_ZOOM, zoom.value + 0.1)
}

const zoomOut = () => {
  zoom.value = Math.max(MIN_ZOOM, zoom.value - 0.1)
}

// 自动整理：重新布局并将当前可见节点适配到视图中
const organizeView = async () => {
  if (!treeContainer.value || visibleNodes.value.length === 0) return

  // 先根据当前数据重新计算一次布局
  calculateLayout()
  await nextTick() // 等待布局数据应用

  const nodes = visibleNodes.value
  if (nodes.length === 0) return

  const containerRect = treeContainer.value.getBoundingClientRect()
  const containerWidth = containerRect.width
  const containerHeight = containerRect.height
  if (containerWidth <= 0 || containerHeight <= 0) return

  // 计算可见节点的边界框（循环求极值，避免 large spread 撑爆调用栈）
  let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity
  for (let i = 0; i < nodes.length; i++) {
    const n = nodes[i]
    if (n.x < minX) minX = n.x
    if (n.x + nodeWidth > maxX) maxX = n.x + nodeWidth
    if (n.y < minY) minY = n.y
    if (n.y + nodeHeight > maxY) maxY = n.y + nodeHeight
  }

  const width = Math.max(1, maxX - minX)
  const height = Math.max(1, maxY - minY)

  // 预留边距，但保证有效尺寸
  const margin = 80
  const effectiveWidth = Math.max(120, containerWidth - margin * 2)
  const effectiveHeight = Math.max(120, containerHeight - margin * 2)

  let targetZoom = Math.min(effectiveWidth / width, effectiveHeight / height)
  // 限制缩放范围，避免过小导致看不见
  targetZoom = Math.max(MIN_ZOOM, Math.min(MAX_ZOOM, targetZoom))

  // 计算中心点，并将其居中到容器
  const centerX = (minX + maxX) / 2
  const centerY = (minY + maxY) / 2

  // 若计算结果异常则回退
  if (!isFinite(targetZoom) || !isFinite(centerX) || !isFinite(centerY)) {
    resetView()
    return
  }

  const targetPanX = containerWidth / (2 * targetZoom) - centerX
  const targetPanY = containerHeight / (2 * targetZoom) - centerY

  // 将画布平移限制在带边距的范围内，防止直接飞出视窗
  const minPanX = margin / targetZoom - minX
  const maxPanX = containerWidth / targetZoom - margin / targetZoom - maxX
  const minPanY = margin / targetZoom - minY
  const maxPanY = containerHeight / targetZoom - margin / targetZoom - maxY

  zoom.value = targetZoom
  panX.value = Math.min(Math.max(targetPanX, maxPanX), minPanX)
  panY.value = Math.min(Math.max(targetPanY, maxPanY), minPanY)
}

// 重置视图
const resetView = () => {
  panX.value = 0
  panY.value = 0
  zoom.value = 1
  collapsedNodes.value = new Set()
}

// 全屏切换
const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
  
  if (isFullscreen.value) {
    // 进入全屏
    document.body.style.overflow = 'hidden'
  } else {
    // 退出全屏
    document.body.style.overflow = ''
  }
}

// 监听 ESC 键退出全屏
const handleKeyDown = (e) => {
  if (e.key === 'Escape' && isFullscreen.value) {
    toggleFullscreen()
  }
}

onMounted(() => {
  // 该组件所在的 tab 已改为懒渲染：只有真正切到这个 tab 才会挂载并发请求
  loadTreeData()
  // 监听键盘事件
  window.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  // 清理事件监听
  window.removeEventListener('keydown', handleKeyDown)
  // 确保退出全屏时恢复 body 样式
  if (isFullscreen.value) {
    document.body.style.overflow = ''
  }
})
</script>

<style lang="stylus" scoped>
.placement-tree-container {
  padding: 24px;
  background: #fff;
  border-radius: 8px;
  min-height: calc(100vh - 200px);
  transition: all 0.3s ease;
  
  // 全屏模式
  &.fullscreen-mode {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    width: 100vw;
    height: 100vh;
    padding: 16px;
    margin: 0;
    border-radius: 0;
    z-index: 9999;
    min-height: 100vh;
    
    .tree-header {
      margin-bottom: 16px;
      padding-bottom: 16px;
    }
    
    .tree-content {
      height: calc(100vh - 100px);
      min-height: calc(100vh - 100px);
      
      .tree-visualization-wrapper {
        height: calc(100vh - 100px);
        min-height: calc(100vh - 100px);
        border-radius: 0;
      }
    }
  }
  
  .tree-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
    padding-bottom: 20px;
    border-bottom: 1px solid #e4e7ed;
    
    .header-left {
      display: flex;
      align-items: center;
      gap: 12px;
      
      .page-title {
        margin: 0;
        font-size: 20px;
        font-weight: 600;
        color: #303133;
        display: flex;
        align-items: center;
        gap: 8px;
        
        .title-icon {
          font-size: 24px;
          color: #409eff;
        }
      }
    }
    
    .header-actions {
      display: flex;
      align-items: center;
      gap: 12px;
    }
  }
  
  .tree-content {
    min-height: 500px;
    
    .empty-state {
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 500px;
      background: #fafafa;
      border-radius: 8px;
    }
    
    .tree-visualization-wrapper {
      position: relative;
      height: calc(100vh - 300px);
      min-height: 600px;
      background: #f5f7fa;
      border-radius: 8px;
      border: 1px solid #e4e7ed;
      overflow: hidden;
      
      .toolbar {
        position: absolute;
        top: 12px;
        right: 12px;
        z-index: 100;
        background: rgba(255, 255, 255, 0.9);
        padding: 8px;
        border-radius: 8px;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
        
        .zoom-controls {
          display: flex;
          align-items: center;
          gap: 8px;
          
          .zoom-level {
            min-width: 50px;
            text-align: center;
            font-size: 12px;
            color: #606266;
          }
        }
      }
      
      .tree-visualization {
        width: 100%;
        height: 100%;
        position: relative;
        overflow: hidden;
        cursor: grab;
        
        &:active {
          cursor: grabbing;
        }
        
        .connections-layer {
          position: absolute;
          top: 0;
          left: 0;
          pointer-events: none;
          
          .connection-line {
            stroke-width: 2;
            // 去掉 drop-shadow：每条线一个滤镜，节点多时是渲染大头
          }
          
          .connection-label {
            font-size: 11px;
            font-weight: bold;
            fill: #dc2626;
            background: white;
            paint-order: stroke;
            stroke: white;
            stroke-width: 3;
            stroke-linecap: round;
            stroke-linejoin: round;
          }
        }
        
        .nodes-layer {
          position: absolute;
          top: 0;
          left: 0;
          transition: transform 0.1s ease-out;
          
          .tree-node {
            position: absolute;
            cursor: move;
            user-select: none;
            
            &.root-node {
              .node-card {
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                color: #fff;
                border: 2px solid rgba(255, 255, 255, 0.3);
                box-shadow: 0 8px 24px rgba(102, 126, 234, 0.4);
                
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
                    background: rgba(255, 255, 255, 0.15);
                    
                    .perf-label {
                      color: rgba(255, 255, 255, 0.8);
                    }
                    
                    .perf-value {
                      color: #fff;
                    }
                    
                    &.left .perf-value {
                      color: #a8d8ff;
                    }
                    
                    &.right .perf-value {
                      color: #b8f5b8;
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
                  
                  .total-perf {
                    border-top-color: rgba(255, 255, 255, 0.2);
                    
                    .total-label {
                      color: rgba(255, 255, 255, 0.8);
                    }
                    
                    .total-value {
                      color: #fff;
                    }
                  }
                }
              }
            }
            
            &.matched {
              .node-card {
                border-color: #67c23a;
                box-shadow: 0 0 20px rgba(103, 194, 58, 0.6);
                animation: pulse 2s infinite;
              }
            }
            
            .node-card {
              width: 170px;
              background: #ffffff;
              border: 2px solid #e4e7ed;
              border-radius: 12px;
              padding: 10px;
              box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
              transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
              
              &:hover {
                transform: translateY(-2px);
                box-shadow: 0 8px 20px rgba(0, 0, 0, 0.12);
                border-color: #409eff;
              }
              
              .node-header {
                display: flex;
                flex-direction: column;
                align-items: center;
                margin-bottom: 10px;
                padding-bottom: 10px;
                border-bottom: 1px solid #f0f0f0;
                position: relative;
                
                .user-badge {
                  background: linear-gradient(135deg, #f5f7fa 0%, #e9ecef 100%);
                  border: 1px solid #e4e7ed;
                  border-radius: 20px;
                  padding: 4px 12px;
                  margin-bottom: 6px;
                  
                  .node-id {
                    font-weight: 600;
                    font-size: 12px;
                    color: #303133;
                  }
                }
                
                .node-username {
                  font-size: 12px;
                  color: #606266;
                  font-weight: 500;
                  max-width: 100%;
                  overflow: hidden;
                  text-overflow: ellipsis;
                  white-space: nowrap;
                }
                
                .collapse-btn {
                  position: absolute;
                  top: 0;
                  right: 0;
                  width: 20px;
                  height: 20px;
                  display: flex;
                  align-items: center;
                  justify-content: center;
                  background: rgba(64, 158, 255, 0.1);
                  border-radius: 50%;
                  cursor: pointer;
                  transition: all 0.3s;
    
    &:hover {
                    background: rgba(64, 158, 255, 0.2);
                    transform: scale(1.1);
                  }
                  
                  .el-icon {
                    font-size: 12px;
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
                  gap: 8px;
                  margin-bottom: 8px;
                  
                  .perf-item {
                    flex: 1;
                    padding: 6px;
                    background: #f8f9fa;
                    border-radius: 6px;
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    
                    .perf-label {
                      font-size: 10px;
                      color: #909399;
                      margin-bottom: 4px;
                    }
                    
                    .perf-value {
                      font-size: 14px;
                      font-weight: 700;
                      color: #303133;
                    }
                    
                    &.left .perf-value {
                      color: #1976d2;
                    }
                    
                    &.right .perf-value {
                      color: #388e3c;
                    }
                  }
                }
                
                .recharge-amount {
                  text-align: center;
                  padding: 6px;
                  background: linear-gradient(135deg, #fff5e6 0%, #ffe8cc 100%);
                  border-radius: 6px;
                  border-top: 1px solid #e4e7ed;
                  margin-top: 6px;
                  display: flex;
                  justify-content: space-between;
                  align-items: center;
                  
                  .recharge-label {
                    font-size: 11px;
                    color: #909399;
                  }
                  
                  .recharge-value {
                    font-size: 14px;
                    font-weight: 700;
                    color: #e6a23c;
                  }
                }
                
                .total-perf {
                  text-align: center;
                  padding: 6px;
                  background: linear-gradient(135deg, #fafafa 0%, #f5f5f5 100%);
                  border-radius: 6px;
                  border-top: 1px solid #e4e7ed;
                  margin-top: 6px;
                  display: flex;
                  justify-content: space-between;
                  align-items: center;
                  
                  .total-label {
                    font-size: 11px;
                    color: #909399;
                  }
                  
                  .total-value {
                    font-size: 14px;
                    font-weight: 700;
                    color: #303133;
                  }
                }
              }
            }
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
  .placement-tree-container {
    padding: 16px;
    
    .tree-header {
      flex-direction: column;
      align-items: flex-start;
      gap: 16px;
      
      .header-actions {
        width: 100%;
        flex-direction: column;
        
        .el-input {
          width: 100% !important;
        }
      }
    }
  }
}
</style>
