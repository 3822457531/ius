<template>
  <div class="video-generate">
    <van-tabs v-model:active="activeTab">
      <van-tab title="视频生成" name="text">
        <van-field
          v-model="title"
          label="标题"
          placeholder="请输入视频标题"
        />

        <van-field
          v-model="prompt"
          type="textarea"
          rows="3"
          label="描述"
          placeholder="请输入视频描述"
        />

        <van-cell-group>
          <van-cell title="视频时长">
            <template #right-icon>
              <van-stepper 
                v-model="duration" 
                :min="5"
                :max="60"
                :step="5"
              />
            </template>
          </van-cell>
        </van-cell-group>

        <van-cell-group title="视频设置">
          <van-cell title="分辨率">
            <template #right-icon>
              <van-dropdown-menu>
                <van-dropdown-item v-model="resolution" :options="resolutionOptions" />
              </van-dropdown-menu>
            </template>
          </van-cell>
        </van-cell-group>

        <van-cell-group title="参考图片">
          <van-uploader
            v-model="imageFiles"
            :max-count="1"
            :after-read="afterRead"
          />
        </van-cell-group>
      </van-tab>
    </van-tabs>

    <div class="action-buttons">
      <van-button type="primary" @click="handleGenerate" :loading="generating">
        生成视频
      </van-button>
    </div>

    <!-- 生成结果展示 -->
    <div v-if="videoResult" class="result-section">
      <van-cell-group>
        <van-cell title="状态">
          <template #right-icon>
            <van-tag :type="statusType">{{ statusText }}</van-tag>
          </template>
        </van-cell>
        <van-cell v-if="videoResult.status === 'SUCCEEDED'" title="进度">
          <van-progress :percentage="(parseFloat(videoResult.progress || 0) * 100).toFixed(0)" />
        </van-cell>
        <van-cell v-if="videoResult.video_url" title="视频">
          <video 
            controls 
            :src="videoResult.video_url"
            class="video-player"
            preload="metadata"
            controlsList="nodownload"
          ></video>
        </van-cell>
      </van-cell-group>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { httpGet, httpPost } from "@/utils/http"
import { showToast, showLoadingToast, closeToast } from 'vant'

const activeTab = ref('text')
const title = ref('')
const prompt = ref('')
const duration = ref(5)
const resolution = ref('1024x576')
const imageFiles = ref([])
const generating = ref(false)
const videoResult = ref(null)

const resolutionOptions = [
  { text: '16:9 横版', value: '1024x576' },
  { text: '9:16 竖版', value: '576x1024' },
  { text: '1:1 方形', value: '768x768' }
]

const statusType = computed(() => {
  if (!videoResult.value) return 'default'
  switch (videoResult.value.status) {
    case 'SUCCEEDED':
      return 'success'
    case 'FAILED':
      return 'danger'
    case 'PROCESSING':
      return 'primary'
    default:
      return 'default'
  }
})

const statusText = computed(() => {
  if (!videoResult.value) return ''
  switch (videoResult.value.status) {
    case 'SUCCEEDED':
      return '生成完成'
    case 'FAILED':
      return '生成失败'
    case 'PROCESSING':
      return `生成中 ${(parseFloat(videoResult.value.progress || 0) * 100).toFixed(0)}%`
    default:
      return videoResult.value.status
  }
})

const afterRead = async (file) => {
  try {
    const formData = new FormData()
    formData.append('file', file.file)
    
    const res = await httpPost('/api/upload', formData)
    if (res.code === 0 && res.data) {
      file.url = res.data.url
      imageFiles.value = [file]
    } else {
      showToast('上传失败')
      imageFiles.value = []
    }
  } catch (error) {
    showToast('上传失败')
    imageFiles.value = []
  }
}

const handleGenerate = async () => {
  if (!title.value || !prompt.value) {
    showToast('请填写标题和描述')
    return
  }

  try {
    generating.value = true
    showLoadingToast({
      message: '提交中...',
      forbidClick: true,
    })

    const data = {
      title: title.value,
      prompt: prompt.value,
      duration: duration.value,
      resolution: resolution.value,
      image_prompt: imageFiles.value.length > 0 ? imageFiles.value[0].url : ''
    }

    const res = await httpPost('/runway/v1/pro/generate', data)

    if (res.code === 0 && res.data) {
      closeToast()
      showToast('提交成功')
      videoResult.value = res.data
      if (res.data.id) {
        pollStatus(res.data.id)
      }
    } else {
      showToast(res.msg || '生成失败')
    }
  } catch (error) {
    showToast('请求失败')
  } finally {
    generating.value = false
  }
}

const pollStatus = async (id) => {
  try {
    const res = await httpGet(`/runway/v1/feed/${id}`)
    if (res.code === 0 && res.data) {
      videoResult.value = res.data
      
      if (res.data.status === 'PROCESSING') {
        setTimeout(() => pollStatus(id), 3000)
      } else if (res.data.status === 'FAILED') {
        showToast('生成失败: ' + (res.data.error || '未知错误'))
      } else if (res.data.status === 'SUCCEEDED') {
        showToast('生成完成')
      }
    }
  } catch (error) {
    console.error('Poll status error:', error)
  }
}
</script>

<style scoped>
.video-generate {
  padding: 16px;
}

.action-buttons {
  margin-top: 16px;
  text-align: right;
}

.result-section {
  margin-top: 20px;
}

.video-player {
  width: 100%;
  max-height: 300px;
  border-radius: 8px;
}

.van-uploader {
  padding: 10px 16px;
}

.van-progress {
  margin: 8px 0;
}
</style> 