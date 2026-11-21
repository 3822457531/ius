import { httpGet } from "@/utils/http";

// 检查用户会话
export function checkSession() {
  return new Promise((resolve, reject) => {
    httpGet('/api/user/session')
      .then(res => {
        resolve(res.data);
      })
      .catch(err => {
        reject(err);
      });
  });
}

// 获取系统信息
export function getSystemInfo() {
  return new Promise((resolve, reject) => {
    httpGet('/api/config/get?key=system')
      .then(res => {
        resolve(res.data);
      })
      .catch(err => {
        reject(err);
      });
  });
}

