# AI搜侧车（GramJS / 极搜）

手机端「AI搜」由：

1. **Go** `/api/ai-search/*`（需登录 JWT）转发到本侧车  
2. **本目录 Node 服务** 调用 `collector/` 内极搜逻辑，媒体默认上传 **Cloudflare R2**  
3. **Vue** `web/src/views/mobile/AiSearch.vue` 展示

## 启动

```bash
cd api/service/aisearch
cp .env.example .env   # 填写 TG_* 与 R2_*
npm install
# 首次需登录采集号（生成 session.txt / back.txt）
npm run login
npm run login:media
npm start              # 默认 127.0.0.1:5690
```

### R2（推荐）

在 `.env` 中配置（对应 `sitesettings` 表）：

```env
MEDIA_STORAGE=r2
R2_ACCOUNT_ID=...
R2_BUCKET_NAME=...
R2_PUBLIC_BASE_URL=https://xxx.r2.dev
R2_ACCESS_KEY_ID=...
R2_SECRET_ACCESS_KEY=...
```

上传成功后返回公网 URL（如 `https://xxx.r2.dev/uploads/...`），浏览器直连 CDN，无需再经 Go 代理文件。  
R2 失败时自动回退到本地 `public/` + `/api/ai-search/files/...`。

Go `config.toml`：

```toml
[AiSearchConfig]
  Enabled = true
  ServiceURL = "http://127.0.0.1:5690"
  TimeoutSec = 120
```

`tg/` 目录仅作参考，不参与正式部署。
