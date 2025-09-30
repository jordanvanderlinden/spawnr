# Contributing to Spawnr

Thank you for your interest in contributing to Spawnr! This document provides guidelines and instructions for contributing.

## Commit Message Convention

This project uses [Conventional Commits](https://www.conventionalcommits.org/) for automated semantic versioning and changelog generation.

### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- **feat**: A new feature (triggers minor version bump)
- **fix**: A bug fix (triggers patch version bump)
- **perf**: Performance improvements (triggers patch version bump)
- **docs**: Documentation changes (triggers patch version bump)
- **style**: Code style changes (formatting, missing semicolons, etc.)
- **refactor**: Code refactoring without feature changes
- **test**: Adding or updating tests
- **build**: Changes to build system or dependencies
- **ci**: Changes to CI/CD configuration
- **chore**: Other changes that don't modify src or test files

### Breaking Changes

Add `BREAKING CHANGE:` in the commit footer or add `!` after the type to trigger a major version bump:

```
feat!: change API endpoint structure

BREAKING CHANGE: The API endpoints have been restructured
```

### Examples

```bash
feat(clusters): add dark mode toggle to UI
fix(jobs): resolve pod cleanup issue on job deletion
docs: update README with new cluster management features
perf(k8s): improve cluster switching performance
refactor(handlers): simplify error handling logic
```

## Development Workflow

1. **Fork the repository**
2. **Create a feature branch**:
   ```bash
   git checkout -b feat/my-new-feature
   # or
   git checkout -b fix/bug-description
   ```

3. **Make your changes** following the code style and conventions

4. **Test your changes**:
   ```bash
   go test ./...
   go vet ./...
   ```

5. **Commit your changes** using conventional commits:
   ```bash
   git commit -m "feat(component): add new capability"
   ```

6. **Push to your fork**:
   ```bash
   git push origin feat/my-new-feature
   ```

7. **Open a Pull Request** against the `master` branch

## Pull Request Process

1. Update the README.md with details of changes if applicable
2. Ensure all tests pass
3. Ensure your code follows Go best practices
4. Use conventional commit messages
5. Link any relevant issues in the PR description
6. Wait for review and address any feedback

## CI/CD Pipeline

When you push to `master`, the following automated processes occur:

1. **Semantic Release**: Analyzes commits and determines the next version
2. **Docker Build**: Builds and pushes multi-arch Docker images to GitHub Container Registry
3. **Helm Chart Release**: Packages and publishes the Helm chart to GitHub Pages

## Code Style

- Follow standard Go formatting (`gofmt`, `goimports`)
- Write clear, descriptive variable and function names
- Add comments for complex logic
- Keep functions focused and concise

## Testing

- Write tests for new features
- Ensure existing tests still pass
- Aim for meaningful test coverage

## Getting Help

- Open an issue for bugs or feature requests
- Join discussions in existing issues
- Check the README for documentation

Thank you for contributing! 🎉
