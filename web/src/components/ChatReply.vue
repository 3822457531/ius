<template>
  <div class="chat-line chat-line-reply">
    <div class="chat-line-inner">
      <div class="chat-icon">
        <img :src="icon" alt="ChatGPT">
      </div>

      <div class="chat-item">
        <div ref="contentRef" class="content markdown-chat" v-html="content"></div>
        <div class="bar" v-if="createdAt !== ''">
          <span class="bar-item"><el-icon><Clock/></el-icon> {{ createdAt }}</span>
          <span class="bar-item">Tokens: {{ tokens }}</span>
          <el-tooltip
              class="box-item"
              effect="dark"
              content="复制回答"
              placement="bottom"
          >
            <el-button type="info" class="copy-reply" :data-clipboard-text="orgContent">
              <el-icon>
                <DocumentCopy/>
              </el-icon>
            </el-button>
          </el-tooltip>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import {defineComponent, nextTick} from "vue"
import {Clock, DocumentCopy, Position} from "@element-plus/icons-vue";
import { enhanceMarkdownRoot } from "@/utils/markdownEnhance";

export default defineComponent({
  name: 'ChatReply',
  components: {Position, Clock, DocumentCopy},
  props: {
    content: {
      type: String,
      default: '',
    },
    orgContent: {
      type: String,
      default: '',
    },
    createdAt: {
      type: String,
      default: '',
    },
    tokens: {
      type: Number,
      default: 0,
    },
    icon: {
      type: String,
      default: 'images/gpt-icon.png',
    }
  },
  data() {
    return {
      finalTokens: this.tokens
    }
  },
  watch: {
    content() {
      nextTick(() => this.enhanceContent())
    },
  },
  mounted() {
    this.enhanceContent()
  },
  updated() {
    this.enhanceContent()
  },
  methods: {
    enhanceContent() {
      enhanceMarkdownRoot(this.$refs.contentRef)
    },
  },
})
</script>

<style lang="stylus">
@import "@/assets/css/markdown-chat.styl"

.common-layout {
  .chat-line-reply {
    justify-content: center;
    background-color: rgba(247, 247, 248, 1);
    width 100%
    padding-bottom: 1.5rem;
    padding-top: 1.5rem;
    border-bottom: 1px solid #ececf1;

    .chat-line-inner {
      display flex;
      width 100%;
      max-width 900px;
      padding-left 10px;

      .chat-icon {
        margin-right 20px;

        img {
          width: 36px;
          height: 36px;
          border-radius: 10px;
          padding: 1px;
        }
      }

      .chat-item {
        position: relative;
        flex 1
        min-width 0
        padding: 0 0 0 5px;
        overflow: hidden;

        .content {
          min-height 20px;
          word-break break-word;
          padding: 4px 8px 8px 0;
          max-width 100%
        }

        .bar {
          padding 10px 0 0;

          .bar-item {
            background-color #ececf1;
            color #888
            padding 3px 8px;
            margin-right 10px;
            border-radius 8px;
            font-size 12px

            .el-icon {
              position relative
              top 2px;
            }
          }

          .el-button {
            height 20px
            padding 5px 2px;
          }
        }

      }

      .tool-box {
        font-size 16px;

        .el-button {
          height 20px
          padding 5px 2px;
        }
      }
    }

  }
}

</style>
