
  Low-level (exported primitives):
  func ExtractTarGz(src, destDir string, opts ...ExtractOpt) error
  func ExtractZip(src, destDir string, opts ...ExtractOpt) error
  func ExtractTar(src, destDir string, opts ...ExtractOpt) error
  func CopyFile(src, dst string) error
  func CreateSymlink(binaryPath string) (string, error)  // already exists

  Mid-level (generic download):
  func Download(ctx context.Context, tc *TaskContext, url string, opts ...DownloadOpt) error
  func FromLocal(ctx context.Context, tc *TaskContext, path string, opts ...DownloadOpt) error

  // Options
  WithDestDir(path string) DownloadOpt
  WithFormat(format string) DownloadOpt  // "tar.gz", "zip", "tar", ""
  WithExtractFiles(names ...string) DownloadOpt
  WithRenameFile(src, dst string) DownloadOpt
  WithSymlink() DownloadOpt
  WithSkipIfExists(path string) DownloadOpt
  WithHTTPHeader(key, value string) DownloadOpt

  High-level (GitHub-specific):
  func DownloadGitHubRelease(ctx context.Context, tc *TaskContext, repo, version string, opts ...GitHubOpt) error

  // Options
  WithAssetPattern(pattern string) GitHubOpt  // supports {version}, {os}, {arch}
  WithToken(token string) GitHubOpt           // for private repos
