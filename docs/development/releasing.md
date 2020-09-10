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

Once the build is complete, check the VERSION and note the sha256 hash.

## 4. Push the Docker images

`make release` should output a command to push the image.  Alternatively, run:

```
$ PATCH_VERSION=v0.23.0
$ docker push gcr.io/cadvisor/cadvisor:$PATCH_VERSION
```

## 5. Cut the release

Go to https://github.com/google/cadvisor/releases and click "Draft a new release"

- "Tag version" and "Release title" should be preceded by 'v' and then the version. Select the tag pushed in step 2.b
- Copy an old release as a template (e.g. github.com/google/cadvisor/releases/tag/v0.23.1)
- Body should start with release notes (from CHANGELOG.md)
- Next is the Docker image: `gcr.io/cadvisor/cadvisor:$VERSION`
- Next are the binary hashes (from step 3)
- Upload the binary build in step 3
- If this is a minor version release, mark the release as a "pre-release"
- Click publish when done

## 6. Finalize the release

~~Once you are satisfied with the release quality (generally we wait until the next minor release), it is time to remove the "pre-release" tag~~

cAdvisor is no longer pushed with the :latest tag.  This is to ensure tagged images are immutable.
