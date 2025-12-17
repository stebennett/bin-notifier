# Bin-notifier

Application that reads data about bin deliveries and sends an SMS with the details.

## CI/CD

Pull requests are automatically built and tested via GitHub Actions.

To create a release, push a semantic version tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This will build binaries for Linux (amd64, arm64) and macOS (arm64), then create a GitHub release with downloadable zip files.
