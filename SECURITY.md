# Security

Found a vulnerability? Report it privately,
[here](https://github.com/ronalder100/homewend/security/advisories/new), or
from the **Security** tab of this repository, **Report a vulnerability**.
Please do not open a public issue.

Only the latest release is supported.

## Checking a release

Every release carries `homewend_checksums.txt`, the SHA-256 of each binary,
and `homewend_checksums.txt.sigstore.json`, its signature, made by the
release workflow of this repository with [Sigstore](https://www.sigstore.dev)
and no key of ours to lose. The installer and `homewend update` check the
binary against the checksums. To check the checksums themselves, with
[cosign](https://docs.sigstore.dev/cosign/system_config/installation/):

```bash
cosign verify-blob \
  --certificate-identity-regexp '^https://github\.com/ronalder100/homewend/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --bundle homewend_checksums.txt.sigstore.json homewend_checksums.txt
```
