# mv2s3 - Move Images to S3

A command-line tool that scans web source code files for local image references, uploads those images to AWS S3 (or compatible storage), and updates the source code with new URLs.

## Features

- 🔍 **Smart Scanning**: Recursively scans directories for image references in multiple file types
- 📁 **Multi-Format Support**: Handles HTML, CSS, JavaScript, Markdown, and more
- ☁️ **S3 Integration**: Seamless upload to AWS S3 or S3-compatible services
- 🔄 **URL Replacement**: Automatically updates source code with new S3 URLs
- 🛡️ **Safe Operations**: Backup originals and dry-run mode for safety
- ⚡ **Concurrent Processing**: Parallel uploads for improved performance
- 🎯 **Flexible Configuration**: Multiple configuration options via files, environment variables, or CLI flags

## Installation

### From Source
```bash
git clone https://github.com/SynapsesTechnologies/mv2s3.git
cd mv2s3
go build -o bin/mv2s3 ./cmd/mv2s3
```

### Using Go Install
```bash
go install github.com/SynapsesTechnologies/mv2s3/cmd/mv2s3@latest
```

## Quick Start

1. **Initialize configuration:**
   ```bash
   mv2s3 config init
   ```

2. **Scan your project for images:**
   ```bash
   mv2s3 scan --source ./website
   ```

3. **Migrate images to S3:**
   ```bash
   mv2s3 migrate --source ./website --bucket my-images-bucket
   ```

## Commands

### `migrate`
Perform the complete migration process: scan, upload, and update references.

```bash
mv2s3 migrate --source ./website --bucket my-images-bucket [options]
```

**Options:**
- `--source, -s`: Source directory to scan (required)
- `--bucket, -b`: S3 bucket name (required)
- `--prefix`: S3 key prefix for uploaded images
- `--region`: AWS region (default: us-east-1)
- `--dry-run`: Preview changes without making them
- `--backup`: Create backups of original files
- `--cleanup`: Remove local images after successful upload
- `--concurrency`: Number of concurrent uploads (default: 5)

### `scan`
Scan directories for image references without uploading.

```bash
mv2s3 scan --source ./website [options]
```

**Options:**
- `--source, -s`: Source directory to scan (required)
- `--extensions`: File extensions to scan (default: html,css,js,md,jsx,tsx)
- `--output, -o`: Save scan results to file
- `--format`: Output format (text, json, csv)

### `verify`
Verify S3 bucket access and upload permissions.

```bash
mv2s3 verify --bucket my-images-bucket [options]
```

### `cleanup`
Clean up local image files (use after successful migration).

```bash
mv2s3 cleanup --source ./website [options]
```

### `config`
Manage configuration files.

```bash
# Initialize new config
mv2s3 config init

# Show current configuration
mv2s3 config show

# Set configuration values
mv2s3 config set s3_bucket my-bucket
```

## Configuration

### Configuration File
Create a `.mv2s3.yaml` file in your project root or home directory:

```yaml
# Source scanning configuration
source_dir: "./website"
file_extensions: ["html", "css", "js", "md", "jsx", "tsx"]
exclude_patterns: 
  - "node_modules/**"
  - ".git/**"
  - "dist/**"

# S3 configuration
s3_bucket: "my-images-bucket"
s3_region: "us-east-1"
s3_prefix: "images/"

# Processing options
dry_run: false
backup_originals: true
cleanup_local: false
concurrency: 5

# Logging
log_level: "info"
verbose: false
```

### Environment Variables
All configuration options can be set via environment variables with the `MV2S3_` prefix:

```bash
export MV2S3_S3_BUCKET=my-images-bucket
export MV2S3_S3_REGION=us-west-2
export MV2S3_SOURCE_DIR=./website
```

### AWS Credentials
Configure AWS credentials using any of these methods:

1. **AWS Credentials File** (`~/.aws/credentials`)
2. **Environment Variables** (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
3. **IAM Roles** (when running on EC2)
4. **AWS Profiles** (`--profile` flag)

## Supported File Types and Patterns

### HTML Files
- `<img src="local/image.jpg">`
- `<link href="assets/icon.png">`
- `<source srcset="images/hero.jpg">`

### CSS Files
- `background-image: url(../images/bg.jpg)`
- `@import url("../fonts/icon.woff")`
- `content: url(./arrow.svg)`

### JavaScript/TypeScript
- `import logo from './assets/logo.png'`
- `const image = require('../images/photo.jpg')`
- String literals: `"./images/banner.jpg"`

### Markdown Files
- `![Alt text](./images/screenshot.png)`
- `<img src="assets/diagram.svg" alt="Diagram">`

## Project Structure

```
mv2s3/
├── cmd/mv2s3/           # CLI entry point
├── internal/
│   ├── cli/             # Command-line interface
│   ├── config/          # Configuration management
│   ├── processor/       # Image processing and URL replacement
│   ├── scanner/         # File scanning and link parsing
│   └── storage/         # S3 client and upload logic
├── pkg/types/           # Shared data types
├── test/                # Tests and fixtures
└── scripts/             # Build and test scripts
```

## Examples

### Basic Migration
```bash
# Migrate all images from a website to S3
mv2s3 migrate --source ./my-website --bucket my-images --prefix assets/
```

### Dry Run Mode
```bash
# Preview what would be changed without making modifications
mv2s3 migrate --source ./docs --bucket my-bucket --dry-run
```

### With Backup and Cleanup
```bash
# Create backups and remove local files after upload
mv2s3 migrate --source ./app --bucket images-bucket --backup --cleanup
```

### Scan Only
```bash
# Just scan and generate a report
mv2s3 scan --source ./website --output scan-results.json --format json
```

### Framework-Specific Examples

#### Hugo Static Site
```bash
mv2s3 migrate --source ./hugo-site/static --bucket my-blog-images --prefix images/
```

#### Next.js Project
```bash
mv2s3 migrate --source ./nextjs-app/public --bucket my-app-assets --prefix assets/
```

#### React Application
```bash
mv2s3 migrate --source ./react-app/src --bucket my-react-images --prefix images/
```

## Error Handling

The tool provides comprehensive error handling and recovery:

- **File Backup**: Original files are backed up before modification
- **Atomic Operations**: File updates are performed atomically
- **Retry Logic**: Network operations include automatic retry with exponential backoff
- **Validation**: Input validation and S3 access verification before processing
- **Rollback**: Failed operations can be rolled back using backup files

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Development

### Building
```bash
make build
```

### Testing
```bash
make test
```

### Running Tests with Coverage
```bash
make test-coverage
```

### Cross-Platform Builds
```bash
make build-all
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- 📖 [Documentation](https://github.com/SynapsesTechnologies/mv2s3/wiki)
- 🐛 [Issue Tracker](https://github.com/SynapsesTechnologies/mv2s3/issues)
- 💬 [Discussions](https://github.com/SynapsesTechnologies/mv2s3/discussions)

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- Uses [Viper](https://github.com/spf13/viper) for configuration management
- AWS SDK for Go v2 for S3 integration