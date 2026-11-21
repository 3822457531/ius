<template>

  <div class="digital-human-generator">
    <van-form @submit="generateVideo">

      <el-input
                    v-model="params.content"
                    :autosize="{ minRows: 6, maxRows: 12 }"
                    type="textarea"
                    ref="promptRef"
                    placeholder="请在此输入数字人需要播报的内容"
                />

      <div class="text-line">选择形象</div>
      <van-row :gutter="10">
        <van-col :span="8" v-for="item in characters" :key="item.value">
          <div :class="item.value === params.character ? 'option active' : 'option'" @click="selectCharacter(item)">
            <van-image :src="item.img" fit="cover" width="100%" height="100px"></van-image>
            <div class="text">{{ item.text }}</div>
          </div>
        </van-col>
      </van-row>

      <div class="text-line">选择背景图</div>
      <van-row :gutter="10">
        <van-col :span="8" v-for="item in backgrounds" :key="item.value">
          <div :class="item.value === params.background ? 'option active' : 'option'" @click="selectBackground(item)">
            <van-image :src="item.img" fit="cover" width="100%" height="100px"></van-image>
            <div class="text">{{ item.text }}</div>
          </div>
        </van-col>
      </van-row>



      <div class="text-line">选择语言</div>
      <div class="param-line" style="padding-top: 10px">
              <el-form-item label="">
                <template #default>
                  <div class="form-item-inner flex-row items-center">
                    <el-select v-model="params.language" placeholder="请选择">
                      <el-option v-for="item in languageOptions"
                                 :key="item.value"
                                 :label="item.label"
                                 :value="item.value">
                      </el-option>
                    </el-select>
                  
                  </div>
                </template>
              </el-form-item>
     </div>


      <van-button round block  type="primary" native-type="submit">生成数字人视频</van-button>
    </van-form>
  <!-- Running jobs section -->
  <h3>任务列表</h3>
    <div class="running-job-list">
      <van-empty v-if="runningJobs.length === 0"
                 image="https://fastly.jsdelivr.net/npm/@vant/assets/custom-empty-image.png"
                 image-size="80"
                 description="暂无记录"
      />
      <van-grid :gutter="10" :column-num="3" v-else>
        <van-grid-item v-for="item in runningJobs" :key="item.id">
          <div v-if="item.progress > 0">
            <van-image :src="item.pre_url">
              <template v-slot:error>加载失败</template>
            </van-image>

            <!-- Progress bar -->
            <!-- <div class="progress">
              <van-circle
                  :current-rate="item.progress"
                  :rate="item.progress"
                  :speed="100"
                  :text="item.progress + '%'"
                  :stroke-width="6"
                  size="90px"
              />
            </div> -->

            <div class="task-in-progress">
             <span class="text">生成中...</span>
           </div>

          </div>

          <div v-else class="task-in-queue">
            <span class="icon"><i class="iconfont icon-quick-start"></i></span>
            <span class="text">生成中</span>
          </div>
        </van-grid-item>
      </van-grid>
    </div>

    <!-- Finished jobs section -->
    <h3>数字人生成记录</h3>
    <div class="finish-job-list">
      <van-empty v-if="videoRecords.length === 0"
                 image="https://fastly.jsdelivr.net/npm/@vant/assets/custom-empty-image.png"
                 image-size="80"
                 description="暂无记录"
      />
      <div v-else>
        <div v-for="(video, index) in videoRecords" :key="index" class="video-container">
          <div class="video-info">
            <span class="video-number">{{ index + 1 }}</span>
            <span class="video-title">{{ formatDateUsingMoment(video.created_at) }}</span>
          </div>

          <video class="video-player" controls :poster="video.pre_url" :style="{ width: '100%', height: '50%' }">
            <!-- <source :src="video.video_url" type="video/mp4"> -->
            <source :src="`${video.video_url}?t=${new Date().getTime()}`" type="video/mp4">
            您的浏览器不支持视频标签。
          </video>
        </div>
      </div>
    </div>
  </div>

</template>

<script setup>
import {onMounted, ref} from "vue";
import {getSessionId} from "@/store/session";
import { Toast} from 'vant';

import {
  showConfirmDialog,
  showFailToast,
  showNotify,
  showToast,
  showDialog,
  showImagePreview,
  showSuccessToast
} from "vant";
import {httpGet, httpPost} from "@/utils/http";
import {checkSession} from "@/action/session"
import {useRouter} from "vue-router";

const params = ref({
  content: '',
  character: '',
  background: '',
  language: '中文',
});



// {
//                         "roleStyle": "business",
//                         "key": "商业",
//                         "val": "https://www.ifaceai.com/img/ai/role/video/harry-business-transparent.png"
//                     },
//                     {
//                         "roleStyle": "casual",
//                         "key": "随意",
//                         "val": "https://www.ifaceai.com/img/ai/role/video/harry-casual-transparent.png"
//                     },
//                     {
//                         "roleStyle": "youthful",
//                         "key": "年轻",
//                         "val": "https://www.ifaceai.com/img/ai/role/video/harry-youthful-transparent.png"
//                     }

// 形象图
const characters = ref([
  { value: 'casual-sitting', text: '随意坐着', img: 'https://www.ifaceai.com/img/ai/role/video/lisa-casual-sitting.png' },
  { value: 'graceful-sitting', text: '优美坐姿', img: 'https://www.ifaceai.com/img/ai/role/video/lisa-graceful-sitting.png' },
  { value: 'graceful-standing', text: '优美站姿', img: 'https://www.ifaceai.com/img/ai/role/video/lisa-graceful-standing.png' },

  { value: 'business', text: '商业', img: 'https://www.ifaceai.com/img/ai/role/video/harry-business-transparent.png' },
  { value: 'casual', text: '随意', img: 'https://www.ifaceai.com/img/ai/role/video/harry-casual-transparent.png' },
  { value: 'youthful', text: '年轻', img: 'https://www.ifaceai.com/img/ai/role/video/harry-youthful-transparent.png' },

]);


// 背景图
const backgrounds = ref([
  { value: 'https://img.erpweb.eu.org/imgs/2024/09/ba23b56197b6a7f7.jpg', text: '潮流', img: 'https://img.erpweb.eu.org/imgs/2024/09/ba23b56197b6a7f7.jpg' },
  { value: 'https://img.erpweb.eu.org/imgs/2024/09/d470aeea2490f4ff.jpg', text: '海边', img: 'https://img.erpweb.eu.org/imgs/2024/09/d470aeea2490f4ff.jpg' },
  { value: 'https://img.erpweb.eu.org/imgs/2024/09/40a21150707f132f.jpg', text: '室内', img: 'https://img.erpweb.eu.org/imgs/2024/09/40a21150707f132f.jpg' },
  
]);

//语言图
const languageOptions = ref([
        { label: '普通话(男)', value: 'zh-CN-YunxiNeural' },
        { label: '普通话(女)', value: 'zh-CN-XiaoxiaoNeural' }  

]);

//selectedLanguage: 'zh-CN-YunxiNeural',
const selectCharacter = (item) => {
  params.value.character = item.value;
};
const selectBackground = (item) => {
  params.value.background = item.value;
};


const loading = ref(false);
const finished = ref(false);
const showVideo = ref(false);
const currentVideo = ref('');
const roleList = ref([])
const haveMint = ref(false)



const runningJobs = ref([]);
const videoRecords = ref([]);

const generateVideo = async () => {
  if (!validateForm()) return;

 

  const role_name = ['casual-sitting', 'graceful-sitting', 'graceful-standing'].includes(params.value.character) ? "小丽" : "小特";
  //console.log( params.value.character);
 
  try {
    const response = await httpPost("/api/digi/createDigi", {
      ...params.value,
      role_name: role_name,
      session_id: getSessionId(),
    });
    showToast("AI数字人提交成功，等待任务完成");
    // Optionally, start polling or WebSocket for job updates
    startPollingJobs();
  } catch (error) {
    showFailToast("任务推送失败：" + error.message);
  }
};

const startPollingJobs = () => {
  // Implement polling logic to fetch job updates periodically
  fetchRunningJobs();
  fetchFinishJobs();
};

const refreshData = () => {
  fetchRunningJobs();
  fetchFinishJobs();
};

onMounted(() => {
  // Fetch initial data when component mounts
  refreshData();

  // Start polling every 30 seconds (adjust interval as needed)
  setInterval(refreshData, 10000); // 30 seconds
});

const fetchRunningJobs = () => {
  httpGet(`/api/digi/jobs?status=0`)
    .then(res => {
      runningJobs.value = res.data;
    })
    .catch(error => {
      console.error("Failed to fetch running jobs:", error);
    });
};

const fetchFinishJobs = () => {
  httpGet(`/api/digi/jobs?status=1`)
    .then(res => {
      // videoRecords.value = []; // 清空视频记录
      // videoRecords.value = res.data; // 重新赋值
      
      videoRecords.value = []; // 清空视频记录
      videoRecords.value = res.data.map(video => ({
        ...video,
        video_url: `${video.video_url}?t=${new Date().getTime()}` // 添加时间戳
      })); // 重新赋值并添加时间戳
      
      //videoRecords.value = res.data;
    })
    .catch(error => {
      console.error("Failed to fetch finished jobs:", error);
    });
};

const formatDateUsingMoment = (dateString) => {

const options = { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' };
  return new Date(dateString).toLocaleDateString('zh-CN', options).replace(/\//g, '-').replace(',', '');
//  return moment(dateString).format('YYYY-MM-DD HH:mm:ss');
};
const validateForm = () => {
  // Validate form inputs before submission
  if (params.value.content === '') {
    showFailToast("请输入文案！");
    return false;
  }
  if (params.value.character === '') {
    showFailToast("请选择形象！");
    return false;
  }
  if (params.value.background === '') {
    showFailToast("请选择背景！");
    return false;
  }
  if (params.value.language === '中文' || params.value.language === '') {
    showFailToast("请选择语言！");
    return false;
  }
  return true;
};


onMounted(() => {
  // Fetch initial data when component mounts
  fetchRunningJobs();
  fetchFinishJobs();
});




</script>

<style scoped>
.digital-human-generator {
  padding: 20px;
}

.option {
  text-align: center;
  cursor: pointer;
  border: 1px solid #e8e8e8;
  border-radius: 4px;
  padding: 10px;
}

.option.active {
  border: 2px solid #1989fa;
}

.text-line {
  margin: 10px 0;
  font-weight: bold;
}



.running-job-list {
  font-family: 'Arial', sans-serif;
}

.video-container {
  margin-bottom: 20px;
  border: 1px solid #ddd;
  padding: 10px;
  border-radius: 8px;
  box-shadow: 0 2px 6px rgba(0,0,0,0.1);
}

.video-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  padding-bottom: 10px;
  border-bottom: 1px solid #eee;
}

.video-number {
  font-size: 16px;
  font-weight: bold;
  color: #333;
}

.video-title {
  font-size: 18px;
  color: #007BFF;
  flex-grow: 1;
  text-align: center;
}

.video-date {
  font-size: 14px;
  color: #666;
}

.video-poster {
  width: 100%;
  height: auto;
  border-radius: 4px;
}

.video-player {
  width: 100%;
  border-radius: 4px;
  outline: none;

}






/* 生成中 */
.digital-human-generator {
  padding: 20px;
}

.task-in-queue, .task-in-progress {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 120px;
  border: 1px solid #ddd;
  border-radius: 8px;
  box-shadow: 0 2px 6px rgba(0,0,0,0.1);
}

.task-in-queue .icon, .task-in-progress .icon {
  font-size: 24px;
  margin-right: 8px;
}

.task-in-queue .text, .task-in-progress .text {
  font-size: 18px;
  color: #333;
}
</style>