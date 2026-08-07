import process from 'node:process'

const severities = new Map([
  ['info', 0],
  ['low', 1],
  ['moderate', 2],
  ['high', 3],
  ['critical', 4],
])

const thresholdName = process.argv[2] ?? 'high'
const threshold = severities.get(thresholdName)
if (threshold === undefined) {
  console.error(`unsupported npm audit threshold: ${thresholdName}`)
  process.exit(2)
}

let input = ''
for await (const chunk of process.stdin) {
  input += chunk
}

let report
try {
  report = JSON.parse(input)
} catch {
  annotation('error', 'npm audit', 'npm audit returned invalid JSON')
  process.exit(1)
}

if (report.error) {
  annotation(
    'error',
    'npm audit',
    report.error.summary ||
      report.error.detail ||
      report.message ||
      'npm audit failed',
  )
  process.exit(1)
}

const findings = Object.values(report.vulnerabilities ?? {})
  .filter((finding) => (severities.get(finding.severity) ?? -1) >= threshold)
  .sort((left, right) => left.name.localeCompare(right.name))

for (const finding of findings) {
  const advisories = (finding.via ?? [])
    .filter((item) => typeof item === 'object')
    .map((item) => item.title)
    .filter(Boolean)
  const detail = [
    `${finding.name} (${finding.severity})`,
    advisories.join('; '),
    finding.range ? `affected ${finding.range}` : '',
  ]
    .filter(Boolean)
    .join(': ')
  annotation('error', `npm audit: ${finding.name}`, detail)
}

const counts = report.metadata?.vulnerabilities ?? {}
console.log(
  `npm audit: ${counts.total ?? 0} total, ${counts.high ?? 0} high, ${counts.critical ?? 0} critical`,
)
process.exit(findings.length === 0 ? 0 : 1)

function annotation(level, title, message) {
  console.log(`::${level} title=${escape(title)}::${escape(message)}`)
}

function escape(value) {
  return String(value)
    .replaceAll('%', '%25')
    .replaceAll('\r', '%0D')
    .replaceAll('\n', '%0A')
}
