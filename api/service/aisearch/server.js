/**
 * AI搜 Node 侧车：封装 collector/jisou-search-service，供 Go /api/ai-search 转发。
 * 启动：cd api/service/aisearch && npm install && npm start
 */
const http = require("http");
const fs = require("fs");
const path = require("path");
const { URL } = require("url");

// 预加载 .env（config.js 内也会再扫一遍）
(() => {
  const envPath = path.join(__dirname, ".env");
  if (!fs.existsSync(envPath)) return;
  for (const line of fs.readFileSync(envPath, "utf8").split(/\r?\n/)) {
    const t = line.trim();
    if (!t || t.startsWith("#")) continue;
    const eq = t.indexOf("=");
    if (eq <= 0) continue;
    const k = t.slice(0, eq).trim();
    let v = t.slice(eq + 1).trim();
    if ((v.startsWith('"') && v.endsWith('"')) || (v.startsWith("'") && v.endsWith("'"))) v = v.slice(1, -1);
    if (!process.env[k]) process.env[k] = v;
  }
})();

const svc = require("./collector/jisou-search-service");
const { PUBLIC_ROOT } = require("./collector/save-media");

const PORT = Number(process.env.AISEARCH_PORT || 5690);
const API_BASE = "/api/ai-search";

function sendJson(res, status, body) {
  const data = JSON.stringify(body);
  res.writeHead(status, {
    "Content-Type": "application/json; charset=utf-8",
    "Cache-Control": "no-store"
  });
  res.end(data);
}

function readJson(req) {
  return new Promise((resolve, reject) => {
    const chunks = [];
    req.on("data", (c) => chunks.push(c));
    req.on("end", () => {
      if (!chunks.length) return resolve({});
      try {
        resolve(JSON.parse(Buffer.concat(chunks).toString("utf8")));
      } catch (e) {
        reject(e);
      }
    });
    req.on("error", reject);
  });
}

function captchaPayload(captcha, extra = {}) {
  return {
    ...extra,
    captcha: {
      challengeId: captcha.challengeId,
      prompt: captcha.prompt,
      options: captcha.options,
      expiresInSec: captcha.expiresInSec,
      imageUrl: `${API_BASE}/captcha/${encodeURIComponent(captcha.challengeId)}/image`,
      ...(captcha.imageDataUrl ? { imageDataUrl: captcha.imageDataUrl } : {})
    }
  };
}

function mapError(err) {
  const mapped = svc.mapGramError(err);
  const code = err?.code || mapped.code;
  const message = err?.message || mapped.message;
  return { code, message, captcha: err?.captcha, query: err?.query };
}

async function handleSearch(req, res) {
  svc.preemptLowPriorityWork?.("search_request");
  let body;
  try {
    body = await readJson(req);
  } catch {
    return sendJson(res, 400, { ok: false, error: "invalid_json" });
  }
  const q = String(body.q ?? "").trim();
  if (!q) return sendJson(res, 400, { ok: false, error: "missing_query" });

  try {
    const result = await svc.searchJisouChannels(q, { webCaptcha: true, signal: req.signal });
    return sendJson(res, 200, {
      ok: true,
      ...result,
      channelCount: result.channels?.length ?? 0,
      cached: false
    });
  } catch (err) {
    const { code, message, captcha, query } = mapError(err);
    if (code === "JISOU_CAPTCHA_REQUIRED" && captcha) {
      return sendJson(res, 428, captchaPayload(captcha, { ok: false, error: code, message, query: query || q }));
    }
    const status =
      code === "NO_SESSION" || code === "SESSION_REVOKED"
        ? 503
        : code === "JISOU_REPLY_TIMEOUT" || code === "FLOOD_WAIT"
          ? 504
          : 500;
    return sendJson(res, status, { ok: false, error: code, message });
  }
}

async function handleCaptchaSolve(req, res) {
  let body;
  try {
    body = await readJson(req);
  } catch {
    return sendJson(res, 400, { ok: false, error: "invalid_json" });
  }
  const challengeId = String(body.challengeId ?? "").trim();
  const answer = String(body.answer ?? "").trim();
  if (!challengeId || !answer) return sendJson(res, 400, { ok: false, error: "missing_params" });

  try {
    const result = await svc.solveJisouCaptchaAndSearch(challengeId, answer);
    return sendJson(res, 200, {
      ok: true,
      ...result,
      channelCount: result.channels?.length ?? 0,
      cached: false
    });
  } catch (err) {
    const { code, message, captcha, query } = mapError(err);
    if (code === "JISOU_CAPTCHA_REQUIRED" && captcha) {
      return sendJson(res, 428, captchaPayload(captcha, { ok: false, error: code, message, query }));
    }
    const status =
      code === "JISOU_CAPTCHA_EXPIRED" || code === "JISOU_CAPTCHA_INVALID_ANSWER"
        ? 400
        : code === "NO_SESSION" || code === "SESSION_REVOKED"
          ? 503
          : 500;
    return sendJson(res, status, { ok: false, error: code, message });
  }
}

async function handleAction(req, res) {
  svc.preemptLowPriorityWork?.("search_action");
  let body;
  try {
    body = await readJson(req);
  } catch {
    return sendJson(res, 400, { ok: false, error: "invalid_json" });
  }
  const replyMessageId = Number(body.replyMessageId) || 0;
  const callback = String(body.callback ?? "").trim();
  const text = String(body.text ?? "").trim();
  const query = String(body.query ?? "").trim();
  if (replyMessageId <= 0 || (!callback && !text)) {
    return sendJson(res, 400, { ok: false, error: "missing_params", message: "缺少操作参数" });
  }

  try {
    const result = await svc.clickJisouSearchButton({
      replyMessageId,
      callback: callback || undefined,
      text: text || undefined,
      query,
      webCaptcha: true
    });
    return sendJson(res, 200, {
      ok: true,
      ...result,
      channelCount: result.channels?.length ?? 0,
      cached: false
    });
  } catch (err) {
    const { code, message, captcha, query: q } = mapError(err);
    if (code === "JISOU_CAPTCHA_REQUIRED" && captcha) {
      return sendJson(res, 428, captchaPayload(captcha, { ok: false, error: code, message, query: q || query }));
    }
    const status =
      code === "JISOU_MESSAGE_NOT_FOUND" || code === "JISOU_BUTTON_NOT_FOUND"
        ? 404
        : code === "NO_SESSION" || code === "SESSION_REVOKED"
          ? 503
          : code === "JISOU_SEARCH_UPDATE_TIMEOUT" || code === "FLOOD_WAIT"
            ? 504
            : 500;
    return sendJson(res, status, { ok: false, error: code, message });
  }
}

function handleCaptchaImage(res, challengeId) {
  const image = svc.getJisouCaptchaImage(challengeId);
  if (!image?.buffer) return sendJson(res, 404, { ok: false, error: "captcha_not_found" });
  res.writeHead(200, {
    "Content-Type": image.mime || "image/jpeg",
    "Cache-Control": "private, no-store"
  });
  res.end(image.buffer);
}

async function handleChannel(req, res, url) {
  const username = String(url.searchParams.get("username") || url.searchParams.get("u") || "").trim();
  const rawSearch = String(url.searchParams.get("search") || url.searchParams.get("q") || "").trim();
  const limit = Math.min(50, Math.max(1, Number(url.searchParams.get("limit")) || 20));
  const messageId = Number(url.searchParams.get("messageId") || url.searchParams.get("mid") || 0);
  const search = messageId > 0 ? "" : rawSearch;
  const includeContext =
    url.searchParams.get("includeContext") === "1" || url.searchParams.get("includeContext") === "true";

  if (!username) return sendJson(res, 400, { ok: false, error: "missing_username" });

  try {
    const result = await svc.fetchChannelMessages(username, {
      limit,
      search: search || undefined,
      messageId: messageId > 0 ? messageId : undefined,
      includeContext: messageId > 0 ? includeContext : undefined,
      signal: req.signal
    });
    return sendJson(res, 200, {
      ok: true,
      ...result,
      alreadyViewed: true,
      needsViewBill: false,
      viewBilled: false
    });
  } catch (err) {
    const { code, message } = mapError(err);
    if (code === "REQUEST_ABORTED") {
      res.writeHead(499);
      return res.end();
    }
    const status =
      code === "CHANNEL_PRIVATE" || code === "USERNAME_INVALID"
        ? 403
        : code === "MESSAGE_NOT_FOUND"
          ? 404
          : 500;
    return sendJson(res, status, { ok: false, error: code, message });
  }
}

async function handleMediaBatch(req, res) {
  let body;
  try {
    body = await readJson(req);
  } catch {
    return sendJson(res, 400, { ok: false, error: "invalid_json" });
  }
  const username = String(body.username ?? "").trim();
  const messageIds = Array.isArray(body.messageIds)
    ? body.messageIds.map((id) => Math.floor(Number(id))).filter((id) => id > 0)
    : [];
  const thumb = body.thumb !== false;
  if (!username || !messageIds.length) return sendJson(res, 400, { ok: false, error: "missing_params" });

  try {
    const result = await svc.resolveMessageMediaBatch(username, messageIds, {
      thumb,
      signal: req.signal
    });
    return sendJson(res, 200, {
      ok: true,
      username: result.username,
      media: result.media,
      partial: Boolean(result.partial)
    });
  } catch (err) {
    const { code, message } = mapError(err);
    if (code === "REQUEST_ABORTED") {
      res.writeHead(499);
      return res.end();
    }
    return sendJson(res, 500, { ok: false, error: code, message });
  }
}

async function handleMediaGet(req, res, url) {
  const username = String(url.searchParams.get("username") || url.searchParams.get("u") || "").trim();
  const messageId = Number(url.searchParams.get("messageId") || url.searchParams.get("mid") || 0);
  const thumbRaw = url.searchParams.get("thumb");
  const thumb = thumbRaw === null ? undefined : thumbRaw !== "0";
  if (!username || messageId <= 0) return sendJson(res, 400, { ok: false, error: "missing_params" });

  try {
    if (thumb !== false) {
      const cachedThumb = await svc.getCachedThumbMediaUrl?.(username, messageId);
      if (cachedThumb?.url) {
        res.writeHead(302, { Location: cachedThumb.url, "Cache-Control": "public, max-age=86400" });
        return res.end();
      }
      return sendJson(res, 410, {
        ok: false,
        error: "THUMB_USE_BATCH",
        message: "缩略图请使用 POST /media/batch"
      });
    }
    const cachedFull = await svc.getCachedFullMediaUrl(username, messageId);
    if (cachedFull?.url) {
      res.writeHead(302, { Location: cachedFull.url, "Cache-Control": "public, max-age=86400" });
      return res.end();
    }
    const result = await svc.resolveMessageMedia(username, messageId, { thumb: false, signal: req.signal });
    if (result?.url) {
      res.writeHead(302, { Location: result.url, "Cache-Control": "public, max-age=3600" });
      return res.end();
    }
    return sendJson(res, 404, { ok: false, error: "media_not_found" });
  } catch (err) {
    const { code, message } = mapError(err);
    return sendJson(res, 500, { ok: false, error: code, message });
  }
}

async function handlePlayInfo(req, res, url) {
  const username = String(url.searchParams.get("username") || "").trim();
  const messageId = Number(url.searchParams.get("messageId") || 0);
  if (!username || messageId <= 0) return sendJson(res, 400, { ok: false, error: "missing_params" });
  try {
    const info = await svc.resolveVideoPlayInfo(username, messageId, { signal: req.signal });
    return sendJson(res, 200, { ok: true, ...info });
  } catch (err) {
    const { code, message } = mapError(err);
    return sendJson(res, 500, { ok: false, error: code, message });
  }
}

async function handleMediaStream(req, res, url) {
  const username = String(url.searchParams.get("username") || "").trim();
  const messageId = Number(url.searchParams.get("messageId") || 0);
  if (!username || messageId <= 0) return sendJson(res, 400, { ok: false, error: "missing_params" });

  try {
    const result = await svc.createVideoStreamResponse(username, messageId, {
      signal: req.signal,
      rangeHeader: req.headers.range || null
    });
    if (result?.redirect) {
      res.writeHead(302, { Location: result.redirect });
      return res.end();
    }
    if (!result?.stream) {
      return sendJson(res, 500, { ok: false, error: "stream_unavailable" });
    }

    const headers = {
      "Content-Type": result.mime || "video/mp4",
      "Cache-Control": "public, max-age=3600",
      "Accept-Ranges": "bytes",
      "X-Accel-Buffering": "no"
    };
    if (result.contentLength != null) headers["Content-Length"] = String(result.contentLength);
    if (result.contentRange) headers["Content-Range"] = result.contentRange;

    const status = result.status || (result.contentRange ? 206 : 200);
    res.writeHead(status, headers);

    const webStream = result.stream;
    if (typeof webStream.getReader === "function") {
      const reader = webStream.getReader();
      const pump = async () => {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          if (!res.write(Buffer.from(value))) {
            await new Promise((resolve) => res.once("drain", resolve));
          }
        }
        res.end();
      };
      req.on("close", () => {
        try {
          reader.cancel("client_abort");
        } catch {
          /* ignore */
        }
      });
      await pump();
      return;
    }
    if (typeof webStream.pipe === "function") {
      webStream.pipe(res);
      return;
    }
    return sendJson(res, 500, { ok: false, error: "stream_unavailable" });
  } catch (err) {
    const { code, message } = mapError(err);
    if (!res.headersSent) {
      return sendJson(res, 500, { ok: false, error: code, message });
    }
    try {
      res.end();
    } catch {
      /* ignore */
    }
  }
}

function handleFiles(req, res, pathname) {
  const rel = pathname.replace(/^\/files\/?/, "").replace(/^\/+/, "");
  if (!rel || rel.includes("..")) return sendJson(res, 400, { ok: false, error: "bad_path" });
  const fsPath = path.join(PUBLIC_ROOT, rel);
  if (!fsPath.startsWith(PUBLIC_ROOT) || !fs.existsSync(fsPath) || !fs.statSync(fsPath).isFile()) {
    return sendJson(res, 404, { ok: false, error: "not_found" });
  }
  const ext = path.extname(fsPath).toLowerCase();
  const mime =
    ext === ".jpg" || ext === ".jpeg"
      ? "image/jpeg"
      : ext === ".png"
        ? "image/png"
        : ext === ".webp"
          ? "image/webp"
          : ext === ".mp4"
            ? "video/mp4"
            : "application/octet-stream";
  res.writeHead(200, { "Content-Type": mime, "Cache-Control": "public, max-age=86400" });
  fs.createReadStream(fsPath).pipe(res);
}

async function onRequest(req, res) {
  const host = req.headers.host || `127.0.0.1:${PORT}`;
  const url = new URL(req.url || "/", `http://${host}`);
  let pathname = url.pathname;
  // 兼容 Go 转发带 /api/ai-search 前缀或裸路径
  if (pathname.startsWith(API_BASE)) {
    pathname = pathname.slice(API_BASE.length) || "/";
  }
  if (!pathname.startsWith("/")) pathname = `/${pathname}`;

  try {
    if (req.method === "GET" && pathname === "/health") {
      return sendJson(res, 200, { ok: true, service: "aisearch" });
    }
    if (req.method === "POST" && pathname === "/search") return await handleSearch(req, res);
    if (req.method === "POST" && pathname === "/action") return await handleAction(req, res);
    if (req.method === "POST" && pathname === "/captcha/solve") return await handleCaptchaSolve(req, res);
    if (req.method === "GET" && pathname.startsWith("/captcha/") && pathname.endsWith("/image")) {
      const id = decodeURIComponent(pathname.slice("/captcha/".length, -"/image".length));
      return handleCaptchaImage(res, id);
    }
    if (req.method === "GET" && pathname === "/channel") return await handleChannel(req, res, url);
    if (req.method === "POST" && pathname === "/media/batch") return await handleMediaBatch(req, res);
    if (req.method === "GET" && pathname === "/media") return await handleMediaGet(req, res, url);
    if (req.method === "GET" && pathname === "/media/play-info") return await handlePlayInfo(req, res, url);
    if (req.method === "GET" && pathname === "/media/stream") return await handleMediaStream(req, res, url);
    if (req.method === "GET" && pathname.startsWith("/files/")) return handleFiles(req, res, pathname);
    // stub history / quota / view-bill for API shape compatibility
    if (req.method === "GET" && pathname === "/history") return sendJson(res, 200, { ok: true, keywords: [] });
    if (req.method === "GET" && pathname === "/quota") {
      return sendJson(res, 200, {
        ok: true,
        quota: {
          used: 0,
          limit: 0,
          remaining: Number.MAX_SAFE_INTEGER,
          searchBonus: 0,
          hasIdentity: true,
          publicId: null,
          unlimited: true
        }
      });
    }
    if (req.method === "POST" && pathname === "/view-bill") {
      return sendJson(res, 200, { ok: true, billed: false, alreadyViewed: true });
    }
    return sendJson(res, 404, { ok: false, error: "not_found", path: pathname });
  } catch (err) {
    console.error("[aisearch]", err);
    return sendJson(res, 500, { ok: false, error: "internal", message: err?.message || String(err) });
  }
}

fs.mkdirSync(PUBLIC_ROOT, { recursive: true });

const server = http.createServer((req, res) => {
  void onRequest(req, res);
});

server.listen(PORT, "127.0.0.1", () => {
  console.log(`[aisearch] listening on http://127.0.0.1:${PORT}`);
  console.log(`[aisearch] public files: ${PUBLIC_ROOT}`);
});
