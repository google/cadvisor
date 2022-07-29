# cAdvisor Release Instructions

## 1. Send Release PR

Example: https://github.com/google/cadvisor/pull/1281

Add release notes to [CHANGELOG.md](../../CHANGELOG.md)

- Tip: Use a github PR search to find changes since the last release
  `is:pr is:merged merged:>2016-04-21`

## 2. Create the release tag

### 2.a Create the release branch (only for major/minor releases)

Skip this step for patch releases.

```
# Example version
VERSION=v0.23
PATCH_VERSION=$VERSION.0
# Sync to HEAD, or the commit to branch at
git fetch upstream && git checkout upstream/master
# Create the branch
git branch release-$VERSION
# Push it to upstream
git push git@github.com:google/cadvisor.git release-$VERSION
```

### 2.b Tag the release (for all releases)

```
# Example patch version
VERSION=v0.23
PATCH_VERSION=$VERSION.0
# Checkout the release branch
git fetch upstream && git checkout upstream/release-$VERSION
# Tag the release commit. If you aren't signing, ommit the -s
git tag -s -a $PATCH_VERSION
# Push it to upstream
git push git@github.com:google/cadvisor.git $PATCH_VERSION
```

## 3. Build release artifacts

Command: `make release`

- Make sure your git client is synced to the release cut point
- Use the same go version as kubernetes: [dependencies.yaml](https://github.com/kubernetes/kubernetes/blob/master/build/dependencies.yaml#L101)
- Tip: use https://github.com/moovweb/gvm to manage multiple go versions.
- Try to build it from the release branch, since we include that in the binary version
- Verify the ldflags output, in particular check the Version, BuildUser, and GoVersion are expected

Once the build is complete, copy the output after `Release info...` and save it to use in step 5

Example:

```
Multi Arch Container:
gcr.io/cadvisor/cadvisor:v0.44.1-test-8

Architecture Specific Containers:
gcr.io/cadvisor/cadvisor-arm:v0.44.1-test-8
gcr.io/cadvisor/cadvisor-arm64:v0.44.1-test-8
gcr.io/cadvisor/cadvisor-amd64:v0.44.1-test-8

Binaries:
SHA256 (cadvisor-v0.44.1-test-8-linux-arm64) = e5e3f9e72208bc6a5ef8b837473f6c12877ace946e6f180bce8d81edadf66767
SHA256 (cadvisor-v0.44.1-test-8-linux-arm) = 7d714e495a4f50d9cc374bd5e6b5c6922ffa40ff1cc7244f2308f7d351c4ccea
SHA256 (cadvisor-v0.44.1-test-8-linux-amd64) = ea95c5a6db8eecb47379715c0ca260a8a8d1522971fd3736f80006c7f6cc9466
```

## 4. Check that the Containers for the release work

The only argument to the script is the tag of the Multi Arch Container from step
3. To verify that the container images for the release were built successfully,
use the check_container.sh script. The script will start each cadvisor image and
curl the `/healthz` endpoint to confirm that it is working.

Running this script requires that you have installed `qemu-user-static` and
configured qemu as a binary interpreter.

```
$ sudo apt install qemu-user-static
$ docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
```

The only argument to the script is the tag of the Multi Arch Container from step
3.

```sh
build/check_container.sh gcr.io/tstapler-gke-dev/cadvisor:v0.44.1-test-8
```

## 5. Cut the release

Go to https://github.com/google/cadvisor/releases and click "Draft a new
release"

- "Tag version" and "Release title" should be preceded by 'v' and then the
  version. Select the tag pushed in step 2.b
- Copy an old release as a template (e.g.
  github.com/google/cadvisor/releases/tag/v0.23.1)
- Body should start with release notes (from CHANGELOG.md)
- Next are the docker images and binary hashes you copied (from step 3).
- Upload the binaries build in step 3, they are located in the `_output`
  directory.
- If this is a minor version release, mark the release as a "pre-release"
- Click publish when done