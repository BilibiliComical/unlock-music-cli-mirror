// generate .drone.yaml, run:
// drone jsonnet --format --stream


local CreateRelease() = {
  name: 'create release',
  image: 'plugins/gitea-release',
  settings: {
    api_key: { from_secret: 'GITEA_API_KEY' },
    base_url: 'https://git.unlock-music.dev',
    files: [
      'um-*.tar.gz',
      'um-*.zip',
    ],
    checksum: 'sha256',
    draft: true,
    title: '${DRONE_TAG}',
  },
};


local StepGoBuild(GOOS, GOARCH) = {
  local windows = GOOS == 'windows',
  local archiveExt = if windows then 'zip' else 'tar.gz',
  local filepath = 'dist/um-%s-%s-%s.%s' % [GOOS, GOARCH, '$(git describe --tags --always)', archiveExt],

  local archive = if windows then [
    // Ensure zip is installed
    'command -v zip >/dev/null || (apt update && apt install -y zip)',
    'zip -9 -j -r "%s" $DIST_DIR' % filepath,
  ] else [
    'tar -c -C $DIST_DIR um | gzip -9 > "%s"' % filepath,
  ],

  name: 'go build %s/%s' % [GOOS, GOARCH],
  image: 'golang:1.22',
  environment: {
    GOOS: GOOS,
    GOARCH: GOARCH,
    GOPROXY: 'https://goproxy.io,direct',
  },
  commands: [
    'DIST_DIR=$(mktemp -d)',
    'go build -v -trimpath -ldflags="-w -s -X main.AppVersion=$(git describe --tags --always)" -o $DIST_DIR ./cmd/um',
    'mkdir -p dist',
  ] + archive,
};

local StepUploadArtifact(GOOS, GOARCH) = {
  local windows = GOOS == 'windows',
  local archiveExt = if windows then 'zip' else 'tar.gz',
  local filename = 'um-%s-%s-%s.%s' % [GOOS, GOARCH, '$(git describe --tags --always)', archiveExt],
  local filepath = 'dist/%s' % filename,
  local pkgname = '${DRONE_REPO_NAME}-build',

  name: 'upload artifact',
  image: 'golang:1.22',  // reuse golang:1.19 for curl
  environment: {
    DRONE_GITEA_SERVER: 'https://git.unlock-music.dev',
    GITEA_API_KEY: { from_secret: 'GITEA_API_KEY' },
  },
  commands: [
    'curl --fail --include --user "um-release-bot:$GITEA_API_KEY" ' +
    '--upload-file "%s" ' % filepath +
    '"$DRONE_GITEA_SERVER/api/packages/${DRONE_REPO_NAMESPACE}/generic/%s/${DRONE_BUILD_NUMBER}/%s"' % [pkgname, filename],
    'sha256sum %s' % filepath,
    'echo $DRONE_GITEA_SERVER/${DRONE_REPO_NAMESPACE}/-/packages/generic/%s/${DRONE_BUILD_NUMBER}' % pkgname,
  ],
};


local PipelineBuild(GOOS, GOARCH, RUN_TEST) = {
  name: 'build %s/%s' % [GOOS, GOARCH],
  kind: 'pipeline',
  type: 'docker',
  steps: [
           {
             name: 'fetch tags',
             image: 'alpine/git',
             commands: ['git fetch --tags'],
           },
         ] +
         (
           if RUN_TEST then [{
             name: 'go test',
             image: 'golang:1.22',
             environment: {
               GOPROXY: 'https://goproxy.io,direct',
             },
             commands: ['go test -v ./...'],
           }] else []
         )
         +
         [
           StepGoBuild(GOOS, GOARCH),
           StepUploadArtifact(GOOS, GOARCH),
         ],
  trigger: {
    event: ['push', 'pull_request'],
  },
};

local PipelineRelease() = {
  name: 'release',
  kind: 'pipeline',
  type: 'docker',
  steps: [
    {
      name: 'fetch tags',
      image: 'alpine/git',
      commands: ['git fetch --tags'],
    },
    {
      name: 'go test',
      image: 'golang:1.22',
      environment: {
        GOPROXY: 'https://goproxy.io,direct',
      },
      commands: ['go test -v ./...'],
    },
    StepGoBuild('linux', 'amd64'),
    StepGoBuild('linux', 'arm64'),
    StepGoBuild('linux', '386'),
    StepGoBuild('windows', 'amd64'),
    StepGoBuild('windows', 'arm64'),
    StepGoBuild('windows', '386'),
    StepGoBuild('darwin', 'amd64'),
    StepGoBuild('darwin', 'arm64'),
    {
      name: 'prepare root',
      image: 'golang:1.22',
      commands: [
        'mv dist/*.tar.gz dist/*.zip ./',
      ],
    },
    CreateRelease(),
  ],
  trigger: {
    event: ['tag'],
  },
};

[
  PipelineBuild('linux', 'amd64', true),
  PipelineBuild('windows', 'amd64', false),
  PipelineBuild('darwin', 'amd64', false),
  PipelineRelease(),
]
