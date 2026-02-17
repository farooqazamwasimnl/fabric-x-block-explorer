# Contributing to Fabric-X Block Explorer

Thank you for your interest in contributing to **Fabric-X Block Explorer**! We welcome contributions from the community to help improve the project.

## Table of Contents

1. [Code of Conduct](#code-of-conduct)
2. [How to Contribute](#how-to-contribute)
3. [Development Setup](#development-setup)
4. [Coding Standards](#coding-standards)
5. [Testing Requirements](#testing-requirements)
6. [Commit Guidelines](#commit-guidelines)
7. [Pull Request Process](#pull-request-process)
8. [Review Process](#review-process)
9. [Community](#community)

---

## Code of Conduct

This project follows the [Hyperledger Code of Conduct](https://wiki.hyperledger.org/display/HYP/Hyperledger+Code+of+Conduct). Please review it before participating. We expect all contributors to:

- Be respectful and professional
- Welcome diverse perspectives
- Focus on constructive feedback
- Prioritize community interests over individual interests

---

## How to Contribute

### Reporting Issues

If you find a bug or have a feature request:

1. **Search existing issues** to avoid duplicates
2. **Create a new issue** with:
   - Clear, descriptive title
   - Detailed description of the problem or feature
   - Steps to reproduce (for bugs)
   - Expected vs actual behavior
   - Environment details (Go version, OS, etc.)

### Contributing Code

We welcome code contributions! Please follow these steps:

1. **Fork the repository**
2. **Create a feature branch** from `main`
3. **Make your changes** following our coding standards
4. **Add tests** for new functionality
5. **Update documentation** as needed
6. **Sign your commits** with DCO
7. **Submit a pull request**

---

## Development Setup

### Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- PostgreSQL 14+
- Make (optional, but recommended)

### Local Setup

1. **Clone your fork**:
   ```bash
   git clone https://github.com/<your-username>/fabric-x-block-explorer.git
   cd fabric-x-block-explorer
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Start PostgreSQL** (using Docker):
   ```bash
   docker run --name explorer-postgres \
     -e POSTGRES_PASSWORD=postgres \
     -e POSTGRES_DB=explorer \
     -p 5432:5432 \
     -d postgres:14
   ```

4. **Initialize database schema**:
   ```bash
   psql -h localhost -U postgres -d explorer -f pkg/db/schema.sql
   ```

5. **Copy configuration template**:
   ```bash
   cp config.yaml.example config.yaml
   ```

6. **Run the application**:
   ```bash
   go run ./cmd/explorer/main.go
   ```

### Using Docker Compose

For a complete development environment:

```bash
docker-compose up
```

This starts both PostgreSQL and the explorer service.

---

## Coding Standards

### Go Conventions

Follow standard Go conventions and best practices:

1. **Code Formatting**: Use `gofmt` or `goimports`
   ```bash
   gofmt -w .
   go fmt ./...
   ```

2. **Code Quality**: Run `go vet` to catch common errors
   ```bash
   go vet ./...
   ```

3. **Linting**: Use `golangci-lint` for comprehensive linting
   ```bash
   golangci-lint run
   ```

### Code Style Guidelines

- **Package Names**: Use lowercase, single-word names (e.g., `parser`, `db`)
- **Function Names**: Use camelCase, start with uppercase for exported functions
- **Variable Names**: Use descriptive names; avoid single-letter variables except in loops
- **Error Handling**: Always check and handle errors explicitly
- **Comments**: 
  - Document all exported functions, types, and constants
  - Use complete sentences with proper punctuation
  - Start with the name being documented
  - Example: `// ParseBlock extracts structured data from a Fabric block.`

### Project-Specific Standards

1. **Structured Logging**: Use `zerolog` for all logging
   ```go
   import "github.com/rs/zerolog/log"
   
   log.Info().
       Str("component", "parser").
       Int64("block_number", blockNum).
       Msg("Block parsed successfully")
   ```

2. **Constants**: Define constants in `pkg/constants/` package
   ```go
   const (
       DefaultChannelSize = 100
       DefaultWorkerCount = 4
   )
   ```

3. **Configuration**: Use the `pkg/config` package for all configuration
   - Never hardcode configuration values
   - Support both YAML and environment variables
   - Validate configuration on startup

4. **Error Messages**: Use descriptive, actionable error messages
   ```go
   return fmt.Errorf("failed to parse block %d: %w", blockNum, err)
   ```

---

## Testing Requirements

All contributions must include appropriate tests.

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./pkg/parser/...
```

### Test Coverage

- Aim for **70%+ code coverage** for new code
- Critical paths (parsing, database writes) should have **90%+ coverage**
- Use table-driven tests for multiple scenarios

### Test Structure

```go
func TestParseBlock(t *testing.T) {
    tests := []struct {
        name    string
        input   *common.Block
        want    *types.ProcessedBlock
        wantErr bool
    }{
        {
            name:    "valid block",
            input:   createTestBlock(),
            want:    expectedResult(),
            wantErr: false,
        },
        {
            name:    "invalid block",
            input:   nil,
            want:    nil,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseBlock(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseBlock() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ParseBlock() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Integration Tests

- Use **testcontainers** for database integration tests
- Clean up resources in `defer` statements
- Test complete workflows end-to-end

---

## Commit Guidelines

### Developer Certificate of Origin (DCO)

All commits **must be signed** with the Developer Certificate of Origin (DCO):

```bash
git commit -s -m "Your commit message"
```

The `-s` flag adds a `Signed-off-by` line to your commit message:

```
Signed-off-by: Your Name <your.email@example.com>
```

**Why?** The DCO certifies that you have the right to submit the code you are contributing.

### GPG Signing (Recommended)

For added security, sign commits with GPG:

```bash
git commit -S -s -m "Your commit message"
```

### Commit Message Format

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `refactor`: Code refactoring (no functional changes)
- `test`: Adding or updating tests
- `chore`: Maintenance tasks (dependencies, build, etc.)
- `perf`: Performance improvements

**Examples**:

```
feat(api): add endpoint for namespace policies

Add GET /policies/{namespace} endpoint to retrieve endorsement
policies for a specific chaincode namespace. Supports optional
'latest' query parameter to return only the most recent policy.

Signed-off-by: John Doe <john@example.com>
```

```
fix(parser): handle nil transaction validation codes

Add nil check for transaction validation codes to prevent panic
when processing malformed blocks from the sidecar.

Fixes #123

Signed-off-by: Jane Smith <jane@example.com>
```

---

## Pull Request Process

### Before Submitting

1. **Rebase on latest `main`**:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run tests**:
   ```bash
   go test ./...
   ```

3. **Run linters**:
   ```bash
   golangci-lint run
   ```

4. **Update documentation**:
   - Update `Docs/Readme.md` if adding features
   - Update `pkg/swagger/swagger.yaml` if changing APIs
   - Add/update inline code comments

5. **Verify commit signoff**:
   ```bash
   git log --show-signature -1
   ```

### Creating the Pull Request

1. **Push your branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Open a pull request** on GitHub with:
   - **Clear title**: Summarize the change in one line
   - **Description**: Explain what, why, and how
   - **Linked issues**: Reference related issues (e.g., "Fixes #123")
   - **Testing**: Describe how you tested the changes
   - **Screenshots**: Include for UI changes (e.g., Swagger)

### Pull Request Template

```markdown
## Description

Brief description of the changes.

## Motivation and Context

Why is this change needed? What problem does it solve?

## How Has This Been Tested?

Describe the testing you performed:
- Unit tests added/updated
- Integration tests added/updated
- Manual testing performed

## Types of Changes

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to change)
- [ ] Documentation update

## Checklist

- [ ] My code follows the code style of this project
- [ ] I have updated the documentation accordingly
- [ ] I have added tests to cover my changes
- [ ] All new and existing tests passed
- [ ] My commits are signed with DCO (`-s` flag)
```

---

## Review Process

### What to Expect

- **Initial review**: Within 3-5 business days
- **Feedback**: Reviewers may request changes or ask questions
- **Iteration**: Address feedback and update your PR
- **Approval**: At least one maintainer approval required
- **Merge**: Maintainers will merge after approval

### Review Labels

Reviewers use labels to categorize feedback:

- **Major**: Critical issues that must be fixed (functionality, security, correctness)
- **Minor**: Suggestions for improvement (style, readability, best practices)
- **Nit**: Minor style/formatting suggestions (optional to address)

### Addressing Feedback

1. **Respond to all comments**: Acknowledge feedback and explain your approach
2. **Make requested changes**: Update your code and push new commits
3. **Mark conversations resolved**: After addressing, mark GitHub conversations as resolved
4. **Request re-review**: Use GitHub's "Re-request review" feature

### Tips for Successful Reviews

- **Be responsive**: Reply to comments within 48 hours
- **Be open to feedback**: Reviews improve code quality
- **Ask questions**: If feedback is unclear, ask for clarification
- **Stay professional**: Keep discussions respectful and constructive

---

## Community

### Communication Channels

- **GitHub Issues**: Bug reports, feature requests, discussions
- **Pull Requests**: Code contributions and reviews
- **Hyperledger Discord**: Real-time chat (if available)

### Getting Help

If you need assistance:

1. **Check documentation**: Review `Docs/Readme.md` and code comments
2. **Search issues**: Someone may have already asked your question
3. **Ask in issues**: Open a "question" issue for help
4. **Join community channels**: Connect with other contributors

### Recognition

Contributors are recognized in:
- **Git history**: Your commits with proper attribution
- **Pull request comments**: Thanks from maintainers and community
- **Project README**: Major contributors may be listed

---

## License

By contributing to this project, you agree that your contributions will be licensed under the **Apache License 2.0**.

---

## Questions?

If you have questions about contributing, please:

1. Review this guide thoroughly
2. Check existing issues and pull requests
3. Open an issue with the "question" label

**Thank you for contributing to Fabric-X Block Explorer!** 🎉
