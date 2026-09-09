import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';

const configPath = new URL('../upstream-monitor.json', import.meta.url);
const config = JSON.parse(readFileSync(configPath, 'utf8'));
const repositoryPattern = /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const branchPattern = /^[A-Za-z0-9._/-]+$/;
const shaPattern = /^[0-9a-f]{40}$/;
const dryRun = process.env.UPSTREAM_MONITOR_DRY_RUN === 'true';
const targetRepository = process.env.GITHUB_REPOSITORY;

function fail(message) {
  throw new Error(message);
}

function run(command, args) {
  try {
    return execFileSync(command, args, {
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'pipe'],
    }).trim();
  } catch (error) {
    const details = error.stderr?.toString().trim();
    throw new Error(`${command} failed${details ? `: ${details}` : ''}`);
  }
}

function runJson(command, args) {
  const output = run(command, args);
  try {
    return JSON.parse(output);
  } catch {
    fail(`Expected JSON from ${command}`);
  }
}

function validateConfig() {
  if (!repositoryPattern.test(config.repository)) {
    fail('Invalid upstream repository in monitor configuration');
  }
  if (!branchPattern.test(config.branch)) {
    fail('Invalid upstream branch in monitor configuration');
  }
  if (!shaPattern.test(config.baseline_sha)) {
    fail('Invalid upstream baseline SHA in monitor configuration');
  }
  if (typeof config.tracker_title !== 'string' || config.tracker_title.length === 0) {
    fail('Invalid tracker title in monitor configuration');
  }
  if (typeof config.tracker_marker !== 'string' || config.tracker_marker.length === 0) {
    fail('Invalid tracker marker in monitor configuration');
  }
  if (!repositoryPattern.test(targetRepository ?? '')) {
    fail('Invalid GITHUB_REPOSITORY');
  }
}

function getUpstreamSha() {
  const output = run('git', [
    'ls-remote',
    `https://github.com/${config.repository}.git`,
    `refs/heads/${config.branch}`,
  ]);
  const sha = output.split(/\s+/)[0];
  if (!shaPattern.test(sha)) {
    fail('Upstream branch did not return a valid commit SHA');
  }
  return sha;
}

function findTrackerIssue() {
  const issues = runJson('gh', [
    'issue', 'list',
    '--repo', targetRepository,
    '--state', 'all',
    '--limit', '1000',
    '--json', 'number,body,title,state,url',
  ]);
  const matches = issues.filter((issue) => (
    issue.title === config.tracker_title && issue.body?.includes(config.tracker_marker)
  ));
  if (matches.length > 1) {
    fail(`Found multiple upstream tracker issues: ${matches.map((issue) => `#${issue.number}`).join(', ')}`);
  }
  return matches[0] ?? null;
}

function getIssueBody(issueNumber) {
  return runJson('gh', [
    'issue', 'view', String(issueNumber),
    '--repo', targetRepository,
    '--json', 'body',
  ]).body ?? '';
}

function getIssueComments(issueNumber) {
  return runJson('gh', [
    'issue', 'view', String(issueNumber),
    '--repo', targetRepository,
    '--json', 'comments',
  ]).comments ?? [];
}

function readLastSeenSha(body) {
  const marker = body.match(/<!-- upstream-monitor-last-seen: ([0-9a-f]{40}) -->/);
  if (!marker) {
    return config.baseline_sha;
  }
  return marker[1];
}

function buildTrackerBody(lastSeenSha, upstreamSha) {
  const lastSeenLine = `<!-- upstream-monitor-last-seen: ${lastSeenSha} -->`;
  const repositoryUrl = `https://github.com/${config.repository}`;
  const commitUrl = `${repositoryUrl}/commit/${upstreamSha}`;
  return [
    config.tracker_marker,
    lastSeenLine,
    '',
    `This issue tracks commits from [${config.repository}](${repositoryUrl}) branch \`${config.branch}\`.`,
    '',
    `Last detected upstream commit: [\`${upstreamSha.slice(0, 7)}\`](${commitUrl}).`,
    '',
    'The monitor checks daily and does not merge upstream changes automatically.',
  ].join('\n');
}

function getComparison(previousSha, upstreamSha) {
  const comparison = runJson('gh', [
    'api',
    `repos/${config.repository}/compare/${previousSha}...${upstreamSha}`,
  ]);
  if (comparison.status !== 'ahead' || !Array.isArray(comparison.commits) || comparison.commits.length === 0) {
    fail(`Cannot safely compare upstream ${previousSha} to ${upstreamSha}: status=${comparison.status}`);
  }
  return comparison;
}

function subjectOf(commit) {
  const message = commit.commit?.message ?? '(no subject)';
  return message.split('\n', 1)[0].slice(0, 500);
}

function buildNotification(previousSha, upstreamSha, comparison) {
  const repositoryUrl = `https://github.com/${config.repository}`;
  const commits = comparison.commits.slice(0, 100).map((commit) => {
    const subject = subjectOf(commit);
    return `- [\`${commit.sha.slice(0, 7)}\`](${commit.html_url}) ${subject}`;
  });
  if (comparison.commits.length > commits.length) {
    commits.push(`- … ${comparison.commits.length - commits.length} more commit(s) in the comparison`);
  }
  return [
    `<!-- upstream-monitor-notified: ${upstreamSha} -->`,
    '',
    '## Upstream update detected',
    '',
    `- Repository: [${config.repository}](${repositoryUrl})`,
    `- Branch: \`${config.branch}\``,
    `- New HEAD: [\`${upstreamSha.slice(0, 7)}\`](${repositoryUrl}/commit/${upstreamSha})`,
    `- Compared from: \`${previousSha}\``,
    `- Comparison: [${comparison.html_url}](${comparison.html_url})`,
    `- Commits ahead: \`${comparison.ahead_by}\``,
    '',
    '### Commits',
    '',
    ...commits,
  ].join('\n');
}

function createIssue(body) {
  return runJson('gh', [
    'api', '--method', 'POST',
    `repos/${targetRepository}/issues`,
    '-f', `title=${config.tracker_title}`,
    '-f', `body=${body}`,
  ]);
}

function createComment(issueNumber, body) {
  run('gh', [
    'api', '--method', 'POST',
    `repos/${targetRepository}/issues/${issueNumber}/comments`,
    '-f', `body=${body}`,
  ]);
}

function updateIssue(issueNumber, body) {
  run('gh', [
    'api', '--method', 'PATCH',
    `repos/${targetRepository}/issues/${issueNumber}`,
    '-f', `body=${body}`,
    '-f', 'state=open',
  ]);
}

function main() {
  validateConfig();
  const upstreamSha = getUpstreamSha();
  console.log(`Upstream ${config.repository}:${config.branch} is at ${upstreamSha}`);

  const tracker = findTrackerIssue();
  if (!tracker && upstreamSha === config.baseline_sha) {
    console.log(`No upstream changes since baseline ${config.baseline_sha}`);
    return;
  }

  let issueNumber = tracker?.number;
  let issueBody = tracker ? getIssueBody(issueNumber) : '';
  const previousSha = readLastSeenSha(issueBody);
  if (!shaPattern.test(previousSha)) {
    fail('Tracker issue contains an invalid last-seen SHA');
  }
  if (upstreamSha === previousSha) {
    console.log(`No upstream changes since ${previousSha}`);
    return;
  }

  const comparison = getComparison(previousSha, upstreamSha);
  const notificationBody = buildNotification(previousSha, upstreamSha, comparison);
  const trackerBody = buildTrackerBody(upstreamSha, upstreamSha);
  if (dryRun) {
    console.log(`Dry run: ${comparison.commits.length} upstream commit(s) from ${previousSha} to ${upstreamSha}`);
    return;
  }

  if (!issueNumber) {
    const created = createIssue(buildTrackerBody(previousSha, previousSha));
    issueNumber = created.number;
    console.log(`Created upstream tracker issue #${issueNumber}`);
  }

  const comments = getIssueComments(issueNumber);
  const notificationMarker = `<!-- upstream-monitor-notified: ${upstreamSha} -->`;
  if (!comments.some((comment) => comment.body?.includes(notificationMarker))) {
    createComment(issueNumber, notificationBody);
    console.log(`Posted notification for ${upstreamSha}`);
  } else {
    console.log(`Notification for ${upstreamSha} already exists`);
  }

  updateIssue(issueNumber, trackerBody);
  console.log(`Updated tracker issue #${issueNumber}`);
}

main();
