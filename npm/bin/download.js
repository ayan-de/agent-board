#!/usr/bin/env node
const fs = require('fs');
const path = require('path');
const https = require('https');
const http = require('http');

const REPO = 'ayan-de/agent-board';
const BASE_URL = `https://github.com/${REPO}/releases/latest/download`;
const INSTALL_DIR = path.join(process.env.HOME, '.local', 'bin');

function detectOS() {
  const platform = process.platform;
  if (platform === 'linux') {
    try {
      const kernel = fs.readFileSync('/proc/version', 'utf8');
      if (kernel.includes('Android')) return 'android';
    } catch {}
    return 'linux';
  }
  if (platform === 'darwin') return 'darwin';
  return null;
}

function detectArch() {
  const arch = process.arch;
  if (arch === 'x64') return 'x86_64';
  if (arch === 'arm64' || arch === 'aarch64') return 'arm64';
  return null;
}

const os = detectOS();
const arch = detectArch();

if (!os || !arch) {
  console.error('Unsupported platform');
  process.exit(1);
}

const binaryMap = {
  'linux-x86_64': 'agentboard-linux-x86_64',
  'linux-arm64': 'agentboard-linux-arm64',
  'darwin-x86_64': 'agentboard-darwin-x86_64',
  'darwin-arm64': 'agentboard-darwin-arm64',
  'android-arm64': 'agentboard-android-arm64',
};

const binaryName = binaryMap[`${os}-${arch}`];
if (!binaryName) {
  console.error(`Unsupported platform: ${os}-${arch}`);
  process.exit(1);
}

const installPath = path.join(INSTALL_DIR, 'agentboard');

if (fs.existsSync(installPath)) {
  console.log('AgentBoard binary already installed');
  process.exit(0);
}

console.log(`Downloading AgentBoard for ${os}/${arch}...`);

const url = `${BASE_URL}/${binaryName}`;

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    const protocol = url.startsWith('https') ? https : http;

    protocol.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        file.close();
        fs.unlinkSync(dest);
        download(response.headers.location, dest).then(resolve).catch(reject);
        return;
      }

      if (response.statusCode !== 200) {
        reject(new Error(`Download failed: ${response.statusCode}`));
        return;
      }

      response.pipe(file);
      file.on('finish', () => {
        file.close();
        fs.chmodSync(dest, 0o755);
        resolve();
      });
    }).on('error', (err) => {
      fs.unlinkSync(dest);
      reject(err);
    });
  });
}

async function install() {
  if (!fs.existsSync(INSTALL_DIR)) {
    fs.mkdirSync(INSTALL_DIR, { recursive: true });
  }

  const tempFile = path.join(INSTALL_DIR, `.${binaryName}.tmp`);

  try {
    await download(url, tempFile);
    fs.renameSync(tempFile, installPath);
    console.log('AgentBoard installed successfully');
  } catch (err) {
    if (fs.existsSync(tempFile)) fs.unlinkSync(tempFile);
    console.error('Download failed:', err.message);
    process.exit(1);
  }
}

install();