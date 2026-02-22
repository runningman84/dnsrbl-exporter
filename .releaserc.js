module.exports = {
  branches: ['main'],
  plugins: [
    '@semantic-release/commit-analyzer',
    '@semantic-release/release-notes-generator',
    '@semantic-release/changelog',
    [
      '@semantic-release/exec',
      {
        prepareCmd: [
          '[ -f helm/Chart.yaml ] && yq e -i \'.version = "${nextRelease.version}"\' helm/Chart.yaml || true',
          '[ -f helm/Chart.yaml ] && yq e -i \'.appVersion = "${nextRelease.version}"\' helm/Chart.yaml || true',
          '[ -f helm/values.yaml ] && yq e -i \'.image.tag = "${nextRelease.version}"\' helm/values.yaml || true',
          '[ -f flux/chart.yaml ] && yq e -i \'.spec.ref.tag = "${nextRelease.version}"\' flux/chart.yaml || true',
          'sed -i "s/--version [0-9]\\+\\.[0-9]\\+\\.[0-9]\\+/--version ${nextRelease.version}/g" README.md',
          'sed -i "s/gh release download v[0-9]\\+\\.[0-9]\\+\\.[0-9]\\+/gh release download v${nextRelease.version}/g" README.md',
          'sed -i "s/dnsrbl-exporter:[0-9]\\+\\.[0-9]\\+\\.[0-9]\\+/dnsrbl-exporter:${nextRelease.version}/g" README.md',
          'MAJOR=$(echo ${nextRelease.version} | cut -d. -f1) && MINOR=$(echo ${nextRelease.version} | cut -d. -f1-2) && sed -i -E "s/Tags: \`latest\`, \`[0-9]+\\.[0-9]+\\.[0-9]+\`, \`[0-9]+\\.[0-9]+\`, \`[0-9]+\`/Tags: \`latest\`, \`${nextRelease.version}\`, \`$MINOR\`, \`$MAJOR\`/g" README.md'
        ].join(' && '),
      },
    ],
    [
      '@semantic-release/git',
      {
        assets: ['helm/Chart.yaml', 'helm/values.yaml', 'CHANGELOG.md', 'README.md', 'flux/chart.yaml'],
        message: 'chore(release): ${nextRelease.version} [skip ci]\n\n${nextRelease.notes}',
      },
    ],
    '@semantic-release/github',
  ],
};
