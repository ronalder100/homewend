# Security

Found a vulnerability? Report it privately,
[here](https://github.com/ronalder100/homewend/security/advisories/new), or
from the **Security** tab of this repository, **Report a vulnerability**.
Please do not open a public issue.

Only the latest release is supported.

## Checking a release

Every release carries, beside the binaries:

- `homewend_checksums.txt`, the SHA-256 of each binary. The installer and
  `homewend update` check the binary against it.
- `homewend_checksums.txt.sigstore.json`, the signature of the checksums,
  made by the release workflow of this repository with
  [Sigstore](https://www.sigstore.dev) and no key of ours to lose.
- `homewend_provenance.intoto.jsonl`, the
  [SLSA provenance](https://slsa.dev/spec/v1.0/provenance) of each binary:
  which workflow built it, from which commit.

A release is published only after the workflow has run both checks below on
it.

The signature, with
[cosign](https://docs.sigstore.dev/cosign/system_config/installation/):

```bash
cosign verify-blob \
  --certificate-identity-regexp '^https://github\.com/ronalder100/homewend/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --bundle homewend_checksums.txt.sigstore.json homewend_checksums.txt
```

The provenance of a binary, with the [GitHub CLI](https://cli.github.com):

```bash
gh attestation verify homewend_linux_amd64 --repo ronalder100/homewend
```
