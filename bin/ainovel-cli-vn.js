#!/usr/bin/env node

'use strict';

const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { pipeline } = require('node:stream/promises');
const { Transform } = require('node:stream');
const { spawn, spawnSync } = require('node:child_process');
const https = require('node:https');
const { URL } = require('node:url');

const REPOSITORY = 'JustinNguyen9979/ainovel-cli';
const BINARY_NAME = 'ainovel-cli';
const MAX_REDIRECTS = 5;
const REQUEST_TIMEOUT_MS = 30_000;
const MAX_ARCHIVE_BYTES = 100 * 1024 * 1024;
const MAX_CHECKSUM_BYTES = 1024 * 1024;
const LOCK_WAIT_MS = 120_000;
const LOCK_STALE_MS = 10 * 60 * 1000;
const PACKAGE_UPDATE_MESSAGE = 'This binary is managed by npm. Run `npm install --global ainovel-cli-vn@latest` to update it.';
const RELEASE_HOSTS = new Set(['github.com', 'objects.githubusercontent.com']);

const TARGETS = {
  darwin: {
    x64: { os: 'Darwin', arch: 'x86_64', extension: '.tar.gz' },
    arm64: { os: 'Darwin', arch: 'arm64', extension: '.tar.gz' },
  },
  linux: {
    x64: { os: 'Linux', arch: 'x86_64', extension: '.tar.gz' },
    arm64: { os: 'Linux', arch: 'arm64', extension: '.tar.gz' },
  },
  win32: {
    x64: { os: 'Windows', arch: 'x86_64', extension: '.zip' },
    arm64: { os: 'Windows', arch: 'arm64', extension: '.zip' },
  },
};

class ByteLimitTransform extends Transform {
  constructor(limit, onProgress) {
    super();
    this.limit = limit;
    this.onProgress = onProgress;
    this.total = 0;
  }

  _transform(chunk, encoding, callback) {
    this.total += chunk.length;
    if (this.total > this.limit) {
      callback(new Error(`Downloaded file exceeds ${this.limit} bytes`));
      return;
    }
    this.onProgress?.(this.total);
    callback(null, chunk);
  }
}

function targetFor(platform = process.platform, arch = process.arch) {
  const target = TARGETS[platform]?.[arch];
  if (!target) {
    throw new Error(`Unsupported platform or architecture: ${platform}/${arch}. Supported targets: darwin/linux/win32 on x64/arm64.`);
  }
  return target;
}

function releaseVersion() {
  const packagePath = path.resolve(__dirname, '..', 'package.json');
  const metadata = JSON.parse(fs.readFileSync(packagePath, 'utf8'));
  return normalizeVersion(metadata.version);
}

function normalizeVersion(value) {
  const version = String(value || '').trim().replace(/^v/, '');
  if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$/.test(version)) {
    throw new Error(`Invalid release version: ${value}`);
  }
  return version;
}

function assetName(version, target) {
  return `${BINARY_NAME}_${version}_${target.os}_${target.arch}${target.extension}`;
}

function releaseURL(version, asset) {
  return `https://github.com/${REPOSITORY}/releases/download/v${version}/${asset}`;
}

function parseChecksums(text) {
  const entries = new Map();
  for (const rawLine of String(text).split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith('#')) continue;
    const match = line.match(/^([0-9a-fA-F]{64})\s+\*?(.+?)\s*$/);
    if (!match) continue;
    const name = match[2].trim();
    if (entries.has(name)) {
      throw new Error(`Checksum list contains duplicate entry for ${name}`);
    }
    entries.set(name, match[1].toLowerCase());
  }
  return entries;
}

function checksumFor(text, asset) {
  const checksum = parseChecksums(text).get(asset);
  if (!checksum) {
    throw new Error(`Checksum list does not contain ${asset}`);
  }
  return checksum;
}

async function sha256File(filePath) {
  const hash = crypto.createHash('sha256');
  for await (const chunk of fs.createReadStream(filePath)) {
    hash.update(chunk);
  }
  return hash.digest('hex');
}

async function verifyChecksum(filePath, expected, asset) {
  if (!/^[0-9a-fA-F]{64}$/.test(expected)) {
    throw new Error(`Invalid SHA-256 checksum for ${asset}`);
  }
  const actual = await sha256File(filePath);
  if (actual !== expected.toLowerCase()) {
    throw new Error(`SHA-256 verification failed for ${asset}`);
  }
}

function cacheRoot() {
  if (process.env.AINOVEL_CACHE_DIR) return process.env.AINOVEL_CACHE_DIR;
  if (process.env.XDG_CACHE_HOME) return path.join(process.env.XDG_CACHE_HOME, 'ainovel-cli-vn');
  if (process.platform === 'darwin' && process.env.HOME) return path.join(process.env.HOME, 'Library', 'Caches', 'ainovel-cli-vn');
  if (process.platform === 'win32' && process.env.LOCALAPPDATA) return path.join(process.env.LOCALAPPDATA, 'ainovel-cli-vn', 'cache');
  return path.join(os.homedir(), '.cache', 'ainovel-cli-vn');
}

function cachedBinaryPath(version, target) {
  const binary = process.platform === 'win32' ? `${BINARY_NAME}.exe` : BINARY_NAME;
  return path.join(cacheRoot(), version, `${target.os}-${target.arch}`, binary);
}

function lockPath(version, target) {
  return path.join(cacheRoot(), `${version}-${target.os}-${target.arch}.lock`);
}

function isRegularFile(filePath) {
  try {
    return fs.lstatSync(filePath).isFile();
  } catch {
    return false;
  }
}

function isExecutable(filePath) {
  if (!isRegularFile(filePath)) return false;
  if (process.platform === 'win32') return true;
  try {
    fs.accessSync(filePath, fs.constants.X_OK);
    return true;
  } catch {
    return false;
  }
}

async function cachedBinaryIsValid(filePath) {
  if (!isExecutable(filePath)) return false;
  const manifest = `${filePath}.sha256`;
  let expected;
  try {
    expected = fs.readFileSync(manifest, 'utf8').trim();
  } catch {
    return false;
  }
  if (!/^[0-9a-f]{64}$/.test(expected)) return false;
  try {
    return (await sha256File(filePath)) === expected;
  } catch {
    return false;
  }
}

function sleep(milliseconds) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}

function processIsAlive(pid) {
  if (!Number.isInteger(pid) || pid <= 0) return false;
  try {
    process.kill(pid, 0);
    return true;
  } catch (error) {
    return error.code === 'EPERM';
  }
}

async function acquireLock(filePath) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  const started = Date.now();
  while (true) {
    try {
      fs.mkdirSync(filePath);
      fs.writeFileSync(path.join(filePath, 'owner'), `${process.pid}\n`, { flag: 'wx', mode: 0o600 });
      return () => fs.rmSync(filePath, { recursive: true, force: true });
    } catch (error) {
      if (error.code !== 'EEXIST') throw error;
      try {
        const stat = fs.statSync(filePath);
        const ownerPath = path.join(filePath, 'owner');
        const owner = Number.parseInt(fs.readFileSync(ownerPath, 'utf8'), 10);
        if (Date.now() - stat.mtimeMs > LOCK_STALE_MS && !processIsAlive(owner)) {
          fs.rmSync(filePath, { recursive: true, force: true });
          continue;
        }
      } catch {
        // Another process may be creating the lock metadata; keep waiting.
      }
      if (Date.now() - started >= LOCK_WAIT_MS) {
        throw new Error(`Timed out waiting for another ainovel-cli-vn download: ${filePath}`);
      }
      await sleep(100);
    }
  }
}

function isReleaseHost(hostname) {
  return RELEASE_HOSTS.has(hostname) || hostname.endsWith('.githubusercontent.com');
}

function request(url, onResponse, redirects = 0) {
  const parsed = new URL(url);
  if (parsed.protocol !== 'https:') {
    throw new Error(`Refusing non-HTTPS download URL: ${url}`);
  }
  return new Promise((resolve, reject) => {
    const req = https.get(parsed, { headers: { 'User-Agent': 'ainovel-cli-vn' } }, (response) => {
      if ([301, 302, 303, 307, 308].includes(response.statusCode)) {
        response.resume();
        if (redirects >= MAX_REDIRECTS || !response.headers.location) {
          reject(new Error(`Too many redirects downloading ${url}`));
          return;
        }
        const next = new URL(response.headers.location, parsed);
        if (next.protocol !== 'https:' || !isReleaseHost(next.hostname)) {
          reject(new Error(`Refusing redirect to unexpected host: ${next.hostname}`));
          return;
        }
        request(next.toString(), onResponse, redirects + 1).then(resolve, reject);
        return;
      }
      if (response.statusCode !== 200) {
        response.resume();
        reject(new Error(`Download failed with HTTP ${response.statusCode}: ${url}`));
        return;
      }
      const contentLength = Number(response.headers['content-length'] || 0);
      if (contentLength > 0 && contentLength > MAX_ARCHIVE_BYTES) {
        response.resume();
        reject(new Error(`Download exceeds ${MAX_ARCHIVE_BYTES} bytes: ${url}`));
        return;
      }
      response.setTimeout(REQUEST_TIMEOUT_MS, () => response.destroy(new Error(`Download timed out: ${url}`)));
      Promise.resolve(onResponse(response)).then(resolve, reject);
    });
    req.setTimeout(REQUEST_TIMEOUT_MS, () => req.destroy(new Error(`Download timed out: ${url}`)));
    req.on('error', reject);
  });
}

function downloadText(url) {
  return request(url, (response) => collectLimited(response, MAX_CHECKSUM_BYTES));
}

async function downloadFile(url, destination) {
  try {
    await request(url, (response) => {
      const contentLength = Number(response.headers['content-length'] || 0);
      let lastReported = 0;
      const reportProgress = (downloaded) => {
        if (downloaded - lastReported < 512 * 1024 && downloaded !== contentLength) return;
        lastReported = downloaded;
        const total = contentLength > 0 ? `/${Math.ceil(contentLength / 1024 / 1024)} MB` : '';
        process.stderr.write(`ainovel-cli-vn: downloading native binary ${Math.ceil(downloaded / 1024 / 1024)} MB${total}\n`);
      };
      const limit = new ByteLimitTransform(MAX_ARCHIVE_BYTES, reportProgress);
      const output = fs.createWriteStream(destination, { flags: 'wx', mode: 0o600 });
      return pipeline(response, limit, output);
    });
  } catch (error) {
    fs.rmSync(destination, { force: true });
    throw error;
  }
}

function collectLimited(stream, limit) {
  return new Promise((resolve, reject) => {
    const chunks = [];
    let total = 0;
    stream.on('data', (chunk) => {
      total += chunk.length;
      if (total > limit) {
        stream.destroy(new Error(`Downloaded text exceeds ${limit} bytes`));
        return;
      }
      chunks.push(chunk);
    });
    stream.on('end', () => resolve(Buffer.concat(chunks).toString('utf8')));
    stream.on('error', reject);
  });
}

function assertSafeArchiveEntry(rawEntry) {
  const entry = String(rawEntry).trim().replace(/\\/g, '/');
  if (!entry) return;
  if (entry.includes('\0') || entry.startsWith('/') || /^[A-Za-z]:\//.test(entry) || entry.split('/').includes('..')) {
    throw new Error(`Release archive contains an unsafe path: ${entry}`);
  }
  if (entry.includes('/')) {
    throw new Error(`Release archive contains an unexpected nested path: ${entry}`);
  }
}

function archiveBinaryName() {
  return process.platform === 'win32' ? `${BINARY_NAME}.exe` : BINARY_NAME;
}

function validateTarArchive(archivePath) {
  const listing = spawnSync('tar', ['-tzf', archivePath], { encoding: 'utf8' });
  if (listing.error || listing.status !== 0) {
    throw new Error(`Unable to inspect ${path.basename(archivePath)}`);
  }
  let binaryCount = 0;
  for (const rawEntry of listing.stdout.split(/\r?\n/)) {
    const entry = rawEntry.trim();
    if (!entry) continue;
    assertSafeArchiveEntry(entry);
    if (entry === archiveBinaryName()) binaryCount++;
  }
  if (binaryCount !== 1) {
    throw new Error(`Release archive must contain exactly one ${archiveBinaryName()}`);
  }

  const details = spawnSync('tar', ['-tvzf', archivePath], { encoding: 'utf8' });
  if (details.error || details.status !== 0) {
    throw new Error(`Unable to inspect archive entry types in ${path.basename(archivePath)}`);
  }
  for (const line of details.stdout.split(/\r?\n/)) {
    if (line && line[0] !== '-') {
      throw new Error('Release archive contains an unsupported entry type');
    }
  }
}

function validateWindowsZipEntries(archivePath) {
  const escapedArchive = escapePowerShell(path.resolve(archivePath));
  const command = [
    'Add-Type -AssemblyName System.IO.Compression.FileSystem',
    `$zip = [System.IO.Compression.ZipFile]::OpenRead('${escapedArchive}')`,
    'try {',
    '  $binaryCount = 0',
    '  foreach ($entry in $zip.Entries) {',
    "    $name = $entry.FullName -replace '\\\\', '/'",
    "    $mode = ($entry.ExternalAttributes -shr 16) -band 0xF000",
    "    if ($name.Contains('/') -or $name.StartsWith('/') -or $name -match '^[A-Za-z]:/' -or $name.Split('/') -contains '..' -or $mode -eq 0xA000) { exit 2 }",
    `    if ($name -eq '${archiveBinaryName()}') { $binaryCount++ }`,
    '  }',
    '  if ($binaryCount -ne 1) { exit 3 }',
    '} finally { $zip.Dispose() }',
  ].join('; ');
  const powershell = spawnSync('powershell.exe', ['-NoProfile', '-NonInteractive', '-Command', command], { stdio: 'ignore' });
  if (powershell.status !== 0) {
    throw new Error(`Unable to validate ${path.basename(archivePath)} on Windows`);
  }
}

function validateArchive(archivePath, target) {
  if (target.extension === '.zip' && process.platform === 'win32') {
    validateWindowsZipEntries(archivePath);
    return;
  }
  validateTarArchive(archivePath);
}

function extractArchive(archivePath, destination, target) {
  validateArchive(archivePath, target);
  fs.mkdirSync(destination, { recursive: true });
  const tarArgs = target.extension === '.tar.gz'
    ? ['-xzf', archivePath, '-C', destination]
    : ['-xf', archivePath, '-C', destination];
  const tarResult = spawnSync('tar', tarArgs, { stdio: 'ignore' });
  if (tarResult.status === 0) return;
  if (target.extension !== '.zip' || process.platform !== 'win32') {
    throw new Error(`Unable to extract ${path.basename(archivePath)} with tar`);
  }
  const escapedArchive = escapePowerShell(path.resolve(archivePath));
  const escapedDestination = escapePowerShell(path.resolve(destination));
  const command = `Expand-Archive -LiteralPath '${escapedArchive}' -DestinationPath '${escapedDestination}' -Force`;
  const powershell = spawnSync('powershell.exe', ['-NoProfile', '-NonInteractive', '-Command', command], { stdio: 'ignore' });
  if (powershell.status !== 0) {
    throw new Error(`Unable to extract ${path.basename(archivePath)} on Windows`);
  }
}

function escapePowerShell(value) {
  return value.replace(/'/g, "''");
}

function locateExtractedBinary(destination) {
  const binary = archiveBinaryName();
  const candidate = path.join(destination, binary);
  if (!isSafeChild(destination, candidate) || !isRegularFile(candidate)) {
    throw new Error(`Release archive does not contain a regular ${binary}`);
  }
  if (fs.statSync(candidate).size === 0) {
    throw new Error(`Release archive contains an empty ${binary}`);
  }
  return candidate;
}

function isSafeChild(parent, child) {
  const relative = path.relative(path.resolve(parent), path.resolve(child));
  return relative && !relative.startsWith('..') && !path.isAbsolute(relative);
}

function writeCachedDigest(filePath, digest) {
  const manifest = `${filePath}.sha256`;
  const temporary = `${manifest}.tmp-${process.pid}`;
  fs.writeFileSync(temporary, `${digest}\n`, { mode: 0o600, flag: 'wx' });
  try {
    fs.renameSync(temporary, manifest);
  } catch (error) {
    fs.rmSync(temporary, { force: true });
    throw error;
  }
}

async function installBinary(version, target) {
  const destination = cachedBinaryPath(version, target);
  if (await cachedBinaryIsValid(destination)) return destination;

  const releaseLock = await acquireLock(lockPath(version, target));
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'ainovel-cli-vn-'));
  try {
    if (await cachedBinaryIsValid(destination)) return destination;

    const asset = assetName(version, target);
    const archive = path.join(temporary, asset);
    const checksumsURL = releaseURL(version, `${BINARY_NAME}_checksums.txt`);
    process.stderr.write(`ainovel-cli-vn: downloading ${asset}...\n`);
    await Promise.all([
      downloadFile(releaseURL(version, asset), archive),
      downloadText(checksumsURL),
    ]).then(async ([, checksumText]) => {
      await verifyChecksum(archive, checksumFor(checksumText, asset), asset);
      const extracted = path.join(temporary, 'extracted');
      extractArchive(archive, extracted, target);
      const binary = locateExtractedBinary(extracted);
      fs.mkdirSync(path.dirname(destination), { recursive: true });
      const staged = `${destination}.tmp-${process.pid}`;
      fs.copyFileSync(binary, staged, fs.constants.COPYFILE_EXCL);
      try {
        fs.chmodSync(staged, 0o755);
        const digest = await sha256File(staged);
        fs.renameSync(staged, destination);
        writeCachedDigest(destination, digest);
      } catch (error) {
        fs.rmSync(staged, { force: true });
        throw error;
      }
    });
    return destination;
  } finally {
    fs.rmSync(temporary, { recursive: true, force: true });
    releaseLock();
  }
}

async function main() {
  if (process.argv[2] === 'update') {
    console.error(`ainovel-cli-vn: ${PACKAGE_UPDATE_MESSAGE}`);
    process.exitCode = 1;
    return;
  }
  const version = releaseVersion();
  const target = targetFor();
  const binary = await installBinary(version, target);
  const child = spawn(binary, process.argv.slice(2), { stdio: 'inherit', windowsHide: false });
  child.on('error', (error) => {
    console.error(`Failed to start ${BINARY_NAME}: ${error.message}`);
    process.exitCode = 1;
  });
  child.on('exit', (code, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
      return;
    }
    process.exitCode = code ?? 1;
  });
}

if (require.main === module) {
  main().catch((error) => {
    console.error(`ainovel-cli-vn: ${error.message}`);
    process.exitCode = 1;
  });
}

module.exports = {
  TARGETS,
  assetName,
  cachedBinaryPath,
  checksumFor,
  normalizeVersion,
  parseChecksums,
  releaseURL,
  targetFor,
  verifyChecksum,
};
