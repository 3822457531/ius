import { defineStore } from 'pinia';
import { getUserToken } from './session';
import { resolveWsHost } from '@/utils/http';

export const useSharedStore = defineStore('shared', {
  state: () => ({
    socket: {
      conn: null, // WebSocket 连接对象
    },
    sora2MessageHandler: null, // Sora2 消息处理器
  }),

  actions: {
    // 初始化 WebSocket 连接
    initWebSocket(userId = null) {
      if (this.socket.conn && this.socket.conn.readyState === WebSocket.OPEN) {
        console.log('WebSocket 已连接');
        return;
      }

      // 如果没有提供 userId，尝试从 token 中解析
      if (!userId) {
        const token = getUserToken();
        if (token) {
          try {
            const payload = JSON.parse(atob(token.split('.')[1]));
            userId = payload.user_id || payload.id;
          } catch (e) {
            console.warn('无法从 token 中解析 user_id:', e);
          }
        }
      }

      if (!userId) {
        console.warn('用户未登录，无法创建 WebSocket 连接');
        return;
      }

      const host = resolveWsHost();
      const wsUrl = `${host}/api/sora2/client?user_id=${userId}`;
      console.log('正在创建 WebSocket 连接:', wsUrl);

      try {
        this.socket.conn = new WebSocket(wsUrl);

        this.socket.conn.onopen = () => {
          console.log('✅ WebSocket 连接已建立');
        };

        this.socket.conn.onerror = (error) => {
          console.error('❌ WebSocket 连接错误:', error);
        };

        this.socket.conn.onclose = () => {
          console.log('🔌 WebSocket 连接已关闭');
          this.socket.conn = null;
        };
      } catch (error) {
        console.error('创建 WebSocket 连接失败:', error);
      }
    },

    // 关闭 WebSocket 连接
    closeWebSocket() {
      if (this.socket.conn) {
        // 移除消息处理器
        if (this.sora2MessageHandler) {
          this.socket.conn.removeEventListener('message', this.sora2MessageHandler);
          this.sora2MessageHandler = null;
        }
        this.socket.conn.close();
        this.socket.conn = null;
      }
    },
  },
});
