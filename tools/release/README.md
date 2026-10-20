# Making a release

```sh
# 1. Add a «## <version>» section at the top of CHANGELOG.md: it becomes the release notes.
# 2. Commit, tag and push:
git tag v0.2.0
git push origin main v0.2.0
```

The tag starts `.github/workflows/release.yml`. It runs the tests, then builds:

| File | For |
|---|---|
| `dad_<os>_<arch>.tar.gz` / `.zip` | macOS, Linux and Windows on Intel and ARM; used by `install.sh`, `install.ps1`, Homebrew and winget |
| `dad_<version>_amd64.deb`, `_arm64.deb` | Debian and Ubuntu (`packages.sh`) |
| `dad-<version>-1.x86_64.rpm`, `.aarch64.rpm` | Fedora and RHEL (`packages.sh`) |
| `dad-lang.vsix` | the VS Code extension |
| `dad.rb` | the Homebrew formula (`manifests.py`) |
| `winget-manifests.zip` | the winget manifests (`manifests.py`) |
| `checksums.txt` | SHA-256 of every file above it |

It publishes the GitHub release. With the right repository secrets it also publishes
to the stores, and skips a store whose secret is missing.

## One-time setup

### Homebrew

1. Create an empty public repository **`mtwalli/homebrew-dad`**. Homebrew finds a tap by
   that name, so `brew install mtwalli/dad/dad` works.
2. Create a fine-grained personal access token with *Contents: read and write* on that
   repository only.
3. Add it to `mtwalli/dad-lang` as the secret **`HOMEBREW_TAP_TOKEN`**.

Each release then commits `Formula/dad.rb` to the tap.

### winget

winget's repository (`microsoft/winget-pkgs`) takes new packages by pull request, and the
first version must be submitted by hand:

1. After the release is published, download `winget-manifests.zip` from it.
2. Fork `microsoft/winget-pkgs`. Put the three files in
   `manifests/m/mtwalli/dad/<version>/` and open a pull request. Alternatively, on Windows,
   run `wingetcreate submit <folder>`.
3. Once it is merged, create a classic personal access token with the `public_repo` scope.
   Add it as the secret **`WINGET_TOKEN`**. Later releases then open the pull request
   themselves, through the `winget-releaser` action.

### VS Code Marketplace and Open VSX

These use the secrets **`VSCE_PAT`** and **`OVSX_PAT`**. Marketplace tokens made with Azure
DevOps personal access tokens stop working on 1 December 2026: move publishing to Microsoft
Entra ID before then.

## Trying the packages locally

```sh
# build the Linux binaries the way the workflow does, then:
tools/release/packages.sh 0.2.0                  # .deb (and .rpm when rpmbuild is installed)
(cd dist && sha256sum * > checksums.txt)
python3 tools/release/manifests.py 0.2.0         # dist/dad.rb and dist/winget/
```
