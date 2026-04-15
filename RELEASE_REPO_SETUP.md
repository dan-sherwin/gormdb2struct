# YUM/DNF Repository Publication

This repository publishes RPM packages to a Yum/DNF repo hosted via GitHub Pages under:

  https://dan-sherwin.github.io/yum/rpm/$basearch/

Where `$basearch` is either `x86_64` or `aarch64`.

## How it works

- On every pushed release tag (for example `v1.0.0`), `.github/workflows/release.yml` runs GoReleaser and then calls `.github/workflows/publish_shared_yum_repo.yml` as a reusable workflow.
- The YUM publish workflow:
  1. Downloads the release RPM artifacts for linux amd64/arm64.
  2. Adds them into shared/yum/rpm/x86_64 and shared/yum/rpm/aarch64, creates/updates repodata with createrepo_c.
  3. Publishes the shared/yum directory to the user Pages repo (dan-sherwin/dan-sherwin.github.io) `main` branch.
  4. GitHub Pages must be enabled for dan-sherwin.github.io with Source: `main` branch / root.
- `.github/workflows/publish_shared_yum_repo.yml` can also be run manually with `workflow_dispatch` for a specific tag if the repo metadata ever needs to be republished.


## Client setup

Users can install via DNF by creating `/etc/yum.repos.d/dan-sherwin.repo`:

```
[dan-sherwin]
name=dan-sherwin packages
baseurl=https://dan-sherwin.github.io/yum/rpm/$basearch/
enabled=1
gpgcheck=0
```

Then run:

```
sudo dnf clean all
sudo dnf makecache
sudo dnf install gormdb2struct
```

Or add the hosted repo file directly:

```
sudo dnf config-manager --add-repo https://dan-sherwin.github.io/dan-sherwin.repo
sudo dnf install gormdb2struct
```

## Optional: GPG signing

If you want to sign RPMs and enable `gpgcheck=1`:
1. Generate a GPG key and store the private key in GitHub secrets.
2. Configure nfpm signing in `.goreleaser.yaml` (rpm.signature section) and pass secrets in the release workflow.
3. Publish the public key at `public.key` on the Pages site, and update the repo file to include:

```
gpgcheck=1
gpgkey=https://dan-sherwin.github.io/public.key
```
