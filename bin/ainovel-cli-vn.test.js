'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const { spawnSync } = require('node:child_process');

const launcher = require('./ainovel-cli-vn.js');

test('maps supported targets to GoReleaser names', () => {
  assert.deepEqual(launcher.targetFor('darwin', 'arm64'), {
    os: 'Darwin', arch: 'arm64', extension: '.tar.gz',
  });
  assert.deepEqual(launcher.targetFor('linux', 'x64'), {
    os: 'Linux', arch: 'x86_64', extension: '.tar.gz',
  });
  assert.deepEqual(launcher.targetFor('win32', 'x64'), {
    os: 'Windows', arch: 'x86_64', extension: '.zip',
  });
});

test('rejects unsupported targets and malformed versions', () => {
  assert.throws(() => launcher.targetFor('freebsd', 'x64'), /Unsupported platform/);
  assert.throws(() => launcher.normalizeVersion('latest'), /Invalid release version/);
  assert.equal(launcher.normalizeVersion('v1.2.3'), '1.2.3');
});

test('builds release asset URLs and names', () => {
  const target = launcher.targetFor('linux', 'arm64');
  const asset = launcher.assetName('1.2.3', target);
  assert.equal(asset, 'ainovel-cli_1.2.3_Linux_arm64.tar.gz');
  assert.equal(
    launcher.releaseURL('1.2.3', asset),
    'https://github.com/JustinNguyen9979/ainovel-cli/releases/download/v1.2.3/ainovel-cli_1.2.3_Linux_arm64.tar.gz',
  );
});

test('parses and selects a unique SHA-256 checksum', () => {
  const checksums = [
    'a'.repeat(64) + '  ainovel-cli_1.2.3_Linux_arm64.tar.gz',
    'b'.repeat(64) + ' *ainovel-cli_1.2.3_Windows_x86_64.zip',
  ].join('\n');
  assert.equal(launcher.checksumFor(checksums, 'ainovel-cli_1.2.3_Linux_arm64.tar.gz'), 'a'.repeat(64));
  assert.throws(() => launcher.checksumFor(checksums, 'missing.tar.gz'), /does not contain/);
  assert.throws(() => launcher.parseChecksums('c'.repeat(64) + ' file\n' + 'c'.repeat(64) + ' file'), /duplicate/);
});

test('verifies archive checksum before installation', async () => {
  const fixture = path.join(__dirname, 'checksum-fixture.bin');
  fs.writeFileSync(fixture, 'fixture');
  try {
    const digest = crypto.createHash('sha256').update('fixture').digest('hex');
    await assert.doesNotReject(() => launcher.verifyChecksum(fixture, digest, 'fixture'));
    await assert.rejects(() => launcher.verifyChecksum(fixture, '0'.repeat(64), 'fixture'), /verification failed/);
  } finally {
    fs.rmSync(fixture, { force: true });
  }
});

test('rejects npm-managed self-update without downloading a binary', () => {
  const result = spawnSync(process.execPath, [path.join(__dirname, 'ainovel-cli-vn.js'), 'update'], {
    encoding: 'utf8',
    env: { ...process.env, AINOVEL_CACHE_DIR: path.join(__dirname, 'missing-cache') },
  });
  assert.equal(result.status, 1);
  assert.match(result.stderr, /managed by npm/);
});

test('package allowlist excludes local build and config files', () => {
  const packageJSON = JSON.parse(fs.readFileSync(path.join(__dirname, '..', 'package.json'), 'utf8'));
  assert.deepEqual(packageJSON.files, ['bin/ainovel-cli-vn.js', 'LICENSE', 'README.md']);
  assert.equal(packageJSON.name, 'ainovel-cli-vn');
  assert.equal(packageJSON.bin['ainovel-cli-vn'], 'bin/ainovel-cli-vn.js');
  assert.deepEqual(packageJSON.os, ['darwin', 'linux', 'win32']);
  assert.deepEqual(packageJSON.cpu, ['x64', 'arm64']);
});
