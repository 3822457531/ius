/**
 * AI搜媒体存储：优先 Cloudflare R2，失败回退本地 public/
 * R2 配置来自环境变量（对齐 SiteSettings 字段）
 */
const fs = require("fs");
const path = require("path");
const { pipeline } = require("stream/promises");
const { PassThrough, Transform } = require("stream");
const { PutObjectCommand, S3Client } = require("@aws-sdk/client-s3");
const { Upload } = require("@aws-sdk/lib-storage");

const R2_REGION = "auto";
const SETTINGS_CACHE_MS = Math.max(5000, Number(process.env.TG_MEDIA_SETTINGS_CACHE_MS) || 60000);

const PUBLIC_ROOT = path.resolve(
  process.env.AISEARCH_PUBLIC_DIR || path.join(__dirname, "..", "public")
);
const PUBLIC_URL_PREFIX = String(process.env.AISEARCH_PUBLIC_URL_PREFIX || "/api/ai-search/files").replace(
  /\/+$/,
  ""
);

/** @type {{ settings: object|null, at: number, s3: import('@aws-sdk/client-s3').S3Client|null, s3Key: string|null }} */
const runtime = { settings: null, at: 0, s3: null, s3Key: null };

function trimBaseUrl(base) {
  return String(base || "").replace(/\/+$/, "");
}

function buildObjectKey(subPath) {
  return `uploads/${String(subPath || "")
    .replace(/^\/+/, "")
    .replace(/\\/g, "/")}`;
}

function loadSettingsFromEnv() {
  const mediaStorage = String(process.env.MEDIA_STORAGE || process.env.AISEARCH_MEDIA_STORAGE || "r2")
    .trim()
    .toLowerCase();
  return {
    mediaStorage: mediaStorage === "local" ? "local" : "r2",
    r2AccountId: String(process.env.R2_ACCOUNT_ID || "").trim(),
    r2BucketName: String(process.env.R2_BUCKET_NAME || "").trim(),
    r2PublicBaseUrl: String(process.env.R2_PUBLIC_BASE_URL || "").trim(),
    r2AccessKeyId: String(process.env.R2_ACCESS_KEY_ID || "").trim(),
    r2SecretAccessKey: String(process.env.R2_SECRET_ACCESS_KEY || "").trim()
  };
}

async function getSiteSettingsCached() {
  const now = Date.now();
  if (runtime.settings && now - runtime.at < SETTINGS_CACHE_MS) {
    return runtime.settings;
  }
  runtime.settings = loadSettingsFromEnv();
  runtime.at = now;
  runtime.s3 = null;
  runtime.s3Key = null;
  return runtime.settings;
}

function resolveR2Credentials(settings) {
  const accessKeyId = process.env.R2_ACCESS_KEY_ID?.trim() || settings.r2AccessKeyId?.trim() || "";
  const secretAccessKey =
    process.env.R2_SECRET_ACCESS_KEY?.trim() || settings.r2SecretAccessKey?.trim() || "";
  if (!accessKeyId || !secretAccessKey) return null;
  return { accessKeyId, secretAccessKey };
}

function isR2Ready(settings) {
  const s = settings || loadSettingsFromEnv();
  if (s.mediaStorage !== "r2") return false;
  const accountId = process.env.R2_ACCOUNT_ID?.trim() || s.r2AccountId?.trim();
  const bucket = s.r2BucketName?.trim();
  const pub = s.r2PublicBaseUrl?.trim();
  if (!accountId || !bucket || !pub) return false;
  return Boolean(resolveR2Credentials(s));
}

function getR2Client(settings) {
  const s = settings || loadSettingsFromEnv();
  const accountId = process.env.R2_ACCOUNT_ID?.trim() || s.r2AccountId.trim();
  const creds = resolveR2Credentials(s);
  const bucket = s.r2BucketName.trim();
  const cacheKey = `${accountId}:${bucket}:${creds?.accessKeyId || ""}`;
  if (runtime.s3 && runtime.s3Key === cacheKey) {
    return { client: runtime.s3, bucket, accountId, creds };
  }
  const client = new S3Client({
    region: R2_REGION,
    endpoint: `https://${accountId}.r2.cloudflarestorage.com`,
    credentials: creds,
    forcePathStyle: false,
    maxAttempts: 3
  });
  runtime.s3 = client;
  runtime.s3Key = cacheKey;
  return { client, bucket, accountId, creds };
}

function r2MultipartPartSize() {
  const mb = Number(process.env.R2_MULTIPART_PART_SIZE_MB) || 8;
  return Math.min(32, Math.max(5, Math.round(mb))) * 1024 * 1024;
}

function r2MultipartQueueSize() {
  return Math.min(8, Math.max(2, Number(process.env.R2_MULTIPART_QUEUE_SIZE) || 4));
}

function localPublicUrlForKey(key) {
  return `${PUBLIC_URL_PREFIX}/${key.replace(/^\/+/, "")}`;
}

function absoluteFsPath(key) {
  return path.join(PUBLIC_ROOT, key.replace(/^\/+/, "").replace(/\\/g, "/"));
}

async function saveLocalFile(bufferOrStream, key, opts = {}) {
  const fsPath = absoluteFsPath(key);
  const partPath = `${fsPath}.part`;
  fs.mkdirSync(path.dirname(fsPath), { recursive: true });
  const { onUploadProgress, expectedBytes, isStream } = opts;

  if (!isStream) {
    fs.writeFileSync(fsPath, bufferOrStream);
    const total = bufferOrStream?.length || 0;
    onUploadProgress?.(total, total);
    return localPublicUrlForKey(key);
  }

  try {
    let written = 0;
    const counter =
      onUploadProgress &&
      new Transform({
        transform(chunk, _enc, cb) {
          written += chunk.length;
          onUploadProgress(written, expectedBytes || undefined);
          cb(null, chunk);
        }
      });
    const sink = fs.createWriteStream(partPath);
    if (counter) {
      await pipeline(bufferOrStream, counter, sink);
    } else {
      await pipeline(bufferOrStream, sink);
    }
    if (fs.existsSync(fsPath)) fs.unlinkSync(fsPath);
    fs.renameSync(partPath, fsPath);
    return localPublicUrlForKey(key);
  } catch (e) {
    try {
      if (fs.existsSync(partPath)) fs.unlinkSync(partPath);
    } catch {
      /* ignore */
    }
    throw e;
  }
}

async function saveMediaStream(readable, subPath, contentType, opts = {}) {
  const settings = await getSiteSettingsCached();
  const key = buildObjectKey(subPath);
  const { onUploadProgress, expectedBytes } = opts;

  if (isR2Ready(settings)) {
    const { client, bucket } = getR2Client(settings);
    try {
      const upload = new Upload({
        client,
        params: {
          Bucket: bucket,
          Key: key,
          Body: readable,
          ContentType: contentType || "application/octet-stream"
        },
        queueSize: r2MultipartQueueSize(),
        partSize: r2MultipartPartSize(),
        leavePartsOnError: false
      });
      if (onUploadProgress) {
        upload.on("httpUploadProgress", (progress) => {
          onUploadProgress(Number(progress.loaded) || 0, Number(progress.total) || expectedBytes || undefined);
        });
      }
      await upload.done();
      return `${trimBaseUrl(settings.r2PublicBaseUrl.trim())}/${key}`;
    } catch (e) {
      console.warn("[aisearch/save-media] R2 stream 失败，回退本地:", e.message);
    }
  }

  return saveLocalFile(readable, key, { onUploadProgress, expectedBytes, isStream: true });
}

async function saveMediaFromChunkIter(chunkIter, subPath, contentType) {
  const pass = new PassThrough();
  const uploadPromise = saveMediaStream(pass, subPath, contentType);
  void (async () => {
    try {
      for await (const chunk of chunkIter) {
        if (!pass.write(chunk)) {
          await new Promise((resolve) => pass.once("drain", resolve));
        }
      }
      pass.end();
    } catch (err) {
      pass.destroy(err instanceof Error ? err : new Error(String(err)));
    }
  })();
  return uploadPromise;
}

async function saveMediaBytes(buffer, subPath, contentType, opts = {}) {
  const settings = await getSiteSettingsCached();
  const key = buildObjectKey(subPath);
  const total = buffer?.length || 0;
  opts.onUploadProgress?.(0, total);

  if (isR2Ready(settings)) {
    const { client, bucket } = getR2Client(settings);
    try {
      await client.send(
        new PutObjectCommand({
          Bucket: bucket,
          Key: key,
          Body: buffer,
          ContentType: contentType || "application/octet-stream"
        })
      );
      opts.onUploadProgress?.(total, total);
      return `${trimBaseUrl(settings.r2PublicBaseUrl.trim())}/${key}`;
    } catch (e) {
      console.warn("[aisearch/save-media] R2 失败，回退本地:", e.message);
    }
  }

  return saveLocalFile(buffer, key, { onUploadProgress: opts.onUploadProgress, isStream: false });
}

module.exports = {
  saveMediaBytes,
  saveMediaStream,
  saveMediaFromChunkIter,
  getSiteSettingsCached,
  isR2Ready,
  buildObjectKey,
  trimBaseUrl,
  resolveR2Credentials,
  getR2Client,
  PUBLIC_ROOT,
  PUBLIC_URL_PREFIX,
  absoluteFsPath,
  loadSettingsFromEnv
};
