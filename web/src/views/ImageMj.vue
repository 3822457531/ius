<template>
  <div class="mobile-mj">
    <van-form @submit="generate">
      <!-- <div class="text-line section-title">图片比例</div> -->
      <!-- <div class="text-line">
        <van-row :gutter="10" justify="center">
          <van-col :span="6" v-for="item in rates" :key="item.value">
            <div
                :class="item.value === params.rate ? 'rate active' : 'rate'"
                @click="changeRate(item)">
              <div class="icon">
                <van-image :src="item.img" fit="cover"></van-image>
              </div>
              <div class="text">{{ item.text }}</div>
            </div>
          </van-col>
        </van-row>
      </div> -->
      <!-- <div class="text-line section-title">模型选择</div> -->
      <div class="text-line">
        <van-row :gutter="12">
          <van-col :span="12" v-for="item in models" :key="item.value">
            <div :class="item.value === params.model ? 'model active' : 'model'"
                 @click="changeModel(item)">
              <div class="icon">
                <van-image :src="item.img" fit="cover"></van-image>
              </div>
              <div class="text">
                <van-text-ellipsis :content="item.text"/>
              </div>
            </div>
          </van-col>
        </van-row>
      </div>

      <!-- <div class="text-line">
        <van-field label="创意度">
          <template #input>
            <van-slider v-model.number="params.chaos" :max="100" :step="1"
                        @update:model-value="showToast('当前值：' + params.chaos)"/>
          </template>
        </van-field>
      </div>

      <div class="text-line">
        <van-field label="风格化">
          <template #input>
            <van-slider v-model.number="params.stylize" :max="1000" :step="1"
                        @update:model-value="showToast('当前值：' + params.stylize)"/>
          </template>
        </van-field>
      </div>

      <div class="text-line">
        <van-field label="原始模式">
          <template #input>
            <van-switch v-model="params.raw"/>
          </template>
        </van-field>
      </div> -->

      <div class="text-line">
        <van-tabs v-model:active="activeName" @change="tabChange" animated>
          <van-tab :title="t('imageMj.tabTxt2Img')" name="txt2img">
            <div class="text-line">
              <van-field
                v-model="params.prompt"
                rows="3"
                autosize
                type="textarea"
                :placeholder="t('imageMj.promptPlaceholderTxt2Img')"
              />
            </div>
          </van-tab>
          <van-tab :title="t('imageMj.tabImg2Img')" name="img2img">
            <div class="text-line">
              <van-field
                v-model="params.prompt"
                rows="3"
                autosize
                type="textarea"
                :placeholder="t('imageMj.promptPlaceholderImg2Img')"
              />
            </div>

            <div class="text-line">
              <van-uploader v-model="imgList" :after-read="uploadImg"/>
            </div>

            <!-- <div class="text-line">
              <van-field label="垫图权重">
                <template #input>
                  <van-slider v-model.number="params.iw" :max="1" :step="0.01"
                              @update:model-value="showToast('当前值：' + params.iw)"/>
                </template>
              </van-field>
            </div> -->

            <!-- <div class="tip-text">提示：只有于 niji6 和 v6 模型支持一致性功能，如果选择其他模型此功能将会生成失败。</div>
            <van-cell-group>
              <van-field
                  v-model="params.cref"
                  center
                  clearable
                  label="角色一致性"
                  placeholder="请输入图片URL或者上传图片"
              >
                <template #button>
                  <van-uploader @click="beforeUpload('cref')" :after-read="uploadImg">
                    <van-button size="mini" type="primary" icon="plus"/>
                  </van-uploader>
                </template>
              </van-field>
            </van-cell-group> -->

          </van-tab>
  <!--
            <van-cell-group>
              <van-field
                  v-model="params.sref"
                  center
                  clearable
                  label="风格一致性"
                  placeholder="请输入图片URL或者上传图片"
              >
                <template #button>
                  <van-uploader @click="beforeUpload('sref')" :after-read="uploadImg">
                    <van-button size="mini" type="primary" icon="plus"/>
                  </van-uploader>
                </template>
              </van-field>
            </van-cell-group>

           <div class="text-line">
              <van-field label="一致性权重">
                <template #input>
                  <van-slider v-model.number="params.cw" :max="100" :step="1"
                              @update:model-value="showToast('当前值：' + params.cw)"/>
                </template>
              </van-field>
            </div> -->
      

          
          <!-- <van-tab title="融图" name="blend">
            <div class="tip-text">请上传两张以上的图片，不超过五张，AI会自动将识别图片中的特征，并将所有图片的特征结合生成一张新的图片</div>
            <div class="text-line">
              <van-uploader v-model="imgList" :after-read="uploadImg"/>
            </div>
          </van-tab>
          <van-tab title="换脸" name="swapFace">
            <div class="tip-text">请上传两张有脸部的图片，AI会将左边图片的脸替换右边图片中的脸。并生成一张新的图片</div>
            <div class="text-line">
              <van-uploader v-model="imgList" :after-read="uploadImg"/>
            </div>
          </van-tab> -->


          <!-- <van-tab title="角色一致" name="oneImage">
            <div class="tip-text">请上传一张有脸部的图片，AI会根据提示来生成与上传图片人脸保持高度一致性的新的图片</div>

            <div class="text-line">
              <van-field v-model="params.prompt"
                         rows="3"
                         autosize
                         type="textarea"
                         placeholder="请在此输入绘画提示词"/>
            </div>
      </van-tab> -->
            <!-- <div class="text-line">
              <van-uploader v-model="imgList" :after-read="uploadImg"/>
            </div> -->


            <!-- <div class="tip-text">提示：只有于 niji6 和 v6 模型支持一致性功能，如果选择其他模型此功能将会生成失败。</div> -->
            <!-- <van-cell-group>
              <van-field
                  v-model="params.cref"
                  center
                  clearable
                  label="角色一致性"
                  placeholder="请输入图片URL或者上传图片"
              >
                <template #button>
                  <van-uploader @click="beforeUpload('cref')" :after-read="uploadImg">
                    <van-button size="mini" type="primary" icon="plus"/>
                  </van-uploader>
                </template>
              </van-field>
            </van-cell-group>

                <div class="text-line">
              <van-field label="一致性权重(默认100)">
                <template #input>
                  <van-slider v-model.number="params.cw" :max="100" :step="1"
                              @update:model-value="showToast('当前值：' + params.cw)"/>
                </template>
              </van-field>
            </div> -->


    

        </van-tabs>
      </div>
<!-- 
      <div class="text-line">
        <van-collapse v-model="activeColspan">
          <van-collapse-item title="反向提示词" name="neg_prompt">
            <van-field
                v-model="params.neg_prompt"
                rows="3"
                autosize
                type="textarea"
                placeholder="不想出现在图片上的元素(例如：树，建筑)"
            />
          </van-collapse-item>
        </van-collapse>
      </div> -->
<!-- 
      <div class="text-line pt-6">
        <el-tag>
          {{ t('imageMj.powerTip', { mjPower, mj2Power, power }) }}
        </el-tag>
      </div> -->

      <div class="text-line">
        <van-button round block type="primary" native-type="submit">
          {{ t('imageMj.generateNow') }}
        </van-button>
      </div>
    </van-form>

    <h3>{{ t('imageMj.runningTitle') }}</h3>
    <div class="running-job-list pt-3 pb-3">
      <van-empty
        v-if="runningJobs.length === 0"
        image="https://fastly.jsdelivr.net/npm/@vant/assets/custom-empty-image.png"
        image-size="80"
        :description="t('imageMj.emptyDesc')"
      />
      <van-grid :gutter="10" :column-num="3" v-else>
        <van-grid-item v-for="item in runningJobs" :key="item.id">
          <div v-if="item.progress > 0">
            <van-image src="/images/img-holder.png"></van-image>
            <div class="progress">
              <van-circle v-model:current-rate="item.progress" :rate="item.progress" :speed="100" :text="item.progress + '%'" :stroke-width="60" size="90px" />
            </div>
          </div>

          <div v-else class="task-in-queue">
            <div class="loading-wrapper">
              <div class="icon-container">
                <i class="iconfont icon-quick-start loading-icon"></i>
                <div class="pulse-ring"></div>
              </div>
              <div class="text-container">
                <span class="text">{{ loadingText }}</span>
                <span class="dots">
                  <span v-for="i in 3" :key="i" class="dot" :class="{ active: loadingDots >= i }"></span>
                </span>
              </div>
            </div>
          </div>

        </van-grid-item>
      </van-grid>
    </div>

    <h3>{{ t('imageMj.historyTitle') }}</h3>
    <div class="finish-job-list">
      <van-empty
        v-if="finishedJobs.length === 0"
        image="https://fastly.jsdelivr.net/npm/@vant/assets/custom-empty-image.png"
        image-size="80"
        :description="t('imageMj.emptyDesc')"
      />

      <van-list
        v-else
        v-model:error="error"
        v-model:loading="loading"
        :finished="finished"
        :error-text="t('imageMj.listErrorText')"
        :finished-text="t('imageMj.listFinishedText')"
        @load="onLoad"
      >
        <van-grid :gutter="10" :column-num="2">
          <van-grid-item v-for="item in finishedJobs">
            <div class="job-item">
              <van-image
                  :src="item['thumb_url']"
                  :class="item['can_opt'] ? '' : 'upscale'"
                  lazy-load
                  @click="imageView(item)"
                  fit="cover">
                <template v-slot:loading>
                  <van-loading type="spinner" size="20"/>
                </template>
              </van-image>

              <!-- <div class="opt" v-if="item['can_opt']">

                <van-grid :gutter="3" :column-num="4">
                  <van-grid-item><a @click="upscale(1, item)" class="opt-btn">U1</a></van-grid-item>
                  <van-grid-item><a @click="upscale(2, item)" class="opt-btn">U2</a></van-grid-item>
                  <van-grid-item><a @click="upscale(3, item)" class="opt-btn">U3</a></van-grid-item>
                  <van-grid-item><a @click="upscale(4, item)" class="opt-btn">U4</a></van-grid-item>
                  <van-grid-item><a @click="variation(1, item)" class="opt-btn">V1</a></van-grid-item>
                  <van-grid-item><a @click="variation(2, item)" class="opt-btn">V2</a></van-grid-item>
                  <van-grid-item><a @click="variation(3, item)" class="opt-btn">V3</a></van-grid-item>
                  <van-grid-item><a @click="variation(4, item)" class="opt-btn">V4</a></van-grid-item>
                </van-grid>
              </div> -->

              <div class="remove">
                <el-button type="danger" :icon="Delete" @click="removeImage(item)" circle/>

                <!-- <el-button type="warning" v-if="item.publish" @click="publishImage(item, false)"
                           circle>
                  <i class="iconfont icon-cancel-share"></i>
                </el-button>

                <el-button type="success" v-else @click="publishImage(item, true)" circle>
                  <i class="iconfont icon-share-bold"></i>
                </el-button> -->

                <el-button type="primary" @click="showPrompt(item)" circle>
                  <i class="iconfont icon-prompt"></i>
                </el-button>
              </div>
            </div>
          </van-grid-item>
        </van-grid>
      </van-list>

    </div>

  </div>
</template>

<script setup>
import {computed, nextTick, onMounted, onUnmounted, ref} from "vue";
import { useI18n } from "vue-i18n";
import {
  showConfirmDialog,
  showFailToast,
  showNotify,
  showToast,
  showDialog,
  showImagePreview,
  showSuccessToast
} from "vant";
import {httpGet, httpPost, resolveWsHost} from "@/utils/http";
import Compressor from "compressorjs";
import {getSessionId} from "@/store/session";
import {randString} from "@/utils/libs";
import {checkSession} from "@/action/session";
import {useRouter} from "vue-router";
import {Delete} from "@element-plus/icons-vue";

const { t } = useI18n();

const activeColspan = ref([""])

const rates = [
  {css: "square", value: "1:1", text: "方屏", img: "/images/mj/rate_1_1.png"},
  {css: "size16-9", value: "16:9", text: "横屏", img: "/images/mj/rate_16_9.png"},
  {css: "size9-16", value: "9:16", text: "竖屏", img: "/images/mj/rate_9_16.png"},
]
const models = [
  {text: "nb2", value: "gemini-3.1-flash-image-preview-4k", img: "/images/mj/mj-v6.png"},
  {text: "nbpro", value: "nano-banana-2-4k", img: "/images/mj/mj-v5.2.png"},

  
]
const imgList = ref([])
const params = ref({
  client_id: randString(10),
  task_type: "image",
  nano_mode: "txt2img",
  rate: rates[0].value,
  model: models[0].value,
  chaos: 0,
  stylize: 0,
  seed: 0,
  img_arr: [],
  raw: false,
  iw: 0,
  prompt: "",
  neg_prompt: "",
  tile: false,
  quality: 0,
  cref: "",
  sref: "",
  cw: 0,
})
const userId = ref(0)
const router = useRouter()
const runningJobs = ref([])
const finishedJobs = ref([])
const socket = ref(null)
const power = ref(0)
const activeName = ref("txt2img")
const loadingDots = ref(0)
let loadingTimer = null
let refreshTimer = null // 用于定期刷新任务列表

// 生成动态的"生成中"文本
const loadingText = computed(() => t('imageMj.loadingText'))

onMounted(() => {
  checkSession().then(user => {
    power.value = user['power']
    userId.value = user.id

    fetchRunningJobs()
    fetchFinishJobs(1)
    try {
      connect()
    } catch (e) {
      console.warn('mj websocket connect failed', e)
    }
    
    // 启动自动刷新机制：每3秒检查一次运行中的任务
    // nano banana是同步接口，任务完成后progress会立即变为100
    // 需要定期刷新来检测任务完成状态
    startAutoRefresh()

  }).catch(() => {
    router.push('/login')
  });

  // 启动加载动画定时器
  loadingTimer = setInterval(() => {
    loadingDots.value = (loadingDots.value + 1) % 4; // 0, 1, 2, 3 循环
  }, 500); // 每500ms切换一次
})

onUnmounted(() => {
  socket.value = null
  // 清理加载动画定时器
  if (loadingTimer) {
    clearInterval(loadingTimer);
    loadingTimer = null;
  }
  // 清理自动刷新定时器
  stopAutoRefresh()
})

const mjPower = ref(1)
const mj2Power = ref(1)
const mjActionPower = ref(1)
httpGet("/api/config/get?key=system").then(res => {
  mjPower.value = res.data["mj_power"]
  mj2Power.value = res.data["mj_power2"] || res.data["mj_power"]
  mjActionPower.value = res.data["mj_action_power"]
}).catch(e => {
  showNotify({type: "danger", message: t("imageMj.configLoadFailPrefix") + e.message})
})

const heartbeatHandle = ref(null)
const connect = () => {
  const host = resolveWsHost()

  // 心跳函数
  const sendHeartbeat = () => {
    clearTimeout(heartbeatHandle.value)
    new Promise((resolve, reject) => {
      if (socket.value !== null) {
        socket.value.send(JSON.stringify({type: "heartbeat", content: "ping"}))
      }
      resolve("success")
    }).then(() => {
      heartbeatHandle.value = setTimeout(() => sendHeartbeat(), 5000)
    });
  }

  const _socket = new WebSocket(host + `/api/mj/client?user_id=${userId.value}`);
  _socket.addEventListener('open', () => {
    socket.value = _socket;

    // 发送心跳消息
    sendHeartbeat()
  });

  _socket.addEventListener('message', event => {
    if (event.data instanceof Blob) {
      // WebSocket收到消息时，立即刷新任务列表
      fetchRunningJobs()
      fetchFinishJobs(1)
      // 确保自动刷新正在运行
      if (!refreshTimer && runningJobs.value.length > 0) {
        startAutoRefresh()
      }
    }
  });

  _socket.addEventListener('close', () => {
    if (socket.value !== null) {
      connect()
    }
  });
}

// 启动自动刷新机制
const startAutoRefresh = () => {
  stopAutoRefresh()
  // 每3秒检查一次运行中的任务
  // nano banana是同步接口，任务完成后progress会立即变为100
  refreshTimer = setInterval(() => {
    // 如果有运行中的任务，则刷新列表
    if (runningJobs.value.length > 0) {
      fetchRunningJobs()
      // 同时刷新已完成任务列表，因为可能有任务从运行中变为已完成
      fetchFinishJobs(1)
    }
  }, 3000) // 每3秒刷新一次
}

// 停止自动刷新
const stopAutoRefresh = () => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

// 获取运行中的任务
const fetchRunningJobs = () => {
  httpGet(`/api/mj/jobs?finish=0&app_source=image`).then(res => {
    const jobs = res.data.items || res.data
    const _jobs = []
    for (let i = 0; i < jobs.length; i++) {
      // 检查任务是否失败
      if (jobs[i].progress === -1) {
        showNotify({
          message: t('imageMj.jobFailPrefix') + (jobs[i]['err_msg'] || ''),
          type: 'danger',
        })
        if (jobs[i].type === 'image') {
          power.value += mjPower.value
        } else {
          power.value += mjActionPower.value
        }
        continue
      }
      // nano banana是同步接口，如果progress >= 100，说明任务已完成
      // 但可能还在运行中列表（因为img_url可能还没下载完成）
      // 这种情况下，任务应该从运行中列表移除，并刷新已完成列表
      if (jobs[i].progress >= 100) {
        // 任务已完成，刷新已完成列表
        fetchFinishJobs(1)
        continue
      }
      _jobs.push(jobs[i])
    }
    runningJobs.value = _jobs
    
    // 如果运行中的任务列表为空，停止自动刷新以节省资源
    if (_jobs.length === 0) {
      stopAutoRefresh()
    } else {
      // 确保自动刷新正在运行
      if (!refreshTimer) {
        startAutoRefresh()
      }
    }
  }).catch(e => {
    showNotify({type: "danger", message: t("imageMj.fetchJobsFailPrefix") + e.message})
  })
}

const loading = ref(false)
const finished = ref(false)
const error = ref(false)
const page = ref(0)
const pageSize = ref(10)
const fetchFinishJobs = (page) => {
  loading.value = true
  // 获取已完成的任务（progress >= 100）
  httpGet(`/api/mj/jobs?finish=1&page=${page}&page_size=${pageSize.value}&app_source=image`).then(res => {
    const jobs = res.data.items || res.data
    for (let i = 0; i < jobs.length; i++) {
      // 检查任务是否失败
      if (jobs[i].progress === -1) {
    showNotify({
      message: t('imageMj.jobFailWithIdPrefix') + jobs[i]['task_id'] + t('imageMj.jobFailWithIdMid') + (jobs[i]['err_msg'] || ''),
      type: 'danger',
    })
        if (jobs[i].type === 'image') {
          power.value += mjPower.value
        } else {
          power.value += mjActionPower.value
        }
        continue
      }

      // nano banana是同步接口，任务完成后progress=100，但img_url可能还在下载中
      // 如果progress >= 100，说明任务已完成，即使img_url为空也应该显示
      // 如果img_url存在，则生成缩略图URL
      if (jobs[i].img_url) {
        if (jobs[i]['use_proxy']) {
          jobs[i]['thumb_url'] = jobs[i]['img_url'] + '?x-oss-process=image/quality,q_60&format=webp'
        } else {
          if (jobs[i].type === 'upscale' || jobs[i].type === 'swapFace') {
            jobs[i]['thumb_url'] = jobs[i]['img_url'] + '?imageView2/1/w/480/h/600/q/75'
          } else {
            jobs[i]['thumb_url'] = jobs[i]['img_url'] + '?imageView2/1/w/480/h/480/q/75'
          }
        }
      } else if (jobs[i].org_url) {
        // 如果img_url为空但org_url存在，使用org_url作为临时显示
        // 这种情况发生在图片还在下载中
        jobs[i]['thumb_url'] = jobs[i]['org_url']
      }

      if (jobs[i].type === 'image' || jobs[i].type === 'variation') {
        jobs[i]['can_opt'] = true
      }
    }
    if (jobs.length < pageSize.value) {
      finished.value = true
    }
    if (page === 1) {
      finishedJobs.value = jobs
    } else {
      finishedJobs.value = finishedJobs.value.concat(jobs)
    }
    nextTick(() => loading.value = false)
  }).catch(e => {
    loading.value = false
    error.value = true
    showFailToast(t("imageMj.fetchJobsFailPrefix") + e.message)
  })
}

const onLoad = () => {
  page.value += 1
  fetchFinishJobs(page.value)
};

// 切换图片比例
const changeRate = (item) => {
  params.value.rate = item.value
}
// 切换模型
const changeModel = (item) => {
  params.value.model = item.value
}

const imgKey = ref("")
const beforeUpload = (key) => {
  imgKey.value = key
}

// 图片上传
const uploadImg = (file) => {
  file.status = "uploading"
  // 压缩图片并上传
  new Compressor(file.file, {
    quality: 0.6,
    success(result) {
      const formData = new FormData();
      formData.append('file', result, result.name);
      // 执行上传操作
      httpPost('/api/upload', formData).then(res => {
        file.url = res.data.url
        if (imgKey.value !== "") { // 单张图片上传
          params.value[imgKey.value] = res.data.url
          imgKey.value = ''
        }
        file.status = "done"
      }).catch(e => {
        file.status = 'failed'
        file.message = t('imageMj.uploadFail')
        showFailToast(t("imageMj.uploadImageFailPrefix") + e.message)
      })
    },
    error(err) {
      console.log(err.message);
    },
  });
};

const send = (url, index, item) => {
  httpPost(url, {
    client_id: params.value.client_id,
    index: index,
    channel_id: item.channel_id,
    message_id: item.message_id,
    message_hash: item.hash,
    session_id: getSessionId(),
    prompt: item.prompt,
  }).then(() => {
    showSuccessToast(t("imageMj.sendTaskSuccess"))
    power.value -= mjActionPower.value
    fetchRunningJobs()
  }).catch(e => {
    showFailToast(t("imageMj.sendTaskFailPrefix") + e.message)
  })
}

// 图片放大任务
const upscale = (index, item) => {
  send('/api/mj/upscale', index, item)
}

// 图片变换任务
const variation = (index, item) => {
  send('/api/mj/variation', index, item)
}

const generate = () => {
  if (params.value.prompt === '' && params.value.task_type === "image") {
    return showFailToast(t("imageMj.promptRequired"))
  }
  if (activeName.value === "img2img" && imgList.value.length === 0) {
    return showFailToast(t("imageMj.refImageRequired"))
  }
  params.value.session_id = getSessionId()
  params.value.img_arr = imgList.value.map(img => img.url)
  
  httpPost("/api/mj/image", params.value).then(() => {
    showToast(t("imageMj.pushSuccess"))
    const powerCost = params.value.model === "nano-banana-2-4k" ? mj2Power.value : mjPower.value
    power.value -= powerCost
    fetchRunningJobs()
  }).catch(e => {
    showFailToast(t("imageMj.sendTaskFailPrefix") + e.message)
  })
}

const removeImage = (item) => {
  showConfirmDialog({
    title: t('imageMj.deleteTitle'),
    message: t('imageMj.deleteMessage'),
  }).then(() => {
    httpPost("/api/mj/remove", {id: item.id, user_id: item.user_id || userId.value}).then(() => {
      showSuccessToast(t("imageMj.deleteSuccess"))
      fetchFinishJobs(1)
    }).catch(e => {
      showFailToast(t("imageMj.deleteFailPrefix") + e.message)
    })
  }).catch(() => {
    showToast(t("imageMj.cancelOperation"))
  });
}
// 发布图片到作品墙
const publishImage = (item, action) => {
  let text = t("imageMj.publishPrefix")
  if (action === false) {
    text = t("imageMj.unpublishPrefix")
  }
  httpPost("/api/mj/publish", {id: item.id, action: action}).then(() => {
    showSuccessToast(text + t("imageMj.publishSuccessSuffix"))
    item.publish = action
  }).catch(e => {
    showFailToast(text + t("imageMj.publishFailSuffix") + e.message)
  })
}

const showPrompt = (item) => {
  showDialog({
    title: t("imageMj.promptDialogTitle"),
    message: item.prompt,
  }).then(() => {
    // on close
  });
}

const imageView = (item) => {
  showImagePreview([item['img_url']]);
}

// 切换菜单
const tabChange = (tab) => {
  params.value.nano_mode = tab
  params.value.task_type = "image"
}
</script>

<style lang="stylus">
@import "@/assets/css/mobile/image-mj.styl"
</style>